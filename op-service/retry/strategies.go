package retry

import (
	"math"
	"math/rand"
	"time"
)

// Strategy is used to calculate how long a particular Operation
// should wait between attempts.
type Strategy interface {
	// Duration returns how long to wait for a given retry attempt.
	Duration(attempt int) time.Duration
}

// ExponentialStrategy performs exponential backoff. The exponential backoff
// function is min(e.Min + (2^attempt * second), e.Max) + randBetween(0, e.MaxJitter)
type ExponentialStrategy struct {
	// Min is the minimum amount of time to wait between attempts.
	Min time.Duration

	// Max is the maximum amount of time to wait between attempts.
	Max time.Duration

	// MaxJitter is the maximum amount of random jitter to insert between attempts.
	// Jitter is added on top of the maximum, if the maximum is reached.
	MaxJitter time.Duration
}

func (e *ExponentialStrategy) Duration(attempt int) time.Duration {
	var jitter time.Duration // non-negative jitter
	if e.MaxJitter > 0 {
		jitter = time.Duration(rand.Int63n(e.MaxJitter.Nanoseconds()))
	}
	if attempt < 0 {
		return e.Min + jitter
	}
	durFloat := float64(e.Min)
	durFloat += math.Pow(2, float64(attempt)) * float64(time.Second)
	dur := time.Duration(durFloat)
	if durFloat > float64(e.Max) {
		dur = e.Max
	}
	dur += jitter

	return dur
}

func Exponential() Strategy {
	return &ExponentialStrategy{
		Min:       0,
		Max:       10 * time.Second,
		MaxJitter: 250 * time.Millisecond,
	}
}

type FixedStrategy struct {
	Dur time.Duration
}

func (f *FixedStrategy) Duration(attempt int) time.Duration {
	return f.Dur
}

func Fixed(dur time.Duration) Strategy {
	return &FixedStrategy{
		Dur: dur,
	}
}
