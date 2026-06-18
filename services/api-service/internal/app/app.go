package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"mephi_vkr_asoc/services/api-service/internal/config"
	"mephi_vkr_asoc/services/api-service/internal/httpapi"
	"mephi_vkr_asoc/services/api-service/internal/integrationstore"
	pkgkafka "mephi_vkr_asoc/services/api-service/internal/kafka"
	"mephi_vkr_asoc/services/api-service/internal/ops"
	"mephi_vkr_asoc/services/api-service/internal/products"
	"mephi_vkr_asoc/services/api-service/internal/service"
	"mephi_vkr_asoc/services/api-service/internal/swaggerui"
)

type App struct {
	server *http.Server
}

func New(cfg config.Config) (*App, error) {
	if cfg.RequireKafkaForFindings && len(cfg.KafkaBrokers) == 0 {
		return nil, fmt.Errorf("APP_REQUIRE_KAFKA_FOR_FINDINGS_INGEST=true but APP_KAFKA_BROKERS is empty")
	}

	jwtBytes := []byte(strings.TrimSpace(cfg.JWTSecret))

	var kafkaBridge *pkgkafka.IngestBridge
	if len(cfg.KafkaBrokers) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		terr := pkgkafka.EnsureTopics(ctx, cfg.KafkaBrokers, cfg.KafkaTopicIngest, cfg.KafkaIngestPartitions, cfg.KafkaTopicResult, cfg.KafkaResultPartitions)
		cancel()
		if terr != nil {
			return nil, fmt.Errorf("kafka ensure topics: %w", terr)
		}
		kafkaBridge = pkgkafka.NewIngestBridge(cfg.KafkaBrokers, cfg.KafkaTopicIngest, cfg.KafkaTopicResult)
		log.Printf("api-service: findings ingest via Kafka (%s:%d partitions, async 202; %s:%d for scan wait)",
			cfg.KafkaTopicIngest, cfg.KafkaIngestPartitions, cfg.KafkaTopicResult, cfg.KafkaResultPartitions)
	} else {
		log.Printf("api-service: findings ingest via HTTP POST to processing-service (для стенда задайте APP_KAFKA_BROKERS и APP_REQUIRE_KAFKA_FOR_FINDINGS_INGEST=true при необходимости)")
	}

	intStore, err := integrationstore.New(cfg.IntegrationOverlayPath)
	if err != nil {
		return nil, fmt.Errorf("integrations store: %w", err)
	}

	orchestrator := service.New(
		cfg.ProcessingServiceURL,
		cfg.JiraServiceURL,
		cfg.SemgrepServiceURL,
		cfg.GitleaksServiceURL,
		cfg.ScaServiceURL,
		cfg.DastServiceURL,
		cfg.FindingsAdapterURL,
		kafkaBridge,
		func(scannerID string) (invokeURL string, scannerName string, runnerCommand string, ok bool) {
			return intStore.LookupDynamicInvoke(scannerID)
		},
	)

	if len(jwtBytes) >= 32 {
		log.Printf("api-service: JWT gate enabled (APP_JWT_SECRET), issuers asoc-auth / asoc-api (legacy)")
	} else {
		jwtBytes = nil
		log.Printf("api-service: user JWT gate disabled (set APP_JWT_SECRET >= 32 bytes for Bearer tokens from auth-service)")
	}

	var dockerOps *ops.Runner
	if cfg.DockerOpsEnabled {
		dockerOps = ops.NewRunner(cfg.DockerCLIPath)
		log.Printf("api-service: docker ops enabled (APP_DOCKER_OPS_ENABLED), cli=%s", cfg.DockerCLIPath)
	}

	var k8sOps *ops.K8sRunner
	if cfg.K8SOpsEnabled {
		cli, err := ops.NewKubernetesClientset()
		if err != nil {
			log.Printf("api-service: APP_K8S_OPS_ENABLED but kubernetes client: %v", err)
		} else {
			k8sOps = ops.NewK8sRunner(cli, cfg.K8SNamespace, cfg.K8SPodContainer)
			log.Printf("api-service: k8s pod ops enabled (namespace=%s container=%s)", cfg.K8SNamespace, cfg.K8SPodContainer)
		}
	}

	var podOps ops.Backend
	if k8sOps != nil {
		podOps = k8sOps
	} else if dockerOps != nil {
		podOps = dockerOps
	}

	var productStore *products.Store
	if dsn := cfg.PostgresDSN; dsn != "" {
		pctx, pcancel := context.WithTimeout(context.Background(), 20*time.Second)
		pool, perr := pgxpool.New(pctx, dsn)
		pcancel()
		if perr != nil {
			return nil, fmt.Errorf("postgres pool: %w", perr)
		}
		productStore = products.NewStore(pool)
		log.Printf("api-service: console products via PostgreSQL")
	} else {
		log.Printf("api-service: APP_POSTGRES_DSN empty — /api/v1/console/products disabled")
	}

	mux := http.NewServeMux()
	h := httpapi.New(
		orchestrator,
		cfg.DefaultScanTargetPath,
		cfg.DefaultSemgrepConfig,
		jwtBytes,
		podOps,
		intStore,
		cfg.ReferenceServiceURL,
		cfg.ProcessingServiceURL,
		productStore,
	)
	h.Register(mux)
	swaggerui.Register(mux)
	mux.Handle("/metrics", promhttp.Handler())

	var wrapJWT []byte
	if len(jwtBytes) >= 32 {
		wrapJWT = jwtBytes
	}
	root := httpapi.WithHTTPMetrics(httpapi.WithAPIKeyOrUserJWT(cfg.AuthAPIKey, wrapJWT, mux))

	return &App{
		server: &http.Server{
			Addr:    ":" + cfg.HTTPPort,
			Handler: root,
		},
	}, nil
}

func (a *App) Run() error {
	log.Printf("api-service listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}
