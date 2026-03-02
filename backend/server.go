package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/rs/cors"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// CORS
	allowedOrigins := []string{"http://localhost:5173"} // Vite dev server
	if corsOrigin := os.Getenv("CORS_ALLOWED_ORIGIN"); corsOrigin != "" {
		for _, origin := range strings.Split(corsOrigin, ",") {
			if o := strings.TrimSpace(origin); o != "" {
				allowedOrigins = append(allowedOrigins, o)
			}
		}
	}
	c := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
	})
	_ = c // TODO: wire up with GraphQL handler in graphql-api context

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// TODO: GraphQL handler + SSE subscriptions wired in graphql-api context

	log.Printf("server listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
