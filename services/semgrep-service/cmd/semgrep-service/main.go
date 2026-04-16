package main

import (
	"log"

	"mephi_vkr_aspm/services/semgrep-service/internal/app"
	"mephi_vkr_aspm/services/semgrep-service/internal/config"
)

func main() {
	cfg := config.Load()
	application := app.New(cfg)
	if err := application.Run(); err != nil {
		log.Fatalf("semgrep-service stopped: %v", err)
	}
}
