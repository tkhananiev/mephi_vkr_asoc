package app

import (
	"log"
	"net/http"

	"mephi_vkr_aspm/services/jira-mock/internal/httpapi"
)

type App struct {
	server *http.Server
}

func New() *App {
	mux := http.NewServeMux()
	handler := httpapi.New()
	handler.Register(mux)

	return &App{
		server: &http.Server{
			Addr:    ":8090",
			Handler: mux,
		},
	}
}

func (a *App) Run() error {
	log.Printf("jira-mock listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}
