package finalize

import (
	"context"
	"math/big"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// TestFinalize tests if L2 finalizes after sufficient time after L1 finalizes
func TestFinalize(t *testing.T) {
	if !op_e2e.VerboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := op_e2e.DefaultSystemConfig(t)

	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]

	// as configured in the extra geth lifecycle in testing setup
	finalizedDistance := uint64(8)
	// Wait enough time for L1 to finalize and L2 to confirm its data in finalized L1 blocks
	<-time.After(time.Duration((finalizedDistance+4)*cfg.L1BlockTime) * time.Second)

	// fetch the finalizes head of geth
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	l2Finalized, err := l2Seq.BlockByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	require.NoError(t, err)

	require.NotZerof(t, l2Finalized.NumberU64(), "must have finalized L2 block")
}
