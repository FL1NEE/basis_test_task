package main

import (
	"context"
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
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := repository.Connect(cfg.MySQLDSN)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := repository.Migrate(db); err != nil {
		return err
	}

	redisClient := cache.NewClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer redisClient.Close()

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cache.Ping(pingCtx, redisClient); err != nil {
		return err
	}

	userRepo := repository.NewUserRepo(db)
	teamRepo := repository.NewTeamRepo(db)
	taskRepo := repository.NewTaskRepo(db)
	historyRepo := repository.NewHistoryRepo(db)
	commentRepo := repository.NewCommentRepo(db)
	statsRepo := repository.NewStatsRepo(db)

	tokenIssuer := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTTTL)
	taskCache := cache.NewTaskListCache(redisClient, cfg.TaskCacheTTL)

	authSvc := service.NewAuthService(userRepo, tokenIssuer)
	teamSvc := service.NewTeamService(db, teamRepo, userRepo)
	taskSvc := service.NewTaskService(db, taskRepo, historyRepo, teamSvc, taskCache)
	commentSvc := service.NewCommentService(commentRepo, taskRepo, teamSvc)
	historySvc := service.NewHistoryService(historyRepo, taskRepo, teamSvc)
	statsSvc := service.NewStatsService(statsRepo, teamSvc)

	router := httpserver.NewRouter(httpserver.Services{
		Auth:     authSvc,
		Teams:    teamSvc,
		Tasks:    taskSvc,
		Comments: commentSvc,
		History:  historySvc,
		Stats:    statsSvc,
	}, tokenIssuer)

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("http server starting", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
