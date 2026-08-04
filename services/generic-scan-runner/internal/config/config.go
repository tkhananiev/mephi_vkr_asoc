package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPPort         int
	ExecTimeout      int // seconds
	AllowedScanRoots []string
}

func Load() Config {
	return Config{
		HTTPPort:         intFromEnv("APP_HTTP_PORT", 8087),
		ExecTimeout:      intFromEnv("APP_EXEC_TIMEOUT_SEC", 900),
		AllowedScanRoots: rootsFromEnv("APP_ALLOWED_SCAN_ROOTS", "/tmp,/app/demo/scan-targets"),
	}
}

func intFromEnv(k string, def int) int {
	s := os.Getenv(k)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func rootsFromEnv(k string, def string) []string {
	raw := strings.TrimSpace(os.Getenv(k))
	if raw == "" {
		raw = def
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
