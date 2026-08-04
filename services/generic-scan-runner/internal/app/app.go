package app

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"mephi_vkr_asoc/services/generic-scan-runner/internal/config"
	"mephi_vkr_asoc/services/generic-scan-runner/internal/httpapi"
)

type App struct {
	server *http.Server
}

func New(cfg config.Config) *App {
	h := &httpapi.Handler{
		ExecTimeout:      time.Duration(cfg.ExecTimeout) * time.Second,
		AllowedScanRoots: cfg.AllowedScanRoots,
	}
	mux := http.NewServeMux()
	h.Register(mux)
	addr := fmt.Sprintf(":%d", cfg.HTTPPort)
	return &App{
		server: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: time.Duration(cfg.ExecTimeout)*time.Second + 60*time.Second,
		},
	}
}

func (a *App) Run() error {
	log.Printf("generic-scan-runner listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}
