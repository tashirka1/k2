package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	EnvFile         string
	DBName          string
	ExternalPort    string
	SessionKey      string
	Username        string
	Password        string
	CollectInterval time.Duration
	Retention       time.Duration
}

func Load() (Config, error) {
	envFile := os.Getenv("K2_ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	if err := godotenv.Load(envFile); err != nil {
		slog.Warn("env file not found, using environment variables", "file", envFile)
	}

	sessionKey := os.Getenv("K2_SESSION_KEY")
	if sessionKey == "" {
		return Config{}, fmt.Errorf("K2_SESSION_KEY is not set")
	}

	port := os.Getenv("K2_PORT")
	if port == "" {
		port = "8000"
	}

	dbName := os.Getenv("K2_DB_NAME")
	if dbName == "" {
		dbName = "./data/k2.db"
	}

	externalPort := os.Getenv("K2_EXTERNAL_PORT")
	if externalPort == "" {
		externalPort = "9000"
	}

	collectInterval := 30 * time.Second
	if v := os.Getenv("K2_COLLECT_INTERVAL"); v != "" {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid K2_COLLECT_INTERVAL: %w", err)
		}
		collectInterval = parsed
	}

	retention := 168 * time.Hour
	if v := os.Getenv("K2_RETENTION"); v != "" {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid K2_RETENTION: %w", err)
		}
		retention = parsed
	}

	return Config{
		Port:            port,
		EnvFile:         envFile,
		DBName:          dbName,
		ExternalPort:    externalPort,
		SessionKey:      sessionKey,
		Username:        os.Getenv("K2_USERNAME"),
		Password:        os.Getenv("K2_PASSWORD"),
		CollectInterval: collectInterval,
		Retention:       retention,
	}, nil
}
