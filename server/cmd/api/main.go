package main

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GnEveLynn/FormTally/server/internal/account"
	"github.com/GnEveLynn/FormTally/server/internal/analysis"
	"github.com/GnEveLynn/FormTally/server/internal/app"
	"github.com/GnEveLynn/FormTally/server/internal/auth"
	"github.com/GnEveLynn/FormTally/server/internal/days"
	"github.com/GnEveLynn/FormTally/server/internal/goals"
	"github.com/GnEveLynn/FormTally/server/internal/httpapi"
	"github.com/GnEveLynn/FormTally/server/internal/idempotency"
	"github.com/GnEveLynn/FormTally/server/internal/meals"
	"github.com/GnEveLynn/FormTally/server/internal/postgres"
	"github.com/GnEveLynn/FormTally/server/internal/profile"
	"github.com/GnEveLynn/FormTally/server/internal/sms"
	"github.com/GnEveLynn/FormTally/server/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	checkConfig, err := parseArgs(os.Args[1:])
	if err != nil {
		logger.Error("invalid arguments", "error", err)
		os.Exit(2)
	}
	cfg, err := app.LoadConfig()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	if checkConfig {
		logger.Info("configuration valid")
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	var smsLogger *slog.Logger
	if cfg.Environment == "development" && cfg.SMSDriver == "test" {
		smsLogger = logger
	}
	authService := auth.NewService(auth.NewPostgresStore(pool), sms.NewTestSender(smsLogger))
	authenticate := func(r *http.Request) (string, error) {
		session, err := authService.GetSession(r.Context(), auth.SessionToken(r))
		return session.User.ID, err
	}
	authHandler := auth.NewHandler(authService, authenticate)
	goalService := goals.NewService(goals.NewPostgresStore(pool))
	profileHandler := profile.NewHandler(profile.NewService(profile.NewPostgresStore(pool), goalService), authenticate)
	goalsHandler := goals.NewHandler(goalService, authenticate)
	var objectStore storage.Store
	var privateImages func(*http.ServeMux)
	if cfg.StorageDriver == "s3" {
		objectStore, err = storage.NewS3Store(storage.S3Config{Endpoint: cfg.S3Endpoint, Region: cfg.S3Region, Bucket: cfg.S3Bucket, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey}, time.Now)
		if err != nil {
			logger.Error("invalid storage configuration", "error", err)
			os.Exit(1)
		}
	} else {
		filesystem := storage.NewFilesystemStore(cfg.StoragePath, []byte(cfg.ImageURLSecret), time.Now)
		objectStore = filesystem
		privateImages = func(mux *http.ServeMux) { mux.Handle("GET /v1/private-images/{key}", filesystem) }
	}
	idempotencyStore := idempotency.NewPostgresStore(pool)
	analyzer := analysis.NewOpenAIAnalyzer(analysis.OpenAIConfig{APIKey: cfg.OpenAIAPIKey, Model: cfg.OpenAIModel, Timeout: cfg.OpenAITimeout})
	analysisHandler := analysis.NewHandler(analysis.NewService(analysis.NewPostgresStore(pool), objectStore, analyzer, idempotencyStore), authenticate)
	mealService := meals.NewService(meals.NewPostgresStore(pool), idempotencyStore, objectStore)
	mealHandler := meals.NewHandler(mealService, authenticate)
	daysHandler := days.NewHandler(days.NewService(days.NewPostgresStore(pool), goals.NewPostgresStore(pool)), authenticate)
	accountHandler := account.NewHandler(account.NewService(account.NewPostgresStore(pool)), authenticate)
	deletionWorker := storage.NewDeletionWorker(storage.NewPostgresDeletionRepository(pool), objectStore)
	go func() {
		for {
			_, _ = deletionWorker.RunOnce(ctx)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Minute):
			}
		}
	}()
	listener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		logger.Error("listen failed", "error", err)
		os.Exit(1)
	}
	logger.Info("api listening", "address", listener.Addr().String())
	register := []func(*http.ServeMux){authHandler.Register, profileHandler.Register, goalsHandler.Register, analysisHandler.Register, mealHandler.Register, daysHandler.Register, accountHandler.Register}
	if privateImages != nil {
		register = append(register, privateImages)
	}
	if err := app.Serve(ctx, app.NewServer(cfg, httpapi.NewRouter(logger, cfg.AllowedOrigins, register...)), listener); err != nil {
		logger.Error("api stopped with error", "error", err)
		os.Exit(1)
	}
	logger.Info("api stopped")
}

func parseArgs(args []string) (bool, error) {
	flags := flag.NewFlagSet("formtally-api", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	checkConfig := flags.Bool("check-config", false, "validate configuration and exit")
	if err := flags.Parse(args); err != nil {
		return false, err
	}
	if flags.NArg() != 0 {
		return false, flag.ErrHelp
	}
	return *checkConfig, nil
}
