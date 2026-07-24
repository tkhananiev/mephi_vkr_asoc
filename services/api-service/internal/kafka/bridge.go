package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"

	"mephi_vkr_asoc/services/api-service/internal/models"
)

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
		Balancer:     &kafkago.Hash{},
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

func (b *IngestBridge) Publish(ctx context.Context, payload models.ProcessingIngestRequest) (string, error) {
	corr := uuid.New().String()
	if err := b.writeIngest(ctx, corr, payload); err != nil {
		return "", err
	}
	return corr, nil
}

func (b *IngestBridge) PublishAndWait(ctx context.Context, payload models.ProcessingIngestRequest) (models.ProcessingResponse, error) {
	corr := uuid.New().String()

	// Capture the result-topic high watermark before publishing so a fast
	// consumer cannot write a reply that a LastOffset reader would skip.
	startOffset, err := b.resultEndOffset(ctx)
	if err != nil {
		return models.ProcessingResponse{}, err
	}

	if err := b.writeIngest(ctx, corr, payload); err != nil {
		return models.ProcessingResponse{}, err
	}

	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:     b.brokers,
		Topic:       b.resultTopic,
		Partition:   0,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: startOffset,
	})
	defer func() { _ = r.Close() }()

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

func (b *IngestBridge) resultEndOffset(ctx context.Context) (int64, error) {
	if len(b.brokers) == 0 {
		return 0, fmt.Errorf("no kafka brokers")
	}
	conn, err := kafkago.DialLeader(ctx, "tcp", b.brokers[0], b.resultTopic, 0)
	if err != nil {
		return 0, fmt.Errorf("kafka dial result topic: %w", err)
	}
	defer func() { _ = conn.Close() }()
	offset, err := conn.ReadLastOffset()
	if err != nil {
		return 0, fmt.Errorf("kafka read result end offset: %w", err)
	}
	return offset, nil
}

func (b *IngestBridge) writeIngest(ctx context.Context, corr string, payload models.ProcessingIngestRequest) error {
	env := IngestEnvelope{CorrelationID: corr, Ingest: payload}
	body, err := json.Marshal(env)
	if err != nil {
		return err
	}
	if err := b.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(corr),
		Value: body,
	}); err != nil {
		return fmt.Errorf("kafka write ingest: %w", err)
	}
	return nil
}
