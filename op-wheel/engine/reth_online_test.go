package engine

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// newTestEnvelope builds a minimal ExecutionPayloadEnvelope whose BlockHash matches
// what CheckBlockHash computes, so the input to buildSyntheticPayload is self-consistent.
func newTestEnvelope(t *testing.T, extra []byte) *eth.ExecutionPayloadEnvelope {
	t.Helper()
	beaconRoot := common.HexToHash("0xbeac04")
	baseFee := uint256.NewInt(7)
	payload := &eth.ExecutionPayload{
		ParentHash:    common.HexToHash("0xa1"),
		FeeRecipient:  common.HexToAddress("0xfee"),
		StateRoot:     eth.Bytes32(common.HexToHash("0x57a7e")),
		ReceiptsRoot:  eth.Bytes32(common.HexToHash("0xec17")),
		PrevRandao:    eth.Bytes32(common.HexToHash("0xdad0")),
		BlockNumber:   42,
		GasLimit:      30_000_000,
		GasUsed:       21_000,
		Timestamp:     1_700_000_000,
		ExtraData:     extra,
		BaseFeePerGas: eth.Uint256Quantity(*baseFee),
		Transactions:  []eth.Data{},
	}
	env := &eth.ExecutionPayloadEnvelope{
		ParentBeaconBlockRoot: &beaconRoot,
		ExecutionPayload:      payload,
	}
	hash, _ := env.CheckBlockHash()
	payload.BlockHash = hash
	got, ok := env.CheckBlockHash()
	require.True(t, ok, "test envelope must have a self-consistent block hash: got %s", got)
	return env
}

func TestBuildSyntheticPayload_NonEmptyExtraData(t *testing.T) {
	original := newTestEnvelope(t, []byte{0x01, 0x02, 0x03})
	originalCopy := *original.ExecutionPayload // value copy for later immutability check
	originalExtra := bytes.Clone(original.ExecutionPayload.ExtraData)
	originalHash := original.ExecutionPayload.BlockHash

	synth := buildSyntheticPayload(original)

	// Extra data: last byte XOR 0xff, prefix unchanged.
	gotExtra := []byte(synth.ExecutionPayload.ExtraData)
	require.Equal(t, len(originalExtra), len(gotExtra))
	require.Equal(t, originalExtra[:len(originalExtra)-1], gotExtra[:len(gotExtra)-1])
	require.Equal(t, originalExtra[len(originalExtra)-1]^0xff, gotExtra[len(gotExtra)-1])

	// Block hash differs, matches CheckBlockHash recomputation.
	require.NotEqual(t, originalHash, synth.ExecutionPayload.BlockHash)
	recomputed, ok := synth.CheckBlockHash()
	require.True(t, ok, "synthetic envelope hash must be self-consistent")
	require.Equal(t, recomputed, synth.ExecutionPayload.BlockHash)

	// Other payload fields are carried over.
	require.Equal(t, originalCopy.ParentHash, synth.ExecutionPayload.ParentHash)
	require.Equal(t, originalCopy.StateRoot, synth.ExecutionPayload.StateRoot)
	require.Equal(t, originalCopy.BlockNumber, synth.ExecutionPayload.BlockNumber)
}

func TestBuildSyntheticPayload_EmptyExtraData(t *testing.T) {
	original := newTestEnvelope(t, nil)

	synth := buildSyntheticPayload(original)

	require.Equal(t, []byte{0x00}, []byte(synth.ExecutionPayload.ExtraData))
	require.NotEqual(t, original.ExecutionPayload.BlockHash, synth.ExecutionPayload.BlockHash)
	recomputed, ok := synth.CheckBlockHash()
	require.True(t, ok)
	require.Equal(t, recomputed, synth.ExecutionPayload.BlockHash)
}

func TestBuildSyntheticPayload_DoesNotMutateOriginal(t *testing.T) {
	original := newTestEnvelope(t, []byte{0x01, 0x02, 0x03})
	originalExtra := bytes.Clone(original.ExecutionPayload.ExtraData)
	originalHash := original.ExecutionPayload.BlockHash
	originalPayloadPtr := original.ExecutionPayload

	synth := buildSyntheticPayload(original)

	// Original envelope's payload pointer must not have been replaced.
	require.Same(t, originalPayloadPtr, original.ExecutionPayload)
	// Original payload contents must be intact.
	require.Equal(t, originalExtra, []byte(original.ExecutionPayload.ExtraData))
	require.Equal(t, originalHash, original.ExecutionPayload.BlockHash)
	// Synth must use a distinct payload object.
	require.NotSame(t, originalPayloadPtr, synth.ExecutionPayload)
}

func TestBuildSyntheticPayload_PreservesBeaconRoot(t *testing.T) {
	original := newTestEnvelope(t, []byte{0xaa})
	require.NotNil(t, original.ParentBeaconBlockRoot)

	synth := buildSyntheticPayload(original)

	require.NotNil(t, synth.ParentBeaconBlockRoot)
	require.Equal(t, *original.ParentBeaconBlockRoot, *synth.ParentBeaconBlockRoot)
}

// expectGetHeaderByTag wires a MockRPC call for getHeader(tag), populating the
// result pointer with the given header (or leaving it nil if hdr is nil).
func expectGetHeaderByTag(rpc *testutils.MockRPC, tag string, hdr *types.Header, err error) {
	rpc.Mock.On("CallContext", mock.Anything, mock.Anything, methodEthGetBlockByNumber, []any{tag, false}).
		Run(func(args mock.Arguments) {
			out := args.Get(1).(**types.Header)
			*out = hdr
		}).
		Return(err).
		Once()
}

func TestVerifyRewindState_AllMatch(t *testing.T) {
	// Use distinct Nonces to make each header's hash unique.
	head := &types.Header{Number: big.NewInt(100), Nonce: types.BlockNonce{0x01}}
	safe := &types.Header{Number: big.NewInt(80), Nonce: types.BlockNonce{0x02}}
	finalized := &types.Header{Number: big.NewInt(60), Nonce: types.BlockNonce{0x03}}

	rpc := new(testutils.MockRPC)
	expectGetHeaderByTag(rpc, "latest", head, nil)
	expectGetHeaderByTag(rpc, "safe", safe, nil)
	expectGetHeaderByTag(rpc, "finalized", finalized, nil)

	err := verifyRewindState(context.Background(), rpc, head.Hash(), safe.Hash(), finalized.Hash())
	require.NoError(t, err)
	rpc.AssertExpectations(t)
}

func TestVerifyRewindState_LatestMismatch(t *testing.T) {
	head := &types.Header{Number: big.NewInt(100), Nonce: types.BlockNonce{0x01}}
	safe := &types.Header{Number: big.NewInt(80), Nonce: types.BlockNonce{0x02}}
	finalized := &types.Header{Number: big.NewInt(60), Nonce: types.BlockNonce{0x03}}

	rpc := new(testutils.MockRPC)
	expectGetHeaderByTag(rpc, "latest", head, nil)
	expectGetHeaderByTag(rpc, "safe", safe, nil)
	expectGetHeaderByTag(rpc, "finalized", finalized, nil)

	want := common.HexToHash("0xdead")
	err := verifyRewindState(context.Background(), rpc, want, safe.Hash(), finalized.Hash())
	require.ErrorContains(t, err, "unexpected latest")
	require.ErrorContains(t, err, want.String())
	require.ErrorContains(t, err, head.Hash().String())
}

func TestVerifyRewindState_SafeMismatch(t *testing.T) {
	head := &types.Header{Number: big.NewInt(100), Nonce: types.BlockNonce{0x01}}
	safe := &types.Header{Number: big.NewInt(80), Nonce: types.BlockNonce{0x02}}
	finalized := &types.Header{Number: big.NewInt(60), Nonce: types.BlockNonce{0x03}}

	rpc := new(testutils.MockRPC)
	expectGetHeaderByTag(rpc, "latest", head, nil)
	expectGetHeaderByTag(rpc, "safe", safe, nil)
	expectGetHeaderByTag(rpc, "finalized", finalized, nil)

	want := common.HexToHash("0xbeef")
	err := verifyRewindState(context.Background(), rpc, head.Hash(), want, finalized.Hash())
	require.ErrorContains(t, err, "unexpected safe")
}

func TestVerifyRewindState_FinalizedMismatch(t *testing.T) {
	head := &types.Header{Number: big.NewInt(100), Nonce: types.BlockNonce{0x01}}
	safe := &types.Header{Number: big.NewInt(80), Nonce: types.BlockNonce{0x02}}
	finalized := &types.Header{Number: big.NewInt(60), Nonce: types.BlockNonce{0x03}}

	rpc := new(testutils.MockRPC)
	expectGetHeaderByTag(rpc, "latest", head, nil)
	expectGetHeaderByTag(rpc, "safe", safe, nil)
	expectGetHeaderByTag(rpc, "finalized", finalized, nil)

	want := common.HexToHash("0xfa11")
	err := verifyRewindState(context.Background(), rpc, head.Hash(), safe.Hash(), want)
	require.ErrorContains(t, err, "unexpected finalized")
}

func TestVerifyRewindState_NilLabel(t *testing.T) {
	head := &types.Header{Number: big.NewInt(100), Nonce: types.BlockNonce{0x01}}
	finalized := &types.Header{Number: big.NewInt(60), Nonce: types.BlockNonce{0x03}}

	rpc := new(testutils.MockRPC)
	// headSafeFinalized issues all three RPCs before we check any result,
	// so every tag needs an expectation even when `safe` is nil.
	expectGetHeaderByTag(rpc, "latest", head, nil)
	expectGetHeaderByTag(rpc, "safe", nil, nil) // engine hasn't advanced safe yet
	expectGetHeaderByTag(rpc, "finalized", finalized, nil)

	err := verifyRewindState(context.Background(), rpc,
		head.Hash(), common.HexToHash("0xbbbb"), finalized.Hash())
	require.ErrorContains(t, err, "unexpected safe: got nil")
}

func TestVerifyRewindState_RPCError(t *testing.T) {
	rpc := new(testutils.MockRPC)
	rpcErr := errors.New("connection refused")
	expectGetHeaderByTag(rpc, "latest", nil, rpcErr)

	err := verifyRewindState(context.Background(), rpc,
		common.HexToHash("0xaaaa"), common.HexToHash("0xbbbb"), common.HexToHash("0xcccc"))
	require.ErrorIs(t, err, rpcErr)
}
