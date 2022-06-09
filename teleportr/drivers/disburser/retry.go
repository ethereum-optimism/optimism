package disburser

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/cenkalti/backoff"
)

var retryRegexes = []*regexp.Regexp{
	regexp.MustCompile("read: connection reset by peer$"),
}

var DefaultBackoff = &backoff.ExponentialBackOff{
	InitialInterval:     backoff.DefaultInitialInterval,
	RandomizationFactor: backoff.DefaultRandomizationFactor,
	Multiplier:          backoff.DefaultMultiplier,
	MaxInterval:         10 * time.Second,
	MaxElapsedTime:      time.Minute,
	Clock:               backoff.SystemClock,
}

func IsRetryableError(err error) bool {
	msg := err.Error()

	if httpErr, ok := err.(rpc.HTTPError); ok {
		if httpErr.StatusCode == 503 || httpErr.StatusCode == 524 || httpErr.StatusCode == 429 {
			return true
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	for _, reg := range retryRegexes {
		if reg.MatchString(msg) {
			return true
		}
	}

	return false
}
