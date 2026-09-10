package postgres

import (
	"context"
	"crypto/rand"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestOpenConnectsToRealPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	pool, err := Open(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	var database string
	if err := pool.QueryRow(context.Background(), "select current_database()").Scan(&database); err != nil {
		t.Fatal(err)
	}
	if database != "formtally" {
		t.Fatalf("database = %q, want formtally", database)
	}
}

func TestMigrationsUpgradeAndRollbackEmptySchema(t *testing.T) {
	databaseURL := isolatedDatabaseURL(t)

	if err := Migrate(context.Background(), databaseURL, "../../migrations", "up"); err != nil {
		t.Fatal(err)
	}
	pool, err := Open(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	var installed bool
	if err := pool.QueryRow(context.Background(), "select exists (select 1 from pg_extension where extname = 'pgcrypto')").Scan(&installed); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	pool.Close()
	if !installed {
		t.Fatal("pgcrypto extension was not installed")
	}

	if err := Migrate(context.Background(), databaseURL, "../../migrations", "down"); err != nil {
		t.Fatal(err)
	}
	pool, err = Open(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.QueryRow(context.Background(), "select exists (select 1 from pg_extension where extname = 'pgcrypto')").Scan(&installed); err != nil {
		t.Fatal(err)
	}
	if installed {
		t.Fatal("pgcrypto extension remains after rollback")
	}
}

func TestMigrateRequiresDatabaseURL(t *testing.T) {
	if err := Migrate(context.Background(), "", "../../migrations", "up"); err == nil {
		t.Fatal("empty database URL accepted")
	}
}

func isolatedDatabaseURL(t *testing.T) string {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	admin, err := Open(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	database := "test_" + strings.ToLower(rand.Text())
	if _, err := admin.Exec(context.Background(), "create database "+pgx.Identifier{database}.Sanitize()); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "drop database "+pgx.Identifier{database}.Sanitize()+" with (force)")
		admin.Close()
	})
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/" + database
	return parsed.String()
}
