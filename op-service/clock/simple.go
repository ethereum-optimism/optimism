package clock

import (
	"time"
)

type SimpleClock interface {
	SetTime(time.Time)
	Now() time.Time
}

type simpleClock struct {
	time time.Time
}

func NewSimpleClock() *simpleClock {
	return &simpleClock{time: time.Now()}
}

func (c *simpleClock) SetTime(t time.Time) {
	c.time = t
}

func (c *simpleClock) Now() time.Time {
	return c.time
}
