package tasks

import "sync"

// Store keeps tasks in memory. It is the only place that knows how tasks
// are stored: in Phase 4 you replace its insides with SQLite and the HTTP
// handler code does not change.
//
// The mutex matters: Go serves each request in its own goroutine, so two
// requests can touch the slice at the same time.
type Store struct {
	mu     sync.Mutex
	tasks  []Task
	nextID int
}

func NewStore() *Store {
	return &Store{nextID: 1}
}

// List returns a copy so callers cannot modify the store's data.
func (s *Store) List() []Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Task, len(s.tasks))
	copy(out, s.tasks)
	return out
}

func (s *Store) Add(title, meta, priority string) Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := Task{ID: s.nextID, Title: title, Meta: meta, Priority: priority}
	s.nextID++
	s.tasks = append(s.tasks, t)
	return t
}

// SetDone reports false if no task has that id.
func (s *Store) SetDone(id int, done bool) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.tasks {
		if s.tasks[i].ID == id {
			s.tasks[i].Done = done
			return s.tasks[i], true
		}
	}
	return Task{}, false
}

// Delete reports false if no task has that id.
func (s *Store) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.tasks {
		if t.ID == id {
			s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
			return true
		}
	}
	return false
}
