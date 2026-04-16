package httpapi

import (
	"encoding/json"
	"net/http"

	"mephi_vkr_aspm/services/semgrep-service/internal/runner"
)

type Handler struct {
	runner *runner.Runner
}

func New(r *runner.Runner) *Handler {
	return &Handler{runner: r}
}

type scanRequest struct {
	TargetPath    string `json:"target_path"`
	SemgrepConfig string `json:"semgrep_config,omitempty"`
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/v1/scan", h.handleScan)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req scanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.TargetPath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "target_path required"})
		return
	}

	payload, err := h.runner.Run(r.Context(), req.TargetPath, req.SemgrepConfig)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
