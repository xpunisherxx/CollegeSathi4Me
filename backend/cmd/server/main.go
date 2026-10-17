package main

import (
	"encoding/json"
	"log"
	"net/http"
)

const addr = ":8080"

// healthHandler answers "are you alive?" with {"status":"ok"}.
// w is how we write the response; r is the request we received.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("health: encode response: %v", err)
	}
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", healthHandler)

	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
