package app

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"mephi_vkr_aspm/services/jira-integration-service/internal/config"
	"mephi_vkr_aspm/services/jira-integration-service/internal/httpapi"
	"mephi_vkr_aspm/services/jira-integration-service/internal/service"
	"mephi_vkr_aspm/services/jira-integration-service/internal/storage/postgres"
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
	ticketService := service.New(repo, cfg.BaseURL, cfg.ProjectKey)

	mux := http.NewServeMux()
	handler := httpapi.New(ticketService)
	handler.Register(mux)

	return &App{
		server: &http.Server{
			Addr:    ":" + cfg.HTTPPort,
			Handler: mux,
		},
		pool: pool,
	}, nil
}

func (a *App) Run() error {
	log.Printf("jira-integration-service listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}

func (a *App) Close() {
	if a.pool != nil {
		a.pool.Close()
	}
}
