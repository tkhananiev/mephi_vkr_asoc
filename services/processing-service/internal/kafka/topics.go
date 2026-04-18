package kafka

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// EnsureTopics создаёт топики с одной партицией; повторяет попытки, пока брокер Kafka не готов (compose / KRaft).
func EnsureTopics(ctx context.Context, brokers []string, topicNames ...string) error {
	if len(brokers) == 0 {
		return errors.New("no kafka brokers")
	}
	const maxWait = 90 * time.Second
	start := time.Now()
	backoff := 500 * time.Millisecond
	var lastErr error
	for attempt := 1; time.Since(start) < maxWait; attempt++ {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("ensure topics: %w (last error: %v)", ctx.Err(), lastErr)
			}
			return ctx.Err()
		default:
		}

		attemptCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		err := ensureTopicsOnce(attemptCtx, brokers, topicNames...)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		if !isRetriableKafkaBootstrapErr(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("ensure topics: %w (last error: %v)", ctx.Err(), lastErr)
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}
	}
	return fmt.Errorf("kafka ensure topics: timeout after %s: %w", maxWait, lastErr)
}

func isRetriableKafkaBootstrapErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "broker not available") ||
		strings.Contains(s, "leader not available") ||
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "no route to host")
}

func ensureTopicsOnce(ctx context.Context, brokers []string, topicNames ...string) error {
	addr := kafkago.TCP(brokers[0])
	cli := &kafkago.Client{Addr: addr}
	topics := make([]kafkago.TopicConfig, len(topicNames))
	for i, name := range topicNames {
		topics[i] = kafkago.TopicConfig{
			Topic:             name,
			NumPartitions:     1,
			ReplicationFactor: 1,
		}
	}
	resp, err := cli.CreateTopics(ctx, &kafkago.CreateTopicsRequest{
		Addr:   addr,
		Topics: topics,
	})
	if err != nil {
		return fmt.Errorf("create topics: %w", err)
	}
	for name, e := range resp.Errors {
		if e == nil {
			continue
		}
		if errors.Is(e, kafkago.TopicAlreadyExists) {
			continue
		}
		if name != "" {
			return fmt.Errorf("topic %q: %w", name, e)
		}
	}
	return nil
}
