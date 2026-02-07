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
- OpenID Provider での アプリケーション登録済み（client_id, client_secret 取得済み）
- `redirect_uri` が Provider側で登録済み
