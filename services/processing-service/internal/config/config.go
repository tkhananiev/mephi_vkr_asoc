package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort              string
	PostgresDSN           string
	KafkaBrokers          []string
	KafkaTopicIngest      string
	KafkaTopicResult      string
	KafkaIngestEnabled    bool
	HTTPFindingsIngestEnabled bool
}

func Load() Config {
	brokers := splitCSV(getEnv("APP_KAFKA_BROKERS", ""))
	kafkaOn := len(brokers) > 0
	// С Kafka: приём находок из топика (основной путь). HTTP POST /findings/ingest по умолчанию выкл.
	// Без брокеров (локальный запуск только processing): HTTP ingest остаётся включённым по умолчанию.
	httpFindings := getBoolEnv("APP_HTTP_FINDINGS_INGEST", !kafkaOn)
	return Config{
		HTTPPort:                  getEnv("APP_HTTP_PORT", "8082"),
		PostgresDSN:               getEnv("APP_POSTGRES_DSN", "postgres://asoc:asoc@localhost:5432/asoc?sslmode=disable"),
		KafkaBrokers:              brokers,
		KafkaTopicIngest:          getEnv("APP_KAFKA_TOPIC_FINDINGS_INGEST", "asoc.findings.ingest"),
		KafkaTopicResult:          getEnv("APP_KAFKA_TOPIC_FINDINGS_RESULT", "asoc.findings.ingest.result"),
		KafkaIngestEnabled:        kafkaOn,
		HTTPFindingsIngestEnabled: httpFindings,
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on", "y":
		return true
	default:
		return false
	}
}
