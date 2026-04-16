package kafka

import (
	"context"
	"log"

	"mephi_vkr_aspm/services/reference-data-service/internal/models"
)

type Publisher interface {
	PublishSyncCompleted(ctx context.Context, result models.SyncResult) error
}

type NoopPublisher struct{}

func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{}
}

func (p *NoopPublisher) PublishSyncCompleted(_ context.Context, result models.SyncResult) error {
	log.Printf("kafka noop publish: source=%s run_id=%d processed=%d", result.SourceCode, result.RunID, result.ItemsProcessed)
	return nil
}
