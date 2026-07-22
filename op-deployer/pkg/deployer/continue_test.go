package deployer

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
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
	require.ErrorContains(t, err, "run op-deployer prepare")
}

func TestContinueStartupGates(t *testing.T) {
	chainID := common.HexToHash("0x01")

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
				st.PreparedDeployment.Deployer = common.Address{}
			},
			wantErr: "no pinned deployer or OPCM",
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
				st.PreparedDeployment.OPCM = common.Address{}
			},
			wantErr: "no pinned deployer or OPCM",
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
				CacheDir:   t.TempDir(),
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
	intent.L1ContractsLocator = artifacts.EmbeddedLocator
	intent.L2ContractsLocator = artifacts.EmbeddedLocator
	privateKey, err := crypto.HexToECDSA(testPrivKey)
	require.NoError(t, err)
	deployer := crypto.PubkeyToAddress(privateKey.PublicKey)
	opcmAddress := *intent.OPCMAddress
	interopDepSet, err := pipeline.BuildInteropDepSet(intent.Chains)
	require.NoError(t, err)
	st := &state.State{
		Version:       1,
		Create2Salt:   common.Hash{0x01},
		InteropDepSet: interopDepSet,
	}
	for _, id := range chainIDs {
		st.SetChainContracts(id, addresses.OpChainContracts{}, false)
		st.PinChainAnchor(id, &state.L1BlockRefJSON{Hash: common.Hash{0x01}, Number: 1, Time: 1}, 2)
	}
	bundle, err := artifacts.DownloadBundle(t.Context(), intent.L1ContractsLocator, intent.L2ContractsLocator, ioutil.NoopProgressor(), t.TempDir())
	require.NoError(t, err)
	st.PreparedDeployment, err = pipeline.NewPreparedDeployment(intent, st, deployer, opcmAddress, bundle)
	require.NoError(t, err)
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

func TestSuccessfulContinuationBroadcast(t *testing.T) {
	txHash := common.HexToHash("0x11")
	valid := broadcaster.BroadcastResult{
		TxHash:  txHash,
		Receipt: &types.Receipt{Status: types.ReceiptStatusSuccessful},
	}
	tests := []struct {
		name         string
		results      []broadcaster.BroadcastResult
		broadcastErr error
		want         broadcaster.BroadcastResult
		wantErr      string
	}{
		{name: "valid", results: []broadcaster.BroadcastResult{valid}, want: valid},
		{name: "broadcast error", broadcastErr: fmt.Errorf("broadcast failed"), wantErr: "broadcast failed"},
		{name: "no result", wantErr: "expected one"},
		{name: "multiple results", results: []broadcaster.BroadcastResult{valid, valid}, wantErr: "expected one"},
		{name: "result error", results: []broadcaster.BroadcastResult{{Err: fmt.Errorf("result failed")}}, wantErr: "result failed"},
		{name: "no receipt", results: []broadcaster.BroadcastResult{{TxHash: txHash}}, wantErr: "no receipt"},
		{name: "failed receipt", results: []broadcaster.BroadcastResult{{TxHash: txHash, Receipt: &types.Receipt{Status: types.ReceiptStatusFailed}}}, wantErr: "receipt status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := successfulContinuationBroadcast(tt.results, tt.broadcastErr)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestContinuationExpectedStateUsesPreparedContracts(t *testing.T) {
	chainID := common.HexToHash("0x01")
	expected := addressesForValidationTest()
	prepared := &state.PreparedChainState{ID: chainID, OpChainContracts: expected}
	recorded := expected
	recorded.FaultDisputeGameImpl = common.Address{0xaa}
	chainState := &state.ChainState{ID: chainID, OpChainContracts: recorded}

	got := continuationExpectedChainState(prepared, chainState)
	require.Equal(t, expected, got.OpChainContracts)
	require.Equal(t, recorded, chainState.OpChainContracts)
}

func TestValidateContinuationGameTypes(t *testing.T) {
	chain := func(gameType embedded.GameType) continuationChain {
		var dci opcm.DeployOPChainInput
		dci.DisputeGameType = uint32(gameType)
		return continuationChain{dci: dci}
	}

	require.NoError(t, validateContinuationGameTypes(nil))
	require.NoError(t, validateContinuationGameTypes([]continuationChain{
		chain(embedded.GameTypePermissionedCannon),
		chain(embedded.GameTypeCannonKona),
	}))
	require.ErrorContains(t, validateContinuationGameTypes([]continuationChain{
		chain(embedded.GameTypeCannonKona),
		chain(embedded.GameTypeSuperCannonKona),
	}), "cannot mix CANNON_KONA and SUPER_CANNON_KONA")
}

func TestClassifyContinuationAddresses(t *testing.T) {
	contracts := continuationVerificationAddresses(embedded.GameTypeCannonKona)
	backend := newContinuationVerificationBackend()
	blockNumber := big.NewInt(123)

	classification, err := classifyContinuationAddresses(t.Context(), backend, contracts, blockNumber)
	require.NoError(t, err)
	require.Equal(t, continuationAddressesAbsent, classification)

	for _, contract := range continuationContractAddresses(contracts) {
		if contract.address != (common.Address{}) {
			backend.code[contract.address] = []byte{0x60}
		}
	}
	classification, err = classifyContinuationAddresses(t.Context(), backend, contracts, blockNumber)
	require.NoError(t, err)
	require.Equal(t, continuationAddressesComplete, classification)

	delete(backend.code, contracts.SystemConfigProxy)
	_, err = classifyContinuationAddresses(t.Context(), backend, contracts, blockNumber)
	require.ErrorContains(t, err, "partial deployment")
	require.ErrorContains(t, err, "SystemConfigProxy")
	require.ErrorContains(t, err, "has no code")
	for _, observedBlockNumber := range backend.codeBlocks {
		require.Equal(t, blockNumber, observedBlockNumber)
	}
}

type continuationNonceReaderStub struct {
	latest  uint64
	pending uint64
}

func (s *continuationNonceReaderStub) NonceAt(context.Context, common.Address, *big.Int) (uint64, error) {
	return s.latest, nil
}

func (s *continuationNonceReaderStub) PendingNonceAt(context.Context, common.Address) (uint64, error) {
	return s.pending, nil
}

func TestContinuationNonceOwnership(t *testing.T) {
	client := &continuationNonceReaderStub{latest: 7, pending: 7}
	nonce, err := readContinuationStartNonce(t.Context(), client, common.Address{0x01})
	require.NoError(t, err)
	require.EqualValues(t, 7, nonce)
	require.NoError(t, requireContinuationNonce(t.Context(), client, common.Address{0x01}, nonce))

	client.pending = 8
	_, err = readContinuationStartNonce(t.Context(), client, common.Address{0x01})
	require.ErrorContains(t, err, "nonce is already moving")
	require.ErrorContains(t, requireContinuationNonce(t.Context(), client, common.Address{0x01}, nonce), "unexpected deployer nonce movement")

	client.latest = 8
	require.ErrorContains(t, requireContinuationNonce(t.Context(), client, common.Address{0x01}, nonce), "unexpected deployer nonce movement")
}

type continuationBlockFetcherStub struct {
	ref state.L1BlockRefJSON
}

func (s *continuationBlockFetcherStub) CallContext(
	_ context.Context,
	result any,
	method string,
	args ...any,
) error {
	if method != "eth_getBlockByNumber" {
		return fmt.Errorf("unexpected method %s", method)
	}
	if len(args) != 2 {
		return fmt.Errorf("unexpected argument count %d", len(args))
	}
	if args[0] != "0x7b" {
		return fmt.Errorf("unexpected block argument %v", args[0])
	}
	ref, ok := result.(*state.L1BlockRefJSON)
	if !ok {
		return fmt.Errorf("unexpected result type %T", result)
	}
	*ref = s.ref
	return nil
}

func TestValidateContinuationReceiptCanonicality(t *testing.T) {
	blockHash := common.HexToHash("0x22")
	receipt := &types.Receipt{BlockNumber: big.NewInt(123), BlockHash: blockHash}
	client := &continuationBlockFetcherStub{ref: state.L1BlockRefJSON{Hash: blockHash, Number: 123}}
	require.NoError(t, validateContinuationReceiptCanonicality(t.Context(), client, receipt))

	client.ref.Hash = common.HexToHash("0x33")
	require.ErrorContains(
		t,
		validateContinuationReceiptCanonicality(t.Context(), client, receipt),
		"receipt block hash mismatch",
	)
}

func TestContinuationFailureHooks(t *testing.T) {
	wantErr := fmt.Errorf("injected failure")
	for _, boundary := range []string{
		"before send",
		"after send before receipt",
		"after receipt before state write",
		"after state write before live validation",
	} {
		t.Run(boundary, func(t *testing.T) {
			called := false
			err := invokeContinuationHook(boundary, func() error {
				called = true
				return wantErr
			})
			require.True(t, called)
			require.ErrorIs(t, err, wantErr)
			require.ErrorContains(t, err, boundary)
		})
	}
	require.NoError(t, invokeContinuationHook("unused", nil))
}

func TestStandardValidatorInput(t *testing.T) {
	contracts := addressesForValidationTest()
	selected := common.Hash{0x01}
	fallback := common.Hash{0x02}
	var dci opcm.DeployOPChainInput
	dci.DisputeGameType = uint32(embedded.GameTypeCannonKona)
	dci.DisputeAbsolutePrestate = selected
	dci.CannonAbsolutePrestate = fallback
	dci.OpChainProxyAdminOwner = common.Address{0x03}
	dci.Challenger = common.Address{0x04}
	input := standardValidatorInput(dci, contracts)
	require.Equal(t, contracts.SystemConfigProxy, input.SystemConfig)
	require.Equal(t, selected, input.CannonKonaPrestate)
	require.Equal(t, fallback, input.CannonPrestate)
	require.Equal(t, dci.OpChainProxyAdminOwner, input.L1PAOMultisig)
	require.Equal(t, dci.Challenger, input.Challenger)
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
