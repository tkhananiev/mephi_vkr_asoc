package main

import (
	"context"
	"log"

	"mephi_vkr_aspm/services/processing-service/internal/app"
	"mephi_vkr_aspm/services/processing-service/internal/config"
)

func main() {
	cfg := config.Load()

	application, err := app.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("failed to initialize processing-service: %v", err)
	}
	defer application.Close()

	if err := application.Run(); err != nil {
		log.Fatalf("processing-service stopped: %v", err)
	}
}
