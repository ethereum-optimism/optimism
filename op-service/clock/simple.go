package clock

import (
	"sync"
	"time"
)

type SimpleClock interface {
	SetTime(time.Time)
	Now() time.Time
}

type simpleClock struct {
	mu   sync.Mutex
	time time.Time
}

func NewSimpleClock() *simpleClock {
	return &simpleClock{time: time.Now()}
}

func (c *simpleClock) SetTime(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.time = t
}

func (c *simpleClock) Now() time.Time {
	return c.time
}
