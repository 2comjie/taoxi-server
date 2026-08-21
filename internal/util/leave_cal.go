package util

import (
	"time"
)

const NoUpdateDt time.Duration = -1

type NextUpdate struct {
	Source string
	Dt     time.Duration
}

type OfflineResult struct {
	UpdateCount int
	SourceCount map[string]int
}

func UpdateAfter(source string, dt time.Duration) NextUpdate {
	return NextUpdate{Source: source, Dt: dt}
}

func MinNextUpdate(tick time.Duration, updates ...NextUpdate) NextUpdate {
	result := NextUpdate{Dt: NoUpdateDt}
	for _, update := range updates {
		if update.Dt < 0 {
			continue
		}
		if result.Dt < 0 || update.Dt < result.Dt {
			result = update
		}
	}
	if result.Dt >= 0 && result.Dt < tick {
		result.Dt = tick
	}
	return result
}

func RunOffline(totalDuration time.Duration, updateFunc func(time.Duration) NextUpdate) OfflineResult {
	if totalDuration <= 0 {
		return OfflineResult{}
	}
	remain := totalDuration
	result := OfflineResult{SourceCount: make(map[string]int)}
	next := updateFunc(0)

	for remain > 0 {
		delta := remain
		if next.Dt >= 0 && next.Dt < delta {
			delta = next.Dt
		}
		if delta == 0 {
			panic("offline update duration不能为0 source=" + next.Source)
		}

		if next.Dt >= 0 && next.Dt <= remain {
			result.SourceCount[next.Source]++
		}
		result.UpdateCount++
		remain -= delta
		next = updateFunc(delta)
	}
	return result
}
