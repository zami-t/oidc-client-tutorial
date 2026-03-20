---
name: presentation-layer
description: Presentation層（handler/dto）のコードを実装するときに使う。HTTPハンドラーの追加・修正、presentation/dto の追加・修正するとき。
---

# Presentation Layer 実装ガイド

Presentation Layer は、外部（HTTP）とアプリケーションの境界に位置する薄い層。
ビジネスロジックは一切置かない。

```
+--------------------+  <-- ここ
| Presentation Layer  |
+--------------------+
          |
          v
+--------------------+
| Usecase Layer       |
+--------------------+
```

**対象ディレクトリ**

```
internal/presentation/
├── handler/      # HTTP Handler
└── dto/          # HTTP リクエスト/レスポンスの JSON 構造
```

---

## handler/ - HTTP Handler

**責務**

- HTTP リクエストを受け取り、Usecase の DTO に変換する
- Usecase を呼び出し、結果を HTTP レスポンスとして返す
- `errors.Is` で usecase.ErrXxx を検出し、適切なHTTPステータスにマッピングする

**エラーハンドリングの規則**

- エラーハンドリングは**各ハンドラーファイル内に直接記述する**

**パターン例**

```go
package handler

import (
    "encoding/json"
    "errors"
    "net/http"
    "example.com/project/internal/usecase"
    usecaseDTO "example.com/project/internal/usecase/dto"
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
    input := usecaseDTO.UserRegistrationInput{
        Email: req.Email,
        Name:  req.Name,
    }

    // 3. Usecase の実行
    output, err := h.usecase.Execute(r.Context(), input)
    if err != nil {
        // このハンドラーが呼ぶユースケースのエラーだけをここで処理する
        switch {
        case errors.Is(err, usecase.ErrUserRegistrationEmailDuplicate):
            writeJson(w, http.StatusConflict, errorResponse{
                ErrorDetailCode: "EMAIL_DUPLICATE",
                Message:         "email already registered",
            })
        case errors.Is(err, usecase.ErrUserRegistrationInvalidInput):
            writeJson(w, http.StatusBadRequest, errorResponse{
                ErrorDetailCode: "INVALID_INPUT",
                Message:         "invalid request",
            })
        default:
            writeServerError(w)
        }
        return
    }

    // 4. Usecase DTO → Presentation DTO → レスポンス返却
    resp := presentationDTO.UserRegistrationResponse{
        UserID: output.UserID,
        Email:  output.Email,
    }
    writeJson(w, http.StatusCreated, resp)
}
```

---

## presentation/dto/ - Presentation の入出力 DTO

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
