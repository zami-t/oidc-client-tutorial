# CLAUDE.md

このファイルは、このリポジトリで作業する Claude Code へのガイダンスを提供する。

## プロジェクト概要

OIDC 関連の外部ライブラリを使わず Go で実装した OIDC Relying Party（Authorization Code Flow）。学習・理解目的。Go 1.25+、モジュール名 `oidc-tutorial`。詳細な仕様は `.claude/rules/` にある。

@README.md も参照。

## 正規ドキュメント（信頼できる唯一の情報源）

正規の情報源は `.claude/rules/` 配下にある。このファイルと以下のドキュメントが矛盾する場合は、**`.claude/rules/` を優先する**。

- `.claude/rules/` — アーキテクチャ・セキュリティ・コーディング規約・Git 運用など、設計・実装の規範となるルール群

## ドキュメント編集ルール

`docs/`、`rules/`、`CLAUDE.md` を編集した後は必ず `/check-docs` を実行して整合性を確認すること。

## アーキテクチャ意思決定記録（ADR）

設計上の意思決定は `.claude/decisions/` 配下に記録する。

- `.claude/decisions/` — 設計上の意思決定とその背景・理由の記録（ADR）

## ビルド・開発コマンド

```bash
# ビルド
go build ./...

# テスト実行
go test ./...

# 単一テストの実行
go test ./internal/domain/model -run TestFunctionName

# フォーマット + インポート整理（コミット前に必ず実行）
goimports -w .

# Lint
golangci-lint run

# 静的解析
go vet ./...
```

必要ツールのインストール:
```bash
go install golang.org/x/tools/cmd/goimports@latest
```
