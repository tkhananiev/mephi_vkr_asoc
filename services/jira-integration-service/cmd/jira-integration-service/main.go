package main

import (
	"context"
	"log"

	"mephi_vkr_aspm/services/jira-integration-service/internal/app"
	"mephi_vkr_aspm/services/jira-integration-service/internal/config"
)

func main() {
	cfg := config.Load()

	application, err := app.New(context.Background(), cfg)
	if err != nil {
		log.Fatalf("failed to initialize jira-integration-service: %v", err)
	}
	defer application.Close()

	if err := application.Run(); err != nil {
		log.Fatalf("jira-integration-service stopped: %v", err)
	}
}
