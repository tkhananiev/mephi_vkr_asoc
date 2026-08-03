package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"mephi_vkr_asoc/services/processing-service/internal/models"
)

type stubProcessor struct {
	calls   atomic.Int32
	failFor int32
}

func (s *stubProcessor) ProcessFindings(ctx context.Context, request models.IngestRequest) (models.ProcessingResult, error) {
	n := s.calls.Add(1)
	if n <= s.failFor {
		return models.ProcessingResult{}, errors.New("purge prior scanner data: db blip")
	}
	return models.ProcessingResult{RunID: int64(n), FindingsProcessed: 1}, nil
}

type stubWriter struct {
	messages []kafkago.Message
}

func (w *stubWriter) WriteMessages(ctx context.Context, msgs ...kafkago.Message) error {
	w.messages = append(w.messages, msgs...)
	return nil
}

func TestProcessWithRetryRecoversAfterTransientFailure(t *testing.T) {
	t.Parallel()

	proc := &stubProcessor{failFor: 1}
	writer := &stubWriter{}
	c := &IngestConsumer{svc: proc, retryWait: time.Millisecond}

	result, err := c.processWithRetry(context.Background(), writer, IngestEnvelope{
		CorrelationID: "corr-1",
		Ingest:        models.IngestRequest{ScannerName: "gitleaks"},
	})
	if err != nil {
		t.Fatalf("processWithRetry: %v", err)
	}
	if proc.calls.Load() != 2 {
		t.Fatalf("expected 2 ProcessFindings attempts, got %d", proc.calls.Load())
	}
	if result.FindingsProcessed != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(writer.messages) != 1 {
		t.Fatalf("expected one error envelope before success, got %d", len(writer.messages))
	}
	var env IngestResultEnvelope
	if err := json.Unmarshal(writer.messages[0].Value, &env); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if env.Error == nil || *env.Error == "" {
		t.Fatal("expected error envelope for PublishAndWait callers")
	}
	if env.Processing != nil {
		t.Fatal("error envelope must not include processing result")
	}
}

func TestProcessWithRetryStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	proc := &stubProcessor{failFor: 1000}
	c := &IngestConsumer{svc: proc, retryWait: 50 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := c.processWithRetry(ctx, &stubWriter{}, IngestEnvelope{
		CorrelationID: "corr-2",
		Ingest:        models.IngestRequest{ScannerName: "gitleaks"},
	})
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context error, got %v", err)
	}
	if proc.calls.Load() < 1 {
		t.Fatal("expected at least one ProcessFindings attempt before cancel")
	}
}
