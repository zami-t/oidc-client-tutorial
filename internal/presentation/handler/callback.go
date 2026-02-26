package handler

import (
	"net/http"

	"oidc-tutorial/internal/usecase"
	ucDto "oidc-tutorial/internal/usecase/dto"
)

// CallbackHandler handles GET /callback.
type CallbackHandler struct {
	usecase      *usecase.CallbackUsecase
	sameSite     http.SameSite
	secureCookie bool
}

// NewCallbackHandler creates a new CallbackHandler.
func NewCallbackHandler(uc *usecase.CallbackUsecase, sameSite http.SameSite, secureCookie bool) *CallbackHandler {
	return &CallbackHandler{usecase: uc, sameSite: sameSite, secureCookie: secureCookie}
}

func (h *CallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	input := ucDto.CallbackInput{
		Code:             q.Get("code"),
		State:            q.Get("state"),
		Error:            q.Get("error"),
		ErrorDescription: q.Get("error_description"),
	}

	output, err := h.usecase.Execute(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}

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
