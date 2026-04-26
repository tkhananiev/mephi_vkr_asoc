package app

import (
	"fmt"
	"log"
	"net/http"

	"mephi_vkr_asoc/services/jira-mock/internal/config"
	"mephi_vkr_asoc/services/jira-mock/internal/httpapi"
)

type App struct {
	server *http.Server
}

func New(cfg config.Config) *App {
	mux := http.NewServeMux()
	handler := httpapi.New(cfg.PublicIssueBase)
	handler.Register(mux)

	return &App{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.HTTPPort),
			Handler: mux,
		},
	}
}

func (a *App) Run() error {
	log.Printf("jira-mock listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}
