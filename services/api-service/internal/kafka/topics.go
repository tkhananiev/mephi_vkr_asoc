package kafka

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// EnsureTopics создаёт топики и доводит число партиций до заданного (идемпотентно).
func EnsureTopics(ctx context.Context, brokers []string, ingestTopic string, ingestPartitions int, resultTopic string, resultPartitions int) error {
	if len(brokers) == 0 {
		return errors.New("no kafka brokers")
	}
	if ingestPartitions < 1 {
		ingestPartitions = 1
	}
	if resultPartitions < 1 {
		resultPartitions = 1
	}
	const maxWait = 90 * time.Second
	start := time.Now()
	backoff := 500 * time.Millisecond
	var lastErr error
	for time.Since(start) < maxWait {
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("ensure topics: %w (last error: %v)", ctx.Err(), lastErr)
			}
			return ctx.Err()
		default:
		}

		attemptCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		err := ensureTopicsOnce(attemptCtx, brokers, ingestTopic, ingestPartitions, resultTopic, resultPartitions)
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
		if backoff < 5*time.Second {
			backoff *= 2
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
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

func ensureTopicsOnce(ctx context.Context, brokers []string, ingestTopic string, ingestPartitions int, resultTopic string, resultPartitions int) error {
	addr := kafkago.TCP(brokers[0])
	cli := &kafkago.Client{Addr: addr}
	for _, spec := range []struct {
		name string
		n    int
	}{
		{ingestTopic, ingestPartitions},
		{resultTopic, resultPartitions},
	} {
		resp, err := cli.CreateTopics(ctx, &kafkago.CreateTopicsRequest{
			Addr: addr,
			Topics: []kafkago.TopicConfig{{
				Topic:             spec.name,
				NumPartitions:     spec.n,
				ReplicationFactor: 1,
			}},
		})
		if err != nil {
			return fmt.Errorf("create topic %q: %w", spec.name, err)
		}
		for name, e := range resp.Errors {
			if e == nil || errors.Is(e, kafkago.TopicAlreadyExists) {
				continue
			}
			if name != "" {
				return fmt.Errorf("topic %q: %w", name, e)
			}
		}
		if err := ensureMinPartitions(ctx, brokers, spec.name, spec.n); err != nil {
			return err
		}
	}
	return nil
}

func ensureMinPartitions(ctx context.Context, brokers []string, topic string, want int) error {
	conn, err := kafkago.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("dial kafka: %w", err)
	}
	defer func() { _ = conn.Close() }()

	parts, err := conn.ReadPartitions(topic)
	if err != nil {
		return fmt.Errorf("read partitions %q: %w", topic, err)
	}
	have := len(parts)
	if have >= want {
		return nil
	}
	// Count в kafka-go — целевое общее число партиций (не дельта).
	addr := kafkago.TCP(brokers[0])
	cli := &kafkago.Client{Addr: addr}
	resp, err := cli.CreatePartitions(ctx, &kafkago.CreatePartitionsRequest{
		Addr: addr,
		Topics: []kafkago.TopicPartitionsConfig{{
			Name:  topic,
			Count: int32(want),
		}},
	})
	if err != nil {
		return fmt.Errorf("create partitions %q: %w", topic, err)
	}
	for name, e := range resp.Errors {
		if e == nil {
			continue
		}
		if name != "" {
			return fmt.Errorf("partition %q: %w", name, e)
		}
	}
	return nil
}
