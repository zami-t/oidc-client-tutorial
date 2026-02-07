# Security

このドキュメントは、本Client（OIDC Authorization Code Flow + PKCE）において想定する攻撃と対策を、仕様に基づいて整理する。

## 前提（スコープ）

- RP はサーバーサイドで動作する **機密クライアント**（Confidential Client）であり、SPA/ネイティブアプリは対象外とする。
- Token Endpoint との通信はバックチャネルで行い、ブラウザ（User-Agent）が `id_token` を受け取って利用する構成は扱わない。
- 初期実装は **PKCE を実装しない**。PKCE は将来対応の TODO として扱う。

---

## 対応一覧

| # | 攻撃 | 対策 | 仕様参照 |
|---|------|------|----------|
| 1 | CSRF (Cross-Site Request Forgery) | `state` パラメータによる検証 | [RFC 6749 §10.12](https://datatracker.ietf.org/doc/html/rfc6749#section-10.12) |
| 2 | Authorization Code の再利用（リプレイ） | OP 側の Authorization Code ワンタイム性で担保 + RPにて使用後セッション削除 | [RFC 6749 §4.1.2](https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.2) |
| 3 | Authorization Code の傍受 | PKCE による code 束縛（TODO: 初期実装は未対応） | [RFC 7636 §1](https://datatracker.ietf.org/doc/html/rfc7636#section-1) |
| 4 | `id_token` の偽造・改ざん | 署名検証（JWS） | [RFC 7515 §5](https://datatracker.ietf.org/doc/html/rfc7515#section-5) |
| 5 | `id_token` の妥当性検証（`iss` / `aud` / `exp`） | Claim 検証（設定した値と一致するか） | [OIDC Core 1.0 §3.1.3.7](https://openid.net/specs/openid-connect-core-1_0.html#IDTokenValidation) |
| 6 | 未使用エントリによるセッション攻撃 | セッションストレージの TTL + 使用後削除 | [RFC 6749 §4.1.2](https://datatracker.ietf.org/doc/html/rfc6749#section-4.1.2) |

---

## 各対策の詳細

### 1. CSRF対策: `state` パラメータ

**攻撃内容**

攻撃者がコールバック URL に偽のリクエストを送り、被害者のブラウザで不正なコールバックを実行する。

**対策**

- Authorization Request に `state`（暗号的に安全なランダム文字列）を付与する
- コールバック時に、レスポンスの `state` がセッションストレージに保持した値と一致するか検証する
- 不一致の場合はリクエストを拒否する

**実装要件**

- `state` は `crypto/rand` で生成し、十分なエントロピーを確保する（最低128 bits）
- 検証後に対応するセッションエントリを削除し、再利用を防止する

---

### 2. Authorization Code の再利用（リプレイ）対策

**攻撃内容**

攻撃者が（何らかの経路で）Authorization Code を入手し、同じ Code を使って Token Request を再実行してトークンを取得しようとする。

**対策**

- OP は Authorization Code を *一度しか利用できない* ように扱う（仕様上の前提）
- クライアント側でも、コールバック処理が完了したセッションエントリを削除して再利用を防止する

**実装要件**

- 認可レスポンス（`code`）を処理したら、対応するセッションエントリを削除する
- セッションエントリには TTL を設定し、一定時間で自動的に無効化する

---

### 3. Authorization Code の傍受対策: PKCE（TODO）

**攻撃内容**

攻撃者が何らかの経路で Authorization Code を傍受し、自身で Token Request を送付してトークンを取得する。

**対策（TODO: 初期実装は未対応）**

- 認証リクエスト時に `code_challenge`（`BASE64URL(SHA256(code_verifier))`）を送付
- Token Request 時に元の `code_verifier` を送付
- OP側で `code_challenge` と `code_verifier` の対応を検証する
- 攻撃者は `code_verifier` を知らないため、傍受した Code では Token を取得できない

**実装要件（TODO）**

- `code_verifier`: 43〜128文字の文字列。使用可能な文字は `[A-Z] [a-z] [0-9] - . _ ~`（[RFC 7636 §4.1](https://datatracker.ietf.org/doc/html/rfc7636#section-4.1)）
- `code_challenge`: `BASE64URL(SHA256(code_verifier))`（[RFC 7636 §4.2](https://datatracker.ietf.org/doc/html/rfc7636#section-4.2)）
- `code_challenge_method`: `S256` を使用する

---

### 4. `id_token` 偽造対策: 署名検証

**攻撃内容**

攻撃者が偽の `id_token` を作成し、本物として受け入れられるよう偽装する。

**対策**

- OP の `jwks_uri` から公開鍵を取得する
- `id_token` の JWS 署名を、取得した公開鍵で検証する
- `kid`（Key ID）を用いて、複数鍵がある場合に適切な鍵を選択する

**実装要件**

- `id_token` の `header.alg` と JWKS の `alg` が一致することを確認する
- 署名検証に失敗した場合は `id_token` を無効とする

---

### 5. `id_token` の妥当性検証（`iss` / `aud` / `exp`）

**攻撃内容**

`id_token` の発行元や宛先、期限を検証しない場合、意図しない `id_token` を受け入れてしまう可能性がある。

**対策**

- `iss`: `.well-known/openid-configuration` で取得した `issuer` と、`id_token` の `iss` Claim が一致するか検証する
- `aud`: `id_token` の `aud` Claim に自身の `client_id` が含まれるか検証する
- `exp`: `id_token` の `exp` Claim が現在時刻より未来であることを検証する
- いずれかを満たさない場合は `id_token` を無効とする

---

### 6. セッション攻撃対策: TTL と使用後削除

**攻撃内容**

古い state や未使用のセッションエントリが長期間残存し、攻撃に利用される。

**対策**

- セッションエントリに TTL を設定し、一定時間後に自動削除する
- エントリが正常に使用された後は即座に削除し、再利用を防止する
