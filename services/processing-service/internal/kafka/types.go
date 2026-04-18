package kafka

import "mephi_vkr_aspm/services/processing-service/internal/models"

// IngestEnvelope сообщение в топик aspm.findings.ingest.
type IngestEnvelope struct {
	CorrelationID string             `json:"correlation_id"`
	Ingest        models.IngestRequest `json:"ingest"`
}

// IngestResultEnvelope ответ в топик aspm.findings.ingest.result.
type IngestResultEnvelope struct {
	CorrelationID string                 `json:"correlation_id"`
	Processing    *models.ProcessingResult `json:"processing,omitempty"`
	Error         *string                  `json:"error,omitempty"`
}
