package integration_test

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"net/url"
	"path"
	"runtime"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/kurtosisutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

const TestParams = `
participants:
  - el_type: geth
    el_extra_params:
      - "--gcmode=archive"
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
	kurtosisutil.Test(t)

	lgr := testlog.Logger(t, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, testFilename, _, ok := runtime.Caller(0)
	require.Truef(t, ok, "failed to get test filename")
	monorepoDir := path.Join(path.Dir(testFilename), "..", "..", "..")
	artifactsDir := path.Join(monorepoDir, "packages", "contracts-bedrock", "forge-artifacts")

	enclaveCtx := kurtosisutil.StartEnclave(t, ctx, lgr, "github.com/ethpandaops/ethereum-package", TestParams)

	service, err := enclaveCtx.GetServiceContext("el-1-geth-lighthouse")
	require.NoError(t, err)

	ip := service.GetMaybePublicIPAddress()
	ports := service.GetPublicPorts()
	rpcURL := fmt.Sprintf("http://%s:%d", ip, ports["rpc"].GetNumber())
	l1Client, err := ethclient.Dial(rpcURL)
	require.NoError(t, err)

	artifactsURL, err := url.Parse(fmt.Sprintf("file://%s", artifactsDir))
	require.NoError(t, err)

	depKey := new(deployerKey)
	l1ChainID := big.NewInt(77799777)
	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	pk, err := dk.Secret(depKey)
	require.NoError(t, err)
	signer := opcrypto.SignerFnFromBind(opcrypto.PrivateKeySignerFn(pk, l1ChainID))

	id := uint256.NewInt(1)

	deployerAddr, err := dk.Address(depKey)
	require.NoError(t, err)

	env := &pipeline.Env{
		Workdir:  t.TempDir(),
		L1Client: l1Client,
		Signer:   signer,
		Deployer: deployerAddr,
		Logger:   lgr,
	}

	t.Run("initial chain", func(t *testing.T) {
		intent, st := makeIntent(t, l1ChainID, artifactsURL, dk, id)

		require.NoError(t, deployer.ApplyPipeline(
			ctx,
			env,
			intent,
			st,
		))

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
			{"DelayedWETHImpl", st.ImplementationsDeployment.DelayedWETHImplAddress},
			{"OptimismPortalImpl", st.ImplementationsDeployment.OptimismPortalImplAddress},
			{"PreimageOracleSingleton", st.ImplementationsDeployment.PreimageOracleSingletonAddress},
			{"MipsSingleton", st.ImplementationsDeployment.MipsSingletonAddress},
			{"SystemConfigImpl", st.ImplementationsDeployment.SystemConfigImplAddress},
			{"L1CrossDomainMessengerImpl", st.ImplementationsDeployment.L1CrossDomainMessengerImplAddress},
			{"L1ERC721BridgeImpl", st.ImplementationsDeployment.L1ERC721BridgeImplAddress},
			{"L1StandardBridgeImpl", st.ImplementationsDeployment.L1StandardBridgeImplAddress},
			{"OptimismMintableERC20FactoryImpl", st.ImplementationsDeployment.OptimismMintableERC20FactoryImplAddress},
			{"DisputeGameFactoryImpl", st.ImplementationsDeployment.DisputeGameFactoryImplAddress},
		}
		for _, addr := range addrs {
			t.Run(addr.name, func(t *testing.T) {
				code, err := l1Client.CodeAt(ctx, addr.addr, nil)
				require.NoError(t, err)
				require.NotEmpty(t, code, "contracts %s at %s has no code", addr.name, addr.addr)
			})
		}

		validateOPChainDeployment(t, ctx, l1Client, st)
	})

	t.Run("subsequent chain", func(t *testing.T) {
		newID := uint256.NewInt(2)
		intent, st := makeIntent(t, l1ChainID, artifactsURL, dk, newID)
		env.Workdir = t.TempDir()

		require.NoError(t, deployer.ApplyPipeline(
			ctx,
			env,
			intent,
			st,
		))

		addrs := []struct {
			name string
			addr common.Address
		}{
			{"SuperchainConfigProxy", st.SuperchainDeployment.SuperchainConfigProxyAddress},
			{"ProtocolVersionsProxy", st.SuperchainDeployment.ProtocolVersionsProxyAddress},
			{"OpcmProxy", st.ImplementationsDeployment.OpcmProxyAddress},
		}
		for _, addr := range addrs {
			t.Run(addr.name, func(t *testing.T) {
				code, err := l1Client.CodeAt(ctx, addr.addr, nil)
				require.NoError(t, err)
				require.NotEmpty(t, code, "contracts %s at %s has no code", addr.name, addr.addr)
			})
		}

		validateOPChainDeployment(t, ctx, l1Client, st)
	})
}

func makeIntent(
	t *testing.T,
	l1ChainID *big.Int,
	artifactsURL *url.URL,
	dk *devkeys.MnemonicDevKeys,
	l2ChainID *uint256.Int,
) (*state.Intent, *state.State) {
	addrFor := func(key devkeys.Key) common.Address {
		addr, err := dk.Address(key)
		require.NoError(t, err)
		return addr
	}

	intent := &state.Intent{
		L1ChainID: l1ChainID.Uint64(),
		SuperchainRoles: state.SuperchainRoles{
			ProxyAdminOwner:       addrFor(devkeys.L1ProxyAdminOwnerRole.Key(l1ChainID)),
			ProtocolVersionsOwner: addrFor(devkeys.SuperchainDeployerKey.Key(l1ChainID)),
			Guardian:              addrFor(devkeys.SuperchainConfigGuardianKey.Key(l1ChainID)),
		},
		UseFaultProofs:       true,
		FundDevAccounts:      true,
		ContractArtifactsURL: (*state.ArtifactsURL)(artifactsURL),
		ContractsRelease:     "dev",
		Chains: []*state.ChainIntent{
			{
				ID: l2ChainID.Bytes32(),
				Roles: state.ChainRoles{
					ProxyAdminOwner:      addrFor(devkeys.L2ProxyAdminOwnerRole.Key(l1ChainID)),
					SystemConfigOwner:    addrFor(devkeys.SystemConfigOwner.Key(l1ChainID)),
					GovernanceTokenOwner: addrFor(devkeys.L2ProxyAdminOwnerRole.Key(l1ChainID)),
					UnsafeBlockSigner:    addrFor(devkeys.SequencerP2PRole.Key(l1ChainID)),
					Batcher:              addrFor(devkeys.BatcherRole.Key(l1ChainID)),
					Proposer:             addrFor(devkeys.ProposerRole.Key(l1ChainID)),
					Challenger:           addrFor(devkeys.ChallengerRole.Key(l1ChainID)),
				},
			},
		},
	}
	st := &state.State{
		Version: 1,
	}
	return intent, st
}

func validateOPChainDeployment(t *testing.T, ctx context.Context, l1Client *ethclient.Client, st *state.State) {
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
			{"AnchorStateRegistryImplAddress", chainState.AnchorStateRegistryImplAddress},
			{"FaultDisputeGameAddress", chainState.FaultDisputeGameAddress},
			{"PermissionedDisputeGameAddress", chainState.PermissionedDisputeGameAddress},
			{"DelayedWETHPermissionedGameProxyAddress", chainState.DelayedWETHPermissionedGameProxyAddress},
			{"DelayedWETHPermissionlessGameProxyAddress", chainState.DelayedWETHPermissionlessGameProxyAddress},
		}
		for _, addr := range chainAddrs {
			// TODO Delete this `if`` block once FaultDisputeGameAddress is deployed.
			if addr.name == "FaultDisputeGameAddress" {
				continue
			}
			t.Run(addr.name, func(t *testing.T) {
				code, err := l1Client.CodeAt(ctx, addr.addr, nil)
				require.NoError(t, err)
				require.NotEmpty(t, code, "contracts %s at %s for chain %s has no code", addr.name, addr.addr, chainState.ID)
			})
		}

		t.Run("l2 genesis", func(t *testing.T) {
			require.Greater(t, len(chainState.Allocs), 0)
		})
	}
}
