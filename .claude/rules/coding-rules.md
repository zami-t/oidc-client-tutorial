---
paths: **/*.go
---

# Go Coding Rules

このドキュメントは、Goの公式スタイルガイドとコミュニティのベストプラクティスに基づいたコーディングルールです。

**参照元**
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Style Guide (Google)](https://google.github.io/styleguide/go/)
- [Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Go Style Best Practices](https://google.github.io/styleguide/go/best-practices.html)

---

## 必須ツール

### コミット前に必ず実行

```bash
# フォーマット + インポート整理（必須）
goimports -w .

# Lint（推奨）
golangci-lint run

# テスト
go test ./...

# Vet（静的解析）
go vet ./...
```

---

## フォーマット

### goimports を使う

- `goimports` は `gofmt` の全機能 + インポート管理を提供する
- コードフォーマット（`gofmt` 相当）とインポート整理を同時に実行
- エディタの保存時フックは `goimports` を設定すれば十分

```bash
# インストール
go install golang.org/x/tools/cmd/goimports@latest

# 実行
goimports -w .
```

**`goimports` が行うこと**
1. `gofmt` と同じコードフォーマット
2. 未使用のインポートを削除
3. 不足しているインポートを自動追加
4. インポートをグループ化（標準ライブラリ / 外部 / 内部）

**参照**: [goimports documentation](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)

---

## 命名規則

### パッケージ名

```go
// ✅ Good: 小文字、短く、明確
package user
package http
package auth

// ❌ Bad: アンダースコア、大文字、複数形、汎用的すぎる名前
package user_service
package HTTP
package users
package common
package util
```

**ルール**
- 小文字のみ（アンダースコアや mixedCaps は使わない）
- 短く簡潔に
- 複数形にしない（`net/url` であって `net/urls` ではない）
- 汎用的すぎる名前を避ける（`common`, `util`, `shared`, `lib`）

**参照**: [Effective Go - Package names](https://go.dev/doc/effective_go#package-names)

### 変数・関数名

```go
// ✅ Good: MixedCaps（定数も含む）
const MaxRetries = 3
var userCount int
func GetUserByID(id string) User

// ❌ Bad: アンダースコア、ALL_CAPS
const MAX_RETRIES = 3
var user_count int
func get_user_by_id(id string) User
```

**ルール**
- MixedCaps（キャメルケース）を使用
- 定数も MixedCaps（`MAX_RETRIES` ではなく `MaxRetries`）
- 1文字変数は避ける（ただし `i` などの慣習的なものは除く）
- 長すぎる名前も短すぎる名前も避ける

**参照**: [Effective Go - MixedCaps](https://go.dev/doc/effective_go#mixed-caps)

### 頭字語（Acronym）の扱い

```go
// ✅ Good: 頭字語の最初の文字だけ大文字
type ApiClient struct{}  // 公開
type apiConfig struct{}  // 非公開

// ❌ Bad: 全て大文字
type APIClient struct{}
type APIConfig struct{}
```

**ルール**
- 頭字語は最初の文字のみ大文字にする
- 例: `someApiUrl` ではなく `someAPIURL` は避け、`someApiUrl` を使う

---

## インターフェース

### 小さなインターフェース

```go
// ✅ Good: 小さく明確な責務
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Writer interface {
    Write(p []byte) (n int, err error)
}

// 必要に応じて合成
type ReadWriter interface {
    Reader
    Writer
}
```

**ルール**
- インターフェースは小さく保つ（1-3メソッド）
- 必要なメソッドだけを含める
- 大きなインターフェースは合成で作る

**参照**: [Go Style Best Practices - Interfaces](https://google.github.io/styleguide/go/best-practices.html)

### インターフェースは消費側で定義

```go
// ❌ Bad: 提供側でインターフェースを定義
// package user
type UserRepository interface {
    Save(User) error
    FindByID(string) (User, error)
}

// ✅ Good: 消費側で必要なメソッドだけ定義
// package service
type userSaver interface {
    Save(User) error
}

func (s *Service) RegisterUser(u User) error {
    return s.userSaver.Save(u)
}
```

**ルール**
- インターフェースは使う側（消費側）で定義する
- 提供側でインターフェースを先に定義しない

**参照**: "Accept interfaces, return structs"

---

## エラーハンドリング

### エラーは無視しない

```go
// ✅ Good
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// ❌ Bad: エラーを無視
_ = doSomething()
```

### エラーラップ

```go
// ✅ Good: %w でエラーをラップ
if err := db.Save(user); err != nil {
    return fmt.Errorf("failed to save user %s: %w", user.ID, err)
}

// ❌ Bad: %v でエラーチェーンが途切れる
if err := db.Save(user); err != nil {
    return fmt.Errorf("failed to save user: %v", err)
}
```

**ルール**
- エラーは常に処理する
- `%w` でエラーをラップし、コンテキストを追加する
- スタックトレースを保持する

**参照**: [Go 1.13 Error wrapping](https://go.dev/blog/go1.13-errors)

---

## 並行処理

### Goroutine リークを防ぐ

```go
// ✅ Good: context で goroutine のライフサイクルを管理
func processData(ctx context.Context, data <-chan string) <-chan Result {
    results := make(chan Result)
    go func() {
        defer close(results)
        for {
            select {
            case item := <-data:
                results <- process(item)
            case <-ctx.Done():
                return // 明示的な終了条件
            }
        }
    }()
    return results
}
```

**ルール**
- goroutine には必ず終了条件を設ける
- `context.Context` でライフサイクルを制御する
- `defer close(ch)` でチャネルを閉じる

**参照**: [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)

---

## テスト

### Table-Driven Tests

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 1, 2, 3},
        {"negative numbers", -1, -2, -3},
        {"zero", 0, 0, 0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Add(tt.a, tt.b)
            if result != tt.expected {
                t.Errorf("Add(%d, %d) = %d, want %d", 
                    tt.a, tt.b, result, tt.expected)
            }
        })
    }
}
```

**ルール**
- テストケースが複数ある場合は table-driven を使う
- `t.Run()` でサブテストを実行
- 並列実行可能なテストは `t.Parallel()` を追加

**参照**: [Go Wiki - TableDrivenTests](https://go.dev/wiki/TableDrivenTests)

---

## コメント

### パッケージコメント

```go
// Package user provides user management functionality.
// It handles user registration, authentication, and profile updates.
package user
```

**ルール**
- パッケージの最初にパッケージコメントを書く
- 何を提供するかを簡潔に説明

### 公開API コメント

```go
// GetUserByID retrieves a user by their unique identifier.
// It returns an error if the user is not found or if a database error occurs.
func GetUserByID(id string) (User, error) {
    // ...
}
```

**ルール**
- 公開される関数・型・定数には必ずコメントを書く
- コメントは対象の名前で始める
- 完全な文章で書く

**参照**: [Effective Go - Commentary](https://go.dev/doc/effective_go#commentary)

---

## 構造体とポインタ

### ポインタを使う基準

```go
// ✅ Good: 大きな構造体はポインタ
type LargeStruct struct {
    data [10000]byte
}

func Process(ls *LargeStruct) {} // ポインタで渡す

// ✅ Good: 小さな構造体は値
type Point struct {
    X, Y int
}

func Distance(p1, p2 Point) float64 {} // 値で渡す
```

**ポインタを使う場合**
- 構造体が大きい（コピーコストが高い）
- 関数内で構造体を変更する必要がある
- nil を返す必要がある

**値を使う場合**
- 構造体が小さい
- 不変性を保ちたい

---

## Import の整理

### Import グループ

```go
import (
    // 1. 標準ライブラリ
    "context"
    "fmt"
    "net/http"

    // 2. 外部ライブラリ
    "github.com/some/external/package"
    
    // 3. 内部パッケージ
    "example.com/project/internal/domain"
    "example.com/project/internal/usecase"
)
```

**ルール**
- 標準ライブラリ → 外部 → 内部の順
- 各グループ間は空行で区切る
- `goimports` が自動で整理する

**参照**: [Go Style Decisions - Import grouping](https://google.github.io/styleguide/go/decisions#import-grouping)

---

## その他のベストプラクティス

### 短い変数宣言

```go
// ✅ Good: := を使う
user := User{Name: "Alice"}

// ❌ Bad: 不必要な var
var user User = User{Name: "Alice"}
```

### defer を活用

```go
func processFile(filename string) error {
    f, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer f.Close() // 関数終了時に必ずクローズ

    // 処理...
    return nil
}
```

### 空の struct を使う

```go
// ✅ Good: signal として使う場合
done := make(chan struct{})
close(done)

// ✅ Good: set として使う場合
seen := make(map[string]struct{})
seen["key"] = struct{}{}
```

---

## 禁止事項

### panic は使わない

```go
// ❌ Bad: panic を使う
if err != nil {
    panic(err)
}

// ✅ Good: エラーを返す
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

**例外**: `init()` や不可能な状態でのみ許容

### init() の濫用を避ける

```go
// ❌ Bad: 複雑な初期化を init() で行う
func init() {
    db = connectDatabase()
    cache = setupCache()
}

// ✅ Good: 明示的な初期化関数
func New() (*Service, error) {
    db, err := connectDatabase()
    if err != nil {
        return nil, err
    }
    return &Service{db: db}, nil
}
```

---

## Linter 設定（推奨）

### golangci-lint

`.golangci.yml` で以下の linter を有効化：

```yaml
linters:
  enable:
    - errcheck      # エラーチェック漏れ検出
    - gosimple      # コードの簡略化提案
    - govet         # go vet の実行
    - ineffassign   # 無駄な代入検出
    - staticcheck   # 静的解析
    - unused        # 未使用コード検出
    - goimports     # インポートとフォーマットチェック
```

**参照**: [golangci-lint](https://golangci-lint.run/)

---

## まとめ

**必ず守るルール**
1. `goimports` でフォーマット + インポート整理
2. エラーは必ず処理
3. 公開APIにはコメント
4. MixedCaps で命名
5. `golangci-lint` の使用

**推奨ルール**
1. 小さなインターフェース
2. Table-driven tests
3. goroutine のライフサイクル管理
4. Effective Go の参照
5. エディタ保存時に `goimports` 実行
