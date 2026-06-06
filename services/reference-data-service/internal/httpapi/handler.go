package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mephi_vkr_asoc/services/reference-data-service/internal/service"
)

func longRunningSyncContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 168*time.Hour)
}

type Handler struct {
	syncService *service.SyncService
}

func New(syncService *service.SyncService) *Handler {
	return &Handler{syncService: syncService}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/v1/sync/bdu", h.handleSyncBDU)
	mux.HandleFunc("/api/v1/sync/bdu/bulk", h.handleSyncBDUBulk)
	mux.HandleFunc("/api/v1/sync/nvd", h.handleSyncNVD)
	mux.HandleFunc("/api/v1/sync/all", h.handleSyncAll)
	mux.HandleFunc("/api/v1/sync/runs", h.handleListRuns)
	mux.HandleFunc("/api/v1/sync/status", h.handleCatalogStatus)
}

func (h *Handler) handleCatalogStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	out, err := h.syncService.CatalogStatus(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleSyncBDU(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	ctx, cancel := longRunningSyncContext()
	defer cancel()
	result, err := h.syncService.SyncBDU(ctx)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (h *Handler) handleSyncBDUBulk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if err := h.syncService.SyncBDUBulkAsync(); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "accepted",
		"hint":   "poll GET /api/v1/sync/status — блок «Сейчас в работе» и счётчики items_* обновляются в процессе",
	})
}

func (h *Handler) handleSyncNVD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	cveID := r.URL.Query().Get("cve_id")
	if cveID != "" {
		result, err := h.syncService.SyncNVDByCVE(r.Context(), cveID)
		if err != nil {
			status := http.StatusBadGateway
			if strings.Contains(err.Error(), "занят") {
				status = http.StatusConflict
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, result)
		return
	}
	if r.URL.Query().Get("full") == "1" {
		_ = h.syncService.ResetNVDCursor(r.Context())
	}
	if err := h.syncService.SyncNVDAsync(); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "accepted",
		"hint":   "poll GET /api/v1/sync/status — счётчики прогресса для NVD пишутся в audit.reference_sync_runs во время загрузки страниц",
	})
}

func (h *Handler) handleSyncAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	go func() {
		ctx, cancel := longRunningSyncContext()
		defer cancel()
		bduResult, bduErr := h.syncService.SyncBDU(ctx)
		nvdResult, nvdErr := h.syncService.SyncNVD(ctx)
		if bduErr != nil || nvdErr != nil {
			log.Printf("sync/all background: bdu_err=%v nvd_err=%v", bduErr, nvdErr)
		}
		_ = bduResult
		_ = nvdResult
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"status": "accepted",
		"hint":   "БДУ RSS затем NVD выполняются в фоне; смотрите GET /api/v1/sync/status",
	})
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
