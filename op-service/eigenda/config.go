package eigenda

import (
	"fmt"
	"math"
	"time"

	"github.com/urfave/cli/v2"
)

type Config struct {
	// TODO(eigenlayer): Update quorum ID command-line parameters to support passing
	// and arbitrary number of quorum IDs.

	// DaRpc is the HTTP provider URL for the Data Availability node.
	RPC string

	// The total amount of time that the batcher will spend waiting for EigenDA to confirm a blob
	StatusQueryTimeout time.Duration

	// The amount of time to wait between status queries of a newly dispersed blob
	StatusQueryRetryInterval time.Duration
}

// We add this because the urfave/cli library doesn't support uint32 specifically
func Uint32(ctx *cli.Context, flagName string) uint32 {
	daQuorumIDLong := ctx.Uint64(flagName)
	daQuorumID, success := SafeConvertUInt64ToUInt32(daQuorumIDLong)
	if !success {
		panic(fmt.Errorf("%s must be in the uint32 range", flagName))
	}
	return daQuorumID
}

func SafeConvertUInt64ToUInt32(val uint64) (uint32, bool) {
	if val <= math.MaxUint32 {
		return uint32(val), true
	}
	return 0, false
}
