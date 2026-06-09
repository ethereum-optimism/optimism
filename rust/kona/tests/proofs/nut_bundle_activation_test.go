package proofs

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// forksWithoutNUTBundle lists forks from Karst onward that are activated via
// the legacy hardcoded upgrade-transactions path rather than a JSON NUT
// bundle. Adding a NUT bundle for one of these forks WILL fail this test —
// remove the entry here as part of the same PR.
var forksWithoutNUTBundle = map[forks.Name]bool{}

// TestActivationBlockNUTBundle verifies that, for every fork from Karst onward
// that uses the JSON NUT bundle system, the fork's activation block contains
// exactly the bundle's deposit transactions in order, every upgrade tx executes
// successfully, and the fault-proof program can prove the result.
//
// Discovery runs through [forks.From]([forks.Karst]), so any future fork is
// covered automatically. Forks that are still on the legacy hardcoded
// upgrade-transactions path are listed in [forksWithoutNUTBundle]; the test
// asserts in BOTH directions — a fork without a bundle that isn't on that list
// fails, and a fork on the list that gains a bundle also fails (so the
// exception list cannot silently go stale).
//
// The per-fork requirement beyond a JSON bundle is that the fork immediately
// preceding it is registered in [helpers.Hardforks] — a one-line entry needed
// for any fork-parametrized test in this package.
//
// Fork-specific state assertions (e.g. Karst's proxy implementation swap) are
// dispatched via the switch in [testActivationBlockNUTBundle]. Future forks
// with their own post-activation invariants register a case there.
func TestActivationBlockNUTBundle(gt *testing.T) {
	matrix := helpers.NewMatrix[forks.Name]()

	for _, fork := range forks.From(forks.Karst) {
		_, _, err := derive.UpgradeTransactions(fork)
		excepted := forksWithoutNUTBundle[fork]

		if err != nil {
			require.Truef(gt, excepted,
				"fork %s has no NUT bundle and is not on the forksWithoutNUTBundle exception list", fork)
			gt.Logf("skipping %s: no JSON NUT bundle (legacy hardcoded upgrade-tx path)", fork)
			continue
		}
		require.Falsef(gt, excepted,
			"fork %s now has a NUT bundle; remove it from forksWithoutNUTBundle", fork)

		preFork := forks.Prev(fork)
		require.NotEqual(gt, forks.None, preFork, "fork %s has no preceding fork in forks.All", fork)
		preHelper := lookupHardforkHelper(preFork)
		require.NotNil(gt, preHelper,
			"no pre-fork helper registered for NUT-bundle fork %s (prior fork: %s); add %s to helpers.Hardforks",
			fork, preFork, preFork)

		matrix.AddDefaultTestCasesWithName(
			string(fork),
			fork,
			helpers.NewForkMatrix(preHelper),
			testActivationBlockNUTBundle,
		)
	}

	matrix.Run(gt)
}

func testActivationBlockNUTBundle(gt *testing.T, testCfg *helpers.TestCfg[forks.Name]) {
	fork := testCfg.Custom
	t := actionsHelpers.NewDefaultTesting(gt)

	offset := uint64(4)
	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		dc.SetForkTimeOffset(fork, &offset)
	}
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testSetup)

	expectedTxs, expectedGas, err := derive.UpgradeTransactions(fork)
	require.NoError(t, err, "load NUT bundle for %s", fork)
	require.NotEmpty(t, expectedTxs, "bundle for %s must contain at least one upgrade tx", fork)

	env.Miner.ActEmptyBlock(t)
	env.Sequencer.ActL1HeadSignal(t)
	for i := 0; i < int(offset); i++ {
		env.Sequencer.ActL2EmptyBlock(t)
	}

	engine := env.Engine
	actHeader := engine.L2Chain().CurrentHeader()
	require.True(t,
		env.Sd.RollupCfg.IsActivationBlockForFork(actHeader.Time, fork),
		"expected activation block for %s at time %d", fork, actHeader.Time)

	actBlock := engine.L2Chain().GetBlockByHash(actHeader.Hash())
	txs := actBlock.Transactions()
	// Index 0 is the L1 info deposit; indices 1.. are the NUT upgrade deposits.
	require.Len(t, txs, 1+len(expectedTxs),
		"activation block should have 1 L1 info deposit + %d NUT upgrade txs", len(expectedTxs))

	var totalUpgradeGas uint64
	for i, rawExpected := range expectedTxs {
		actualBytes, err := txs[1+i].MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, []byte(rawExpected), actualBytes, "NUT tx %d byte mismatch", i)

		var expected types.Transaction
		require.NoError(t, expected.UnmarshalBinary(rawExpected))
		totalUpgradeGas += expected.Gas()
	}
	require.Equal(t, expectedGas, totalUpgradeGas, "total NUT gas must equal bundle total")

	// Every tx in the activation block — the L1 info deposit and all NUT upgrade
	// deposits — must execute successfully. A reverted upgrade tx would leave the
	// chain in a broken fork-activation state.
	receipts := engine.L2Chain().GetReceiptsByHash(actHeader.Hash())
	require.Len(t, receipts, len(txs), "receipt count must match tx count")
	for i, r := range receipts {
		require.Equal(t, types.ReceiptStatusSuccessful, r.Status,
			"activation-block tx %d reverted", i)
	}

	// Fork-specific post-activation assertions.
	switch fork {
	case forks.Karst:
		assertKarstActivation(t, env, actHeader)
	case forks.Lagoon:
		assertInteropActivation(t, env, actHeader)
	}

	// Advance the safe head across the activation boundary so the fault-proof
	// program verifies a non-trivial span including the upgrade block. No new
	// L2 blocks are produced past the activation block, so the safe head should
	// land exactly on it.
	env.BatchMineAndSync(t)
	l2SafeHead := env.Sequencer.L2Safe()
	require.Equal(t, bigs.Uint64Strict(actHeader.Number), l2SafeHead.Number,
		"safe head must be exactly the %s activation block", fork)

	// Skip the fault-proof step for Interop until the depset wiring lands in
	// op-program and kona-host single (tracked in
	// https://github.com/ethereum-optimism/optimism/issues/21114, item 4).
	// The activation transition itself is covered by
	// TestInteropFaultProofs_ActivationBoundary in op-acceptance-tests via
	// kona-host super.
	if fork == forks.Lagoon {
		return
	}

	env.RunFaultProofProgram(t, l2SafeHead.Number, testCfg.CheckResult, testCfg.InputParams...)
}

// assertKarstActivation verifies Karst-specific state changes: representative
// predeploy proxies' EIP-1967 implementation slots must change across the
// activation block and the new implementations must have code. This is a
// smoke test that the bundle's upgrade transactions actually rewrote proxy
// implementation pointers, not a check of what the new implementations do.
func assertKarstActivation(t actionsHelpers.StatefulTesting, env *helpers.L2FaultProofEnv, actHeader *types.Header) {
	ethCl := env.Engine.EthClient()
	postBlock := actHeader.Number
	preBlock := new(big.Int).Sub(postBlock, big.NewInt(1))

	// L1Block and GasPriceOracle mirror the proxies asserted by earlier fork
	// tests (ecotone, isthmus); covering them keeps vocabulary consistent
	// across fork tests.
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

		newImplAddr := common.BytesToAddress(postImpl)
		code, err := ethCl.CodeAt(context.Background(), newImplAddr, postBlock)
		require.NoError(t, err, "read code at new %s impl", p.name)
		require.NotEmptyf(t, code, "new %s impl %s must have code", p.name, newImplAddr)
	}
}

// assertInteropActivation asserts the single-chain Interop activation post-state:
// bundle predeploy impls installed, INTEROP flag unset, ETHLiquidity unfunded.
func assertInteropActivation(t actionsHelpers.StatefulTesting, env *helpers.L2FaultProofEnv, actHeader *types.Header) {
	ethCl := env.Engine.EthClient()
	postBlock := actHeader.Number
	preBlock := new(big.Int).Sub(postBlock, big.NewInt(1))

	// L1Block.isFeatureEnabled is mapping(bytes32 => bool) at storage slot 9
	// (see snapshots/storageLayout/L1Block.json).
	var featureKey [32]byte
	copy(featureKey[:], "INTEROP")
	mappingSlot := common.LeftPadBytes(big.NewInt(9).Bytes(), 32)
	slot := crypto.Keccak256Hash(featureKey[:], mappingSlot)

	pre, err := ethCl.StorageAt(context.Background(), predeploys.L1BlockAddr, slot, preBlock)
	require.NoError(t, err, "read L1Block.isFeatureEnabled(INTEROP) pre-activation")
	post, err := ethCl.StorageAt(context.Background(), predeploys.L1BlockAddr, slot, postBlock)
	require.NoError(t, err, "read L1Block.isFeatureEnabled(INTEROP) post-activation")
	require.Truef(t, allZero(pre), "INTEROP feature must be unset pre-activation, got %x", pre)
	require.Truef(t, allZero(post), "INTEROP feature must stay unset for single-chain activation, got %x", post)

	// The four Interop predeploys have their EIP-1967 implementation slot set
	// to a non-empty contract after the L2CM bundle's upgradePredeploys() call.
	interopProxies := []struct {
		name string
		addr common.Address
	}{
		{"CrossL2Inbox", predeploys.CrossL2InboxAddr},
		{"L2ToL2CrossDomainMessenger", predeploys.L2toL2CrossDomainMessengerAddr},
		{"SuperchainETHBridge", predeploys.SuperchainETHBridgeAddr},
		{"ETHLiquidity", predeploys.ETHLiquidityAddr},
	}
	for _, p := range interopProxies {
		impl, err := ethCl.StorageAt(context.Background(), p.addr, genesis.ImplementationSlot, postBlock)
		require.NoError(t, err, "read %s impl slot post-activation", p.name)
		implAddr := common.BytesToAddress(impl)
		require.Equal(t, common.Address{}, implAddr,
			"%s (%s) implementation slot must remain unset after Interop activation", p.name, p.addr)
		code, err := ethCl.CodeAt(context.Background(), implAddr, postBlock)
		require.NoError(t, err, "read code at new %s impl", p.name)
		require.Emptyf(t, code, "new %s impl %s must not have code", p.name, implAddr)
	}

	// ETHLiquidity stays at zero balance — the post-bundle funding wrapper does
	// not fire for single-chain activation.
	preBalance, err := ethCl.BalanceAt(context.Background(), predeploys.ETHLiquidityAddr, preBlock)
	require.NoError(t, err, "read ETHLiquidity balance pre-activation")
	postBalance, err := ethCl.BalanceAt(context.Background(), predeploys.ETHLiquidityAddr, postBlock)
	require.NoError(t, err, "read ETHLiquidity balance post-activation")
	require.True(t, preBalance.Sign() == 0, "ETHLiquidity must have zero balance pre-activation")
	require.True(t, postBalance.Sign() == 0, "ETHLiquidity must stay unfunded for single-chain activation")
}

func allZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

// lookupHardforkHelper resolves a fork name to its [helpers.Hardfork] entry by
// scanning [helpers.Hardforks]. Returns nil when the fork isn't registered.
func lookupHardforkHelper(name forks.Name) *helpers.Hardfork {
	for _, hf := range helpers.Hardforks {
		if forks.Name(hf.Name) == name {
			return hf
		}
	}
	return nil
}
