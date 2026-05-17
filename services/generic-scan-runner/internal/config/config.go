package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPPort    int
	ExecTimeout int // seconds
}

func Load() Config {
	return Config{
		HTTPPort:    intFromEnv("APP_HTTP_PORT", 8087),
		ExecTimeout: intFromEnv("APP_EXEC_TIMEOUT_SEC", 900),
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
