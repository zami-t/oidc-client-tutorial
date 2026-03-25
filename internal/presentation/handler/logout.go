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
	usecase      *usecase.LogoutUsecase
	sameSite     http.SameSite
	secureCookie bool
	log          *logger.Logger
}

// NewLogoutHandler creates a new LogoutHandler.
func NewLogoutHandler(uc *usecase.LogoutUsecase, sameSite http.SameSite, secureCookie bool, log *logger.Logger) *LogoutHandler {
	return &LogoutHandler{usecase: uc, sameSite: sameSite, secureCookie: secureCookie, log: log}
}

func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cookie, err := r.Cookie("session_id")
	if err != nil {
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
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: h.sameSite,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusOK)
}
