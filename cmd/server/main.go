package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"task-tracker/internal/config"
	"task-tracker/internal/database"
	httphandler "task-tracker/internal/handlers/http"
	postgresrepo "task-tracker/internal/repository/postgres"
)

func main() {
	logger := newLogger()
	if err := run(logger); err != nil {
		logger.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func newLogger() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			switch attr.Key {
			case slog.TimeKey:
				attr.Key = "timestamp"
			case slog.LevelKey:
				attr.Key = "level"
				attr.Value = slog.StringValue(strings.ToLower(attr.Value.String()))
			case slog.MessageKey:
				attr.Key = "message"
			}
			return attr
		},
	})

	return slog.New(handler).With("service.name", "task-tracker")
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.MigrateOnStart {
		logger.Info("applying database migrations on startup")
		if err := database.MigrateUp(cfg.DatabaseURL); err != nil {
			return fmt.Errorf("apply startup migrations: %w", err)
		}
	}

	connectCtx, cancelConnect := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelConnect()
	pool, err := database.NewPool(connectCtx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return err
	}
	defer pool.Close()

	repo := postgresrepo.New(pool)
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           httphandler.New(repo, pool, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownSignal, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server started", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-shutdownSignal.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	logger.Info("http server stopped")
	return nil
}
