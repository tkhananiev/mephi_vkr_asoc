package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort             string
	ProcessingServiceURL string
	JiraServiceURL       string
	SemgrepServiceURL    string
	KafkaBrokers         []string
	KafkaTopicIngest     string
	KafkaTopicResult     string
	// DefaultScanTargetPath — путь к каталогу исходников в контейнере semgrep-service (если в теле запроса нет target_path).
	DefaultScanTargetPath string
	// DefaultSemgrepConfig — например p/java; передаётся в semgrep-service, если в запросе нет semgrep_config.
	DefaultSemgrepConfig string
}

func Load() Config {
	return Config{
		HTTPPort:             getEnv("APP_HTTP_PORT", "8080"),
		ProcessingServiceURL: getEnv("APP_PROCESSING_SERVICE_URL", "http://localhost:8082"),
		JiraServiceURL:      getEnv("APP_JIRA_SERVICE_URL", "http://localhost:8083"),
		SemgrepServiceURL:    getEnv("APP_SEMGREP_SERVICE_URL", "http://localhost:8085"),
		KafkaBrokers:         splitCSV(getEnv("APP_KAFKA_BROKERS", "")),
		KafkaTopicIngest:     getEnv("APP_KAFKA_TOPIC_FINDINGS_INGEST", "aspm.findings.ingest"),
		KafkaTopicResult:     getEnv("APP_KAFKA_TOPIC_FINDINGS_RESULT", "aspm.findings.ingest.result"),
		DefaultScanTargetPath: getEnv("APP_DEFAULT_SCAN_TARGET_PATH", ""),
		DefaultSemgrepConfig:  getEnv("APP_DEFAULT_SEMGREP_CONFIG", ""),
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
