package app

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"mephi_vkr_asoc/services/reference-data-service/internal/config"
	"mephi_vkr_asoc/services/reference-data-service/internal/httpapi"
	"mephi_vkr_asoc/services/reference-data-service/internal/kafka"
	"mephi_vkr_asoc/services/reference-data-service/internal/scheduler"
	"mephi_vkr_asoc/services/reference-data-service/internal/service"
	"mephi_vkr_asoc/services/reference-data-service/internal/source/bdu"
	"mephi_vkr_asoc/services/reference-data-service/internal/source/nvd"
	"mephi_vkr_asoc/services/reference-data-service/internal/storage/postgres"
)

type App struct {
	server               *http.Server
	pool                 *pgxpool.Pool
	syncService          *service.SyncService
	syncSchedulerEnabled bool
	syncInterval         time.Duration
	syncInitialDelay     time.Duration
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	pool, err := connectPostgresWithRetry(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}

	repo := postgres.New(pool)
	publisher := kafka.NewNoopPublisher()
	bduClient, err := bdu.New(cfg.BDUFeedURL, cfg.BDUInsecure, cfg.BDURootCAFile)
	if err != nil {
		return nil, err
	}

	var bduBulk *bdu.BulkImporter
	zipViaLocal := strings.TrimSpace(cfg.BDUVulxmlZIPPath) != ""
	xlsxViaLocal := strings.TrimSpace(cfg.BDUVullistXLSXPath) != ""
	zipViaURL := strings.TrimSpace(cfg.BDUVulxmlZIPURL) != ""
	xlsxViaURL := strings.TrimSpace(cfg.BDUVullistXLSXURL) != ""
	zipOK := zipViaLocal || zipViaURL
	xlsxOK := xlsxViaLocal || xlsxViaURL

	needsBulkHTTP := zipViaURL || xlsxViaURL

	if cfg.BDUBulkEnabled && zipOK && xlsxOK {
		var bulkHTTP *http.Client
		if needsBulkHTTP {
			var errHTTP error
			bulkHTTP, errHTTP = bdu.NewHTTPClient(cfg.BDUInsecure, cfg.BDURootCAFile, 0)
			if errHTTP != nil {
				return nil, errHTTP
			}
		}
		bduBulk = bdu.NewBulkImporter(
			bulkHTTP,
			cfg.BDUVulxmlZIPURL,
			cfg.BDUVullistXLSXURL,
			cfg.BDUBulkBatchSize,
			cfg.BDUVulxmlZIPPath,
			cfg.BDUVullistXLSXPath,
		)
	}

	syncService := service.NewSyncService(
		repo,
		publisher,
		bduClient,
		nvd.New(cfg.NVDAPIBaseURL, cfg.NVDAPIKey, cfg.NVDPageSize, cfg.NVDMaxPages, cfg.NVDHTTPRequestTimeout),
		bduBulk,
	)

	mux := http.NewServeMux()
	handler := httpapi.New(syncService)
	handler.Register(mux)

	server := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: mux,
	}

	return &App{
		server:               server,
		pool:                 pool,
		syncService:          syncService,
		syncSchedulerEnabled: cfg.SyncSchedulerEnabled,
		syncInterval:         cfg.SyncInterval,
		syncInitialDelay:     cfg.SyncInitialDelay,
	}, nil
}

func (a *App) Run() error {
	if a.syncSchedulerEnabled && a.syncInterval > 0 && a.syncService != nil {
		scheduler.Start(context.Background(), a.syncService, a.syncInterval, a.syncInitialDelay)
	}
	log.Printf("reference-data-service listening on %s", a.server.Addr)
	return a.server.ListenAndServe()
}

func (a *App) Close() {
	if a.pool != nil {
		a.pool.Close()
	}
}
