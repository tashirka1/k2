package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/tashirka1/k2"

	auth_handler "github.com/tashirka1/k2/internal/auth/handler"
	auth_model "github.com/tashirka1/k2/internal/auth/model"
	auth_service "github.com/tashirka1/k2/internal/auth/service"
	auth_storage "github.com/tashirka1/k2/internal/auth/storage"
	"github.com/tashirka1/k2/internal/core/config"
	"github.com/tashirka1/k2/internal/core/db"
	"github.com/tashirka1/k2/internal/core/health"
	"github.com/tashirka1/k2/internal/core/stats"
	metrics_handler "github.com/tashirka1/k2/internal/metrics/handler"
	metrics_service "github.com/tashirka1/k2/internal/metrics/service"
	metrics_storage "github.com/tashirka1/k2/internal/metrics/storage"

	"github.com/gorilla/sessions"
	echo_session "github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/moby/moby/client"
	"github.com/spf13/cobra"
)

var cfg config.Config

func applyGCLimits() {
	if os.Getenv("GOGC") == "" {
		debug.SetGCPercent(50)
	}
	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(40 << 20)
	}
}

func main() {
	applyGCLimits()

	rootCmd := &cobra.Command{
		Use:   "k2",
		Short: "K2 server monitoring tool",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if cmd.Name() == "help" {
				return nil
			}

			logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
			slog.SetDefault(logger)

			var err error
			cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			return nil
		},
		RunE: startServer,
	}

	credentialsCmd := &cobra.Command{
		Use:   "credentials",
		Short: "Display auth credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			database, err := db.NewDB(cfg.DBName)
			if err != nil {
				return fmt.Errorf("database: %w", err)
			}
			defer database.Close()

			authStrg := auth_storage.NewAuth(database)
			authSvc := auth_service.NewAuth(authStrg)

			creds, err := authSvc.EnsureCredentials(cmd.Context(), cfg.Username, cfg.Password)
			if err != nil {
				return fmt.Errorf("ensure credentials: %w", err)
			}

			printCredentials(creds)
			return nil
		},
	}

	rootCmd.AddCommand(credentialsCmd)

	if err := rootCmd.Execute(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func printCredentials(creds auth_model.Credentials) {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  K2 Server Monitor")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  Username:  %s\n", creds.Username)
	fmt.Printf("  Password:  %s\n", creds.Password)
	fmt.Println(strings.Repeat("=", 60))
}

func startServer(cmd *cobra.Command, _ []string) error {
	database, err := db.NewDB(cfg.DBName)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	sessionStore := sessions.NewCookieStore([]byte(cfg.SessionKey))

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Logger.SetOutput(io.Discard)

	e.Use(middleware.RequestLogger())
	e.Use(echo_session.Middleware(sessionStore))
	e.Use(middleware.ContextTimeout(10 * time.Second))
	e.Pre(middleware.RemoveTrailingSlash())
	e.StaticFS("/static", echo.MustSubFS(k2.EmbeddedStatic, "static"))
	e.GET("/health", health.Handler(database))
	e.GET("/debug/stats", stats.Handler())

	authStrg := auth_storage.NewAuth(database)
	authSvc := auth_service.NewAuth(authStrg)

	creds, err := authSvc.EnsureCredentials(cmd.Context(), cfg.Username, cfg.Password)
	if err != nil {
		return fmt.Errorf("ensure credentials: %w", err)
	}

	printCredentials(creds)

	auth_handler.SetupHandlers(e, authSvc)

	dockerClient, err := client.New(client.FromEnv)
	if err != nil {
		slog.Warn("docker client init failed, container metrics disabled", "error", err)
	} else {
		defer dockerClient.Close()
	}

	metricsStrg := metrics_storage.NewMetrics(database)
	metricsSvc := metrics_service.NewMetrics(metricsStrg, dockerClient, cfg.Retention)
	metrics_handler.SetupHandlers(e, metricsSvc, cfg.CollectInterval.String())

	collectorCtx, cancelCollector := context.WithCancel(cmd.Context())
	defer cancelCollector()
	go func() {
		if err := metricsSvc.RunCollector(collectorCtx, cfg.CollectInterval); err != nil {
			slog.Error("collector stopped", "error", err)
		}
	}()
	go func() {
		if err := metricsSvc.RunMaintenance(collectorCtx); err != nil {
			slog.Error("maintenance stopped", "error", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := fmt.Sprintf(":%s", cfg.Port)
	go func() {
		slog.Info("server starting", "addr", addr)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")
	cancelCollector()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}

	slog.Info("server stopped")
	return nil
}
