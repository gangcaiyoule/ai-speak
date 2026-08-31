// Package main starts the ai-speak HTTP server.
package main

import (
	"encoding/json"
	"github.com/gangcaiyoule/ai-speak/server/internal/identity"
	"log"
	"net/http"
	"os"
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
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	identity.NewHTTPHandler(identity.StubAuthService{}).RegisterRoutes(mux)
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
	address := host + ":" + port
	log.Printf("ai-speak server listening on %s", address)
	if err := http.ListenAndServe(address, buildRouter()); err != nil {
		log.Fatal(err)
	}
}
