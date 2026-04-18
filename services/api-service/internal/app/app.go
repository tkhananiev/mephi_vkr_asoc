package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"mephi_vkr_aspm/services/api-service/internal/config"
	"mephi_vkr_aspm/services/api-service/internal/httpapi"
	pkgkafka "mephi_vkr_aspm/services/api-service/internal/kafka"
	"mephi_vkr_aspm/services/api-service/internal/service"
)

type App struct {
	server *http.Server
}

func New(cfg config.Config) (*App, error) {
	var kafkaBridge *pkgkafka.IngestBridge
	if len(cfg.KafkaBrokers) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := pkgkafka.EnsureTopics(ctx, cfg.KafkaBrokers, cfg.KafkaTopicIngest, cfg.KafkaTopicResult)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("kafka ensure topics: %w", err)
		}
		kafkaBridge = pkgkafka.NewIngestBridge(cfg.KafkaBrokers, cfg.KafkaTopicIngest, cfg.KafkaTopicResult)
		log.Printf("api-service: findings ingest via Kafka (%s -> %s)", cfg.KafkaTopicIngest, cfg.KafkaTopicResult)
	} else {
		log.Printf("api-service: findings ingest via HTTP (APP_KAFKA_BROKERS empty)")
	}

	orchestrator := service.New(
		cfg.ProcessingServiceURL,
		cfg.JiraServiceURL,
		cfg.SemgrepServiceURL,
		kafkaBridge,
	)

	mux := http.NewServeMux()
	handler := httpapi.New(orchestrator)
	handler.Register(mux)

	return &App{
		server: &http.Server{
			Addr:    ":" + cfg.HTTPPort,
			Handler: mux,
		},
	}, nil
}

func (a *App) Run() error {
	log.Printf("api-service listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}
