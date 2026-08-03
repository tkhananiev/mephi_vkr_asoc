package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"mephi_vkr_asoc/services/processing-service/internal/models"
	"mephi_vkr_asoc/services/processing-service/internal/service"
)

type findingsProcessor interface {
	ProcessFindings(ctx context.Context, request models.IngestRequest) (models.ProcessingResult, error)
}

type IngestConsumer struct {
	brokers     []string
	ingestTopic string
	resultTopic string
	groupID     string
	svc         findingsProcessor

	// retryWait is the pause between ProcessFindings retries for the same
	// Kafka message. Overridable in tests.
	retryWait time.Duration
}

func NewIngestConsumer(brokers []string, ingestTopic, resultTopic string, svc *service.ProcessingService) *IngestConsumer {
	return &IngestConsumer{
		brokers:     brokers,
		ingestTopic: ingestTopic,
		resultTopic: resultTopic,
		groupID:     "processing-findings-ingest",
		svc:         svc,
		retryWait:   2 * time.Second,
	}
}

func (c *IngestConsumer) Run(ctx context.Context) error {
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  c.brokers,
		GroupID:  c.groupID,
		Topic:    c.ingestTopic,
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer func() { _ = reader.Close() }()

	writer := &kafkago.Writer{
		Addr:         kafkago.TCP(c.brokers...),
		Topic:        c.resultTopic,
		RequiredAcks: kafkago.RequireAll,
		Async:        false,
		Balancer:     &kafkago.LeastBytes{},
	}
	defer func() { _ = writer.Close() }()

	log.Printf("kafka ingest consumer started: topic=%s group=%s", c.ingestTopic, c.groupID)

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		var env IngestEnvelope
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			log.Printf("kafka: skip invalid envelope: %v", err)
			if err := reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("kafka: commit after bad json: %v", err)
			}
			continue
		}
		if env.CorrelationID == "" {
			log.Printf("kafka: skip empty correlation_id")
			if err := reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("kafka: commit: %v", err)
			}
			continue
		}

		result, err := c.processWithRetry(ctx, writer, env)
		if err != nil {
			// Context cancelled while retrying: do not commit. The message remains
			// at the last committed offset for redelivery after restart.
			return err
		}

		out := IngestResultEnvelope{
			CorrelationID: env.CorrelationID,
			Processing:    &result,
		}
		payload, err := json.Marshal(out)
		if err != nil {
			log.Printf("kafka: marshal result: %v", err)
			continue
		}

		if err := writer.WriteMessages(ctx, kafkago.Message{
			Key:   []byte(env.CorrelationID),
			Value: payload,
		}); err != nil {
			log.Printf("kafka: write result: %v", err)
			continue
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("kafka: commit: %v", err)
		}
	}
}

type resultWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafkago.Message) error
}

// processWithRetry runs ProcessFindings until it succeeds. On failure it publishes
// an error envelope (so PublishAndWait callers unblock) and retries the same
// payload without advancing the consumer offset.
//
// This is required because ProcessFindings purges prior scanner rows before
// insert: committing a failed message permanently drops both the previous DB
// snapshot and the only Kafka copy of the replacement findings. Kafka offsets
// are a per-partition high-water mark, so we must not fetch/commit later
// messages until this one succeeds.
func (c *IngestConsumer) processWithRetry(ctx context.Context, writer resultWriter, env IngestEnvelope) (models.ProcessingResult, error) {
	wait := c.retryWait
	if wait <= 0 {
		wait = 2 * time.Second
	}

	for {
		result, procErr := c.svc.ProcessFindings(ctx, env.Ingest)
		if procErr == nil {
			return result, nil
		}

		errMsg := procErr.Error()
		out := IngestResultEnvelope{CorrelationID: env.CorrelationID, Error: &errMsg}
		if payload, err := json.Marshal(out); err != nil {
			log.Printf("kafka: marshal error result: %v", err)
		} else if err := writer.WriteMessages(ctx, kafkago.Message{
			Key:   []byte(env.CorrelationID),
			Value: payload,
		}); err != nil {
			log.Printf("kafka: write error result: %v", err)
		}

		log.Printf("kafka: process findings failed (will retry, not committing): correlation_id=%s err=%v", env.CorrelationID, procErr)

		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return models.ProcessingResult{}, ctx.Err()
		case <-timer.C:
		}
	}
}
