package proofs

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// TestKarstActivationBlockGasLimitProof reproduces, through the fault-proof
// program, the Karst activation-block gas-limit bug first observed in the op-e2e
// action test TestKarstActivationBlockGasLimit
// (op-e2e/actions/upgrades/karst_fork_test.go).
//
// The Karst activation block is granted extra gas (sysConfig.GasLimit + nutGas)
// to fit the NUT bundle's upgrade transactions
// (op-node/rollup/derive/attributes.go: gasLimit = sysConfig.GasLimit + upgradeGas).
// The bump is intended to last for the activation block ONLY; the very next block
// should return to sysConfig.GasLimit.
//
// It does not. The SystemConfig gas limit is encoded in the L2 block solely via
// the header's GasLimit field (the L1-info deposit tx carries no gas limit), and
// both clients recover it from there:
//   - op-node:  PayloadToSystemConfig -> GasLimit = payload.GasLimit
//     (op-node/rollup/derive/payload_util.go)
//   - kona:     to_system_config      -> gas_limit = block.header.gas_limit
//     (rust/kona/crates/protocol/protocol/src/utils.rs)
//
// So the activation block's bumped header gas limit is read back as the system
// config gas limit for the next block, and the one-block bump becomes permanent.
//
// This is a RED reproduction test: it FAILS on the current (buggy) code. Because
// op-node and kona share the bug, they agree, so the fault-proof program PASSES
// (kona re-derives the same elevated post-activation block the engine produced).
// The failure is therefore asserted at the engine level — the final assertion
// below demands the intended-correct behavior and fails on the bug. See that
// assertion's comment for the explicit want/got.
func TestKarstActivationBlockGasLimitProof(gt *testing.T) {
	t := actionsHelpers.NewDefaultTesting(gt)

	// Genesis on Jovian (the fork preceding Karst); Karst activates a few blocks
	// later so we observe pre-, activation-, and post-activation blocks.
	testCfg := &helpers.TestCfg[any]{
		Hardfork:    helpers.Jovian,
		CheckResult: helpers.ExpectNoError(),
	}
	offset := uint64(4)
	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		dc.SetForkTimeOffset(forks.Karst, &offset)
	}
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testSetup)

	engine := env.Engine
	sequencer := env.Sequencer
	rollupCfg := env.Sd.RollupCfg

	// The Karst NUT bundle reserves additional gas for the activation block.
	_, nutGas, err := derive.UpgradeTransactions(forks.Karst)
	require.NoError(t, err)
	require.NotZero(t, nutGas, "Karst NUT bundle should reserve gas")

	// Karst must not be active at genesis. The normal block gas limit is the
	// system config gas limit.
	require.False(t, rollupCfg.IsKarst(sequencer.L2Unsafe().Time), "Karst should not be active at genesis")
	normalGasLimit := engine.L2Chain().CurrentHeader().GasLimit
	require.NotZero(t, normalGasLimit)

	env.Miner.ActEmptyBlock(t)
	sequencer.ActL1HeadSignal(t)

	// Build empty L2 blocks until Karst activates. ActBuildL2ToFork stops on the
	// first block in which the fork is active, i.e. the activation block.
	sequencer.ActBuildL2ToFork(t, forks.Karst)
	actHeader := engine.L2Chain().CurrentHeader()
	require.True(t, rollupCfg.IsKarstActivationBlock(actHeader.Time),
		"expected to land exactly on the Karst activation block")

	// Expected and correct: the activation block gas limit is bumped by the NUT
	// gas allocation. This assertion passes.
	require.Equalf(t, normalGasLimit+nutGas, actHeader.GasLimit,
		"Karst activation block gas limit should be sysConfig gas + NUT gas: want=%d (normal=%d + nut=%d) got=%d",
		normalGasLimit+nutGas, normalGasLimit, nutGas, actHeader.GasLimit)

	// Build one more block past the activation block.
	sequencer.ActL2EmptyBlock(t)
	postHeader := engine.L2Chain().CurrentHeader()
	require.False(t, rollupCfg.IsKarstActivationBlock(postHeader.Time),
		"the block after activation should not be the activation block")

	// Batch the unsafe chain (through the post-activation block) to L1 and derive,
	// so the safe head covers the activation and post-activation transitions.
	l2SafeHead := env.BatchMineAndSync(t)
	require.GreaterOrEqual(t, l2SafeHead.Number, bigs.Uint64Strict(postHeader.Number),
		"safe head must reach the post-activation block")

	// Run the fault-proof program for every transition from genesis through the
	// post-activation block. This PASSES: kona-proof re-derives the post-activation
	// block and arrives at the SAME (elevated) gas limit the engine produced, so
	// the output-root claims match. That is the proof-side replication — the bug
	// lives identically in kona's to_system_config, not just in op-node.
	env.RunFaultProofProgramFromGenesis(t, l2SafeHead.Number, helpers.ExpectNoError())

	// ====================================================================
	// THIS ASSERTION IS THE BUG. IT FAILS ON PURPOSE ON THE CURRENT CODE.
	// ====================================================================
	// The post-activation block must return to the system config gas limit; the
	// Karst NUT-gas bump is meant for the activation block ONLY. Instead the gas
	// limit stays permanently elevated by nutGas, because both op-node
	// (PayloadToSystemConfig) and kona (to_system_config) recover SystemConfig.GasLimit
	// from the block header — so the activation block's bumped header gas limit is
	// read back as the system config gas limit for this and every later block.
	//
	// On the current (buggy) code this fails with, e.g.:
	//     want=30000000 (normal)  got=85370657 (normal + nutGas)
	//
	// When the bug is fixed (in op-node AND kona, in lockstep, behind the Karst
	// fork gate) this assertion will pass and the test goes green.
	require.Equalf(t, normalGasLimit, postHeader.GasLimit,
		"BUG: gas limit must return to the system config value after the Karst activation block, "+
			"but it stays elevated by the NUT gas: want=%d (normal) got=%d (normal=%d + nut=%d)",
		normalGasLimit, postHeader.GasLimit, normalGasLimit, nutGas)
}
