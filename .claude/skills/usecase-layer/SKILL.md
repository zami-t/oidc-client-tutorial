---
name: usecase-layer
description: Usecase層のコードを実装するときに使う。ユースケースの追加・修正、usecase/dto の追加・修正するとき。
---

# Usecase Layer 実装ガイド

Usecase は、Domain オブジェクト/Domain サービス/ポート呼び出しを組み合わせてユースケースを達成する。

```
+--------------------+
| Presentation Layer  |
+--------------------+
          |
          v
+--------------------+      +--------------------+
| Usecase Layer       | ---> | Domain Layer      |
+--------------------+      +--------------------+
```

**対象ディレクトリ**

```
internal/usecase/
├── xxx.go        # ユースケース実装（+ センチネルエラー）
└── dto/          # 入出力 DTO
    ├── xxx_input.go
    └── xxx_output.go
```

---

## usecase/ - Usecase（Orchestration）

**責務**
- Domain Service やポート（Repository/Client）の呼び出しを組み合わせる
- 入力 DTO を Domain の型に変換し、ユースケースを実行する
- 各ユースケースが返しうるエラーをファイル先頭の `var` ブロックで定義する
- port.ErrXxx を `errors.Is` で検出し、自身の ErrXxx でラップして返す

**ログの責務**: `.claude/rules/logging.md` 参照

**パターン例**

```go
package usecase

import (
    "context"
    "errors"
    "fmt"
    "example.com/project/internal/domain/model"
    "example.com/project/internal/domain/port"
    "example.com/project/internal/domain/service"
    "example.com/project/internal/logger"
    "example.com/project/internal/usecase/dto"
)

// センチネルエラーはユースケースファイルの先頭に定義する
var (
    ErrUserRegistrationEmailDuplicate = errors.New("email already registered")
)

type UserRegistrationUsecase struct {
    userRepo        port.UserRepository
    emailService    *service.EmailService
    notificationCli port.NotificationClient
    log             *logger.Logger
}

func NewUserRegistrationUsecase(
    userRepo port.UserRepository,
    emailService *service.EmailService,
    notificationCli port.NotificationClient,
    log *logger.Logger,
) *UserRegistrationUsecase {
    return &UserRegistrationUsecase{
        userRepo:        userRepo,
        emailService:    emailService,
        notificationCli: notificationCli,
        log:             log,
    }
}

// Usecase メソッドは DTO を受け取り、DTO を返す
func (u *UserRegistrationUsecase) Execute(ctx context.Context, input dto.UserRegistrationInput) (dto.UserRegistrationOutput, error) {
    // 1. 入力 DTO を Domain の型に変換
    email, err := model.NewEmail(input.Email)
    if err != nil {
        return dto.UserRegistrationOutput{}, fmt.Errorf("invalid email: %w", err)
    }

    userID := model.NewUserID()
    user := model.NewUser(userID, email, input.Name)

    // 2. Domain Service を使ったビジネスロジック
    if err := u.emailService.ValidateUniqueness(ctx, email); err != nil {
        if errors.Is(err, port.ErrEmailAlreadyExists) {
            // ユーザー起因のエラー → Info
            u.log.Info(ctx, "user-registration: email already registered")
            return dto.UserRegistrationOutput{}, fmt.Errorf("email %s: %w", input.Email, ErrUserRegistrationEmailDuplicate)
        }
        // システムエラー → Error
        wrapped := fmt.Errorf("failed to validate email uniqueness: %w", err)
        u.log.Error(ctx, "user-registration: failed to validate email", "USER_REGISTRATION_EMAIL_VALIDATION_FAILED", wrapped)
        return dto.UserRegistrationOutput{}, wrapped
    }

    // 3. Repository への保存
    if err := u.userRepo.Save(ctx, user); err != nil {
        wrapped := fmt.Errorf("failed to save user: %w", err)
        u.log.Error(ctx, "user-registration: failed to save user", "USER_REGISTRATION_SAVE_FAILED", wrapped)
        return dto.UserRegistrationOutput{}, wrapped
    }

    // 4. 正常完了ログ
    u.log.Info(ctx, "user-registration: user registered")

    // 5. Domain の型を DTO に変換して返す
    return dto.UserRegistrationOutput{
        UserID: user.ID().String(),
        Email:  user.Email().String(),
    }, nil
}
```

---

## usecase/dto/ - Usecase の入出力 DTO

**責務**
- Usecase の入力と出力を表現する
- Presentation Layer が Domain の型を直接知らないようにする

**パターン例（シンプル）**

フィールドが少ない、または型が異なり取り違えのリスクがない場合はそのまま構造体リテラルで生成してよい。

```go
package dto

// 入力 DTO
type UserRegistrationInput struct {
    Email string
    Name  string
}

// 出力 DTO
type UserRegistrationOutput struct {
    UserID string
    Email  string
}
```

**パターン例（Params + バリデーション付きコンストラクタ）**

同じ型のフィールドが複数ある入力 DTO は、呼び出し側でパラメータを取り違えるリスクがある。
この場合は **生の値を受け取る `XxxParams` 構造体**と、**バリデーション済みの `XxxInput` 構造体**を分離し、
`NewXxxInput(p XxxParams)` を唯一の生成経路にすることでバリデーション呼び忘れを防ぐ。

```go
package dto

// XxxParams は Presentation 層が HTTP リクエストから組み立てる生の値の入れ物。
type XxxParams struct {
    FieldA string
    FieldB string
    FieldC string
}

// XxxInput はバリデーション済みの Usecase 入力。NewXxxInput 経由でのみ生成できる。
type XxxInput struct {
    FieldA string
    FieldB string
    FieldC string
}

// NewXxxInput は params を検証し XxxInput を返す。
// バリデーションエラーは Presentation 層で 400 にマッピングする。
func NewXxxInput(p XxxParams) (XxxInput, error) {
    if p.FieldA == "" {
        return XxxInput{}, errors.New("FieldA is required")
    }
    return XxxInput{
        FieldA: p.FieldA,
        FieldB: p.FieldB,
        FieldC: p.FieldC,
    }, nil
}
```

呼び出し側（Presentation 層）:

```go
input, err := dto.NewXxxInput(dto.XxxParams{
    FieldA: q.Get("field_a"),
    FieldB: q.Get("field_b"),
    FieldC: q.Get("field_c"),
})
if err != nil {
    writeJson(w, http.StatusBadRequest, errorResponse{...})
    return
}
```

**どちらを使うか**

| 条件 | パターン |
|------|---------|
| フィールドが少ない / 型が異なる | 構造体リテラルで直接生成 |
| 同じ型のフィールドが複数ある | Params + コンストラクタ |
