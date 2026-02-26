package handler

import (
	"net/http"
)

// HealthHandler handles GET /health.
type HealthHandler struct{}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeJson(w, http.StatusOK, map[string]string{"status": "ok"})
}
