package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/forking"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func Test_makeDCI_OpcmAddress(t *testing.T) {
	chainID := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000300")
	baseIntent, baseChainIntent, baseState := makeDCITestInputs(chainID)
	opcmV2Addr := baseState.ImplementationsDeployment.OpcmV2Impl

	zeroOpcmState := *baseState
	zeroOpcmState.ImplementationsDeployment = &addresses.ImplementationsContracts{}

	tests := []struct {
		name           string
		intent         *state.Intent
		thisIntent     *state.ChainIntent
		chainID        common.Hash
		st             *state.State
		expectedOpcm   common.Address
		shouldThrowErr bool
		expectedErrMsg string
	}{
		{
			name:           "uses_opcm_v2",
			intent:         baseIntent,
			thisIntent:     baseChainIntent,
			chainID:        chainID,
			st:             baseState,
			expectedOpcm:   opcmV2Addr,
			shouldThrowErr: false,
			expectedErrMsg: "",
		},
		{
			name:           "opcm_v2_impl_zero_reverts",
			intent:         baseIntent,
			thisIntent:     baseChainIntent,
			chainID:        chainID,
			st:             &zeroOpcmState,
			expectedOpcm:   common.Address{},
			shouldThrowErr: true,
			expectedErrMsg: "OPCM implementation is not deployed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := makeDCI(tt.intent, tt.thisIntent, tt.chainID, tt.st)
			if gotErr != nil {
				if !tt.shouldThrowErr {
					t.Errorf("makeDCI() failed: %v", gotErr)
				}
				if tt.expectedErrMsg != "" && !strings.Contains(gotErr.Error(), tt.expectedErrMsg) {
					t.Errorf("makeDCI() error = %v, want error containing %q", gotErr, tt.expectedErrMsg)
				}
				return
			}
			if tt.shouldThrowErr {
				t.Fatal("makeDCI() succeeded unexpectedly")
			}
			if got.Opcm != tt.expectedOpcm {
				t.Errorf("makeDCI() Opcm = %v, want %v", got.Opcm, tt.expectedOpcm)
			}
		})
	}
}

func TestMakeDCIProofInputs(t *testing.T) {
	chainID := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000300")
	statePrestate := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	stateAnchorRoot := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	overridePrestate := common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333")
	ignoredOverridePrestate := common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444")
	ignoredStatePrestate := common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555")
	ignoredStateAnchorRoot := common.HexToHash("0x6666666666666666666666666666666666666666666666666666666666666666")

	tests := []struct {
		name                 string
		globalDeployOverride map[string]any
		chainDeployOverride  map[string]any
		chains               []*state.ChainState
		expectedPrestate     common.Hash
		expectedAnchorRoot   common.Hash
		expectedErrMsg       string
	}{
		{
			name:               "permissioned default",
			expectedPrestate:   standard.DisputeAbsolutePrestate,
			expectedAnchorRoot: opcm.DefaultStartingAnchorRoot.Root,
		},
		{
			name: "permissionless cannon uses state",
			chainDeployOverride: map[string]any{
				"respectedGameType": uint32(gameTypes.CannonGameType),
			},
			chains: []*state.ChainState{{
				ID:                 chainID,
				Prestate:           statePrestate,
				StartingAnchorRoot: stateAnchorRoot,
			}},
			expectedPrestate:   statePrestate,
			expectedAnchorRoot: stateAnchorRoot,
		},
		{
			name: "permissionless cannon kona uses state",
			chainDeployOverride: map[string]any{
				"respectedGameType": uint32(gameTypes.CannonKonaGameType),
			},
			chains: []*state.ChainState{{
				ID:                 chainID,
				Prestate:           statePrestate,
				StartingAnchorRoot: stateAnchorRoot,
			}},
			expectedPrestate:   statePrestate,
			expectedAnchorRoot: stateAnchorRoot,
		},
		{
			name: "permissionless missing prestate",
			chainDeployOverride: map[string]any{
				"respectedGameType": uint32(gameTypes.CannonGameType),
			},
			chains: []*state.ChainState{{
				ID:                 chainID,
				StartingAnchorRoot: stateAnchorRoot,
			}},
			expectedErrMsg: "op-deployer prestate",
		},
		{
			name: "permissionless missing anchor root",
			chainDeployOverride: map[string]any{
				"respectedGameType": uint32(gameTypes.CannonGameType),
			},
			chains: []*state.ChainState{{
				ID:       chainID,
				Prestate: statePrestate,
			}},
			expectedErrMsg: "genesis-output-root stage",
		},
		{
			name: "permissionless ignores prestate override",
			chainDeployOverride: map[string]any{
				"respectedGameType":         uint32(gameTypes.CannonGameType),
				"faultGameAbsolutePrestate": ignoredOverridePrestate,
			},
			chains: []*state.ChainState{{
				ID:                 chainID,
				Prestate:           statePrestate,
				StartingAnchorRoot: stateAnchorRoot,
			}},
			expectedPrestate:   statePrestate,
			expectedAnchorRoot: stateAnchorRoot,
		},
		{
			name: "permissioned ignores state",
			chainDeployOverride: map[string]any{
				"faultGameAbsolutePrestate": overridePrestate,
			},
			chains: []*state.ChainState{{
				ID:                 chainID,
				Prestate:           ignoredStatePrestate,
				StartingAnchorRoot: ignoredStateAnchorRoot,
			}},
			expectedPrestate:   overridePrestate,
			expectedAnchorRoot: opcm.DefaultStartingAnchorRoot.Root,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			intent, chainIntent, st := makeDCITestInputs(chainID)
			intent.GlobalDeployOverrides = test.globalDeployOverride
			chainIntent.DeployOverrides = test.chainDeployOverride
			st.Chains = test.chains

			got, err := makeDCI(intent, chainIntent, chainID, st)
			if test.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErrMsg)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expectedPrestate, got.DisputeAbsolutePrestate)
			require.Equal(t, test.expectedAnchorRoot, got.StartingAnchorRoot)
		})
	}
}

func makeDCITestInputs(chainID common.Hash) (*state.Intent, *state.ChainIntent, *state.State) {
	chainIntent := &state.ChainIntent{
		ID: chainID,
		Roles: state.ChainRoles{
			L1ProxyAdminOwner: common.HexToAddress("0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"),
			SystemConfigOwner: common.HexToAddress("0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"),
			Batcher:           common.HexToAddress("0xCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC"),
			UnsafeBlockSigner: common.HexToAddress("0xDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"),
			Proposer:          common.HexToAddress("0xEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE"),
			Challenger:        common.HexToAddress("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		},
		GasLimit: 60_000_000,
	}
	return &state.Intent{}, chainIntent, &state.State{
		Create2Salt: common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234"),
		SuperchainDeployment: &addresses.SuperchainContracts{
			SuperchainConfigProxy: common.HexToAddress("0x3333333333333333333333333333333333333333"),
		},
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			OpcmV2Impl: common.HexToAddress("0x2222222222222222222222222222222222222222"),
		},
	}
}

func TestDeployOPChain_WithForge(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
	require.NoError(t, err)

	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
	require.NoError(t, err)

	_, afacts := testutil.LocalArtifacts(t)
	lgr := testlog.Logger(t, slog.LevelInfo)
	anvil, err := devnet.NewAnvil(lgr)
	require.NoError(t, err)
	require.NoError(t, anvil.Start())
	t.Cleanup(func() {
		require.NoError(t, anvil.Stop())
	})

	l1RPCUrl := anvil.RPCUrl()
	privateKey := "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	l1RPC, err := rpc.Dial(l1RPCUrl)
	require.NoError(t, err)
	l1Client := ethclient.NewClient(l1RPC)

	host, err := env.DefaultScriptHost(
		broadcaster.NoopBroadcaster(),
		lgr,
		common.Address{'D'},
		afacts,
		script.WithForkHook(func(cfg *script.ForkConfig) (forking.ForkSource, error) {
			src, err := forking.RPCSourceByNumber(cfg.URLOrAlias, l1RPC, *cfg.BlockNumber)
			if err != nil {
				return nil, fmt.Errorf("failed to create RPC fork source: %w", err)
			}
			return forking.Cache(src), nil
		}),
	)
	require.NoError(t, err)

	latest, err := l1Client.HeaderByNumber(ctx, nil)
	require.NoError(t, err)

	_, err = host.CreateSelectFork(
		script.ForkWithURLOrAlias("main"),
		script.ForkWithBlockNumberU256(latest.Number),
	)
	require.NoError(t, err)

	// Load scripts
	opcmScripts := &opcm.Scripts{}

	chainID := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000300")
	salt := common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")

	// Create test input
	intent := &state.Intent{
		GlobalDeployOverrides: make(map[string]any),
		Chains: []*state.ChainIntent{
			{
				ID: chainID,
				Roles: state.ChainRoles{
					L1ProxyAdminOwner: common.Address{'A'},
					SystemConfigOwner: common.Address{'B'},
					Batcher:           common.Address{'C'},
					UnsafeBlockSigner: common.Address{'D'},
					Proposer:          common.Address{'E'},
					Challenger:        common.Address{'F'},
				},
				GasLimit: 60_000_000,
			},
		},
		SuperchainRoles: &addresses.SuperchainRoles{
			SuperchainProxyAdminOwner: common.Address{'S'},
			SuperchainGuardian:        common.Address{'G'},
			Challenger:                common.HexToAddress("0xEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE"),
		},
	}

	st := &state.State{
		Version:     1,
		Create2Salt: salt,
	}

	pEnv := &Env{
		Logger:       lgr,
		Scripts:      opcmScripts,
		ForgeClient:  forgeClient,
		UseForge:     true,
		Context:      ctx,
		Broadcaster:  broadcaster.NoopBroadcaster(),
		StateWriter:  NoopStateWriter(),
		L1ScriptHost: host,
		L1RPCUrl:     l1RPCUrl,
		PrivateKey:   privateKey,
	}

	err = DeploySuperchain(pEnv, intent, st)
	require.NoError(t, err)

	err = DeployImplementations(pEnv, intent, st)
	require.NoError(t, err)

	err = DeployOPChain(pEnv, intent, st, chainID)
	require.NoError(t, err)

	require.Len(t, st.Chains, 1)
	require.Equal(t, chainID, st.Chains[0].ID)

	chainState := st.Chains[0]
	require.NotEqual(t, common.Address{}, chainState.OpChainContracts.OpChainProxyAdminImpl)
	require.NotEqual(t, common.Address{}, chainState.OpChainContracts.AddressManagerImpl)
	require.NotEqual(t, common.Address{}, chainState.OpChainContracts.L1Erc721BridgeProxy)
	require.NotEqual(t, common.Address{}, chainState.OpChainContracts.SystemConfigProxy)
	require.NotEqual(t, common.Address{}, chainState.OpChainContracts.OptimismMintableErc20FactoryProxy)
	require.NotEqual(t, common.Address{}, chainState.OpChainContracts.L1StandardBridgeProxy)
	require.NotEqual(t, common.Address{}, chainState.OpChainContracts.L1CrossDomainMessengerProxy)
}
