package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPPort      string
	PostgresDSN   string
	JWTSecret     string
	JWTTTL        string
	BootstrapLogin    string
	BootstrapPassword string
	SMTPHost      string
	SMTPPort      string
	SMTPLogin     string
	SMTPPassword  string
	SMTPFrom      string
}

func Load() Config {
	return Config{
		HTTPPort:          getEnv("AUTH_HTTP_PORT", "8091"),
		PostgresDSN:       os.Getenv("AUTH_POSTGRES_DSN"),
		JWTSecret:         os.Getenv("AUTH_JWT_SECRET"),
		JWTTTL:            getEnv("AUTH_JWT_TTL", "168h"),
		BootstrapLogin:    os.Getenv("AUTH_BOOTSTRAP_ADMIN_LOGIN"),
		BootstrapPassword: os.Getenv("AUTH_BOOTSTRAP_ADMIN_PASSWORD"),
		SMTPHost:          getEnv("AUTH_SMTP_HOST", "smtp.mail.selcloud.ru"),
		SMTPPort:          getEnv("AUTH_SMTP_PORT", "1126"),
		SMTPLogin:         getEnv("AUTH_SMTP_LOGIN", "8512"),
		SMTPPassword:      os.Getenv("AUTH_SMTP_PASSWORD"),
		SMTPFrom:          getEnv("AUTH_SMTP_FROM", "no-reply@atomic-asoc.ru"),
	}
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
