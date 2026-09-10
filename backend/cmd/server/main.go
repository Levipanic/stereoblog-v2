package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
	"github.com/Levipanic/stereoblog-v2/backend/internal/httpapi"
	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}
	db, schema, err := database.Open(context.Background(), cfg.Storage.DatabasePath)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer db.Close()
	migration, err := database.Migrate(context.Background(), db, cfg.Storage.DatabasePath, schema, cfg.IsProduction())
	if err != nil {
		return fmt.Errorf("database migration: %w", err)
	}
	logger.Info("database ready", "path", cfg.Storage.DatabasePath, "initial_schema", schema, "migration_version", migration.Version, "migrations_applied", migration.Applied, "backup_created", migration.BackupPath != "", "backup_path", migration.BackupPath)
	router, err := httpapi.NewRouter(cfg, logger)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              cfg.ListenAddress(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		// ponytail: global 30s limits suit current small requests; use route-aware deadlines if large uploads or backups need longer.
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server starting", "address", server.Addr, "environment", cfg.Server.Environment)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("server shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}
