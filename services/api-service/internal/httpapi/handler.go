package httpapi

import (
	"encoding/json"
	"net/http"

	"mephi_vkr_aspm/services/api-service/internal/models"
	"mephi_vkr_aspm/services/api-service/internal/service"
)

type Handler struct {
	orchestrator *service.Orchestrator
}

func New(orchestrator *service.Orchestrator) *Handler {
	return &Handler{orchestrator: orchestrator}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/v1/scans/semgrep", h.handleSemgrepScan)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleSemgrepScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var request models.ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if request.ScannerName == "" {
		request.ScannerName = "semgrep"
	}

	passport, err := h.orchestrator.RunSemgrepScenario(r.Context(), request)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, passport)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
