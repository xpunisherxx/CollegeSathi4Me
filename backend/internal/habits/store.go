package habits

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when no habit has the requested id.
var ErrNotFound = errors.New("habit not found")

// pgForeignKeyViolation is the Postgres error code for "that habit_id
// does not exist in the habits table".
const pgForeignKeyViolation = "23503"

// Habit is what the API sends to the frontend. Streak and DoneToday are
// computed on every request, never stored.
type Habit struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Streak    int    `json:"streak"`
	DoneToday bool   `json:"doneToday"`
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Create(ctx context.Context, name string) (Habit, error) {
	var h Habit
	err := s.db.QueryRow(ctx,
		`INSERT INTO habits (name) VALUES ($1) RETURNING id, name`,
		name,
	).Scan(&h.ID, &h.Name)
	if err != nil {
		return Habit{}, fmt.Errorf("create habit: %w", err)
	}
	return h, nil
}

// List returns every habit with its current streak, counted as of today.
func (s *Store) List(ctx context.Context, today time.Time) ([]Habit, error) {
	habits, err := s.listHabits(ctx)
	if err != nil {
		return nil, err
	}
	days, err := s.checkinDays(ctx)
	if err != nil {
		return nil, err
	}

	for i := range habits {
		habitDays := days[habits[i].ID]
		habits[i].Streak = CurrentStreak(habitDays, today)
		habits[i].DoneToday = containsDay(habitDays, today)
	}
	return habits, nil
}

// SetChecked ticks (checked=true) or unticks (checked=false) a habit for
// one day. Both are safe to repeat. Ticking an unknown habit returns
// ErrNotFound.
func (s *Store) SetChecked(ctx context.Context, habitID int, day time.Time, checked bool) error {
	if !checked {
		_, err := s.db.Exec(ctx,
			`DELETE FROM habit_checkins WHERE habit_id = $1 AND day = $2`,
			habitID, day)
		if err != nil {
			return fmt.Errorf("untick habit: %w", err)
		}
		return nil
	}

	_, err := s.db.Exec(ctx,
		`INSERT INTO habit_checkins (habit_id, day) VALUES ($1, $2)
         ON CONFLICT DO NOTHING`,
		habitID, day)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("tick habit: %w", err)
	}
	return nil
}

func (s *Store) listHabits(ctx context.Context) ([]Habit, error) {
	rows, err := s.db.Query(ctx, `SELECT id, name FROM habits ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list habits: %w", err)
	}
	defer rows.Close()

	habits := []Habit{}
	for rows.Next() {
		var h Habit
		if err := rows.Scan(&h.ID, &h.Name); err != nil {
			return nil, fmt.Errorf("scan habit: %w", err)
		}
		habits = append(habits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list habits: %w", err)
	}
	return habits, nil
}

// checkinDays returns every ticked day, grouped by habit id.
func (s *Store) checkinDays(ctx context.Context) (map[int][]time.Time, error) {
	rows, err := s.db.Query(ctx, `SELECT habit_id, day FROM habit_checkins`)
	if err != nil {
		return nil, fmt.Errorf("list check-ins: %w", err)
	}
	defer rows.Close()

	days := map[int][]time.Time{}
	for rows.Next() {
		var habitID int
		var day time.Time
		if err := rows.Scan(&habitID, &day); err != nil {
			return nil, fmt.Errorf("scan check-in: %w", err)
		}
		days[habitID] = append(days[habitID], day)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list check-ins: %w", err)
	}
	return days, nil
}

func containsDay(days []time.Time, target time.Time) bool {
	for _, d := range days {
		if dayKey(d) == dayKey(target) {
			return true
		}
	}
	return false
}
