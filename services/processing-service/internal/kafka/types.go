package kafka

import "mephi_vkr_asoc/services/processing-service/internal/models"

type IngestEnvelope struct {
	CorrelationID string             `json:"correlation_id"`
	Ingest        models.IngestRequest `json:"ingest"`
}

type IngestResultEnvelope struct {
	CorrelationID string                 `json:"correlation_id"`
	Processing    *models.ProcessingResult `json:"processing,omitempty"`
	Error         *string                  `json:"error,omitempty"`
}
