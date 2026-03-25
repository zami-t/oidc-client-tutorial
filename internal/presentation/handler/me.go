package handler

import (
	"errors"
	"net/http"

	"oidc-tutorial/internal/usecase"
	ucDto "oidc-tutorial/internal/usecase/dto"
)

// MeHandler handles GET /me.
type MeHandler struct {
	usecase *usecase.MeUsecase
}

// NewMeHandler creates a new MeHandler.
func NewMeHandler(uc *usecase.MeUsecase) *MeHandler {
	return &MeHandler{usecase: uc}
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cookie, err := r.Cookie("session_id")
	if err != nil {
		writeJson(w, http.StatusUnauthorized, errorResponse{
			ErrorDetailCode: "SESSION_NOT_FOUND",
			Message:         "not authenticated",
		})
		return
	}

	output, err := h.usecase.Execute(ctx, ucDto.MeInput{SessionId: cookie.Value})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrMeSessionNotFound):
			writeJson(w, http.StatusUnauthorized, errorResponse{
				ErrorDetailCode: "SESSION_NOT_FOUND",
				Message:         "not authenticated",
			})
		default:
			writeServerError(w)
		}
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
