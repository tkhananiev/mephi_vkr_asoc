package main

import (
	"log"

	"mephi_vkr_asoc/services/trivy-sca-service/internal/app"
	"mephi_vkr_asoc/services/trivy-sca-service/internal/config"
)

func main() {
	cfg := config.Load()
	application := app.New(cfg)
	if err := application.Run(); err != nil {
		log.Fatalf("trivy-sca-service stopped: %v", err)
	}
}
