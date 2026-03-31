# OIDC Client
GoでOpenID Connect (OIDC) のRelying Partyを実装  
学習・理解を目的として、OIDC関連の外部ライブラリを使用せず独自実装としている。

## スコープ
**対応**
- Authorization Code Flow ([OIDC Core 1.0 §3.1](https://openid.net/specs/openid-connect-core-1_0.html#CodeFlowAuth))

**対応しない（MVPスコープ外）**
- PKCE
- Refresh Token の管理
- Client Credentials Flow
- Hybrid Flow / Implicit Flow
- Dynamic Client Registration
- UserInfo Endpoint へのアクセス

## 使用方法
### 前提条件
- Go 1.25 以降
- OpenID Provider でのアプリケーション登録済み（client_id, client_secret 取得済み）
- `redirect_uri` が Provider 側で登録済み

### 環境変数

**必須**

| 変数名 | 説明 | 例 |
|---|---|---|
| `GOOGLE_CLIENT_ID` | OP に登録した client_id | `xxx.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET` | OP に登録した client_secret | `GOCSPX-...` |
| `REDIRECT_URI` | OP に登録したコールバック URI | `https://rp.example.com/callback` |
| `ALLOWED_RETURN_TO_ORIGINS` | ログイン後のリダイレクト先として許可する SPA のオリジン（カンマ区切りで複数指定可） | `https://app.example.com` |

**任意（デフォルト値あり）**

| 変数名 | 説明 | デフォルト |
|---|---|---|
| `PORT` | サーバーのリッスンポート | `8080` |
| `REDIS_ADDR` | Redis の接続先 | `localhost:6379` |
| `SESSION_TTL_MINUTES` | ログインセッションの有効期限（分） | `60` |
| `TRANSACTION_TTL_MINUTES` | 認可トランザクション（state/nonce）の有効期限（分） | `10` |
| `SECURE_COOKIE` | Cookie に `Secure` 属性を付与するか（本番では `true` 推奨） | `false` |
| `AUTH_METHOD` | Token Endpoint の認証方式（`basic` or `post`） | `basic` |
| `DISCOVERY_TIMEOUT_SECONDS` | OIDC Discovery リクエストのタイムアウト（秒） | `10` |
| `VERSION` | ログに出力するサービスバージョン | `unknown` |
