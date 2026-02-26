package handler

import (
	"net/http"

	"oidc-tutorial/internal/usecase"
	ucDto "oidc-tutorial/internal/usecase/dto"
)

// LoginHandler handles GET /login.
type LoginHandler struct {
	usecase *usecase.LoginUsecase
}

// NewLoginHandler creates a new LoginHandler.
func NewLoginHandler(uc *usecase.LoginUsecase) *LoginHandler {
	return &LoginHandler{usecase: uc}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	input := ucDto.LoginInput{
		Idp:      r.URL.Query().Get("idp"),
		ReturnTo: r.URL.Query().Get("return_to"),
	}

	output, err := h.usecase.Execute(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}

	http.Redirect(w, r, output.RedirectUrl, http.StatusFound)
}
