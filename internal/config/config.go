package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL     string
	Port            int
	DBMaxConns      int32
	ShutdownTimeout time.Duration
	MigrateOnStart  bool
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		Port:            8080,
		DBMaxConns:      10,
		ShutdownTimeout: 10 * time.Second,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	var err error
	if value := os.Getenv("PORT"); value != "" {
		cfg.Port, err = strconv.Atoi(value)
		if err != nil || cfg.Port < 1 || cfg.Port > 65535 {
			return Config{}, fmt.Errorf("PORT must be a number between 1 and 65535")
		}
	}

	if value := os.Getenv("DB_MAX_CONNS"); value != "" {
		parsed, parseErr := strconv.ParseInt(value, 10, 32)
		if parseErr != nil || parsed < 1 {
			return Config{}, fmt.Errorf("DB_MAX_CONNS must be a positive integer")
		}
		cfg.DBMaxConns = int32(parsed)
	}

	if value := os.Getenv("SHUTDOWN_TIMEOUT"); value != "" {
		cfg.ShutdownTimeout, err = time.ParseDuration(value)
		if err != nil || cfg.ShutdownTimeout <= 0 {
			return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT must be a positive duration")
		}
	}

	if value := os.Getenv("MIGRATE_ON_START"); value != "" {
		cfg.MigrateOnStart, err = strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("MIGRATE_ON_START must be a boolean")
		}
	}

	return cfg, nil
}
