package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"mephi_vkr_asoc/services/processing-service/internal/models"
	"mephi_vkr_asoc/services/processing-service/internal/service"
)

type Handler struct {
	processingService         *service.ProcessingService
	httpFindingsIngestEnabled bool
}

const headerConsoleUserID = "X-ASOC-Console-User-ID"

func New(processingService *service.ProcessingService, httpFindingsIngestEnabled bool) *Handler {
	return &Handler{
		processingService:        processingService,
		httpFindingsIngestEnabled: httpFindingsIngestEnabled,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	if h.httpFindingsIngestEnabled {
		mux.HandleFunc("/api/v1/findings/ingest", h.handleIngest)
	}
	mux.HandleFunc("/api/v1/groups", h.handleGroups)
	mux.HandleFunc("/api/v1/report/vulnerabilities", h.handleReportVulnerabilities)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var request models.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	mergeIngestOwnerFromHeader(r, &request)

	result, err := h.processingService.ProcessFindings(r.Context(), request)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (h *Handler) handleGroups(w http.ResponseWriter, r *http.Request) {
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

	groups, err := h.processingService.ListGroups(r.Context(), limit, parseOwnerHeader(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h *Handler) handleReportVulnerabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	rows, err := h.processingService.ListVulnerabilityReport(r.Context(), limit, parseOwnerHeader(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func mergeIngestOwnerFromHeader(r *http.Request, req *models.IngestRequest) {
	if req.OwnerUserID != nil && *req.OwnerUserID > 0 {
		return
	}
	req.OwnerUserID = parseOwnerHeader(r)
}

func parseOwnerHeader(r *http.Request) *int64 {
	h := strings.TrimSpace(r.Header.Get(headerConsoleUserID))
	if h == "" {
		return nil
	}
	id, err := strconv.ParseInt(h, 10, 64)
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}
