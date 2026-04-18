package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort           string
	PostgresDSN        string
	KafkaBrokers       []string
	KafkaTopicIngest   string
	KafkaTopicResult   string
	KafkaIngestEnabled bool
}

func Load() Config {
	brokers := splitCSV(getEnv("APP_KAFKA_BROKERS", ""))
	return Config{
		HTTPPort:    getEnv("APP_HTTP_PORT", "8082"),
		PostgresDSN: getEnv("APP_POSTGRES_DSN", "postgres://aspm:aspm@localhost:5432/aspm?sslmode=disable"),
		KafkaBrokers: brokers,
		KafkaTopicIngest: getEnv("APP_KAFKA_TOPIC_FINDINGS_INGEST", "aspm.findings.ingest"),
		KafkaTopicResult: getEnv("APP_KAFKA_TOPIC_FINDINGS_RESULT", "aspm.findings.ingest.result"),
		KafkaIngestEnabled: len(brokers) > 0,
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
