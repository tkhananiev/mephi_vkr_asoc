package app

import (
	"log"
	"net/http"

	"mephi_vkr_asoc/services/trivy-sca-service/internal/config"
	"mephi_vkr_asoc/services/trivy-sca-service/internal/httpapi"
	"mephi_vkr_asoc/services/trivy-sca-service/internal/runner"
)

type App struct {
	server *http.Server
}

func New(cfg config.Config) *App {
	r := runner.New(cfg.TrivyBinary)
	h := httpapi.New(r, cfg.DefaultScanTargetPath, cfg.GitWorkspaceRoot)

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
	log.Printf("trivy-sca-service listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}
