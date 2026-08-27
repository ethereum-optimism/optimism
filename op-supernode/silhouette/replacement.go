package silhouette

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// ReplacementBuilder prepares the stock deposits-only replacement payload in P's real execution
// engine. It is deliberately narrower than an Engine API proxy: the shim calls it only after the
// supernode has durably denied the exact proof fact immediately above parent.
type ReplacementBuilder interface {
	BuildReplacement(ctx context.Context, parent eth.L2BlockRef, attrs *eth.PayloadAttributes) (*eth.ExecutionPayloadEnvelope, error)
}

// EngineReplacementBuilder performs the same build dance a normal op-node performs, against P's
// private EL: build, seal, import, then make the valid payload the engine head. P's LightCL observes
// and adopts the replacement through the verifier's ordinary rollup route; no op-node behavior is
// changed or reimplemented here.
type EngineReplacementBuilder struct {
	engine *sources.EngineClient
}

func NewEngineReplacementBuilder(engine *sources.EngineClient) *EngineReplacementBuilder {
	return &EngineReplacementBuilder{engine: engine}
}

func (b *EngineReplacementBuilder) BuildReplacement(ctx context.Context, parent eth.L2BlockRef, attrs *eth.PayloadAttributes) (*eth.ExecutionPayloadEnvelope, error) {
	if b == nil || b.engine == nil {
		return nil, fmt.Errorf("replacement engine is not configured")
	}
	if attrs == nil || !attrs.NoTxPool || !attrs.IsDepositsOnly() {
		return nil, fmt.Errorf("replacement attributes must be deposits-only with the transaction pool disabled")
	}

	safe, err := b.engine.L2BlockRefByLabel(ctx, eth.Safe)
	if err != nil {
		return nil, fmt.Errorf("read private EL safe head: %w", err)
	}
	finalized, err := b.engine.L2BlockRefByLabel(ctx, eth.Finalized)
	if err != nil {
		return nil, fmt.Errorf("read private EL finalized head: %w", err)
	}
	if finalized.Number > parent.Number {
		return nil, fmt.Errorf("cannot replace block above parent %s: private EL finalized head is %s", parent, finalized)
	}
	if safe.Number > parent.Number {
		safe = parent
	}

	state := &eth.ForkchoiceState{
		HeadBlockHash:      parent.Hash,
		SafeBlockHash:      safe.Hash,
		FinalizedBlockHash: finalized.Hash,
	}
	started, err := b.engine.ForkchoiceUpdate(ctx, state, attrs)
	if err != nil {
		return nil, fmt.Errorf("start deposits-only replacement on private EL: %w", err)
	}
	if started.PayloadStatus.Status != eth.ExecutionValid || started.PayloadID == nil {
		return nil, fmt.Errorf("private EL refused replacement build: status %s, payload ID %v, validation error %v",
			started.PayloadStatus.Status, started.PayloadID, started.PayloadStatus.ValidationError)
	}
	env, err := b.engine.GetPayload(ctx, eth.PayloadInfo{ID: *started.PayloadID, Timestamp: uint64(attrs.Timestamp)})
	if err != nil {
		return nil, fmt.Errorf("seal deposits-only replacement on private EL: %w", err)
	}
	if env == nil || env.ExecutionPayload == nil {
		return nil, fmt.Errorf("private EL returned an empty replacement payload")
	}
	status, err := b.engine.NewPayload(ctx, env.ExecutionPayload, env.ParentBeaconBlockRoot)
	if err != nil {
		return nil, fmt.Errorf("import deposits-only replacement into private EL: %w", err)
	}
	if status.Status != eth.ExecutionValid {
		return nil, fmt.Errorf("private EL rejected deposits-only replacement: status %s, validation error %v",
			status.Status, status.ValidationError)
	}
	// NewPayload only imports the block. Normal op-node ingestion immediately follows it with FCU;
	// without that step eth_getBlockByNumber("latest") and the proof batcher remain on the denied
	// branch even though the replacement is present in the database.
	state.HeadBlockHash = env.ExecutionPayload.BlockHash
	promoted, err := b.engine.ForkchoiceUpdate(ctx, state, nil)
	if err != nil {
		return nil, fmt.Errorf("make deposits-only replacement canonical in private EL: %w", err)
	}
	if promoted.PayloadStatus.Status != eth.ExecutionValid {
		return nil, fmt.Errorf("private EL refused replacement forkchoice: status %s, validation error %v",
			promoted.PayloadStatus.Status, promoted.PayloadStatus.ValidationError)
	}
	return env, nil
}

type opaqueTransactions []eth.Data

func (txs opaqueTransactions) Len() int { return len(txs) }
func (txs opaqueTransactions) EncodeIndex(i int, w *bytes.Buffer) {
	w.Write(txs[i])
}

func headerFromPayload(env *eth.ExecutionPayloadEnvelope) (*types.Header, error) {
	if env == nil || env.ExecutionPayload == nil {
		return nil, fmt.Errorf("replacement payload is nil")
	}
	p := env.ExecutionPayload
	hasher := trie.NewStackTrie(nil)
	h := &types.Header{
		ParentHash:       p.ParentHash,
		UncleHash:        types.EmptyUncleHash,
		Coinbase:         p.FeeRecipient,
		Root:             common.Hash(p.StateRoot),
		TxHash:           types.DeriveSha(opaqueTransactions(p.Transactions), hasher),
		ReceiptHash:      common.Hash(p.ReceiptsRoot),
		Bloom:            types.Bloom(p.LogsBloom),
		Difficulty:       common.Big0,
		Number:           new(big.Int).SetUint64(uint64(p.BlockNumber)),
		GasLimit:         uint64(p.GasLimit),
		GasUsed:          uint64(p.GasUsed),
		Time:             uint64(p.Timestamp),
		Extra:            p.ExtraData,
		MixDigest:        common.Hash(p.PrevRandao),
		Nonce:            types.BlockNonce{},
		BaseFee:          (*uint256.Int)(&p.BaseFeePerGas).ToBig(),
		BlobGasUsed:      (*uint64)(p.BlobGasUsed),
		ExcessBlobGas:    (*uint64)(p.ExcessBlobGas),
		ParentBeaconRoot: env.ParentBeaconBlockRoot,
	}
	if p.WithdrawalsRoot != nil {
		h.WithdrawalsHash = p.WithdrawalsRoot
		h.RequestsHash = &types.EmptyRequestsHash
	} else if p.Withdrawals != nil {
		root := types.DeriveSha(*p.Withdrawals, hasher)
		h.WithdrawalsHash = &root
	}
	if h.Hash() != p.BlockHash {
		return nil, fmt.Errorf("private EL replacement hash mismatch: payload says %s, header hashes to %s", p.BlockHash, h.Hash())
	}
	return h, nil
}

func factFromReplacement(rollupCfg *rollup.Config, parent Fact, attrs *eth.PayloadAttributes, env *eth.ExecutionPayloadEnvelope) (Fact, Rendering, error) {
	p := env.ExecutionPayload
	if p.ParentHash != parent.Hash || uint64(p.BlockNumber) != parent.Number+1 || uint64(p.Timestamp) != uint64(attrs.Timestamp) {
		return Fact{}, Rendering{}, fmt.Errorf("private EL replacement does not extend denied block's parent")
	}
	if len(p.Transactions) != len(attrs.Transactions) {
		return Fact{}, Rendering{}, fmt.Errorf("private EL replacement contains %d transactions, attributes require %d", len(p.Transactions), len(attrs.Transactions))
	}
	for i := range p.Transactions {
		if !bytes.Equal(p.Transactions[i], attrs.Transactions[i]) {
			return Fact{}, Rendering{}, fmt.Errorf("private EL replacement transaction %d differs from deposits-only attributes", i)
		}
	}
	if p.WithdrawalsRoot == nil {
		return Fact{}, Rendering{}, fmt.Errorf("private EL replacement has no Isthmus message-passer root")
	}
	ref, err := derive.PayloadToBlockRef(rollupCfg, p)
	if err != nil {
		return Fact{}, Rendering{}, fmt.Errorf("read replacement L2 reference: %w", err)
	}
	hdr, err := headerFromPayload(env)
	if err != nil {
		return Fact{}, Rendering{}, err
	}
	stateRoot := common.Hash(p.StateRoot)
	messagePasserRoot := *p.WithdrawalsRoot
	outputRoot := common.Hash(eth.OutputRoot(&eth.OutputV0{
		StateRoot:                eth.Bytes32(stateRoot),
		MessagePasserStorageRoot: eth.Bytes32(messagePasserRoot),
		BlockHash:                p.BlockHash,
	}))
	txs := make([][]byte, len(p.Transactions))
	for i := range p.Transactions {
		txs[i] = append([]byte(nil), p.Transactions[i]...)
	}
	fact := Fact{
		Number:                   ref.Number,
		Timestamp:                ref.Time,
		Hash:                     ref.Hash,
		StateRoot:                stateRoot,
		MessagePasserStorageRoot: messagePasserRoot,
		OutputRoot:               outputRoot,
		L1Origin:                 ref.L1Origin,
		SeqNumber:                ref.SequenceNumber,
		Replacement:              true,
		ExecMsgsKnown:            true,
	}
	return fact, Rendering{Header: hdr, Txs: txs, Hash: fact.Hash}, nil
}
