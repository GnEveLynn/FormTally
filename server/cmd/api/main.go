package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/GnEveLynn/FormTally/server/internal/app"
	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
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
	listener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		logger.Error("listen failed", "error", err)
		os.Exit(1)
	}
	logger.Info("api listening", "address", listener.Addr().String())
	if err := app.Serve(ctx, app.NewServer(cfg, httpapi.NewRouter()), listener); err != nil {
		logger.Error("api stopped with error", "error", err)
		os.Exit(1)
	}
	logger.Info("api stopped")
}
