---
name: domain-layer
description: Domain層（model/service/port）のコードを実装するときに使う。Entity・Value Object・Domain Service・Port Interface を追加・修正するとき。
---

# Domain Layer 実装ガイド

Domain はシステムの中心（Core）であり、他レイヤーへは依存しない。

```
+--------------------+      +--------------------+
| Usecase Layer       | ---> | Domain Layer      | <-- ここ
+--------------------+      +--------------------+
                                      ^
                                      |
                           +--------------------+
                           | Infrastructure Layer|
                           +--------------------+
```

**対象ディレクトリ**

```
internal/domain/
├── model/    # Entity, Value Object
├── service/  # Domain Service
└── port/     # Port Interface（+ センチネルエラー）
```

---

## model/ - Value Object / Entity

**責務**
- ドメインの概念を不変な型として表現する
- 生成時にバリデーションを行い、不正な状態を作らせない（Always Valid Domain Model）
- すべてのフィールドは非公開（小文字）、アクセスはメソッド経由のみ
- コンストラクタ（`New〇〇`）経由でのみ生成する

**パターン例**

```go
package model

// バリデーションが必要な Value Object（コンストラクタが error を返す）
type Email struct {
    value string
}

func NewEmail(value string) (Email, error) {
    if !isValidEmail(value) {
        return Email{}, errors.New("invalid email format")
    }
    return Email{value: value}, nil
}

func (e Email) String() string { return e.value }

// バリデーションが不要な Value Object（defined type、error なし）
type Issuer string

func NewIssuer(value string) Issuer { return Issuer(value) }
func (i Issuer) String() string     { return string(i) }

// Entity（フィールドは非公開、アクセスはメソッド経由のみ）
type User struct {
    id    UserID
    email Email
    name  string
}

func NewUser(id UserID, email Email, name string) User {
    return User{id: id, email: email, name: name}
}

func (u User) ID() UserID   { return u.id }
func (u User) Email() Email { return u.email }
func (u User) Name() string { return u.name }
```

---

## service/ - Domain Service

**責務**
- 複数の Entity/Value Object にまたがるロジックを扱う
- 単一の Entity に属さないドメインロジックを実装する
- ポート（インターフェース）に依存してもよい

**パターン例**

```go
package service

type AuthorizationService struct {
    randomGenerator RandomGenerator
}

func NewAuthorizationService(randomGen RandomGenerator) *AuthorizationService {
    return &AuthorizationService{randomGenerator: randomGen}
}

func (s *AuthorizationService) GenerateState() (string, error) {
    return s.randomGenerator.Generate(32)
}
```

---

## port/ - Port Interface（+ センチネルエラー）

**責務**
- 外部システム（DB, HTTP, KVS）への依存を抽象化する
- 戻り値は Domain の型のみを使う
- Infrastructure 層が返しうるエラーはポートファイルにセンチネルエラーとして定義する

**パターン例**

```go
package port

import (
    "context"
    "example.com/project/internal/domain/model"
)

// センチネルエラーはポートファイルに定義する
var ErrUserNotFound = errors.New("user not found")

// Repository のポート
type UserRepository interface {
    Save(ctx context.Context, user model.User) error
    FindByID(ctx context.Context, id model.UserID) (model.User, error)
}

// Client のポート
type ExternalAPIClient interface {
    FetchData(ctx context.Context, req DataRequest) (model.Data, error)
}

// ポートの入力パラメータは Domain 内で定義する
type DataRequest struct {
    Endpoint string
    Params   map[string]string
}
```
