package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tavily-proxy/server/internal/config"
	"tavily-proxy/server/internal/db"
	"tavily-proxy/server/internal/httpserver"
	"tavily-proxy/server/internal/jobs"
	"tavily-proxy/server/internal/services"
)

//go:embed public
var embeddedPublic embed.FS

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.FromEnv()

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("db open failed", "err", err)
		os.Exit(1)
	}

	masterKeyService := services.NewMasterKeyService(database, logger)
	if err := masterKeyService.LoadOrCreateWithDefault(context.Background(), cfg.MasterKey); err != nil {
		logger.Error("master key init failed", "err", err)
		os.Exit(1)
	}

	settingsService := services.NewSettingsService(database)
	keyService := services.NewKeyService(database, logger)
	logService := services.NewLogService(database, logger)
	statsService := services.NewStatsService(database)

	var distributedKeyService *services.DistributedKeyService
	var distributedKeyUsageService *services.DistributedKeyUsageService
	var distributedRateLimiter *services.DistributedRateLimiter
	if strings.TrimSpace(cfg.UserKeyEncryptionKey) == "" {
		logger.Info("distributed user key feature disabled: USER_KEY_ENCRYPTION_KEY not set")
	} else {
		userKeyCipher, err := services.NewTokenCipher(cfg.UserKeyEncryptionKey)
		if err != nil {
			logger.Error("user key cipher init failed", "err", err)
			os.Exit(1)
		}
		distributedKeyService = services.NewDistributedKeyService(database, logger, userKeyCipher, cfg.UserKeyRateLimitDefault)
		distributedKeyUsageService = services.NewDistributedKeyUsageService(database)
		distributedRateLimiter = services.NewDistributedRateLimiter(cfg.UserKeyRateLimitWindow)
	}

	if err := statsService.BackfillFromLogsIfEmpty(context.Background()); err != nil {
		logger.Error("stats backfill failed", "err", err)
	}

	tavilyProxy := services.NewTavilyProxy(cfg.TavilyBaseURL, cfg.UpstreamTimeout, keyService, logService, statsService, logger).
		WithSettings(settingsService)
	keyBatchCreateJob := services.NewKeyBatchCreateJobService(keyService, logger)
	quotaSyncService := services.NewQuotaSyncService(keyService, tavilyProxy, logger)
	quotaSyncJob := services.NewQuotaSyncJobService(keyService, quotaSyncService, logger)

	srv := httpserver.New(httpserver.Dependencies{
		Config:                     cfg,
		EmbeddedPublic:             embeddedPublic,
		MasterKeyService:           masterKeyService,
		DistributedKeyService:      distributedKeyService,
		DistributedKeyUsageService: distributedKeyUsageService,
		DistributedRateLimiter:     distributedRateLimiter,
		SettingsService:            settingsService,
		KeyService:                 keyService,
		KeyBatchCreateJob:          keyBatchCreateJob,
		QuotaSyncService:           quotaSyncService,
		QuotaSyncJob:               quotaSyncJob,
		LogService:                 logService,
		StatsService:               statsService,
		TavilyProxy:                tavilyProxy,
		Logger:                     logger,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	jobs.StartMonthlyReset(ctx, keyService, logger)
	jobs.StartAutoQuotaSync(ctx, settingsService, quotaSyncService, logger)
	jobs.StartLogCleanup(ctx, settingsService, logService, logger)

	go func() {
		logger.Info("server listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil {
			logger.Error("http server stopped", "err", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
