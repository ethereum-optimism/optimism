package integration_test

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-service/testutils/anvil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/kurtosisutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const TestParams = `
participants:
  - el_type: geth
    el_extra_params:
      - "--gcmode=archive"
      - "--rpc.txfeecap=0"
    cl_type: lighthouse
network_params:
  prefunded_accounts: '{ "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266": { "balance": "1000000ETH" } }'
  additional_preloaded_contracts: '{
    "0x4e59b44847b379578588920cA78FbF26c0B4956C": {
      balance: "0ETH",
      code: "0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf3",
      storage: {},
      nonce: 0,
      secretKey: "0x"
    }
  }'
  network_id: "77799777"
  seconds_per_slot: 3
  genesis_delay: 0
`

type deployerKey struct{}

func (d *deployerKey) HDPath() string {
	return "m/44'/60'/0'/0/0"
}

func (d *deployerKey) String() string {
	return "deployer-key"
}

func TestEndToEndApply(t *testing.T) {
	op_e2e.InitParallel(t)
	kurtosisutil.Test(t)

	lgr := testlog.Logger(t, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	enclaveCtx := kurtosisutil.StartEnclave(t, ctx, lgr, "github.com/ethpandaops/ethereum-package", TestParams)

	service, err := enclaveCtx.GetServiceContext("el-1-geth-lighthouse")
	require.NoError(t, err)

	ip := service.GetMaybePublicIPAddress()
	ports := service.GetPublicPorts()
	rpcURL := fmt.Sprintf("http://%s:%d", ip, ports["rpc"].GetNumber())
	l1Client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)

	depKey := new(deployerKey)
	l1ChainID := big.NewInt(77799777)
	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	pk, err := dk.Secret(depKey)
	require.NoError(t, err)
	signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(pk, l1ChainID))

	l2ChainID1 := uint256.NewInt(1)
	l2ChainID2 := uint256.NewInt(2)

	deployerAddr, err := dk.Address(depKey)
	require.NoError(t, err)

	loc := localArtifacsLocator(t)

	bcaster, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:  log.NewLogger(log.DiscardHandler()),
		ChainID: l1ChainID,
		Client:  l1Client,
		Signer:  signer,
		From:    deployerAddr,
	})
	require.NoError(t, err)

	env, bundle, _ := createEnv(t, ctx, lgr, l1Client, bcaster, deployerAddr)
	intent, st := newIntent(t, l1ChainID, dk, l2ChainID1, loc, loc)
	cg := ethClientCodeGetter(ctx, l1Client)

	t.Run("initial chain", func(t *testing.T) {
		require.NoError(t, deployer.ApplyPipeline(
			ctx,
			env,
			bundle,
			intent,
			st,
		))

		validateSuperchainDeployment(t, st, cg)
		validateOPChainDeployment(t, cg, st, intent)
	})

	t.Run("subsequent chain", func(t *testing.T) {
		// create a new environment with wiped state to ensure we can continue using the
		// state from the previous deployment
		env, bundle, _ = createEnv(t, ctx, lgr, l1Client, bcaster, deployerAddr)
		intent.Chains = append(intent.Chains, newChainIntent(t, dk, l1ChainID, l2ChainID2))

		require.NoError(t, deployer.ApplyPipeline(
			ctx,
			env,
			bundle,
			intent,
			st,
		))

		validateOPChainDeployment(t, cg, st, intent)
	})
}

func localArtifacsLocator(t *testing.T) *opcm.ArtifactsLocator {
	_, testFilename, _, ok := runtime.Caller(0)
	require.Truef(t, ok, "failed to get test filename")
	monorepoDir := path.Join(path.Dir(testFilename), "..", "..", "..", "..")
	artifactsDir := path.Join(monorepoDir, "packages", "contracts-bedrock", "forge-artifacts")
	artifactsURL, err := url.Parse(fmt.Sprintf("file://%s", artifactsDir))
	require.NoError(t, err)
	loc := &opcm.ArtifactsLocator{
		URL: artifactsURL,
	}
	return loc
}

func createEnv(
	t *testing.T,
	ctx context.Context,
	lgr log.Logger,
	l1Client *ethclient.Client,
	bcaster broadcaster.Broadcaster,
	deployerAddr common.Address,
) (*pipeline.Env, pipeline.ArtifactsBundle, *script.Host) {
	_, testFilename, _, ok := runtime.Caller(0)
	require.Truef(t, ok, "failed to get test filename")
	monorepoDir := path.Join(path.Dir(testFilename), "..", "..", "..", "..")
	artifactsDir := path.Join(monorepoDir, "packages", "contracts-bedrock", "forge-artifacts")
	artifactsURL, err := url.Parse(fmt.Sprintf("file://%s", artifactsDir))
	require.NoError(t, err)
	artifactsLocator := &opcm.ArtifactsLocator{
		URL: artifactsURL,
	}

	artifactsFS, cleanupArtifacts, err := pipeline.DownloadArtifacts(ctx, artifactsLocator, pipeline.NoopDownloadProgressor)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, cleanupArtifacts())
	}()

	host, err := pipeline.DefaultScriptHost(
		bcaster,
		lgr,
		deployerAddr,
		artifactsFS,
		0,
	)
	require.NoError(t, err)

	env := &pipeline.Env{
		StateWriter:  pipeline.NoopStateWriter(),
		L1ScriptHost: host,
		L1Client:     l1Client,
		Broadcaster:  bcaster,
		Deployer:     deployerAddr,
		Logger:       lgr,
	}

	bundle := pipeline.ArtifactsBundle{
		L1: artifactsFS,
		L2: artifactsFS,
	}

	return env, bundle, host
}

func addrFor(t *testing.T, dk *devkeys.MnemonicDevKeys, key devkeys.Key) common.Address {
	addr, err := dk.Address(key)
	require.NoError(t, err)
	return addr
}

func newIntent(
	t *testing.T,
	l1ChainID *big.Int,
	dk *devkeys.MnemonicDevKeys,
	l2ChainID *uint256.Int,
	l1Loc *opcm.ArtifactsLocator,
	l2Loc *opcm.ArtifactsLocator,
) (*state.Intent, *state.State) {
	intent := &state.Intent{
		DeploymentStrategy: state.DeploymentStrategyLive,
		L1ChainID:          l1ChainID.Uint64(),
		SuperchainRoles: &state.SuperchainRoles{
			ProxyAdminOwner:       addrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainID)),
			ProtocolVersionsOwner: addrFor(t, dk, devkeys.SuperchainDeployerKey.Key(l1ChainID)),
			Guardian:              addrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainID)),
		},
		FundDevAccounts:    true,
		L1ContractsLocator: l1Loc,
		L2ContractsLocator: l2Loc,
		Chains: []*state.ChainIntent{
			newChainIntent(t, dk, l1ChainID, l2ChainID),
		},
	}
	st := &state.State{
		Version: 1,
	}
	return intent, st
}

func newChainIntent(t *testing.T, dk *devkeys.MnemonicDevKeys, l1ChainID *big.Int, l2ChainID *uint256.Int) *state.ChainIntent {
	return &state.ChainIntent{
		ID:                         l2ChainID.Bytes32(),
		BaseFeeVaultRecipient:      addrFor(t, dk, devkeys.BaseFeeVaultRecipientRole.Key(l1ChainID)),
		L1FeeVaultRecipient:        addrFor(t, dk, devkeys.L1FeeVaultRecipientRole.Key(l1ChainID)),
		SequencerFeeVaultRecipient: addrFor(t, dk, devkeys.SequencerFeeVaultRecipientRole.Key(l1ChainID)),
		Eip1559Denominator:         50,
		Eip1559Elasticity:          6,
		Roles: state.ChainRoles{
			ProxyAdminOwner:      addrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainID)),
			SystemConfigOwner:    addrFor(t, dk, devkeys.SystemConfigOwner.Key(l1ChainID)),
			GovernanceTokenOwner: addrFor(t, dk, devkeys.L2ProxyAdminOwnerRole.Key(l1ChainID)),
			UnsafeBlockSigner:    addrFor(t, dk, devkeys.SequencerP2PRole.Key(l1ChainID)),
			Batcher:              addrFor(t, dk, devkeys.BatcherRole.Key(l1ChainID)),
			Proposer:             addrFor(t, dk, devkeys.ProposerRole.Key(l1ChainID)),
			Challenger:           addrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainID)),
		},
	}
}

type codeGetter func(t *testing.T, addr common.Address) []byte

func ethClientCodeGetter(ctx context.Context, client *ethclient.Client) codeGetter {
	return func(t *testing.T, addr common.Address) []byte {
		code, err := client.CodeAt(ctx, addr, nil)
		require.NoError(t, err)
		return code
	}
}

func stateDumpCodeGetter(st *state.State) codeGetter {
	return func(t *testing.T, addr common.Address) []byte {
		acc, ok := st.L1StateDump.Data.Accounts[addr]
		require.True(t, ok, "no account found for address %s", addr)
		return acc.Code
	}
}

func validateSuperchainDeployment(t *testing.T, st *state.State, cg codeGetter) {
	addrs := []struct {
		name string
		addr common.Address
	}{
		{"SuperchainProxyAdmin", st.SuperchainDeployment.ProxyAdminAddress},
		{"SuperchainConfigProxy", st.SuperchainDeployment.SuperchainConfigProxyAddress},
		{"SuperchainConfigImpl", st.SuperchainDeployment.SuperchainConfigImplAddress},
		{"ProtocolVersionsProxy", st.SuperchainDeployment.ProtocolVersionsProxyAddress},
		{"ProtocolVersionsImpl", st.SuperchainDeployment.ProtocolVersionsImplAddress},
		{"OpcmProxy", st.ImplementationsDeployment.OpcmProxyAddress},
		{"PreimageOracleSingleton", st.ImplementationsDeployment.PreimageOracleSingletonAddress},
		{"MipsSingleton", st.ImplementationsDeployment.MipsSingletonAddress},
	}
	for _, addr := range addrs {
		t.Run(addr.name, func(t *testing.T) {
			code := cg(t, addr.addr)
			require.NotEmpty(t, code, "contract %s at %s has no code", addr.name, addr.addr)
		})
	}
}

func validateOPChainDeployment(t *testing.T, cg codeGetter, st *state.State, intent *state.Intent) {
	// Validate that the implementation addresses are always set, even in subsequent deployments
	// that pull from an existing OPCM deployment.
	implAddrs := []struct {
		name string
		addr common.Address
	}{
		{"DelayedWETHImplAddress", st.ImplementationsDeployment.DelayedWETHImplAddress},
		{"OptimismPortalImplAddress", st.ImplementationsDeployment.OptimismPortalImplAddress},
		{"SystemConfigImplAddress", st.ImplementationsDeployment.SystemConfigImplAddress},
		{"L1CrossDomainMessengerImplAddress", st.ImplementationsDeployment.L1CrossDomainMessengerImplAddress},
		{"L1ERC721BridgeImplAddress", st.ImplementationsDeployment.L1ERC721BridgeImplAddress},
		{"L1StandardBridgeImplAddress", st.ImplementationsDeployment.L1StandardBridgeImplAddress},
		{"OptimismMintableERC20FactoryImplAddress", st.ImplementationsDeployment.OptimismMintableERC20FactoryImplAddress},
		{"DisputeGameFactoryImplAddress", st.ImplementationsDeployment.DisputeGameFactoryImplAddress},
		{"MipsSingletonAddress", st.ImplementationsDeployment.MipsSingletonAddress},
		{"PreimageOracleSingletonAddress", st.ImplementationsDeployment.PreimageOracleSingletonAddress},
	}
	for _, addr := range implAddrs {
		require.NotEmpty(t, addr.addr, "%s should be set", addr.name)
		code := cg(t, addr.addr)
		require.NotEmpty(t, code, "contract %s at %s has no code", addr.name, addr.addr)
	}

	for _, chainState := range st.Chains {
		chainAddrs := []struct {
			name string
			addr common.Address
		}{
			{"ProxyAdminAddress", chainState.ProxyAdminAddress},
			{"AddressManagerAddress", chainState.AddressManagerAddress},
			{"L1ERC721BridgeProxyAddress", chainState.L1ERC721BridgeProxyAddress},
			{"SystemConfigProxyAddress", chainState.SystemConfigProxyAddress},
			{"OptimismMintableERC20FactoryProxyAddress", chainState.OptimismMintableERC20FactoryProxyAddress},
			{"L1StandardBridgeProxyAddress", chainState.L1StandardBridgeProxyAddress},
			{"L1CrossDomainMessengerProxyAddress", chainState.L1CrossDomainMessengerProxyAddress},
			{"OptimismPortalProxyAddress", chainState.OptimismPortalProxyAddress},
			{"DisputeGameFactoryProxyAddress", chainState.DisputeGameFactoryProxyAddress},
			{"AnchorStateRegistryProxyAddress", chainState.AnchorStateRegistryProxyAddress},
			{"FaultDisputeGameAddress", chainState.FaultDisputeGameAddress},
			{"PermissionedDisputeGameAddress", chainState.PermissionedDisputeGameAddress},
			{"DelayedWETHPermissionedGameProxyAddress", chainState.DelayedWETHPermissionedGameProxyAddress},
			// {"DelayedWETHPermissionlessGameProxyAddress", chainState.DelayedWETHPermissionlessGameProxyAddress},
		}
		for _, addr := range chainAddrs {
			// TODO Delete this `if`` block once FaultDisputeGameAddress is deployed.
			if addr.name == "FaultDisputeGameAddress" {
				continue
			}
			code := cg(t, addr.addr)
			require.NotEmpty(t, code, "contract %s at %s for chain %s has no code", addr.name, addr.addr, chainState.ID)
		}

		alloc := chainState.Allocs.Data.Accounts

		firstChainIntent := intent.Chains[0]
		checkImmutable(t, alloc, predeploys.BaseFeeVaultAddr, firstChainIntent.BaseFeeVaultRecipient)
		checkImmutable(t, alloc, predeploys.L1FeeVaultAddr, firstChainIntent.L1FeeVaultRecipient)
		checkImmutable(t, alloc, predeploys.SequencerFeeVaultAddr, firstChainIntent.SequencerFeeVaultRecipient)

		require.Equal(t, int(firstChainIntent.Eip1559Denominator), 50, "EIP1559Denominator should be set")
		require.Equal(t, int(firstChainIntent.Eip1559Elasticity), 6, "EIP1559Elasticity should be set")
	}
}

func getEIP1967ImplementationAddress(t *testing.T, allocations types.GenesisAlloc, proxyAddress common.Address) common.Address {
	storage := allocations[proxyAddress].Storage
	storageValue := storage[genesis.ImplementationSlot]
	require.NotEmpty(t, storageValue, "Implementation address for %s should be set", proxyAddress)
	return common.HexToAddress(storageValue.Hex())
}

func checkImmutable(t *testing.T, allocations types.GenesisAlloc, proxyContract common.Address, feeRecipient common.Address) {
	implementationAddress := getEIP1967ImplementationAddress(t, allocations, proxyContract)
	account, ok := allocations[implementationAddress]
	require.True(t, ok, "%s not found in allocations", implementationAddress.Hex())
	require.NotEmpty(t, account.Code, "%s should have code", implementationAddress.Hex())
	require.Contains(
		t,
		strings.ToLower(common.Bytes2Hex(account.Code)),
		strings.ToLower(strings.TrimPrefix(feeRecipient.Hex(), "0x")),
		"%s code should contain %s immutable", implementationAddress.Hex(), feeRecipient.Hex(),
	)
}

func TestApplyExistingOPCM(t *testing.T) {
	anvil.Test(t)

	forkRPCUrl := os.Getenv("SEPOLIA_RPC_URL")
	if forkRPCUrl == "" {
		t.Skip("no fork RPC URL provided")
	}

	lgr := testlog.Logger(t, slog.LevelDebug)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	runner, err := anvil.New(
		forkRPCUrl,
		lgr,
	)
	require.NoError(t, err)

	require.NoError(t, runner.Start(ctx))
	t.Cleanup(func() {
		require.NoError(t, runner.Stop())
	})

	l1Client, err := ethclient.Dial(runner.RPCUrl())
	require.NoError(t, err)

	l1ChainID := big.NewInt(11155111)
	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	// index 0 from Anvil's test set
	priv, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)
	signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(priv, l1ChainID))
	deployerAddr := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")

	l2ChainID := uint256.NewInt(1)

	bcaster, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:  lgr,
		ChainID: l1ChainID,
		Client:  l1Client,
		Signer:  signer,
		From:    deployerAddr,
	})
	require.NoError(t, err)

	env, bundle, _ := createEnv(t, ctx, lgr, l1Client, bcaster, deployerAddr)

	intent, st := newIntent(
		t,
		l1ChainID,
		dk,
		l2ChainID,
		opcm.DefaultL1ContractsLocator,
		opcm.DefaultL2ContractsLocator,
	)

	require.NoError(t, deployer.ApplyPipeline(
		ctx,
		env,
		bundle,
		intent,
		st,
	))

	validateOPChainDeployment(t, ethClientCodeGetter(ctx, l1Client), st, intent)
}

func TestL2BlockTimeOverride(t *testing.T) {
	op_e2e.InitParallel(t)
	kurtosisutil.Test(t)

	lgr := testlog.Logger(t, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	enclaveCtx := kurtosisutil.StartEnclave(t, ctx, lgr, "github.com/ethpandaops/ethereum-package", TestParams)

	service, err := enclaveCtx.GetServiceContext("el-1-geth-lighthouse")
	require.NoError(t, err)

	ip := service.GetMaybePublicIPAddress()
	ports := service.GetPublicPorts()
	rpcURL := fmt.Sprintf("http://%s:%d", ip, ports["rpc"].GetNumber())
	l1Client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)

	depKey := new(deployerKey)
	l1ChainID := big.NewInt(77799777)
	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	pk, err := dk.Secret(depKey)
	require.NoError(t, err)
	signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(pk, l1ChainID))

	l2ChainID := uint256.NewInt(1)

	deployerAddr, err := dk.Address(depKey)
	require.NoError(t, err)

	loc := localArtifacsLocator(t)

	bcaster, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
		Logger:  lgr,
		ChainID: l1ChainID,
		Client:  l1Client,
		Signer:  signer,
		From:    deployerAddr,
	})
	require.NoError(t, err)

	env, bundle, _ := createEnv(t, ctx, lgr, l1Client, bcaster, deployerAddr)

	intent, st := newIntent(
		t,
		l1ChainID,
		dk,
		l2ChainID,
		loc,
		loc,
	)

	intent.GlobalDeployOverrides = map[string]interface{}{
		"l2BlockTime": float64(3),
	}

	require.NoError(t, deployer.ApplyPipeline(
		ctx,
		env,
		bundle,
		intent,
		st,
	))

	cfg, err := state.CombineDeployConfig(intent, &state.ChainIntent{}, st, st.Chains[0])
	require.NoError(t, err)

	require.Equal(t, uint64(3), cfg.L2InitializationConfig.L2CoreDeployConfig.L2BlockTime, "L2 block time should be 3 seconds")
}

func TestApplyGenesisStrategy(t *testing.T) {
	op_e2e.InitParallel(t)

	lgr := testlog.Logger(t, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	depKey := new(deployerKey)
	l1ChainID := big.NewInt(77799777)
	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	l2ChainID1 := uint256.NewInt(1)
	l2ChainID2 := uint256.NewInt(2)

	deployerAddr, err := dk.Address(depKey)
	require.NoError(t, err)

	_, testFilename, _, ok := runtime.Caller(0)
	require.Truef(t, ok, "failed to get test filename")
	monorepoDir := path.Join(path.Dir(testFilename), "..", "..", "..", "..")
	artifactsDir := path.Join(monorepoDir, "packages", "contracts-bedrock", "forge-artifacts")
	artifactsURL, err := url.Parse(fmt.Sprintf("file://%s", artifactsDir))
	require.NoError(t, err)
	loc := &opcm.ArtifactsLocator{
		URL: artifactsURL,
	}

	env, bundle, _ := createEnv(t, ctx, lgr, nil, broadcaster.NoopBroadcaster(), deployerAddr)
	intent, st := newIntent(t, l1ChainID, dk, l2ChainID1, loc, loc)
	intent.Chains = append(intent.Chains, newChainIntent(t, dk, l1ChainID, l2ChainID2))
	intent.DeploymentStrategy = state.DeploymentStrategyGenesis

	require.NoError(t, deployer.ApplyPipeline(
		ctx,
		env,
		bundle,
		intent,
		st,
	))

	cg := stateDumpCodeGetter(st)
	validateSuperchainDeployment(t, st, cg)

	for i := range intent.Chains {
		t.Run(fmt.Sprintf("chain-%d", i), func(t *testing.T) {
			validateOPChainDeployment(t, cg, st, intent)
		})
	}
}
