package backoff

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
	// Min is the minimum amount of time to wait between attempts in ms.
	Min float64

	// Max is the maximum amount of time to wait between attempts in ms.
	Max float64

	// MaxJitter is the maximum amount of random jitter to insert between
	// attempts in ms.
	MaxJitter int
}

func (e *ExponentialStrategy) Duration(attempt int) time.Duration {
	var jitter int
	if e.MaxJitter > 0 {
		jitter = rand.Intn(e.MaxJitter)
	}
	dur := e.Min + (math.Pow(2, float64(attempt)) * 1000)
	dur += float64(jitter)
	if dur > e.Max {
		return time.Millisecond * time.Duration(e.Max)
	}

	return time.Millisecond * time.Duration(dur)
}

func Exponential() Strategy {
	return &ExponentialStrategy{
		Max:       10000,
		MaxJitter: 250,
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
