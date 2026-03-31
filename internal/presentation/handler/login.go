package handler

import (
	"errors"
	"net/http"

	"oidc-tutorial/internal/logger"
	"oidc-tutorial/internal/usecase"
	ucDto "oidc-tutorial/internal/usecase/dto"
)

// LoginHandler handles GET /login.
type LoginHandler struct {
	usecase *usecase.LoginUsecase
	log     *logger.Logger
}

// NewLoginHandler creates a new LoginHandler.
func NewLoginHandler(uc *usecase.LoginUsecase, log *logger.Logger) *LoginHandler {
	return &LoginHandler{usecase: uc, log: log}
}

func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	input := ucDto.LoginInput{
		Idp:      r.URL.Query().Get("idp"),
		ReturnTo: r.URL.Query().Get("return_to"),
	}

	output, err := h.usecase.Execute(ctx, input)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrLoginUnknownIdp):
			writeJson(w, http.StatusBadRequest, errorResponse{
				ErrorDetailCode: "UNKNOWN_IDP",
				Message:         "unknown identity provider",
			})
		case errors.Is(err, usecase.ErrLoginInvalidReturnTo):
			writeJson(w, http.StatusBadRequest, errorResponse{
				ErrorDetailCode: "INVALID_RETURN_TO",
				Message:         "return_to is not an allowed origin",
			})
		default:
			writeServerError(w)
		}
		return
	}

	h.log.Info(ctx, "login: redirecting to authorization endpoint")
	http.Redirect(w, r, output.RedirectUrl, http.StatusFound)
}
