package kafka

import "mephi_vkr_aspm/services/api-service/internal/models"

// IngestEnvelope должно совпадать по JSON с processing-service/internal/kafka.
type IngestEnvelope struct {
	CorrelationID string                       `json:"correlation_id"`
	Ingest        models.ProcessingIngestRequest `json:"ingest"`
}

// IngestResultEnvelope — ответ из топика aspm.findings.ingest.result.
type IngestResultEnvelope struct {
	CorrelationID string                   `json:"correlation_id"`
	Processing    *models.ProcessingResponse `json:"processing,omitempty"`
	Error         *string                  `json:"error,omitempty"`
}
