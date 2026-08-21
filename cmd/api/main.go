package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FL1NEE/basis_test_task/internal/auth"
	"github.com/FL1NEE/basis_test_task/internal/cache"
	"github.com/FL1NEE/basis_test_task/internal/config"
	"github.com/FL1NEE/basis_test_task/internal/httpserver"
	"github.com/FL1NEE/basis_test_task/internal/repository"
	"github.com/FL1NEE/basis_test_task/internal/service"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := repository.Connect(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()

	if err := repository.Migrate(db); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	slog.Info("migrations applied")

	redisClient := cache.NewClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer redisClient.Close()
	if err := cache.Ping(ctx, redisClient); err != nil {
		log.Fatalf("ping redis: %v", err)
	}

	instrumentedDB := repository.Instrument(db)
	userRepo := repository.NewUserRepo(instrumentedDB)
	teamRepo := repository.NewTeamRepo(instrumentedDB)
	taskRepo := repository.NewTaskRepo(instrumentedDB)
	historyRepo := repository.NewHistoryRepo(instrumentedDB)
	commentRepo := repository.NewCommentRepo(instrumentedDB)
	statsRepo := repository.NewStatsRepo(instrumentedDB)

	tokens := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTTTL)
	taskCache := cache.NewCachedTaskList(cache.NewTaskListCache(redisClient, cfg.TaskCacheTTL))

	authSvc := service.NewAuthService(userRepo, tokens)
	teamSvc := service.NewTeamService(db, teamRepo, userRepo)
	taskSvc := service.NewTaskService(db, taskRepo, historyRepo, teamSvc, taskCache)
	commentSvc := service.NewCommentService(commentRepo, taskSvc)
	historySvc := service.NewHistoryService(historyRepo, taskSvc)
	statsSvc := service.NewStatsService(statsRepo, teamSvc)

	router := httpserver.NewRouter(httpserver.Services{
		Auth:     authSvc,
		Teams:    teamSvc,
		Tasks:    taskSvc,
		Comments: commentSvc,
		History:  historySvc,
		Stats:    statsSvc,
	}, tokens)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	slog.Info("server stopped")
}
