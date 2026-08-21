package util

import (
	"slices"
	"testing"
	"time"
)

func TestMinNextUpdateUsesTickAsMinimum(t *testing.T) {
	next := MinNextUpdate(time.Second,
		UpdateAfter("building", time.Hour),
		UpdateAfter("farm", 10*time.Millisecond),
	)
	if next.Source != "farm" || next.Dt != time.Second {
		t.Fatalf("next=%+v", next)
	}
}

func TestRunOffline(t *testing.T) {
	updates := []NextUpdate{
		UpdateAfter("building", time.Hour),
		UpdateAfter("farm", 2*time.Hour),
		UpdateAfter("", NoUpdateDt),
	}
	deltas := make([]time.Duration, 0, 4)

	result := RunOffline(10*time.Hour, func(delta time.Duration) NextUpdate {
		deltas = append(deltas, delta)
		if len(updates) == 0 {
			return NextUpdate{Dt: NoUpdateDt}
		}
		next := updates[0]
		updates = updates[1:]
		return next
	})

	expected := []time.Duration{0, time.Hour, 2 * time.Hour, 7 * time.Hour}
	if !slices.Equal(deltas, expected) {
		t.Fatalf("deltas=%v", deltas)
	}
	if result.UpdateCount != 3 || result.SourceCount["building"] != 1 || result.SourceCount["farm"] != 1 {
		t.Fatalf("result=%+v", result)
	}
}
