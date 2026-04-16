package app

import (
	"log"
	"net/http"

	"mephi_vkr_aspm/services/semgrep-service/internal/config"
	"mephi_vkr_aspm/services/semgrep-service/internal/httpapi"
	"mephi_vkr_aspm/services/semgrep-service/internal/runner"
)

type App struct {
	server *http.Server
}

func New(cfg config.Config) *App {
	r := runner.New(cfg.SemgrepBinary, cfg.SemgrepConfig)
	h := httpapi.New(r)

	mux := http.NewServeMux()
	h.Register(mux)

	return &App{
		server: &http.Server{
			Addr:    ":" + cfg.HTTPPort,
			Handler: mux,
		},
	}
}

func (a *App) Run() error {
	log.Printf("semgrep-service listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}
