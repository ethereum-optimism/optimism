package clock

import (
	"sync/atomic"
	"time"
)

type SimpleClock interface {
	SetTime(uint64)
	Now() time.Time
}

type simpleClock struct {
	unix atomic.Uint64
}

func NewSimpleClock() *simpleClock {
	return &simpleClock{}
}

func (c *simpleClock) SetTime(u uint64) {
	c.unix.Store(u)
}

func (c *simpleClock) Now() time.Time {
	return time.Unix(int64(c.unix.Load()), 0)
}
