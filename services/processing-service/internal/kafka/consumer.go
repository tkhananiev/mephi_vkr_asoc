package kafka

import (
	"context"
	"encoding/json"
	"log"

	kafkago "github.com/segmentio/kafka-go"

	"mephi_vkr_aspm/services/processing-service/internal/service"
)

// IngestConsumer читает aspm.findings.ingest и публикует результат в aspm.findings.ingest.result.
type IngestConsumer struct {
	brokers     []string
	ingestTopic string
	resultTopic string
	groupID     string
	svc         *service.ProcessingService
}

func NewIngestConsumer(brokers []string, ingestTopic, resultTopic string, svc *service.ProcessingService) *IngestConsumer {
	return &IngestConsumer{
		brokers:     brokers,
		ingestTopic: ingestTopic,
		resultTopic: resultTopic,
		groupID:     "processing-findings-ingest",
		svc:         svc,
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

		result, procErr := c.svc.ProcessFindings(ctx, env.Ingest)
		out := IngestResultEnvelope{CorrelationID: env.CorrelationID}
		if procErr != nil {
			errMsg := procErr.Error()
			out.Error = &errMsg
		} else {
			out.Processing = &result
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
