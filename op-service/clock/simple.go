package clock

import (
	"math"
	"sync/atomic"
	"time"
)

type SimpleClock struct {
	unix atomic.Uint64
}

func NewSimpleClock() *SimpleClock {
	return &SimpleClock{}
}

func (s *SimpleClock) Add(d time.Duration) time.Time {
	now := s.Now()
	if d < 0 && math.Abs(d.Seconds()) > float64(now.Unix()) {
		return time.Unix(0, 0)
	}
	return now.Add(d)
}

func (s *SimpleClock) SetTime(u uint64) {
	s.unix.Store(u)
}

func (s *SimpleClock) Now() time.Time {
	return time.Unix(int64(s.unix.Load()), 0)
}
