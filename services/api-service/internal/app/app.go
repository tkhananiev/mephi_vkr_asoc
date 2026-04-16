package app

import (
	"log"
	"net/http"

	"mephi_vkr_aspm/services/api-service/internal/config"
	"mephi_vkr_aspm/services/api-service/internal/httpapi"
	"mephi_vkr_aspm/services/api-service/internal/service"
)

type App struct {
	server *http.Server
}

func New(cfg config.Config) *App {
	orchestrator := service.New(
		cfg.ProcessingServiceURL,
		cfg.JiraServiceURL,
		cfg.SemgrepServiceURL,
	)

	mux := http.NewServeMux()
	handler := httpapi.New(orchestrator)
	handler.Register(mux)

	return &App{
		server: &http.Server{
			Addr:    ":" + cfg.HTTPPort,
			Handler: mux,
		},
	}
}

func (a *App) Run() error {
	log.Printf("api-service listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}
