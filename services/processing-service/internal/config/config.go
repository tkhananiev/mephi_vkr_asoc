package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPPort              string
	PostgresDSN           string
	KafkaBrokers          []string
	KafkaTopicIngest        string
	KafkaTopicResult        string
	KafkaIngestPartitions   int
	KafkaResultPartitions   int
	KafkaIngestEnabled    bool
	HTTPFindingsIngestEnabled bool
}

func Load() Config {
	brokers := splitCSV(getEnv("APP_KAFKA_BROKERS", ""))
	kafkaOn := len(brokers) > 0


	httpFindings := getBoolEnv("APP_HTTP_FINDINGS_INGEST", !kafkaOn)
	return Config{
		HTTPPort:                  getEnv("APP_HTTP_PORT", "8082"),
		PostgresDSN:               getEnv("APP_POSTGRES_DSN", "postgres://asoc:asoc@localhost:5432/asoc?sslmode=disable"),
		KafkaBrokers:              brokers,
		KafkaTopicIngest:          getEnv("APP_KAFKA_TOPIC_FINDINGS_INGEST", "asoc.findings.ingest"),
		KafkaTopicResult:          getEnv("APP_KAFKA_TOPIC_FINDINGS_RESULT", "asoc.findings.ingest.result"),
		KafkaIngestPartitions:     getIntEnv("APP_KAFKA_INGEST_PARTITIONS", 6),
		KafkaResultPartitions:     getIntEnv("APP_KAFKA_RESULT_PARTITIONS", 1),
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

func getIntEnv(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fallback
	}
	return n
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
