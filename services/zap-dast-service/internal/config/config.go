package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPPort       string
	ZapHome        string
	ScanTimeoutMin int
	UseStub        bool
}

func Load() Config {
	timeoutMin := 8
	if v := strings.TrimSpace(os.Getenv("APP_ZAP_SCAN_TIMEOUT_MIN")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutMin = n
		}
	}
	return Config{
		HTTPPort:       getEnv("APP_HTTP_PORT", "8089"),
		ZapHome:        getEnv("APP_ZAP_HOME", "/zap"),
		ScanTimeoutMin: timeoutMin,
		UseStub:        envBool("APP_ZAP_USE_STUB"),
	}
}

func envBool(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
