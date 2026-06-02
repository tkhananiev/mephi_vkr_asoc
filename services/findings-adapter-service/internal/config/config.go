package config

import "os"

type Config struct {
	HTTPPort string
}

func Load() Config {
	return Config{
		HTTPPort: getEnv("APP_HTTP_PORT", "8090"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
