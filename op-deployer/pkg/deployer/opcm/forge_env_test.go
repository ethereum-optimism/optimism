package opcm

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
)

func TestReplayBroadcast(t *testing.T) {
	from, to := common.Address{'F'}, common.Address{'T'}
	chainID := big.NewInt(31337)
	for _, tt := range []struct {
		name   string
		modify func(*forge.Transaction)
		want   broadcaster.BroadcastType
		err    string
	}{
		{name: "call", want: broadcaster.BroadcastCall},
		{name: "create", modify: func(tx *forge.Transaction) { tx.To = nil }, want: broadcaster.BroadcastCreate},
		{name: "foreign sender", modify: func(tx *forge.Transaction) { tx.From = to }, err: "does not match deployer"},
		{name: "nonce drift", modify: func(tx *forge.Transaction) { tx.Nonce++ }, err: "nonce mismatch"},
		{name: "wrong chain", modify: func(tx *forge.Transaction) { tx.ChainID = (*hexutil.Big)(big.NewInt(1)) }, err: "does not match L1"},
		{name: "missing chain", modify: func(tx *forge.Transaction) { tx.ChainID = nil }, err: "does not match L1"},
		{name: "access list", modify: func(tx *forge.Transaction) { tx.AccessList = types.AccessList{{Address: to}} }, err: "unsupported"},
		{name: "unsupported type", modify: func(tx *forge.Transaction) { tx.Type = types.AccessListTxType }, err: "unsupported"},
		{name: "missing value", modify: func(tx *forge.Transaction) { tx.Value = nil }, err: "invalid simulated transaction value"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tx := forge.Transaction{
				From: from, To: &to, Nonce: 3, Value: (*hexutil.U256)(uint256.NewInt(7)),
				Input: []byte{1, 2}, ChainID: (*hexutil.Big)(chainID), Type: types.DynamicFeeTxType,
			}
			if tt.modify != nil {
				tt.modify(&tx)
			}
			out, err := replayBroadcast(tx, chainID, from, 3)
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, out.Type)
			require.Equal(t, from, out.From)
			require.Equal(t, uint64(tx.Nonce), out.Nonce)
			require.Equal(t, []byte(tx.Input), []byte(out.Input))
			require.EqualValues(t, 7, (*uint256.Int)(out.Value).Uint64())
			if tx.To != nil {
				require.Equal(t, *tx.To, out.To)
			}
		})
	}
}

func TestRequireReplayNonce(t *testing.T) {
	from := common.Address{'F'}
	for _, tt := range []struct {
		name            string
		latest, pending uint64
		err             string
	}{
		{name: "settled", latest: 5, pending: 5},
		{name: "pending transaction", latest: 5, pending: 6, err: "nonce mismatch"},
		{name: "account moved on", latest: 6, pending: 6, err: "nonce mismatch"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     json.RawMessage   `json:"id"`
					Method string            `json:"method"`
					Params []json.RawMessage `json:"params"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
				require.Equal(t, "eth_getTransactionCount", req.Method)
				nonce := tt.latest
				if string(req.Params[1]) == `"pending"` {
					nonce = tt.pending
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"%#x"}`, req.ID, nonce)
			}))
			defer server.Close()
			client, err := ethclient.Dial(server.URL)
			require.NoError(t, err)
			defer client.Close()
			err = requireReplayNonce(context.Background(), client, from, 5)
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNonceGuardSigner(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	from := crypto.PubkeyToAddress(key.PublicKey)
	chainID := big.NewInt(31337)
	expected := uint64(3)
	var stopped error
	sign := nonceGuardSigner(key, chainID, &expected, func(err error) { stopped = err })
	tx := func(nonce uint64, tip int64) *types.Transaction {
		return types.NewTx(&types.DynamicFeeTx{ChainID: chainID, Nonce: nonce, GasTipCap: big.NewInt(tip), GasFeeCap: big.NewInt(100), Gas: 21_000})
	}

	_, err = sign(context.Background(), from, tx(3, 1))
	require.NoError(t, err)
	_, err = sign(context.Background(), from, tx(3, 2))
	require.NoError(t, err, "a fee bump signs again at the same nonce")
	require.NoError(t, stopped)
	_, err = sign(context.Background(), from, tx(4, 1))
	require.ErrorContains(t, err, "deployment nonce changed")
	require.Equal(t, stopped, err, "a refusal stops the send")
	expected++
	_, err = sign(context.Background(), from, tx(4, 1))
	require.NoError(t, err)
	_, err = sign(context.Background(), from, tx(3, 1))
	require.ErrorContains(t, err, "deployment nonce changed")
}

// rpcRequest is a JSON-RPC request that a test RPC proxy reads.
type rpcRequest struct {
	ID     json.RawMessage   `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

// newRPCProxy serves target's RPC at a new URL. answer can answer a request in target's place.
func newRPCProxy(t *testing.T, target string, answer func(rpcRequest) (json.RawMessage, bool)) string {
	t.Helper()
	u, err := url.Parse(target)
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(u)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		var req rpcRequest
		if json.Unmarshal(body, &req) == nil {
			if result, ok := answer(req); ok {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, result)
				return
			}
		}
		proxy.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// fakeLiveCaller returns a script caller that runs no forge. Like a simulation on the L1 head, it
// returns one call from the deployer at its nonce. It calls onRun after it reads the nonce.
func fakeLiveCaller(live *ethclient.Client, from common.Address, onRun func(ctx context.Context)) newScriptCaller[byte, forge.None] {
	return func(*forge.Client) forge.ScriptCaller[byte, forge.None] {
		return func(ctx context.Context, _ byte, _ ...string) (forge.None, []forge.Transaction, error) {
			nonce, err := live.NonceAt(ctx, from, nil)
			if err != nil {
				return forge.None{}, nil, err
			}
			chainID, err := live.ChainID(ctx)
			if err != nil {
				return forge.None{}, nil, err
			}
			if onRun != nil {
				onRun(ctx)
			}
			to := common.Address{'T'}
			return forge.None{}, []forge.Transaction{{
				From: from, To: &to, Nonce: hexutil.Uint64(nonce), Value: (*hexutil.U256)(uint256.NewInt(0)),
				ChainID: (*hexutil.Big)(chainID), Type: types.DynamicFeeTxType,
			}}, nil
		}
	}
}

// sendTransfer sends a transfer from key and waits for its receipt.
func sendTransfer(t *testing.T, ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey) {
	t.Helper()
	from := crypto.PubkeyToAddress(key.PublicKey)
	chainID, err := client.ChainID(ctx)
	require.NoError(t, err)
	nonce, err := client.PendingNonceAt(ctx, from)
	require.NoError(t, err)
	head, err := client.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	to := common.Address{'X'}
	tx, err := types.SignNewTx(key, types.LatestSignerForChainID(chainID), &types.DynamicFeeTx{
		ChainID: chainID, Nonce: nonce, GasTipCap: big.NewInt(1), GasFeeCap: new(big.Int).Add(new(big.Int).Mul(head.BaseFee, big.NewInt(2)), big.NewInt(1)),
		Gas: 21_000, To: &to, Value: big.NewInt(1),
	})
	require.NoError(t, err)
	require.NoError(t, client.SendTransaction(ctx, tx))
	require.Eventually(t, func() bool {
		_, err := client.TransactionReceipt(ctx, tx.Hash())
		return err == nil
	}, 30*time.Second, 50*time.Millisecond)
}

// TestBroadcastLiveScriptWaitsForPendingTransactions checks that a pending transaction of the
// deployer, such as a funding transfer from the same key, delays the deployment instead of failing it.
func TestBroadcastLiveScriptWaitsForPendingTransactions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	logger := testlog.Logger(t, slog.LevelInfo)
	liveURL, live := devnet.DefaultAnvilRPC(t, logger)
	key, err := crypto.HexToECDSA(strings.TrimPrefix(testutil.AnvilDefaultPrivateKey, "0x"))
	require.NoError(t, err)
	from := crypto.PubkeyToAddress(key.PublicKey)

	// More failed reads than a fixed 10s retry would allow.
	var mu sync.Mutex
	pendingReads := 6
	proxyURL := newRPCProxy(t, liveURL, func(req rpcRequest) (json.RawMessage, bool) {
		if req.Method != "eth_getTransactionCount" || len(req.Params) != 2 || string(req.Params[1]) != `"pending"` {
			return nil, false
		}
		mu.Lock()
		defer mu.Unlock()
		if pendingReads == 0 {
			return nil, false
		}
		pendingReads--
		nonce, err := live.NonceAt(ctx, from, nil)
		if err != nil {
			return nil, false
		}
		return json.RawMessage(fmt.Sprintf(`"%#x"`, nonce+1)), true
	})

	env := &ForgeEnv{Context: ctx, L1RPCUrl: proxyURL, PrivateKey: testutil.AnvilDefaultPrivateKey, Logger: logger, ReceiptQueryInterval: 50 * time.Millisecond}
	_, err = broadcastLiveScript(ctx, env, fakeLiveCaller(live, from, nil), byte(0))
	require.NoError(t, err)
	nonce, err := live.NonceAt(ctx, from, nil)
	require.NoError(t, err)
	require.EqualValues(t, 1, nonce)
}

// TestBroadcastLiveScriptForkRetries checks that a failed run on the fork runs again only when a
// later run can succeed: an RPC error, but not a compile or a revert.
func TestBroadcastLiveScriptForkRetries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	logger := testlog.Logger(t, slog.LevelInfo)
	liveURL, _ := devnet.DefaultAnvilRPC(t, logger)

	failingCaller := func(runs *int, errs ...error) newScriptCaller[byte, forge.None] {
		return func(*forge.Client) forge.ScriptCaller[byte, forge.None] {
			return func(context.Context, byte, ...string) (forge.None, []forge.Transaction, error) {
				*runs++
				if *runs <= len(errs) {
					return forge.None{}, nil, fmt.Errorf("failed to run forge script: %w", errs[*runs-1])
				}
				return forge.None{}, nil, nil
			}
		}
	}
	rpcErr := errors.New("failed to get block number: 500; latest block number: 5")
	for _, tc := range []struct {
		name    string
		errs    []error
		runs    int
		wantErr error
	}{
		{name: "revert", errs: []error{forge.ErrScriptReverted}, runs: 1, wantErr: forge.ErrScriptReverted},
		{name: "compile", errs: []error{forge.ErrCompiled}, runs: 1, wantErr: forge.ErrCompiled},
		{name: "rpc error", errs: []error{rpcErr}, runs: 2},
		{name: "rpc errors", errs: []error{rpcErr, rpcErr, rpcErr}, runs: forkAttempts, wantErr: rpcErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runs := 0
			env := &ForgeEnv{Context: ctx, L1RPCUrl: liveURL, PrivateKey: testutil.AnvilDefaultPrivateKey, Logger: logger}
			_, err := broadcastLiveScript(ctx, env, failingCaller(&runs, tc.errs...), byte(0))
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.runs, runs)
		})
	}
}

// TestBroadcastLiveScriptSimulatesAgain checks that a deployer transaction that lands between the
// simulation and the first send makes the script run again at the new nonce, instead of failing.
func TestBroadcastLiveScriptSimulatesAgain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	logger := testlog.Logger(t, slog.LevelInfo)
	liveURL, live := devnet.DefaultAnvilRPC(t, logger)
	key, err := crypto.HexToECDSA(strings.TrimPrefix(testutil.AnvilDefaultPrivateKey, "0x"))
	require.NoError(t, err)
	from := crypto.PubkeyToAddress(key.PublicKey)

	runs := 0
	onRun := func(ctx context.Context) {
		runs++
		if runs == 1 {
			sendTransfer(t, ctx, live, key)
		}
	}
	env := &ForgeEnv{Context: ctx, L1RPCUrl: liveURL, PrivateKey: testutil.AnvilDefaultPrivateKey, Logger: logger, ReceiptQueryInterval: 50 * time.Millisecond}
	_, err = broadcastLiveScript(ctx, env, fakeLiveCaller(live, from, onRun), byte(0))
	require.NoError(t, err)
	require.Equal(t, 2, runs)
	nonce, err := live.NonceAt(ctx, from, nil)
	require.NoError(t, err)
	require.EqualValues(t, 2, nonce, "the transfer, then the deployment at the next nonce")
}

// TestSendReplayStopsOnNonceRefusal checks that a send that txmgr signs at another nonce than the
// simulated one fails at once with the signer's error, not after txmgr's retries.
func TestSendReplayStopsOnNonceRefusal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	logger := testlog.Logger(t, slog.LevelInfo)
	_, live := devnet.DefaultAnvilRPC(t, logger)
	key, err := crypto.HexToECDSA(strings.TrimPrefix(testutil.AnvilDefaultPrivateKey, "0x"))
	require.NoError(t, err)
	from := crypto.PubkeyToAddress(key.PublicKey)
	chainID, err := live.ChainID(ctx)
	require.NoError(t, err)

	// The deployer's nonce is 0, but the transaction was simulated at 5.
	bcasts := []broadcaster.Broadcast{{From: from, To: common.Address{'T'}, Type: broadcaster.BroadcastCall, Nonce: 5, Value: (*hexutil.U256)(uint256.NewInt(0))}}
	start := time.Now()
	_, err = sendReplay(ctx, logger, live, key, chainID, 5, bcasts, 50*time.Millisecond)
	require.ErrorContains(t, err, "deployment nonce changed")
	require.Less(t, time.Since(start), 15*time.Second)
}

// TestBroadcastLiveScriptSeesEarlierSends checks that a script after a send forks at the block of the
// send's receipt or later, even while the RPC serves an older head, as a lagging backend of a
// load-balanced RPC does.
func TestBroadcastLiveScriptSeesEarlierSends(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	logger := testlog.Logger(t, slog.LevelInfo)
	liveURL, live := devnet.DefaultAnvilRPC(t, logger)
	key, err := crypto.HexToECDSA(strings.TrimPrefix(testutil.AnvilDefaultPrivateKey, "0x"))
	require.NoError(t, err)
	from := crypto.PubkeyToAddress(key.PublicKey)

	var mu sync.Mutex
	laggingHeads := 0
	// While it lags, the backend serves the block before the send and the nonce at that block, as a
	// backend that has not seen the send does.
	proxyURL := newRPCProxy(t, liveURL, func(req rpcRequest) (json.RawMessage, bool) {
		mu.Lock()
		defer mu.Unlock()
		if laggingHeads == 0 || len(req.Params) != 2 {
			return nil, false
		}
		head, err := live.BlockNumber(ctx)
		if err != nil {
			return nil, false
		}
		switch {
		case req.Method == "eth_getBlockByNumber" && string(req.Params[0]) == `"latest"`:
			laggingHeads--
			var block json.RawMessage
			if err := live.Client().CallContext(ctx, &block, "eth_getBlockByNumber", hexutil.EncodeUint64(head-1), false); err != nil {
				return nil, false
			}
			return block, true
		case req.Method == "eth_getTransactionCount" && (string(req.Params[1]) == `"latest"` || string(req.Params[1]) == `"pending"`):
			nonce, err := live.NonceAt(ctx, from, new(big.Int).SetUint64(head-1))
			if err != nil {
				return nil, false
			}
			return json.RawMessage(fmt.Sprintf(`"%#x"`, nonce)), true
		}
		return nil, false
	})

	env := &ForgeEnv{Client: new(forge.Client), Context: ctx, L1RPCUrl: proxyURL, PrivateKey: testutil.AnvilDefaultPrivateKey, Logger: logger, ReceiptQueryInterval: 50 * time.Millisecond}
	_, err = broadcastScript(env, fakeLiveCaller(live, from, nil), byte(0))
	require.NoError(t, err)
	sent, err := live.BlockNumber(ctx)
	require.NoError(t, err)

	mu.Lock()
	laggingHeads = 2
	mu.Unlock()
	var forkBlock string
	next := func(*forge.Client) forge.ScriptCaller[byte, forge.None] {
		return func(_ context.Context, _ byte, opts ...string) (forge.None, []forge.Transaction, error) {
			if i := slices.Index(opts, "--fork-block-number"); i >= 0 && i+1 < len(opts) {
				forkBlock = opts[i+1]
			}
			return forge.None{}, nil, nil
		}
	}
	_, err = broadcastScript(env, next, byte(0))
	require.NoError(t, err)
	require.Equal(t, strconv.FormatUint(sent, 10), forkBlock, "the next script forks at the block of the send, not at the lagging head")
	mu.Lock()
	defer mu.Unlock()
	require.Zero(t, laggingHeads, "the next script waits for the head to catch up")
}

// TestForgeLiveReplay exercises real forge/anvil execution and real txmgr delivery. The first
// live transaction and its identical retries are silently dropped by the RPC. txmgr must
// rebroadcast and eventually bump its fee for the deployment to finish.
func TestForgeLiveReplay(t *testing.T) {
	// txmgr bumps the fee only after its 48 s resubmission timeout, and the first forge run can
	// download solc, so the test gets ample time.
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	logger := testlog.Logger(t, slog.LevelInfo)
	liveURL, live := devnet.DefaultAnvilRPC(t, logger)
	key, err := crypto.HexToECDSA(strings.TrimPrefix(testutil.AnvilDefaultPrivateKey, "0x"))
	require.NoError(t, err)
	from := crypto.PubkeyToAddress(key.PublicKey)

	project := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(project, "foundry.toml"), []byte("[profile.default]\nsolc_version = '0.8.15'\n"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(project, "script"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(project, "script", "Replay.s.sol"), []byte(replayTestScript), 0o644))
	client, err := forge.NewStandardClient(project)
	require.NoError(t, err)
	client.Stdout, client.Stderr = io.Discard, io.Discard
	require.NoError(t, client.Build(ctx))

	target, err := url.Parse(liveURL)
	require.NoError(t, err)
	proxy := httputil.NewSingleHostReverseProxy(target)
	var mu sync.Mutex
	var sent []*types.Transaction
	var serveStaleNonce, servePendingAhead bool
	var unexpectedRPC int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		_ = json.Unmarshal(body, &req)
		if req.Method == "eth_sendTransaction" || (req.Method == "eth_getBlockByNumber" && len(req.Params) == 2 && string(req.Params[1]) == "true") {
			// No unlocked sends or block-body harvesting: only forge's private file is trusted.
			mu.Lock()
			unexpectedRPC++
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"untrusted transaction source"}}`, req.ID)
			return
		}
		mu.Lock()
		stale := serveStaleNonce && len(sent) > 1
		pendingAhead := servePendingAhead
		mu.Unlock()
		if pendingAhead && req.Method == "eth_getTransactionCount" && len(req.Params) == 2 && string(req.Params[1]) == `"pending"` {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x63"}`, req.ID)
			return
		}
		if stale && req.Method == "eth_getTransactionCount" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x0"}`, req.ID)
			return
		}
		if req.Method == "eth_sendRawTransaction" && len(req.Params) == 1 {
			var raw string
			if err := json.Unmarshal(req.Params[0], &raw); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			tx := new(types.Transaction)
			if err := tx.UnmarshalBinary(common.FromHex(raw)); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			sent = append(sent, tx)
			drop := tx.Nonce() == 0 && tx.Hash() == sent[0].Hash()
			mu.Unlock()
			if drop {
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%q}`, req.ID, tx.Hash().Hex())
				return
			}
		}
		proxy.ServeHTTP(w, r)
	}))
	defer server.Close()

	privateKey := fmt.Sprintf("%x", crypto.FromECDSA(key))
	env := &ForgeEnv{Client: client, Context: ctx, L1RPCUrl: server.URL, PrivateKey: privateKey, Logger: logger, ReceiptQueryInterval: 50 * time.Millisecond}
	caller := func(c *forge.Client) forge.ScriptCaller[byte, forge.None] {
		return func(ctx context.Context, mode byte, opts ...string) (forge.None, []forge.Transaction, error) {
			// A private key or keystore must never be passed to forge, even during simulation.
			args := strings.Join(opts, " ")
			require.NotContains(t, args, privateKey)
			require.NotContains(t, args, "--private-key")
			require.NotContains(t, args, "--keystore")
			require.NotContains(t, args, "--broadcast")
			require.NotContains(t, args, "--unlocked")
			require.Contains(t, args, "--rpc-url "+server.URL)
			_, transactions, err := c.RunScript(ctx, "script/Replay.s.sol:Replay", "runWithBytes(bytes)", []byte{mode}, opts...)
			return forge.None{}, transactions, err
		}
	}

	// A script that emits no transactions remains a successful no-op.
	_, err = broadcastScript(env, caller, byte(3))
	require.NoError(t, err)

	// A simulation that reverts after creating a contract must send nothing to L1.
	_, err = broadcastScript(env, caller, byte(1))
	require.ErrorContains(t, err, "deployment simulation failed")
	nonce, err := live.NonceAt(ctx, from, nil)
	require.NoError(t, err)
	require.Zero(t, nonce)
	mu.Lock()
	sendCount := len(sent)
	mu.Unlock()
	require.Zero(t, sendCount)

	// A valid transaction followed by a foreign sender must also fail before any live send.
	_, err = broadcastScript(env, caller, byte(2))
	require.ErrorContains(t, err, "does not match deployer")
	nonce, err = live.NonceAt(ctx, from, nil)
	require.NoError(t, err)
	require.Zero(t, nonce)

	// A deployer transaction that stays pending would shift every simulated CREATE address: nothing
	// is sent.
	oldWait := l1WaitTimeout
	l1WaitTimeout = 2 * l1PollInterval
	mu.Lock()
	servePendingAhead = true
	mu.Unlock()
	_, err = broadcastScript(env, caller, byte(0))
	l1WaitTimeout = oldWait
	require.ErrorContains(t, err, "nonce mismatch")
	mu.Lock()
	servePendingAhead = false
	sendCount = len(sent)
	mu.Unlock()
	require.Zero(t, sendCount)

	// After the first send, simulate a provider whose nonce backend lags its receipt backend.
	mu.Lock()
	serveStaleNonce = true
	mu.Unlock()
	_, err = broadcastScript(env, caller, byte(0))
	require.NoError(t, err, "later sends must use txmgr's nonce cache, not lagging latest/pending RPC reads")
	mu.Lock()
	serveStaleNonce = false
	mu.Unlock()
	nonce, err = live.NonceAt(ctx, from, nil)
	require.NoError(t, err)
	require.EqualValues(t, 2, nonce, "the CREATE and subsequent call are delivered in order")
	created := crypto.CreateAddress(from, 0)
	balance, err := live.BalanceAt(ctx, created, nil)
	require.NoError(t, err)
	require.EqualValues(t, 7, bigs.Uint64Strict(balance), "CREATE value is preserved")
	result, err := live.CallContract(ctx, ethereum.CallMsg{To: &created, Data: crypto.Keccak256([]byte("value()"))[:4]}, nil)
	require.NoError(t, err)
	require.EqualValues(t, 42, bigs.Uint64Strict(new(big.Int).SetBytes(result)))
	mu.Lock()
	delivered := append([]*types.Transaction(nil), sent...)
	mu.Unlock()
	require.GreaterOrEqual(t, len(delivered), 3, "txmgr resends the silently dropped transaction")
	require.Equal(t, delivered[0].Hash(), delivered[1].Hash(), "the rebroadcast retries the same signed transaction")
	var bumped *types.Transaction
	for _, tx := range delivered[1:] {
		if tx.Nonce() == 0 && tx.Hash() != delivered[0].Hash() {
			bumped = tx
			break
		}
	}
	require.NotNil(t, bumped, "txmgr must replace the stuck transaction with a fee bump")
	require.Positive(t, bumped.GasTipCap().Cmp(delivered[0].GasTipCap()))
	require.Positive(t, bumped.GasFeeCap().Cmp(delivered[0].GasFeeCap()))

	// A later script forks the updated live state and keeps CREATE address prediction correct.
	_, err = broadcastScript(env, caller, byte(0))
	require.NoError(t, err)
	code, err := live.CodeAt(ctx, crypto.CreateAddress(from, 2), nil)
	require.NoError(t, err)
	require.NotEmpty(t, code)

	// Forge records CREATE2 as a call to the deterministic proxy, including salt + init code.
	_, err = broadcastScript(env, caller, byte(4))
	require.NoError(t, err)
	mu.Lock()
	delivered = append([]*types.Transaction(nil), sent...)
	untrustedReads := unexpectedRPC
	mu.Unlock()
	require.Zero(t, untrustedReads)
	create2 := delivered[len(delivered)-2]
	require.NotNil(t, create2.To())
	require.Equal(t, predeploys.DeterministicDeploymentProxyAddr, *create2.To())
	require.Equal(t, common.Hash{31: 1}.Bytes(), create2.Data()[:32])
	called := delivered[len(delivered)-1].To()
	require.NotNil(t, called)
	code, err = live.CodeAt(ctx, *called, nil)
	require.NoError(t, err)
	require.NotEmpty(t, code)
}

const replayTestScript = `// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.15;
interface Vm {
    function startBroadcast() external;
    function startBroadcast(address sender) external;
    function stopBroadcast() external;
}
contract Target {
    uint256 public value;
    constructor() payable {}
    function set(uint256 v) external { value = v; }
}
contract Replay {
    Vm constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));
    function runWithBytes(bytes calldata input) external {
        if (uint8(input[0]) == 3) return;
        vm.startBroadcast();
        Target target;
        if (uint8(input[0]) == 4) {
            target = new Target{salt: bytes32(uint256(1)), value: 7}();
        } else {
            target = new Target{value: 7}();
        }
        require(uint8(input[0]) != 1, "simulation reverted");
        if (uint8(input[0]) == 2) {
            vm.stopBroadcast();
            vm.startBroadcast(0x70997970C51812dc3A010C7d01b50e0d17dc79C8);
        }
        target.set(42);
        vm.stopBroadcast();
    }
}
`
