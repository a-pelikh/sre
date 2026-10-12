package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"

	"task-tracker/internal/config"
	"task-tracker/internal/database"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger, os.Args[1:]); err != nil {
		logger.Error("migration command failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	migrator, err := database.NewMigrator(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() {
		sourceErr, databaseErr := migrator.Close()
		if sourceErr != nil {
			logger.Warn("close migration source", "error", sourceErr)
		}
		if databaseErr != nil {
			logger.Warn("close migration database", "error", databaseErr)
		}
	}()

	command := "up"
	if len(args) > 0 {
		command = args[0]
	}
	switch command {
	case "up":
		err = migrator.Up()
	case "down":
		steps := 1
		if len(args) > 1 {
			steps, err = strconv.Atoi(args[1])
			if err != nil || steps < 1 {
				return errors.New("down steps must be a positive integer")
			}
		}
		err = migrator.Steps(-steps)
	case "version":
		version, dirty, versionErr := migrator.Version()
		if errors.Is(versionErr, migrate.ErrNilVersion) {
			logger.Info("migration version", "version", 0, "dirty", false)
			return nil
		}
		if versionErr != nil {
			return versionErr
		}
		logger.Info("migration version", "version", version, "dirty", dirty)
		return nil
	default:
		return fmt.Errorf("unknown command %q; use up, down, or version", command)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		logger.Info("no migration changes")
		return nil
	}
	if err != nil {
		return err
	}
	logger.Info("migration completed", "command", command)
	return nil
}
