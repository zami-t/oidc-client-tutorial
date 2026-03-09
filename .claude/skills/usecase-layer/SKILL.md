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

**パターン例**

```go
package usecase

import (
    "context"
    "example.com/project/internal/domain/model"
    "example.com/project/internal/domain/port"
    "example.com/project/internal/domain/service"
    "example.com/project/internal/usecase/dto"
)

// センチネルエラーはユースケースファイルの先頭に定義する
var (
    ErrUserRegistrationEmailDuplicate = errors.New("email already registered")
    ErrUserRegistrationInvalidInput   = errors.New("invalid input")
)

type UserRegistrationUsecase struct {
    userRepo        port.UserRepository
    emailService    *service.EmailService
    notificationCli port.NotificationClient
}

func NewUserRegistrationUsecase(
    userRepo port.UserRepository,
    emailService *service.EmailService,
    notificationCli port.NotificationClient,
) *UserRegistrationUsecase {
    return &UserRegistrationUsecase{
        userRepo:        userRepo,
        emailService:    emailService,
        notificationCli: notificationCli,
    }
}

// Usecase メソッドは DTO を受け取り、DTO を返す
func (u *UserRegistrationUsecase) Execute(ctx context.Context, input dto.UserRegistrationInput) (dto.UserRegistrationOutput, error) {
    // 1. 入力 DTO を Domain の型に変換
    email, err := model.NewEmail(input.Email)
    if err != nil {
        return dto.UserRegistrationOutput{}, fmt.Errorf("invalid email: %w", ErrUserRegistrationInvalidInput)
    }

    userID := model.NewUserID()
    user := model.NewUser(userID, email, input.Name)

    // 2. Domain Service を使ったビジネスロジック
    if err := u.emailService.ValidateUniqueness(ctx, email); err != nil {
        if errors.Is(err, port.ErrEmailAlreadyExists) {
            return dto.UserRegistrationOutput{}, fmt.Errorf("email %s: %w", input.Email, ErrUserRegistrationEmailDuplicate)
        }
        return dto.UserRegistrationOutput{}, fmt.Errorf("failed to validate email: %w", err)
    }

    // 3. Repository への保存
    if err := u.userRepo.Save(ctx, user); err != nil {
        return dto.UserRegistrationOutput{}, fmt.Errorf("failed to save user: %w", err)
    }

    // 4. Domain の型を DTO に変換して返す
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

**パターン例**

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
