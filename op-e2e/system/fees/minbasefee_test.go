package fees

import (
	"context"
	"math/big"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	legacybindings "github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip1559"
	"github.com/stretchr/testify/require"
)

// TestMinBaseFeeFactors checks that we can successfully change minBaseFeeFactors parameter via SystemConfig
// with the Jovian upgrade and that it's properly encoded in block extra data.
func TestMinBaseFeeFactors(t *testing.T) {
	op_e2e.InitParallel(t)

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// Create our system configuration for L1/L2 and start it
	cfg := e2esys.JovianSystemConfig(t, new(hexutil.Uint64))
	cfg.DeployConfig.L2GenesisBlockBaseFeePerGas = (*hexutil.Big)(big.NewInt(100_000_000))
	sys, err := cfg.Start(t)
	require.NoError(t, err, "Error starting up system")

	// Obtain our sequencer, verifier, and transactor keypair.
	l1Client := sys.NodeClient("l1")
	l2Seq := sys.NodeClient("sequencer")
	ethPrivKey := cfg.Secrets.Deployer

	_, err = l2Seq.HeaderByNumber(context.Background(), big.NewInt(0))
	require.NoError(t, err)

	// Bind to the SystemConfig contract
	sysconfig, err := legacybindings.NewSystemConfig(cfg.L1Deployments.SystemConfigProxy, l1Client)
	require.NoError(t, err)

	// Obtain our signer.
	opts, err := bind.NewKeyedTransactorWithChainID(ethPrivKey, cfg.L1ChainIDBig())
	require.NoError(t, err)

	// Define our L1 transaction timeout duration.
	txTimeoutDuration := 10 * time.Duration(cfg.DeployConfig.L1BlockTime) * time.Second

	var cancel context.CancelFunc

	// Confirm minBaseFeeSignificand and minBaseFeeExponent are initialized to 0
	minBaseFeeSignificand, err := sysconfig.MinBaseFeeSignificand(&bind.CallOpts{})
	require.NoError(t, err, "reading minBaseFeeSignificand")
	require.Equal(t, uint8(0), minBaseFeeSignificand)
	minBaseFeeExponent, err := sysconfig.MinBaseFeeExponent(&bind.CallOpts{})
	require.NoError(t, err, "reading minBaseFeeExponent")
	require.Equal(t, uint8(0), minBaseFeeExponent)

	// Set both EIP-1559 parameters and minBaseFeeFactors in a single transaction sequence
	// This matches how they would be used in production
	expectedDenom := uint32(10)
	expectedElasticity := uint32(2)
	// Set the minimum base fee to 1 gwei.
	expectedMinBaseFeeSignificand := uint8(1)
	expectedMinBaseFeeExponent := uint8(9)
	expectedMinBaseFeeFactors := eip1559.EncodeMinBaseFeeFactors(expectedMinBaseFeeSignificand, expectedMinBaseFeeExponent)
	expectedMinBaseFee := big.NewInt(1_000_000_000)

	// Set EIP-1559 parameters first
	opts.Context, cancel = context.WithTimeout(ctx, txTimeoutDuration)
	tx, err := sysconfig.SetEIP1559Params(opts, expectedDenom, expectedElasticity)
	cancel()
	require.NoError(t, err, "SetEIP1559Params update tx")

	_, err = wait.ForReceiptOK(ctx, l1Client, tx.Hash())
	require.NoError(t, err, "Waiting for sysconfig set EIP1559Params update tx")

	// Then set MinBaseFeeFactors
	opts.Context, cancel = context.WithTimeout(ctx, txTimeoutDuration)
	tx, err = sysconfig.SetMinBaseFee(opts, expectedMinBaseFeeSignificand, expectedMinBaseFeeExponent)
	cancel()
	require.NoError(t, err, "SetMinBaseFee update tx")

	receipt, err := wait.ForReceiptOK(ctx, l1Client, tx.Hash())
	require.NoError(t, err, "Waiting for sysconfig set minBaseFee update tx")

	minBaseFeeSignificand, err = sysconfig.MinBaseFeeSignificand(&bind.CallOpts{})
	require.NoError(t, err, "reading minBaseFeeSignificand")
	require.Equal(t, expectedMinBaseFeeSignificand, minBaseFeeSignificand)
	minBaseFeeExponent, err = sysconfig.MinBaseFeeExponent(&bind.CallOpts{})
	require.NoError(t, err, "reading minBaseFeeExponent")
	require.Equal(t, expectedMinBaseFeeExponent, minBaseFeeExponent)

	_, err = geth.WaitForL1OriginOnL2(sys.RollupConfig, receipt.BlockNumber.Uint64(), l2Seq, txTimeoutDuration)
	require.NoError(t, err, "waiting for L2 block to include the sysconfig update")

	h, err := l2Seq.HeaderByNumber(context.Background(), nil)
	require.NoError(t, err)

	// Debug: print the actual ExtraData
	t.Logf("Actual ExtraData: %x", h.Extra)
	t.Logf("Expected MinBaseFee: %d * 10^%d = %d", expectedMinBaseFeeSignificand, expectedMinBaseFeeExponent, expectedMinBaseFee)

	// Decode and check what we actually got
	if len(h.Extra) == 10 {
		actualDenom, actualElasticity, actualMinBaseFeeSignificand, actualMinBaseFeeExponent := eip1559.DecodeMinBaseFeeExtraData(h.Extra)
		t.Logf("Decoded - Denom: %d, Elasticity: %d, MinBaseFee: %d * 10^%d", actualDenom, actualElasticity, actualMinBaseFeeSignificand, actualMinBaseFeeExponent)
	}

	// Confirm the extraData is being set as expected with Jovian encoding
	expectedExtraData := eip1559.EncodeMinBaseFeeExtraData(uint64(expectedDenom), uint64(expectedElasticity), expectedMinBaseFeeFactors)
	require.Equal(t, expectedExtraData, h.Extra, "Extra data should match Jovian encoding with minBaseFeeFactors")

	// Verify the minimum base fee is enforced
	require.True(t, h.BaseFee.Cmp(expectedMinBaseFee) >= 0,
		"Current base fee (%s) should be >= minimum base fee (%s)",
		h.BaseFee.String(), expectedMinBaseFee.String())

	// Wait for the next block to confirm the constraint is maintained
	b, err := geth.WaitForBlock(big.NewInt(h.Number.Int64()+1), l2Seq)
	require.NoError(t, err, "waiting for next L2 block")

	// Confirm the extraData is still being set as expected in the next block
	require.Equal(t, expectedExtraData, b.Header().Extra, "Extra data should still match Jovian encoding with minBaseFeeFactors")

	// Verify the minimum base fee constraint is still enforced
	require.True(t, b.Header().BaseFee.Cmp(expectedMinBaseFee) >= 0,
		"Next block base fee (%s) should be >= minimum base fee (%s)",
		b.Header().BaseFee.String(), expectedMinBaseFee.String())
}
