# grpc-cqrs-go

Feature-first + CQRS アーキテクチャで構築した gRPC サーバーです。
レシピとユーザーの管理、および ReBAC による公開範囲・共有制御を行う API を提供します。

## 技術スタック

- **Go 1.27** - アプリケーション言語
- **gRPC** - API プロトコル
- **PostgreSQL 18** - データベース
- **Wire** - 依存性注入 (DI)
- **sqlc** - SQL からの型安全なコード生成
- **Atlas** - データベースマイグレーション
- **Buf** - Protobuf コード生成
- **Envoy / ESPv2（前段ゲートウェイ）+ OIDC (JWKS)** - 認証（ゲートウェイが JWT を検証し検証済み ID をヘッダ伝播）
- **zerolog** - 構造化ログ
- **Docker (distroless) / GitHub Actions → GHCR** - コンテナ配布（linux/arm64）

## アーキテクチャ

Feature-first + CQRS パターンを採用し、ドメイン（フィーチャ）ごとに Command（書き込み）と Query（読み取り）を分離しています。

### ディレクトリ構成

```
.
├── cmd/
│   └── serve/
│       ├── main.go                 # サーバーエントリーポイント
│       ├── wire.go                 # Wire DI 設定
│       └── wire_gen.go             # Wire 生成コード
├── database/
│   ├── migrations/                 # Atlas マイグレーションファイル
│   ├── queries/                    # sqlc クエリ定義 (recipe/tuple/user.sql)
│   └── schema.pg.hcl               # Atlas 宣言的スキーマ
├── internal/
│   ├── app/                        # アプリケーション層（フィーチャ単位）
│   │   ├── recipe/                 # Recipe ドメイン
│   │   │   ├── command/            #   書き込み (Create, Update, UpdateVisibility)
│   │   │   ├── query/              #   読み取り (GetRecipe)
│   │   │   ├── entity/             #   エンティティ (Recipe, Visibility)
│   │   │   ├── handler.go          #   gRPC ハンドラー
│   │   │   ├── convert.go          #   pb ⇔ ドメインの変換 (Visibility)
│   │   │   └── wire.go
│   │   ├── user/                   # User ドメイン
│   │   │   ├── command/            #   書き込み (CreateUser)
│   │   │   ├── entity/             #   エンティティ (User)
│   │   │   ├── handler.go
│   │   │   └── wire.go
│   │   ├── share/                  # Share 機能（レシピ共有）
│   │   │   ├── command.go          #   ShareRecipe（tuple 付与）
│   │   │   ├── handler.go
│   │   │   └── wire.go
│   │   ├── registrar.go            #   サービス登録
│   │   └── wire.go
│   ├── identity/                   # 現在ユーザーの identity（ctx 伝播・grpc 非依存）
│   │   ├── type.go                 #   UIDKey / UserID(ctx)→string / ErrUnauthenticated
│   │   └── gateway/                #   ゲートウェイヘッダの解釈 (Parse→sub, HeaderUserInfo)
│   ├── authz/                      # 認可 (ReBAC)
│   │   ├── check.go                #   Checker (Can*/Is*)
│   │   ├── model.go                #   Tuple, ObjectType, Relation, Permission
│   │   ├── storage.go              #   Storage インターフェース
│   │   ├── error.go
│   │   └── wire.go
│   ├── db/                         # データベース層（エンティティ単位）
│   │   ├── recipe/                 #   recipe 永続化 (command + query)
│   │   ├── user/                   #   user 永続化
│   │   ├── tuple/                  #   relation_tuples 永続化
│   │   ├── gen/                    #   sqlc 生成コード (DO NOT EDIT)
│   │   ├── pool.go                 #   コネクションプール
│   │   └── wire.go
│   ├── grpcerr/                    # gRPC エラー変換 (WithStatus: cause 保持 + 安全な status)
│   ├── health/                     # gRPC ヘルスチェック (grpc.health.v1.Health, DB 疎通連動)
│   ├── interceptor/                # gRPC インターセプター
│   │   ├── auth.go                 #   認証（ヘッダ→sub, GatewayAuthUnaryInterceptor + 公開メソッド除外）
│   │   ├── error.go                #   エラー処理（内部エラー秘匿 + cause ログ + code 別ログレベル）
│   │   ├── logging.go              #   リクエストログ
│   │   └── recovery.go             #   パニックリカバリー
│   └── logger/
│       └── zerolog.go
├── envoy/                          # Envoy 前段ゲートウェイの参照実装 (jwt_authn + JWKS)
├── espv2/                          # ESPv2 (Cloud Endpoints) 導入 scaffold（GCP 前提の代替案）
├── Dockerfile                      # app イメージ (multi-stage / distroless / arm64)
├── Dockerfile.migrate              # migrate イメージ (Atlas + migrations 同梱)
├── .github/workflows/build.yml     # Release publish で GHCR へ arm64 push
├── pkg/pb/                         # Protobuf 生成コード (recipe/share/user の v1)
├── proto/                          # Protobuf 定義 (<name>/v1/<name>.proto)
├── atlas.hcl                       # Atlas 設定
├── buf.yaml / buf.gen.yaml         # Buf 設定
├── compose.yaml                    # Docker Compose
├── mise.toml                       # ツールバージョン管理
└── sqlc.yaml                       # sqlc 設定
```

### CQRS パターン

各ドメイン（フィーチャ）は以下のレイヤーで構成されています:

```
handler.go          ← gRPC リクエストの受付・バリデーション・エラー変換
  ├── command/      ← 書き込み操作（エンティティ操作 → Storage インターフェース）
  ├── query/        ← 読み取り操作（Storage → 読み取りモデル）
  └── entity/       ← エンティティの定義（Recipe / User など）
```

- Storage インターフェースにより app 層と DB 層が疎結合（依存の向きは `db → app`）
- DB 層はエンティティ単位（`db/recipe`, `db/user`, `db/tuple`）で実装。CQRS の read/write の差は app 側の interface が表現する
- エンティティは非公開フィールド + コンストラクタ + getter でカプセル化（不正な生成を型で防止）

### 認証 / identity

app 内でトークン検証はしない。**前段(ゲートウェイ)が検証済みの ID をヘッダで渡す前提**で、
app はそれを信頼して現在ユーザーの identity を確定・伝播するだけ（`internal/identity`）。

- **本番**: 前段ゲートウェイ（Envoy の `jwt_authn` 等）が OIDC の ID トークンを **JWKS で検証**し、
  検証済み payload を `X-Endpoint-API-UserInfo`(base64url JSON) ヘッダに載せる
  → `GatewayAuthUnaryInterceptor` が `sub` を取り出し ctx へ（参照実装は `envoy/`）
- **ローカル**: ゲートウェイ無し。開発者がヘッダを直接埋め込む（検証なし・impersonate 用）
- 以降 `identity.UserID(ctx)` で **`string`（IdP の `sub`）** を取得（ctx を読むのは handler だけ、下層へは typed で渡す）
- **user id = IdP の `sub` をそのまま採用**（ULID とは限らない不透明文字列。例: Google OAuth は数値文字列）。
  `CreateUser` は認証済み `sub` を user.id として登録する
- reflection / health は app 側 interceptor では認証不要。ただし外部公開経路では reflection も
  ゲートウェイで JWT 必須にすること（無認証はヘルスチェックのみ。`envoy/` 参照）

> ⚠️ app はヘッダを信頼するため、**ゲートウェイ経由でのみ到達可能**にすること
> （直接到達できるとヘッダ偽造で認証を素通りできる）。ゲートウェイ側もクライアント送信の
> `X-Endpoint-API-UserInfo` を除去し、検証済みの値だけを渡すこと。

> **IdP について**: JWKS を公開する OIDC provider（Google OAuth / Auth0 / Keycloak /
> Cloudflare Access 等）を使う。Firebase の securetoken は公開鍵が x509 で JWKS 非対応のため、
> Envoy の `remote_jwks` では直接利用できない。

### 認可 (ReBAC)

Relationship-Based Access Control を採用。

- **Visibility**: `public`（誰でも） / `private`（owner のみ） / `restricted`（owner + 共有相手）
- **Relation**: `owner ⊃ editor ⊃ viewer`（owner は複数可）
- **Tuple ストア**: `relation_tuples` テーブルで「誰が・何に・どの relation か」を管理。subject は不透明な文字列として扱い、ULID かどうかは authz の関心外
- **Permission**: `map[ObjectType]map[Permission][]Relation` で action → 許可 relation を定義
- view の認可は `authz.Checker.CanViewRecipe` が Visibility に応じて判定、edit/share/公開範囲変更は owner/editor を tuple で判定

### エラー処理

- ハンドラは既知エラーを `grpcerr.WithStatus(cause, code, msg)` で変換（client には安全な status、ログには cause を保持）
- 未変換の生エラーは `ErrorHandlingUnaryInterceptor` が汎用 `Internal` + cause ログに（**内部エラーを client に漏らさない**）
- 横断的な `PermissionDenied` は interceptor で集約
- DB 境界で低レベルエラーを翻訳（`23505` → AlreadyExists / `pgx.ErrNoRows` → NotFound）
- **ログレベルは gRPC コードで決める**（`levelForCode`）: 想定内のクライアントエラー
  （`InvalidArgument` / `PermissionDenied` / `AlreadyExists` / `Unauthenticated`）は `info`、
  `NotFound` や一時障害（`Unavailable` 等）は `warn`、サーバ起因（`Internal` 等）は `error`

### ヘルスチェック

`grpc.health.v1.Health`（`internal/health`）を登録し、**プロセス生存ではなく実際に serve 可能か**を返す。
バックグラウンドで DB に定期 ping し、成否で `SERVING` / `NOT_SERVING` を切り替える（状態遷移時のみログ）。
前段の Envoy は app cluster の `grpc_health_check` でこれを叩き、readiness に反映する
（TCP チェックだと app ハング/DB 断でも healthy に見えてしまうため）。

## セットアップ

### 前提条件

- [mise](https://mise.jdx.dev/) (ツールバージョン管理)
- Docker / Docker Compose

### 1. ツールのインストール

```bash
mise install
```

### 2. データベースの起動

```bash
docker compose up -d
```

PostgreSQL が `localhost:25432` で起動します。

### 3. マイグレーションの実行

```bash
atlas migrate apply --env local
```

### 4. サーバーの起動

```bash
go run ./cmd/serve
```

サーバーが `localhost:50051` で起動します。ポートは `-port` フラグで変更可能です。

## gRPC API

認証は前段のゲートウェイが検証済み ID を `x-endpoint-api-userinfo` ヘッダで渡す前提（[認証 / identity](#認証--identity) 参照）。

- **本番**（ゲートウェイ前段）: クライアントは `authorization: Bearer <OIDC の ID トークン>` を送る
  （ゲートウェイが JWKS で検証し `x-endpoint-api-userinfo` を付与。例: Google OAuth の ID トークン）
- **ローカル**（ゲートウェイ無し）: `x-endpoint-api-userinfo` を手で埋め込む。以下の例はこの形。

```bash
# 任意の sub（本番では IdP の sub に対応）を base64url(JSON) にしてヘッダ値に（app は padding 有無どちらも可）
SUB=local-user-1
INFO=$(printf '{"sub":"%s"}' "$SUB" | basenc --base64url | tr -d '=')
```

### UserService

#### CreateUser（要認証）

認証済みユーザーを app 側の users に登録します（user id = IdP の `sub`）。

```bash
grpcurl -plaintext -H "x-endpoint-api-userinfo: $INFO" -d '{
  "name": "Kodai",
  "email": "kodai@example.com"
}' localhost:50051 user.v1.UserService/CreateUser
```

### RecipeService

#### CreateRecipe（要認証）

作成者が owner として登録されます（`private` 初期）。

```bash
grpcurl -plaintext -H "x-endpoint-api-userinfo: $INFO" -d '{
  "title": "カレーライス",
  "description": "スパイスから作る本格カレー"
}' localhost:50051 recipe.v1.RecipeService/CreateRecipe
```

#### UpdateRecipe（要認証・editor 以上）

```bash
grpcurl -plaintext -H "x-endpoint-api-userinfo: $INFO" -d '{
  "id": "<recipe_id>",
  "title": "カレーライス 改",
  "description": "改良版"
}' localhost:50051 recipe.v1.RecipeService/UpdateRecipe
```

#### ChangeVisibility（要認証・owner のみ）

```bash
grpcurl -plaintext -H "x-endpoint-api-userinfo: $INFO" -d '{
  "id": "<recipe_id>",
  "visibility": "VISIBILITY_PUBLIC"
}' localhost:50051 recipe.v1.RecipeService/ChangeVisibility
```

#### GetRecipe

visibility に応じて認可されます（public は誰でも / private は owner / restricted は共有相手）。

```bash
grpcurl -plaintext -H "x-endpoint-api-userinfo: $INFO" -d '{
  "id": "<recipe_id>"
}' localhost:50051 recipe.v1.RecipeService/GetRecipe
```

レスポンス:
```json
{
  "recipe": {
    "id": "550e8400-...",
    "userId": "101128207731415003767",
    "title": "カレーライス",
    "description": "スパイスから作る本格カレー",
    "visibility": "VISIBILITY_PRIVATE",
    "createdAt": "2026-03-08T...",
    "updatedAt": "2026-03-08T..."
  },
  "user": { "id": "101128207731415003767", "name": "Kodai", "email": "kodai@example.com" }
}
```
（`userId` / `user.id` は IdP の `sub`。recipe の `id` は UUID）

### ShareService

#### ShareRecipe（要認証・owner のみ）

対象ユーザーに `viewer` / `editor` を付与します。重複付与は `AlreadyExists` になります。

```bash
grpcurl -plaintext -H "x-endpoint-api-userinfo: $INFO" -d '{
  "recipe_id": "<recipe_id>",
  "target_user_id": "<user_id>",
  "relation": "viewer"
}' localhost:50051 share.v1.ShareService/ShareRecipe
```

### サービス一覧の確認

gRPC リフレクションが有効です:

```bash
grpcurl -plaintext localhost:50051 list
```

## デプロイ / コンテナ

- **イメージ**: `Dockerfile`（app、static/distroless/`linux/arm64`）と `Dockerfile.migrate`
  （`arigaio/atlas` + migrations 同梱、`atlas migrate apply --env k8s`）。
- **配布**: GitHub **Release を publish** すると `.github/workflows/build.yml` が
  `ghcr.io/kodaikumatani/grpc-cqrs-go{,-migrate}` を arm64 で GHCR に push（tag = リリースの semver）。
- **前段ゲートウェイ**: `envoy/`（`jwt_authn` + JWKS で JWT 検証 → `X-Endpoint-API-UserInfo` 付与、
  app cluster の `grpc_health_check`）。app は `-host 127.0.0.1`（loopback）で起動し、Envoy 経由のみ到達。
- **k8s (self-hosted) 例**: initContainer(migrate) + app(loopback) + envoy(sidecar) で 1 Pod。
  共有 PostgreSQL に `sub` を user id とするスキーマを流す（Flux/GitOps でのデプロイ構成は別リポジトリ）。

## コード生成

```bash
# Protobuf → Go コード生成
buf generate

# SQL → Go コード生成
sqlc generate

# Wire DI コード生成
cd cmd/serve && wire

# スキーマ変更からマイグレーション生成 (Atlas)
atlas migrate diff <name> --env local
```
