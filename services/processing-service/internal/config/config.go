package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort    string
	PostgresDSN string
}

func Load() Config {
	return Config{
		HTTPPort:    getEnv("APP_HTTP_PORT", "8082"),
		PostgresDSN: getEnv("APP_POSTGRES_DSN", "postgres://aspm:aspm@localhost:5432/aspm?sslmode=disable"),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
