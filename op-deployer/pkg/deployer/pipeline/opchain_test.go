package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"math"
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
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
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

func Test_makeDCI_OwnsStartingAnchorSequenceNumber(t *testing.T) {
	chainID := common.HexToHash("0x0300")
	intent := &state.Intent{GlobalDeployOverrides: make(map[string]any)}
	chainIntent := &state.ChainIntent{ID: chainID}
	st := &state.State{
		SuperchainDeployment: &addresses.SuperchainContracts{},
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			OpcmV2Impl: common.HexToAddress("0x01"),
		},
	}

	first, err := makeDCI(intent, chainIntent, chainID, st)
	require.NoError(t, err)
	second, err := makeDCI(intent, chainIntent, chainID, st)
	require.NoError(t, err)

	require.Zero(t, first.StartingAnchorRoot.L2SequenceNumber.Sign())
	require.Zero(t, second.StartingAnchorRoot.L2SequenceNumber.Sign())
	require.NotSame(t, first.StartingAnchorRoot.L2SequenceNumber, second.StartingAnchorRoot.L2SequenceNumber)

	first.StartingAnchorRoot.L2SequenceNumber.SetUint64(1)
	require.Zero(t, second.StartingAnchorRoot.L2SequenceNumber.Sign())
}

func Test_makeDCI_RejectsPermissionlessGameType(t *testing.T) {
	chainID := common.HexToHash("0x0300")
	intent := &state.Intent{GlobalDeployOverrides: make(map[string]any)}
	st := &state.State{
		Create2Salt: common.HexToHash("0x01"),
		SuperchainDeployment: &addresses.SuperchainContracts{
			SuperchainConfigProxy: common.HexToAddress("0x3333333333333333333333333333333333333333"),
		},
		ImplementationsDeployment: &addresses.ImplementationsContracts{
			OpcmV2Impl: common.HexToAddress("0x2222222222222222222222222222222222222222"),
		},
	}

	tests := []struct {
		name     string
		gameType embedded.GameType
	}{
		{name: "CANNON_KONA", gameType: embedded.GameTypeCannonKona},
		{name: "SUPER_CANNON_KONA", gameType: embedded.GameTypeSuperCannonKona},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainIntent := &state.ChainIntent{
				ID:              chainID,
				DeployOverrides: map[string]any{"respectedGameType": tt.gameType},
			}

			_, err := makeDCI(intent, chainIntent, chainID, st)
			require.ErrorContains(t, err, "permissionless")
		})
	}
}

func Test_makeDCI_RejectsInvalidInitialGameTypeBeforePermissionlessHandling(t *testing.T) {
	chainID := common.HexToHash("0x0300")
	intent := &state.Intent{GlobalDeployOverrides: make(map[string]any)}

	tests := []struct {
		name     string
		gameType uint32
		wantErr  string
	}{
		{
			name:     "CANNON",
			gameType: uint32(embedded.GameTypeCannon),
			wantErr:  "unsupported initial dispute game type 0",
		},
		{
			name:     "SUPER_PERMISSIONED",
			gameType: uint32(embedded.GameTypeSuperPermissioned),
			wantErr:  "derived fallback and is not an initial-deploy selector",
		},
		{
			name:     "ZK_DISPUTE_GAME",
			gameType: uint32(embedded.GameTypeZKDisputeGame),
			wantErr:  "unsupported initial dispute game type 10",
		},
		{
			name:     "unknown",
			gameType: math.MaxUint32,
			wantErr:  fmt.Sprintf("unsupported initial dispute game type %d", uint32(math.MaxUint32)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainIntent := &state.ChainIntent{
				ID:              chainID,
				DeployOverrides: map[string]any{"respectedGameType": tt.gameType},
			}

			_, err := makeDCI(intent, chainIntent, chainID, &state.State{})
			require.ErrorContains(t, err, tt.wantErr)
			require.NotContains(t, err.Error(), "apply only supports permissioned deploys")
		})
	}
}

func TestResolveChainProofParams(t *testing.T) {
	t.Run("uses defaults", func(t *testing.T) {
		got, err := ResolveChainProofParams(&state.Intent{}, &state.ChainIntent{})
		require.NoError(t, err)
		require.Equal(t, state.ChainProofParams{
			DisputeGameType:         standard.DisputeGameType,
			DisputeAbsolutePrestate: standard.DisputeAbsolutePrestate,
			DisputeMaxGameDepth:     standard.DisputeMaxGameDepth,
			DisputeSplitDepth:       standard.DisputeSplitDepth,
			DisputeClockExtension:   standard.DisputeClockExtension,
			DisputeMaxClockDuration: standard.DisputeMaxClockDuration,
		}, got)
	})

	t.Run("chain overrides global", func(t *testing.T) {
		globalPrestate := common.HexToHash("0x11")
		intent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"respectedGameType":         embedded.GameTypeCannonKona,
			"faultGameAbsolutePrestate": globalPrestate,
			"faultGameMaxDepth":         uint64(101),
		}}
		chain := &state.ChainIntent{DeployOverrides: map[string]any{
			"respectedGameType": embedded.GameTypeSuperCannonKona,
			"faultGameMaxDepth": uint64(202),
		}}

		got, err := ResolveChainProofParams(intent, chain)
		require.NoError(t, err)
		require.Equal(t, uint32(embedded.GameTypeSuperCannonKona), got.DisputeGameType)
		require.Equal(t, globalPrestate, got.DisputeAbsolutePrestate)
		require.Equal(t, uint64(202), got.DisputeMaxGameDepth)
	})

	t.Run("rejects malformed absolute prestate", func(t *testing.T) {
		intent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"faultGameAbsolutePrestate": "not-a-hash",
		}}

		_, err := ResolveChainProofParams(intent, &state.ChainIntent{})
		require.Error(t, err)
	})
}

func TestResolvePreparedGameType(t *testing.T) {
	chainID := common.HexToHash("0x01")

	t.Run("requires recorded type", func(t *testing.T) {
		_, err := ResolvePreparedGameType(
			&state.Intent{},
			&state.ChainIntent{ID: chainID},
			&state.ChainState{ID: chainID},
		)
		require.ErrorContains(t, err, "no initial game type recorded by prepare")
		require.ErrorContains(t, err, "rerun op-deployer prepare")
	})

	for _, gameType := range []embedded.GameType{
		embedded.GameTypePermissionedCannon,
		embedded.GameTypeSuperPermissioned,
		embedded.GameTypeCannonKona,
		embedded.GameTypeSuperCannonKona,
	} {
		t.Run(initialGameTypeName(uint32(gameType)), func(t *testing.T) {
			intent := &state.Intent{GlobalDeployOverrides: make(map[string]any)}
			chain := &state.ChainIntent{
				ID:              chainID,
				DeployOverrides: map[string]any{"respectedGameType": gameType},
			}
			chainState := &state.ChainState{
				ID:              chainID,
				InitialGameType: ptr.New(uint32(gameType)),
			}

			got, err := ResolvePreparedGameType(intent, chain, chainState)
			require.NoError(t, err)
			require.Equal(t, uint32(gameType), got)
		})
	}

	t.Run("rejects drift with names and numbers", func(t *testing.T) {
		intent := &state.Intent{GlobalDeployOverrides: make(map[string]any)}
		chain := &state.ChainIntent{
			ID:              chainID,
			DeployOverrides: map[string]any{"respectedGameType": embedded.GameTypeCannonKona},
		}
		chainState := &state.ChainState{
			ID:              chainID,
			InitialGameType: ptr.New(uint32(embedded.GameTypePermissionedCannon)),
		}

		_, err := ResolvePreparedGameType(intent, chain, chainState)
		require.ErrorContains(t, err, "prepared PERMISSIONED_CANNON (1)")
		require.ErrorContains(t, err, "intent CANNON_KONA (8)")
		require.ErrorContains(t, err, "rerun op-deployer prepare")
	})

	t.Run("allows non-type proof parameter changes", func(t *testing.T) {
		intent := &state.Intent{GlobalDeployOverrides: map[string]any{
			"respectedGameType":         embedded.GameTypeCannonKona,
			"faultGameAbsolutePrestate": common.HexToHash("0x22"),
		}}
		chain := &state.ChainIntent{ID: chainID}
		chainState := &state.ChainState{
			ID:              chainID,
			InitialGameType: ptr.New(uint32(embedded.GameTypeCannonKona)),
		}

		got, err := ResolvePreparedGameType(intent, chain, chainState)
		require.NoError(t, err)
		require.Equal(t, uint32(embedded.GameTypeCannonKona), got)
	})
}

func TestResolveInitialDeployRequirements(t *testing.T) {
	tests := []struct {
		name     string
		gameType uint32
		want     InitialDeployRequirements
		wantErr  string
	}{
		{
			name:     "CANNON",
			gameType: uint32(embedded.GameTypeCannon),
			wantErr:  "unsupported initial dispute game type 0",
		},
		{
			name:     "PERMISSIONED_CANNON",
			gameType: uint32(embedded.GameTypePermissionedCannon),
			want:     InitialDeployRequirements{},
		},
		{
			name:     "SUPER_PERMISSIONED",
			gameType: uint32(embedded.GameTypeSuperPermissioned),
			wantErr:  "SUPER_PERMISSIONED (5) is a derived fallback and is not an initial-deploy selector",
		},
		{
			name:     "CANNON_KONA",
			gameType: uint32(embedded.GameTypeCannonKona),
			want: InitialDeployRequirements{
				Permissionless:   true,
				RequiresPrestate: true,
			},
		},
		{
			name:     "SUPER_CANNON_KONA",
			gameType: uint32(embedded.GameTypeSuperCannonKona),
			want: InitialDeployRequirements{
				Permissionless:   true,
				RequiresPrestate: true,
			},
		},
		{
			name:     "ZK_DISPUTE_GAME",
			gameType: uint32(embedded.GameTypeZKDisputeGame),
			wantErr:  "unsupported initial dispute game type 10",
		},
		{
			name:     "unknown",
			gameType: math.MaxUint32,
			wantErr:  fmt.Sprintf("unsupported initial dispute game type %d", uint32(math.MaxUint32)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveInitialDeployRequirements(tt.gameType)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Zero(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			if got.Permissionless {
				require.True(t, got.RequiresPrestate)
			}
		})
	}
}

func TestValidateInitialGameTypeSet(t *testing.T) {
	tests := []struct {
		name      string
		gameTypes []uint32
		wantErr   string
	}{
		{name: "empty"},
		{
			name:      "all CANNON_KONA",
			gameTypes: []uint32{uint32(embedded.GameTypeCannonKona), uint32(embedded.GameTypeCannonKona)},
		},
		{
			name:      "all SUPER_CANNON_KONA",
			gameTypes: []uint32{uint32(embedded.GameTypeSuperCannonKona), uint32(embedded.GameTypeSuperCannonKona)},
		},
		{
			name: "mixed",
			gameTypes: []uint32{
				uint32(embedded.GameTypeCannonKona),
				uint32(embedded.GameTypeSuperCannonKona),
			},
			wantErr: "an intent cannot mix CANNON_KONA and SUPER_CANNON_KONA initial games",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInitialGameTypeSet(tt.gameTypes)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestBuildDeployOPChainInputCannonAbsolutePrestate(t *testing.T) {
	selectedPrestate := common.HexToHash("0x1234")
	tests := []struct {
		name     string
		gameType embedded.GameType
		want     common.Hash
	}{
		{
			name:     "PERMISSIONED_CANNON mirrors selected prestate",
			gameType: embedded.GameTypePermissionedCannon,
			want:     selectedPrestate,
		},
		{
			name:     "CANNON_KONA uses canonical fallback",
			gameType: embedded.GameTypeCannonKona,
			want:     opcm.PermissionedCannonFallbackPrestatePlaceholder,
		},
		{
			name:     "SUPER_CANNON_KONA leaves unread field zero",
			gameType: embedded.GameTypeSuperCannonKona,
			want:     common.Hash{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proofParams := state.ChainProofParams{
				DisputeGameType:         uint32(tt.gameType),
				DisputeAbsolutePrestate: selectedPrestate,
			}
			got := BuildDeployOPChainInput(
				proofParams,
				state.ChainRoles{},
				common.Address{},
				common.Address{},
				common.Hash{},
				"",
				0,
				opcm.Proposal{},
				&state.ChainIntent{},
			)

			require.Equal(t, uint32(tt.gameType), got.DisputeGameType)
			require.Equal(t, selectedPrestate, got.DisputeAbsolutePrestate)
			require.Equal(t, tt.want, got.CannonAbsolutePrestate)
			if tt.gameType == embedded.GameTypeCannonKona {
				require.NotEqual(t, got.DisputeAbsolutePrestate, got.CannonAbsolutePrestate)
			}
		})
	}
}

func TestShouldDeployOPChain(t *testing.T) {
	chainID := common.HexToHash("0x0a")
	other := common.HexToHash("0x0b")

	t.Run("absent chain is deployed", func(t *testing.T) {
		require.True(t, shouldDeployOPChain(&state.State{}, chainID))
	})

	t.Run("deployed chain is skipped", func(t *testing.T) {
		st := &state.State{Chains: []*state.ChainState{{ID: chainID, Deployed: ptr.New(true)}}}
		require.False(t, shouldDeployOPChain(st, chainID))
	})

	t.Run("predicted-only chain is still deployed", func(t *testing.T) {
		// prepare writes the chain with Deployed=false; apply/continue must still
		// broadcast it.
		st := &state.State{Chains: []*state.ChainState{{ID: chainID, Deployed: ptr.New(false)}}}
		require.True(t, shouldDeployOPChain(st, chainID))
	})

	t.Run("legacy chain without deployed flag is skipped", func(t *testing.T) {
		// States written by older pipelines have no Deployed field, so it decodes
		// to nil. Those chains were already deployed.
		st := &state.State{Chains: []*state.ChainState{{ID: chainID, Deployed: nil}}}
		require.False(t, shouldDeployOPChain(st, chainID))
	})

	t.Run("only matches the requested chain id", func(t *testing.T) {
		st := &state.State{Chains: []*state.ChainState{{ID: other, Deployed: ptr.New(true)}}}
		require.True(t, shouldDeployOPChain(st, chainID))
	})
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
