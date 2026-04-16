package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"mephi_vkr_aspm/services/jira-integration-service/internal/models"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) UpsertTicketLink(ctx context.Context, req models.TicketRequest, issueKey, issueURL, status, idempotencyKey string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO integration.ticket_links (
			group_id, jira_issue_key, jira_issue_url, sync_status, idempotency_key
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (group_id)
		DO UPDATE SET
			jira_issue_key = EXCLUDED.jira_issue_key,
			jira_issue_url = EXCLUDED.jira_issue_url,
			sync_status = EXCLUDED.sync_status,
			idempotency_key = EXCLUDED.idempotency_key,
			updated_at = NOW()
	`, req.GroupID, issueKey, issueURL, status, idempotencyKey)
	return err
}
