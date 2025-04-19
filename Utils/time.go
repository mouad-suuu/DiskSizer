package Utils

import "time"

type Timer struct {
	start time.Time
}

func StartTimer() Timer {
	return Timer{start: time.Now()}
}

func (t Timer) Elapsed() float64 {
	return time.Since(t.start).Seconds()
}
