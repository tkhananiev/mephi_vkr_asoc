package app

import (
	"log"
	"net/http"

	"mephi_vkr_asoc/services/zap-dast-service/internal/config"
	"mephi_vkr_asoc/services/zap-dast-service/internal/httpapi"
	"mephi_vkr_asoc/services/zap-dast-service/internal/runner"
)

type App struct {
	server *http.Server
}

func New(cfg config.Config) *App {
	r := runner.New(cfg.ZapHome, cfg.ScanTimeoutMin, cfg.UseStub)
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

func (a *App) Run(cfg config.Config) error {
	if cfg.UseStub {
		log.Printf("zap-dast-service listening on %s (HTTP probe stub; set APP_ZAP_USE_STUB=false for OWASP ZAP baseline)", a.server.Addr)
	} else {
		log.Printf("zap-dast-service listening on %s (OWASP ZAP baseline, home=%s, timeout=%dm)", a.server.Addr, cfg.ZapHome, cfg.ScanTimeoutMin)
	}
	return a.server.ListenAndServe()
}
