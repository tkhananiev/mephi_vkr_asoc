package main

import (
	"log"

	"mephi_vkr_asoc/services/generic-scan-runner/internal/app"
	"mephi_vkr_asoc/services/generic-scan-runner/internal/config"
)

func main() {
	cfg := config.Load()
	if err := app.New(cfg).Run(); err != nil {
		log.Fatalf("generic-scan-runner stopped: %v", err)
	}
}
