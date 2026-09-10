package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/GnEveLynn/FormTally/server/internal/app"
	"github.com/GnEveLynn/FormTally/server/internal/auth"
	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
	"github.com/GnEveLynn/FormTally/server/internal/postgres"
	"github.com/GnEveLynn/FormTally/server/internal/sms"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := app.LoadConfig()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	authHandler := auth.NewHandler(auth.NewService(auth.NewPostgresStore(pool), sms.NewTestSender()))
	listener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		logger.Error("listen failed", "error", err)
		os.Exit(1)
	}
	logger.Info("api listening", "address", listener.Addr().String())
	if err := app.Serve(ctx, app.NewServer(cfg, httpapi.NewRouter(logger, cfg.AllowedOrigins, authHandler.Register)), listener); err != nil {
		logger.Error("api stopped with error", "error", err)
		os.Exit(1)
	}
	logger.Info("api stopped")
}
