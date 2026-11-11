package worker

import (
	"testing"
	"time"
)

func TestNextMidnight(t *testing.T) {
	loc := time.Local
	cases := []struct {
		now  time.Time
		want time.Time
	}{
		{
			now:  time.Date(2026, 11, 11, 9, 30, 0, 0, loc),
			want: time.Date(2026, 11, 12, 0, 0, 0, 0, loc),
		},
		{
			now:  time.Date(2026, 11, 11, 23, 59, 59, 0, loc),
			want: time.Date(2026, 11, 12, 0, 0, 0, 0, loc),
		},
		{
			now:  time.Date(2026, 12, 31, 23, 0, 0, 0, loc),
			want: time.Date(2027, 1, 1, 0, 0, 0, 0, loc),
		},
	}
	for _, c := range cases {
		if got := NextMidnight(c.now); !got.Equal(c.want) {
			t.Errorf("NextMidnight(%s) = %s, want %s", c.now, got, c.want)
		}
	}
}
