# Google OIDC Relying Party の仕様

## 概要

このシステムは、Google OpenID Connect仕様に準拠したRelying Party (RP) として機能する。
Authorization Code Flowを基本フローとし、IDトークンの検証、UserInfo取得、セッション管理までをカバーする。
PKCEは将来の実装対象として仕様に含める。

## 準拠する仕様

このシステムの設計と実装は、以下の仕様に準拠する。

| 仕様                              | 用途                             | URL                                                                  |
| --------------------------------- | -------------------------------- | -------------------------------------------------------------------- |
| OpenID Connect Core 1.0           | フロー全体、IDトークン、クレーム | https://openid.net/specs/openid-connect-core-1_0.html                |
| OpenID Connect Discovery 1.0      | プロバイダメタデータ取得         | https://openid.net/specs/openid-connect-discovery-1_0.html           |
| OAuth 2.0 (RFC 6749)              | 認可フレームワーク基盤           | https://www.rfc-editor.org/rfc/rfc6749                               |
| OAuth 2.0 PKCE (RFC 7636)         | code_verifier / code_challenge   | https://www.rfc-editor.org/rfc/rfc7636                               |
| JWT (RFC 7519)                    | IDトークン構造                   | https://www.rfc-editor.org/rfc/rfc7519                               |
| JWS (RFC 7515)                    | トークン署名検証                 | https://www.rfc-editor.org/rfc/rfc7515                               |
| JWK (RFC 7517)                    | 公開鍵セット                     | https://www.rfc-editor.org/rfc/rfc7517                               |
| OAuth 2.0 Bearer Token (RFC 6750) | アクセストークン利用             | https://www.rfc-editor.org/rfc/rfc6750                               |
| Google OIDC ドキュメント          | Google固有の実装詳細             | https://developers.google.com/identity/openid-connect/openid-connect |

## ログインフロー

このシステムは以下のフェーズで構成される。

### 1. メタデータ取得と設定 (Discovery)

Googleの Discovery Endpoint からプロバイダメタデータを取得する。

```
GET https://accounts.google.com/.well-known/openid-configuration
```

**必須取得フィールド:**

- `authorization_endpoint` — 認可エンドポイント
- `token_endpoint` — トークンエンドポイント
- `userinfo_endpoint` — UserInfo取得先
- `jwks_uri` — 公開鍵セット取得先
- `issuer` — IDトークンの `iss` クレームと比較するために必要
- `response_types_supported` — "code" をサポートしていることを確認
- `subject_types_supported` — "public" をサポートしていることを確認
- `id_token_signing_alg_values_supported` — RS256が含まれることを確認

**キャッシュ:**

取得したメタデータは共有インメモリストア（Redis 等）にキャッシュし、TTL で自動更新する。

- キー: `oidc:discovery:metadata:{issuer}`
- 水平スケール時に複数インスタンスが同じキャッシュを参照できるようにするため、ローカルメモリへの保存は行わない。

**根拠:** OpenID Connect Discovery 1.0 Section 4

### 2. 認可リクエスト (Authorization Request)

ユーザーをGoogleの認可エンドポイントにリダイレクトする。このフェーズでは、認可サーバーからユーザーを認証し、ユーザーの同意を得る。

**必須パラメータ:**

| パラメータ              | 値                                               | 根拠                          |
| ----------------------- | ------------------------------------------------ | ----------------------------- |
| `client_id`             | Google Cloud Consoleで取得                       | RFC 6749 Section 4.1.1        |
| `response_type`         | `code`                                           | OIDC Core Section 3.1.2.1     |
| `scope`                 | `openid email profile`（最低限 `openid` は必須） | OIDC Core Section 3.1.2.1     |
| `redirect_uri`          | 事前登録済みのコールバックURL                    | RFC 6749 Section 3.1.2        |
| `state`                 | CSRF防止用のランダム値（`crypto/rand`で生成）    | RFC 6749 Section 4.1.1, 10.12 |
| `nonce`                 | リプレイ攻撃防止用のランダム値                   | OIDC Core Section 3.1.2.1     |

**stateとnonceの生成:**
`crypto/rand` を使い、最低128ビットのエントロピーを確保する。`math/rand` は使用禁止。

**stateとnonceの保存:** 共有インメモリストアに保存し、コールバック時に照合する。

### 3. コールバック処理 (Token Exchange)

Googleからリダイレクトされた後、受け取った認可コードをトークンに交換する。

**ステップ 3.1: コールバックパラメータの検証**

1. `error` パラメータがあればエラー処理（RFC 6749 Section 4.1.2.1）
2. `state` パラメータをセッション保存値と比較。不一致なら処理中断（CSRF防止）
3. `code` パラメータを取得

**ステップ 3.2: トークンリクエスト**

Token Endpoint に `POST` する。クライアント認証方式はサーバー設定値に従い、以下の2方式をサポートする。

**client_secret_basic**（デフォルト。`client_id:client_secret` を Base64 エンコードして Authorization ヘッダで送る）

```
POST https://oauth2.googleapis.com/token
Authorization: Basic <base64(client_id:client_secret)>
Content-Type: application/x-www-form-urlencoded

code=<authorization_code>
&redirect_uri=<redirect_uri>
&grant_type=authorization_code
```

**client_secret_post**（リクエストボディに `client_id` / `client_secret` を含める）

```
POST https://oauth2.googleapis.com/token
Content-Type: application/x-www-form-urlencoded

code=<authorization_code>
&client_id=<client_id>
&client_secret=<client_secret>
&redirect_uri=<redirect_uri>
&grant_type=authorization_code
```

**レスポンス:**

```json
{
  "access_token": "...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "id_token": "..."
}
```

**レスポンスの検証:**

- `token_type` が `Bearer` であることを確認する。理解できないトークンタイプの場合は access_token を使用してはならない（RFC 6750）。
- `scope` フィールドが含まれる場合、付与されたスコープを確認する。`openid` が含まれていない場合は処理を中断する。`email` / `profile` 等の任意スコープが付与されていない場合は認証自体は成功とし、`/me` レスポンスで該当フィールドを省略する（RFC 6749 Section 5.1）。

**エラーレスポンス:**

Token Endpoint がエラーを返した場合、RP はコールバックを 403 で終了する（エラーコードの定義: RFC 6749 Section 5.2、403 への統一はコールバックエラーレスポンスの設計方針に基づく）。

| エラーコード | 意味 |
|---|---|
| `invalid_request` | 必須パラメータ欠落・不正な値 |
| `invalid_client` | クライアント認証失敗（`client_id` / `client_secret` 不正） |
| `invalid_grant` | Authorization Code が無効・期限切れ・使用済み |
| `unauthorized_client` | このクライアントへの grant_type 使用が不許可 |
| `unsupported_grant_type` | OP が `authorization_code` をサポートしていない |
| `invalid_scope` | 要求スコープが不正 |

### 4. IDトークン検証

IDトークンはJWS (RFC 7515) 形式で受け取る。以下の手順に従って検証する。

**ステップ 4.1: JWTのデコード**

1. ドット(`.`)で3パートに分割: `header.payload.signature`
2. ヘッダーをBase64URLデコードし、`alg` と `kid` を取得

**ステップ 4.2: JWKの取得**

- JWK セットは共有インメモリストア（キー: `oidc:discovery:jwks:{issuer}`）から取得する。
- `kid` が共有ストアのキャッシュに見つからない場合は、OP 側の鍵ローテーションを考慮して `jwks_uri` から再取得しキャッシュを更新する。

**ステップ 4.3: 署名検証**

1. ヘッダーの `alg` が RP のサポートするアルゴリズムのホワイトリスト（例: `RS256`, `ES256`）に含まれるか確認。含まれない場合はトークンを拒否する
2. `kid` を使って取得した JWK セットから対応する公開鍵を取得
3. `header.payload` 部分に対してヘッダーの `alg` に従って署名検証する

**根拠:** OIDC Core Section 3.1.3.7 Step 6, RFC 7515 Section 5.2

**ステップ 4.4: クレーム検証**　

以下を全て検証する。1つでも失敗したらトークンを拒否する。

| クレーム | 検証内容                                                              | 根拠                                      |
| -------- | --------------------------------------------------------------------- | ----------------------------------------- |
| `iss`    | `"https://accounts.google.com"` または `"accounts.google.com"` と一致 | OIDC Core 3.1.3.7 Step 2                  |
| `aud`    | 自アプリの `client_id` を含む                                         | OIDC Core 3.1.3.7 Step 3                  |
| `azp`    | `aud` が複数値の場合、`azp` が自アプリの `client_id` と一致           | OIDC Core 3.1.3.7 Step 4, Google OIDC Doc |
| `exp`    | OP が定めた絶対的な有効期限。現在時刻が `exp` を超えている場合は必ず拒否する（clockskew許容: 最大5分程度） | OIDC Core 3.1.3.7 Step 9  |
| `iat`    | RP の独自ポリシーによる追加検証。発行から一定時間（例: 10分）を超えたトークンを拒否することで、古いトークンの再利用を防ぐ | OIDC Core 3.1.3.7 Step 10 |
| `nonce`  | 認可リクエスト時にセッションに保存した値と一致                        | OIDC Core 3.1.3.7 Step 11                 |
| `sub`    | 存在すること（Googleユーザーの一意識別子）                            | OIDC Core 2                               |

**`aud` が配列の場合の処理:**
`aud` クレームは文字列または文字列配列のどちらかで来る可能性がある（RFC 7519 Section 4.1.3）。
両方のケースに対応する。

### 5. UserInfo取得（オプション）

IDトークンのクレームで不足する情報がある場合、UserInfoエンドポイントを利用してユーザー情報を取得する。

```
GET https://openidconnect.googleapis.com/v1/userinfo
Authorization: Bearer <access_token>
```

**根拠:** OIDC Core Section 5.3

**注意:** IDトークンの `sub` と UserInfoレスポンスの `sub` が一致することを確認する（OIDC Core Section 5.3.2）。

### 6. セッション管理

認証成功後、サーバーサイドセッションを作成し、ユーザーの認証状態を維持する。システムは以下のセッション管理機構を備える。

- セッションIDは `crypto/rand` で生成
- セッションIDはSecure, HttpOnly, SameSite=Lax のCookieで送信
- セッションの実体（ユーザー識別子等）は共有インメモリストアに保存する
- セッション有効期限を設定し、期限切れ時は再認証を要求する

## セキュリティ要件

このシステムは以下のセキュリティ要件を満たす。

1. **TLS必須**: 全てのエンドポイント通信はHTTPS（RFC 6749 Section 3.1.2.1）
2. **state検証**: コールバック時にstateを検証することでCSRF攻撃を防止（RFC 6749 Section 10.12）
3. **nonce検証**: IDトークンのnonce検証によりリプレイ攻撃を防止（OIDC Core Section 3.1.2.1）
4. **redirect_uri完全一致**: 認可リクエストとトークンリクエストで同一のredirect_uriを使用（RFC 6749 Section 4.1.3）
5. **client_secretの安全な管理**: 環境変数またはシークレットマネージャで管理し、コードにハードコードをしない
6. **IDトークンの全クレーム検証**: セクション4のステップ4.3に記載された全ての検証項目を実装する
7. **JWKキャッシュの適切な更新**: キャッシュミス時（`kid` が見つからない場合）にJWKセットを再取得する仕組みを備える

## 将来の実装対象

以下の機能は実装予定である。

### PKCE (RFC 7636)

Authorization Code Injection攻撃への対策として、PKCE の実装を予定している。実装時は以下の仕様に従う。

**パラメータ:**

- `code_verifier`: 43〜128文字のランダム文字列（unreserved characters: `[A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"`）
- `code_challenge`: `BASE64URL(SHA256(code_verifier))`（パディングなし）
- `code_challenge_method`: `S256`

**根拠:** RFC 7636 Section 4.1, 4.2
