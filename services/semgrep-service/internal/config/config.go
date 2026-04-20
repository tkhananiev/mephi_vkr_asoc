package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort              string
	SemgrepBinary         string
	SemgrepConfig         string
	DefaultScanTargetPath string // если в POST не передан target_path
}

func Load() Config {
	return Config{
		HTTPPort:              getEnv("APP_HTTP_PORT", "8085"),
		SemgrepBinary:         getEnv("APP_SEMGREP_BINARY", "semgrep"),
		SemgrepConfig:         getEnv("APP_SEMGREP_CONFIG", "/app/demo/semgrep-rules.yml"),
		DefaultScanTargetPath: getEnv("APP_DEFAULT_SCAN_TARGET_PATH", "/app/demo/vulnerable-app"),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
