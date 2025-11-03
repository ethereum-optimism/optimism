package forks

import "fmt"

// Name identifies a hardfork by name.
type Name string

const (
	Bedrock  Name = "bedrock"
	Regolith Name = "regolith"
	Canyon   Name = "canyon"
	Delta    Name = "delta"
	Ecotone  Name = "ecotone"
	Fjord    Name = "fjord"
	Granite  Name = "granite"
	Holocene Name = "holocene"
	Isthmus  Name = "isthmus"
	Jovian   Name = "jovian"
	Interop  Name = "interop"
	// ADD NEW MAINLINE FORKS TO [All] BELOW!

	// Optional Forks - not part of mainline
	PectraBlobSchedule Name = "pectrablobschedule"

	None Name = ""
)

// All lists all known mainline forks in chronological order.
var All = []Name{
	Bedrock,
	Regolith,
	Canyon,
	Delta,
	Ecotone,
	Fjord,
	Granite,
	Holocene,
	Isthmus,
	Jovian,
	Interop,
	// ADD NEW MAINLINE FORKS HERE!
}

// AllOpt lists all optional forks in chronological order.
var AllOpt = []Name{
	PectraBlobSchedule,
	// ADD NEW OPTIONAL FORKS HERE!
}

// Latest returns the most recent fork in [All].
var Latest = All[len(All)-1]

// From returns the list of forks starting from the provided fork, inclusive.
func From(start Name) []Name {
	for i, f := range All {
		if f == start {
			return All[i:]
		}
	}
	panic(fmt.Sprintf("invalid fork: %s", start))
}

var next = func() map[Name]Name {
	m := make(map[Name]Name, len(All))
	for i, f := range All {
		if i == len(All)-1 {
			m[f] = None
			break
		}
		m[f] = All[i+1]
	}
	return m
}()

// IsValid returns true if the provided fork is a known fork.
func IsValid(f Name) bool {
	_, ok := next[f]
	return ok
}

// Next returns the fork that follows the provided fork, or None if it is the last.
func Next(f Name) Name { return next[f] }
