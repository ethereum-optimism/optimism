package pipeline

import (
	"context"
	"encoding/json"
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

func TestExecuteOPChainDeploymentRejectsInvalidChainIDBeforeExecution(t *testing.T) {
	chainID := common.HexToHash("0x0300")
	otherChainID := common.HexToHash("0x0301")

	tests := []struct {
		name       string
		l2ChainID  *big.Int
		wantErrMsg string
	}{
		{
			name:       "nil",
			l2ChainID:  nil,
			wantErrMsg: "nil L2 chain ID",
		},
		{
			name:       "mismatch",
			l2ChainID:  otherChainID.Big(),
			wantErrMsg: "does not match requested chain ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dci opcm.DeployOPChainInput
			dci.L2ChainId = tt.l2ChainID

			expectedState := &state.State{
				Chains: []*state.ChainState{{
					ID:       chainID,
					Deployed: ptr.New(false),
				}},
			}
			st := &state.State{
				Chains: []*state.ChainState{{
					ID:       chainID,
					Deployed: ptr.New(false),
				}},
			}

			result, err := ExecuteOPChainDeployment(&Env{}, st, chainID, dci)
			require.ErrorContains(t, err, tt.wantErrMsg)
			require.Equal(t, OPChainDeploymentResult{}, result)
			require.Equal(t, expectedState, st)
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

func TestRecordOPChainDeploymentRejectsUninitializedResult(t *testing.T) {
	var result OPChainDeploymentResult
	err := RecordOPChainDeployment(&state.State{}, result)
	require.ErrorContains(t, err, "uninitialized OP chain deployment result")
}

func TestRecordOPChainDeploymentRecordsBoundChainWithoutImplementations(t *testing.T) {
	chainID := common.HexToHash("0x0300")
	otherChainID := common.HexToHash("0x0301")

	var contracts addresses.OpChainContracts
	contracts.SystemConfigProxy = common.HexToAddress("0x1111111111111111111111111111111111111111")

	var readback opcm.ReadImplementationAddressesOutput
	readback.SystemConfig = common.HexToAddress("0x2222222222222222222222222222222222222222")

	result := OPChainDeploymentResult{
		chainID:     chainID,
		contracts:   contracts,
		readback:    readback,
		initialized: true,
	}
	st := &state.State{
		Chains: []*state.ChainState{
			{
				ID:       chainID,
				Deployed: ptr.New(false),
				Prestate: common.HexToHash("0x33"),
			},
			{
				ID:       otherChainID,
				Deployed: ptr.New(false),
			},
		},
		ImplementationsDeployment: nil,
	}

	require.NoError(t, RecordOPChainDeployment(st, result))
	boundChain, err := st.Chain(chainID)
	require.NoError(t, err)
	require.Equal(t, contracts, boundChain.OpChainContracts)
	require.True(t, *boundChain.Deployed)
	require.Equal(t, common.HexToHash("0x33"), boundChain.Prestate)

	otherChain, err := st.Chain(otherChainID)
	require.NoError(t, err)
	require.False(t, *otherChain.Deployed)
	var emptyContracts addresses.OpChainContracts
	require.Equal(t, emptyContracts, otherChain.OpChainContracts)
	require.Nil(t, st.ImplementationsDeployment)

	afterFirstRecord, err := json.Marshal(st)
	require.NoError(t, err)
	require.NoError(t, RecordOPChainDeployment(st, result))
	afterSecondRecord, err := json.Marshal(st)
	require.NoError(t, err)
	require.Equal(t, afterFirstRecord, afterSecondRecord)
}

func TestRecordOPChainDeploymentAppliesImplementationReadback(t *testing.T) {
	chainID := common.HexToHash("0x0300")

	var readback opcm.ReadImplementationAddressesOutput
	readback.SystemConfig = common.HexToAddress("0x1111111111111111111111111111111111111111")
	readback.SuperFaultDisputeGame = common.HexToAddress("0x2222222222222222222222222222222222222222")

	var contracts addresses.OpChainContracts
	result := OPChainDeploymentResult{
		chainID:     chainID,
		contracts:   contracts,
		readback:    readback,
		initialized: true,
	}
	var implementations addresses.ImplementationsContracts
	st := &state.State{ImplementationsDeployment: &implementations}

	require.NoError(t, RecordOPChainDeployment(st, result))
	require.Equal(t, readback.SystemConfig, st.ImplementationsDeployment.SystemConfigImpl)
	require.Equal(t, readback.SuperFaultDisputeGame, st.ImplementationsDeployment.SuperFaultDisputeGameImpl)
}

func TestBuildContinuationDCI_PermissionlessInputs(t *testing.T) {
	chainID := common.HexToHash("0x0300")

	tests := []struct {
		name             string
		gameType         embedded.GameType
		expectedFallback common.Hash
	}{
		{
			name:             "CANNON_KONA",
			gameType:         embedded.GameTypeCannonKona,
			expectedFallback: opcm.PermissionedCannonFallbackPrestatePlaceholder,
		},
		{
			name:             "SUPER_CANNON_KONA",
			gameType:         embedded.GameTypeSuperCannonKona,
			expectedFallback: common.Hash{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, chain, st := continuationDCITestInputs(chainID, tt.gameType)

			got, err := BuildContinuationDCI(intent, chainID, st)
			require.NoError(t, err)
			require.Equal(t, uint32(tt.gameType), got.DisputeGameType)
			require.Equal(t, st.Chains[0].Prestate, got.DisputeAbsolutePrestate)
			require.Equal(t, st.Chains[0].StartingAnchorRoot.Root, got.StartingAnchorRoot.Root)
			require.Equal(t, big.NewInt(42), got.StartingAnchorRoot.L2SequenceNumber)
			require.Equal(t, tt.expectedFallback, got.CannonAbsolutePrestate)
			require.Equal(t, *st.L1PredictOPCMAddress, got.Opcm)
			require.NotEqual(t, *intent.OPCMAddress, got.Opcm)
			require.Equal(t, *intent.SuperchainConfigProxy, got.SuperchainConfig)
			require.Equal(t, chain.Roles.L1ProxyAdminOwner, got.OpChainProxyAdminOwner)
			require.Equal(t, chain.Roles.SystemConfigOwner, got.SystemConfigOwner)
			require.Equal(t, chainID.Big(), got.L2ChainId)
			require.Equal(t, st.Create2Salt.String(), got.SaltMixer)
			require.Nil(t, st.ImplementationsDeployment)
			require.Nil(t, st.SuperchainDeployment)
		})
	}
}

func TestBuildContinuationDCI_PermissionedInputs(t *testing.T) {
	chainID := common.HexToHash("0x0300")
	intent, chain, st := continuationDCITestInputs(chainID, embedded.GameTypePermissionedCannon)
	proofPrestate := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	chain.DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = proofPrestate

	st.Chains[0].Prestate = common.Hash{}
	st.Chains[0].StartingAnchorRoot = nil

	got, err := BuildContinuationDCI(intent, chainID, st)
	require.NoError(t, err)
	second, err := BuildContinuationDCI(intent, chainID, st)
	require.NoError(t, err)
	require.Equal(t, uint32(embedded.GameTypePermissionedCannon), got.DisputeGameType)
	require.Equal(t, proofPrestate, got.DisputeAbsolutePrestate)
	require.Equal(t, proofPrestate, got.CannonAbsolutePrestate)
	require.Equal(t, opcm.DefaultStartingAnchorRoot.Root, got.StartingAnchorRoot.Root)
	require.Zero(t, got.StartingAnchorRoot.L2SequenceNumber.Sign())
	require.Zero(t, second.StartingAnchorRoot.L2SequenceNumber.Sign())
	require.NotSame(t, got.StartingAnchorRoot.L2SequenceNumber, second.StartingAnchorRoot.L2SequenceNumber)
	got.StartingAnchorRoot.L2SequenceNumber.SetUint64(1)
	require.Zero(t, second.StartingAnchorRoot.L2SequenceNumber.Sign())
	require.Equal(t, *st.L1PredictOPCMAddress, got.Opcm)
	require.NotEqual(t, *intent.OPCMAddress, got.Opcm)
	require.Equal(t, *intent.SuperchainConfigProxy, got.SuperchainConfig)
	require.Nil(t, st.ImplementationsDeployment)
	require.Nil(t, st.SuperchainDeployment)
}

func TestBuildContinuationDCI_FailClosedGates(t *testing.T) {
	chainID := common.HexToHash("0x0300")
	otherChainID := common.HexToHash("0x0301")

	tests := []struct {
		name       string
		mutate     func(*state.Intent, *state.ChainIntent, *state.State)
		wantErrors []string
	}{
		{
			name: "gate 1 requires prepared state",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.Prepared = false
			},
			wantErrors: []string{"op-deployer prepare"},
		},
		{
			name: "gate 2 requires CREATE2 salt",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.Create2Salt = common.Hash{}
			},
			wantErrors: []string{"CREATE2 salt", "op-deployer prepare"},
		},
		{
			name: "gate 3 requires sender pin",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.L1PredictSenderAddress = nil
			},
			wantErrors: []string{"predicted sender", "op-deployer prepare"},
		},
		{
			name: "gate 3 rejects zero sender pin",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.L1PredictSenderAddress = ptr.New(common.Address{})
			},
			wantErrors: []string{"predicted sender", "op-deployer prepare"},
		},
		{
			name: "gate 4 requires OPCM pin",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.L1PredictOPCMAddress = nil
			},
			wantErrors: []string{"predicted OPCM", "op-deployer prepare"},
		},
		{
			name: "gate 4 rejects zero OPCM pin",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.L1PredictOPCMAddress = ptr.New(common.Address{})
			},
			wantErrors: []string{"predicted OPCM", "op-deployer prepare"},
		},
		{
			name: "gate 5 requires superchain config",
			mutate: func(intent *state.Intent, _ *state.ChainIntent, _ *state.State) {
				intent.SuperchainConfigProxy = nil
			},
			wantErrors: []string{"intent.superchainConfigProxy must be set"},
		},
		{
			name: "gate 5 rejects zero superchain config",
			mutate: func(intent *state.Intent, _ *state.ChainIntent, _ *state.State) {
				intent.SuperchainConfigProxy = ptr.New(common.Address{})
			},
			wantErrors: []string{"intent.superchainConfigProxy must be set"},
		},
		{
			name: "gate 6 binds chain state",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.Chains[0].ID = otherChainID
			},
			wantErrors: []string{"state prepared for chain", "chain not found", "op-deployer prepare"},
		},
		{
			name: "gate 6 binds chain intent",
			mutate: func(intent *state.Intent, _ *state.ChainIntent, _ *state.State) {
				intent.Chains[0].ID = otherChainID
			},
			wantErrors: []string{"failed to get chain intent", "not found"},
		},
		{
			name: "gate 7 requires prepared game type",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.Chains[0].InitialGameType = nil
			},
			wantErrors: []string{"no initial game type recorded by prepare", "op-deployer prepare"},
		},
		{
			name: "gate 7 rejects game type drift",
			mutate: func(_ *state.Intent, chain *state.ChainIntent, _ *state.State) {
				chain.DeployOverrides["respectedGameType"] = embedded.GameTypeSuperCannonKona
			},
			wantErrors: []string{"initial game type changed after prepare", "op-deployer prepare"},
		},
		{
			name: "gate 8 rejects unknown prepared game type",
			mutate: func(_ *state.Intent, chain *state.ChainIntent, st *state.State) {
				const unknownGameType = uint32(999)
				chain.DeployOverrides["respectedGameType"] = unknownGameType
				st.Chains[0].InitialGameType = ptr.New(unknownGameType)
			},
			wantErrors: []string{"unsupported initial dispute game type 999", "op-deployer prepare"},
		},
		{
			name: "gate 8 rejects SUPER_PERMISSIONED selector",
			mutate: func(_ *state.Intent, chain *state.ChainIntent, st *state.State) {
				chain.DeployOverrides["respectedGameType"] = embedded.GameTypeSuperPermissioned
				st.Chains[0].InitialGameType = ptr.New(uint32(embedded.GameTypeSuperPermissioned))
			},
			wantErrors: []string{"derived fallback and is not an initial-deploy selector", "op-deployer prepare"},
		},
		{
			name: "gate 9 requires committed prestate",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.Chains[0].Prestate = common.Hash{}
			},
			wantErrors: []string{"no prestate committed", "op-deployer prestate"},
		},
		{
			name: "gate 10 rejects reserved prestate",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.Chains[0].Prestate = opcm.PermissionedCannonFallbackPrestatePlaceholder
			},
			wantErrors: []string{"reserved permissioned prestate placeholder", "op-deployer prestate"},
		},
		{
			name: "gate 11 rejects prestate override drift",
			mutate: func(_ *state.Intent, chain *state.ChainIntent, _ *state.State) {
				chain.DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = common.HexToHash("0x99")
			},
			wantErrors: []string{"override differs from the committed prestate", "op-deployer prestate"},
		},
		{
			name: "gate 12 requires starting anchor proposal",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.Chains[0].StartingAnchorRoot = nil
			},
			wantErrors: []string{"starting anchor proposal", "proposal-producing stage"},
		},
		{
			name: "gate 12 rejects zero starting anchor root",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.Chains[0].StartingAnchorRoot.Root = common.Hash{}
			},
			wantErrors: []string{"starting anchor proposal", "proposal-producing stage"},
		},
		{
			name: "gate 12 rejects permissioned anchor placeholder",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.Chains[0].StartingAnchorRoot.Root = opcm.DefaultStartingAnchorRoot.Root
			},
			wantErrors: []string{"permissioned starting anchor placeholder", "proposal-producing stage"},
		},
		{
			name: "gate 12 rejects maximum anchor sequence",
			mutate: func(_ *state.Intent, _ *state.ChainIntent, st *state.State) {
				st.Chains[0].StartingAnchorRoot.L2SequenceNumber = math.MaxUint64
			},
			wantErrors: []string{"starting anchor sequence number that is too large", "proposal-producing stage"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, chain, st := continuationDCITestInputs(chainID, embedded.GameTypeCannonKona)
			tt.mutate(intent, chain, st)

			_, err := BuildContinuationDCI(intent, chainID, st)
			require.Error(t, err)
			for _, want := range tt.wantErrors {
				require.ErrorContains(t, err, want)
			}
		})
	}
}

func TestBuildContinuationDCI_PrestateOverrideDrift(t *testing.T) {
	chainID := common.HexToHash("0x0300")

	tests := []struct {
		name       string
		configure  func(*state.Intent, *state.ChainIntent, common.Hash)
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "no override uses committed value instead of the default",
		},
		{
			name: "agreeing chain override",
			configure: func(_ *state.Intent, chain *state.ChainIntent, prestate common.Hash) {
				chain.DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = prestate
			},
		},
		{
			name: "agreeing global override",
			configure: func(intent *state.Intent, _ *state.ChainIntent, prestate common.Hash) {
				intent.GlobalDeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = prestate
			},
		},
		{
			name: "differing chain override",
			configure: func(_ *state.Intent, chain *state.ChainIntent, _ common.Hash) {
				chain.DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = common.HexToHash("0x99")
			},
			wantErr:    true,
			wantErrMsg: "op-deployer prestate",
		},
		{
			name: "differing global override",
			configure: func(intent *state.Intent, _ *state.ChainIntent, _ common.Hash) {
				intent.GlobalDeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = common.HexToHash("0x99")
			},
			wantErr:    true,
			wantErrMsg: "op-deployer prestate",
		},
		{
			name: "chain override shadows differing global override",
			configure: func(intent *state.Intent, chain *state.ChainIntent, prestate common.Hash) {
				intent.GlobalDeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = common.HexToHash("0x99")
				chain.DeployOverrides[state.FaultGameAbsolutePrestateOverrideKey] = prestate
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, chain, st := continuationDCITestInputs(chainID, embedded.GameTypeCannonKona)
			prestate := st.Chains[0].Prestate
			require.NotEqual(t, standard.DisputeAbsolutePrestate, prestate)
			if tt.configure != nil {
				tt.configure(intent, chain, prestate)
			}

			got, err := BuildContinuationDCI(intent, chainID, st)
			if tt.wantErr {
				require.ErrorContains(t, err, tt.wantErrMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, prestate, got.DisputeAbsolutePrestate)
		})
	}
}

func TestBuildContinuationDCI_LosslessAnchorSequenceTransport(t *testing.T) {
	chainID := common.HexToHash("0x0300")

	t.Run("uint64 max minus one is transported exactly", func(t *testing.T) {
		intent, _, st := continuationDCITestInputs(chainID, embedded.GameTypeCannonKona)
		st.Chains[0].StartingAnchorRoot.L2SequenceNumber = math.MaxUint64 - 1

		got, err := BuildContinuationDCI(intent, chainID, st)
		require.NoError(t, err)
		require.Equal(
			t,
			new(big.Int).SetUint64(math.MaxUint64-1),
			got.StartingAnchorRoot.L2SequenceNumber,
		)
	})

	t.Run("uint64 max is rejected by the generic transport bound", func(t *testing.T) {
		intent, _, st := continuationDCITestInputs(chainID, embedded.GameTypeCannonKona)
		st.Chains[0].StartingAnchorRoot.L2SequenceNumber = math.MaxUint64

		_, err := BuildContinuationDCI(intent, chainID, st)
		require.ErrorContains(t, err, "starting anchor sequence number that is too large")
	})
}

func continuationDCITestInputs(
	chainID common.Hash,
	gameType embedded.GameType,
) (*state.Intent, *state.ChainIntent, *state.State) {
	opcmAddr := common.HexToAddress("0x2222222222222222222222222222222222222222")
	intentOPCMAddr := common.HexToAddress("0x5555555555555555555555555555555555555555")
	superchainConfig := common.HexToAddress("0x3333333333333333333333333333333333333333")
	predictSender := common.HexToAddress("0x4444444444444444444444444444444444444444")
	chain := &state.ChainIntent{
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
		DeployOverrides: map[string]any{
			"respectedGameType": gameType,
		},
	}
	intent := &state.Intent{
		OPCMAddress:           &intentOPCMAddr,
		SuperchainConfigProxy: &superchainConfig,
		GlobalDeployOverrides: make(map[string]any),
		Chains:                []*state.ChainIntent{chain},
	}
	st := &state.State{
		Prepared:                  true,
		Create2Salt:               common.HexToHash("0x1234567890123456789012345678901234567890123456789012345678901234"),
		L1PredictSenderAddress:    &predictSender,
		L1PredictOPCMAddress:      &opcmAddr,
		ImplementationsDeployment: nil,
		SuperchainDeployment:      nil,
		Chains: []*state.ChainState{{
			ID:              chainID,
			Prestate:        common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
			InitialGameType: ptr.New(uint32(gameType)),
			StartingAnchorRoot: &state.StartingAnchorProposal{
				Root:             common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
				L2SequenceNumber: 42,
			},
		}},
	}
	return intent, chain, st
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
			"respectedGameType":                        embedded.GameTypeCannonKona,
			state.FaultGameAbsolutePrestateOverrideKey: globalPrestate,
			"faultGameMaxDepth":                        uint64(101),
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
			state.FaultGameAbsolutePrestateOverrideKey: "not-a-hash",
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
			"respectedGameType":                        embedded.GameTypeCannonKona,
			state.FaultGameAbsolutePrestateOverrideKey: common.HexToHash("0x22"),
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

	startingAnchorRoot := opcm.Proposal{
		Root:             common.HexToHash("0x02f4397b2de6fce03b3f9982378c2b4c4deff9c92c662dcc6f9643267aeb5e47"),
		L2SequenceNumber: big.NewInt(1234),
	}
	secondChainID := common.BigToHash(new(big.Int).Add(chainID.Big(), big.NewInt(1)))
	secondDCI := opcm.DeployOPChainInput{
		OpChainProxyAdminOwner:       common.Address{'A'},
		SystemConfigOwner:            common.Address{'B'},
		Batcher:                      common.Address{'C'},
		UnsafeBlockSigner:            common.Address{'D'},
		Proposer:                     common.Address{'E'},
		Challenger:                   common.Address{'F'},
		BasefeeScalar:                standard.BasefeeScalar,
		BlobBaseFeeScalar:            standard.BlobBaseFeeScalar,
		L2ChainId:                    secondChainID.Big(),
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
	}
	beforeExecution, err := json.Marshal(st)
	require.NoError(t, err)

	deploymentResult, err := ExecuteOPChainDeployment(pEnv, st, secondChainID, secondDCI)
	require.NoError(t, err)
	afterExecution, err := json.Marshal(st)
	require.NoError(t, err)
	require.Equal(t, beforeExecution, afterExecution)

	getStartingAnchorRoot := w3.MustNewFunc("getStartingAnchorRoot()", "(bytes32 root, uint256 l2SequenceNumber)")
	callData, err := getStartingAnchorRoot.EncodeArgs()
	require.NoError(t, err)
	result, err := l1Client.CallContract(ctx, ethereum.CallMsg{
		To:   &deploymentResult.contracts.AnchorStateRegistryProxy,
		Data: callData,
	}, nil)
	require.NoError(t, err)
	var actualAnchorRoot opcm.Proposal
	require.NoError(t, getStartingAnchorRoot.DecodeReturns(result, &actualAnchorRoot))
	require.Equal(t, startingAnchorRoot.Root, actualAnchorRoot.Root)
	require.Equal(t, startingAnchorRoot.L2SequenceNumber, actualAnchorRoot.L2SequenceNumber)

	require.NoError(t, RecordOPChainDeployment(st, deploymentResult))
	secondChain, err := st.Chain(secondChainID)
	require.NoError(t, err)
	require.True(t, *secondChain.Deployed)
}
