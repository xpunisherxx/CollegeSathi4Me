package tasks

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultTestDatabaseURL = "postgres://collegesathi:devpassword@localhost:5432/collegesathi_test"

// newTestStore connects to the TEST database and empties the tasks table,
// so every test starts from a clean slate. It refuses to run against a
// database whose name does not end in _test, so it can never wipe real data.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = defaultTestDatabaseURL
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if name := pool.Config().ConnConfig.Database; !strings.HasSuffix(name, "_test") {
		t.Fatalf("refusing to run tests on database %q: name must end in _test", name)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE tasks RESTART IDENTITY`); err != nil {
		t.Fatalf("clean tasks table (is the test database migrated?): %v", err)
	}
	return NewStore(pool)
}

func mustAdd(t *testing.T, store *Store, title string) Task {
	t.Helper()
	task, err := store.Add(context.Background(), title, "", "medium")
	if err != nil {
		t.Fatalf("add %q: %v", title, err)
	}
	return task
}

func TestAddAssignsUniqueIDs(t *testing.T) {
	store := newTestStore(t)

	a := mustAdd(t, store, "first")
	b := mustAdd(t, store, "second")

	if a.ID == b.ID {
		t.Errorf("expected unique ids, both were %d", a.ID)
	}
}

func TestListReturnsAddedTasks(t *testing.T) {
	store := newTestStore(t)
	mustAdd(t, store, "first")

	got, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].Title != "first" {
		t.Errorf("expected one task titled %q, got %+v", "first", got)
	}
}

func TestSetDone(t *testing.T) {
	store := newTestStore(t)
	task := mustAdd(t, store, "finish me")

	got, err := store.SetDone(context.Background(), task.ID, true)
	if err != nil {
		t.Fatalf("set done: %v", err)
	}
	if !got.Done {
		t.Error("expected task to be done")
	}
}

func TestSetDoneNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.SetDone(context.Background(), 999, true)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	store := newTestStore(t)
	task := mustAdd(t, store, "delete me")

	if err := store.Delete(context.Background(), task.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no tasks left, got %+v", got)
	}
}

func TestDeleteNotFound(t *testing.T) {
	store := newTestStore(t)

	err := store.Delete(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
