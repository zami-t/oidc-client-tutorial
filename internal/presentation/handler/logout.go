package handler

import (
	"net/http"

	"oidc-tutorial/internal/usecase"
	ucDto "oidc-tutorial/internal/usecase/dto"
)

// LogoutHandler handles POST /logout.
type LogoutHandler struct {
	usecase      *usecase.LogoutUsecase
	sameSite     http.SameSite
	secureCookie bool
}

// NewLogoutHandler creates a new LogoutHandler.
func NewLogoutHandler(uc *usecase.LogoutUsecase, sameSite http.SameSite, secureCookie bool) *LogoutHandler {
	return &LogoutHandler{usecase: uc, sameSite: sameSite, secureCookie: secureCookie}
}

func (h *LogoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		writeError(w, usecase.ErrLogoutSessionNotFound)
		return
	}

	if err := h.usecase.Execute(r.Context(), ucDto.LogoutInput{SessionId: cookie.Value}); err != nil {
		writeError(w, err)
		return
	}

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
