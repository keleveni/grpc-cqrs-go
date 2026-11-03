# ESPv2 (Extensible Service Proxy V2) 導入 scaffold

gRPC サーバーの前段に ESPv2 を **sidecar / ゲートウェイ**として置き、Firebase の
ID トークン検証をオフロードする構成の雛形です。**この scaffold だけでは動きません**
（Cloud Endpoints への service config デプロイ等が別途必要）。

## 構成

```
client ──(Bearer <ID token>)──▶ ESPv2 ──(X-Endpoint-API-UserInfo)──▶ app (gRPC, loopback)
                                 │ JWT 検証                          │ ヘッダから UID を取得
                                 └ 検証済みユーザー情報をヘッダ付与    └ ReBAC 等の認可は app 側
```

- **認証（誰か）** は ESPv2（`api_config.yaml` の Firebase provider）
- **認可（何してよいか / ReBAC）** は app 側（`internal/authz`）に残る
- app は**常に** `internal/authn/gateway.AuthUnaryInterceptor` で `X-Endpoint-API-UserInfo`
  から UID を取り出す（app 内でトークン検証はしない）。ヘッダの出所だけが環境で異なる:
  - **本番**: ESP が Firebase の ID トークンを検証してヘッダを付与
  - **ローカル**: ESP 無し。開発者がヘッダを直接埋め込む（検証なし・impersonate）

## セキュリティ前提（重要）
app が付与済みヘッダを **信頼する**ので、app は **ESPv2 経由でのみ到達可能**にすること
（compose では app のポートを公開せず `expose` のみ）。直接叩ければヘッダ偽造で認証を破れる。

## 実際に動かす手順

1. **proto descriptor を生成**
   ```bash
   # buf で descriptor set を出力（--include-imports 相当）
   buf build -o espv2/api_descriptor.pb
   # or protoc:
   # protoc --include_imports --descriptor_set_out=espv2/api_descriptor.pb \
   #   -I proto proto/**/*.proto
   ```
2. **`api_config.yaml` の `<PROJECT_ID>` を置換**
3. **Cloud Endpoints に service config をデプロイ**
   ```bash
   gcloud endpoints services deploy espv2/api_descriptor.pb espv2/api_config.yaml
   ```
4. **app をコンテナ化**（Dockerfile を用意。ポートは公開しない）
5. **起動**
   ```bash
   docker compose -f compose.yaml -f espv2/compose.espv2.yaml up
   ```
6. **app 側の設定は不要**：app は常に `gateway.AuthUnaryInterceptor`（ヘッダ信頼）。
   本番は ESP がヘッダを付与、ローカルは手で埋め込む。

## ローカルでヘッダを直接埋め込む（ESP 無し）

`X-Endpoint-API-UserInfo` は base64url(JSON)。uid を `sub` に入れて渡す:

```bash
UID=01J...                     # 対象ユーザーの ULID
INFO=$(printf '{"sub":"%s"}' "$UID" | basenc --base64url | tr -d '=')
grpcurl -plaintext -H "x-endpoint-api-userinfo: $INFO" -d '{
  "title": "カレー", "description": "..."
}' localhost:50051 recipe.v1.RecipeService/CreateRecipe
```
（`basenc` が無ければ `base64` でも可。app 側は padding 有無どちらも受理）

## 未対応 / 注意
- **ローカル(GCP 無し)での完全起動は未対応**。ESPv2 の managed mode は GCP 認証が必要。
  静的 config で動かす場合は service config JSON の生成が別途必要（fiddly）
- **public レシピの未認証 GetRecipe**：現状 config は全メソッド認証必須。未認証閲覧を許すなら
  `allow_without_credential` + app 側の UID 無し許容が必要
- **HTTP/JSON トランスコーディング**：必要なら proto に `google.api.http` を付与して有効化
