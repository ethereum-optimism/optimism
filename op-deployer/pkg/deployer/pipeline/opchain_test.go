package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/forking"
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
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/lmittmann/w3"
	"github.com/stretchr/testify/require"
)

func Test_makeDCI_OpcmAddress(t *testing.T) {
	opcmV2Addr := common.HexToAddress("0x2222222222222222222222222222222222222222")
	chainID := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000300")
	salt := common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234")
	superchainConfig := common.HexToAddress("0x3333333333333333333333333333333333333333")

	baseIntent := &state.Intent{
		GlobalDeployOverrides: make(map[string]any),
	}

	baseChainIntent := &state.ChainIntent{
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
			name:       "uses_opcm_v2",
			intent:     baseIntent,
			thisIntent: baseChainIntent,
			chainID:    chainID,
			st: &state.State{
				Create2Salt: salt,
				SuperchainDeployment: &addresses.SuperchainContracts{
					SuperchainConfigProxy: superchainConfig,
				},
				ImplementationsDeployment: &addresses.ImplementationsContracts{
					OpcmV2Impl: opcmV2Addr,
				},
			},
			expectedOpcm:   opcmV2Addr,
			shouldThrowErr: false,
			expectedErrMsg: "",
		},
		{
			name:       "opcm_v2_impl_zero_reverts",
			intent:     baseIntent,
			thisIntent: baseChainIntent,
			chainID:    chainID,
			st: &state.State{
				Create2Salt: salt,
				SuperchainDeployment: &addresses.SuperchainContracts{
					SuperchainConfigProxy: superchainConfig,
				},
				ImplementationsDeployment: &addresses.ImplementationsContracts{
					OpcmV2Impl: common.Address{}, // zero address
				},
			},
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
			require.Equal(t, standard.DisputeAbsolutePrestate, got.CannonAbsolutePrestate)
			require.Equal(t, opcm.DefaultStartingAnchorRoot.Root, got.StartingAnchorRoot.Root)
			require.Equal(t, common.Big0, got.StartingAnchorRoot.L2SequenceNumber)
		})
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

	startingAnchorRoot := opcm.Proposal{
		Root:             common.HexToHash("0x02f4397b2de6fce03b3f9982378c2b4c4deff9c92c662dcc6f9643267aeb5e47"),
		L2SequenceNumber: big.NewInt(1234),
	}
	forgeOutput, err := opcm.DeployOPChainViaForge(&opcm.ForgeEnv{
		Client:     forgeClient,
		Context:    ctx,
		L1RPCUrl:   l1RPCUrl,
		PrivateKey: privateKey,
	}, opcm.DeployOPChainInput{
		OpChainProxyAdminOwner:       common.Address{'A'},
		SystemConfigOwner:            common.Address{'B'},
		Batcher:                      common.Address{'C'},
		UnsafeBlockSigner:            common.Address{'D'},
		Proposer:                     common.Address{'E'},
		Challenger:                   common.Address{'F'},
		BasefeeScalar:                standard.BasefeeScalar,
		BlobBaseFeeScalar:            standard.BlobBaseFeeScalar,
		L2ChainId:                    new(big.Int).Add(chainID.Big(), big.NewInt(1)),
		Opcm:                         st.ImplementationsDeployment.OpcmV2Impl,
		SaltMixer:                    "starting-anchor-root-regression",
		GasLimit:                     60_000_000,
		DisputeGameType:              standard.DisputeGameType,
		DisputeAbsolutePrestate:      standard.DisputeAbsolutePrestate,
		StartingAnchorRoot:           startingAnchorRoot,
		CannonAbsolutePrestate:       standard.DisputeAbsolutePrestate,
		DisputeMaxGameDepth:          new(big.Int).SetUint64(standard.DisputeMaxGameDepth),
		DisputeSplitDepth:            new(big.Int).SetUint64(standard.DisputeSplitDepth),
		DisputeClockExtension:        standard.DisputeClockExtension,
		DisputeMaxClockDuration:      standard.DisputeMaxClockDuration,
		AllowCustomDisputeParameters: false,
		OperatorFeeScalar:            0,
		OperatorFeeConstant:          0,
		SuperchainConfig:             st.SuperchainDeployment.SuperchainConfigProxy,
		UseCustomGasToken:            false,
	})
	require.NoError(t, err)

	getStartingAnchorRoot := w3.MustNewFunc("getStartingAnchorRoot()", "(bytes32 root, uint256 l2SequenceNumber)")
	callData, err := getStartingAnchorRoot.EncodeArgs()
	require.NoError(t, err)
	result, err := l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &forgeOutput.AnchorStateRegistryProxy,
		Data: callData,
	}, nil)
	require.NoError(t, err)
	var actualAnchorRoot opcm.Proposal
	require.NoError(t, getStartingAnchorRoot.DecodeReturns(result, &actualAnchorRoot))
	require.Equal(t, startingAnchorRoot.Root, actualAnchorRoot.Root)
	require.Equal(t, startingAnchorRoot.L2SequenceNumber, actualAnchorRoot.L2SequenceNumber)

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
