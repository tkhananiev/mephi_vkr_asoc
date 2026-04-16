package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"mephi_vkr_aspm/services/reference-data-service/internal/models"
	"mephi_vkr_aspm/services/reference-data-service/internal/service"
)

type Handler struct {
	syncService *service.SyncService
}

func New(syncService *service.SyncService) *Handler {
	return &Handler{syncService: syncService}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/v1/sync/bdu", h.handleSyncBDU)
	mux.HandleFunc("/api/v1/sync/nvd", h.handleSyncNVD)
	mux.HandleFunc("/api/v1/sync/all", h.handleSyncAll)
	mux.HandleFunc("/api/v1/sync/runs", h.handleListRuns)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleSyncBDU(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	result, err := h.syncService.SyncBDU(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (h *Handler) handleSyncNVD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	cveID := r.URL.Query().Get("cve_id")
	var (
		result models.SyncResult
		err    error
	)
	if cveID != "" {
		result, err = h.syncService.SyncNVDByCVE(r.Context(), cveID)
	} else {
		result, err = h.syncService.SyncNVD(r.Context())
	}
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (h *Handler) handleSyncAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	type payload struct {
		BDU models.SyncResult `json:"bdu"`
		NVD models.SyncResult `json:"nvd"`
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	bduResult, bduErr := h.syncService.SyncBDU(ctx)
	nvdResult, nvdErr := h.syncService.SyncNVD(ctx)
	if bduErr != nil || nvdErr != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"bdu_error": errString(bduErr),
			"nvd_error": errString(nvdErr),
			"bdu":       bduResult,
			"nvd":       nvdResult,
		})
		return
	}

	writeJSON(w, http.StatusAccepted, payload{BDU: bduResult, NVD: nvdResult})
}

func (h *Handler) handleListRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	runs, err := h.syncService.ListRuns(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
