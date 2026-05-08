package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"travel-planner-viewer/backend/internal/handlers"
	"travel-planner-viewer/backend/internal/store"
)

func main() {
	adminMode := strings.ToLower(os.Getenv("ADMIN_MODE")) == "true"
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	st, err := store.NewPlansStore(dataDir)
	if err != nil {
		log.Fatalf("failed to initialize store: %v", err)
	}

	mux := http.NewServeMux()
	plansHandler := handlers.NewPlansHandler(st, adminMode)
	plansHandler.Register(mux)
	importHandler := handlers.NewImportHandler(st, adminMode)
	importHandler.Register(mux)
	mux.Handle("/", http.FileServer(http.Dir("../frontend")))

	addr := ":" + getPort()
	log.Printf("server listening on %s (ADMIN_MODE=%v)", addr, adminMode)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func getPort() string {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		return "8080"
	}
	return port
}

func withCORS(next http.Handler) http.Handler {
	allowedOrigins := map[string]bool{
		"http://localhost:3000": true,
		"http://localhost:4200": true,
		"http://localhost:5173": true,
		"http://localhost:5500": true,
		"http://127.0.0.1:3000": true,
		"http://127.0.0.1:4200": true,
		"http://127.0.0.1:5173": true,
		"http://127.0.0.1:5500": true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
