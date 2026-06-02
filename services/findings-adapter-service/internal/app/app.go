package app

import (
	"log"
	"net/http"

	"mephi_vkr_asoc/services/findings-adapter-service/internal/config"
	"mephi_vkr_asoc/services/findings-adapter-service/internal/httpapi"
)

type App struct {
	server *http.Server
}

func New(cfg config.Config) *App {
	mux := http.NewServeMux()
	httpapi.New().Register(mux)
	return &App{
		server: &http.Server{
			Addr:    ":" + cfg.HTTPPort,
			Handler: mux,
		},
	}
}

func (a *App) Run() error {
	log.Printf("findings-adapter-service listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}
