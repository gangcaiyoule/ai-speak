// Package main provides the local database migration entry point.
package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/gangcaiyoule/ai-speak/server/internal/platform/migrate"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	migrations, err := migrate.Load(os.DirFS("migrations"))
	if err != nil {
		log.Fatal(err)
	}
	if err := migrate.Run(context.Background(), migrate.PostgresStore{DB: db}, migrations); err != nil {
		log.Fatal(err)
	}
	log.Printf("migration complete: %d files available", len(migrations))
}
