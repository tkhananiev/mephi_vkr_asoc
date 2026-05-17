package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"mephi_vkr_asoc/services/api-service/internal/auth"
)

// Только JWT Atomic-admin (role=admin). Не API-ключ — ops слишком опасны.
func (h *Handler) requireAdminRole(r *http.Request) (*auth.Claims, bool) {
	if len(h.jwtSecret) < 32 {
		return nil, false
	}
	a := r.Header.Get("Authorization")
	if len(a) < 8 || !strings.EqualFold(a[:7], "bearer ") {
		return nil, false
	}
	tok := strings.TrimSpace(a[7:])
	c, err := auth.ParseJWT(h.jwtSecret, tok)
	if err != nil || c == nil || c.Role != "admin" {
		return nil, false
	}
	return c, true
}

func (h *Handler) handleDockerLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	_, ok := h.requireAdminRole(r)
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin bearer token required"})
		return
	}
	if h.podOps == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Логи и рестарт недоступны: включите APP_K8S_OPS_ENABLED в Kubernetes (см. RBAC и ServiceAccount api-service) или APP_DOCKER_OPS_ENABLED в docker-compose с /var/run/docker.sock.",
		})
		return
	}
	svc := strings.TrimSpace(r.URL.Query().Get("service"))
	if svc == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query service is required (api, auth, ref, …)"})
		return
	}
	tail := 200
	if v := strings.TrimSpace(r.URL.Query().Get("tail")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			tail = n
		}
	}
	out, err := h.podOps.Logs(r.Context(), svc, tail)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		if len(out) > 0 {
			_, _ = w.Write(out)
			_, _ = io.WriteString(w, "\n")
		}
		_, _ = io.WriteString(w, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

func (h *Handler) handleDockerRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	_, ok := h.requireAdminRole(r)
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin bearer token required"})
		return
	}
	if h.podOps == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "Логи и рестарт недоступны: включите APP_K8S_OPS_ENABLED в Kubernetes (см. RBAC и ServiceAccount api-service) или APP_DOCKER_OPS_ENABLED в docker-compose с /var/run/docker.sock.",
		})
		return
	}
	var body struct {
		Service string `json:"service"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	svc := strings.TrimSpace(body.Service)
	if svc == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "service is required"})
		return
	}
	out, err := h.podOps.Restart(context.Background(), svc)
	if err != nil {
		msg := err.Error()
		if len(out) > 0 {
			msg = string(out) + "\n" + msg
		}
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": msg})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}
