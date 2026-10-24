package habits

import (
	"testing"
	"time"
)

func date(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("bad test date %q: %v", s, err)
	}
	return d
}

func TestCurrentStreak(t *testing.T) {
	today := date(t, "2026-10-24")

	tests := []struct {
		name string
		days []string
		want int
	}{
		{"no check-ins", nil, 0},
		{"only today", []string{"2026-10-24"}, 1},
		{"today and the two days before", []string{"2026-10-24", "2026-10-23", "2026-10-22"}, 3},
		{"today not done yet, yesterday keeps it alive", []string{"2026-10-23", "2026-10-22"}, 2},
		{"a gap breaks the streak", []string{"2026-10-24", "2026-10-22"}, 1},
		{"last check-in two days ago means streak is over", []string{"2026-10-22", "2026-10-21"}, 0},
		{"duplicates and any order", []string{"2026-10-22", "2026-10-24", "2026-10-23", "2026-10-24"}, 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var days []time.Time
			for _, s := range tc.days {
				days = append(days, date(t, s))
			}
			if got := CurrentStreak(days, today); got != tc.want {
				t.Errorf("CurrentStreak = %d, want %d", got, tc.want)
			}
		})
	}
}
