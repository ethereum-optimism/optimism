package batcher

import (
	"bytes"
	"io"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

func TestProofChannelEncodesRealBlockWithoutTransactions(t *testing.T) {
	block := newMiniL2BlockWithNumberParent(2, big.NewInt(1), common.HexToHash("0x01"))
	payload := mustPayloadFromGeth(block)
	require.Len(t, payload.Transactions, 3, "fixture must contain L1 info plus two user transactions")

	cfg := ProofBatchConfig{
		Inbox:            common.HexToAddress("0xc0be1b01"),
		RollupConfigHash: common.HexToHash("0x11"),
		DepSetHash:       common.HexToHash("0x22"),
		WireVersion:      proofbatch.Version,
		MaxBlocks:        1,
	}
	encoder, err := NewProofBatchEncoder(cfg)
	require.NoError(t, err)
	parentRoot := common.HexToHash("0xaa")
	export := proofbatch.BlockExport{
		Number:                   1,
		Timestamp:                uint64(payload.Timestamp),
		Hash:                     payload.BlockHash,
		StateRoot:                common.HexToHash("0xbb"),
		MessagePasserStorageRoot: common.HexToHash("0xcc"),
	}
	encoder.blocks[payload.BlockHash] = preparedProofBlock{export: export, parentOutputRoot: parentRoot}

	channel, err := newProofChannelOut(encoder, defaultTestChannelConfig(), defaultTestRollupConfig)
	require.NoError(t, err)
	_, err = channel.AddBlock(defaultTestRollupConfig, payload)
	require.NoError(t, err)
	require.ErrorIs(t, channel.FullErr(), errProofBatchFull)
	require.NoError(t, channel.Close())

	var frames []frameData
	for {
		var out bytes.Buffer
		frameNumber, frameErr := channel.OutputFrame(&out, eth.MaxBlobDataSize)
		if out.Len() > 0 {
			frames = append(frames, frameData{data: out.Bytes(), raw: channel.RawFrames(), id: frameID{frameNumber: frameNumber}})
		}
		if frameErr == io.EOF {
			break
		}
		require.NoError(t, frameErr)
	}
	require.NotEmpty(t, frames)

	blobs, err := (&txData{frames: frames, asBlob: true}).Blobs()
	require.NoError(t, err)
	wire, _, err := proofbatch.FromBlobPrefix(blobs)
	require.NoError(t, err)
	envelope, err := proofbatch.Decode(wire)
	require.NoError(t, err)
	require.Empty(t, envelope.Proof)
	require.Equal(t, parentRoot, envelope.Batch.PrevOutputRoot)
	require.Equal(t, export.OutputRoot(), envelope.Batch.NewOutputRoot)
	require.Len(t, envelope.Batch.Blocks, 1)
	decoded := envelope.Batch.Blocks[0]
	require.Equal(t, export.Number, decoded.Number)
	require.Equal(t, export.Timestamp, decoded.Timestamp)
	require.Equal(t, export.Hash, decoded.Hash)
	require.Equal(t, export.StateRoot, decoded.StateRoot)
	require.Equal(t, export.MessagePasserStorageRoot, decoded.MessagePasserStorageRoot)
	require.Empty(t, decoded.Logs)
	require.Empty(t, decoded.ExecMsgs)

	for _, tx := range payload.Transactions[1:] {
		require.False(t, bytes.Contains(wire, tx), "proof envelope must not contain a user transaction")
	}
}

func TestProofBatchHooksObserveQueuedSubmission(t *testing.T) {
	hooks := NewProofBatchTestHooks()
	hooks.ProofBytesOnNext([]byte{0xaa})
	hooks.MutateUntilApplied(func(batch *proofbatch.ProofBatch) bool {
		batch.Blocks[0].Timestamp = 99
		return true
	})
	batch := &proofbatch.ProofBatch{Blocks: []proofbatch.BlockExport{{Number: 1}}}
	proof, mutated := hooks.beforeEncode(batch)
	require.True(t, mutated)
	require.Equal(t, []byte{0xaa}, proof)

	var id derive.ChannelID
	id[0] = 1
	hooks.prepared(id, batch, proof, mutated)
	require.Empty(t, hooks.Envelopes(), "encoding alone is not a submitted wire payload")

	hooks.RecordProofBatchSubmission([]derive.ChannelID{id})
	require.Len(t, hooks.Envelopes(), 1)
	require.Equal(t, uint64(99), hooks.Envelopes()[0].Batch.Blocks[0].Timestamp)
	require.Equal(t, []byte{0xaa}, hooks.Envelopes()[0].Proof)
}

func TestProofChannelPrependsImmediateParentOverlap(t *testing.T) {
	parentHash := common.HexToHash("0x1234")
	block := newMiniL2BlockWithNumberParent(0, big.NewInt(2), parentHash)
	payload := mustPayloadFromGeth(block)
	payload.Timestamp = 2
	parentParentRoot := common.HexToHash("0xaa")
	origin := eth.BlockID{Hash: common.HexToHash("0xcc"), Number: 100}

	encoder, err := NewProofBatchEncoder(ProofBatchConfig{
		Inbox:            common.HexToAddress("0xc0be1b01"),
		RollupConfigHash: common.HexToHash("0x11"),
		DepSetHash:       common.HexToHash("0x22"),
		WireVersion:      proofbatch.Version,
		MaxBlocks:        2,
	})
	require.NoError(t, err)
	parent := proofbatch.BlockExport{
		Number: 1, Timestamp: 1,
		Hash: parentHash, StateRoot: common.HexToHash("0x01"), MessagePasserStorageRoot: common.HexToHash("0x02"),
		L1Origin: origin, SequenceNumber: 0,
	}
	parentRoot := common.Hash(parent.OutputRoot())
	current := proofbatch.BlockExport{
		Number: 2, Timestamp: uint64(payload.Timestamp), Hash: payload.BlockHash,
		StateRoot: common.HexToHash("0x03"), MessagePasserStorageRoot: common.HexToHash("0x04"),
		L1Origin: origin, SequenceNumber: 1,
	}
	encoder.blocks[parentHash] = preparedProofBlock{export: parent, parentOutputRoot: parentParentRoot}
	encoder.blocks[payload.BlockHash] = preparedProofBlock{export: current, parentOutputRoot: parentRoot}

	channel, err := newProofChannelOut(encoder, defaultTestChannelConfig(), defaultTestRollupConfig)
	require.NoError(t, err)
	_, err = channel.AddBlock(defaultTestRollupConfig, payload)
	require.NoError(t, err)
	require.NoError(t, channel.Close())
	wire := append([]byte(nil), channel.payload...)
	env, err := proofbatch.Decode(wire)
	require.NoError(t, err)
	require.Equal(t, parentParentRoot, env.Batch.PrevOutputRoot)
	require.Len(t, env.Batch.Blocks, 2)
	require.Equal(t, parentHash, env.Batch.Blocks[0].Hash)
	require.Equal(t, payload.BlockHash, env.Batch.Blocks[1].Hash)
}
