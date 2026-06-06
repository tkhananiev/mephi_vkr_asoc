package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPPort                string
	ProcessingServiceURL    string
	JiraServiceURL          string
	SemgrepServiceURL       string
	GitleaksServiceURL      string
	ScaServiceURL           string
	DastServiceURL          string
	FindingsAdapterURL      string
	KafkaBrokers            []string
	KafkaTopicIngest        string
	KafkaTopicResult        string
	KafkaIngestPartitions   int
	KafkaResultPartitions   int
	DefaultScanTargetPath   string
	DefaultSemgrepConfig    string
	AuthAPIKey              string
	RequireKafkaForFindings bool


	IntegrationOverlayPath string


	JWTSecret string
	JWTTTL    string


	DockerOpsEnabled bool
	DockerCLIPath    string


	K8SOpsEnabled   bool
	K8SNamespace    string
	K8SPodContainer string // обычно app (см. workloads.yaml)


	PostgresDSN string
}

func Load() Config {
	return Config{
		HTTPPort:                getEnv("APP_HTTP_PORT", "8080"),
		ProcessingServiceURL:    getEnv("APP_PROCESSING_SERVICE_URL", "http://localhost:8082"),
		JiraServiceURL:          getEnv("APP_JIRA_SERVICE_URL", "http://localhost:8083"),
		SemgrepServiceURL:       getEnv("APP_SEMGREP_SERVICE_URL", "http://localhost:8085"),
		GitleaksServiceURL:      getEnv("APP_GITLEAKS_SERVICE_URL", "http://localhost:8086"),
		ScaServiceURL:           getEnv("APP_SCA_SERVICE_URL", "http://localhost:8088"),
		DastServiceURL:          getEnv("APP_DAST_SERVICE_URL", "http://localhost:8089"),
		FindingsAdapterURL:      getEnv("APP_FINDINGS_ADAPTER_URL", "http://localhost:8090"),
		KafkaBrokers:            splitCSV(getEnv("APP_KAFKA_BROKERS", "")),
		KafkaTopicIngest:        getEnv("APP_KAFKA_TOPIC_FINDINGS_INGEST", "asoc.findings.ingest"),
		KafkaTopicResult:        getEnv("APP_KAFKA_TOPIC_FINDINGS_RESULT", "asoc.findings.ingest.result"),
		KafkaIngestPartitions:   getIntEnv("APP_KAFKA_INGEST_PARTITIONS", 6),
		KafkaResultPartitions:   getIntEnv("APP_KAFKA_RESULT_PARTITIONS", 1),
		DefaultScanTargetPath:   getEnv("APP_DEFAULT_SCAN_TARGET_PATH", "/app/demo/scan-targets/WebGoat/"),
		DefaultSemgrepConfig:    getEnv("APP_DEFAULT_SEMGREP_CONFIG", "p/java"),
		AuthAPIKey:              os.Getenv("APP_AUTH_API_KEY"),
		RequireKafkaForFindings: getBoolEnv("APP_REQUIRE_KAFKA_FOR_FINDINGS_INGEST", false),

		IntegrationOverlayPath: getEnv("APP_INTEGRATIONS_OVERLAY_PATH", ""),

		JWTSecret: os.Getenv("APP_JWT_SECRET"),
		JWTTTL:    getEnv("APP_JWT_TTL", "168h"),

		DockerOpsEnabled: getBoolEnv("APP_DOCKER_OPS_ENABLED", false),
		DockerCLIPath:    getEnv("APP_DOCKER_CLI", "docker"),

		K8SOpsEnabled:   getBoolEnv("APP_K8S_OPS_ENABLED", false),
		K8SNamespace:    getEnv("APP_K8S_NAMESPACE", "asoc"),
		K8SPodContainer: getEnv("APP_K8S_POD_CONTAINER", "app"),

		PostgresDSN: strings.TrimSpace(os.Getenv("APP_POSTGRES_DSN")),
	}
}

func getBoolEnv(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on", "y":
		return true
	default:
		return false
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getIntEnv(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
