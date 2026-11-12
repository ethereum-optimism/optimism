package p2p

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-e2e/system/helpers"
)

// TestTxForwarding checks that a transaction sent to the verifier reaches the sequencer and is
// included in a block. The two ELs are not peered, so the verifier's sequencer endpoint is the
// only path.
func TestTxForwarding(t *testing.T) {
	op_e2e.InitParallel(t)
	cfg := e2esys.DefaultSystemConfig(t)
	sys, err := cfg.Start(t)
	require.NoError(t, err, "Start system")

	// This prevents the below tx-sending from flaking in CI
	_, err = geth.WaitForBlock(big.NewInt(10), sys.NodeClient("verifier"))
	require.NoError(t, err)

	helpers.SendL2Tx(t, cfg, sys.NodeSourceClient("verifier"), cfg.Secrets.Alice, func(opts *helpers.TxOpts) {
		opts.ToAddr = &common.Address{0xaa}
		opts.Value = common.Big1
		opts.VerifyOnClients(sys.NodeSourceClient("sequencer"), sys.NodeSourceClient("verifier"))
	})
}
