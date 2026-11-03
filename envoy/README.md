# Envoy ゲートウェイ (ESP の代替)

アプリの前段に Envoy を置き、JWT を検証して検証済みユーザー情報を
`X-Endpoint-API-UserInfo` ヘッダで backend に渡す構成。**GCP 非依存 / arm64 対応**なので
self-hosted k8s(Talos/RPi5 等)向け。ESP(Cloud Endpoints)を使う場合は `../espv2/` を参照。

## 仕組み

```
client ─(authorization: Bearer <JWT>)─▶ Envoy :8080
       ─(x-endpoint-api-userinfo: base64url(payload))─▶ app 127.0.0.1:50051
```

- **認証**: Envoy `jwt_authn` が JWKS で署名検証。`forward_payload_header` で payload を
  base64url(JSON) にしてヘッダへ → ESP の `X-Endpoint-API-UserInfo` と互換。
- **app は無改修**: `internal/identity/gateway.Parse` がこのヘッダから `sub` を読む。
- **認可(ReBAC)** は app 側(`internal/authz`)に残る。

## `envoy.yaml` の位置づけ

これは**そのままコピーして使う完成品ではなく、参照実装**。issuer / audience / JWKS の
ホスト名や CA バンドルのパスは環境ごとに違うので、**実際に動かす設定は deploy 側の
リポジトリが持つ**のが前提(例: k8s なら ConfigMap)。

したがって重要なのは下の「満たすべき不変条件」であって、ファイルの丸写しではない。
**deploy 側で設定を書くときは、必ずこのチェックリストを満たすこと。**

## 満たすべき不変条件

app のセキュリティモデルに直結する。**1 つでも欠けると認証を破られる。**

- [ ] **app を loopback bind する** (`-host 127.0.0.1`)。Envoy と同一 Pod の sidecar とし、
      Service は Envoy の `:8080` だけを晒す。app に直接到達できると後述のヘッダ偽造が通る。
- [ ] **クライアント由来の `x-endpoint-api-userinfo` を必ず除去する** (`header_mutation` を
      jwt_authn より**前**に置く)。app はこのヘッダを**無条件に信頼する**ので、これが無いと
      **クライアントが自分でヘッダを送るだけで任意ユーザーになりすませる**。
      経路を Envoy だけに絞っても防げない — Envoy が素通しするなら意味がない。
- [ ] **`forward: false`** にして `authorization` ヘッダを backend に流さない。
- [ ] **無認証で通すのは `/grpc.health.` のみ**。`/grpc.reflection.` を開けると、
      外部公開経路で全 service/method を匿名で列挙される。
      開発中に列挙したいなら Envoy を挟まず app を直接叩く。
- [ ] **`forward_payload_header` は `x-endpoint-api-userinfo`**、値は **base64url(JSON)**。
      `gateway.Parse` の入力仕様なので名前も encoding も変えない。

## 環境依存の設定 (deploy 側が持つ)

`envoy.yaml` の以下を置換する。

| プレースホルダ | 例(汎用 OIDC) |
|---|---|
| `<ISSUER>` | `https://your-idp.example.com/` |
| `<AUDIENCE>` | アプリの client_id / audience |
| `<JWKS_URI>` | `https://your-idp.example.com/.well-known/jwks.json` |
| `<JWKS_HOST>` | `your-idp.example.com` |

`<ISSUER>` は **トークンの `iss` と完全一致**でなければならない(末尾スラッシュも区別される)。
IdP によっては `iss` が複数形態を取る(例: Google は `https://accounts.google.com` と
`accounts.google.com` の両方)。その場合は provider を 2 つ定義して `requires_any` で受ける。

変更したら必ず検証する:
```bash
docker run --rm -v "$PWD/envoy.yaml:/e.yaml:ro" \
  envoyproxy/envoy:distroless-v1.34.1 --mode validate -c /e.yaml
```

## IdP の制約

- **JWKS を公開している IdP が必要**。Firebase の securetoken は公開鍵を **x509(PEM)** で
  配布し JWKS 非対応なので、Envoy の `remote_jwks` では**使えない**。
  Auth0 / Keycloak / Cloudflare Access / Google Sign-In 等を使う。
- **アクセストークンではなく ID トークンを送ること**。OAuth プロバイダによっては
  アクセストークンが不透明トークン(例: Google の `ya29.` 始まり)で、JWKS では検証できず
  全リクエストが 401 になる。
- **`sub` は任意の文字列でよい**(v0.3.0 以降)。identity は不透明文字列として扱うので、
  Google の 21 桁の数値文字列や Auth0 の `auth0|xxx` もそのまま user id になる
  (`users.id` / `recipes.user_id` は varchar(255))。
  ~~v0.2.0 までは ULID 前提だった~~ — この制約は撤廃済み。

## 運用上の落とし穴

実際に self-hosted k8s へ載せたときに踏んだもの。

- **外向き cluster には `dns_lookup_family: V4_ONLY` を指定する**。既定の AUTO は
  「AAAA が 1 件でも返れば A を引かない」ため、IPv6 egress の無いクラスタでは到達不能な
  IPv6 だけを掴み、有効なトークンでも `Jwks remote fetch is failed` になる。
  **DNS 解決自体は成功して endpoint が healthy 扱いになるうえ、`log_level: info` では
  ログに何も出ない**ので極めて気づきにくい。admin(`:9901`)の
  `/clusters` と `/stats`(`jwks_fetch_failed`, `ssl.handshake`)で切り分ける。
- **`validation_context` を省かない**。省くと Envoy は上流証明書を検証せず、JWKS を
  差し替えられればトークン偽造が通る。CA バンドルのパスは envoy イメージに合わせる。
- **ConfigMap を変えても Envoy は自動で読み直さない**。起動時に config を読むので、
  k8s なら Pod の再起動が要る(`kubectl rollout restart`)。ハッシュ付き名前になる
  `configMapGenerator` を使うと自動化できる。
- **gRPC を HTTP プロキシ経由で外部公開するときは trailer に注意**。`grpc-status` は
  HTTP/2 の trailer で返るため、経路に HTTP/1.1 のホップがあると
  **成功応答(body 有り)だけが落ちる**。
  ⚠️ **切り分けを body 0 バイトの応答でやらないこと。** `Jwt is missing` / `NotFound` /
  `Unimplemented` は **trailers-only 応答**で `grpc-status` が通常ヘッダに畳まれるため、
  **trailer を壊す経路でも素通りしてしまい「正常」に見える**。必ず実データが返る RPC で
  試し、`grpcurl -v` で受信メッセージ数と trailer を確認する。
  どうしても通らない経路がある場合は **gRPC-Web**(trailer を body 内に埋め込む)への
  切り替えを検討する。ブラウザは生の gRPC を喋れないので、Web クライアントを作るなら
  どのみち必要になる。

## ローカル動作確認(任意)

ローカルは通常 Envoy を挟まず、`x-endpoint-api-userinfo` を手で埋め込んで app を直接叩く
(`../README.md` の gRPC API 参照)。ヘッダは payload の base64url:

```bash
PAYLOAD='{"iss":"https://accounts.google.com","sub":"1234567890","email":"me@example.com"}'
USERINFO=$(printf '%s' "$PAYLOAD" | base64 | tr '+/' '-_' | tr -d '=\n')
grpcurl -plaintext -H "x-endpoint-api-userinfo: $USERINFO" \
  -d '{"id":"..."}' localhost:50051 recipe.v1.RecipeService/GetRecipe
```

Envoy 込みで試す場合は本物の IdP と JWT が要る。
