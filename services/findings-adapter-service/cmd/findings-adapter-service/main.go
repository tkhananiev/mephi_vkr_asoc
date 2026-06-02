package main

import (
	"log"

	"mephi_vkr_asoc/services/findings-adapter-service/internal/app"
	"mephi_vkr_asoc/services/findings-adapter-service/internal/config"
)

func main() {
	cfg := config.Load()
	if err := app.New(cfg).Run(); err != nil {
		log.Fatalf("findings-adapter-service stopped: %v", err)
	}
}
