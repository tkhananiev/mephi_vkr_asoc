package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"mephi_vkr_asoc/services/api-service/internal/integrationstore"
	"mephi_vkr_asoc/services/api-service/internal/models"
	"mephi_vkr_asoc/services/api-service/internal/ops"
	"mephi_vkr_asoc/services/api-service/internal/products"
	"mephi_vkr_asoc/services/api-service/internal/service"
)

type Handler struct {
	orchestrator          *service.Orchestrator
	defaultScanTargetPath string
	defaultSemgrepConfig  string
	jwtSecret             []byte
	podOps                ops.Backend
	integrationStore      *integrationstore.Store
	processingURL         string
	httpUpstream          *http.Client
	productStore          *products.Store
}

func New(
	orchestrator *service.Orchestrator,
	defaultScanTargetPath, defaultSemgrepConfig string,
	jwtSecret []byte,
	podOps ops.Backend,
	integrations *integrationstore.Store,
	processingURL string,
	productStore *products.Store,
) *Handler {
	up := defaultHTTPUpstream()
	return &Handler{
		orchestrator:          orchestrator,
		defaultScanTargetPath: defaultScanTargetPath,
		defaultSemgrepConfig:  defaultSemgrepConfig,
		jwtSecret:             jwtSecret,
		podOps:                podOps,
		integrationStore:      integrations,
		processingURL:         strings.TrimRight(processingURL, "/"),
		httpUpstream:          up,
		productStore:          productStore,
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/api/v1/integrations", h.handleIntegrationsList)
	mux.HandleFunc("/api/v1/admin/integrations", h.handleAdminIntegrations)
	mux.HandleFunc("/api/v1/console/repository-branches", h.handleConsoleRepositoryBranches)
	mux.HandleFunc("/api/v1/console/products/", h.handleConsoleProductByPath)
	mux.HandleFunc("/api/v1/console/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.handleConsoleProductsList(w, r)
		case http.MethodPost:
			h.handleConsoleProductsCreate(w, r)
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	})
	mux.HandleFunc("/api/v1/groups", h.handleGroupsRoute)
	mux.HandleFunc("/api/v1/groups/", h.handleGroupsRoute)
	mux.HandleFunc("/api/v1/report/vulnerabilities/stats", h.handleReportVulnerabilityStatsProxy)
	mux.HandleFunc("/api/v1/report/vulnerabilities", h.handleReportVulnerabilitiesProxy)
	mux.HandleFunc("/api/v1/findings/ingest", h.handlePublicFindingsIngest)
	mux.HandleFunc("/api/v1/scans", h.handleUnifiedScan)
	mux.HandleFunc("/api/v1/scans/semgrep", h.handleSemgrepScan)
	mux.HandleFunc("/api/v1/scans/gitleaks", h.handleGitleaksScan)
	mux.HandleFunc("/api/v1/scans/sca", h.handleScaScan)
	mux.HandleFunc("/api/v1/scans/dast", h.handleDastScan)
	mux.HandleFunc("/api/v1/admin/ops/docker/logs", h.handleDockerLogs)
	mux.HandleFunc("/api/v1/admin/ops/docker/restart", h.handleDockerRestart)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// тот же JSON, что internal POST processing /api/v1/findings/ingest. Владелец выставляется только
// Поле console_product_id допускается только с JWT пользователя и только если продукт принадлежит ему;
// с одним API-ключом отбрасывается.
func (h *Handler) handlePublicFindingsIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var payload models.ProcessingIngestRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	payload.OwnerUserID = nil
	payload.ConsoleProductID = normalizeConsoleProductID(payload.ConsoleProductID)

	uid, jwtOK := ConsoleUserFromRequest(r)
	if jwtOK {
		id := uid
		payload.OwnerUserID = &id
	} else {
		payload.ConsoleProductID = nil
	}

	if payload.ConsoleProductID != nil {
		if h.productStore == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "products store unavailable (set APP_POSTGRES_DSN)"})
			return
		}
		owned, err := h.productStore.ProductOwnedBy(r.Context(), *payload.ConsoleProductID, uid)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if !owned {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "console_product_id not found or forbidden"})
			return
		}
	}

	if strings.TrimSpace(payload.ScannerName) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "scanner_name required"})
		return
	}
	if strings.TrimSpace(payload.Channel) == "" {
		payload.Channel = "ci"
	}
	if h.orchestrator.KafkaIngestEnabled() {
		corr, err := h.orchestrator.EnqueueFindings(r.Context(), payload)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, models.IngestAcceptedResponse{
			Status:         "accepted",
			CorrelationID:  corr,
			FindingsQueued: len(payload.Findings),
		})
		return
	}
	res, err := h.orchestrator.IngestFindings(r.Context(), payload)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, res)
}

func normalizeConsoleProductID(p *int64) *int64 {
	if p == nil || *p <= 0 {
		return nil
	}
	return p
}

func (h *Handler) assertConsoleProductAccess(w http.ResponseWriter, r *http.Request, pid *int64) bool {
	if pid == nil {
		return true
	}
	uid, jwtOK := ConsoleUserFromRequest(r)
	if !jwtOK {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console_product_id requires console user JWT"})
		return false
	}
	if h.productStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "products store unavailable (set APP_POSTGRES_DSN)"})
		return false
	}
	ok, err := h.productStore.ProductOwnedBy(r.Context(), *pid, uid)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return false
	}
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "console_product_id not found or forbidden"})
		return false
	}
	return true
}

func (h *Handler) handleUnifiedScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	var u models.UnifiedScanRequest
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	u.ConsoleProductID = normalizeConsoleProductID(u.ConsoleProductID)
	if !h.assertConsoleProductAccess(w, r, u.ConsoleProductID) {
		return
	}
	sid := strings.TrimSpace(u.ScannerID)
	if sid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "scanner_id required"})
		return
	}
	request := u.ToScanRequest()
	if err := h.prepareScannerTarget(sid, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var owner int64
	if uid, ok := ConsoleUserFromRequest(r); ok {
		owner = uid
	}

	passport, err := h.orchestrator.RunScan(r.Context(), sid, request, owner)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, service.ErrUnsupportedScannerID) {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, passport)
}

func (h *Handler) prepareScannerTarget(scannerID string, request *models.ScanRequest) error {
	sid := strings.ToLower(strings.TrimSpace(scannerID))
	switch sid {
	case "semgrep":
		gitURL := strings.TrimSpace(request.GitRepositoryURL)
		if gitURL != "" {
			request.GitRepositoryURL = gitURL
			request.GitRepositoryRef = strings.TrimSpace(request.GitRepositoryRef)
			request.TargetPath = strings.TrimSpace(request.TargetPath)
			return nil
		}
		if err := h.prepareFilesystemScanDefaults(request); err != nil {
			return err
		}
		if strings.TrimSpace(request.SemgrepConfig) == "" {
			request.SemgrepConfig = h.defaultSemgrepConfig
		}
		return nil
	case "gitleaks":
		return h.prepareFilesystemScanDefaults(request)
	case "trivy-sca", "sca", "trivy":
		return h.prepareFilesystemScanDefaults(request)
	case "zap-dast", "dast", "zap":
		if strings.TrimSpace(request.TargetURL) == "" {
			return fmt.Errorf("target_url required for DAST scan")
		}
		request.TargetURL = strings.TrimSpace(request.TargetURL)
		return nil
	default:
		if it, ok := h.integrationStore.LookupAdditional(scannerID); ok && strings.TrimSpace(it.ScannerInvokeURL) != "" {
			return h.prepareFilesystemScanDefaults(request)
		}
		return nil
	}
}

func (h *Handler) prepareFilesystemScanDefaults(request *models.ScanRequest) error {
	gitURL := strings.TrimSpace(request.GitRepositoryURL)
	if gitURL != "" {
		request.GitRepositoryURL = gitURL
		request.GitRepositoryRef = strings.TrimSpace(request.GitRepositoryRef)
		request.TargetPath = strings.TrimSpace(request.TargetPath)
		return nil
	}
	if strings.TrimSpace(request.TargetPath) == "" {
		request.TargetPath = h.defaultScanTargetPath
	}
	request.TargetPath = strings.TrimSpace(request.TargetPath)
	if request.TargetPath == "" {
		return fmt.Errorf("target_path required (or set APP_DEFAULT_SCAN_TARGET_PATH)")
	}
	return nil
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
	request.ConsoleProductID = normalizeConsoleProductID(request.ConsoleProductID)
	if !h.assertConsoleProductAccess(w, r, request.ConsoleProductID) {
		return
	}
	if err := h.prepareScannerTarget("semgrep", &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var owner int64
	if uid, ok := ConsoleUserFromRequest(r); ok {
		owner = uid
	}

	passport, err := h.orchestrator.RunScan(r.Context(), "semgrep", request, owner)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, passport)
}

func (h *Handler) handleGitleaksScan(w http.ResponseWriter, r *http.Request) {
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
		request.ScannerName = "gitleaks"
	}
	request.ConsoleProductID = normalizeConsoleProductID(request.ConsoleProductID)
	if !h.assertConsoleProductAccess(w, r, request.ConsoleProductID) {
		return
	}
	if err := h.prepareScannerTarget("gitleaks", &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var owner int64
	if uid, ok := ConsoleUserFromRequest(r); ok {
		owner = uid
	}

	passport, err := h.orchestrator.RunGitleaksScenario(r.Context(), request, owner)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, passport)
}

func (h *Handler) handleScaScan(w http.ResponseWriter, r *http.Request) {
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
		request.ScannerName = "trivy-sca"
	}
	request.ConsoleProductID = normalizeConsoleProductID(request.ConsoleProductID)
	if !h.assertConsoleProductAccess(w, r, request.ConsoleProductID) {
		return
	}
	if err := h.prepareScannerTarget("trivy-sca", &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var owner int64
	if uid, ok := ConsoleUserFromRequest(r); ok {
		owner = uid
	}

	passport, err := h.orchestrator.RunScaScenario(r.Context(), request, owner)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, passport)
}

func (h *Handler) handleDastScan(w http.ResponseWriter, r *http.Request) {
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
		request.ScannerName = "zap-dast"
	}
	request.ConsoleProductID = normalizeConsoleProductID(request.ConsoleProductID)
	if !h.assertConsoleProductAccess(w, r, request.ConsoleProductID) {
		return
	}
	if err := h.prepareScannerTarget("zap-dast", &request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var owner int64
	if uid, ok := ConsoleUserFromRequest(r); ok {
		owner = uid
	}

	passport, err := h.orchestrator.RunDastScenario(r.Context(), request, owner)
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
