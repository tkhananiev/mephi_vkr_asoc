package scheduler

import (
	"context"
	"log"
	"time"

	"mephi_vkr_aspm/services/reference-data-service/internal/service"
)

// Start запускает цикл: после initialDelay — полная синхронизация БДУ и NVD, затем каждые interval.
func Start(ctx context.Context, svc *service.SyncService, interval, initialDelay time.Duration) {
	if interval <= 0 || svc == nil {
		return
	}
	go run(ctx, svc, interval, initialDelay)
}

func run(ctx context.Context, svc *service.SyncService, interval, initialDelay time.Duration) {
	log.Printf("sync scheduler: interval=%v initial_delay=%v", interval, initialDelay)
	if initialDelay > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(initialDelay):
		}
	}
	for {
		runOnce(svc)
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func runOnce(svc *service.SyncService) {
	// Полная выгрузка NVD может занимать много часов (лимиты API); не обрывать через 6h.
	reqCtx, cancel := context.WithTimeout(context.Background(), 168*time.Hour)
	defer cancel()

	log.Printf("scheduled sync: BDU start")
	if _, err := svc.SyncBDU(reqCtx); err != nil {
		log.Printf("scheduled sync BDU failed: %v", err)
	}
	log.Printf("scheduled sync: NVD start")
	if _, err := svc.SyncNVD(reqCtx); err != nil {
		log.Printf("scheduled sync NVD failed: %v", err)
	}
	log.Printf("scheduled sync: finished")
}
