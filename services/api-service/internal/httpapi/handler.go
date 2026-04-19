package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"mephi_vkr_aspm/services/api-service/internal/models"
	"mephi_vkr_aspm/services/api-service/internal/service"
)

type Handler struct {
	orchestrator           *service.Orchestrator
	defaultScanTargetPath  string
	defaultSemgrepConfig     string
}

func New(orchestrator *service.Orchestrator, defaultScanTargetPath, defaultSemgrepConfig string) *Handler {
	return &Handler{
		orchestrator:          orchestrator,
		defaultScanTargetPath: defaultScanTargetPath,
		defaultSemgrepConfig:    defaultSemgrepConfig,
	}
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
	if strings.TrimSpace(request.TargetPath) == "" {
		request.TargetPath = h.defaultScanTargetPath
	}
	if strings.TrimSpace(request.SemgrepConfig) == "" {
		request.SemgrepConfig = h.defaultSemgrepConfig
	}
	if strings.TrimSpace(request.TargetPath) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "target_path required (or set APP_DEFAULT_SCAN_TARGET_PATH)"})
		return
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
