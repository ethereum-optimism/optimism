package projection_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/codec"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// The adversarial review's P0-1 probe (TestReviewInvalidTxCarriersAreAdmitted), kept as a
// regression test. The base three-block span carries an import re-signed either with gas 21000
// (below its intrinsic gas and its EIP-7623 floor) or with nonce 1,000,000. Before the fix both
// were admitted with a genuine sp1-private-projection-v1 mock envelope over the admission public
// values, and op-reth's payload builder then skipped the invalid import: a hidden executing
// message. Now admission rejects the under-gassed span, and the nonce gap, which pure admission
// cannot see, invalidates the block at execution (ProjectionSequencerTxInvalid).

// probeImportIndex is the import in block 2 of withOutputs(base): output record, export, import.
const probeImportIndex = 2

func probeSpan(t *testing.T, change func(*types.DynamicFeeTx)) (spanBuilder, span) {
	b := spanBuilder{t, sp1Config(true), context()}
	s := b.withOutputs(b.base(nil))
	if change != nil {
		s[2].Transactions[probeImportIndex] = mutate(t, s[2].Transactions[probeImportIndex], change)
	}
	return b, s
}

// probeRecordsRoot rebuilds the records root (public-values word 17) of s from its published
// transactions, independently of admission: per block, u64be(number, timestamp, epoch, txCount),
// then per transaction sender, u64be(nonce, gas), to, u64be(len(data)), data, with the claim's
// proof stripped.
func probeRecordsRoot(t *testing.T, first uint64, s span) common.Hash {
	signer := types.LatestSignerForChainID(context().ChainID)
	var leaves []common.Hash
	for i, blk := range s {
		var rec bytes.Buffer
		put := func(n uint64) { _ = binary.Write(&rec, binary.BigEndian, n) }
		put(first + uint64(i))
		put(blk.Timestamp)
		put(blk.Epoch)
		put(uint64(len(blk.Transactions)))
		for j, raw := range blk.Transactions {
			var tx types.Transaction
			require.NoError(t, tx.UnmarshalBinary(raw))
			sender, err := types.Sender(signer, &tx)
			require.NoError(t, err)
			data := tx.Data()
			if i == 0 && j == 0 {
				c, err := wire.DecodeClaim(data)
				require.NoError(t, err)
				c.Proof = nil
				data, err = wire.EncodePostClaim(c)
				require.NoError(t, err)
			}
			rec.Write(sender.Bytes())
			put(tx.Nonce())
			put(tx.Gas())
			rec.Write(tx.To().Bytes())
			put(uint64(len(data)))
			rec.Write(data)
		}
		leaves = append(leaves, crypto.Keccak256Hash([]byte{0}, rec.Bytes()))
	}
	return projection.RecordsRoot(leaves)
}

func withMockEnvelope(t *testing.T, b spanBuilder, s span, statement *projection.Statement) (span, []byte) {
	pv := projection.PublicValues(statement)
	env := projection.MockEnvelope(b.cfg.ProgramVKey, pv)
	out := s.clone()
	mutateClaim(t, out, func(c *codec.RangeClaim) { c.Proof = env })
	return out, env
}

func probeVerifier(t *testing.T, b spanBuilder) projection.ProofVerifier {
	v, err := projection.VerifierFor(&b.cfg, b.ctx.ChainID)
	require.NoError(t, err)
	return v
}

// TestInvalidCarrierProbeUnderGassedImportRejected: an import signed with gas 21000 is rejected by
// admission even with an envelope that is genuine for the statement the old admission computed.
func TestInvalidCarrierProbeUnderGassedImportRejected(t *testing.T) {
	b, honest := probeSpan(t, nil)
	honestStatement, err := projection.ValidateProjectionRange(&b.cfg, b.ctx, honest, projection.StubVerifier{})
	require.NoError(t, err)
	require.Equal(t, honestStatement.ProjectionHash, probeRecordsRoot(t, 10, honest), "independent records root")

	_, under := probeSpan(t, func(tx *types.DynamicFeeTx) { tx.Gas = 21_000 })
	var imp types.Transaction
	require.NoError(t, imp.UnmarshalBinary(under[2].Transactions[probeImportIndex]))
	require.Equal(t, predeploys.CrossL2InboxAddr, *imp.To())
	intrinsic, err := core.IntrinsicGas(imp.Data(), imp.AccessList(), nil, false, true, true, true)
	require.NoError(t, err)
	floor, err := core.FloorDataGas(imp.Data())
	require.NoError(t, err)
	require.Equal(t, uint64(28128), intrinsic, "the review's intrinsic gas")
	require.Equal(t, uint64(23320), floor, "the review's EIP-7623 floor")
	need, err := projection.MinTxGas(imp.Data(), imp.AccessList())
	require.NoError(t, err)
	require.Equal(t, intrinsic, need)

	// The statement the pre-fix admission computed for the under-gassed span: gas enters only the
	// records root; outputs, messages and the claim are unchanged. Its mock envelope is genuine.
	old := *honestStatement
	old.ProjectionHash = probeRecordsRoot(t, 10, under)
	require.NotEqual(t, honestStatement.ProjectionHash, old.ProjectionHash)
	published, env := withMockEnvelope(t, b, under, &old)
	verifier := probeVerifier(t, b)
	require.NoError(t, verifier.Verify(old, env), "the envelope is genuine for the old statement")

	_, err = projection.ValidateProjectionRange(&b.cfg, b.ctx, published, verifier)
	require.ErrorContains(t, err, "transaction gas below intrinsic or calldata floor")

	// One gas short of the minimum is rejected too; the minimum itself is admitted.
	for gas, accept := range map[uint64]bool{need - 1: false, need: true} {
		b, s := probeSpan(t, func(tx *types.DynamicFeeTx) { tx.Gas = gas })
		statement, err := projection.ValidateProjectionRange(&b.cfg, b.ctx, s, projection.StubVerifier{})
		if !accept {
			require.ErrorContains(t, err, "transaction gas below intrinsic or calldata floor", "gas %d", gas)
			continue
		}
		require.NoError(t, err, "gas %d", gas)
		published, _ := withMockEnvelope(t, b, s, statement)
		_, err = projection.ValidateProjectionRange(&b.cfg, b.ctx, published, probeVerifier(t, b))
		require.NoError(t, err, "gas %d", gas)
	}
}

// TestInvalidCarrierProbeNonceGapInvalidAtExecution: admission is pure and cannot see nonces, so
// an import re-signed with nonce 1,000,000 is still admitted with a genuine envelope. Execution
// then invalidates its block: the reference model rejects it as ProjectionSequencerTxInvalid, and
// the shared execution vector nonce_gap pins that outcome for op-reth's import, payload job and
// FCU pre-check and for Kona.
func TestInvalidCarrierProbeNonceGapInvalidAtExecution(t *testing.T) {
	b, gap := probeSpan(t, func(tx *types.DynamicFeeTx) { tx.Nonce = 1_000_000 })
	statement, err := projection.ValidateProjectionRange(&b.cfg, b.ctx, gap, projection.StubVerifier{})
	require.NoError(t, err)
	published, _ := withMockEnvelope(t, b, gap, statement)
	_, err = projection.ValidateProjectionRange(&b.cfg, b.ctx, published, probeVerifier(t, b))
	require.NoError(t, err, "nonces are not an admission rule")

	decode := func(raw hexutil.Bytes) *types.Transaction {
		var tx types.Transaction
		require.NoError(t, tx.UnmarshalBinary(raw))
		return &tx
	}
	imp := decode(gap[2].Transactions[probeImportIndex])
	index := uint64(0)
	require.Equal(t,
		executionOutcome{Valid: false, Error: "ProjectionSequencerTxInvalid", TxIndex: &index},
		executeReference(t, []*types.Transaction{imp}, true))
	_, honest := probeSpan(t, nil)
	require.True(t, executeReference(t, []*types.Transaction{decode(honest[2].Transactions[probeImportIndex])}, true).Valid,
		"the same import at the account's nonce executes")

	raw, err := os.ReadFile(executionVectorsPath)
	require.NoError(t, err)
	var vectors executionVectors
	require.NoError(t, json.Unmarshal(raw, &vectors))
	found := false
	for _, c := range vectors.Cases {
		if c.Name == "nonce_gap" {
			found = true
			require.Equal(t, "ProjectionSequencerTxInvalid", c.Projection.Error)
		}
	}
	require.True(t, found, "the nonce_gap execution vector")
}
