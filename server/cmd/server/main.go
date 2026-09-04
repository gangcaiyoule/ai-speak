// Package main starts the ai-speak HTTP server.
package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gangcaiyoule/ai-speak/server/internal/agent"
	"github.com/gangcaiyoule/ai-speak/server/internal/coaching"
	"github.com/gangcaiyoule/ai-speak/server/internal/identity"
	"github.com/gangcaiyoule/ai-speak/server/internal/voiceecho"
	_ "github.com/lib/pq"
)

// HealthResponse is the response returned by the health endpoint.
type HealthResponse struct {
	Status string `json:"status"`
}

// healthHandler reports whether the HTTP process is running.
func healthHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(HealthResponse{Status: "ok"})
}

// buildRouter creates the server's HTTP routes.
func buildRouter() http.Handler {
	return buildRouterWithRepository(identity.NewMemoryRepository())
}

func buildRouterWithRepository(repository identity.Repository) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	auth := identity.NewService(repository)
	identity.NewHTTPHandler(auth).RegisterRoutes(mux)
	agent.NewHTTPHandler(agent.StubService{}).RegisterRoutes(mux)
	coaching.NewHTTPHandler().RegisterRoutes(mux)
	mux.Handle("GET /ws/voice/echo", voiceecho.NewWSSHandler())
	return mux
}

// main starts the HTTP server on SERVER_HOST:SERVER_PORT.
func main() {
	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}
	var repository identity.Repository = identity.NewMemoryRepository()
	var db *sql.DB
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		var err error
		db, err = sql.Open("postgres", databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		if err = db.Ping(); err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		repository = identity.NewPostgresRepository(db)
	}
	address := host + ":" + port
	log.Printf("ai-speak server listening on %s", address)
	if err := http.ListenAndServe(address, buildRouterWithRepository(repository)); err != nil {
		log.Fatal(err)
	}
}
