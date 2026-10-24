package habits

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultTestDatabaseURL = "postgres://collegesathi:devpassword@localhost:5432/collegesathi_test"

// A fixed "today" keeps the tests independent of the real date.
var testToday = time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)

func daysAgo(n int) time.Time {
	return testToday.AddDate(0, 0, -n)
}

// newTestStore connects to the TEST database and empties the habit tables.
// It refuses to run on any database whose name does not end in _test.
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
	if _, err := pool.Exec(ctx, `TRUNCATE habits RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("clean habit tables (is 002_habits.sql applied to the test database?): %v", err)
	}
	return NewStore(pool)
}

func mustCreate(t *testing.T, store *Store, name string) Habit {
	t.Helper()
	h, err := store.Create(context.Background(), name)
	if err != nil {
		t.Fatalf("create %q: %v", name, err)
	}
	return h
}

func mustTick(t *testing.T, store *Store, id int, day time.Time, checked bool) {
	t.Helper()
	if err := store.SetChecked(context.Background(), id, day, checked); err != nil {
		t.Fatalf("set checked: %v", err)
	}
}

// onlyHabit lists habits and expects exactly one.
func onlyHabit(t *testing.T, store *Store) Habit {
	t.Helper()
	got, err := store.List(context.Background(), testToday)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 habit, got %+v", got)
	}
	return got[0]
}

func TestCreateAndList(t *testing.T) {
	store := newTestStore(t)
	mustCreate(t, store, "Morning rise")

	h := onlyHabit(t, store)
	if h.Name != "Morning rise" || h.Streak != 0 || h.DoneToday {
		t.Errorf("unexpected habit: %+v", h)
	}
}

func TestTickToday(t *testing.T) {
	store := newTestStore(t)
	created := mustCreate(t, store, "Read")
	mustTick(t, store, created.ID, testToday, true)

	h := onlyHabit(t, store)
	if !h.DoneToday || h.Streak != 1 {
		t.Errorf("expected done today with streak 1, got %+v", h)
	}
}

func TestStreakAcrossDays(t *testing.T) {
	store := newTestStore(t)
	created := mustCreate(t, store, "Workout")
	for n := 0; n < 3; n++ {
		mustTick(t, store, created.ID, daysAgo(n), true)
	}

	if h := onlyHabit(t, store); h.Streak != 3 {
		t.Errorf("expected streak 3, got %+v", h)
	}
}

func TestTickTwiceIsHarmless(t *testing.T) {
	store := newTestStore(t)
	created := mustCreate(t, store, "Water")
	mustTick(t, store, created.ID, testToday, true)
	mustTick(t, store, created.ID, testToday, true)

	if h := onlyHabit(t, store); h.Streak != 1 {
		t.Errorf("expected streak 1 after double tick, got %+v", h)
	}
}

func TestUntick(t *testing.T) {
	store := newTestStore(t)
	created := mustCreate(t, store, "Journal")
	mustTick(t, store, created.ID, testToday, true)
	mustTick(t, store, created.ID, testToday, false)

	h := onlyHabit(t, store)
	if h.DoneToday || h.Streak != 0 {
		t.Errorf("expected not done and streak 0, got %+v", h)
	}
}

func TestTickUnknownHabit(t *testing.T) {
	store := newTestStore(t)

	err := store.SetChecked(context.Background(), 999, testToday, true)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
