package main

import (
	"log"

	"mephi_vkr_asoc/services/jira-mock/internal/app"
	"mephi_vkr_asoc/services/jira-mock/internal/config"
)

func main() {
	application := app.New(config.Load())
	if err := application.Run(); err != nil {
		log.Fatalf("jira-mock stopped: %v", err)
	}
}
