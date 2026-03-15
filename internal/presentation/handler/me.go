package handler

import (
	"net/http"

	"oidc-tutorial/internal/logger"
	"oidc-tutorial/internal/usecase"
	ucDto "oidc-tutorial/internal/usecase/dto"
)

// MeHandler handles GET /me.
type MeHandler struct {
	usecase *usecase.MeUsecase
	log     *logger.Logger
}

// NewMeHandler creates a new MeHandler.
func NewMeHandler(uc *usecase.MeUsecase, log *logger.Logger) *MeHandler {
	return &MeHandler{usecase: uc, log: log}
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cookie, err := r.Cookie("session_id")
	if err != nil {
		writeError(w, usecase.ErrMeSessionNotFound)
		return
	}

	output, err := h.usecase.Execute(ctx, ucDto.MeInput{SessionId: cookie.Value})
	if err != nil {
		h.log.Error(ctx, "me: session lookup failed", "SESSION_LOOKUP_FAILED", err)
		writeError(w, err)
		return
	}

	resp := map[string]any{
		"subject": output.Subject,
		"issuer":  output.Issuer,
	}
	if output.Email != "" {
		resp["email"] = output.Email
	}
	if output.Name != "" {
		resp["name"] = output.Name
	}
	if output.Picture != "" {
		resp["picture"] = output.Picture
	}

	writeJson(w, http.StatusOK, resp)
}
