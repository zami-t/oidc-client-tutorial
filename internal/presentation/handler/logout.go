package handler

import (
	"errors"
	"net/http"

	"oidc-tutorial/internal/logger"
	"oidc-tutorial/internal/usecase"
	ucDto "oidc-tutorial/internal/usecase/dto"
)

// LogoutHandler handles POST /logout.
type LogoutHandler struct {
	usecase       *usecase.LogoutUsecase
	log           *logger.Logger
	cookieManager *CookieManager
}

// NewLogoutHandler creates a new LogoutHandler.
func NewLogoutHandler(uc *usecase.LogoutUsecase, log *logger.Logger, cookieManager *CookieManager) *LogoutHandler {
	return &LogoutHandler{usecase: uc, log: log, cookieManager: cookieManager}
}

func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cookie, err := r.Cookie("session_id")
	if err != nil {
		h.log.InfoWithError(ctx, "logout: session cookie not found", err)
		writeJson(w, http.StatusUnauthorized, errorResponse{
			ErrorDetailCode: "SESSION_NOT_FOUND",
			Message:         "not authenticated",
		})
		return
	}

	if err := h.usecase.Execute(ctx, ucDto.LogoutInput{SessionId: cookie.Value}); err != nil {
		switch {
		case errors.Is(err, usecase.ErrLogoutSessionNotFound):
			writeJson(w, http.StatusUnauthorized, errorResponse{
				ErrorDetailCode: "SESSION_NOT_FOUND",
				Message:         "not authenticated",
			})
		default:
			writeServerError(w)
		}
		return
	}

	h.log.Info(ctx, "logout: session deleted")
	http.SetCookie(w, h.cookieManager.clearSessionCookie())
	w.WriteHeader(http.StatusOK)
}
