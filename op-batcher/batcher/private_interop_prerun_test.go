package batcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-private-interop/builder"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
)

// revertError is what an execution client returns for a reverting eth_call: a JSON-RPC error
// with revert data.
type revertError struct{ data string }

func (e *revertError) Error() string          { return "execution reverted" }
func (e *revertError) ErrorCode() int         { return 3 }
func (e *revertError) ErrorData() interface{} { return e.data }

// fakeCaller is a projection execution client whose simulations succeed unless fail says
// otherwise.
type fakeCaller struct {
	mu        sync.Mutex
	calls     []ethereum.CallMsg
	at        []common.Hash
	overrides []StateOverrides
	fail      func(call int, msg ethereum.CallMsg) error
}

func (f *fakeCaller) CallContractAtHash(_ context.Context, msg ethereum.CallMsg, blockHash common.Hash, overrides StateOverrides) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, msg)
	f.at = append(f.at, blockHash)
	f.overrides = append(f.overrides, overrides)
	if f.fail != nil {
		if err := f.fail(len(f.calls)-1, msg); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func (f *fakeCaller) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func signedCarrier(t *testing.T, nonce uint64, to common.Address, gas uint64, data []byte, al types.AccessList) hexutil.Bytes {
	t.Helper()
	tx, err := types.SignNewTx(piKey, types.LatestSignerForChainID(piChainIDBig), &types.DynamicFeeTx{
		ChainID: piChainIDBig, Nonce: nonce, GasTipCap: new(big.Int), GasFeeCap: new(big.Int),
		Gas: gas, To: &to, Value: new(big.Int), Data: data, AccessList: al,
	})
	require.NoError(t, err)
	raw, err := tx.MarshalBinary()
	require.NoError(t, err)
	return raw
}

// TestCarrierStaticGasCheck: a carrier must declare at least max(intrinsic, EIP-7623 floor).
func TestCarrierStaticGasCheck(t *testing.T) {
	data := bytes.Repeat([]byte{0xff}, 1024)
	al := types.AccessList{{Address: predeploys.CrossL2InboxAddr, StorageKeys: []common.Hash{{0x01}}}}
	intrinsic, err := core.IntrinsicGas(data, al, nil, false, true, true, true)
	require.NoError(t, err)
	floor, err := core.FloorDataGas(data)
	require.NoError(t, err)
	require.Greater(t, floor, intrinsic, "a calldata-heavy carrier is bound by the floor")
	need := max(intrinsic, floor)

	decode := func(raw hexutil.Bytes) *types.Transaction {
		var tx types.Transaction
		require.NoError(t, tx.UnmarshalBinary(raw))
		return &tx
	}
	require.NoError(t, checkCarrierGas(decode(signedCarrier(t, 0, predeploys.CrossL2InboxAddr, need, data, al))))
	require.ErrorContains(t, checkCarrierGas(decode(signedCarrier(t, 0, predeploys.CrossL2InboxAddr, need-1, data, al))),
		"below the")
	// Between intrinsic and the floor: includable under pre-Prague rules, not on the projection.
	require.Error(t, checkCarrierGas(decode(signedCarrier(t, 0, predeploys.CrossL2InboxAddr, intrinsic, data, al))))

	built := &builder.BuiltRange{Blocks: []builder.BuiltBlock{
		{Number: 7, Txs: []hexutil.Bytes{signedCarrier(t, 0, predeploys.ClaimRegistryAddr, 100_000, nil, nil)}},
		{Number: 8, Txs: []hexutil.Bytes{
			signedCarrier(t, 1, predeploys.ClaimRegistryAddr, 100_000, nil, nil),
			signedCarrier(t, 2, predeploys.CrossL2InboxAddr, need-1, data, al),
		}},
	}}
	err = checkRangeCarrierGas(built)
	require.ErrorIs(t, err, ErrCarrierPreRun)
	require.ErrorContains(t, err, "block 8 tx 1")

	// The static check runs before any simulation.
	caller := &fakeCaller{}
	err = preRunCarriers(context.Background(), caller, common.Address{0x42}, piTerminal, built)
	require.ErrorIs(t, err, ErrCarrierPreRun)
	require.Zero(t, caller.callCount())
}

// TestCarrierPreRunSimulatesEveryCarrier: each carrier is eth_call'ed in order at the span parent
// with the batcher as sender, its own gas and access list, and a zero gas price.
func TestCarrierPreRunSimulatesEveryCarrier(t *testing.T) {
	al := types.AccessList{{Address: predeploys.CrossL2InboxAddr, StorageKeys: []common.Hash{{0x01}}}}
	built := &builder.BuiltRange{Blocks: []builder.BuiltBlock{
		{Number: 7, Txs: []hexutil.Bytes{
			signedCarrier(t, 0, predeploys.ClaimRegistryAddr, 500_000, []byte{1, 2, 3}, nil),
			signedCarrier(t, 1, predeploys.ClaimRegistryAddr, 100_000, []byte{4}, nil),
		}},
		{Number: 8, Txs: []hexutil.Bytes{signedCarrier(t, 2, predeploys.CrossL2InboxAddr, 60_000, []byte{5}, al)}},
	}}
	caller := &fakeCaller{}
	batcher := common.Address{0x42}
	require.NoError(t, preRunCarriers(context.Background(), caller, batcher, piTerminal, built))
	require.Len(t, caller.calls, 3)
	for i, want := range []struct {
		to   common.Address
		gas  uint64
		data []byte
	}{
		{predeploys.ClaimRegistryAddr, 500_000, []byte{1, 2, 3}},
		{predeploys.ClaimRegistryAddr, 100_000, []byte{4}},
		{predeploys.CrossL2InboxAddr, 60_000, []byte{5}},
	} {
		msg := caller.calls[i]
		require.Equal(t, batcher, msg.From)
		require.Equal(t, want.to, *msg.To)
		require.Equal(t, want.gas, msg.Gas)
		require.Equal(t, want.data, msg.Data)
		require.Zero(t, msg.GasPrice.Sign())
		require.Equal(t, piTerminal, caller.at[i], "simulated at the span parent")
	}
	require.Equal(t, al, caller.calls[2].AccessList)

	// No client configured is a pre-run failure, not a silent skip.
	require.ErrorIs(t, preRunCarriers(context.Background(), nil, batcher, piTerminal, built), ErrCarrierPreRun)

	// A transport failure is not a verdict on the span and stays retryable.
	flaky := &fakeCaller{fail: func(int, ethereum.CallMsg) error { return errors.New("connection refused") }}
	err := preRunCarriers(context.Background(), flaky, batcher, piTerminal, built)
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrCarrierPreRun)
}

// TestCarrierPreRunRejectsUnderGassedReplay drives the seam: an under-gassed import replay
// (GasPolicyOverride, as the reverted-replay acceptance test sets it) reverts in the simulation,
// the range is never proven or framed, and the verdict is sticky rather than silently retried.
func TestCarrierPreRunRejectsUnderGassedReplay(t *testing.T) {
	// Every fixture block renders an export and an import: block 901 is the claim, the output
	// record, the export replay and the import replay (tx 3).
	enc, ranges, _ := piEncoderWithTxs(t, piProjectionTxs())
	caller := piExecutionMock(enc, ranges)
	caller.fail = func(_ int, msg ethereum.CallMsg) error {
		if *msg.To == predeploys.CrossL2InboxAddr {
			return &revertError{data: "0x"}
		}
		return nil
	}
	proved := 0
	enc.cfg.Prove = func(context.Context, *builder.BuiltRange, []byte, RangeStart) ([]byte, error) {
		proved++
		return nil, errors.New("must not be reached")
	}
	co := piFill(t, enc, 901, piCadence)

	var closeErr error
	require.Eventually(t, func() bool {
		closeErr = co.Close()
		return closeErr != nil && !errors.Is(closeErr, errPrivateProofPending)
	}, 5*time.Second, time.Millisecond)
	require.ErrorIs(t, closeErr, ErrCarrierPreRun)
	require.ErrorContains(t, closeErr, "block 901 tx 3")
	require.ErrorContains(t, closeErr, "revert data: 0x")
	require.Zero(t, proved, "a span with a failing carrier is never proven")
	require.Nil(t, co.BuiltRange())
	var buf bytes.Buffer
	_, err := co.OutputFrame(&buf, 100_000)
	require.ErrorIs(t, err, io.EOF)

	calls := caller.callCount()
	require.ErrorIs(t, co.Close(), ErrCarrierPreRun)
	require.Equal(t, calls, caller.callCount(), "the verdict is sticky, not re-simulated")

	// The hook disables the pre-run so the chain's own rejection can be tested.
	enc2, ranges2, _ := piEncoderWithTxs(t, piProjectionTxs())
	piExecutionMock(enc2, ranges2)
	enc2.cfg.Caller = caller
	enc2.cfg.TestHooks = &PrivateInteropTestHooks{SkipCarrierPreRun: true}
	enc2.cfg.Prove = piMockProver(enc2)
	co2 := piFill(t, enc2, 901, piCadence)
	require.Eventually(t, func() bool { return co2.Close() == nil }, 5*time.Second, time.Millisecond)
	require.NotEmpty(t, co2.BuiltRange().Frames)
	require.Equal(t, calls, caller.callCount(), "SkipCarrierPreRun makes no calls")
}

// TestPrivateInteropTestHooks covers the remaining hooks: GasPolicyOverride applies while set and
// is undone once cleared, MutateProof alters the verified envelope, and SkipAdmissionPreflight is
// what lets such an envelope be framed at all.
func TestPrivateInteropTestHooks(t *testing.T) {
	newSeam := func(hooks *PrivateInteropTestHooks) *PrivateInteropEncoder {
		enc, ranges, _ := piEncoderWithTxs(t, piProjectionTxs())
		piExecutionMock(enc, ranges)
		enc.cfg.TestHooks = hooks
		enc.cfg.Prove = piMockProver(enc)
		return enc
	}
	closeOK := func(co *renderChannelOut) error {
		var err error
		require.Eventually(t, func() bool {
			err = co.Close()
			return !errors.Is(err, errPrivateProofPending)
		}, 5*time.Second, time.Millisecond)
		return err
	}
	claimGas := func(co *renderChannelOut) uint64 {
		var tx types.Transaction
		require.NoError(t, tx.UnmarshalBinary(co.BuiltRange().Blocks[0].Txs[0]))
		return tx.Gas()
	}

	override := render.DefaultGasPolicy()
	override.GasLimitClaim = 777_777
	hooks := &PrivateInteropTestHooks{GasPolicyOverride: &override}
	enc := newSeam(hooks)
	co := piFill(t, enc, 901, piCadence)
	require.NoError(t, closeOK(co))
	require.Equal(t, uint64(777_777), claimGas(co))
	hooks.GasPolicyOverride = nil
	co = piFill(t, enc, 901, piCadence)
	require.NoError(t, closeOK(co))
	require.Equal(t, render.DefaultGasPolicy().GasLimitClaim, claimGas(co), "a cleared override restores the policy")

	flip := func(p []byte) []byte {
		out := bytes.Clone(p)
		out[len(out)-1] ^= 1
		return out
	}
	enc = newSeam(&PrivateInteropTestHooks{MutateProof: flip})
	co = piFill(t, enc, 901, piCadence)
	require.ErrorContains(t, closeOK(co), "projection admission", "a mutated envelope fails the admission preflight")
	require.Nil(t, co.BuiltRange())

	enc = newSeam(&PrivateInteropTestHooks{MutateProof: flip, SkipAdmissionPreflight: true})
	co = piFill(t, enc, 901, piCadence)
	require.NoError(t, closeOK(co))
	require.NotEmpty(t, co.BuiltRange().Frames)
	statement, err := projectionPreflight(enc.cfg.Rollup, co.BuiltRange(), enc.cfg.Ranges.(*staticRanges).start)
	require.NoError(t, err)
	require.Error(t, projection.ExecutionMockVerifier{}.Verify(*statement, co.BuiltRange().Claim.Proof),
		"the published proof is the mutated one")
}

// piProjectionTxs is a transaction builder against the real predeploy addresses.
func piProjectionTxs() *render.BatcherTxBuilder {
	txs := render.NewBatcherTxBuilder(piChainIDBig, render.DefaultGasPolicy(), render.PrivateKeySigner(piKey, piChainIDBig))
	txs.SetRegistry(predeploys.ClaimRegistryAddr)
	txs.SetEventReplayer(predeploys.EventReplayerAddr)
	return txs
}

// piMockProver returns the execution-mock envelope of the candidate's own preflight statement.
func piMockProver(enc *PrivateInteropEncoder) func(context.Context, *builder.BuiltRange, []byte, RangeStart) ([]byte, error) {
	return func(_ context.Context, candidate *builder.BuiltRange, _ []byte, start RangeStart) ([]byte, error) {
		statement, err := projectionPreflight(enc.cfg.Rollup, candidate, start)
		if err != nil {
			return nil, err
		}
		return projection.ExecutionMockProof(*statement), nil
	}
}

// piFill opens a channel and adds count private blocks from first.
func piFill(t *testing.T, enc *PrivateInteropEncoder, first, count uint64) *renderChannelOut {
	t.Helper()
	out, err := enc.ChannelOut(ChannelConfig{MaxFrameSize: 100_000, CompressorConfig: compressor.Config{CompressionAlgo: derive.Zlib}}, enc.cfg.PrivateRollup)
	require.NoError(t, err)
	co := out.(*renderChannelOut)
	for i := range count {
		p := piPayload(t, first+i)
		require.NoError(t, enc.PrepareBlock(t.Context(), p))
		_, err := co.AddBlock(enc.cfg.PrivateRollup, p)
		require.NoError(t, err)
	}
	return co
}

// TestCarrierPreRunAtGenesisParent: at the projection genesis L1Block has no batcher hash yet, so a
// plain eth_call of the first claim reverts ClaimRegistry_NotBatcher. The pre-run overrides the
// batcher hash to the batcher's own, which is what the span's first L1-info deposit writes.
func TestCarrierPreRunAtGenesisParent(t *testing.T) {
	batcher := common.Address{0x42}
	genesis := common.Hash{0x9e}
	// A projection whose ClaimRegistry behaves like the real one at the genesis parent: its
	// batcher check reads L1Block's batcher hash slot, empty unless overridden.
	registry := &fakeCaller{fail: func(call int, msg ethereum.CallMsg) error { return nil }}
	registry.fail = func(call int, msg ethereum.CallMsg) error {
		if *msg.To != predeploys.ClaimRegistryAddr {
			return nil
		}
		hash := registry.overrides[call][predeploys.L1BlockAddr][common.BigToHash(big.NewInt(4))]
		if hash != common.BytesToHash(msg.From.Bytes()) {
			return &revertError{data: "0xcd9fa000"} // ClaimRegistry_NotBatcher()
		}
		return nil
	}
	built := &builder.BuiltRange{Blocks: []builder.BuiltBlock{{Number: 1, Txs: []hexutil.Bytes{
		signedCarrier(t, 0, predeploys.ClaimRegistryAddr, 500_000, []byte{0x46, 0xe3, 0xee, 0xf2}, nil),
		signedCarrier(t, 1, predeploys.ClaimRegistryAddr, 100_000, []byte{0x61, 0x84, 0xd0, 0x8e}, nil),
	}}}}
	require.NoError(t, preRunCarriers(context.Background(), registry, batcher, genesis, built))
	require.Len(t, registry.overrides, 2)
	for i, o := range registry.overrides {
		require.Equal(t, StateOverrides{predeploys.L1BlockAddr: {
			common.BigToHash(big.NewInt(4)): common.BytesToHash(batcher.Bytes()),
		}}, o, "call %d", i)
		require.Equal(t, genesis, registry.at[i])
	}

	// Without the override the same registry reverts, as a real projection did at genesis.
	_, err := registry.CallContractAtHash(context.Background(), ethereum.CallMsg{From: batcher, To: &predeploys.ClaimRegistryAddr}, genesis, nil)
	require.ErrorContains(t, err, "execution reverted")
}

// TestRPCProjectionCallerEncodesStateOverride pins the eth_call wire form: call object, block hash,
// and the stateDiff override.
func TestRPCProjectionCallerEncodesStateOverride(t *testing.T) {
	var got []json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Equal(t, "eth_call", req.Method)
		got = req.Params
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x01"}`, req.ID)
	}))
	defer srv.Close()
	cl, err := rpc.Dial(srv.URL)
	require.NoError(t, err)
	defer cl.Close()

	batcher := common.Address{0x42}
	to := predeploys.ClaimRegistryAddr
	out, err := newRPCProjectionCaller(cl).CallContractAtHash(context.Background(), ethereum.CallMsg{
		From: batcher, To: &to, Gas: 500_000, GasPrice: new(big.Int), Value: new(big.Int), Data: []byte{1, 2},
	}, common.Hash{0x9e}, carrierOverrides(batcher))
	require.NoError(t, err)
	require.Equal(t, []byte{1}, out)
	require.Len(t, got, 3)
	require.JSONEq(t, `{"from":"0x4200000000000000000000000000000000000000","to":"0x420000000000000000000000000000000000002e",
		"gas":"0x7a120","gasPrice":"0x0","value":"0x0","input":"0x0102"}`, string(got[0]))
	require.JSONEq(t, `{"blockHash":"0x9e00000000000000000000000000000000000000000000000000000000000000"}`, string(got[1]))
	require.JSONEq(t, `{"0x4200000000000000000000000000000000000015":{"stateDiff":{
		"0x0000000000000000000000000000000000000000000000000000000000000004":"0x0000000000000000000000004200000000000000000000000000000000000000"}}}`, string(got[2]))
}
