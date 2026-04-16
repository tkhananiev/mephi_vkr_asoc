package repository

import (
	"context"

	"mephi_vkr_aspm/services/jira-integration-service/internal/models"
)

type TicketRepository interface {
	UpsertTicketLink(ctx context.Context, req models.TicketRequest, issueKey, issueURL, status, idempotencyKey string) error
}
