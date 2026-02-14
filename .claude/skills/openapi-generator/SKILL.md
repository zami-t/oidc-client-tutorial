---
name: openapi-generator
description: OpenAPI仕様（openapi-base.yml/openapi.yaml）の作成・分割・$ref整理のベストプラクティス。OpenAPIを新規作成/分割/リファクタするときに使う。
metadata:
  argument-hint: "[openapi-base.yml]"
---

# OpenAPI Generator

## 前提

- 仕様のルートは **openapi-base.yml**（リポジトリ直下）
- OpenAPIの分割ディレクトリ構造は、このスキルが **redocly split** を使って生成・更新する（手で「推奨構造」を維持しない）
- 生成されたファイルは **openapi/** 配下に置く（ルートに置かない）
- 生成されたファイルは **直接編集しない**（変更はルートのopenapi-base.ymlに加える。分割はあくまで構造化およびレビューのための出力で、編集対象はルートのみ）
- ルートのopenapi-base.ymlは、分割後も常に **完全なOpenAPI仕様** として機能する
- ルートのopenapi-base.ymlは、分割後も **lintエラーが0件** であることを目指す

## このスキルがやること

1. `openapi-base.yml` を基準に、分割(split)を行う
2. `openapi-base.yml` を lint し、**エラーが出なくなるまで**修正を繰り返す

## 使い方（コマンドは scripts に分離）

- Split: [scripts/split.sh](scripts/split.sh)
- Lint: [scripts/lint.sh](scripts/lint.sh)

## 自律的な lint 修正ループ（重要）

### ループ手順

1. `lint.sh` を実行し、エラー内容を取得する
2. エラーの原因となるOpenAPIファイル（`openapi-base.yml` または split 後の `openapi/` 配下）を特定する
3. エラーを修正する
4. `lint.sh` を再実行する
5. エラーが0件になるまで 1〜4 を繰り返す

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
