package timeutil_test

import (
	"testing"
	"time"

	"github.com/vortex-fintech/go-lib/foundation/timeutil"
)

func TestFirstDayOfNextMonthUTC(t *testing.T) {
	t.Run("regular month", func(t *testing.T) {
		got := timeutil.FirstDayOfNextMonthUTC(time.Date(2026, time.February, 9, 10, 30, 0, 0, time.UTC))
		want := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("got=%v, want=%v", got, want)
		}
	})

	t.Run("december rollover", func(t *testing.T) {
		got := timeutil.FirstDayOfNextMonthUTC(time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC))
		want := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("got=%v, want=%v", got, want)
		}
	})

	t.Run("non utc input", func(t *testing.T) {
		loc := time.FixedZone("UTC+3", 3*60*60)
		got := timeutil.FirstDayOfNextMonthUTC(time.Date(2026, time.June, 15, 10, 0, 0, 0, loc))
		want := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Fatalf("got=%v, want=%v", got, want)
		}
	})
}

func TestIsNotFutureUTC(t *testing.T) {
	now := time.Date(2026, time.February, 9, 12, 0, 0, 0, time.UTC)

	if timeutil.IsNotFutureUTC(time.Time{}, now) {
		t.Fatal("expected false for zero now")
	}
	if timeutil.IsNotFutureUTC(now, time.Time{}) {
		t.Fatal("expected false for zero at")
	}

	if !timeutil.IsNotFutureUTC(now, now) {
		t.Fatal("expected true for equal timestamps")
	}
	if !timeutil.IsNotFutureUTC(now, now.Add(-time.Second)) {
		t.Fatal("expected true for past timestamp")
	}
	if timeutil.IsNotFutureUTC(now, now.Add(time.Second)) {
		t.Fatal("expected false for future timestamp")
	}

	loc := time.FixedZone("UTC+3", 3*60*60)
	nowLocal := time.Date(2026, time.February, 9, 15, 0, 0, 0, loc) // == 12:00 UTC
	atLocal := time.Date(2026, time.February, 9, 15, 0, 0, 0, loc)
	if !timeutil.IsNotFutureUTC(nowLocal, atLocal) {
		t.Fatal("expected true for equal instants across location")
	}
}
