package clock

import (
	"sync/atomic"
	"time"
)

type SimpleClock struct {
	unix atomic.Uint64
}

func NewSimpleClock() *SimpleClock {
	return &SimpleClock{}
}

func (c *SimpleClock) SetTime(u uint64) {
	c.unix.Store(u)
}

func (c *SimpleClock) Now() time.Time {
	return time.Unix(int64(c.unix.Load()), 0)
}
