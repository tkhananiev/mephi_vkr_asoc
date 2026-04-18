package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"mephi_vkr_aspm/services/processing-service/internal/config"
	"mephi_vkr_aspm/services/processing-service/internal/httpapi"
	pkgkafka "mephi_vkr_aspm/services/processing-service/internal/kafka"
	"mephi_vkr_aspm/services/processing-service/internal/service"
	"mephi_vkr_aspm/services/processing-service/internal/storage/postgres"
)

type App struct {
	server *http.Server
	pool   *pgxpool.Pool
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	pool, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	repo := postgres.New(pool)
	processingService := service.New(repo)

	if cfg.KafkaIngestEnabled {
		if err := pkgkafka.EnsureTopics(ctx, cfg.KafkaBrokers, cfg.KafkaTopicIngest, cfg.KafkaTopicResult); err != nil {
			return nil, fmt.Errorf("kafka ensure topics: %w", err)
		}
		cons := pkgkafka.NewIngestConsumer(cfg.KafkaBrokers, cfg.KafkaTopicIngest, cfg.KafkaTopicResult, processingService)
		go func() {
			if err := cons.Run(context.Background()); err != nil {
				log.Printf("kafka ingest consumer exited: %v", err)
			}
		}()
		log.Printf("kafka ingest consumer running (topics %s -> %s)", cfg.KafkaTopicIngest, cfg.KafkaTopicResult)
	}

	mux := http.NewServeMux()
	handler := httpapi.New(processingService)
	handler.Register(mux)

	server := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: mux,
	}

	return &App{
		server: server,
		pool:   pool,
	}, nil
}

func (a *App) Run() error {
	log.Printf("processing-service listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}

func (a *App) Close() {
	if a.pool != nil {
		a.pool.Close()
	}
}
