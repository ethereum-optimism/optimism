package projection_test

import (
	stdcontext "context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

type history struct {
	blocks  map[uint64]*eth.ExecutionPayloadEnvelope
	missing uint64
	reads   int
}

func (h *history) PayloadByNumber(_ stdcontext.Context, n uint64) (*eth.ExecutionPayloadEnvelope, error) {
	h.reads++
	if n == h.missing {
		return nil, errors.New("temporarily unavailable")
	}
	return h.blocks[n], nil
}

func recoveryHistory(t *testing.T, last uint64) (*history, eth.BlockID, eth.BlockID) {
	h := &history{blocks: make(map[uint64]*eth.ExecutionPayloadEnvelope), missing: ^uint64(0)}
	genesis := eth.BlockID{Hash: common.Hash{0xee}, Number: 0}
	parent := genesis
	for n := uint64(1); n <= last; n++ {
		p := &eth.ExecutionPayload{BlockNumber: hexutil.Uint64(n), Timestamp: hexutil.Uint64(1000 + 2*n), BlockHash: common.BigToHash(new(big.Int).SetUint64(n)), ParentHash: parent.Hash, Transactions: []hexutil.Bytes{{0x7e, byte(n)}}}
		if n == 2 {
			p.Transactions = append(p.Transactions, transaction(t, predeploys.ClaimRegistryAddr, wire.EncodeOutput(common.Hash{2}), nil))
		}
		h.blocks[n] = &eth.ExecutionPayloadEnvelope{ExecutionPayload: p}
		parent = p.ID()
	}
	return h, parent, genesis
}

func TestContinuationUsesSurvivingBlockNotPreviousRangeEndpoint(t *testing.T) {
	h, parent, genesis := recoveryHistory(t, 5)
	var collector projection.ContextCollector
	ctx, err := collector.Resolve(t.Context(), h, parent, genesis, common.Hash{8})
	require.NoError(t, err)
	require.Equal(t, uint64(2), ctx.Anchor.Number)
	require.Equal(t, common.Hash{2}, ctx.OutputRoot)
	want := common.Hash{}
	for n := uint64(5); n > 2; n-- {
		want = projection.RecoveryStep(want, h.blocks[n].ExecutionPayload)
	}
	require.Equal(t, want, ctx.RecoveryHash)
	// Missing data never authorizes silently skipping a checkpoint.
	collector.Reset()
	h.missing = 2
	_, err = collector.Resolve(t.Context(), h, parent, genesis, common.Hash{8})
	require.ErrorIs(t, err, projection.ErrContextUnavailable)
	h.missing = ^uint64(0)
	got, err := collector.Resolve(t.Context(), h, parent, genesis, common.Hash{8})
	require.NoError(t, err)
	require.Equal(t, ctx, got)
}

func TestContinuationLongOutageIsIncrementalAndReorgBound(t *testing.T) {
	h, parent, genesis := recoveryHistory(t, 400)
	var collector projection.ContextCollector
	for range 3 {
		before := h.reads
		_, err := collector.Resolve(t.Context(), h, parent, genesis, common.Hash{8})
		require.ErrorIs(t, err, projection.ErrContextUnavailable)
		require.LessOrEqual(t, h.reads-before, 129)
	}
	result, err := collector.Resolve(t.Context(), h, parent, genesis, common.Hash{8})
	require.NoError(t, err)
	require.Equal(t, uint64(2), result.Anchor.Number)
	// Even a completed cache cannot authenticate history after its parent reorgs.
	h.blocks[400].ExecutionPayload.BlockHash = common.Hash{0xaa}
	_, err = collector.Resolve(t.Context(), h, parent, genesis, common.Hash{8})
	require.ErrorIs(t, err, projection.ErrContextUnavailable)
	collector.Reset()
	_, err = collector.Resolve(t.Context(), h, parent, genesis, common.Hash{8})
	require.ErrorIs(t, err, projection.ErrContextUnavailable)
}

func TestCanonicalCheckpointGrammar(t *testing.T) {
	output := transaction(t, predeploys.ClaimRegistryAddr, wire.EncodeOutput(common.Hash{2}), nil)
	for _, tc := range []struct {
		name   string
		txs    []hexutil.Bytes
		accept bool
	}{
		{"output", []hexutil.Bytes{output}, true},
		{"fallback", []hexutil.Bytes{{0x7e}, {0x7d}}, true},
		{"no_output", []hexutil.Bytes{export(t)}, false},
		{"duplicate", []hexutil.Bytes{output, output}, false},
		{"late", []hexutil.Bytes{export(t), output}, false},
		{"empty_transaction", []hexutil.Bytes{nil}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := projection.CanonicalOutput(tc.txs)
			require.Equal(t, tc.accept, err == nil)
		})
	}
	// A protocol transaction cannot manufacture a checkpoint through its payload.
	root, err := projection.CanonicalOutput([]hexutil.Bytes{append([]byte{0x7e}, output...)})
	require.NoError(t, err)
	require.Zero(t, root)
}

func TestContinuationAdmissionAndIntermediateRoots(t *testing.T) {
	for _, field := range []string{"anchor", "root", "recovery", "parent"} {
		t.Run(field, func(t *testing.T) {
			v := vectors(t)[0]
			var tx types.Transaction
			require.NoError(t, tx.UnmarshalBinary(v.Blocks[0].Transactions[0]))
			c, err := wire.DecodeClaim(tx.Data())
			require.NoError(t, err)
			switch field {
			case "anchor":
				c.AnchorBlock--
			case "root":
				c.AnchorOutputRoot[0]++
			case "recovery":
				c.RecoveryHash[0]++
			case "parent":
				c.ParentOutputRoot[0]++
			}
			data, err := wire.EncodePostClaim(c)
			require.NoError(t, err)
			v.Blocks[0].Transactions[0] = transaction(t, predeploys.ClaimRegistryAddr, data, nil)
			_, err = projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, projection.StubVerifier{})
			require.Error(t, err)
		})
	}
	v := vectors(t)[0]
	want, err := projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, projection.StubVerifier{})
	require.NoError(t, err)
	v.Blocks[1].Transactions[0] = transaction(t, predeploys.ClaimRegistryAddr, wire.EncodeOutput(common.Hash{99}), nil)
	got, err := projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, projection.StubVerifier{})
	require.NoError(t, err)
	require.NotEqual(t, want.ProjectionHash, got.ProjectionHash)
	// A later proof rejection must reject the entire submitted range.
	_, err = projection.ValidateProjectionRange(&v.Config, context(), v.Blocks, bindingVerifier{*want, []byte("dummy")})
	require.Error(t, err)
	// Recovery permits a different parent output only as an explicitly unproven
	// transition from the canonical anchor; equality alone cannot authenticate it.
	ctx := context()
	ctx.Continuation.Anchor.Number = 2
	ctx.Continuation.RecoveryHash = common.Hash{77}
	c := spanBuilder{t, mockConfig(), ctx}.claimFields(10, 12, []byte("dummy"))
	c.ParentOutputRoot = common.Hash{78}
	data, err := wire.EncodePostClaim(c)
	require.NoError(t, err)
	v.Blocks[0].Transactions[0] = transaction(t, predeploys.ClaimRegistryAddr, data, nil)
	_, err = projection.ValidateProjectionRange(&v.Config, ctx, v.Blocks, projection.StubVerifier{})
	require.NoError(t, err)
}

func TestPrivateOutputRequiresWithdrawalCommitment(t *testing.T) {
	_, err := projection.PrivateOutput(&eth.ExecutionPayload{})
	require.Error(t, err)
	withdrawals := common.Hash{4}
	p := &eth.ExecutionPayload{BlockHash: common.Hash{2}, StateRoot: eth.Bytes32{3}, WithdrawalsRoot: &withdrawals}
	root, err := projection.PrivateOutput(p)
	require.NoError(t, err)
	require.Equal(t, common.Hash(eth.OutputRoot(&eth.OutputV0{StateRoot: p.StateRoot, MessagePasserStorageRoot: eth.Bytes32(withdrawals), BlockHash: p.BlockHash})), root)
}

// Headers are RLP-encoded so both clients hash the exact same canonical history.
type recoveryVectorBlock struct {
	Header       hexutil.Bytes   `json:"header"`
	Transactions []hexutil.Bytes `json:"transactions"`
}
type recoveryVector struct {
	Blocks       []recoveryVectorBlock `json:"blocks"`
	RecoveryHash common.Hash           `json:"recovery_hash"`
}

func TestSharedRecoveryTranscript(t *testing.T) {
	const path = "testdata/recovery.json"
	if *updateVectors {
		var v recoveryVector
		parent := common.Hash{0xaa}
		for n := uint64(3); n <= 5; n++ {
			deposit := &optypes.DepositTx{SourceHash: common.Hash{byte(n)}, From: common.Address{1}, To: &predeploys.L1BlockAddr, Value: new(big.Int), Gas: 100000, Data: []byte{byte(n), 0x01, 0x02}}
			raw, err := deposit.MarshalBinary()
			require.NoError(t, err)
			header := &types.Header{ParentHash: parent, UncleHash: types.EmptyUncleHash, Root: common.Hash{byte(n)}, Number: new(big.Int).SetUint64(n), Difficulty: new(big.Int), GasLimit: 30000000, Time: 1000 + 2*n, Extra: []byte("recovery")}
			encoded, err := rlp.EncodeToBytes(header)
			require.NoError(t, err)
			v.Blocks = append(v.Blocks, recoveryVectorBlock{Header: encoded, Transactions: []hexutil.Bytes{raw}})
			parent = header.Hash()
		}
		for i := len(v.Blocks) - 1; i >= 0; i-- {
			v.RecoveryHash = projection.RecoveryStep(v.RecoveryHash, vectorPayload(t, v.Blocks[i]))
		}
		raw, err := json.MarshalIndent(v, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, append(raw, '\n'), 0644))
	}
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var v recoveryVector
	require.NoError(t, json.Unmarshal(raw, &v))
	var hash common.Hash
	for i := len(v.Blocks) - 1; i >= 0; i-- {
		hash = projection.RecoveryStep(hash, vectorPayload(t, v.Blocks[i]))
	}
	require.Equal(t, v.RecoveryHash, hash)
}
func vectorPayload(t *testing.T, b recoveryVectorBlock) *eth.ExecutionPayload {
	var header types.Header
	require.NoError(t, rlp.DecodeBytes(b.Header, &header))
	return &eth.ExecutionPayload{BlockHash: header.Hash(), ParentHash: header.ParentHash, BlockNumber: hexutil.Uint64(bigs.Uint64Strict(header.Number)), Timestamp: hexutil.Uint64(header.Time), Transactions: b.Transactions}
}
