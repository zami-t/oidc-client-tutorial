# ログ設計

## 相関ID設計

- 各リクエスト受信時に TraceId（16バイト、UUID v4等）を発行する
- TraceId はリクエスト全体を通じて一意に識別する
- グリーンスレッド生成時に SpanId（8バイト）を新たに発行する
- ルート処理の SpanId を ParentSpanId として子スレッドに引き継ぐ

## 出力設定

- 標準出力にログを出力する
- 形式は JSON

## ログレベル設計

| レベル | 用途                                                     | 対応                                       |
| ------ | -------------------------------------------------------- | ------------------------------------------ |
| INFO   | オブザーバビリティのための情報                           | 定期確認                                   |
| WARN   | 単体では軽微だが、頻発した場合に障害の予兆となりうる事象 | 監視基盤での頻度アラートと組み合わせて運用 |
| ERROR  | 即時障害調査が必要な事象                                 | 即時対応                                   |

### ログレベル判断基準

**ユーザー起因のエラーは WARN にしない。**

WARN は「頻発時にアラートを鳴らす」運用と組み合わせることで意味を持つ。ユーザーが不正なパラメータを送ってきた場合（バリデーションエラー、必須パラメータ欠損など）は、アプリケーション側が検知・対応すべき障害の予兆ではなく、通常運用上の事象である。そのため INFO で記録する。

- ユーザー入力バリデーション失敗 → **INFO**
- システム内部の一時的な異常（外部サービス障害、タイムアウトなど） → **WARN**
- 即時調査が必要な障害 → **ERROR**

## フィールド定義

### 必須フィールド

| フィールド       | 形式・値                      | 説明                                |
| ---------------- | ----------------------------- | ----------------------------------- |
| `timestamp`      | RFC 3339（ナノ秒精度推奨）    | ログ出力時刻                        |
| `level`          | `DEBUG / INFO / WARN / ERROR` | ログレベル                          |
| `service`        | 文字列                        | サービス名                          |
| `version`        | 文字列                        | サービスバージョン                  |
| `trace_id`       | 32文字 hex                    | リクエスト単位の識別子              |
| `span_id`        | 16文字 hex                    | 処理単位の識別子                    |
| `parent_span_id` | 16文字 hex                    | 親スレッドの SpanId（ルートは省略） |
| `message`        | 文字列                        | 人間が読める説明文                  |

### エラー時の追加フィールド

| フィールド    | 説明                             |
| ------------- | -------------------------------- |
| `error_code`  | アプリ独自のエラー識別子         |
| `error`       | エラーメッセージ                 |
| `stack_trace` | スタックトレース（エラー時のみ） |

## セキュリティ

- パスワード・トークン・個人情報（メールアドレス等）はログに出力しない、または `***` でマスクする
- IDトークン・Authorization ヘッダは絶対に出力しない

## 運用

- ログレベルは再起動なしに動的変更できることが望ましい
- 本番環境では DEBUG ログのサンプリング方針を設計段階で決める

## ログ出力例

```json
{
  "timestamp": "2026-03-14T12:00:00.123456789Z",
  "level": "ERROR",
  "service": "oidc-client",
  "version": "1.0.0",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "parent_span_id": "00f067aa0ba902b6",
  "message": "token exchange failed",
  "error_code": "TOKEN_EXCHANGE_FAILED",
  "error": "unexpected status code: 400",
  "stack_trace": "..."
}
```

## 参考

- [OpenTelemetry Log Data Model](https://opentelemetry.io/docs/specs/otel/logs/data-model/)
- [OpenTelemetry Tracing API](https://opentelemetry.io/docs/specs/otel/trace/api/)
- [RFC 3339 - Date and Time on the Internet](https://www.rfc-editor.org/rfc/rfc3339)
