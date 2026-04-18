package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/google/uuid"

	"mephi_vkr_aspm/services/api-service/internal/models"
)

// IngestBridge публикует находки в Kafka и ждёт ответ в топике результатов (одна партиция).
type IngestBridge struct {
	brokers     []string
	ingestTopic string
	resultTopic string
	writer      *kafkago.Writer
}

func NewIngestBridge(brokers []string, ingestTopic, resultTopic string) *IngestBridge {
	w := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Topic:        ingestTopic,
		RequiredAcks: kafkago.RequireAll,
		Async:        false,
		Balancer:     &kafkago.LeastBytes{},
	}
	return &IngestBridge{
		brokers:     brokers,
		ingestTopic: ingestTopic,
		resultTopic: resultTopic,
		writer:      w,
	}
}

func (b *IngestBridge) Close() error {
	return b.writer.Close()
}

// PublishAndWait отправляет ingest и блокируется до ответа processing или таймаута.
func (b *IngestBridge) PublishAndWait(ctx context.Context, payload models.ProcessingIngestRequest) (models.ProcessingResponse, error) {
	corr := uuid.New().String()
	env := IngestEnvelope{CorrelationID: corr, Ingest: payload}
	body, err := json.Marshal(env)
	if err != nil {
		return models.ProcessingResponse{}, err
	}

	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:     b.brokers,
		Topic:       b.resultTopic,
		Partition:   0,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafkago.LastOffset,
	})
	defer func() { _ = r.Close() }()

	if err := b.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(corr),
		Value: body,
	}); err != nil {
		return models.ProcessingResponse{}, fmt.Errorf("kafka write ingest: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	for {
		msg, err := r.ReadMessage(waitCtx)
		if err != nil {
			return models.ProcessingResponse{}, fmt.Errorf("kafka read result: %w", err)
		}
		var out IngestResultEnvelope
		if err := json.Unmarshal(msg.Value, &out); err != nil {
			continue
		}
		if out.CorrelationID != corr {
			continue
		}
		if out.Error != nil && *out.Error != "" {
			return models.ProcessingResponse{}, fmt.Errorf("processing: %s", *out.Error)
		}
		if out.Processing == nil {
			return models.ProcessingResponse{}, fmt.Errorf("processing: empty result")
		}
		return *out.Processing, nil
	}
}
