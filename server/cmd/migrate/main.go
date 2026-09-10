package main

import (
	"context"
	"fmt"
	"os"

	"github.com/GnEveLynn/FormTally/server/internal/postgres"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate up|down|status")
		os.Exit(2)
	}
	if err := postgres.Migrate(context.Background(), os.Getenv("DATABASE_URL"), "migrations", os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
