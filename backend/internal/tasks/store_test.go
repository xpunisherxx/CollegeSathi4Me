package tasks

import "testing"

func TestAddGivesUniqueIDs(t *testing.T) {
	s := NewStore()
	a := s.Add("one", "m", "high")
	b := s.Add("two", "m", "medium")

	if a.ID == b.ID {
		t.Fatalf("ids must be unique, got %d twice", a.ID)
	}
	if got := len(s.List()); got != 2 {
		t.Fatalf("want 2 tasks, got %d", got)
	}
}

func TestSetDone(t *testing.T) {
	s := NewStore()
	task := s.Add("one", "m", "high")

	updated, found := s.SetDone(task.ID, true)
	if !found || !updated.Done {
		t.Fatalf("expected task to be found and done, got found=%v done=%v", found, updated.Done)
	}
	if _, found := s.SetDone(999, true); found {
		t.Fatal("unknown id should not be found")
	}
}

func TestDelete(t *testing.T) {
	s := NewStore()
	task := s.Add("one", "m", "high")

	if !s.Delete(task.ID) {
		t.Fatal("first delete should succeed")
	}
	if s.Delete(task.ID) {
		t.Fatal("deleting twice should report not found")
	}
	if got := len(s.List()); got != 0 {
		t.Fatalf("want 0 tasks, got %d", got)
	}
}
