package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort    string
	PostgresDSN string
	BaseURL     string
	ProjectKey  string
}

func Load() Config {
	return Config{
		HTTPPort:    getEnv("APP_HTTP_PORT", "8083"),
		PostgresDSN: getEnv("APP_POSTGRES_DSN", "postgres://asoc:asoc@localhost:5432/asoc?sslmode=disable"),
		BaseURL:     getEnv("APP_JIRA_BASE_URL", "https://example.atlassian.net"),
		ProjectKey:  getEnv("APP_JIRA_PROJECT_KEY", "ASOC"),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
