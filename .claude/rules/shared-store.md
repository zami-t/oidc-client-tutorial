# 共有インメモリストア仕様（Redis等）

このドキュメントは、本レポジトリにおける「共有インメモリストア（例: Redis）」の扱いを定義する。

## 選定理由（背景要求）

本レポジトリで「共有インメモリストア（例: Redis）」を採用する背景要求は以下。

- **複数インスタンス前提**: 水平スケール時にどのインスタンスが処理しても同じ結果になる必要がある。
- **整合性/運用管理**: インスタンスごとのローカルキャッシュを持つと、更新タイミング差・デバッグ難易度・運用コストが増えるため避ける。
- **低レイテンシ**: Discovery/JWKS を都度OPへ取りに行く遅延を避け、かつファイルI/Oに依存しない（メモリアクセス前提）。
- **TTLによる自動破棄**: 認可トランザクションの使い捨てデータ（state/nonce 等）やDiscovery/JWKSキャッシュを期限で自然に失効させたい。

## 保管対象

共有インメモリストアに保存するデータは以下。

- 認可トランザクション（Authorization Request〜Callback までの一時データ）
  - `state` と、それに紐づく `nonce`
- アプリケーションセッション（ログイン済み状態）
  - セッションIDとユーザー識別子（例: `sub`）の対応
- OIDC Discovery キャッシュ
  - OP メタデータ（`/.well-known/openid-configuration` の取得結果）
  - JWKS（`jwks_uri` の取得結果）

## 名前空間（キーのプレフィックス）

用途ごとにキーのプレフィックスを分ける。

- 認可トランザクション（state/nonce 等）: `oidc:tx:`
- アプリケーションセッション（ログイン済み状態）: `oidc:sess:`
- Discovery メタデータキャッシュ: `oidc:discovery:metadata:`
- JWKS キャッシュ: `oidc:discovery:jwks:`

## 認可トランザクション（state/nonce）の仕様

### キー構成

`state` の値をキーにする。`state` はランダム生成されるため自然にユニークキーとなる。

```
Key: oidc:tx:{state}
Value: {
    nonce:         string   // id_token 検証で使用
    created_at:    int64    // エントリの作成時刻（TTL管理用）
}
```

### ライフサイクル

| タイミング | オペレーション | 理由 |
|-----------|----------------|------|
| Authorization Request直前 | WRITE | state に対応する nonce（等）を保持する |
| コールバック受信時 | READ | 受け取った state で対応する値を検索・検証する |
| 検証完了後 | DELETE | RP側でも `state`/`nonce` をワンタイム化し、同一 `state` での再処理を防ぐ。Authorization Code 自体はOP側で単回使用でも、RP側の状態が残ると「state検証だけは通る」リクエストがTTL内に再度到達し得るため、処理完了後に破棄する。 |
| TTL 到達時 | DELETE | 未使用エントリを自動破棄する |

### TTL

- デフォルト TTL: 10分（例）
- `state` は End-User の認証/同意に時間がかかると、認可コードより先にTTL切れすることがある。
- 本レポジトリの方針: **認可コードが有効でも `state` が見つからない/期限切れならエラーとして返す**（セキュリティ優先、再ログインを促す）。
- 参考: [RFC 6749 §4.1.2](https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.2) は Authorization Code の短いライフタイムを推奨している。

## アプリケーションログインセッションの仕様

方針:

- RPはログイン完了時にランダムなセッションIDを発行する。
- ブラウザにはセッションIDのみを Cookie で返し、セッション実体は共有インメモリストアに保存する。

### キー構成

```
Key: oidc:sess:{session_id}
Value: {
  subject:        string   // 例: id_token の sub
  issuer:         string   // どのOPで認証したか（マルチOP想定時）
  created_at:     int64
  last_seen_at:   int64
}
```

### ライフサイクル

| タイミング | オペレーション | 理由 |
|-----------|----------------|------|
| ログイン成功時（id_token検証後） | WRITE | セッションを作成し、以降のリクエストでログイン済み判定を可能にする |
| 通常リクエスト受信時 | READ | CookieのセッションIDからログイン済み状態を復元する |
| ログアウト時 | DELETE | 明示的にログイン状態を破棄する |
| TTL 到達時 | DELETE | 放置セッションを自動失効させる |

### TTL

- セッション TTL は運用要件に依存する（例: 数十分〜数時間）。
- セッションが見つからない/期限切れの場合はログイン済みとして扱わず、再ログインを促す。

### Cookie（概要）

- Cookie にはセッションIDのみを格納し、個人情報やトークンを直接入れない。
- 属性は少なくとも `HttpOnly` / `Secure` を有効化し、`SameSite` は要件に合わせて設定する（一般に `Lax` から検討）。

## Discovery メタデータキャッシュの仕様

### キー構成

OPの `issuer` をキーに、Discoveryで得たメタデータをキャッシュする。

```
Key: oidc:discovery:metadata:{issuer}
Value: {
    issuer:                    string
    authorization_endpoint:    string
    token_endpoint:            string
    jwks_uri:                  string
    fetched_at:                int64
}
```

### TTL

TTL は運用都合に応じて設定する（例: 数時間〜1日）。

## JWKS キャッシュの仕様

### キー構成

```
Key: oidc:discovery:jwks:{issuer}
Value: {
    jwks:       object_or_string  // 実装に応じてJSON文字列等
    fetched_at: int64
}
```

### TTL

- TTL はメタデータより短めに設定する（例: 1時間〜数時間）。

### 未知の `kid` と強制リフレッシュ

ID Token のヘッダにある `kid` が、共有インメモリストアにキャッシュされている JWKS 内のどの鍵にも一致しない場合、OP側の鍵ローテーションの可能性を考慮し、`jwks_uri` から JWKS を再取得してキャッシュを更新する（必要に応じて検証をリトライする）。

## 失効・障害時の考え方（簡易）

- 共有インメモリストアが利用できない場合、認可トランザクション（state/nonce）の検証ができず、ログイン継続が困難になる。
- そのため、可用性（HA）や接続設定（タイムアウト/リトライ方針）はシステム要件として別途定義する。
