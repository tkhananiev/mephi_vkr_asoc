package httpapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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
	mux.HandleFunc("/api/v1/groups/", h.handleGroups)
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
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/groups")
	path = strings.Trim(path, "/")
	if path != "" {
		segments := strings.Split(path, "/")
		groupIDRaw := segments[0]
		if len(segments) == 2 && segments[1] == "jira-context" {
			h.handleGroupJiraContext(w, r, groupIDRaw)
			return
		}
		h.handleGroupByID(w, r, groupIDRaw)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	if statusFilter == "" {
		statusFilter = "open"
	}

	cp, cpErr := parseConsoleProductFilter(r)
	if cpErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": *cpErr})
		return
	}
	owner := parseOwnerHeader(r)
	if cp != nil && owner == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console_product_id requires X-ASOC-Console-User-ID"})
		return
	}

	groups, err := h.processingService.ListGroups(r.Context(), limit, owner, cp, statusFilter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h *Handler) handleGroupJiraContext(w http.ResponseWriter, r *http.Request, idRaw string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	groupID, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil || groupID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid group id"})
		return
	}
	cp, cpErr := parseConsoleProductFilter(r)
	if cpErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": *cpErr})
		return
	}
	owner := parseOwnerHeader(r)
	if cp != nil && owner == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console_product_id requires X-ASOC-Console-User-ID"})
		return
	}
	ctx, err := h.processingService.GetGroupJiraContext(r.Context(), groupID, owner, cp)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "forbidden") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, ctx)
}

func (h *Handler) handleGroupByID(w http.ResponseWriter, r *http.Request, idRaw string) {
	if r.Method != http.MethodPatch {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	groupID, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil || groupID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid group id"})
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	cp, cpErr := parseConsoleProductFilter(r)
	if cpErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": *cpErr})
		return
	}
	owner := parseOwnerHeader(r)
	if cp != nil && owner == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console_product_id requires X-ASOC-Console-User-ID"})
		return
	}
	updated, err := h.processingService.UpdateGroupStatus(r.Context(), groupID, body.Status, owner, cp)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "forbidden") {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "invalid status") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
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

	cp, cpErr := parseConsoleProductFilter(r)
	if cpErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": *cpErr})
		return
	}
	owner := parseOwnerHeader(r)
	if cp != nil && owner == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console_product_id requires X-ASOC-Console-User-ID"})
		return
	}

	rows, err := h.processingService.ListVulnerabilityReport(r.Context(), limit, owner, cp, parseReportFilter(r.URL.Query()))
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

func parseConsoleProductFilter(r *http.Request) (*int64, *string) {
	raw := strings.TrimSpace(r.URL.Query().Get("console_product_id"))
	if raw == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		msg := "invalid console_product_id"
		return nil, &msg
	}
	return &id, nil
}

func parseQueryInt64(v url.Values, key string) *int64 {
	raw := strings.TrimSpace(v.Get(key))
	if raw == "" {
		return nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}

func parseReportTimeFilter(v url.Values, key string, endOfDayInclusive bool) *time.Time {
	raw := strings.TrimSpace(v.Get(key))
	if raw == "" {
		return nil
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"}
	var parsed time.Time
	var usedLayout string
	for _, ly := range layouts {
		t, parseErr := time.Parse(ly, raw)
		if parseErr != nil {
			continue
		}
		parsed = t
		usedLayout = ly
		break
	}
	if usedLayout == "" {
		return nil
	}
	if usedLayout == "2006-01-02" {
		if endOfDayInclusive {
			parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 999_999_999, time.UTC)
		} else {
			parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
		}
	}
	u := parsed.UTC()
	return &u
}

func parseReportFilter(v url.Values) *models.VulnerabilityReportFilter {
	get := func(key string) *string {
		raw := strings.TrimSpace(v.Get(key))
		if raw == "" {
			return nil
		}
		return &raw
	}
	gk := get("group_key")
	cve := get("cve")
	bdu := get("bdu_id")
	scn := get("scanner_name")
	sev := get("severity")
	cat := get("catalog_source")
	rch := get("run_channel")
	ap := get("asset_path")
	ver := get("version")
	cwe := get("cwe")
	gid := parseQueryInt64(v, "group_id")
	vid := parseQueryInt64(v, "vulnerability_id")
	from := parseReportTimeFilter(v, "run_at_from", false)
	to := parseReportTimeFilter(v, "run_at_to", true)
	f := models.VulnerabilityReportFilter{
		GroupKey:        gk,
		CVE:             cve,
		BDUID:           bdu,
		ScannerName:     scn,
		Severity:        sev,
		CatalogSource:   cat,
		RunChannel:      rch,
		AssetPath:       ap,
		Version:         ver,
		CWE:             cwe,
		GroupID:         gid,
		VulnerabilityID: vid,
		RunAtFrom:       from,
		RunAtTo:         to,
	}
	if filterReportEmpty(&f) {
		return nil
	}
	return &f
}

func filterReportEmpty(f *models.VulnerabilityReportFilter) bool {
	if f == nil {
		return true
	}
	return f.GroupKey == nil && f.CVE == nil && f.BDUID == nil && f.ScannerName == nil &&
		f.Severity == nil && f.CatalogSource == nil && f.RunChannel == nil && f.AssetPath == nil &&
		f.Version == nil && f.CWE == nil && f.GroupID == nil && f.VulnerabilityID == nil &&
		f.RunAtFrom == nil && f.RunAtTo == nil
}
