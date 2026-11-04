package proofs

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	nutsstate "github.com/ethereum-optimism/optimism/op-core/nuts/state"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
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
//
// Every fork from Lagoon onward also gets a multi-chain case. In that case, the
// chain's dependency set holds a second chain, so the Lagoon activation block
// activates the interop contracts. A fork after Lagoon boots its multi-chain
// case from the predecessor's multi-chain pre-fork state. The multi-chain case
// then checks with [assertCrossL2InboxAccessList] that the CrossL2Inbox proxy
// enforces its access list under the fork's rules.
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
			func(gt *testing.T, testCfg *helpers.TestCfg[forks.Name]) {
				testActivationBlockNUTBundle(gt, testCfg, false)
			},
		)
		if nutsstate.HasMultiChainState(fork) {
			matrix.AddDefaultTestCasesWithName(
				string(fork)+"-multichain",
				fork,
				helpers.NewForkMatrix(preHelper),
				func(gt *testing.T, testCfg *helpers.TestCfg[forks.Name]) {
					testActivationBlockNUTBundle(gt, testCfg, true)
				},
			)
		}
	}

	matrix.Run(gt)
}

// dependencySetPeer is the second chain in the dependency set of a multi-chain case. The chain
// does not exist. Only the size of the dependency set changes the activation block.
var dependencySetPeer = eth.ChainIDFromUInt64(4_242_424)

func testActivationBlockNUTBundle(gt *testing.T, testCfg *helpers.TestCfg[forks.Name], multiChain bool) {
	fork := testCfg.Custom
	t := actionsHelpers.NewDefaultTesting(gt)

	if multiChain {
		testCfg.DependencySetPeers = []eth.ChainID{dependencySetPeer}
	}
	env, actHeader := activateFork(t, testCfg, false)
	if multiChain {
		require.Len(t, env.Sd.DependencySet.Chains(), 2, "multi-chain case must have a dependency set of 2 chains")
	}

	expectedTxs, expectedGas, err := derive.UpgradeTransactions(fork)
	require.NoError(t, err, "load NUT bundle for %s", fork)
	require.NotEmpty(t, expectedTxs, "bundle for %s must contain at least one upgrade tx", fork)
	reservedTxs := expectedTxs
	if fork == forks.Lagoon {
		// A multi-chain activation puts setFeature(INTEROP) before the bundle and the
		// ETHLiquidity funding deposit after it.
		expectedTxs, _, err = derive.LagoonActivationUpgradeTransactions(multiChain)
		require.NoError(t, err)
	}

	engine := env.Engine
	_, txs, err := engine.SourceClient(t, 10).InfoAndDepositsByHash(t.Ctx(), actHeader.Hash())
	require.NoError(t, err)
	// Index 0 is the L1 info deposit; indices 1.. are the upgrade deposits.
	require.Len(t, txs, 1+len(expectedTxs),
		"activation block should have 1 L1 info deposit + %d upgrade txs", len(expectedTxs))

	var totalUpgradeGas uint64
	for i, rawExpected := range expectedTxs {
		actualBytes, err := txs[1+i].MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, []byte(rawExpected), actualBytes, "upgrade tx %d byte mismatch", i)

		expected, err := optypes.UnmarshalDepositTx(rawExpected)
		require.NoError(t, err)
		totalUpgradeGas += expected.Gas
	}
	// The reservation covers wrapper deposits that only a multi-chain activation emits. Sum the
	// multi-chain deposit set — the one that actually adds up to the reservation — rather than
	// trusting the getter that reported it.
	if fork == forks.Lagoon {
		reservedTxs, _, err = derive.LagoonActivationUpgradeTransactions(true)
		require.NoError(t, err)
	}
	var reservedGas uint64
	for _, raw := range reservedTxs {
		tx, err := optypes.UnmarshalDepositTx(raw)
		require.NoError(t, err)
		reservedGas += tx.Gas
	}
	require.Equal(t, expectedGas, reservedGas, "reserved gas must equal its activation deposits' total")
	require.LessOrEqual(t, totalUpgradeGas, reservedGas, "executed deposits must fit within the reservation")

	// Every tx in the activation block — the L1 info deposit and all NUT upgrade
	// deposits — must execute successfully. A reverted upgrade tx would leave the
	// chain in a broken fork-activation state.
	_, receipts, err := engine.SourceClient(t, 10).FetchReceipts(t.Ctx(), actHeader.Hash())
	require.NoError(t, err)
	require.Len(t, receipts, len(txs), "receipt count must match deposit count")
	for i, r := range receipts {
		require.Equal(t, types.ReceiptStatusSuccessful, r.Status,
			"activation-block tx %d reverted", i)
	}

	// Fork-specific post-activation assertions.
	switch fork {
	case forks.Karst:
		assertKarstActivation(t, env, actHeader)
	case forks.Lagoon:
		assertInteropActivation(t, env, actHeader, multiChain)
	}
	if multiChain {
		assertCrossL2InboxAccessList(t, env, fork, actHeader)
	}

	// Advance the safe head across the activation boundary so the fault-proof
	// program verifies a non-trivial span including the upgrade block. No new
	// L2 blocks are produced past the activation block, so the safe head should
	// land exactly on it.
	env.BatchMineAndSync(t)
	l2SafeHead := env.Sequencer.L2Safe()
	require.Equal(t, bigs.Uint64Strict(actHeader.Number), l2SafeHead.Number,
		"safe head must be exactly the %s activation block", fork)

	// Prove the activation-block span: kona-client verifies the activation transition itself.
	env.RunFaultProofProgram(t, l2SafeHead.Number, testCfg.CheckResult, testCfg.InputParams...)

	// The NUT-bundle upgrade gas applies only to the activation block. Build one
	// more block and confirm its gas limit drops back to the pre-activation
	// value — a regression would leak the upgrade gas onto every block after the
	// activation block.
	//
	// The bump is UpgradeGas(fork), not the bundle's own total: Lagoon additionally reserves gas
	// for the wrapper deposits that only a multi-chain activation emits. A single-chain activation
	// leaves that reservation unused.
	upgradeGas, err := derive.UpgradeGas(fork)
	require.NoError(t, err, "read upgrade gas for %s", fork)
	env.Sequencer.ActL2EmptyBlock(t)
	postActivation := engine.LatestHeader(t)
	preActivation := engine.HeaderByNumber(t, bigs.Uint64Strict(actHeader.Number)-1)
	require.Equal(t, preActivation.GasLimit+upgradeGas, actHeader.GasLimit,
		"activation block gas limit must be the pre-activation limit plus the one-time upgrade gas")
	require.Equal(t, preActivation.GasLimit, postActivation.GasLimit,
		"upgrade gas must not persist past the %s activation block", fork)

	// Advance the safe head across the post-activation block and prove that span too, so
	// kona-client itself verifies the gas-limit revert — not just op-node's assertion above.
	env.BatchMineAndSync(t)
	l2SafeHead = env.Sequencer.L2Safe()
	require.Equal(t, bigs.Uint64Strict(postActivation.Number), l2SafeHead.Number,
		"safe head must advance to the post-activation block")
	env.RunFaultProofProgram(t, l2SafeHead.Number, testCfg.CheckResult, testCfg.InputParams...)
}

// TestKarstActivationKeepUpgradeGas covers the keep_karst_upgrade_gas opt-out — the other branch of
// testActivationBlockNUTBundle's post-activation assertion. A chain that activated Karst with the
// upgrade-gas leak baked into its history keeps the inflated activation-block gas limit on later
// blocks rather than reverting, and the fault-proof program (kona-client) must agree.
func TestKarstActivationKeepUpgradeGas(gt *testing.T) {
	matrix := helpers.NewMatrix[forks.Name]()
	preHelper := lookupHardforkHelper(forks.Prev(forks.Karst))
	require.NotNil(gt, preHelper, "no pre-fork helper registered for Karst")
	matrix.AddDefaultTestCasesWithName("Karst", forks.Karst,
		helpers.NewForkMatrix(preHelper), testKarstActivationKeepUpgradeGas)
	matrix.Run(gt)
}

func testKarstActivationKeepUpgradeGas(gt *testing.T, testCfg *helpers.TestCfg[forks.Name]) {
	t := actionsHelpers.NewDefaultTesting(gt)

	// keep_karst_upgrade_gas is set in the deploy config (see activateFork), so the whole chain —
	// sequencer, derivation and the fault proof — runs with the opt-out from genesis.
	env, actHeader := activateFork(t, testCfg, true)

	karstGas, err := derive.UpgradeGas(forks.Karst)
	require.NoError(t, err)

	engine := env.Engine
	preActivation := engine.HeaderByNumber(t, bigs.Uint64Strict(actHeader.Number)-1)
	require.Equal(t, preActivation.GasLimit+karstGas, actHeader.GasLimit,
		"activation block gas limit must be the pre-activation limit plus the Karst upgrade gas")

	// The block after activation keeps the inflated limit instead of reverting.
	env.Sequencer.ActL2EmptyBlock(t)
	postActivation := engine.LatestHeader(t)
	require.Equal(t, actHeader.GasLimit, postActivation.GasLimit,
		"keep_karst_upgrade_gas must keep the inflated gas limit past the Karst activation block")

	// Prove the span through the post-activation block: kona-client (also opted out) must agree the
	// limit stays inflated.
	env.BatchMineAndSync(t)
	l2SafeHead := env.Sequencer.L2Safe()
	require.Equal(t, bigs.Uint64Strict(postActivation.Number), l2SafeHead.Number,
		"safe head must advance to the post-activation block")
	env.RunFaultProofProgram(t, l2SafeHead.Number, testCfg.CheckResult, testCfg.InputParams...)
}

// activateFork boots a fault-proof env for testCfg.Custom and advances to that
// fork's activation block, returning the env and the activation block header.
// It requires a committed pre-fork state for the fork (the state as of the
// predecessor fork) and overlays it onto the genesis predeploy set. When
// testCfg.DependencySetPeers is not empty, the chain is in a multi-chain
// dependency set and boots from the multi-chain pre-fork state.
// Shared by the validation test (testActivationBlockNUTBundle) and the artifact
// generator (TestGenerateForkState), so they exercise the same flow.
func activateFork(t actionsHelpers.Testing, testCfg *helpers.TestCfg[forks.Name], keepKarstUpgradeGas bool) (*helpers.L2FaultProofEnv, *types.Header) {
	fork := testCfg.Custom
	loadState := nutsstate.PreForkState
	if len(testCfg.DependencySetPeers) > 0 {
		loadState = nutsstate.PreForkMultiChainState
	}

	offset := uint64(4)
	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		dc.SetForkTimeOffset(fork, &offset)
		dc.KeepKarstUpgradeGas = keepKarstUpgradeGas
	}

	alloc, err := loadState(fork)
	require.NoErrorf(t, err, "no committed pre-fork state for %s — generate the predecessor fork's state (see op-core/nuts/state)", fork)
	testCfg.Allocs = &e2eutils.AllocParams{
		PrefundTestUsers:            true,
		L2Alloc:                     alloc,
		L2AllocIsFrozenPreForkState: true,
	}

	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), testSetup)

	env.Miner.ActEmptyBlock(t)
	env.Sequencer.ActL1HeadSignal(t)
	for i := 0; i < int(offset); i++ {
		env.Sequencer.ActL2EmptyBlock(t)
	}

	actHeader := env.Engine.LatestHeader(t)
	require.True(t,
		env.Sd.RollupCfg.IsActivationBlockForFork(actHeader.Time, fork),
		"expected activation block for %s at time %d", fork, actHeader.Time)
	return env, actHeader
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

// assertInteropActivation asserts the Interop activation post-state. A single-chain activation
// leaves the INTEROP flag unset, the interop proxies without an implementation and ETHLiquidity
// unfunded. A multi-chain activation sets the flag, gives each interop proxy an implementation
// with code and funds ETHLiquidity.
func assertInteropActivation(t actionsHelpers.StatefulTesting, env *helpers.L2FaultProofEnv, actHeader *types.Header, multiChain bool) {
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
	require.Equalf(t, multiChain, !allZero(post), "INTEROP feature set post-activation: got %x", post)

	interopProxies := []struct {
		name string
		addr common.Address
	}{
		{"CrossL2Inbox", predeploys.CrossL2InboxAddr},
		{"L2ToL2CrossDomainMessenger", predeploys.L2toL2CrossDomainMessengerAddr},
		{"SuperchainETHBridge", predeploys.SuperchainETHBridgeAddr},
		{"ETHLiquidity", predeploys.ETHLiquidityAddr},
		{"UndeliveredMessageExporter", predeploys.UndeliveredMessageExporterAddr},
	}
	for _, p := range interopProxies {
		impl, err := ethCl.StorageAt(context.Background(), p.addr, genesis.ImplementationSlot, postBlock)
		require.NoError(t, err, "read %s impl slot post-activation", p.name)
		implAddr := common.BytesToAddress(impl)
		code, err := ethCl.CodeAt(context.Background(), implAddr, postBlock)
		require.NoError(t, err, "read code at new %s impl", p.name)
		if multiChain {
			require.NotEqualf(t, common.Address{}, implAddr,
				"%s (%s) implementation slot must be set after multi-chain Interop activation", p.name, p.addr)
			require.NotEmptyf(t, code, "new %s impl %s must have code", p.name, implAddr)
		} else {
			require.Equalf(t, common.Address{}, implAddr,
				"%s (%s) implementation slot must remain unset after single-chain Interop activation", p.name, p.addr)
			require.Emptyf(t, code, "new %s impl %s must not have code", p.name, implAddr)
		}
	}

	expectedBalance := new(big.Int)
	if multiChain {
		expectedBalance = derive.InteropETHLiquidityFundingAmount()
	}
	preBalance, err := ethCl.BalanceAt(context.Background(), predeploys.ETHLiquidityAddr, preBlock)
	require.NoError(t, err, "read ETHLiquidity balance pre-activation")
	postBalance, err := ethCl.BalanceAt(context.Background(), predeploys.ETHLiquidityAddr, postBlock)
	require.NoError(t, err, "read ETHLiquidity balance post-activation")
	require.True(t, preBalance.Sign() == 0, "ETHLiquidity must have zero balance pre-activation")
	require.Equalf(t, 0, expectedBalance.Cmp(postBalance),
		"ETHLiquidity balance post-activation: got %s, want %s", postBalance, expectedBalance)
}

// assertCrossL2InboxAccessList checks that the CrossL2Inbox proxy enforces its access list.
// CrossL2Inbox finds an undeclared message from the gas cost of one SLOAD. A compiler that removes
// the SLOAD, or a fork that changes the SLOAD gas cost, can let every message through.
//
// The implementation runs by DELEGATECALL, so the SLOAD reads proxy storage. The access list must
// therefore declare the checksum for the proxy address.
func assertCrossL2InboxAccessList(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, fork forks.Name, actHeader *types.Header) {
	inbox := predeploys.CrossL2InboxAddr
	ethCl := env.Engine.EthClient()
	impl, err := ethCl.StorageAt(t.Ctx(), inbox, genesis.ImplementationSlot, actHeader.Number)
	require.NoError(t, err, "read CrossL2Inbox impl slot")
	require.NotEqualf(t, common.Address{}, common.BytesToAddress(impl),
		"CrossL2Inbox proxy must have an implementation after multi-chain %s activation", fork)

	msg := messages.Message{
		Identifier: messages.Identifier{
			Origin:      common.Address{0xaa},
			BlockNumber: 1,
			LogIndex:    2,
			Timestamp:   3,
			ChainID:     dependencySetPeer,
		},
		PayloadHash: common.Hash{0xbb},
	}
	trigger := &txintent.ExecTrigger{Executor: inbox, Msg: msg}
	calldata, err := trigger.EncodeInput()
	require.NoError(t, err)
	accessList, err := trigger.AccessList()
	require.NoError(t, err)

	// An eth_call without a gas price runs with a zero base fee. A gas price above the base fee keeps
	// the real base fee and passes the CrossL2Inbox deposit check.
	require.Positive(t, actHeader.BaseFee.Sign(), "activation block base fee must be positive")
	call := ethereum.CallMsg{
		From:     env.Alice.Address(),
		To:       &inbox,
		Gas:      1_000_000,
		GasPrice: new(big.Int).Mul(actHeader.BaseFee, big.NewInt(2)),
		Data:     calldata,
	}
	_, err = ethCl.CallContract(t.Ctx(), call, actHeader.Number)
	require.Equal(t, crypto.Keccak256([]byte("NotInAccessList()"))[:4], revertData(t, err),
		"validateMessage without an access list must revert with NotInAccessList")

	call.AccessList = accessList
	_, err = ethCl.CallContract(t.Ctx(), call, actHeader.Number)
	require.NoError(t, err, "validateMessage with the checksum in the access list must succeed")
}

// revertData returns the revert data of a failed eth_call.
func revertData(t actionsHelpers.Testing, err error) []byte {
	var dataErr rpc.DataError
	require.ErrorAs(t, err, &dataErr, "eth_call must revert with data")
	hexData, ok := dataErr.ErrorData().(string)
	require.Truef(t, ok, "revert data must be a hex string, got %T", dataErr.ErrorData())
	data, err := hexutil.Decode(hexData)
	require.NoError(t, err)
	return data
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
