package main

import (
	"errors"
	"strings"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
)

func removeArg(args []string, remove string) []string {
	filter := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := strings.Split(args[i], "=")
		if a[0] == remove {
			// don't advance if the arg value is in the same arg
			if !strings.Contains(args[i], "=") && len(a) > 0 {
				i++
			}
		} else {
			filter = append(filter, args[i])
		}
	}
	return filter
}

func parseVersionFlag(ver string) (versions.StateVersion, error) {
	switch ver {
	case "single":
		return versions.VersionSingleThreaded, nil
	case "multi":
		return versions.VersionMultiThreaded, nil
	default:
		return versions.StateVersion(0), errors.New("unknown state version")
	}
}
