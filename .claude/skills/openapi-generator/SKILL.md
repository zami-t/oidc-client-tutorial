---
name: openapi-generator
description: OpenAPI仕様の編集・lint・bundleのベストプラクティス。OpenAPIを編集/検証/バンドルするときに使う。
metadata:
  argument-hint: "[openapi/openapi.yml]"
---

# OpenAPI Generator

## 前提

- OpenAPI仕様は **openapi/** 配下の複数ファイルで管理し、編集対象は **openapi/** 配下のファイル
- エントリポイントは **openapi/openapi.yml**（このファイルから paths/components などに $ref していく）
- 最終的に配布/生成に使う単一ファイルは **redocly bundle** で生成する

## このスキルがやること

1. `openapi/openapi.yml` を lint し、**エラーが出なくなるまで**修正を繰り返す
2. `openapi/openapi.yml` を bundle し、単一ファイル（例: `openapi/openapi.bundle.yaml`）を生成する

## 使い方（コマンドは scripts に分離）

- Lint: [scripts/lint.sh](scripts/lint.sh)
- Bundle: [scripts/bundle.sh](scripts/bundle.sh)

## 自律的な lint 修正ループ（重要）

### ループ手順

1. `scripts/lint.sh` を実行し、エラー内容を取得する
2. エラーの原因となるOpenAPIファイル（`openapi/` 配下）を特定する
3. エラーを修正する
4. `scripts/lint.sh` を再実行する
5. エラーが0件になるまで 1〜4 を繰り返す

lint が通ったら `scripts/bundle.sh` で単一ファイルにバンドルする。

### 修正の優先順位

- **参照解決の失敗（$ref / ファイルパス）** を最優先で直す（他の検証が進まない）
- 次に **必須フィールド不足 / 型の不一致 / 構文エラー**
- 最後に **スタイル・推奨（命名、説明文、unusedなど）**

### ユーザー確認が必要なとき

lint エラーが「単純な欠落の補完」ではなく、仕様の意味や要件に踏み込む場合は、勝手に決めずに確認する。

例:

- 認証方式（`securitySchemes` の種類、cookie/JWT/OAuth2など）をどうするか
- ステータスコード（`200`/`201`/`204`/`4xx`）の期待値
- エラーレスポンスのフォーマット統一方針
- enum/format/pattern/nullable など制約の強さ
- 互換性（既存クライアントがいる前提か、破壊的変更OKか）

確認テンプレ:

- 「lintが要求しているのはA/Bのどちらの意図ですか？」
- 「既存のレスポンス実装に合わせて X に寄せて良いですか？」
