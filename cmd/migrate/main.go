// Command migrate runs GORM AutoMigrate for all models.
// Usage: DATABASE_DSN=... go run ./cmd/migrate
package main

import (
	"fmt"
	"os"

	"github.com/EricGusmao/taskify/internal/auth"
	"github.com/EricGusmao/taskify/internal/infra"
	"github.com/EricGusmao/taskify/internal/teams"
)

func main() {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "migrate: DATABASE_DSN is required")
		os.Exit(1)
	}

	db, err := infra.NewDB(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: open db: %v\n", err)
		os.Exit(1)
	}

	if err := db.AutoMigrate(&auth.User{}, &teams.Team{}); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: auto migrate: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("migrate: done")
}
