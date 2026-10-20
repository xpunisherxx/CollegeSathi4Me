package tasks

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxTitleLen = 120

var validPriorities = map[string]bool{"high": true, "medium": true, "gentle": true}

// Handler turns HTTP requests into Store calls and Store results into JSON.
type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Register attaches all task routes to the router.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tasks", h.list)
	mux.HandleFunc("POST /api/tasks", h.create)
	mux.HandleFunc("PATCH /api/tasks/{id}", h.update)
	mux.HandleFunc("DELETE /api/tasks/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.store.List())
}

type createRequest struct {
	Title    string `json:"title"`
	Priority string `json:"priority"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !decodeBody(w, r, &req) {
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if utf8.RuneCountInString(title) > maxTitleLen {
		writeError(w, http.StatusBadRequest, "title is too long (max 120 characters)")
		return
	}

	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}
	if !validPriorities[priority] {
		writeError(w, http.StatusBadRequest, "priority must be high, medium or gentle")
		return
	}

	task := h.store.Add(title, "Added today", priority)
	writeJSON(w, http.StatusCreated, task)
}

// updateRequest uses *bool so we can tell "false" apart from "missing".
type updateRequest struct {
	Done *bool `json:"done"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req updateRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.Done == nil {
		writeError(w, http.StatusBadRequest, "done is required")
		return
	}

	task, found := h.store.SetDone(id, *req.Done)
	if !found {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if !h.store.Delete(id) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- helpers ----------

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return 0, false
	}
	return id, true
}

// decodeBody reads JSON from the request (max 1 MB). On failure it has
// already replied with 400, so the caller just returns.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
