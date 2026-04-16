package main

import (
	"context"
	"log"

	"mephi_vkr_aspm/services/reference-data-service/internal/app"
	"mephi_vkr_aspm/services/reference-data-service/internal/config"
)

func main() {
	cfg := config.Load()

	application, err := app.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}
	defer application.Close()

	if err := application.Run(); err != nil {
		log.Fatalf("service stopped: %v", err)
	}
}
