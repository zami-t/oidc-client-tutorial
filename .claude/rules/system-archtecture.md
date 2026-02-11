# System Architecture

## 登場者

| 登場者 | 説明 | 仕様上の呼び名 |
|--------|------|----------------|
| ブラウザ | End-Userが操作するUAO | User Agent |
| このClient | 認証を要求するアプリ | Relying Party (RP) |
| OpenID Provider | 認証を行うサーバー | OpenID Provider (OP) |
| 共有インメモリストア | 認可トランザクション（state/nonce）とDiscovery（メタデータ/JWKS）を共有・TTL付きで管理するストレージ（例: Redis） | Shared Store |

## フロー
### 起動時/初回利用時: OIDC Discovery（メタデータ/JWKSの取得）

Discovery は OP のエンドポイント情報や検証用鍵の取得先を確定するための処理であり、毎回OPへ取りに行くと遅延要因になる。
一方で、インスタンスごとにキャッシュを持つと整合性・運用管理コストが上がるため、共有インメモリストアにキャッシュし、TTLで更新する。

```mermaid
sequenceDiagram
    participant RP as Client (RP)
    participant Store as 共有インメモリストア
    participant OP as OP

    RP->>Store: READ metadata/jwks
    alt HIT
        Store-->>RP: metadata/jwks
    else MISS または未知の kid
        RP->>OP: GET /.well-known/openid-configuration
        OP-->>RP: 200 Provider Metadata
        RP->>Store: WRITE metadata (TTL)

        RP->>OP: GET {jwks_uri}
        OP-->>RP: 200 JWKS (公開鍵セット)
        RP->>Store: WRITE jwks (TTL)
    end
```

補足:

- 未知の kid については [shared-store.md](shared-store.md) を参照

### 認可コードフロー（/login → OP → /callback）と共有インメモリストア

```mermaid
sequenceDiagram
    participant Browser as ブラウザ
    participant RP as Client (RP)
    participant OP as OP
    participant Store as 共有インメモリストア

    Browser->>RP: GET /login
    RP->>RP: state/nonce/code_verifier を生成
    RP->>Store: WRITE {state, nonce, code_verifier} (TTL)
    RP->>RP: Authorization Request を組み立て（authorization_endpoint + 各種パラメータ）
    RP-->>Browser: 302 Location: OP /authorize?... (Authorization Request)

    Browser->>OP: GET /authorize?... (Authorization Request)
    Note over Browser,OP: End-User が認証
    OP-->>Browser: 302 redirect_uri?code=xxx&state=yyy

    Browser->>RP: GET /callback?code=xxx&state=yyy
    RP->>Store: READ {state, nonce, code_verifier}
    Store-->>RP: {state, nonce, code_verifier}
    RP->>RP: state を検証

    RP->>OP: POST Token Request (code + code_verifier)
    OP-->>RP: {id_token, access_token}
    RP->>RP: id_token を検証 (nonce)

    RP->>Store: DELETE {state, nonce, code_verifier}
    RP->>Store: WRITE session (TTL)
    RP-->>Browser: 200 (Set-Cookie: session_id, 認証結果)
```

## 共有インメモリストアの仕様

共有インメモリストアの具体仕様は、下記を参照。

- [shared-store.md](shared-store.md)

仕様参照: [OpenID Connect Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderMetadata)
