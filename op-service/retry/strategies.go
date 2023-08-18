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
// function is min(e.Min + (2^attempt * 1000) + randBetween(0, e.MaxJitter), e.Max)
type ExponentialStrategy struct {
	// Min is the minimum amount of time to wait between attempts.
	Min time.Duration

	// Max is the maximum amount of time to wait between attempts.
	Max time.Duration

	// MaxJitter is the maximum amount of random jitter to insert between attempts.
	MaxJitter time.Duration
}

func (e *ExponentialStrategy) Duration(attempt int) time.Duration {
	var jitter time.Duration
	if e.MaxJitter > 0 {
		jitter = time.Duration(rand.Int63n(e.MaxJitter.Nanoseconds()))
	}
	dur := e.Min + time.Duration(int(math.Pow(2, float64(attempt))*1000))*time.Millisecond
	dur += jitter
	if dur > e.Max {
		return e.Max
	}

	return dur
}

func Exponential() Strategy {
	return &ExponentialStrategy{
		Max:       time.Duration(10000 * time.Millisecond),
		MaxJitter: time.Duration(250 * time.Millisecond),
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
