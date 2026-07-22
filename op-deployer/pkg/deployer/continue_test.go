package deployer

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestContinueConfigCheck(t *testing.T) {
	valid := ContinueConfig{
		Workdir:    t.TempDir(),
		L1RPCUrl:   testL1RPCUrl,
		PrivateKey: testPrivKey,
		Logger:     log.NewLogger(log.DiscardHandler()),
	}

	tests := []struct {
		name    string
		mutate  func(*ContinueConfig)
		wantErr string
	}{
		{name: "valid"},
		{name: "missing workdir", mutate: func(cfg *ContinueConfig) { cfg.Workdir = "" }, wantErr: "workdir must be specified"},
		{name: "missing logger", mutate: func(cfg *ContinueConfig) { cfg.Logger = nil }, wantErr: "logger must be specified"},
		{name: "missing key", mutate: func(cfg *ContinueConfig) { cfg.PrivateKey = "" }, wantErr: "private key must be specified"},
		{name: "invalid key", mutate: func(cfg *ContinueConfig) { cfg.PrivateKey = "invalid" }, wantErr: "failed to parse private key"},
		{name: "missing RPC", mutate: func(cfg *ContinueConfig) { cfg.L1RPCUrl = "" }, wantErr: "l1 RPC URL must be specified"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}
			err := cfg.Check()
			if tt.wantErr == "" {
				require.NoError(t, err)
				require.NotNil(t, cfg.privateKeyECDSA)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestNewContinueConfig(t *testing.T) {
	app := cli.NewApp()
	flagSet := flag.NewFlagSet("test-continue", flag.ContinueOnError)
	for _, cliFlag := range append(ContinueFlags, GlobalFlags...) {
		require.NoError(t, cliFlag.Apply(flagSet))
	}
	require.NoError(t, flagSet.Set(WorkdirFlagName, "/tmp/workdir"))
	require.NoError(t, flagSet.Set(L1RPCURLFlagName, testL1RPCUrl))
	require.NoError(t, flagSet.Set(PrivateKeyFlagName, testPrivKey))
	require.NoError(t, flagSet.Set(CacheDirFlagName, "/tmp/cache"))

	cfg := newContinueConfig(cli.NewContext(app, flagSet, nil), log.NewLogger(log.DiscardHandler()))
	require.Equal(t, "/tmp/workdir", cfg.Workdir)
	require.Equal(t, testL1RPCUrl, cfg.L1RPCUrl)
	require.Equal(t, testPrivKey, cfg.PrivateKey)
	require.Equal(t, "/tmp/cache", cfg.CacheDir)
}

func TestContinueFlagsExcludeDeploymentSelectors(t *testing.T) {
	flagNames := make(map[string]bool)
	for _, cliFlag := range ContinueFlags {
		for _, name := range cliFlag.Names() {
			flagNames[name] = true
		}
	}
	require.False(t, flagNames[UseForgeFlagName])
	require.False(t, flagNames[DeploymentTargetFlag.Name])
	require.False(t, flagNames[GenesisTimeOffsetFlagName])
	require.Equal(t, map[string]bool{
		WorkdirFlagName:    true,
		OutdirFlagName:     true,
		PrivateKeyFlagName: true,
		L1RPCURLFlagName:   true,
	}, flagNames)
}

func TestContinueRejectsUnpreparedStateBeforeRPC(t *testing.T) {
	intent := newPrestateWorkflowIntent(t, []common.Hash{common.HexToHash("0x01")})
	workdir := t.TempDir()
	require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))
	require.NoError(t, pipeline.WriteState(workdir, &state.State{Version: 1}))

	err := Continue(context.Background(), ContinueConfig{
		Workdir:    workdir,
		L1RPCUrl:   "http://127.0.0.1:1",
		PrivateKey: testPrivKey,
		Logger:     testlog.Logger(t, slog.LevelInfo),
	})
	require.ErrorContains(t, err, "Run op-deployer prepare")
}

func TestContinueStartupGates(t *testing.T) {
	chainID := common.HexToHash("0x01")
	secondChainID := common.HexToHash("0x02")

	tests := []struct {
		name       string
		chainIDs   []common.Hash
		rpcChainID uint64
		privateKey string
		mutate     func(*state.Intent, *state.State)
		wantErr    string
	}{
		{
			name:       "L1 chain ID mismatch",
			chainIDs:   []common.Hash{chainID},
			rpcChainID: 2,
			wantErr:    "l1 chain ID mismatch",
		},
		{
			name:     "prepared chain set drift",
			chainIDs: []common.Hash{chainID},
			mutate: func(_ *state.Intent, st *state.State) {
				st.InteropDepSet = nil
			},
			wantErr: "prepared interop dependency set is missing",
		},
		{
			name:     "missing pinned sender",
			chainIDs: []common.Hash{chainID},
			mutate: func(_ *state.Intent, st *state.State) {
				st.L1PredictSenderAddress = nil
			},
			wantErr: "no pinned deployer address",
		},
		{
			name:       "wrong deployer key",
			chainIDs:   []common.Hash{chainID},
			privateKey: "0000000000000000000000000000000000000000000000000000000000000002",
			wantErr:    "deployer address mismatch",
		},
		{
			name:     "missing pinned OPCM",
			chainIDs: []common.Hash{chainID},
			mutate: func(_ *state.Intent, st *state.State) {
				st.L1PredictOPCMAddress = nil
			},
			wantErr: "no pinned OPCM address",
		},
		{
			name:     "changed intent OPCM",
			chainIDs: []common.Hash{chainID},
			mutate: func(intent *state.Intent, _ *state.State) {
				changed := common.Address{0xaa}
				intent.OPCMAddress = &changed
			},
			wantErr: "intent OPCM address changed",
		},
		{
			name:     "zero pending chains",
			chainIDs: []common.Hash{chainID},
			mutate: func(_ *state.Intent, st *state.State) {
				deployed := true
				st.Chains[0].Deployed = &deployed
			},
			wantErr: "reconciliation is not supported",
		},
		{
			name:     "multiple pending chains",
			chainIDs: []common.Hash{chainID, secondChainID},
			wantErr:  "exactly one pending chain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, st, workdir := continueGateInputs(t, tt.chainIDs)
			if tt.mutate != nil {
				tt.mutate(intent, st)
			}
			require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))
			require.NoError(t, pipeline.WriteState(workdir, st))

			rpcChainID := tt.rpcChainID
			if rpcChainID == 0 {
				rpcChainID = 1
			}
			rpcURL := chainIDRPC(t, rpcChainID)
			privateKey := tt.privateKey
			if privateKey == "" {
				privateKey = testPrivKey
			}
			err := Continue(t.Context(), ContinueConfig{
				Workdir:    workdir,
				L1RPCUrl:   rpcURL,
				PrivateKey: privateKey,
				Logger:     log.NewLogger(log.DiscardHandler()),
			})
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func continueGateInputs(
	t *testing.T,
	chainIDs []common.Hash,
) (*state.Intent, *state.State, string) {
	t.Helper()
	intent := newPrestateWorkflowIntent(t, chainIDs)
	privateKey, err := crypto.HexToECDSA(testPrivKey)
	require.NoError(t, err)
	deployer := crypto.PubkeyToAddress(privateKey.PublicKey)
	opcmAddress := *intent.OPCMAddress
	interopDepSet, err := pipeline.BuildInteropDepSet(intent.Chains)
	require.NoError(t, err)
	st := &state.State{
		Version:                1,
		Prepared:               true,
		Create2Salt:            common.Hash{0x01},
		L1PredictSenderAddress: &deployer,
		L1PredictOPCMAddress:   &opcmAddress,
		InteropDepSet:          interopDepSet,
	}
	for _, id := range chainIDs {
		st.SetChainContracts(id, addresses.OpChainContracts{}, false)
	}
	return intent, st, t.TempDir()
}

func chainIDRPC(t *testing.T, chainID uint64) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "eth_chainId", request.Method)
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      request.ID,
			"result":  fmt.Sprintf("0x%x", chainID),
		}))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func TestValidateContinuationBroadcast(t *testing.T) {
	deployer := common.Address{0x01}
	opcmAddress := common.Address{0x02}
	valid := script.Broadcast{Type: script.BroadcastCall, From: deployer, To: opcmAddress}

	tests := []struct {
		name       string
		broadcasts []script.Broadcast
		wantErr    string
	}{
		{name: "valid", broadcasts: []script.Broadcast{valid}},
		{name: "none", wantErr: "exactly one"},
		{name: "multiple", broadcasts: []script.Broadcast{valid, valid}, wantErr: "exactly one"},
		{name: "type", broadcasts: []script.Broadcast{{Type: script.BroadcastCreate, From: deployer, To: opcmAddress}}, wantErr: "broadcast type"},
		{name: "target", broadcasts: []script.Broadcast{{Type: script.BroadcastCall, From: deployer, To: common.Address{0x03}}}, wantErr: "target mismatch"},
		{name: "sender", broadcasts: []script.Broadcast{{Type: script.BroadcastCall, From: common.Address{0x04}, To: opcmAddress}}, wantErr: "sender mismatch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContinuationBroadcast(tt.broadcasts, deployer, opcmAddress)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateContinuationReceipts(t *testing.T) {
	valid := broadcaster.BroadcastResult{Receipt: &types.Receipt{Status: types.ReceiptStatusSuccessful}}
	require.NoError(t, validateContinuationReceipts([]broadcaster.BroadcastResult{valid}, nil))
	require.ErrorContains(t, validateContinuationReceipts(nil, nil), "expected one")
	require.ErrorContains(t, validateContinuationReceipts([]broadcaster.BroadcastResult{{}}, nil), "no receipt")
	require.ErrorContains(t, validateContinuationReceipts(
		[]broadcaster.BroadcastResult{{Receipt: &types.Receipt{Status: types.ReceiptStatusFailed}}},
		nil,
	), "receipt status")
}

func TestStandardValidatorInput(t *testing.T) {
	contracts := addressesForValidationTest()
	selected := common.Hash{0x01}
	fallback := common.Hash{0x02}
	var dci opcm.DeployOPChainInput
	dci.DisputeGameType = uint32(embedded.GameTypeCannonKona)
	dci.DisputeAbsolutePrestate = selected
	dci.CannonAbsolutePrestate = fallback
	input := standardValidatorInput(dci, contracts)
	require.Equal(t, contracts.SystemConfigProxy, input.SystemConfig)
	require.Equal(t, selected, input.CannonKonaPrestate)
	require.Equal(t, fallback, input.CannonPrestate)
	require.True(t, input.UseDevFeaturesInput)

	dci.DisputeGameType = uint32(embedded.GameTypeSuperCannonKona)
	input = standardValidatorInput(dci, contracts)
	require.True(t, input.UseDevFeaturesInput)
}

func addressesForValidationTest() addresses.OpChainContracts {
	return addresses.OpChainContracts{
		OpChainCoreContracts: addresses.OpChainCoreContracts{
			SystemConfigProxy: common.Address{0x03},
		},
	}
}
