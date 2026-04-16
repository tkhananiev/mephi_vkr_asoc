package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"mephi_vkr_aspm/services/reference-data-service/internal/config"
	"mephi_vkr_aspm/services/reference-data-service/internal/httpapi"
	"mephi_vkr_aspm/services/reference-data-service/internal/kafka"
	"mephi_vkr_aspm/services/reference-data-service/internal/service"
	"mephi_vkr_aspm/services/reference-data-service/internal/source/bdu"
	"mephi_vkr_aspm/services/reference-data-service/internal/source/nvd"
	"mephi_vkr_aspm/services/reference-data-service/internal/storage/postgres"
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
	publisher := kafka.NewNoopPublisher()
	syncService := service.NewSyncService(
		repo,
		publisher,
		bdu.New(cfg.BDUFeedURL, cfg.BDUInsecure),
		nvd.New(cfg.NVDAPIBaseURL),
	)

	mux := http.NewServeMux()
	handler := httpapi.New(syncService)
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
	log.Printf("reference-data-service listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}

func (a *App) Close() {
	if a.pool != nil {
		a.pool.Close()
	}
}
