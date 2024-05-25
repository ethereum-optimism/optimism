package node

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

	"github.com/ethereum/go-ethereum/params"
)

func TestHaltMaybe(t *testing.T) {
	haltTest := func(opt string, halts ...params.ProtocolVersionComparison) {
		t.Run(opt, func(t *testing.T) {
			for _, h := range []params.ProtocolVersionComparison{
				params.AheadMajor,
				params.OutdatedMajor,
				params.AheadMinor,
				params.OutdatedMinor,
				params.AheadPatch,
				params.OutdatedPatch,
				params.AheadPrerelease,
				params.OutdatedPrerelease,
				params.Matching,
				params.DiffVersionType,
				params.DiffBuild,
				params.EmptyVersion,
			} {
				expectedHalt := slices.Contains(halts, h)
				gotHalt := haltMaybe(opt, h)
				require.Equal(t, expectedHalt, gotHalt, "%s %d", opt, h)
			}
		})
	}
	haltTest("")
	haltTest("major", params.OutdatedMajor)
	haltTest("minor", params.OutdatedMajor, params.OutdatedMinor)
	haltTest("patch", params.OutdatedMajor, params.OutdatedMinor, params.OutdatedPatch)
}
