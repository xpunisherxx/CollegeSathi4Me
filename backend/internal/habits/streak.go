package habits

import "time"

// CurrentStreak counts consecutive checked-in days ending today. If today
// is not checked in yet, the streak is still alive when yesterday was, so
// it counts back from yesterday. days may be in any order and may contain
// duplicates; only the calendar date matters.
func CurrentStreak(days []time.Time, today time.Time) int {
	checked := make(map[string]bool, len(days))
	for _, d := range days {
		checked[dayKey(d)] = true
	}

	cursor := today
	if !checked[dayKey(cursor)] {
		cursor = cursor.AddDate(0, 0, -1)
	}

	streak := 0
	for checked[dayKey(cursor)] {
		streak++
		cursor = cursor.AddDate(0, 0, -1)
	}
	return streak
}

func dayKey(t time.Time) string {
	return t.Format("2006-01-02")
}
