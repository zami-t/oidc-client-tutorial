# ADR: クライアント認証方式として client_secret_post と client_secret_basic をサポートする

## Status

Accepted

## Context

Token Endpoint へのリクエスト時、RP は OP に対してクライアント自身を認証する必要がある（RFC 6749 Section 2.3）。
主な方式として以下がある。

| 方式 | 概要 |
|---|---|
| `client_secret_post` | `client_secret` をリクエストボディに含める |
| `client_secret_basic` | HTTP Basic Auth ヘッダで送る |
| `client_secret_jwt` | `client_secret` で署名した JWT を送る |
| `private_key_jwt` | 秘密鍵で署名した JWT を送る |

本 RP は複数の OP に対応できる汎用実装を目指しており、OP ごとにサポートする認証方式が異なる場合がある。

## Decision

`client_secret_post` と `client_secret_basic` の2方式をサポートし、使用する方式はサーバー設定値で切り替える。

### 選定理由

- **広範な OP 互換性**: 主要な OP（Google、Keycloak 等）はどちらか一方または両方をサポートしており、設定で切り替えられることで幅広い OP に対応できる。
- **実装コストと効果のバランス**: `client_secret_jwt` / `private_key_jwt` は FAPI 等の高セキュリティ要件向けであり、現時点のスコープでは過剰。

## Consequences

- デフォルトは `client_secret_basic`。OP の設定で `client_secret_post` に切り替え可能。
- `client_secret_basic` では `client_id` / `client_secret` を URL エンコード後に Base64 エンコードする処理が必要（RFC 6749 Section 2.3.1）。
- 将来的に `private_key_jwt` 等が必要になった場合はこの ADR を更新して対応する。
