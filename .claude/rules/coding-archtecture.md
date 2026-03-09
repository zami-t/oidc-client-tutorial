---
paths:
  - **/*.go
---

# Coding Architecture

DDD (Domain-Driven Design) を基盤とし、Onion Architecture を採用する。

---

## レイヤー構成

```
+--------------------+
| Presentation Layer  |
+--------------------+
          |
          v
+--------------------+      +--------------------+
| Usecase Layer       | ---> | Domain Layer      |
+--------------------+      +--------------------+
          ^                         ^
          |                         |
+--------------------+              |
| Infrastructure Layer| ------------+
+--------------------+
```

依存関係の向きは常に内側へのみ。

- Presentation → Usecase → Domain
- Infrastructure → Domain

実行時の組み立ては依存関係解決専用のファイルに切り出す。

各レイヤーの実装パターン詳細はスキルを参照:

- `/domain-layer` — model / service / port
- `/usecase-layer` — usecase / usecase/dto
- `/infra-layer` — repository / client
- `/presentation-layer` — handler / presentation/dto

---

## ディレクトリ構成

```
cmd/
├── main.go                       # エントリポイント
internal/
├── bootstrap/
│   └── wire.go                   # 依存関係解決の専用ファイル
├── presentation/
│   ├── handlers/
│   │   ├── login_handler.go
│   │   └── callback_handler.go
│   └── dto/
│       ├── login_request.go
│       └── login_response.go
├── domain/
│   ├── model/                    # Entity, Value Object
│   ├── service/                  # Domain Service
│   └── port/                     # Port Interface + センチネルエラー
├── usecase/                      # Usecase + センチネルエラー
│   └── dto/                      # Usecase 入出力 DTO
└── infrastructure/
    ├── client/
    │   ├── dto/
    │   └── xxx_client.go
    └── repository/
        ├── dto/
        └── xxx_repository.go
```

---

## エラーハンドリング規約

### エラーフロー全体図

```
Infrastructure (repository)
  └─ port.ErrXxx を fmt.Errorf でラップして返す
        ↓
Usecase
  └─ errors.Is で port.ErrXxx を検出し、自身の ErrXxx でラップして返す
        ↓
Presentation (handler)
  └─ errors.Is で usecase.ErrXxx を検出し、HTTP ステータスにマッピングする
```

### ポートファイルにセンチネルエラーを定義する

Infrastructure 層が返しうるエラーは、対応する **ポートファイルにセンチネルエラーとして定義**する。

```go
// domain/port/user_repository.go
var ErrUserNotFound = errors.New("user not found")
```

Infrastructure 実装はそれを `fmt.Errorf` でラップして返す。

```go
// infrastructure/repository/user_repository.go
if errors.Is(err, sql.ErrNoRows) {
    return model.User{}, fmt.Errorf("user %s: %w", id, port.ErrUserNotFound)
}
```

### ユースケースファイルにセンチネルエラーを定義する

各ユースケースが返しうるエラーは、**そのユースケースファイルの先頭に `var` ブロックで定義**する。

```go
// usecase/callback.go
var (
    ErrCallbackAuthorizationError      = errors.New("authorization error from OP")
    ErrCallbackStateMismatch           = errors.New("state mismatch")
    ErrCallbackTokenVerificationFailed = errors.New("token verification failed")
)
```

### Presentation 層は `errors.Is` でエラーを検出する

```go
// presentation/handler/helpers.go
switch {
case errors.Is(err, usecase.ErrCallbackStateMismatch),
    errors.Is(err, usecase.ErrCallbackAuthorizationError):
    writeJson(w, http.StatusForbidden, ...)
case errors.Is(err, usecase.ErrMeSessionNotFound):
    writeJson(w, http.StatusUnauthorized, ...)
default:
    writeJson(w, http.StatusInternalServerError, ...)
}
```

---

## Bootstrap（依存関係解決）

`internal/bootstrap/wire.go` に全レイヤーの組み立てを集約する。

```go
package bootstrap

func InitializeApp(db *sql.DB, apiBaseURL string) *handler.UserRegistrationHandler {
    // Infrastructure Layer
    userRepo := repository.NewUserRepository(db)
    notificationCli := client.NewExternalAPIClient(apiBaseURL)

    // Domain Service
    emailService := service.NewEmailService(userRepo)

    // Usecase Layer
    registrationUC := usecase.NewUserRegistrationUsecase(
        userRepo,
        emailService,
        notificationCli,
    )

    // Presentation Layer
    return handler.NewUserRegistrationHandler(registrationUC)
}
```
