package verifier

import (
	"math/big"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/stretchr/testify/require"
)

// TestFinalize tests if L2 finalizes after sufficient time after L1 finalizes
func TestFinalize(t *testing.T) {
	op_e2e.InitParallel(t)

	cfg := e2esys.DefaultSystemConfig(t)

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")

	l2Seq := sys.NodeClient("sequencer")

	l2Finalized, err := geth.WaitForBlockToBeFinalized(big.NewInt(12), l2Seq, 1*time.Minute)
	require.NoError(t, err, "must be able to fetch a finalized L2 block")
	require.NotZerof(t, l2Finalized.NumberU64(), "must have finalized L2 block")
}
