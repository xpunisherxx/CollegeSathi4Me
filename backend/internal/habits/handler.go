package habits

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const maxNameLen = 60

// Handler turns HTTP requests into Store calls and Store results into JSON.
type Handler struct {
	store *Store
	now   func() time.Time // replaceable in tests
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store, now: time.Now}
}

// Register attaches all habit routes to the router.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/habits", h.list)
	mux.HandleFunc("POST /api/habits", h.create)
	mux.HandleFunc("PATCH /api/habits/{id}", h.update)
}

// today returns today's date in the server's local time zone as a plain
// calendar day (midnight UTC), which is what the DATE column expects.
func (h *Handler) today() time.Time {
	n := h.now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context(), h.today())
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type createRequest struct {
	Name string `json:"name"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !decodeBody(w, r, &req) {
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if utf8.RuneCountInString(name) > maxNameLen {
		writeError(w, http.StatusBadRequest, "name is too long (max 60 characters)")
		return
	}

	habit, err := h.store.Create(r.Context(), name)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, habit)
}

// updateRequest uses *bool so we can tell "false" apart from "missing".
type updateRequest struct {
	Done *bool `json:"done"`
}

// update ticks or unticks today. It replies 204; the frontend reloads the
// list to get the new streak.
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

	if err := h.store.SetChecked(r.Context(), id, h.today(), *req.Done); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- helpers ----------

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid habit id")
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

// writeStoreError maps a store error to an HTTP reply: 404 for a missing
// habit, 500 (logged, details hidden from the client) for anything else.
func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "habit not found")
		return
	}
	log.Printf("habits store: %v", err)
	writeError(w, http.StatusInternalServerError, "internal error")
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
