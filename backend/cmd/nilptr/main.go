// Command nilptr is the blog backend.
//
// Usage:
//
//	nilptr [serve]   run the HTTP server (default)
//	nilptr migrate   apply embedded SQL migrations and exit
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

	"github.com/jackc/pgx/v5"

	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/auth"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/config"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/httpapi"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/migrate"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/objstore"
	"github.com/RuslanKarabalin/nilptr.tech/backend/internal/store/postgres"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if err := run(cmd, log); err != nil {
		log.Error("fatal", "cmd", cmd, "err", err)
		os.Exit(1)
	}
}

func run(cmd string, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch cmd {
	case "serve":
		return serve(ctx, log)
	case "migrate":
		return runMigrations(ctx, log)
	case "-h", "--help", "help":
		fmt.Println("usage: nilptr [serve|migrate]")
		return nil
	default:
		return fmt.Errorf("unknown command %q (want serve or migrate)", cmd)
	}
}

func runMigrations(ctx context.Context, log *slog.Logger) error {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL is required")
	}
	connCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(connCtx, url)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()
	if err := migrate.Run(ctx, conn, log); err != nil {
		return err
	}
	log.Info("migrations up to date")
	return nil
}

func serve(ctx context.Context, log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	objects, err := objstore.New(objstore.Options{
		Endpoint:  cfg.S3Endpoint,
		Bucket:    cfg.S3Bucket,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
		Region:    cfg.S3Region,
		UseSSL:    cfg.S3UseSSL,
	})
	if err != nil {
		return fmt.Errorf("s3 client: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err = objects.Ping(pingCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("s3 bucket check: %w", err)
	}

	if cfg.AdminLogin != "" {
		hash, err := auth.HashPassword(cfg.AdminPassword)
		if err != nil {
			return err
		}
		if err := db.UpsertUser(ctx, cfg.AdminLogin, hash); err != nil {
			return fmt.Errorf("upsert admin: %w", err)
		}
		log.Info("admin user ensured", "login", cfg.AdminLogin)
	}

	api := httpapi.New(httpapi.Options{
		Store:          db,
		Auth:           auth.NewService(db, cfg.JWTSecret, nil),
		Objects:        objects,
		Log:            log,
		TrustedProxies: cfg.TrustedProxies,
		IPHashSecret:   cfg.IPHashSecret,
		CookieSecure:   cfg.CookieSecure,
		MaxUploadBytes: cfg.MaxUploadBytes,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
