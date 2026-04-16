package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mephi_vkr_aspm/services/jira-integration-service/internal/models"
	"mephi_vkr_aspm/services/jira-integration-service/internal/repository"
)

type TicketService struct {
	repo       repository.TicketRepository
	baseURL    string
	projectKey string
	httpClient *http.Client
}

func New(repo repository.TicketRepository, baseURL, projectKey string) *TicketService {
	return &TicketService{
		repo:       repo,
		baseURL:    strings.TrimRight(baseURL, "/"),
		projectKey: projectKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *TicketService) CreateOrUpdateTicket(ctx context.Context, req models.TicketRequest) (models.TicketResponse, error) {
	idempotencyKey := fmt.Sprintf("group:%d:%s", req.GroupID, req.GroupKey)
	issueKey, issueURL, err := s.createTicketInJira(ctx, req)
	if err != nil {
		return models.TicketResponse{}, err
	}

	if err := s.repo.UpsertTicketLink(ctx, req, issueKey, issueURL, "synced", idempotencyKey); err != nil {
		return models.TicketResponse{}, err
	}

	return models.TicketResponse{
		GroupID:        req.GroupID,
		JiraIssueKey:   issueKey,
		JiraIssueURL:   issueURL,
		SyncStatus:     "synced",
		IdempotencyKey: idempotencyKey,
	}, nil
}

func (s *TicketService) createTicketInJira(ctx context.Context, req models.TicketRequest) (string, string, error) {
	payload := map[string]any{
		"fields": map[string]any{
			"project": map[string]any{
				"key": s.projectKey,
			},
			"summary": fmt.Sprintf("Vulnerability group %s", req.GroupKey),
			"description": fmt.Sprintf(
				"Severity: %s\nAssets count: %d\nCorrelation ref: %s",
				req.Severity,
				req.AssetsCount,
				req.CorrelationRef,
			),
			"issuetype": map[string]any{
				"name": "Task",
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/rest/api/2/issue", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("jira mock returned status %d", resp.StatusCode)
	}

	var jiraResp struct {
		Key  string `json:"key"`
		Self string `json:"self"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jiraResp); err != nil {
		return "", "", err
	}

	return jiraResp.Key, jiraResp.Self, nil
}
