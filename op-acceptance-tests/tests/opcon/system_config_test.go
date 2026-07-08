package opcon

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
)

// These tests exercise op-con-node's handling of an L1 SystemConfig change — the
// consensus-critical path where operator config on L1 must flow into the L2 chain.
// op-con-node parses the SystemConfig `ConfigUpdate` log in
// src/sql/system_config/module.sql and resolves `effective_system_config` per L1
// block; a verifier feeds that config into the payload attributes it derives, and
// a sequencer feeds it into the blocks it builds. We change the L2 gas limit
// (`setGasLimit`, ConfigUpdate type 2) because it is directly observable on every
// L2 block header, and — per the OP spec — it takes effect instantly on the L2
// block that adopts the L1 origin carrying the change (no 1/1024 parent-delta
// bound), so a stale or mis-derived config surfaces as a wrong block gas limit.

// setSystemConfigGasLimit sends SystemConfig.setGasLimit(newGasLimit) on L1 signed
// by the SystemConfig owner, waits for it to be included, and returns the L1 block
// number the change landed in. Mirrors the operator-fee DSL's owner-signed write.
func setSystemConfigGasLimit(t devtest.T, l2Chain *dsl.L2Network, l1EL *dsl.L1ELNode, funderL1 *dsl.Funder, newGasLimit uint64) uint64 {
	systemConfig := bindings.NewBindings[bindings.SystemConfig](
		bindings.WithClient(l1EL.EthClient()),
		bindings.WithTo(l2Chain.Escape().Deployment().SystemConfigProxyAddr()),
		bindings.WithTest(t))

	ownerKey := devkeys.SystemConfigOwner.Key(l2Chain.ChainID().ToBig())
	owner := dsl.NewKey(t, l2Chain.Escape().Keys().Secret(ownerKey)).User(l1EL)
	funderL1.FundAtLeast(owner, eth.OneTenthEther)

	receipt, err := contractio.Write(systemConfig.SetGasLimit(newGasLimit), t.Ctx(), owner.Plan())
	t.Require().NoError(err, "SystemConfig.setGasLimit write failed")
	t.Require().Equal(gethtypes.ReceiptStatusSuccessful, receipt.Status, "SystemConfig.setGasLimit reverted")
	changeL1Block := receipt.BlockNumber.Uint64()
	t.Logger().Info("changed L1 SystemConfig gas limit", "newGasLimit", newGasLimit, "l1Block", changeL1Block)
	return changeL1Block
}

// waitForGasLimitAtOriginPast waits for the given L2 EL's head (at safety `label`)
// to advance to a block whose L1 origin is past changeL1Block, and returns that
// block's gas limit. A chain that never adopts the new origin — or a build/derive
// path that wedged on the config change — times out here.
func waitForGasLimitAtOriginPast(t devtest.T, el *dsl.L2ELNode, label eth.BlockLabel, changeL1Block uint64) uint64 {
	require := t.Require()
	msg := fmt.Sprintf("L2 %s head did not advance to an L1 origin past the SystemConfig gas-limit change", label)
	var gasLimit uint64
	require.Eventually(func() bool {
		head := el.BlockRefByLabel(label)
		if head.L1Origin.Number <= changeL1Block {
			return false
		}
		info, err := el.EthClient().InfoByNumber(t.Ctx(), head.Number)
		if err != nil {
			return false
		}
		gasLimit = info.GasLimit()
		return true
	}, 150*time.Second, 2*time.Second, msg)
	return gasLimit
}

// TestOpConSequencerAppliesGasLimitUpdate is the sequencer-slot path: op-con-node
// runs AS the L2 sequencer, so it must apply an L1 SystemConfig gas-limit change to
// the blocks it BUILDS. It reads `effective_system_config.gas_limit` at each block's
// L1 origin and sets the built block's gas limit accordingly; a sequencer that kept
// building from the genesis/stale config would produce blocks with the old gas
// limit (and op-node-compatible derivation of those blocks would then diverge).
//
// With the default op-node CL kind L2CLOpConSequencer() is a no-op, so this
// degrades to an ordinary op-node sequencer gas-limit-change check — a valid suite
// addition rather than an op-con-only test.
//
//	DEVSTACK_L2CL_KIND=op-con-node DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConSequencerAppliesGasLimitUpdate ./op-acceptance-tests/tests/opcon/
func TestOpConSequencerAppliesGasLimitUpdate(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalNoFaultProofs(t, opConSequencerOpt())
	require := t.Require()

	// The sequencer is live and producing its unsafe chain.
	sys.L2CL.Advanced(types.LocalUnsafe, 2, 60)

	// The genesis gas limit, read from an early block the sequencer produced.
	oldInfo, err := sys.L2EL.EthClient().InfoByNumber(t.Ctx(), 1)
	require.NoError(err)
	oldGasLimit := oldInfo.GasLimit()
	newGasLimit := oldGasLimit + 2_000_000

	// Change the L2 gas limit on the L1 SystemConfig.
	changeL1Block := setSystemConfigGasLimit(t, sys.L2Chain, sys.L1EL, sys.FunderL1, newGasLimit)

	// The sequencer must build blocks with the new gas limit once its L1 origin
	// passes the change. A sequencer ignoring the config change would keep emitting
	// oldGasLimit blocks and this assertion would fail.
	got := waitForGasLimitAtOriginPast(t, sys.L2EL, eth.Unsafe, changeL1Block)
	require.Equal(newGasLimit, got,
		"sequencer did not apply the new SystemConfig gas limit to the blocks it produced")
}

// TestOpConVerifierDerivesGasLimitUpdate is the derivation path: an op-con-node
// VERIFIER must derive an L1 SystemConfig gas-limit change (produced by the op-node
// sequencer) onto its safe chain. If op-con-node's SystemConfig derivation computed
// the wrong effective gas limit, the block it drives its engine to build would not
// match the op-node sequencer's, and its safe head would stall or diverge at the
// change.
//
// With the default op-node CL kind both slots are op-node, so this degrades to an
// ordinary op-node gas-limit-change derivation check — a valid suite addition.
//
//	DEVSTACK_L2CL_KIND=op-con-node DEVSTACK_L2EL_KIND=op-reth \
//	RUST_BINARY_PATH_OP_CON_NODE=<opql>/target/debug/op-con-node \
//	go test -run TestOpConVerifierDerivesGasLimitUpdate ./op-acceptance-tests/tests/opcon/
func TestOpConVerifierDerivesGasLimitUpdate(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNodeNoFaultProofsWithoutP2PWithoutCheck(t)
	require := t.Require()

	// The op-node sequencer produces + batches; the op-con-node verifier derives the
	// same safe chain from L1. Both advance their safe heads.
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, 60),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 60),
	)

	// The genesis gas limit, read from an early safe block on the verifier.
	oldInfo, err := sys.L2ELB.EthClient().InfoByNumber(t.Ctx(), 1)
	require.NoError(err)
	oldGasLimit := oldInfo.GasLimit()
	newGasLimit := oldGasLimit + 2_000_000

	// Change the L2 gas limit on the L1 SystemConfig.
	changeL1Block := setSystemConfigGasLimit(t, sys.L2Chain, sys.L1EL, sys.FunderL1, newGasLimit)

	// The op-con verifier must DERIVE the change onto its safe chain: once its safe
	// head's L1 origin passes the change, the derived block must carry the new gas
	// limit.
	got := waitForGasLimitAtOriginPast(t, sys.L2ELB, eth.Safe, changeL1Block)
	require.Equal(newGasLimit, got,
		"op-con verifier derived the wrong gas limit after the SystemConfig update")

	// The verifier stays in sync with the sequencer across the config change.
	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 60)
}
