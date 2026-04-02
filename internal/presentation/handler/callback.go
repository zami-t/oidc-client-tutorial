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
	usecase       *usecase.CallbackUsecase
	log           *logger.Logger
	cookieManager *CookieManager
}

// NewCallbackHandler creates a new CallbackHandler.
func NewCallbackHandler(uc *usecase.CallbackUsecase, log *logger.Logger, cookies *CookieManager) *CallbackHandler {
	return &CallbackHandler{usecase: uc, log: log, cookieManager: cookies}
}

func (h *CallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	input, err := ucDto.NewCallbackInput(ucDto.CallbackParams{
		Code:             q.Get("code"),
		State:            q.Get("state"),
		Error:            q.Get("error"),
		ErrorDescription: q.Get("error_description"),
	})
	if err != nil {
		h.log.InfoWithError(ctx, "callback: invalid input", err)
		writeJson(w, http.StatusBadRequest, errorResponse{
			ErrorDetailCode: "INVALID_REQUEST",
			Message:         "invalid callback request",
		})
		return
	}

	output, err := h.usecase.Execute(ctx, input)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrCallbackAuthorizationError):
			writeJson(w, http.StatusForbidden, errorResponse{
				ErrorDetailCode: "AUTHORIZATION_ERROR",
				Message:         "authorization error from identity provider",
			})
		case errors.Is(err, usecase.ErrCallbackStateMismatch):
			writeJson(w, http.StatusForbidden, errorResponse{
				ErrorDetailCode: "STATE_MISMATCH",
				Message:         "state validation failed",
			})
		case errors.Is(err, usecase.ErrCallbackTokenVerificationFailed):
			writeJson(w, http.StatusForbidden, errorResponse{
				ErrorDetailCode: "TOKEN_VERIFICATION_FAILED",
				Message:         "token verification failed",
			})
		default:
			writeServerError(w)
		}
		return
	}

	h.log.Info(ctx, "callback: session created, redirecting")
	http.SetCookie(w, h.cookieManager.newSessionCookie(output.SessionId))
	http.Redirect(w, r, output.ReturnTo, http.StatusFound)
}
