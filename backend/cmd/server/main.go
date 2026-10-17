package main

import (
	"encoding/json"
	"log"
	"net/http"

	// Replace with the `module` line in backend/go.mod, plus /internal/tasks
	"github.com/xpunisherxx/collegesathi/internal/tasks"
)

const addr = ":8080"

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("health: encode response: %v", err)
	}
}

// seedTasks gives the in-memory store the same sample data the dashboard had.
func seedTasks(store *tasks.Store) {
	store.Add("Review economics lecture notes", "Study · 45 min", "high")
	store.Add("Submit design research brief", "College · Due 4:00 PM", "high")
	store.Add("Evening walk around campus", "Health · 30 min", "gentle")
	store.Add("Read 20 pages of Meditations", "Personal · 25 min", "medium")
	store.SetDone(3, true)
}

func main() {
	store := tasks.NewStore()
	seedTasks(store)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", healthHandler)
	tasks.NewHandler(store).Register(mux)

	// Serve the dashboard from the same server, so the frontend and the API
	// share one origin (no CORS needed). Run the server from backend/.
	mux.Handle("GET /", http.FileServer(http.Dir("../frontend")))

	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
