package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"oidc-tutorial/internal/usecase"
)

type errorResponse struct {
	ErrorDetailCode string `json:"error_detail_code"`
	Message         string `json:"message"`
}

// writeError maps an error to the appropriate HTTP status and JSON body.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, usecase.ErrLoginUnknownIdp):
		writeJson(w, http.StatusBadRequest, errorResponse{
			ErrorDetailCode: "UNKNOWN_IDP",
			Message:         "unknown identity provider",
		})
	case errors.Is(err, usecase.ErrCallbackStateMismatch),
		errors.Is(err, usecase.ErrCallbackTokenVerificationFailed),
		errors.Is(err, usecase.ErrCallbackAuthorizationError):
		writeJson(w, http.StatusForbidden, errorResponse{
			ErrorDetailCode: errDetailCode(err),
			Message:         err.Error(),
		})
	case errors.Is(err, usecase.ErrMeSessionNotFound),
		errors.Is(err, usecase.ErrLogoutSessionNotFound):
		writeJson(w, http.StatusUnauthorized, errorResponse{
			ErrorDetailCode: "SESSION_NOT_FOUND",
			Message:         "not authenticated",
		})
	default:
		writeJson(w, http.StatusInternalServerError, errorResponse{
			ErrorDetailCode: "SERVER_ERROR",
			Message:         "internal server error",
		})
	}
}

func errDetailCode(err error) string {
	switch {
	case errors.Is(err, usecase.ErrCallbackStateMismatch):
		return "STATE_MISMATCH"
	case errors.Is(err, usecase.ErrCallbackTokenVerificationFailed):
		return "TOKEN_VERIFICATION_FAILED"
	case errors.Is(err, usecase.ErrCallbackAuthorizationError):
		return "AUTHORIZATION_ERROR"
	default:
		return "SERVER_ERROR"
	}
}

func writeJson(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
