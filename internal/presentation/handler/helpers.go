package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"oidc-tutorial/internal/domain/model"
)

type errorResponse struct {
	ErrorDetailCode string `json:"error_detail_code"`
	Message         string `json:"message"`
}

// writeError maps an error to the appropriate HTTP status and JSON body.
func writeError(w http.ResponseWriter, err error) {
	var appErr *model.AppError
	if errors.As(err, &appErr) {
		switch appErr.Code {
		case model.ErrCodeUnknownIdp:
			writeJson(w, http.StatusBadRequest, errorResponse{
				ErrorDetailCode: string(appErr.Code),
				Message:         appErr.Message,
			})
		case model.ErrCodeStateMismatch,
			model.ErrCodeTokenVerificationFailed,
			model.ErrCodeAuthorizationError:
			writeJson(w, http.StatusForbidden, errorResponse{
				ErrorDetailCode: string(appErr.Code),
				Message:         appErr.Message,
			})
		case model.ErrCodeSessionNotFound:
			writeJson(w, http.StatusUnauthorized, errorResponse{
				ErrorDetailCode: string(appErr.Code),
				Message:         appErr.Message,
			})
		default:
			writeJson(w, http.StatusInternalServerError, errorResponse{
				ErrorDetailCode: string(model.ErrCodeServerError),
				Message:         "internal server error",
			})
		}
		return
	}
	writeJson(w, http.StatusInternalServerError, errorResponse{
		ErrorDetailCode: string(model.ErrCodeServerError),
		Message:         "internal server error",
	})
}

func writeJson(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
