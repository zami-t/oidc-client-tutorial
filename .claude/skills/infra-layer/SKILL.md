---
name: infra-layer
description: Infrastructure層（repository/client）のコードを実装するときに使う。ポートの具体実装（DB/KVS/HTTP）を追加・修正するとき。
---

# Infrastructure Layer 実装ガイド

Infrastructure は Domain/Usecase が定義したポートに対する「外部I/Oの具体実装」を置く。

```
+--------------------+      +--------------------+
| Usecase Layer       | ---> | Domain Layer      |
+--------------------+      +--------------------+
          ^                         ^
          |                         |
+--------------------+              |
| Infrastructure Layer| ------------+  <-- ここ
+--------------------+
```

**対象ディレクトリ**

```
internal/infrastructure/
├── repository/
│   ├── dto/          # DB レコード構造を表現する DTO
│   └── xxx_repository.go  # port.XxxRepository の実装
└── client/
    ├── dto/          # 外部 API レスポンス構造を表現する DTO
    └── xxx_client.go      # port.XxxClient の実装
```

---

## repository/ - Repository の実装

**責務**
- Domain の Repository ポートを実装する
- DB/KVS 固有の処理を行う
- Infrastructure DTO と Domain Model の相互変換を行う
- port.ErrXxx を `fmt.Errorf` でラップして返す

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

    if errors.Is(err, sql.ErrNoRows) {
        return model.User{}, fmt.Errorf("user %s: %w", id.String(), port.ErrUserNotFound)
    }
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

---

## repository/dto/ - Repository 固有の DTO

**責務**
- DB のレコード構造を表現する

**パターン例**

```go
package dto

type UserDTO struct {
    ID    string
    Email string
    Name  string
}
```

---

## client/ - External API Client の実装

**責務**
- Domain の Client ポートを実装する
- HTTP/gRPC などの外部通信を行う
- Infrastructure DTO と Domain Model の相互変換を行う

**パターン例**

```go
package client

import (
    "context"
    "encoding/json"
    "net/http"
    "time"
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

---

## client/dto/ - Client 固有の DTO

**責務**
- 外部 API のレスポンス構造を表現する

**パターン例**

```go
package dto

type DataResponse struct {
    Value string `json:"value"`
}
```
