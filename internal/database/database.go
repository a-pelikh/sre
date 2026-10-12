package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func NewPool(ctx context.Context, databaseURL string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	cfg.MaxConns = maxConns

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}

func NewMigrator(databaseURL string) (*migrate.Migrate, error) {
	source, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return nil, fmt.Errorf("create embedded migration source: %w", err)
	}

	migrationURL, err := pgxMigrationURL(databaseURL)
	if err != nil {
		return nil, err
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, migrationURL)
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}

	return m, nil
}

func pgxMigrationURL(databaseURL string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("parse migration database URL: %w", err)
	}

	switch parsed.Scheme {
	case "postgres", "postgresql", "pgx5":
		parsed.Scheme = "pgx5"
	default:
		return "", fmt.Errorf("unsupported database URL scheme %q", parsed.Scheme)
	}

	return parsed.String(), nil
}

func MigrateUp(databaseURL string) error {
	migrator, err := NewMigrator(databaseURL)
	if err != nil {
		return err
	}

	err = migrator.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		err = nil
	}

	sourceErr, databaseErr := migrator.Close()
	return errors.Join(err, sourceErr, databaseErr)
}
