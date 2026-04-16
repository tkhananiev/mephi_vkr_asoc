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
}

func Load() Config {
	return Config{
		HTTPPort:             getEnv("APP_HTTP_PORT", "8080"),
		ProcessingServiceURL: getEnv("APP_PROCESSING_SERVICE_URL", "http://localhost:8082"),
		JiraServiceURL:       getEnv("APP_JIRA_SERVICE_URL", "http://localhost:8083"),
		SemgrepServiceURL:    getEnv("APP_SEMGREP_SERVICE_URL", "http://localhost:8085"),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
