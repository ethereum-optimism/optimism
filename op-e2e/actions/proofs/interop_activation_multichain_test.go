package proofs

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// L2DevFeatureFlags predeploy address (`@custom:predeploy 0x...2D`) and the storage slot it uses
// for the dev-feature bitmap (BITMAP_SLOT = bytes32(uint256(keccak256("l2devfeatureflags.bitmap")) - 1)).
// Mirrors src/L2/L2DevFeatureFlags.sol. Kept inline because no Go constants exist for either today.
var (
	l2DevFeatureFlagsAddr       = common.HexToAddress("0x420000000000000000000000000000000000002D")
	l2DevFeatureFlagsBitmapSlot = common.HexToHash("0xc8bc8f9195cfb2d040744aac63412d02ffc186ea9bd519039edc4666ee9032bc")
)

// TestInteropActivation_MultiChain asserts the Interop activation block when the depset has more than one chain — the setFeature and ETHLiquidity-funding wrappers fire alongside the L2CM bundle.
func TestInteropActivation_MultiChain(gt *testing.T) {
	matrix := helpers.NewMatrix[forks.Name]()
	matrix.AddDefaultTestCasesWithName(
		"interop-multichain",
		forks.Interop,
		helpers.NewForkMatrix(helpers.Karst),
		testInteropActivation_MultiChain,
	)
	matrix.Run(gt)
}

func testInteropActivation_MultiChain(gt *testing.T, testCfg *helpers.TestCfg[forks.Name]) {
	// The KONA_HOST path only invokes `kona-host single`, does not support the super mode.
	helpers.SkipIfKona(gt)

	t := actionsHelpers.NewDefaultTesting(gt)

	interopOffset := uint64(4)
	testSetup := func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		dc.SetForkTimeOffset(forks.Interop, &interopOffset)
	}

	multiDepSet, err := depset.NewStaticConfigDependencySet(map[eth.ChainID]*depset.StaticConfigDependency{
		eth.ChainIDFromUInt64(901): {},
		eth.ChainIDFromUInt64(902): {},
	})
	require.NoError(t, err)

	// Seed the L2DevFeatureFlags predeploy with OPTIMISM_PORTAL_INTEROP at genesis. L2CM's upgrade()
	// reverts with L2ContractsManager_FeatureFlagMismatch when interop is scheduled but the on-chain
	// dev-feature bitmap doesn't have this bit set. The shared test alloc set leaves the bitmap zero
	// because intent.UseInterop is false; this hook patches genesis state only for this test.
	seedInteropDevFeatureFlag := func(sd *e2eutils.SetupData) {
		acct, ok := sd.L2Cfg.Alloc[l2DevFeatureFlagsAddr]
		require.Truef(t, ok, "L2DevFeatureFlags predeploy missing from L2 allocs at %s", l2DevFeatureFlagsAddr)
		if acct.Storage == nil {
			acct.Storage = map[common.Hash]common.Hash{}
		}
		acct.Storage[l2DevFeatureFlagsBitmapSlot] = devfeatures.OptimismPortalInteropFlag
		sd.L2Cfg.Alloc[l2DevFeatureFlagsAddr] = acct
	}

	env := helpers.NewL2FaultProofEnvWithDepSet(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), multiDepSet, seedInteropDevFeatureFlag, testSetup)

	env.Miner.ActEmptyBlock(t)
	env.Sequencer.ActL1HeadSignal(t)
	for i := 0; i < int(interopOffset); i++ {
		env.Sequencer.ActL2EmptyBlock(t)
	}

	engine := env.Engine
	actHeader := engine.L2Chain().CurrentHeader()
	require.True(t,
		env.Sd.RollupCfg.IsActivationBlockForFork(actHeader.Time, forks.Interop),
		"expected Interop activation block at time %d", actHeader.Time)

	actBlock := engine.L2Chain().GetBlockByHash(actHeader.Hash())
	bundleTxs, _, err := derive.UpgradeTransactions(forks.Interop)
	require.NoError(t, err)
	require.Len(t, actBlock.Transactions(), 1+2+len(bundleTxs),
		"activation block must contain L1 info + setFeature + bundle + funding")

	receipts := engine.L2Chain().GetReceiptsByHash(actHeader.Hash())
	for i, r := range receipts {
		require.Equal(t, types.ReceiptStatusSuccessful, r.Status,
			"activation-block tx %d reverted", i)
	}

	assertInteropMultiChainActivation(t, env, actHeader)

	env.BatchMineAndSync(t)
	l2SafeHead := env.Sequencer.L2Safe()
	require.Equal(t, bigs.Uint64Strict(actHeader.Number), l2SafeHead.Number,
		"safe head must be exactly the Interop activation block")

	env.RunFaultProofProgram(t, l2SafeHead.Number, testCfg.CheckResult, testCfg.InputParams...)
}

// assertInteropMultiChainActivation asserts impls installed, INTEROP flag set, and ETHLiquidity funded with u128::MAX.
func assertInteropMultiChainActivation(t actionsHelpers.StatefulTesting, env *helpers.L2FaultProofEnv, actHeader *types.Header) {
	ethCl := env.Engine.EthClient()
	postBlock := actHeader.Number
	preBlock := new(big.Int).Sub(postBlock, big.NewInt(1))

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
		require.NotEqualf(t, common.Address{}, implAddr,
			"%s (%s) implementation slot must be non-zero after Interop activation", p.name, p.addr)
		code, err := ethCl.CodeAt(context.Background(), implAddr, postBlock)
		require.NoError(t, err, "read code at new %s impl", p.name)
		require.NotEmptyf(t, code, "new %s impl %s must have code", p.name, implAddr)
	}

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
	require.Equalf(t, byte(1), post[31], "INTEROP feature must be set post-activation, got %x", post)

	preBalance, err := ethCl.BalanceAt(context.Background(), predeploys.ETHLiquidityAddr, preBlock)
	require.NoError(t, err, "read ETHLiquidity balance pre-activation")
	postBalance, err := ethCl.BalanceAt(context.Background(), predeploys.ETHLiquidityAddr, postBlock)
	require.NoError(t, err, "read ETHLiquidity balance post-activation")
	require.True(t, preBalance.Sign() == 0, "ETHLiquidity must have zero balance pre-activation")
	require.Equal(t, derive.InteropETHLiquidityFundingAmount(), postBalance,
		"ETHLiquidity must be funded with u128::MAX post-activation")
}
