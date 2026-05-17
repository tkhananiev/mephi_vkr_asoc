package main

import (
	"log"

	"mephi_vkr_asoc/services/gitleaks-service/internal/app"
	"mephi_vkr_asoc/services/gitleaks-service/internal/config"
)

func main() {
	cfg := config.Load()
	application := app.New(cfg)
	if err := application.Run(); err != nil {
		log.Fatalf("gitleaks-service stopped: %v", err)
	}
}
