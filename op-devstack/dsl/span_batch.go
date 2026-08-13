package dsl

import (
	"bytes"
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	deriveparams "github.com/ethereum-optimism/optimism/op-node/rollup/derive/params"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

const spanBatchMaxL1TxSize = 120_000

type spanBatchConfig struct {
	transactions map[uint64][]*types.Transaction
}

// SpanBatchOption configures a span batch built by [L2Network.NewSpanBatch].
type SpanBatchOption func(*spanBatchConfig)

// WithSpanBatchTransaction appends tx to the selected source block.
func WithSpanBatchTransaction(blockNumber uint64, tx *types.Transaction) SpanBatchOption {
	return func(cfg *spanBatchConfig) {
		if cfg.transactions == nil {
			cfg.transactions = make(map[uint64][]*types.Transaction)
		}
		cfg.transactions[blockNumber] = append(cfg.transactions[blockNumber], tx)
	}
}

// SpanBatch is a prepared span batch that can be submitted after its source L2 blocks are reorged.
type SpanBatch struct {
	commonImpl
	batcher *EOA
	inbox   common.Address
	frames  [][]byte
	start   uint64
	end     uint64
}

// NewSpanBatch prepares a span batch from the canonical L2 blocks in the inclusive [start, end]
// range. Preparing and submitting are separate so a test can retain one version while another
// version reorgs its source blocks out of the canonical chain.
func (n *L2Network) NewSpanBatch(start, end uint64, opts ...SpanBatchOption) *SpanBatch {
	cfg := spanBatchConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	n.require.GreaterOrEqual(start, uint64(1), "span batch cannot include the L2 genesis block")
	n.require.GreaterOrEqual(end, start, "span batch end must not precede start")
	for blockNumber, txs := range cfg.transactions {
		n.require.GreaterOrEqual(blockNumber, start, "transaction block must be in the span batch")
		n.require.LessOrEqual(blockNumber, end, "transaction block must be in the span batch")
		for _, tx := range txs {
			n.require.NotNil(tx, "span batch transaction must not be nil")
		}
	}

	rollupCfg := n.inner.RollupConfig()
	channel, err := derive.NewSpanChannelOut(
		spanBatchMaxL1TxSize,
		derive.Zlib,
		rollup.NewChainSpec(rollupCfg),
	)
	n.require.NoError(err, "must create span batch channel")

	for blockNumber := start; blockNumber <= end; blockNumber++ {
		payload := n.primaryEL.PayloadByNumber(blockNumber)
		for _, tx := range cfg.transactions[blockNumber] {
			encoded, err := tx.MarshalBinary()
			n.require.NoError(err, "must encode span batch transaction")
			payload.ExecutionPayload.Transactions = append(payload.ExecutionPayload.Transactions, encoded)
		}
		_, err = channel.AddBlock(rollupCfg, payload.ExecutionPayload)
		n.require.NoError(err, "must add L2 block %d to span batch", blockNumber)
	}
	n.require.NoError(channel.Close(), "must close span batch channel")

	var frames [][]byte
	for {
		frame := new(bytes.Buffer)
		frame.WriteByte(deriveparams.DerivationVersion0)
		_, err := channel.OutputFrame(frame, spanBatchMaxL1TxSize-1)
		frames = append(frames, frame.Bytes())
		if errors.Is(err, io.EOF) {
			break
		}
		n.require.NoError(err, "must output span batch frame")
	}

	batcherKey := n.inner.Keys().Secret(devkeys.BatcherRole.Key(n.ChainID().ToBig()))
	inbox := rollupCfg.BatchInboxAddress
	return &SpanBatch{
		commonImpl: commonFromT(n.t),
		batcher:    NewKey(n.t, batcherKey).User(n.primaryL1),
		inbox:      inbox,
		frames:     frames,
		start:      start,
		end:        end,
	}
}

// Submit posts every prepared frame to L1 from the configured batcher account and waits for each
// transaction to be included.
func (b *SpanBatch) Submit() {
	b.log.Info("Submitting prepared span batch", "start", b.start, "end", b.end, "frames", len(b.frames))
	for i, frame := range b.frames {
		b.log.Info("Submitting span batch frame", "frame", i, "size", len(frame))
		b.batcher.Transact(
			b.batcher.Plan(),
			txplan.WithTo(&b.inbox),
			txplan.WithData(frame),
		)
	}
}
