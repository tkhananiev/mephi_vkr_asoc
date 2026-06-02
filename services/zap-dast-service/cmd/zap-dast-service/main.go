package main

import (
	"log"

	"mephi_vkr_asoc/services/zap-dast-service/internal/app"
	"mephi_vkr_asoc/services/zap-dast-service/internal/config"
)

func main() {
	cfg := config.Load()
	application := app.New(cfg)
	if err := application.Run(cfg); err != nil {
		log.Fatalf("zap-dast-service stopped: %v", err)
	}
}
