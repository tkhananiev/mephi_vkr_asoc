package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	apikafka "mephi_vkr_asoc/services/api-service/internal/kafka"
	"mephi_vkr_asoc/services/api-service/internal/models"
)

const processingConsoleUserHeader = "X-ASOC-Console-User-ID"

// ErrUnsupportedScannerID — scanner_id не зарегистрирован в RunScan (клиенту — 400).
var ErrUnsupportedScannerID = errors.New("unsupported scanner_id")

// DynamicScannerLookup — доп. сканеры из админ-каталога с полем scanner_invoke_url и опционально runner_command.
type DynamicScannerLookup func(scannerID string) (invokeURL string, scannerName string, runnerCommand string, ok bool)

type Orchestrator struct {
	processingURL string
	jiraURL       string
	semgrepURL    string
	gitleaksURL   string
	httpClient    *http.Client
	kafkaIngest   *apikafka.IngestBridge
	dynamicLookup DynamicScannerLookup
}

func New(processingURL, jiraURL, semgrepURL, gitleaksURL string, kafkaIngest *apikafka.IngestBridge, dynamicLookup DynamicScannerLookup) *Orchestrator {
	return &Orchestrator{
		processingURL: strings.TrimRight(processingURL, "/"),
		jiraURL:       strings.TrimRight(jiraURL, "/"),
		semgrepURL:    strings.TrimRight(semgrepURL, "/"),
		gitleaksURL:   strings.TrimRight(gitleaksURL, "/"),
		httpClient:    &http.Client{Timeout: 10 * time.Minute},
		kafkaIngest:   kafkaIngest,
		dynamicLookup: dynamicLookup,
	}
}

// RunScan выполняет сценарий по scanner_id (semgrep / gitleaks → ingest / группы / Jira).
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
	default:
		if o.dynamicLookup != nil {
			if invokeURL, scannerName, runnerCmd, ok := o.dynamicLookup(id); ok && strings.TrimSpace(invokeURL) != "" {
				return o.runDynamicHTTPScannerScenario(ctx, id, invokeURL, scannerName, runnerCmd, request, ownerUserID)
			}
		}
		return models.PassportResponse{}, fmt.Errorf("%w: %q (supported: semgrep, gitleaks, or additional catalog with scanner_invoke_url)", ErrUnsupportedScannerID, id)
	}
}

// RunSemgrepScenario — совместимость с POST /api/v1/scans/semgrep; предпочтительнее POST /api/v1/scans.
func (o *Orchestrator) RunSemgrepScenario(ctx context.Context, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	return o.RunScan(ctx, "semgrep", request, ownerUserID)
}

func (o *Orchestrator) runSemgrepScenario(ctx context.Context, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	scanResult, err := o.callSemgrepService(ctx, request)
	if err != nil {
		return models.PassportResponse{}, err
	}

	findings := findingsFromSemgrepResult(scanResult)
	return o.passportAfterFindings(ctx, request, findings, ownerUserID)
}

func findingsFromSemgrepResult(sr models.SemgrepResult) []models.ProcessingFindingItem {
	findings := make([]models.ProcessingFindingItem, 0, len(sr.Results))
	for _, result := range sr.Results {
		cwe := ""
		if len(result.Extra.Metadata.CWE) > 0 {
			cwe = result.Extra.Metadata.CWE[0]
		}

		findings = append(findings, models.ProcessingFindingItem{
			AssetID:    filepath.Base(result.Path),
			Identifier: result.CheckID,
			Severity:   normalizeSeverity(result.Extra.Severity),
			Component:  result.Path,
			Version:    "",
			CVE:        strings.TrimSpace(result.Extra.Metadata.CVE),
			CWE:        cwe,
			Metadata: map[string]any{
				"message": result.Extra.Message,
				"path":    result.Path,
			},
			RawPayload: map[string]any{
				"check_id": result.CheckID,
			},
		})
	}
	return findings
}

func (o *Orchestrator) passportAfterFindings(ctx context.Context, request models.ScanRequest, findings []models.ProcessingFindingItem, ownerUserID int64) (models.PassportResponse, error) {
	ingest := models.ProcessingIngestRequest{
		ScannerName: request.ScannerName,
		Findings:    findings,
	}
	if ownerUserID > 0 {
		id := ownerUserID
		ingest.OwnerUserID = &id
	}
	processingResponse, err := o.IngestFindings(ctx, ingest)
	if err != nil {
		return models.PassportResponse{}, err
	}

	groups, err := o.fetchGroups(ctx, ownerUserID)
	if err != nil {
		return models.PassportResponse{}, err
	}

	tickets := make([]models.TicketResponse, 0, len(groups))
	for _, group := range groups {
		ticket, err := o.createTicket(ctx, models.TicketRequest{
			GroupID:        group.ID,
			GroupKey:       group.GroupKey,
			Severity:       group.SeverityMax,
			AssetsCount:    group.AssetsCount,
			CorrelationRef: group.GroupKey,
		})
		if err != nil {
			return models.PassportResponse{}, err
		}
		tickets = append(tickets, ticket)
	}

	scanLabel := DescribeScanTarget(request)
	return models.PassportResponse{
		ScannerName: request.ScannerName,
		ScanTarget:  scanLabel,
		Findings:    findings,
		Processing:  processingResponse,
		Groups:      groups,
		Tickets:     tickets,
	}, nil
}

func (o *Orchestrator) runGitleaksScenario(ctx context.Context, request models.ScanRequest, ownerUserID int64) (models.PassportResponse, error) {
	rawBody, err := o.callGitleaksService(ctx, request)
	if err != nil {
		return models.PassportResponse{}, err
	}
	glFindings, err := parseGitleaksFindings(rawBody)
	if err != nil {
		return models.PassportResponse{}, err
	}

	findings := findingsFromGitleaksFindings(glFindings)

	return o.passportAfterFindings(ctx, request, findings, ownerUserID)
}

func findingsFromGitleaksFindings(gl []models.GitleaksFinding) []models.ProcessingFindingItem {
	findings := make([]models.ProcessingFindingItem, 0, len(gl))
	for _, f := range gl {
		path := strings.TrimSpace(f.File)
		if path == "" {
			path = "unknown"
		}
		id := strings.TrimSpace(f.RuleID)
		if id == "" {
			id = "gitleaks"
		}
		meta := map[string]any{
			"description": f.Description,
			"line":        f.StartLine,
		}
		if len(f.Tags) > 0 {
			meta["tags"] = f.Tags
		}
		findings = append(findings, models.ProcessingFindingItem{
			AssetID:    filepath.Base(path),
			Identifier: id,
			Severity:   "high",
			Component:  path,
			Version:    "",
			CVE:        "",
			CWE:        "",
			Metadata:   meta,
			RawPayload: map[string]any{
				"rule_id":     f.RuleID,
				"fingerprint": f.Fingerprint,
			},
		})
	}
	return findings
}

func parseGitleaksFindings(raw []byte) ([]models.GitleaksFinding, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	var findings []models.GitleaksFinding
	if err := json.Unmarshal(raw, &findings); err != nil {
		return nil, fmt.Errorf("decode gitleaks json: %w", err)
	}
	return findings, nil
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
	findings, err := decodeFlexibleScannerResponse(raw)
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

func decodeFlexibleScannerResponse(raw []byte) ([]models.ProcessingFindingItem, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty response body")
	}
	switch raw[0] {
	case '[':
		var norm []models.ProcessingFindingItem
		if err := json.Unmarshal(raw, &norm); err == nil {
			return norm, nil
		}
		var gl []models.GitleaksFinding
		if err := json.Unmarshal(raw, &gl); err == nil {
			return findingsFromGitleaksFindings(gl), nil
		}
		return nil, fmt.Errorf("JSON array is neither normalized findings nor gitleaks report")
	case '{':
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(raw, &probe); err != nil {
			return nil, err
		}
		if _, ok := probe["results"]; ok {
			var sr models.SemgrepResult
			if err := json.Unmarshal(raw, &sr); err != nil {
				return nil, err
			}
			if len(sr.Results) == 0 && len(sr.Errors) > 0 {
				var msgs []string
				for _, e := range sr.Errors {
					if strings.EqualFold(strings.TrimSpace(e.Level), "error") && strings.TrimSpace(e.Message) != "" {
						msgs = append(msgs, strings.TrimSpace(e.Message))
					}
				}
				if len(msgs) > 0 {
					return nil, fmt.Errorf("%s", strings.Join(msgs, "; "))
				}
			}
			return findingsFromSemgrepResult(sr), nil
		}
		if _, ok := probe["findings"]; ok {
			var wrap struct {
				Findings []models.ProcessingFindingItem `json:"findings"`
			}
			if err := json.Unmarshal(raw, &wrap); err != nil {
				return nil, err
			}
			if wrap.Findings == nil {
				return []models.ProcessingFindingItem{}, nil
			}
			return wrap.Findings, nil
		}
		return nil, fmt.Errorf(`JSON object must contain "results" (semgrep) or "findings" (normalized)`)
	default:
		return nil, fmt.Errorf("response must be JSON array or object")
	}
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

type semgrepScanRequest struct {
	TargetPath       string `json:"target_path,omitempty"`
	SemgrepConfig    string `json:"semgrep_config,omitempty"`
	GitRepositoryURL string `json:"git_repository_url,omitempty"`
	GitRepositoryRef string `json:"git_repository_ref,omitempty"`
}

// DescribeScanTarget — человекочитаемое описание источника для паспорта.
func DescribeScanTarget(r models.ScanRequest) string {
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

func (o *Orchestrator) callSemgrepService(ctx context.Context, request models.ScanRequest) (models.SemgrepResult, error) {
	body, err := json.Marshal(semgrepScanRequest{
		TargetPath:       request.TargetPath,
		SemgrepConfig:    request.SemgrepConfig,
		GitRepositoryURL: request.GitRepositoryURL,
		GitRepositoryRef: request.GitRepositoryRef,
	})
	if err != nil {
		return models.SemgrepResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.semgrepURL+"/api/v1/scan", bytes.NewReader(body))
	if err != nil {
		return models.SemgrepResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return models.SemgrepResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		msg := fmt.Sprintf("semgrep-service returned status %d", resp.StatusCode)
		if errBody["error"] != "" {
			msg = errBody["error"]
		}
		return models.SemgrepResult{}, fmt.Errorf("%s", msg)
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return models.SemgrepResult{}, err
	}

	var result models.SemgrepResult
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return models.SemgrepResult{}, err
	}

	if len(result.Results) == 0 && len(result.Errors) > 0 {
		var msgs []string
		for _, e := range result.Errors {
			if strings.EqualFold(strings.TrimSpace(e.Level), "error") && strings.TrimSpace(e.Message) != "" {
				msgs = append(msgs, strings.TrimSpace(e.Message))
			}
		}
		if len(msgs) > 0 {
			return models.SemgrepResult{}, fmt.Errorf("semgrep: %s", strings.Join(msgs, "; "))
		}
	}

	return result, nil
}

// IngestFindings передаёт уже нормализованные находки в processing (Kafka или HTTP).
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

func normalizeSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "error":
		return "high"
	case "warning":
		return "medium"
	case "info":
		return "low"
	default:
		return "unknown"
	}
}
