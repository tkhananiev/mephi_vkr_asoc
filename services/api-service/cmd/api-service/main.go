package main

import (
	"log"

	"mephi_vkr_aspm/services/api-service/internal/app"
	"mephi_vkr_aspm/services/api-service/internal/config"
)

func main() {
	cfg := config.Load()
	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize api-service: %v", err)
	}
	if err := application.Run(); err != nil {
		log.Fatalf("api-service stopped: %v", err)
	}
}
