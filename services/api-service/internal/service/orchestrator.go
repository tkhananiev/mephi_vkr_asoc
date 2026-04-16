package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"mephi_vkr_aspm/services/api-service/internal/models"
)

type Orchestrator struct {
	processingURL string
	jiraURL       string
	semgrepURL    string
	httpClient    *http.Client
}

func New(processingURL, jiraURL, semgrepURL string) *Orchestrator {
	return &Orchestrator{
		processingURL: strings.TrimRight(processingURL, "/"),
		jiraURL:       strings.TrimRight(jiraURL, "/"),
		semgrepURL:    strings.TrimRight(semgrepURL, "/"),
		httpClient:    &http.Client{Timeout: 10 * time.Minute},
	}
}

func (o *Orchestrator) RunSemgrepScenario(ctx context.Context, request models.ScanRequest) (models.PassportResponse, error) {
	scanResult, err := o.callSemgrepService(ctx, request.TargetPath, request.SemgrepConfig)
	if err != nil {
		return models.PassportResponse{}, err
	}

	findings := make([]models.ProcessingFindingItem, 0, len(scanResult.Results))
	for _, result := range scanResult.Results {
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

	processingResponse, err := o.sendToProcessing(ctx, models.ProcessingIngestRequest{
		ScannerName: request.ScannerName,
		Findings:    findings,
	})
	if err != nil {
		return models.PassportResponse{}, err
	}

	groups, err := o.fetchGroups(ctx)
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

	return models.PassportResponse{
		ScannerName: request.ScannerName,
		ScanTarget:  request.TargetPath,
		Findings:    findings,
		Processing:  processingResponse,
		Groups:      groups,
		Tickets:     tickets,
	}, nil
}

type semgrepScanRequest struct {
	TargetPath    string `json:"target_path"`
	SemgrepConfig string `json:"semgrep_config,omitempty"`
}

func (o *Orchestrator) callSemgrepService(ctx context.Context, targetPath, semgrepConfig string) (models.SemgrepResult, error) {
	body, err := json.Marshal(semgrepScanRequest{
		TargetPath:    targetPath,
		SemgrepConfig: semgrepConfig,
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

	var result models.SemgrepResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return models.SemgrepResult{}, err
	}
	return result, nil
}

func (o *Orchestrator) sendToProcessing(ctx context.Context, payload models.ProcessingIngestRequest) (models.ProcessingResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return models.ProcessingResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.processingURL+"/api/v1/findings/ingest", bytes.NewReader(body))
	if err != nil {
		return models.ProcessingResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

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

func (o *Orchestrator) fetchGroups(ctx context.Context) ([]models.GroupResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.processingURL+"/api/v1/groups?limit=20", nil)
	if err != nil {
		return nil, err
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
