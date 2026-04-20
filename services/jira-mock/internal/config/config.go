package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort         string
	PublicIssueBase  string // URL без завершающего /; в ответе issue: {base}/browse/{KEY}
}

func Load() Config {
	return Config{
		HTTPPort:        getEnv("APP_HTTP_PORT", "8090"),
		PublicIssueBase: strings.TrimRight(getEnv("APP_JIRA_PUBLIC_BASE_URL", "http://localhost:8090"), "/"),
	}
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
