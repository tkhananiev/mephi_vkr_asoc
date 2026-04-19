package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPPort              string
	PostgresDSN           string
	KafkaBrokers          []string
	BDUFeedURL            string
	BDUInsecure           bool
	BDURootCAFile         string // PEM с доп. корневым УЦ (напр. Минцифры); если задан — проверка TLS без InsecureSkipVerify
	NVDAPIBaseURL         string
	NVDAPIKey             string
	NVDPageSize           int
	NVDMaxPages           int
	SyncInterval          time.Duration
	SyncSchedulerEnabled  bool
	SyncInitialDelay      time.Duration
}

func Load() Config {
	return Config{
		HTTPPort:             getEnv("APP_HTTP_PORT", "8081"),
		PostgresDSN:          getEnv("APP_POSTGRES_DSN", "postgres://aspm:aspm@localhost:5432/aspm?sslmode=disable"),
		KafkaBrokers:         splitCSV(getEnv("APP_KAFKA_BROKERS", "localhost:9092")),
		BDUFeedURL:           getEnv("APP_BDU_FEED_URL", "https://bdu.fstec.ru/feed"),
		BDUInsecure:          getBool("APP_BDU_INSECURE_SKIP_VERIFY", true),
		BDURootCAFile:        getEnv("APP_BDU_ROOT_CA_FILE", ""),
		NVDAPIBaseURL:        getEnv("APP_NVD_API_BASE_URL", "https://services.nvd.nist.gov/rest/json/cves/2.0"),
		NVDAPIKey:            getEnv("APP_NVD_API_KEY", ""),
		NVDPageSize:          getInt("APP_NVD_PAGE_SIZE", 2000),
		NVDMaxPages:          getInt("APP_NVD_MAX_PAGES", 0),
		SyncInterval:         getDuration("APP_SYNC_INTERVAL", 24*time.Hour),
		SyncSchedulerEnabled: getBool("APP_SYNC_SCHEDULER_ENABLED", true),
		SyncInitialDelay:     getDuration("APP_SYNC_INITIAL_DELAY", time.Minute),
	}
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}

func splitCSV(value string) []string {
	chunks := strings.Split(value, ",")
	result := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if trimmed := strings.TrimSpace(chunk); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func getInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func getBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes"
}
