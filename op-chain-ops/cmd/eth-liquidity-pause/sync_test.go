package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

// method is one entry point of a test contract: it emits log (if set), then returns ret.
type method struct {
	ret []byte
	// log's Topics and Data are emitted; its Address is ignored.
	log *types.Log
}

// contractCode assembles runtime code that dispatches on the 4-byte selector to methods, keyed
// by signature. Any other selector reverts with empty data.
func contractCode(methods map[string]method) []byte {
	sigs := make([]string, 0, len(methods))
	for sig := range methods {
		sigs = append(sigs, sig)
	}
	slices.Sort(sigs)
	code := []byte{0x60, 0x00, 0x35, 0x60, 0xe0, 0x1c} // calldataload(0) >> 224
	jumps := make([]int, len(sigs))
	for i, sig := range sigs {
		s := selector(sig)
		code = append(code, 0x80, 0x63) // DUP1 PUSH4 selector
		code = append(code, s[:]...)
		code = append(code, 0x14, 0x61, 0, 0, 0x57) // EQ PUSH2 dest JUMPI
		jumps[i] = len(code) - 3
	}
	code = append(code, 0x60, 0x00, 0x80, 0xfd) // revert(0, 0)
	type blob struct {
		at   int
		data []byte
	}
	var blobs []blob
	for i, sig := range sigs {
		m := methods[sig]
		binary.BigEndian.PutUint16(code[jumps[i]:], uint16(len(code)))
		code = append(code, 0x5b) // JUMPDEST
		if m.log != nil {
			n := len(m.log.Data)
			code = append(code, 0x61, byte(n>>8), byte(n), 0x61, 0, 0, 0x60, 0x00, 0x39) // codecopy(0, data, n)
			blobs = append(blobs, blob{at: len(code) - 5, data: m.log.Data})
			for j := len(m.log.Topics) - 1; j >= 0; j-- {
				code = append(code, 0x7f) // PUSH32 topic
				code = append(code, m.log.Topics[j][:]...)
			}
			code = append(code, 0x61, byte(n>>8), byte(n), 0x60, 0x00, 0xa0+byte(len(m.log.Topics))) // logN(0, n, ...)
		}
		code = append(code, 0x7f) // PUSH32 ret
		code = append(code, common.LeftPadBytes(m.ret, 32)...)
		code = append(code, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3) // mstore(0, ret); return(0, 32)
	}
	for _, b := range blobs {
		binary.BigEndian.PutUint16(code[b.at:], uint16(len(code)))
		code = append(code, b.data...)
	}
	return code
}

// versionedNonce is the L1CrossDomainMessenger nonce n with message version 1.
func versionedNonce(n int64) *big.Int {
	return crossdomain.EncodeVersionedNonce(big.NewInt(n), big.NewInt(1))
}

// messengerCode is an L1CrossDomainMessenger stub at addr: paused() returns paused, and
// syncETHLiquidityPause() emits the SentMessage of setPaused(paused, nonce) the real one would.
func messengerCode(t *testing.T, addr common.Address, paused bool, nonce *big.Int) []byte {
	message, err := setPausedFn.EncodeArgs(paused, nonce)
	require.NoError(t, err)
	data, err := abi.Arguments{
		{Type: mustABIType("address")}, {Type: mustABIType("bytes")},
		{Type: uint256Type}, {Type: uint256Type},
	}.Pack(addr, message, nonce, big.NewInt(200_000))
	require.NoError(t, err)
	return contractCode(map[string]method{
		"paused()": {ret: word(paused)},
		syncSignature: {log: &types.Log{
			Topics: []common.Hash{sentMessageEvt.Topic0, common.BytesToHash(predeploys.ETHLiquidityAddr.Bytes())},
			Data:   data,
		}},
	})
}

// simSender sends through eth_simulateV1 on the simulated L1: each candidate runs in the EVM
// and gets the receipt status and logs that a mined tx would get. It never calls
// simulated.Backend.Commit, because Commit can block forever: SimulatedBeacon waits in
// Payload.WaitFull, a bare sync.Cond.Wait, and the payload builder can broadcast its only update
// before that wait starts.
type simSender struct {
	rpc  *gethrpc.Client
	from common.Address
	mu   sync.Mutex
	sent int
}

func newSimSender(l1 *simulated.Backend, from common.Address) *simSender {
	// simulated.Client hides the RPC client: its embedded *ethclient.Client field, named Client,
	// shadows the Client() method. The field is exported, so reflect can read it.
	eth := reflect.ValueOf(l1.Client()).Field(0).Interface().(*ethclient.Client)
	return &simSender{rpc: eth.Client(), from: from}
}

func (s *simSender) From() common.Address { return s.from }

func (s *simSender) Send(ctx context.Context, c txmgr.TxCandidate) (*types.Receipt, error) {
	s.mu.Lock()
	s.sent++
	n := s.sent
	s.mu.Unlock()
	call := map[string]any{"from": s.from, "to": c.To, "input": hexutil.Bytes(c.TxData)}
	opts := map[string]any{"blockStateCalls": []any{map[string]any{"calls": []any{call}}}}
	var blocks []struct {
		Number hexutil.Uint64 `json:"number"`
		Calls  []struct {
			Status hexutil.Uint64 `json:"status"`
			Logs   []*types.Log   `json:"logs"`
		} `json:"calls"`
	}
	if err := s.rpc.CallContext(ctx, &blocks, "eth_simulateV1", opts, "latest"); err != nil {
		return nil, err
	}
	if len(blocks) != 1 || len(blocks[0].Calls) != 1 {
		return nil, fmt.Errorf("eth_simulateV1 returned %d blocks", len(blocks))
	}
	res := blocks[0].Calls[0]
	return &types.Receipt{
		Status:      uint64(res.Status),
		Logs:        res.Logs,
		TxHash:      common.BigToHash(big.NewInt(int64(n))),
		BlockNumber: new(big.Int).SetUint64(uint64(blocks[0].Number)),
	}, nil
}

func (s *simSender) SendAsync(ctx context.Context, c txmgr.TxCandidate, ch chan txmgr.SendResponse) {
	r, err := s.Send(ctx, c)
	ch <- txmgr.SendResponse{Receipt: r, Err: err}
}

func (s *simSender) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sent
}

func testWait(timeout time.Duration) waitConfig {
	return waitConfig{Timeout: timeout, PollInterval: 10 * time.Millisecond}
}

// syncEnv is a simulated L1 with a sender for the sync.
type syncEnv struct {
	ctx    context.Context
	sender common.Address
	l1     *simulated.Backend
	conn   *connection
	mgr    *simSender
}

func newSyncEnv(t *testing.T, alloc types.GenesisAlloc) *syncEnv {
	env := &syncEnv{sender: common.HexToAddress("0x00000000000000000000000000000000005e4de7")}
	alloc[env.sender] = types.Account{Balance: oneETH}
	env.l1 = simulated.NewBackend(alloc)
	t.Cleanup(func() { _ = env.l1.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	env.ctx = ctx
	l1ChainID, err := env.l1.Client().ChainID(ctx)
	require.NoError(t, err)
	env.conn = &connection{L1: env.l1.Client(), L1ChainID: bigs.Uint64Strict(l1ChainID)}
	env.mgr = newSimSender(env.l1, env.sender)
	return env
}

// addChain dials a fake L2 and pairs it with messenger.
func (e *syncEnv) addChain(t *testing.T, l2 *fakeChain, l1 *l1Contracts) {
	cl, err := dialL2(e.ctx, serve(t, l2))
	require.NoError(t, err)
	t.Cleanup(cl.Close)
	e.conn.Chains = append(e.conn.Chains, chain{L2: cl, L1: l1})
}

func (e *syncEnv) sent() int { return e.mgr.count() }

// deliverAfter makes l2 apply setPaused(paused, nonce) once ready() holds, counting the
// eth_calls it answered before that.
func deliverAfter(l2 *fakeChain, paused bool, nonce *big.Int, ready func(calls int) bool) *atomic.Bool {
	var delivered atomic.Bool
	calls := 0
	l2.beforeCall = func(c *fakeChain) {
		calls++
		if !delivered.Load() && ready(calls) {
			c.setL2Pause(paused, nonce)
			delivered.Store(true)
		}
	}
	return &delivered
}

var (
	messengerA = common.HexToAddress("0x00000000000000000000000000000000000a11ce")
	messengerB = common.HexToAddress("0x00000000000000000000000000000000000b0b00")
)

func TestSyncWaitsForItsOwnDeposit(t *testing.T) {
	env := newSyncEnv(t, types.GenesisAlloc{messengerA: {Code: messengerCode(t, messengerA, true, versionedNonce(5))}})
	// L2 already reads paused=true from an earlier sync (nonce 4), so paused() alone would
	// look done before this sync's deposit lands.
	l2 := fakeL2(901, true)
	l2.setL2Pause(true, versionedNonce(4))
	delivered := deliverAfter(l2, true, versionedNonce(5), func(calls int) bool { return calls > 8 })
	env.addChain(t, l2, &l1Contracts{Messenger: messengerA})

	var out bytes.Buffer
	require.NoError(t, syncChains(env.ctx, &out, env.conn, env.mgr, testWait(10*time.Second), false))
	require.Contains(t, out.String(), "chain 901: L1 paused=true, L2 paused=true: in sync now, skipping (this compares current state only")
	require.Zero(t, env.sent())
	require.False(t, delivered.Load())

	out.Reset()
	require.NoError(t, syncChains(env.ctx, &out, env.conn, env.mgr, testWait(10*time.Second), true))
	require.True(t, delivered.Load(), "returned before L2 applied the sync's deposit")
	require.Contains(t, out.String(), "with L1 message nonce 5 (version 1), carrying L1 paused=true")
	require.Contains(t, out.String(), "chain 901: L2 applied the sync; L2 paused=true")
	require.Equal(t, 1, env.sent())
}

func TestSyncSendsEveryChainThenWaitsTogether(t *testing.T) {
	env := newSyncEnv(t, types.GenesisAlloc{
		messengerA: {Code: messengerCode(t, messengerA, true, versionedNonce(1))},
		messengerB: {Code: messengerCode(t, messengerB, true, versionedNonce(7))},
	})
	l2A, l2B := fakeL2(901, false), fakeL2(902, false)
	deliveredB := deliverAfter(l2B, true, versionedNonce(7), func(calls int) bool { return calls > 4 })
	// Chain A applies its deposit only after chain B has, so waiting on the chains one after
	// another would time out on A.
	deliveredA := deliverAfter(l2A, true, versionedNonce(1), func(int) bool { return deliveredB.Load() })
	env.addChain(t, l2A, &l1Contracts{Messenger: messengerA})
	env.addChain(t, l2B, &l1Contracts{Messenger: messengerB})

	var out bytes.Buffer
	require.NoError(t, syncChains(env.ctx, &out, env.conn, env.mgr, testWait(5*time.Second), false))
	require.True(t, deliveredA.Load())
	require.Equal(t, 2, env.sent())
	require.Contains(t, out.String(), "chain 901: L2 applied the sync; L2 paused=true")
	require.Contains(t, out.String(), "chain 902: L2 applied the sync; L2 paused=true")
}

func TestSyncL2Failures(t *testing.T) {
	reverting := common.HexToAddress("0x00000000000000000000000000000000000ba11e")
	env := newSyncEnv(t, types.GenesisAlloc{
		messengerA: {Code: messengerCode(t, messengerA, true, versionedNonce(3))},
		reverting:  {Code: contractCode(nil)},
		common.HexToAddress("0x00000000000000000000000000000000000bad00"): {Code: contractCode(map[string]method{
			"paused()": {ret: word(true)},
		})},
	})

	t.Run("message failed on L2", func(t *testing.T) {
		l2 := fakeL2(901, false)
		l2.views[predeploys.L2CrossDomainMessengerAddr] = map[string][]byte{sel("failedMessages(bytes32)"): word(true)}
		env.conn.Chains = nil
		env.addChain(t, l2, &l1Contracts{Messenger: messengerA})
		err := syncChains(env.ctx, &bytes.Buffer{}, env.conn, env.mgr, testWait(10*time.Second), true)
		require.ErrorIs(t, err, errMessageFailed)
		require.ErrorContains(t, err, "chain 901")
	})

	t.Run("deposit never lands", func(t *testing.T) {
		env.conn.Chains = nil
		env.addChain(t, fakeL2(901, false), &l1Contracts{Messenger: messengerA})
		err := syncChains(env.ctx, &bytes.Buffer{}, env.conn, env.mgr, testWait(2*time.Second), true)
		require.ErrorContains(t, err, "L2 did not apply L1 message nonce 3 (version 1)")
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("applied with a different value", func(t *testing.T) {
		l2 := fakeL2(901, false)
		l2.setL2Pause(false, versionedNonce(3))
		env.conn.Chains = nil
		env.addChain(t, l2, &l1Contracts{Messenger: messengerA})
		err := syncChains(env.ctx, &bytes.Buffer{}, env.conn, env.mgr, testWait(10*time.Second), true)
		require.ErrorIs(t, err, errAppliedMismatch)
	})

	t.Run("superseded by a later sync", func(t *testing.T) {
		l2 := fakeL2(901, false)
		l2.setL2Pause(false, versionedNonce(9))
		env.conn.Chains = nil
		env.addChain(t, l2, &l1Contracts{Messenger: messengerA})
		var out bytes.Buffer
		require.NoError(t, syncChains(env.ctx, &out, env.conn, env.mgr, testWait(10*time.Second), true))
		require.Contains(t, out.String(), "L2 already applied a later sync (L1 message nonce 9 (version 1)); L2 paused=false")
	})

	t.Run("a reverting sync is skipped and uses no nonce", func(t *testing.T) {
		// paused() answers, but syncETHLiquidityPause() reverts.
		revertingMessenger := common.HexToAddress("0x00000000000000000000000000000000000bad00")
		l2B := fakeL2(902, false)
		deliveredB := deliverAfter(l2B, true, versionedNonce(3), func(calls int) bool { return calls > 2 })
		env.conn.Chains = nil
		env.addChain(t, fakeL2(901, false), &l1Contracts{Messenger: revertingMessenger})
		env.addChain(t, l2B, &l1Contracts{Messenger: messengerA})
		before := env.sent()
		err := syncChains(env.ctx, &bytes.Buffer{}, env.conn, env.mgr, testWait(5*time.Second), true)
		require.ErrorContains(t, err, "chain 901: "+syncSignature+": would revert")
		require.NotContains(t, err.Error(), "chain 902")
		require.True(t, deliveredB.Load())
		require.Equal(t, before+1, env.sent())
	})

	t.Run("a candidate has a gas margin and a revert is caught before the send", func(t *testing.T) {
		data, err := syncFn.EncodeArgs()
		require.NoError(t, err)
		_, err = newCandidate(env.ctx, env.conn.L1, env.sender, reverting, data)
		require.ErrorContains(t, err, "would revert")

		estimate, err := env.conn.L1.EstimateGas(env.ctx, ethereum.CallMsg{From: env.sender, To: &messengerA, Data: data})
		require.NoError(t, err)
		candidate, err := newCandidate(env.ctx, env.conn.L1, env.sender, messengerA, data)
		require.NoError(t, err)
		require.Equal(t, estimate*6/5, candidate.GasLimit)
		require.Equal(t, messengerA, *candidate.To)
	})
}

func TestDevnetPauseForcesTheSync(t *testing.T) {
	guardian := common.HexToAddress("0x000000000000000000000000000000000000600d")
	scPaused, scFailing := common.HexToAddress("0x5c01"), common.HexToAddress("0x5c02")
	env := newSyncEnv(t, types.GenesisAlloc{
		guardian:   {Balance: oneETH},
		messengerA: {Code: messengerCode(t, messengerA, true, versionedNonce(2))},
		messengerB: {Code: messengerCode(t, messengerB, false, versionedNonce(2))},
		// Already paused: nothing to do, but the chain is still synced.
		scPaused: {Code: contractCode(map[string]method{
			"guardian()": {ret: word(guardian)}, "pausable(address)": {ret: word(false)}, "paused(address)": {ret: word(true)},
		})},
		// Not paused, and pause(address) reverts.
		scFailing: {Code: contractCode(map[string]method{
			"guardian()": {ret: word(guardian)}, "pausable(address)": {ret: word(true)}, "paused(address)": {ret: word(false)},
		})},
	})
	// L2 already reads paused=true, as L1 does: a plain sync would skip this chain.
	l2A := fakeL2(901, true)
	l2A.setL2Pause(true, versionedNonce(1))
	delivered := deliverAfter(l2A, true, versionedNonce(2), func(calls int) bool { return calls > 4 })
	env.addChain(t, l2A, &l1Contracts{Messenger: messengerA, SuperchainConfig: scPaused, Lockbox: common.HexToAddress("0xb1")})
	env.addChain(t, fakeL2(902, false), &l1Contracts{Messenger: messengerB, SuperchainConfig: scFailing, Lockbox: common.HexToAddress("0xb2")})

	var out bytes.Buffer
	err := devnetSetPause(env.ctx, &out, env.conn, newSimSender(env.l1, guardian), env.mgr, true, testWait(10*time.Second))
	require.ErrorContains(t, err, "pause("+common.HexToAddress("0xb2").Hex()+"): would revert")
	require.ErrorContains(t, err, "chains [902] not synced")
	require.ErrorContains(t, err, "rerun the command")
	require.True(t, delivered.Load())
	s := out.String()
	require.Contains(t, s, "(chains [901]): already paused; nothing to do")
	require.Contains(t, s, "chain 901: L1 paused=true, L2 paused=true: sync")
	require.Contains(t, s, "chain 901: L2 applied the sync; L2 paused=true")
	require.NotContains(t, s, "chain 902:")
	require.Equal(t, 1, env.sent())
}

// TestDevnetPauseSharesTheGuardianTxMgr covers devnet-* without a separate sync sender: one
// txmgr sends the guardian pause and then the sync from the same account.
func TestDevnetPauseSharesTheGuardianTxMgr(t *testing.T) {
	guardian := common.HexToAddress("0x000000000000000000000000000000000000600d")
	sc := common.HexToAddress("0x5c03")
	env := newSyncEnv(t, types.GenesisAlloc{
		guardian:   {Balance: oneETH},
		messengerA: {Code: messengerCode(t, messengerA, true, versionedNonce(2))},
		sc: {Code: contractCode(map[string]method{
			"guardian()": {ret: word(guardian)}, "pausable(address)": {ret: word(true)},
			"paused(address)": {ret: word(false)}, "pause(address)": {},
		})},
	})
	l2 := fakeL2(901, false)
	delivered := deliverAfter(l2, true, versionedNonce(2), func(calls int) bool { return calls > 2 })
	env.addChain(t, l2, &l1Contracts{Messenger: messengerA, SuperchainConfig: sc, Lockbox: common.HexToAddress("0xb1")})

	mgr := newSimSender(env.l1, guardian)
	var out bytes.Buffer
	require.NoError(t, devnetSetPause(env.ctx, &out, env.conn, mgr, mgr, true, testWait(10*time.Second)))
	require.Contains(t, out.String(), "pause("+common.HexToAddress("0xb1").Hex()+") included in L1 block")
	require.True(t, delivered.Load())
	require.Equal(t, 2, mgr.count(), "the guardian sends the pause and then the sync")
	require.Zero(t, env.sent())
}
