package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xpunisherxx/collegesathi/internal/habits"
	"github.com/xpunisherxx/collegesathi/internal/tasks"
)

const (
	addr = ":8080"

	// Local development default only. In Docker or Kubernetes you will set
	// the DATABASE_URL environment variable instead.
	defaultDatabaseURL = "postgres://collegesathi:devpassword@localhost:5432/collegesathi"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("health: encode response: %v", err)
	}
}

func databaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return defaultDatabaseURL
}

// connectDB opens a connection pool and checks the database really answers.
func connectDB(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

// seedTasks adds the sample task only when the table is empty.
func seedTasks(ctx context.Context, store *tasks.Store) error {
	existing, err := store.List(ctx)
	if err != nil {
		return fmt.Errorf("seed: list tasks: %w", err)
	}
	if len(existing) > 0 {
		return nil
	}
	if _, err := store.Add(ctx, "Enter your tasks here cheeky boi!", "", "medium"); err != nil {
		return fmt.Errorf("seed: add task: %w", err)
	}
	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := connectDB(ctx, databaseURL())
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	store := tasks.NewStore(pool)
	if err := seedTasks(ctx, store); err != nil {
		log.Fatalf("%v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", healthHandler)
	tasks.NewHandler(store).Register(mux)
	habits.NewHandler(habits.NewStore(pool)).Register(mux)

	// Serve the dashboard from the same server, so the frontend and the API
	// share one origin (no CORS needed). Run the server from backend/.
	
	mux.Handle("GET /{$}", http.RedirectHandler("/HTML/", http.StatusFound))
	mux.Handle("GET /", http.FileServer(http.Dir("../frontend")))

	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
