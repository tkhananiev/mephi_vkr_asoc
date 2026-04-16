package main

import (
	"log"

	"mephi_vkr_aspm/services/jira-mock/internal/app"
)

func main() {
	application := app.New()
	if err := application.Run(); err != nil {
		log.Fatalf("jira-mock stopped: %v", err)
	}
}
