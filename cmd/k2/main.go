package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tashirka1/k2"

	admin_handler "github.com/tashirka1/k2/internal/admin/handler"
	admin_service "github.com/tashirka1/k2/internal/admin/service"
	admin_storage "github.com/tashirka1/k2/internal/admin/storage"
	"github.com/tashirka1/k2/internal/core/config"
	"github.com/tashirka1/k2/internal/core/db"
	"github.com/tashirka1/k2/internal/core/health"
	metrics_handler "github.com/tashirka1/k2/internal/metrics/handler"
	metrics_service "github.com/tashirka1/k2/internal/metrics/service"
	metrics_storage "github.com/tashirka1/k2/internal/metrics/storage"

	"github.com/gorilla/sessions"
	echosession "github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
)

var cfg config.Config

func main() {
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
		Short: "Display admin credentials",
		RunE: func(cmd *cobra.Command, _ []string) error {
			database, err := db.NewDB(cfg.DBName)
			if err != nil {
				return fmt.Errorf("database: %w", err)
			}
			defer database.Close()

			adminStrg := admin_storage.NewAdmin(database)
			adminSvc := admin_service.NewAdmin(adminStrg)

			username, password, err := adminSvc.EnsureCredentials(cmd.Context())
			if err != nil {
				return fmt.Errorf("ensure credentials: %w", err)
			}

			fmt.Printf("Username:  %s\n", username)
			fmt.Printf("Password:  %s\n", password)
			return nil
		},
	}

	rootCmd.AddCommand(credentialsCmd)

	if err := rootCmd.Execute(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
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

	e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup:    "header:X-CSRF-Token",
		CookieName:     "_csrf",
		CookiePath:     "/",
		CookieHTTPOnly: true,
		CookieSameSite: http.SameSiteLaxMode,
	}))
	e.Use(middleware.RequestLogger())
	e.Use(echosession.Middleware(sessionStore))
	e.Use(middleware.ContextTimeout(10 * time.Second))
	e.Pre(middleware.RemoveTrailingSlash())
	e.StaticFS("/static", echo.MustSubFS(k2.EmbeddedStatic, "static"))
	e.GET("/health", health.Handler(database))

	adminStrg := admin_storage.NewAdmin(database)
	adminSvc := admin_service.NewAdmin(adminStrg)

	username, password, err := adminSvc.EnsureCredentials(cmd.Context())
	if err != nil {
		return fmt.Errorf("ensure credentials: %w", err)
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("  K2 Server Monitor")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("  Username:  %s\n", username)
	fmt.Printf("  Password:  %s\n", password)
	fmt.Println(strings.Repeat("=", 60))

	admin_handler.SetupHandlers(e, adminSvc)

	metricsStrg := metrics_storage.NewMetrics(database)
	metricsSvc := metrics_service.NewMetrics(metricsStrg)
	metrics_handler.SetupHandlers(e, metricsSvc)

	collectorCtx, cancelCollector := context.WithCancel(cmd.Context())
	defer cancelCollector()
	go func() {
		if err := metricsSvc.RunCollector(collectorCtx, 10*time.Second); err != nil {
			slog.Error("collector stopped", "error", err)
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
