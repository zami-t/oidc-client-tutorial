package handler

import (
	"errors"
	"net/http"

	"oidc-tutorial/internal/logger"
	"oidc-tutorial/internal/usecase"
	ucDto "oidc-tutorial/internal/usecase/dto"
)

// CallbackHandler handles GET /callback.
type CallbackHandler struct {
	usecase      *usecase.CallbackUsecase
	sameSite     http.SameSite
	secureCookie bool
	log          *logger.Logger
}

// NewCallbackHandler creates a new CallbackHandler.
func NewCallbackHandler(uc *usecase.CallbackUsecase, sameSite http.SameSite, secureCookie bool, log *logger.Logger) *CallbackHandler {
	return &CallbackHandler{usecase: uc, sameSite: sameSite, secureCookie: secureCookie, log: log}
}

func (h *CallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	input := ucDto.CallbackInput{
		Code:             q.Get("code"),
		State:            q.Get("state"),
		Error:            q.Get("error"),
		ErrorDescription: q.Get("error_description"),
	}

	output, err := h.usecase.Execute(ctx, input)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrCallbackAuthorizationError),
			errors.Is(err, usecase.ErrCallbackStateMismatch):
			h.log.Warn(ctx, "callback failed", err)
		default:
			h.log.Error(ctx, "callback failed", "CALLBACK_FAILED", err)
		}
		writeError(w, err)
		return
	}

	h.log.Info(ctx, "callback: session created, redirecting")
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    output.SessionId,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: h.sameSite,
	})
	http.Redirect(w, r, output.ReturnTo, http.StatusFound)
}
