---
paths: **/*.go
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

各レイヤーの責務の詳細は、以降の各セクションに記載する。

## Presentation Layer

Presentation Layer は、外部（HTTP）とアプリケーションの境界に位置する薄い層とする。
入力を受け取り必要最小限の変換を行って Usecase に委譲し、結果を HTTP レスポンスとして返す。
ビジネスロジックは一切ハンドラーに置かない。

**responsibilities**

- リクエストの受け取りと Usecase Layer へのパラメータの受け渡し
- Usecase Layer からの戻り値を HTTP レスポンスとして返す
- エラーの場合は適切なHTTPステータスコードで返す

---

## Usecase Layer

Usecase は、1つ以上の Domain オブジェクト/Domainサービス/ポート呼び出しを組み合わせて「アプリケーションとしての目的（ユースケース）」を達成する。

**responsibilities**

- 入力（DTOなど）を Domain の型/値に変換し、ユースケースを実行する
- Domain のメソッド/関数の呼び出し順序や分岐を制御する
- ポート（Repository/Client）を通じて外部I/Oを行う
- Domain のエラーを、Presentation が扱いやすい形に整理する
- トランザクション境界（DBを使う場合の begin/commit/rollback の責務分離）

---

## Domain Layer

Domain はシステムの中心（Core）であり、他レイヤーから参照されうるが、他レイヤーへは依存しない。

**responsibilities**

- ドメインの概念を型として表現する（Value Object / Entity）
- 不変条件・整合性をモデル内に閉じ込める（生成時検証、メソッドの事前条件/事後条件など）
- 外部I/Oへの依存を抽象化する（必要なポートのインターフェースを定義する）

---

## Infrastructure Layer

Infrastructure は Domain/Usecase が定義したポートに対する「外部I/Oの具体実装」を置く。

**responsibilities**

- Repository/Client などのポート実装を提供する（DB/KVS/HTTP 等）
- 外部システム固有の表現を、Domain の型に変換して返す（DTO → Domain）
- タイムアウト、リトライ、接続設定など「外部I/Oの運用上の詳細」を扱う

---

## ディレクトリ構成（例）

```
cmd/
├── main.go                       # エントリポイント
internal/
├── bootstrap/
│   └── wire.go                   # 依存関係解決の専用ファイル
├── presentation/
│   ├── handlers/                 # handler一覧
│   │   ├── login_handler.go
│   │   └── callback_handler.go
│   └── dto/                      # handlerで使用するrequest/responseのDTO
│       ├── login_request.go
│       └── login_response.go
├── domain/
│   ├── model/                    # Entity, Value Object
│   │   ├── authorization.go
│   │   ├── token.go
│   │   └── provider.go
│   ├── service/                  # domain service
│   │   └── authorization_service.go
│   └── port/                     # port interfaceの定義
│       ├── session_repository.go
│       ├── token_fetcher.go
│       └── jwks_fetcher.go
├── usecase/                      # usecase(オーケストレーション)
│   ├── auth_service.go
│   └── token_service.go
│   └── dto/                      # usecaseの入出力用のDTO
│       ├── login_param.go
│       └── login_result.go
└── infrastructure/
    ├── client/
	│   ├── dto/                  # clientで使用するrequest/responseのDTO
	│   │  ├── token_fetch_request.go
	│   │  └── token_fetch_response.go
    │   ├── token_fetcher.go      # client実装
    │   └── jwks_fetcher.go
    └── repository/
        ├── dto/                  # clientで使用するrequest/responseのDTO
        │  ├── token_fetch_request.go
        │  └── token_fetch_response.go
        └── memory_session_repository.go # repository実装
```

## レイヤー別実装パターン

### Domain Layer

Domain Layer では、ビジネスの概念を型として表現し、外部依存を抽象化する。

#### model/ - Value Object / Entity

**責務**
- ドメインの概念を不変な型として表現する
- 生成時にバリデーションを行い、不正な状態を作らせない
- 他の型への依存は最小限にする

**パターン例**

```go
package model

// Value Object の例
type Email struct {
    value string
}

// コンストラクタでバリデーションを行い、不正な Email を作らせない
func NewEmail(email string) (Email, error) {
    if !isValidEmail(email) {
        return Email{}, errors.New("invalid email format")
    }
    return Email{value: email}, nil
}

func (e Email) String() string {
    return e.value
}

// Entity の例
type User struct {
    id    UserID
    email Email
    name  string
}

func NewUser(id UserID, email Email, name string) User {
    return User{id: id, email: email, name: name}
}

func (u User) ID() UserID {
    return u.id
}

func (u User) Email() Email {
    return u.email
}
```

#### service/ - Domain Service

**責務**
- 複数の Entity/Value Object にまたがるロジックを扱う
- 単一の Entity に属さないドメインロジックを実装する
- ポート（インターフェース）に依存してもよい

**パターン例**

```go
package service

// ドメインサービスは、複数のモデルや外部依存（ポート）を組み合わせる
type AuthorizationService struct {
    randomGenerator RandomGenerator
}

func NewAuthorizationService(randomGen RandomGenerator) *AuthorizationService {
    return &AuthorizationService{randomGenerator: randomGen}
}

// ドメインロジックをメソッドとして実装
func (s *AuthorizationService) GenerateState() (string, error) {
    return s.randomGenerator.Generate(32)
}
```

#### port/ - Port (Interface)

**責務**
- 外部システム（DB, HTTP, KVS）への依存を抽象化する
- Domain が外部実装の詳細を知らないようにする
- 戻り値は Domain の型のみを使う

**パターン例**

```go
package port

import (
    "context"
    "example.com/project/internal/domain/model"
)

// Repository のポート
type UserRepository interface {
    Save(ctx context.Context, user model.User) error
    FindByID(ctx context.Context, id model.UserID) (model.User, error)
}

// Client のポート
type ExternalAPIClient interface {
    FetchData(ctx context.Context, req DataRequest) (model.Data, error)
}

// ポートの入力パラメータは、Domain 内で定義する
type DataRequest struct {
    Endpoint string
    Params   map[string]string
}
```

---

### Usecase Layer

Usecase Layer では、Domain の機能を組み合わせて「アプリケーションとしてのユースケース」を実現する。

#### usecase/ - Usecase (Orchestration)

**責務**
- Domain Service やポート（Repository/Client）の呼び出しを組み合わせる
- 複数のドメインオブジェクトの操作順序を制御する
- トランザクション境界を管理する（DBを使う場合）

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
        return dto.UserRegistrationOutput{}, err
    }

    userID := model.NewUserID()
    user := model.NewUser(userID, email, input.Name)

    // 2. Domain Service を使ったビジネスロジック
    if err := u.emailService.ValidateUniqueness(ctx, email); err != nil {
        return dto.UserRegistrationOutput{}, err
    }

    // 3. Repository への保存
    if err := u.userRepo.Save(ctx, user); err != nil {
        return dto.UserRegistrationOutput{}, err
    }

    // 4. 外部システムへの通知
    if err := u.notificationCli.NotifyUserRegistered(ctx, user); err != nil {
        // ログのみで続行（補助的な処理）
        log.Printf("failed to notify: %v", err)
    }

    // 5. Domain の型を DTO に変換して返す
    return dto.UserRegistrationOutput{
        UserID: user.ID().String(),
        Email:  user.Email().String(),
    }, nil
}
```

#### usecase/dto/ - Usecase の入出力 DTO

**責務**
- Usecase の入力と出力を表現する
- Presentation Layer や Infrastructure Layer が Domain の型を直接知らないようにする

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

---

### Infrastructure Layer

Infrastructure Layer では、ポートの具体実装を提供する。

#### repository/ - Repository の実装

**責務**
- Domain の Repository ポートを実装する
- DB/KVS 固有の処理を行う
- Infrastructure の DTO と Domain Model の相互変換を行う

**パターン例**

```go
package repository

import (
    "context"
    "database/sql"
    "example.com/project/internal/domain/model"
    "example.com/project/internal/domain/port"
    "example.com/project/internal/infrastructure/repository/dto"
)

type userRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) port.UserRepository {
    return &userRepository{db: db}
}

func (r *userRepository) Save(ctx context.Context, user model.User) error {
    // Domain Model → Infrastructure DTO
    userDTO := dto.UserDTO{
        ID:    user.ID().String(),
        Email: user.Email().String(),
        Name:  user.Name(),
    }

    // DB への保存処理
    _, err := r.db.ExecContext(ctx,
        "INSERT INTO users (id, email, name) VALUES (?, ?, ?)",
        userDTO.ID, userDTO.Email, userDTO.Name,
    )
    return err
}

func (r *userRepository) FindByID(ctx context.Context, id model.UserID) (model.User, error) {
    var userDTO dto.UserDTO
    err := r.db.QueryRowContext(ctx,
        "SELECT id, email, name FROM users WHERE id = ?",
        id.String(),
    ).Scan(&userDTO.ID, &userDTO.Email, &userDTO.Name)
    
    if err != nil {
        return model.User{}, err
    }

    // Infrastructure DTO → Domain Model
    email, err := model.NewEmail(userDTO.Email)
    if err != nil {
        return model.User{}, err
    }

    userID, err := model.NewUserIDFromString(userDTO.ID)
    if err != nil {
        return model.User{}, err
    }

    return model.NewUser(userID, email, userDTO.Name), nil
}
```

#### repository/dto/ - Repository 固有の DTO

**責務**
- DB のレコード構造を表現する
- ORM のマッピング対象となる型

**パターン例**

```go
package dto

type UserDTO struct {
    ID    string
    Email string
    Name  string
}
```

#### client/ - External API Client の実装

**責務**
- Domain の Client ポートを実装する
- HTTP/gRPC などの外部通信を行う
- Infrastructure の DTO と Domain Model の相互変換を行う

**パターン例**

```go
package client

import (
    "context"
    "encoding/json"
    "net/http"
    "example.com/project/internal/domain/model"
    "example.com/project/internal/domain/port"
    "example.com/project/internal/infrastructure/client/dto"
)

type externalAPIClient struct {
    httpClient *http.Client
    baseURL    string
}

func NewExternalAPIClient(baseURL string) port.ExternalAPIClient {
    return &externalAPIClient{
        httpClient: &http.Client{Timeout: 10 * time.Second},
        baseURL:    baseURL,
    }
}

func (c *externalAPIClient) FetchData(ctx context.Context, req port.DataRequest) (model.Data, error) {
    // HTTP リクエストの構築と送信
    httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+req.Endpoint, nil)
    if err != nil {
        return model.Data{}, err
    }

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return model.Data{}, err
    }
    defer resp.Body.Close()

    // Infrastructure DTO へのデコード
    var respDTO dto.DataResponse
    if err := json.NewDecoder(resp.Body).Decode(&respDTO); err != nil {
        return model.Data{}, err
    }

    // Infrastructure DTO → Domain Model
    return model.NewData(respDTO.Value), nil
}
```

#### client/dto/ - Client 固有の DTO

**責務**
- 外部 API のレスポンス構造を表現する

**パターン例**

```go
package dto

type DataResponse struct {
    Value string `json:"value"`
}
```

---

### Presentation Layer

Presentation Layer では、外部からのリクエストを受け取り、Usecase に委譲する。

#### handler/ - HTTP Handler

**責務**
- HTTP リクエストを受け取る
- リクエストから必要な値を抽出し、Usecase の DTO に変換する
- Usecase を呼び出し、結果を HTTP レスポンスとして返す
- エラーハンドリングと適切なステータスコードの返却

**パターン例**

```go
package handler

import (
    "encoding/json"
    "net/http"
    "example.com/project/internal/usecase"
    "example.com/project/internal/usecase/dto"
    presentationDTO "example.com/project/internal/presentation/dto"
)

type UserRegistrationHandler struct {
    usecase *usecase.UserRegistrationUsecase
}

func NewUserRegistrationHandler(uc *usecase.UserRegistrationUsecase) *UserRegistrationHandler {
    return &UserRegistrationHandler{usecase: uc}
}

func (h *UserRegistrationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 1. リクエストボディを Presentation DTO にパース
    var req presentationDTO.UserRegistrationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request body", http.StatusBadRequest)
        return
    }

    // 2. Presentation DTO → Usecase DTO
    input := dto.UserRegistrationInput{
        Email: req.Email,
        Name:  req.Name,
    }

    // 3. Usecase の実行
    output, err := h.usecase.Execute(r.Context(), input)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // 4. Usecase DTO → Presentation DTO
    resp := presentationDTO.UserRegistrationResponse{
        UserID: output.UserID,
        Email:  output.Email,
    }

    // 5. レスポンスの返却
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(resp)
}
```

#### presentation/dto/ - Presentation の入出力 DTO

**責務**
- HTTP リクエスト/レスポンスの JSON 構造を表現する

**パターン例**

```go
package dto

type UserRegistrationRequest struct {
    Email string `json:"email"`
    Name  string `json:"name"`
}

type UserRegistrationResponse struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
}
```

---

### Bootstrap (依存関係解決)

#### bootstrap/wire.go

**責務**
- 各レイヤーの実装を組み立てる
- 依存関係の注入を行う
- アプリケーション起動時の初期化処理

**パターン例**

```go
package bootstrap

import (
    "database/sql"
    "example.com/project/internal/domain/service"
    "example.com/project/internal/infrastructure/client"
    "example.com/project/internal/infrastructure/repository"
    "example.com/project/internal/presentation/handler"
    "example.com/project/internal/usecase"
)

func InitializeApp(db *sql.DB, apiBaseURL string) *handler.UserRegistrationHandler {
    // Infrastructure Layer の構築
    userRepo := repository.NewUserRepository(db)
    notificationCli := client.NewExternalAPIClient(apiBaseURL)

    // Domain Service の構築
    emailService := service.NewEmailService(userRepo)

    // Usecase Layer の構築
    registrationUC := usecase.NewUserRegistrationUsecase(
        userRepo,
        emailService,
        notificationCli,
    )

    // Presentation Layer の構築
    return handler.NewUserRegistrationHandler(registrationUC)
}
