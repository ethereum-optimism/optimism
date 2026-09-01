package integration_test

import (
	"context"
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// Private interop has one op-deployer genesis. Consumers deterministically derive its public
// projection from that artifact.
var (
	privateInteropLockVault = common.HexToAddress("0x2222000000000000000000000000000000002222")

	// counterpartyChainID is the chain holding the ETHLockVault the private chain mints against.
	// Deliberately not the private chain's own ID.
	counterpartyChainID = uint64(901)
)

// generatePrivateInteropL2Genesis runs the full apply pipeline for a private chain.
func generatePrivateInteropL2Genesis(t *testing.T) generatedL2Genesis {
	t.Helper()

	gen, _ := generatePrivateInteropPair(t)
	return gen
}

// generatePrivateInteropPair also hands back the applied deployer state and intent.
func generatePrivateInteropPair(t *testing.T) (generatedL2Genesis, *appliedDeployment) {
	t.Helper()

	mode := allocMode{
		name:             "private-interop",
		customGasToken:   true,
		devFeatureBitmap: devfeatures.OptimismPortalInteropFlag,
		configure: func(t *testing.T, intent *state.Intent) {
			intent.UseInterop = true
			intent.GlobalDeployOverrides = map[string]any{
				"devFeatureBitmap": devfeatures.OptimismPortalInteropFlag,
				// Interop predeploys only reach genesis when Lagoon activates there.
				"l2GenesisLagoonTimeOffset": "0x0",
			}

			// The private chain's native unit is not ETH, which is why the protocol ETH path is
			// closed and the mint bridge exists.
			enableCustomGasToken(t, intent)
			intent.Chains[0].PrivateInterop = &state.PrivateInterop{
				CounterpartyChainID: counterpartyChainID,
				LockVault:           privateInteropLockVault,
			}
		},
	}

	return generatePrivateInteropGenesis(t, mode)
}

// appliedDeployment is the state and intent an apply run produced, together.
type appliedDeployment struct {
	st     *state.State
	intent *state.Intent
}

// generatePrivateInteropGenesis mirrors generateL2Genesis for the private chain.
func generatePrivateInteropGenesis(t *testing.T, mode allocMode) (generatedL2Genesis, *appliedDeployment) {
	t.Helper()

	lgr := testlog.Logger(t, slog.LevelWarn)
	_, pk, dk := shared.DefaultPrivkey(t)
	l1ChainID := big.NewInt(900)
	l2ChainID := uint256.NewInt(1)
	loc, _ := testutil.LocalArtifacts(t)
	intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, testCustomGasLimit)

	mode.configure(t, intent)

	require.NoError(t, deployer.ApplyPipeline(context.Background(), deployer.ApplyPipelineOpts{
		DeploymentTarget:   deployer.DeploymentTargetGenesis,
		L1RPCUrl:           "",
		DeployerPrivateKey: pk,
		Intent:             intent,
		State:              st,
		Logger:             lgr,
		StateWriter:        pipeline.NoopStateWriter(),
		CacheDir:           testutils.IsolatedTestDirWithAutoCleanup(t),
	}))

	require.NotEmpty(t, st.Chains)
	require.NotNil(t, st.Chains[0].Allocs)

	cfg, err := state.CombineDeployConfig(intent, intent.Chains[0], st, st.Chains[0])
	require.NoError(t, err)

	gen := generatedL2Genesis{
		mode:        mode,
		allocs:      st.Chains[0].Allocs.Data.Accounts,
		cfg:         cfg,
		chainIntent: intent.Chains[0],
	}

	require.NoErrorf(t, genesis.CheckL2GenesisAllocs(
		&foundry.ForgeAllocs{Accounts: gen.allocs},
		genesis.CheckL2AllocsOpts{},
	), "[%s] global alloc invariants", mode.name)

	return gen, &appliedDeployment{st: st, intent: intent}
}

// TestPrivateInteropGenesis asserts what makes a genesis private: the custom
// gas token feature is on, the mint bridge exists and is an authorized liquidity minter, and the
// initial liquidity remains in reserve until backed bridge mints occur.
func TestPrivateInteropGenesis(t *testing.T) {
	op_e2e.InitParallel(t)

	gen := generatePrivateInteropL2Genesis(t)

	t.Run("the custom gas token feature is on", func(t *testing.T) {
		assertFeatureEnabled(t, gen, predeploys.L1BlockAddr, "CUSTOM_GAS_TOKEN", true)
		assertActiveProxy(t, gen, predeploys.LiquidityControllerAddr)
		assertActiveProxy(t, gen, predeploys.NativeAssetLiquidityAddr)
	})

	t.Run("the NativeMintBridge is a live predeploy bound to the counterparty", func(t *testing.T) {
		assertActiveProxy(t, gen, predeploys.NativeMintBridgeAddr)

		// The counterparty chain ID and the lock vault are immutables, so they live in the
		// implementation's code rather than in storage.
		impl := requireAccount(t, gen, codeNamespace(predeploys.NativeMintBridgeAddr))
		require.Contains(t, string(impl.Code), string(privateInteropLockVault.Bytes()),
			"the lock vault immutable is baked into the implementation code")
		require.Contains(t, string(impl.Code),
			string(common.BigToHash(new(big.Int).SetUint64(counterpartyChainID)).Bytes()),
			"the counterparty chain ID immutable is baked into the implementation code")
	})

	t.Run("the mint bridge is an authorized liquidity minter", func(t *testing.T) {
		// LiquidityController.minters is `mapping(address => bool)` at slot 101 (see
		// snapshots/storageLayout/LiquidityController.json: OpenZeppelin's Initializable and
		// OwnableUpgradeable gaps sit ahead of it).
		key := crypto.Keccak256Hash(
			common.LeftPadBytes(predeploys.NativeMintBridgeAddr.Bytes(), 32),
			slot(101).Bytes(),
		)
		value := requireAccount(t, gen, predeploys.LiquidityControllerAddr).Storage[key]
		require.Equal(t, uint64(1), value.Big().Uint64(), "minters[NativeMintBridge]")
	})

	t.Run("opening liquidity remains in the reserve", func(t *testing.T) {
		reserve := requireAccount(t, gen, predeploys.NativeAssetLiquidityAddr)
		require.Zero(t, reserve.Balance.Cmp(gen.chainIntent.GetInitialLiquidity()), "NativeAssetLiquidity reserve")
	})

	t.Run("the public-projection predeploys are absent", func(t *testing.T) {
		assertInactiveProxy(t, gen, predeploys.ClaimRegistryAddr)
		assertInactiveProxy(t, gen, predeploys.EventReplayerAddr)
	})

	t.Run("the stock ETH path is absent", func(t *testing.T) {
		assertInactiveProxy(t, gen, predeploys.SuperchainETHBridgeAddr)
		assertInactiveProxy(t, gen, predeploys.ETHLiquidityAddr)
		require.Zero(t, requireAccount(t, gen, predeploys.ETHLiquidityAddr).Balance.Sign())
	})

	t.Run("the messenger keeps the stock implementation", func(t *testing.T) {
		_, artifactsFS := testutil.LocalArtifacts(t)
		implAccount := requireAccount(t, gen, codeNamespace(predeploys.L2toL2CrossDomainMessengerAddr))
		stockCode := deployedCode(t, artifactsFS,
			"L2ToL2CrossDomainMessenger.sol", "L2ToL2CrossDomainMessenger")
		require.Equal(t, stockCode, implAccount.Code,
			"the private chain's own messenger is stock; only its public projection replays")
	})
}

// TestPrivateInteropAbsentFromOrdinaryGenesis pins the byte-neutrality claim: with no privateInterop
// stanza, none of the three predeploys exist and the messenger is stock.
func TestPrivateInteropAbsentFromOrdinaryGenesis(t *testing.T) {
	op_e2e.InitParallel(t)

	gen := generateL2Genesis(t, allocMode{
		name:             "interop",
		devFeatureBitmap: devfeatures.OptimismPortalInteropFlag,
		configure: func(t *testing.T, intent *state.Intent) {
			intent.UseInterop = true
			intent.GlobalDeployOverrides = map[string]any{
				"devFeatureBitmap":          devfeatures.OptimismPortalInteropFlag,
				"l2GenesisLagoonTimeOffset": "0x0",
			}
		},
	})

	assertInactiveProxy(t, gen, predeploys.ClaimRegistryAddr)
	assertInactiveProxy(t, gen, predeploys.EventReplayerAddr)
	assertInactiveProxy(t, gen, predeploys.NativeMintBridgeAddr)

	_, artifactsFS := testutil.LocalArtifacts(t)
	implAccount := requireAccount(t, gen, codeNamespace(predeploys.L2toL2CrossDomainMessengerAddr))
	require.Equal(t,
		deployedCode(t, artifactsFS, "L2ToL2CrossDomainMessenger.sol", "L2ToL2CrossDomainMessenger"),
		implAccount.Code,
	)

}

func deployedCode(t *testing.T, fs foundry.StatDirFs, file, contract string) []byte {
	t.Helper()

	af := &foundry.ArtifactsFS{FS: fs}
	artifact, err := af.ReadArtifact(file, contract)
	require.NoError(t, err)
	return artifact.DeployedBytecode.Object
}

// TestPrivateInteropDepositGate pins the L1 side of the design: the private chain and its public
// projection are undepositable because their SystemConfig is initialized with maxResourceLimit = 0
// on a completely stock portal. Every depositTransaction is metered against that limit with a gas
// limit of at least 21000, so every deposit reverts on L1 and derivation never sees one -- which
// lets the public projection's ETH solvency story hold, since a deposit would mint uncapped ETH on it.
//
// Two things worth knowing that this test cannot assert. The revert reason is the generic "cannot
// buy more gas than available gas limit", so a depositor gets no hint that the chain is gated by
// design. And the config is written only by SystemConfig.initialize -- this version has no
// owner-callable setter -- so the gate is governance-protected rather than permanent: the proxy
// admin can upgrade-and-reinitialize, which is exactly how the trust model should state it.
func TestPrivateInteropDepositGate(t *testing.T) {
	op_e2e.InitParallel(t)

	const systemTxMaxGas = 1_000_000

	_, applied := generatePrivateInteropGenesis(t, allocMode{
		name:             "private-interop-deposit-gate",
		devFeatureBitmap: devfeatures.OptimismPortalInteropFlag,
		configure: func(t *testing.T, intent *state.Intent) {
			intent.UseInterop = true
			intent.GlobalDeployOverrides = map[string]any{
				"devFeatureBitmap":          devfeatures.OptimismPortalInteropFlag,
				"l2GenesisLagoonTimeOffset": "0x0",
			}
			enableCustomGasToken(t, intent)
			intent.Chains[0].PrivateInterop = &state.PrivateInterop{
				CounterpartyChainID: counterpartyChainID,
				LockVault:           privateInteropLockVault,
			}
		},
	})

	require.NotNil(t, applied.st.L1StateDump)
	systemConfig := applied.st.Chains[0].SystemConfigProxy
	require.NotEqual(t, common.Address{}, systemConfig)

	account, ok := applied.st.L1StateDump.Data.Accounts[systemConfig]
	require.True(t, ok, "SystemConfig proxy is in the L1 allocs")

	// SystemConfig._resourceConfig lives at slot 105 and packs the whole struct into one word:
	// maxResourceLimit (bytes 0-3), elasticityMultiplier (4), baseFeeMaxChangeDenominator (5),
	// minimumBaseFee (6-9), systemTxMaxGas (10-13), maximumBaseFee (14-29), counting from the
	// least significant byte.
	packed := account.Storage[slot(105)]
	require.NotEqual(t, common.Hash{}, packed, "the resource config was written at initialization")

	require.Zero(t, packedUint32(packed, 0), "maxResourceLimit is zero: deposits are impossible")
	require.Equal(t, uint8(1), packedUint8(packed, 4), "elasticityMultiplier")
	require.Equal(t, uint32(systemTxMaxGas), packedUint32(packed, 10), "systemTxMaxGas")
}

// TestOrdinaryChainKeepsTheDefaultResourceConfig is the other side of the gate: with no
// resourceConfig on the intent, a chain is initialized with the gas-limit-derived default and stays
// depositable.
func TestOrdinaryChainKeepsTheDefaultResourceConfig(t *testing.T) {
	op_e2e.InitParallel(t)

	_, applied := generatePrivateInteropGenesis(t, allocMode{
		name:      "default-resource-config",
		configure: func(t *testing.T, intent *state.Intent) {},
	})

	account, ok := applied.st.L1StateDump.Data.Accounts[applied.st.Chains[0].SystemConfigProxy]
	require.True(t, ok)

	packed := account.Storage[slot(105)]
	require.NotZero(t, packedUint32(packed, 0), "maxResourceLimit is positive: deposits work")
}

func packedUint32(value common.Hash, offset int) uint32 {
	var out uint32
	for i := 0; i < 4; i++ {
		out |= uint32(value[31-offset-i]) << (8 * i)
	}
	return out
}
