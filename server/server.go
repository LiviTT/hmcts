package main

import (
	"log"
	"net/http"

	"github.com/hmcts/server/internal/database"
	"github.com/hmcts/server/internal/handler"
	"github.com/rs/cors"
)

func main() {
	db, err := database.New("./tasks.db")
	if err != nil {
		log.Fatalf("Failed to initialise database: %v", err)
	}

	mux := http.NewServeMux()

	h := handler.New(db)
	h.Routes(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	})

	log.Println("🚀 REST API running at http://localhost:8080")
	log.Println("   GET    /api/tasks")
	log.Println("   POST   /api/tasks")
	log.Println("   GET    /api/tasks/{id}")
	log.Println("   PATCH  /api/tasks/{id}/status")
	log.Println("   DELETE /api/tasks/{id}")

	if err := http.ListenAndServe(":8080", c.Handler(mux)); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}