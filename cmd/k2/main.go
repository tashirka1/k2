package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tashirka1/k2"

	auth_handler "github.com/tashirka1/k2/internal/auth/handler"
	auth_service "github.com/tashirka1/k2/internal/auth/service"
	auth_storage "github.com/tashirka1/k2/internal/auth/storage"
	"github.com/tashirka1/k2/internal/core/db"
	"github.com/tashirka1/k2/internal/core/health"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	if err := Run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func Run() error {
	// env
	if err := godotenv.Load(); err != nil {
		slog.Warn(".env file not found, using environment variables")
	}

	sessionKey := os.Getenv("SESSION_KEY")
	if sessionKey == "" {
		return fmt.Errorf("SESSION_KEY is not set")
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return fmt.Errorf("DB_NAME is not set")
	}

	// logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// database
	database, err := db.NewDB(dbName)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer database.Close()

	// session
	sessionStore := sessions.NewCookieStore([]byte(sessionKey))

	// echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Logger.SetOutput(io.Discard)

	// middleware
	protection := http.NewCrossOriginProtection()
	csrfMiddleware := echo.WrapMiddleware(func(next http.Handler) http.Handler {
		return protection.Handler(next)
	})
	e.Use(csrfMiddleware)
	e.Use(middleware.RequestLogger())
	e.Use(session.Middleware(sessionStore))
	e.Use(middleware.ContextTimeout(10 * time.Second))
	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			return c.Request().Method != http.MethodPost && c.Path() != "/link/" && c.Path() != "/link"
		},
		Store: middleware.NewRateLimiterMemoryStore(3),
	}))
	e.StaticFS("/static", echo.MustSubFS(k2.EmbeddedStatic, "static"))
	e.GET("/health", health.Handler(database))

	// handler
	authStrg := auth_storage.NewUser(database)
	authSvc := auth_service.NewUser(authStrg)
	auth_handler.SetupHandlers(e, authSvc)

	// signal
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting on :8000")
		if err := e.Start(":8000"); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}

	slog.Info("server stopped")
	return nil
}
