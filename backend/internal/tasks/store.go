package tasks

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when no task has the requested id.
var ErrNotFound = errors.New("task not found")

// Store keeps tasks in PostgreSQL. It is the only place that knows how
// tasks are stored, so the handler code barely changes.
type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) List(ctx context.Context) ([]Task, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, title, meta, priority, done FROM tasks ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Meta, &t.Priority, &t.Done); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	return tasks, nil
}

func (s *Store) Add(ctx context.Context, title, meta, priority string) (Task, error) {
	var t Task
	err := s.db.QueryRow(ctx,
		`INSERT INTO tasks (title, meta, priority) VALUES ($1, $2, $3)
         RETURNING id, title, meta, priority, done`,
		title, meta, priority,
	).Scan(&t.ID, &t.Title, &t.Meta, &t.Priority, &t.Done)
	if err != nil {
		return Task{}, fmt.Errorf("add task: %w", err)
	}
	return t, nil
}

// SetDone returns ErrNotFound if no task has that id.
func (s *Store) SetDone(ctx context.Context, id int, done bool) (Task, error) {
	var t Task
	err := s.db.QueryRow(ctx,
		`UPDATE tasks SET done = $2 WHERE id = $1
         RETURNING id, title, meta, priority, done`,
		id, done,
	).Scan(&t.ID, &t.Title, &t.Meta, &t.Priority, &t.Done)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	if err != nil {
		return Task{}, fmt.Errorf("set done: %w", err)
	}
	return t, nil
}

// Delete returns ErrNotFound if no task has that id.
func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
