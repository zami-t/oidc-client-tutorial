package handler

import (
	"errors"
	"net/http"

	"oidc-tutorial/internal/logger"
	"oidc-tutorial/internal/usecase"
	ucDto "oidc-tutorial/internal/usecase/dto"
)

type meResponse struct {
	Subject string `json:"subject"`
	Issuer  string `json:"issuer"`
	Email   string `json:"email,omitempty"`
	Name    string `json:"name,omitempty"`
	Picture string `json:"picture,omitempty"`
}

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
		h.log.InfoWithError(ctx, "me: session cookie not found", err)
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

	resp := meResponse{
		Subject: output.Subject,
		Issuer:  output.Issuer,
		Email:   output.Email,
		Name:    output.Name,
		Picture: output.Picture,
	}
	writeJson(w, http.StatusOK, resp)
}
