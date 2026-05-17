package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort              string
	GitleaksBinary        string
	DefaultScanTargetPath string
	GitWorkspaceRoot      string
}

func Load() Config {
	return Config{
		HTTPPort:              getEnv("APP_HTTP_PORT", "8086"),
		GitleaksBinary:        getEnv("APP_GITLEAKS_BINARY", "gitleaks"),
		DefaultScanTargetPath: getEnv("APP_DEFAULT_SCAN_TARGET_PATH", "/app/demo/scan-targets/WebGoat/"),
		GitWorkspaceRoot:      getEnv("APP_GITLEAKS_GIT_WORK_ROOT", "/tmp/asoc-gitleaks-git-work"),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
