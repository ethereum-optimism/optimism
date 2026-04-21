package proofs

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// TestKarstPredeployImplementationsUpgraded verifies that Karst activation
// replaces the implementation address stored in the EIP-1967 slot of
// representative predeploy proxies. It is a smoke test that the bundle's
// upgrade transactions actually rewrote proxy implementation pointers — not
// a check of what the new implementations do (that's the job of the per-
// contract test suites).
//
// The invariant is Karst-specific: every "Deploy X Implementation" intent in
// the Karst bundle lands at a fresh address, and L2ProxyAdmin's batch upgrade
// (bundle intent 31) repoints predeploy proxies to those fresh addresses. A
// future fork that doesn't upgrade proxies wouldn't satisfy this, which is
// why the assertion lives in a Karst-specific test rather than in the generic
// NUT bundle activation test.
func TestKarstPredeployImplementationsUpgraded(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	matrix.AddDefaultTestCases(
		nil,
		helpers.NewForkMatrix(helpers.Jovian),
		testKarstPredeployImplementationsUpgraded,
	)
	matrix.Run(gt)
}

func testKarstPredeployImplementationsUpgraded(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)

	offset := uint64(4)
	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		dc.SetForkTimeOffset(forks.Karst, &offset)
	}
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testSetup)

	env.Miner.ActEmptyBlock(t)
	env.Sequencer.ActL1HeadSignal(t)
	for i := 0; i < int(offset); i++ {
		env.Sequencer.ActL2EmptyBlock(t)
	}

	engine := env.Engine
	actHeader := engine.L2Chain().CurrentHeader()
	require.Equal(t, forks.Karst,
		env.Sd.RollupCfg.IsActivationBlock(actHeader.Time-env.Sd.RollupCfg.BlockTime, actHeader.Time),
		"expected Karst activation block at time %d", actHeader.Time)

	postBlock := actHeader.Number
	preBlock := new(big.Int).Sub(postBlock, big.NewInt(1))

	ethCl := engine.EthClient()

	// Representative predeploys whose implementation Karst replaces. L1Block and
	// GasPriceOracle mirror the proxies asserted by earlier fork tests (ecotone,
	// isthmus); covering them keeps vocabulary consistent across fork tests.
	proxies := []struct {
		name string
		addr common.Address
	}{
		{"L1Block", predeploys.L1BlockAddr},
		{"GasPriceOracle", predeploys.GasPriceOracleAddr},
	}
	for _, p := range proxies {
		preImpl, err := ethCl.StorageAt(context.Background(), p.addr, genesis.ImplementationSlot, preBlock)
		require.NoError(t, err, "read %s impl slot pre-activation", p.name)
		postImpl, err := ethCl.StorageAt(context.Background(), p.addr, genesis.ImplementationSlot, postBlock)
		require.NoError(t, err, "read %s impl slot post-activation", p.name)

		require.NotEqualf(t, preImpl, postImpl,
			"%s (%s) implementation slot must change across Karst activation", p.name, p.addr)

		// New impl must actually be deployed — guards against an upgrade that
		// repoints a proxy at an empty address.
		newImplAddr := common.BytesToAddress(postImpl)
		code, err := ethCl.CodeAt(context.Background(), newImplAddr, postBlock)
		require.NoError(t, err, "read code at new %s impl", p.name)
		require.NotEmptyf(t, code, "new %s impl %s must have code", p.name, newImplAddr)
	}
}
