package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort        string
	SemgrepBinary   string
	SemgrepConfig   string
}

func Load() Config {
	return Config{
		HTTPPort:      getEnv("APP_HTTP_PORT", "8085"),
		SemgrepBinary: getEnv("APP_SEMGREP_BINARY", "semgrep"),
		SemgrepConfig: getEnv("APP_SEMGREP_CONFIG", "/app/demo/semgrep-rules.yml"),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
