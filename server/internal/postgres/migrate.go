package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func Migrate(ctx context.Context, databaseURL, directory, command string) error {
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	config, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	database := stdlib.OpenDB(*config)
	defer database.Close()
	if err := database.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if err := goose.RunContext(ctx, command, database, directory); err != nil {
		return fmt.Errorf("goose %s: %w", command, err)
	}
	return nil
}
