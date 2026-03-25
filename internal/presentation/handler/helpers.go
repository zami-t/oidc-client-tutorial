package handler

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	ErrorDetailCode string `json:"error_detail_code"`
	Message         string `json:"message"`
}

func writeServerError(w http.ResponseWriter) {
	writeJson(w, http.StatusInternalServerError, errorResponse{
		ErrorDetailCode: "SERVER_ERROR",
		Message:         "internal server error",
	})
}

func writeJson(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
