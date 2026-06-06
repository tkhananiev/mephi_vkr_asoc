package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort              string
	SemgrepBinary         string
	SemgrepConfig         string
	DefaultScanTargetPath string // если в POST не передан target_path и нет git

	GitWorkspaceRoot string
}

func Load() Config {
	return Config{
		HTTPPort:              getEnv("APP_HTTP_PORT", "8085"),
		SemgrepBinary:         getEnv("APP_SEMGREP_BINARY", "semgrep"),
		SemgrepConfig:         getEnv("APP_SEMGREP_CONFIG", "p/java"),
		DefaultScanTargetPath: getEnv("APP_DEFAULT_SCAN_TARGET_PATH", "/app/demo/scan-targets/WebGoat/"),
		GitWorkspaceRoot:      getEnv("APP_SEMGREP_GIT_WORK_ROOT", "/tmp/asoc-semgrep-git-work"),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
