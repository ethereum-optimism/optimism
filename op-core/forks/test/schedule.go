// Package test provides fork schedule helpers for tests of fork-aware configs.
package test

import (
	"slices"
	"strings"

	"github.com/ethereum-optimism/optimism/op-core/forks"
)

// TimestampELForks lists the forks of [forks.AllEL] that activate by timestamp, in
// chronological order: all but the first, Bedrock, which activates by block number.
var TimestampELForks = forks.AllEL[1:]

// Schedule gives the activation time of each fork in [TimestampELForks]; nil leaves a
// fork unscheduled.
type Schedule func(forks.Name) *uint64

// Staggered activates the i-th fork of [TimestampELForks] at 1000*(i+1).
func Staggered() Schedule {
	return func(fork forks.Name) *uint64 {
		t := uint64(1000 * (slices.Index(TimestampELForks, fork) + 1))
		return &t
	}
}

// AtGenesisThrough activates every fork of [TimestampELForks] up to and including last at
// genesis, and leaves the later ones unscheduled.
func AtGenesisThrough(last forks.Name) Schedule {
	return func(fork forks.Name) *uint64 {
		if slices.Index(TimestampELForks, fork) > slices.Index(TimestampELForks, last) {
			return nil
		}
		return new(uint64)
	}
}

// Override schedules fork at t, and every other fork as in base.
func Override(base Schedule, fork forks.Name, t *uint64) Schedule {
	return func(f forks.Name) *uint64 {
		if f == fork {
			return t
		}
		return base(f)
	}
}

// Title is the fork name with its first letter upper-cased, as it appears in config
// field and method names such as <Fork>Time and Is<Fork>.
func Title(fork forks.Name) string {
	return strings.ToUpper(string(fork[:1])) + string(fork[1:])
}
