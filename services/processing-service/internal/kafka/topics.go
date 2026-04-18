package kafka

import (
	"context"
	"errors"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"
)

// EnsureTopics создаёт топики с одной партицией (для упрощённого reply-паттерна).
func EnsureTopics(ctx context.Context, brokers []string, topicNames ...string) error {
	if len(brokers) == 0 {
		return errors.New("no kafka brokers")
	}
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
