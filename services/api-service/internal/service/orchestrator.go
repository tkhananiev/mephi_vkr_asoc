package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	apikafka "mephi_vkr_asoc/services/api-service/internal/kafka"
	"mephi_vkr_asoc/services/api-service/internal/models"
)

const processingConsoleUserHeader = "X-ASOC-Console-User-ID"

var ErrUnsupportedScannerID = errors.New("unsupported scanner_id")

type DynamicScannerLookup func(scannerID string) (invokeURL string, scannerName string, runnerCommand string, ok bool)

type Orchestrator struct {
	processingURL string
	jiraURL       string
	semgrepURL    string
	gitleaksURL   string
	scaURL        string
	dastURL       string
	adapterURL    string
	httpClient    *http.Client
	kafkaIngest   *apikafka.IngestBridge
	dynamicLookup DynamicScannerLookup
}

func New(processingURL, jiraURL, semgrepURL, gitleaksURL, scaURL, dastURL, adapterURL string, kafkaIngest *apikafka.IngestBridge, dynamicLookup DynamicScannerLookup) *Orchestrator {
	return &Orchestrator{
		processingURL: strings.TrimRight(processingURL, "/"),
		jiraURL:       strings.TrimRight(jiraURL, "/"),
		semgrepURL:    strings.TrimRight(semgrepURL, "/"),
		gitleaksURL:   strings.TrimRight(gitleaksURL, "/"),
		scaURL:        strings.TrimRight(scaURL, "/"),
		dastURL:       strings.TrimRight(dastURL, "/"),
		adapterURL:    strings.TrimRight(adapterURL, "/"),
		httpClient:    &http.Client{Timeout: 10 * time.Minute},
		kafkaIngest:   kafkaIngest,
		dynamicLookup: dynamicLookup,
	}
}

func (o *Orchestrator) RunScan(ctx context.Context, scannerID string, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	id := strings.ToLower(strings.TrimSpace(scannerID))
	switch id {
	case "semgrep":
		req := request
		if strings.TrimSpace(req.ScannerName) == "" {
			req.ScannerName = "semgrep"
		}
		return o.runSemgrepScenario(ctx, req, ownerUserID)
	case "gitleaks":
		req := request
		if strings.TrimSpace(req.ScannerName) == "" {
			req.ScannerName = "gitleaks"
		}
		return o.runGitleaksScenario(ctx, req, ownerUserID)
	case "trivy-sca", "sca", "trivy":
		req := request
		if strings.TrimSpace(req.ScannerName) == "" {
			req.ScannerName = "trivy-sca"
		}
		return o.runScaScenario(ctx, req, ownerUserID)
	case "zap-dast", "dast", "zap":
		req := request
		if strings.TrimSpace(req.ScannerName) == "" {
			req.ScannerName = "zap-dast"
		}
		return o.runDastScenario(ctx, req, ownerUserID)
	default:
		if o.dynamicLookup != nil {
			if invokeURL, scannerName, runnerCmd, ok := o.dynamicLookup(id); ok && strings.TrimSpace(invokeURL) != "" {
				return o.runDynamicHTTPScannerScenario(ctx, id, invokeURL, scannerName, runnerCmd, request, ownerUserID)
			}
		}
		return models.PassportResponse{}, fmt.Errorf("%w: %q (supported: semgrep, gitleaks, trivy-sca, zap-dast, or additional catalog with scanner_invoke_url)", ErrUnsupportedScannerID, id)
	}
}

func (o *Orchestrator) RunSemgrepScenario(ctx context.Context, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	return o.RunScan(ctx, "semgrep", request, ownerUserID)
}

func (o *Orchestrator) RunGitleaksScenario(ctx context.Context, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	return o.RunScan(ctx, "gitleaks", request, ownerUserID)
}

// RunScaScenario — POST /api/v1/scans/sca (Trivy SCA).
func (o *Orchestrator) RunScaScenario(ctx context.Context, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	return o.RunScan(ctx, "trivy-sca", request, ownerUserID)
}

func (o *Orchestrator) RunDastScenario(ctx context.Context, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	return o.RunScan(ctx, "zap-dast", request, ownerUserID)
}

func (o *Orchestrator) runSemgrepScenario(ctx context.Context, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	raw, err := o.callSemgrepService(ctx, request)
	if err != nil {
		return models.PassportResponse{}, err
	}
	findings, err := o.adaptScannerOutput(ctx, "semgrep", raw, "")
	if err != nil {
		return models.PassportResponse{}, err
	}
	return o.passportAfterFindings(ctx, request, findings, ownerUserID)
}

func (o *Orchestrator) passportAfterFindings(ctx context.Context, request models.ScanRequest, findings []models.ProcessingFindingItem, ownerUserID int64) (models.PassportResponse, error) {
	ingest := models.ProcessingIngestRequest{
		ScannerName: request.ScannerName,
		Findings:    findings,
		Channel:     "manual",
	}
	if ownerUserID > 0 {
		id := ownerUserID
		ingest.OwnerUserID = &id
	}
	if request.ConsoleProductID != nil && *request.ConsoleProductID > 0 {
		cid := *request.ConsoleProductID
		ingest.ConsoleProductID = &cid
	}
	processingResponse, err := o.IngestFindings(ctx, ingest)
	if err != nil {
		return models.PassportResponse{}, err
	}

	groups := []models.GroupResponse{}
	if ownerUserID > 0 {
		var err error
		groups, err = o.fetchGroups(ctx, ownerUserID)
		if err != nil {
			return models.PassportResponse{}, err
		}
	}

	scanLabel := DescribeScanTarget(request)
	return models.PassportResponse{
		ScannerName: request.ScannerName,
		ScanTarget:  scanLabel,
		Findings:    findings,
		Processing:  processingResponse,
		Groups:      groups,
		Tickets:     []models.TicketResponse{},
	}, nil
}

func (o *Orchestrator) runGitleaksScenario(ctx context.Context, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	rawBody, err := o.callGitleaksService(ctx, request)
	if err != nil {
		return models.PassportResponse{}, err
	}
	findings, err := o.adaptScannerOutput(ctx, "gitleaks", rawBody, "")
	if err != nil {
		return models.PassportResponse{}, err
	}
	return o.passportAfterFindings(ctx, request, findings, ownerUserID)
}

func (o *Orchestrator) runScaScenario(ctx context.Context, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	rawBody, err := o.callScaService(ctx, request)
	if err != nil {
		return models.PassportResponse{}, err
	}
	findings, err := o.adaptScannerOutput(ctx, "trivy", rawBody, "")
	if err != nil {
		return models.PassportResponse{}, err
	}
	return o.passportAfterFindings(ctx, request, findings, ownerUserID)
}

func (o *Orchestrator) runDastScenario(ctx context.Context, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	rawBody, err := o.callDastService(ctx, request)
	if err != nil {
		return models.PassportResponse{}, err
	}
	findings, err := o.adaptScannerOutput(ctx, "auto", rawBody, request.TargetURL)
	if err != nil {
		return models.PassportResponse{}, err
	}
	return o.passportAfterFindings(ctx, request, findings, ownerUserID)
}

func (o *Orchestrator) runDynamicHTTPScannerScenario(ctx context.Context, scannerID string, invokeURL string, scannerName string, runnerCommand string, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	reqCopy := request
	if strings.TrimSpace(reqCopy.ScannerName) == "" {
		reqCopy.ScannerName = scannerName
	}
	raw, err := o.postDynamicScanPayload(ctx, invokeURL, reqCopy, scannerID, runnerCommand)
	if err != nil {
		return models.PassportResponse{}, err
	}
	findings, err := o.adaptScannerOutput(ctx, "auto", raw, request.TargetURL)
	if err != nil {
		return models.PassportResponse{}, fmt.Errorf("scanner %q: %w", scannerID, err)
	}
	return o.passportAfterFindings(ctx, reqCopy, findings, ownerUserID)
}

func mergeDynamicScannerJSON(request models.ScanRequest, scannerID string, runnerCommand string) ([]byte, error) {
	b, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	sidRaw, err := json.Marshal(strings.TrimSpace(scannerID))
	if err != nil {
		return nil, err
	}
	m["scanner_id"] = sidRaw
	rc := strings.TrimSpace(runnerCommand)
	if rc != "" {
		rcRaw, err := json.Marshal(rc)
		if err != nil {
			return nil, err
		}
		m["runner_command"] = rcRaw
	}
	return json.Marshal(m)
}

func (o *Orchestrator) postDynamicScanPayload(ctx context.Context, invokeURL string, request models.ScanRequest, scannerID string, runnerCommand string) ([]byte, error) {
	var body []byte
	var err error
	if strings.TrimSpace(runnerCommand) != "" {
		body, err = mergeDynamicScannerJSON(request, scannerID, runnerCommand)
	} else {
		body, err = json.Marshal(request)
	}
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, invokeURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := fmt.Sprintf("scanner HTTP %d", resp.StatusCode)
		if errBody["error"] != "" {
			msg = errBody["error"]
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return io.ReadAll(resp.Body)
}

type gitleaksScanRequest struct {
	TargetPath       string `json:"target_path,omitempty"`
	GitRepositoryURL string `json:"git_repository_url,omitempty"`
	GitRepositoryRef string `json:"git_repository_ref,omitempty"`
}

func (o *Orchestrator) callGitleaksService(ctx context.Context, request models.ScanRequest) ([]byte, error) {
	body, err := json.Marshal(gitleaksScanRequest{
		TargetPath:       request.TargetPath,
		GitRepositoryURL: request.GitRepositoryURL,
		GitRepositoryRef: request.GitRepositoryRef,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.gitleaksURL+"/api/v1/scan", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := fmt.Sprintf("gitleaks-service returned status %d", resp.StatusCode)
		if errBody["error"] != "" {
			msg = errBody["error"]
		}
		return nil, fmt.Errorf("%s", msg)
	}

	return io.ReadAll(resp.Body)
}

type scaScanRequest struct {
	TargetPath       string `json:"target_path,omitempty"`
	GitRepositoryURL string `json:"git_repository_url,omitempty"`
	GitRepositoryRef string `json:"git_repository_ref,omitempty"`
}

func (o *Orchestrator) callScaService(ctx context.Context, request models.ScanRequest) ([]byte, error) {
	body, err := json.Marshal(scaScanRequest{
		TargetPath:       request.TargetPath,
		GitRepositoryURL: request.GitRepositoryURL,
		GitRepositoryRef: request.GitRepositoryRef,
	})
	if err != nil {
		return nil, err
	}
	return o.postExecutorScan(ctx, o.scaURL+"/api/v1/scan", body, "trivy-sca-service")
}

type dastScanRequest struct {
	TargetURL string `json:"target_url"`
}

func (o *Orchestrator) callDastService(ctx context.Context, request models.ScanRequest) ([]byte, error) {
	body, err := json.Marshal(dastScanRequest{TargetURL: strings.TrimSpace(request.TargetURL)})
	if err != nil {
		return nil, err
	}
	return o.postExecutorScan(ctx, o.dastURL+"/api/v1/scan", body, "zap-dast-service")
}

func (o *Orchestrator) postExecutorScan(ctx context.Context, url string, body []byte, serviceLabel string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := fmt.Sprintf("%s returned status %d", serviceLabel, resp.StatusCode)
		if errBody["error"] != "" {
			msg = errBody["error"]
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return io.ReadAll(resp.Body)
}

type semgrepScanRequest struct {
	TargetPath       string `json:"target_path,omitempty"`
	SemgrepConfig    string `json:"semgrep_config,omitempty"`
	GitRepositoryURL string `json:"git_repository_url,omitempty"`
	GitRepositoryRef string `json:"git_repository_ref,omitempty"`
}

func DescribeScanTarget(r models.ScanRequest) string {
	if u := strings.TrimSpace(r.TargetURL); u != "" {
		return u
	}
	if strings.TrimSpace(r.GitRepositoryURL) != "" {
		ref := strings.TrimSpace(r.GitRepositoryRef)
		if ref == "" {
			ref = "(default-branch)"
		}
		sub := strings.TrimSpace(r.TargetPath)
		if sub != "" {
			return fmt.Sprintf("%s@%s/%s", strings.TrimSpace(r.GitRepositoryURL), ref, sub)
		}
		return fmt.Sprintf("%s@%s", strings.TrimSpace(r.GitRepositoryURL), ref)
	}
	return r.TargetPath
}

func (o *Orchestrator) callSemgrepService(ctx context.Context, request models.ScanRequest) ([]byte, error) {
	body, err := json.Marshal(semgrepScanRequest{
		TargetPath:       request.TargetPath,
		SemgrepConfig:    request.SemgrepConfig,
		GitRepositoryURL: request.GitRepositoryURL,
		GitRepositoryRef: request.GitRepositoryRef,
	})
	if err != nil {
		return nil, err
	}
	return o.postExecutorScan(ctx, o.semgrepURL+"/api/v1/scan", body, "semgrep-service")
}

func (o *Orchestrator) adaptScannerOutput(ctx context.Context, format string, raw []byte, targetURL string) ([]models.ProcessingFindingItem, error) {
	if strings.TrimSpace(o.adapterURL) == "" {
		return nil, fmt.Errorf("findings-adapter URL not configured")
	}
	url := o.adapterURL + "/api/v1/adapt/" + strings.TrimSpace(format)
	if strings.TrimSpace(targetURL) != "" {
		url += "?target_url=" + urlQueryEscape(targetURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		var errBody map[string]string
		_ = json.Unmarshal(respBody, &errBody)
		msg := fmt.Sprintf("findings-adapter returned status %d", resp.StatusCode)
		if errBody["error"] != "" {
			msg = errBody["error"]
		}
		return nil, fmt.Errorf("%s", msg)
	}

	var adapted struct {
		Findings []models.ProcessingFindingItem `json:"findings"`
	}
	if err := json.Unmarshal(respBody, &adapted); err != nil {
		return nil, fmt.Errorf("decode adapter response: %w", err)
	}
	if adapted.Findings == nil {
		return []models.ProcessingFindingItem{}, nil
	}
	return adapted.Findings, nil
}

func urlQueryEscape(s string) string {
	return url.QueryEscape(s)
}

func (o *Orchestrator) KafkaIngestEnabled() bool {
	return o.kafkaIngest != nil
}

func (o *Orchestrator) EnqueueFindings(ctx context.Context, ingest models.ProcessingIngestRequest) (string, error) {
	if o.kafkaIngest == nil {
		return "", fmt.Errorf("kafka ingest not configured")
	}
	return o.kafkaIngest.Publish(ctx, ingest)
}

// Поле OwnerUserID в payload задаёт владельца для мультиарендности консоли (как в ingest на processing).
func (o *Orchestrator) IngestFindings(ctx context.Context, ingest models.ProcessingIngestRequest) (models.ProcessingResponse, error) {
	var ownerHdr int64
	if ingest.OwnerUserID != nil && *ingest.OwnerUserID > 0 {
		ownerHdr = *ingest.OwnerUserID
	}
	if o.kafkaIngest != nil {
		return o.kafkaIngest.PublishAndWait(ctx, ingest)
	}
	return o.sendToProcessing(ctx, ingest, ownerHdr)
}

func (o *Orchestrator) sendToProcessing(ctx context.Context, payload models.ProcessingIngestRequest, ownerUserID int64) (models.ProcessingResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return models.ProcessingResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.processingURL+"/api/v1/findings/ingest", bytes.NewReader(body))
	if err != nil {
		return models.ProcessingResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if ownerUserID > 0 {
		req.Header.Set(processingConsoleUserHeader, strconv.FormatInt(ownerUserID, 10))
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return models.ProcessingResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return models.ProcessingResponse{}, fmt.Errorf("processing-service returned status %d", resp.StatusCode)
	}

	var result models.ProcessingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return models.ProcessingResponse{}, err
	}
	return result, nil
}

func (o *Orchestrator) fetchGroups(ctx context.Context, ownerUserID int64) ([]models.GroupResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.processingURL+"/api/v1/groups?limit=20", nil)
	if err != nil {
		return nil, err
	}
	if ownerUserID > 0 {
		req.Header.Set(processingConsoleUserHeader, strconv.FormatInt(ownerUserID, 10))
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("processing-service groups returned status %d", resp.StatusCode)
	}

	var result []models.GroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (o *Orchestrator) CreateGroupTicket(ctx context.Context, payload models.TicketRequest) (models.TicketResponse, error) {
	return o.createTicket(ctx, payload)
}

func (o *Orchestrator) createTicket(ctx context.Context, payload models.TicketRequest) (models.TicketResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return models.TicketResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.jiraURL+"/api/v1/tickets", bytes.NewReader(body))
	if err != nil {
		return models.TicketResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return models.TicketResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return models.TicketResponse{}, fmt.Errorf("jira-integration-service returned status %d", resp.StatusCode)
	}

	var result models.TicketResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return models.TicketResponse{}, err
	}
	return result, nil
}
