package integration_test

import (
	"context"
	"encoding/binary"
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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

var (
	codeNamespacePrefix = common.HexToAddress("0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000")

	conditionalDeployerAddr = common.HexToAddress("0x420000000000000000000000000000000000002C")
	l2DevFeatureFlagsAddr   = common.HexToAddress("0x420000000000000000000000000000000000002D")

	l2DevFeatureBitmapSlot = common.HexToHash("0xc8bc8f9195cfb2d040744aac63412d02ffc186ea9bd519039edc4666ee9032bc")
	ozV5InitializableSlot  = common.HexToHash("0xf0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a00")
)

type allocMode struct {
	name             string
	l2cm             bool
	customGasToken   bool
	devFeatureBitmap common.Hash
	configure        func(t *testing.T, intent *state.Intent)
}

type generatedL2Genesis struct {
	mode        allocMode
	allocs      types.GenesisAlloc
	cfg         genesis.DeployConfig
	chainIntent *state.ChainIntent
}

func allocModes(t *testing.T) []allocMode {
	t.Helper()

	return []allocMode{
		{
			name:      "default",
			configure: func(t *testing.T, intent *state.Intent) {},
		},
		{
			name:             "l2cm",
			l2cm:             true,
			devFeatureBitmap: devfeatures.L2CMFlag,
			configure: func(t *testing.T, intent *state.Intent) {
				intent.GlobalDeployOverrides = map[string]any{
					"devFeatureBitmap": devfeatures.L2CMFlag,
				}
			},
		},
		{
			name:           "cgt",
			customGasToken: true,
			configure: func(t *testing.T, intent *state.Intent) {
				enableCustomGasToken(t, intent)
			},
		},
		{
			name:             "interop",
			devFeatureBitmap: devfeatures.OptimismPortalInteropFlag,
			configure: func(t *testing.T, intent *state.Intent) {
				intent.UseInterop = true
				intent.GlobalDeployOverrides = map[string]any{
					"devFeatureBitmap": devfeatures.OptimismPortalInteropFlag,
				}
			},
		},
		{
			name:             "cgt+interop",
			customGasToken:   true,
			devFeatureBitmap: devfeatures.OptimismPortalInteropFlag,
			configure: func(t *testing.T, intent *state.Intent) {
				enableCustomGasToken(t, intent)
				intent.UseInterop = true
				intent.GlobalDeployOverrides = map[string]any{
					"devFeatureBitmap": devfeatures.OptimismPortalInteropFlag,
				}
			},
		},
	}
}

func enableCustomGasToken(t *testing.T, intent *state.Intent) {
	t.Helper()

	intent.Chains[0].CustomGasToken = state.CustomGasToken{
		Name:             "Custom Gas Token",
		Symbol:           "CGT",
		InitialLiquidity: (*hexutil.Big)(cgtInitialLiquidity(t)),
	}
}

func cgtInitialLiquidity(t *testing.T) *big.Int {
	t.Helper()

	amount, ok := new(big.Int).SetString("1000000000000000000000", 10)
	require.True(t, ok)
	return amount
}

func generateL2Genesis(t *testing.T, mode allocMode) generatedL2Genesis {
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

	return generatedL2Genesis{
		mode:        mode,
		allocs:      st.Chains[0].Allocs.Data.Accounts,
		cfg:         cfg,
		chainIntent: intent.Chains[0],
	}
}

func assertL2GenesisInvariants(t *testing.T, gen generatedL2Genesis) {
	t.Helper()

	require.NotEmptyf(t, gen.allocs, "[%s] generated allocs", gen.mode.name)
	// L2Genesis.s.sol resets the proxy admin owner nonce (#21339) so it no longer leaks
	// into the dump. The empty opts guards against that regressing.
	require.NoErrorf(
		t,
		genesis.CheckL2GenesisAllocs(&foundry.ForgeAllocs{Accounts: gen.allocs}, genesis.CheckL2AllocsOpts{}),
		"[%s] global alloc invariants",
		gen.mode.name,
	)
	assertActivePredeploys(t, gen)
	assertInactivePredeploys(t, gen)
	assertProxyConfig(t, gen)
	assertInitializerState(t, gen)
}

func assertActivePredeploys(t *testing.T, gen generatedL2Genesis) {
	t.Helper()

	for _, proxy := range activePredeploys(gen.mode) {
		assertActiveProxy(t, gen, proxy)
	}
}

func assertInactivePredeploys(t *testing.T, gen generatedL2Genesis) {
	t.Helper()

	if !gen.mode.customGasToken {
		assertInactiveProxy(t, gen, predeploys.NativeAssetLiquidityAddr)
		assertInactiveProxy(t, gen, predeploys.LiquidityControllerAddr)
	}
	assertInactiveProxy(t, gen, predeploys.CrossL2InboxAddr)
	assertInactiveProxy(t, gen, predeploys.L2toL2CrossDomainMessengerAddr)
	assertInactiveProxy(t, gen, predeploys.SuperchainETHBridgeAddr)
	assertInactiveProxy(t, gen, predeploys.ETHLiquidityAddr)
}

func activePredeploys(mode allocMode) []common.Address {
	proxies := []common.Address{
		predeploys.LegacyMessagePasserAddr,
		predeploys.DeployerWhitelistAddr,
		predeploys.GasPriceOracleAddr,
		predeploys.L2CrossDomainMessengerAddr,
		predeploys.L2StandardBridgeAddr,
		predeploys.L2ERC721BridgeAddr,
		predeploys.OptimismMintableERC20FactoryAddr,
		predeploys.L1BlockNumberAddr,
		predeploys.OptimismMintableERC721FactoryAddr,
		predeploys.SequencerFeeVaultAddr,
		predeploys.BaseFeeVaultAddr,
		predeploys.L1FeeVaultAddr,
		predeploys.OperatorFeeVaultAddr,
		predeploys.L1BlockAddr,
		predeploys.L2ToL1MessagePasserAddr,
		predeploys.ProxyAdminAddr,
		predeploys.SchemaRegistryAddr,
		predeploys.EASAddr,
	}
	if mode.customGasToken {
		proxies = append(proxies, predeploys.LiquidityControllerAddr, predeploys.NativeAssetLiquidityAddr)
	}
	if mode.l2cm {
		proxies = append(proxies, conditionalDeployerAddr, l2DevFeatureFlagsAddr)
	}
	return proxies
}

func assertActiveProxy(t *testing.T, gen generatedL2Genesis, proxy common.Address) {
	t.Helper()

	account := requireAccount(t, gen, proxy)
	require.NotEmptyf(t, account.Code, "[%s] proxy %s code", gen.mode.name, proxy)
	require.Equalf(
		t,
		common.BytesToHash(predeploys.ProxyAdminAddr.Bytes()),
		storageAt(t, account, genesis.AdminSlot),
		"[%s] proxy %s admin",
		gen.mode.name,
		proxy,
	)

	impl := codeNamespace(proxy)
	implSlot, ok := account.Storage[genesis.ImplementationSlot]
	require.Truef(t, ok, "[%s] proxy %s impl slot", gen.mode.name, proxy)
	require.Equalf(
		t,
		common.BytesToHash(impl.Bytes()),
		implSlot,
		"[%s] proxy %s impl",
		gen.mode.name,
		proxy,
	)

	implAccount := requireAccount(t, gen, impl)
	require.NotEmptyf(t, implAccount.Code, "[%s] implementation %s code", gen.mode.name, impl)
}

func assertInactiveProxy(t *testing.T, gen generatedL2Genesis, proxy common.Address) {
	t.Helper()

	account := requireAccount(t, gen, proxy)
	require.NotEmptyf(t, account.Code, "[%s] inactive proxy %s code", gen.mode.name, proxy)
	require.Equalf(
		t,
		common.BytesToHash(predeploys.ProxyAdminAddr.Bytes()),
		storageAt(t, account, genesis.AdminSlot),
		"[%s] inactive proxy %s admin",
		gen.mode.name,
		proxy,
	)
	require.Equalf(t, common.Hash{}, account.Storage[genesis.ImplementationSlot], "[%s] inactive proxy %s impl", gen.mode.name, proxy)

	if implAccount, ok := gen.allocs[codeNamespace(proxy)]; ok {
		require.Emptyf(t, implAccount.Code, "[%s] inactive implementation %s code", gen.mode.name, codeNamespace(proxy))
	}
}

func assertProxyConfig(t *testing.T, gen generatedL2Genesis) {
	t.Helper()

	l1Deps := gen.cfg.L1DependenciesConfig
	vaults := gen.cfg.L2InitializationConfig.L2VaultsDeployConfig
	gasToken := gen.cfg.L2InitializationConfig.GasTokenDeployConfig

	assertAddressSlot(t, gen, predeploys.ProxyAdminAddr, slot(0), gen.chainIntent.Roles.L2ProxyAdminOwner)

	assertAddressSlot(t, gen, predeploys.L2CrossDomainMessengerAddr, slot(207), l1Deps.L1CrossDomainMessengerProxy)
	assertAddressSlot(t, gen, predeploys.L2StandardBridgeAddr, slot(3), predeploys.L2CrossDomainMessengerAddr)
	assertAddressSlot(t, gen, predeploys.L2StandardBridgeAddr, slot(4), l1Deps.L1StandardBridgeProxy)
	assertAddressSlot(t, gen, predeploys.L2ERC721BridgeAddr, slot(1), predeploys.L2CrossDomainMessengerAddr)
	assertAddressSlot(t, gen, predeploys.L2ERC721BridgeAddr, slot(2), l1Deps.L1ERC721BridgeProxy)
	assertAddressSlot(t, gen, predeploys.OptimismMintableERC20FactoryAddr, slot(1), predeploys.L2StandardBridgeAddr)
	assertPackedAddress(t, gen, predeploys.OptimismMintableERC721FactoryAddr, slot(1), 2, predeploys.L2ERC721BridgeAddr)
	assertUintSlot(t, gen, predeploys.OptimismMintableERC721FactoryAddr, slot(2), new(big.Int).SetUint64(gen.cfg.L1ChainID))

	assertFeeVault(t, gen, predeploys.SequencerFeeVaultAddr,
		vaults.SequencerFeeVaultRecipient,
		vaults.SequencerFeeVaultMinimumWithdrawalAmount.ToInt(),
		vaults.SequencerFeeVaultWithdrawalNetwork,
	)
	assertFeeVault(t, gen, predeploys.BaseFeeVaultAddr,
		vaults.BaseFeeVaultRecipient,
		vaults.BaseFeeVaultMinimumWithdrawalAmount.ToInt(),
		vaults.BaseFeeVaultWithdrawalNetwork,
	)
	assertFeeVault(t, gen, predeploys.L1FeeVaultAddr,
		vaults.L1FeeVaultRecipient,
		vaults.L1FeeVaultMinimumWithdrawalAmount.ToInt(),
		vaults.L1FeeVaultWithdrawalNetwork,
	)
	assertFeeVault(t, gen, predeploys.OperatorFeeVaultAddr,
		vaults.OperatorFeeVaultRecipient,
		vaults.OperatorFeeVaultMinimumWithdrawalAmount.ToInt(),
		vaults.OperatorFeeVaultWithdrawalNetwork,
	)

	checkStorageSlot(t, gen.allocs, l2DevFeatureFlagsAddr, l2DevFeatureBitmapSlot, gen.mode.devFeatureBitmap)

	if gen.mode.customGasToken {
		assertAddressSlot(t, gen, predeploys.LiquidityControllerAddr, slot(51), gasToken.LiquidityControllerOwner)
		assertShortStringSlot(t, gen, predeploys.LiquidityControllerAddr, slot(102), gasToken.GasPayingTokenName)
		assertShortStringSlot(t, gen, predeploys.LiquidityControllerAddr, slot(103), gasToken.GasPayingTokenSymbol)
		require.Equalf(
			t,
			gasToken.NativeAssetLiquidityAmount.ToInt(),
			requireAccount(t, gen, predeploys.NativeAssetLiquidityAddr).Balance,
			"[%s] native asset liquidity balance",
			gen.mode.name,
		)
		assertFeatureEnabled(t, gen, predeploys.L1BlockAddr, "CUSTOM_GAS_TOKEN", true)
	} else {
		assertFeatureEnabled(t, gen, predeploys.L1BlockAddr, "CUSTOM_GAS_TOKEN", false)
	}

	// Off even in the modes that set UseInterop (#20812).
	assertFeatureEnabled(t, gen, predeploys.L1BlockAddr, "INTEROP", false)
}

func assertInitializerState(t *testing.T, gen generatedL2Genesis) {
	t.Helper()

	assertInitializedV4(t, gen, predeploys.L2CrossDomainMessengerAddr, slot(0), 20)
	assertInitializedV4(t, gen, codeNamespace(predeploys.L2CrossDomainMessengerAddr), slot(0), 20)
	assertAddressSlot(t, gen, codeNamespace(predeploys.L2CrossDomainMessengerAddr), slot(207), common.Address{})

	for _, item := range []struct {
		proxy      common.Address
		initSlot   common.Hash
		initOffset int
	}{
		{predeploys.L2StandardBridgeAddr, slot(0), 0},
		{predeploys.L2ERC721BridgeAddr, slot(0), 0},
		{predeploys.OptimismMintableERC20FactoryAddr, slot(0), 0},
		{predeploys.OptimismMintableERC721FactoryAddr, slot(1), 0},
	} {
		assertInitializedV4(t, gen, item.proxy, item.initSlot, item.initOffset)
		assertInitializedV4(t, gen, codeNamespace(item.proxy), item.initSlot, item.initOffset)
	}

	assertAddressSlot(t, gen, codeNamespace(predeploys.L2StandardBridgeAddr), slot(4), common.Address{})
	assertAddressSlot(t, gen, codeNamespace(predeploys.L2ERC721BridgeAddr), slot(2), common.Address{})
	assertAddressSlot(t, gen, codeNamespace(predeploys.OptimismMintableERC20FactoryAddr), slot(1), common.Address{})
	assertPackedAddress(t, gen, codeNamespace(predeploys.OptimismMintableERC721FactoryAddr), slot(1), 2, common.Address{})
	assertUintSlot(t, gen, codeNamespace(predeploys.OptimismMintableERC721FactoryAddr), slot(2), big.NewInt(0))

	for _, proxy := range []common.Address{
		predeploys.SequencerFeeVaultAddr,
		predeploys.BaseFeeVaultAddr,
		predeploys.L1FeeVaultAddr,
		predeploys.OperatorFeeVaultAddr,
	} {
		assertInitializedV5(t, gen, proxy)
		assertInitializedV5(t, gen, codeNamespace(proxy))
		assertFeeVault(t, gen, codeNamespace(proxy), common.Address{}, maxUint256(), genesis.FromUint8(0))
	}

	if gen.mode.customGasToken {
		assertInitializedV4(t, gen, predeploys.LiquidityControllerAddr, slot(0), 0)
		assertInitializedV4(t, gen, codeNamespace(predeploys.LiquidityControllerAddr), slot(0), 0)
		assertAddressSlot(t, gen,
			codeNamespace(predeploys.LiquidityControllerAddr),
			slot(51),
			gen.cfg.L2InitializationConfig.GasTokenDeployConfig.LiquidityControllerOwner,
		)
		assertShortStringSlot(t, gen, codeNamespace(predeploys.LiquidityControllerAddr), slot(102), "")
		assertShortStringSlot(t, gen, codeNamespace(predeploys.LiquidityControllerAddr), slot(103), "")
	}
}

func assertFeeVault(
	t *testing.T,
	gen generatedL2Genesis,
	addr common.Address,
	recipient common.Address,
	minWithdrawal *big.Int,
	network genesis.WithdrawalNetwork,
) {
	t.Helper()

	assertUintSlot(t, gen, addr, slot(1), minWithdrawal)
	assertPackedAddress(t, gen, addr, slot(2), 0, recipient)

	expectedNetwork := network
	value := requireAccount(t, gen, addr).Storage[slot(2)]
	require.Equalf(
		t,
		expectedNetwork.ToUint8(),
		packedUint8(value, 20),
		"[%s] %s withdrawal network",
		gen.mode.name,
		addr,
	)
}

func assertInitializedV4(t *testing.T, gen generatedL2Genesis, addr common.Address, initSlot common.Hash, initOffset int) {
	t.Helper()

	value := storageAt(t, requireAccount(t, gen, addr), initSlot)
	require.NotZerof(t, packedUint8(value, initOffset), "[%s] %s initialized", gen.mode.name, addr)
	require.Zerof(t, packedUint8(value, initOffset+1), "[%s] %s initializing", gen.mode.name, addr)
}

func assertInitializedV5(t *testing.T, gen generatedL2Genesis, addr common.Address) {
	t.Helper()

	value := storageAt(t, requireAccount(t, gen, addr), ozV5InitializableSlot)
	require.NotZerof(t, binary.BigEndian.Uint64(value[24:32]), "[%s] %s initialized", gen.mode.name, addr)
	require.Zerof(t, packedUint8(value, 8), "[%s] %s initializing", gen.mode.name, addr)
}

func assertFeatureEnabled(t *testing.T, gen generatedL2Genesis, addr common.Address, feature string, expected bool) {
	t.Helper()

	featureSlot := crypto.Keccak256Hash(bytes32String(feature).Bytes(), slot(9).Bytes())
	value := requireAccount(t, gen, addr).Storage[featureSlot]
	require.Equalf(t, expected, value != (common.Hash{}), "[%s] %s feature %s", gen.mode.name, addr, feature)
}

func assertAddressSlot(t *testing.T, gen generatedL2Genesis, addr common.Address, key common.Hash, expected common.Address) {
	t.Helper()
	assertPackedAddress(t, gen, addr, key, 0, expected)
}

func assertPackedAddress(t *testing.T, gen generatedL2Genesis, addr common.Address, key common.Hash, offset int, expected common.Address) {
	t.Helper()

	value := requireAccount(t, gen, addr).Storage[key]
	require.Equalf(t, expected, packedAddress(value, offset), "[%s] %s slot %s offset %d", gen.mode.name, addr, key, offset)
}

func assertUintSlot(t *testing.T, gen generatedL2Genesis, addr common.Address, key common.Hash, expected *big.Int) {
	t.Helper()

	value := requireAccount(t, gen, addr).Storage[key]
	require.Zerof(t, value.Big().Cmp(expected), "[%s] %s slot %s", gen.mode.name, addr, key)
}

func assertShortStringSlot(t *testing.T, gen generatedL2Genesis, addr common.Address, key common.Hash, expected string) {
	t.Helper()

	require.LessOrEqual(t, len(expected), 31)

	var encoded common.Hash
	copy(encoded[:], []byte(expected))
	encoded[31] = byte(len(expected) * 2)

	checkStorageSlot(t, gen.allocs, addr, key, encoded)
}

func requireAccount(t *testing.T, gen generatedL2Genesis, addr common.Address) types.Account {
	t.Helper()

	account, ok := gen.allocs[addr]
	require.Truef(t, ok, "[%s] missing account %s", gen.mode.name, addr)
	return account
}

func storageAt(t *testing.T, account types.Account, key common.Hash) common.Hash {
	t.Helper()

	value, ok := account.Storage[key]
	require.Truef(t, ok, "missing storage slot %s", key)
	return value
}

func packedAddress(value common.Hash, offset int) common.Address {
	var out common.Address
	copy(out[:], value[32-offset-20:32-offset])
	return out
}

func packedUint8(value common.Hash, offset int) uint8 {
	return value[31-offset]
}

func codeNamespace(addr common.Address) common.Address {
	out := codeNamespacePrefix
	out[18] = addr[18]
	out[19] = addr[19]
	return out
}

func slot(i uint64) common.Hash {
	return common.BigToHash(new(big.Int).SetUint64(i))
}

func bytes32String(s string) common.Hash {
	var out common.Hash
	copy(out[:], []byte(s))
	return out
}

func maxUint256() *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
}

func TestL2GenesisAllocsInvariants(t *testing.T) {
	op_e2e.InitParallel(t)

	for _, mode := range allocModes(t) {
		t.Run(mode.name, func(t *testing.T) {
			assertL2GenesisInvariants(t, generateL2Genesis(t, mode))
		})
	}
}
