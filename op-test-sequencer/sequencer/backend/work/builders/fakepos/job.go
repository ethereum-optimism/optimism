package fakepos

import (
	"context"
	"encoding/binary"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	opeth "github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

type FakePoSEnvelope struct {
	engine.ExecutionPayloadEnvelope
}

func (e *FakePoSEnvelope) ID() eth.BlockID {
	return eth.BlockID{Hash: e.ExecutionPayload.BlockHash, Number: e.ExecutionPayload.Number}
}

func (e *FakePoSEnvelope) String() string {
	return e.ID().String()
}

type Job struct {
	id     seqtypes.BuildJobID
	mu     sync.Mutex
	logger log.Logger

	b *Builder

	parent   common.Hash
	envelope engine.ExecutionPayloadEnvelope

	head      *types.Header
	safe      *types.Header
	finalized *types.Header

	parentBeaconBlockRoot common.Hash
}

func (j *Job) ID() seqtypes.BuildJobID {
	return j.id
}

func (j *Job) Cancel(ctx context.Context) error {
	return nil
}

func (j *Job) setHeadSafeAndFinalized() {
	j.head = j.b.geth.Backend.BlockChain().CurrentBlock() // default head
	if j.parent != (common.Hash{}) {
		j.head = j.b.geth.Backend.BlockChain().GetHeaderByHash(j.parent) // override head if parent is set
	}

	j.finalized = j.b.geth.Backend.BlockChain().CurrentFinalBlock()
	if j.finalized == nil { // fallback to genesis if nothing is finalized
		j.finalized = j.b.geth.Backend.BlockChain().Genesis().Header()
	}
	j.safe = j.b.geth.Backend.BlockChain().CurrentSafeBlock()
	if j.safe == nil { // fallback to finalized if nothing is safe
		j.safe = j.finalized
	}

	if j.head.Number.Uint64() > j.b.finalizedDistance { // progress finalized block, if we can
		j.finalized = j.b.geth.Backend.BlockChain().GetHeaderByNumber(j.head.Number.Uint64() - j.b.finalizedDistance)
	}
	if j.head.Number.Uint64() > j.b.safeDistance { // progress safe block, if we can
		j.safe = j.b.geth.Backend.BlockChain().GetHeaderByNumber(j.head.Number.Uint64() - j.b.safeDistance)
	}

	j.parentBeaconBlockRoot = fakeBeaconBlockRoot(j.head.Time) // parent beacon block root
}

func (j *Job) Open(ctx context.Context) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.logger.Info("Open job", "id", j.id)

	j.setHeadSafeAndFinalized()

	envelope, ok := j.b.envelopes[j.head.Hash()]
	if !ok { // we haven't build a block with this parent yet, so we need to build one
		newBlockTime := j.head.Time + j.b.blockTime

		attrs := &engine.PayloadAttributes{
			Timestamp:             newBlockTime,
			Random:                common.Hash{},
			SuggestedFeeRecipient: j.head.Coinbase,
			Withdrawals:           randomWithdrawals(j.b.withdrawalsIndex),
			BeaconRoot:            &j.parentBeaconBlockRoot,
		}
		fcState := engine.ForkchoiceStateV1{
			HeadBlockHash:      j.head.Hash(),
			SafeBlockHash:      j.safe.Hash(),
			FinalizedBlockHash: j.finalized.Hash(),
		}
		j.logger.Info("ForkchoiceUpdatedV3", "fcState", fcState)

		res, err := j.b.engine.ForkchoiceUpdatedV3(fcState, attrs)
		if err != nil {
			j.logger.Error("failed to start building L1 block", "err", err)
			return err
		}
		if res.PayloadID == nil {
			j.logger.Error("failed to start block building", "res", res)
			return errors.New("failed to start block building")
		}

		j.logger.Info("got res.payloadID", "res.payloadID", res.PayloadID)

		// wait for the block building to finish
		time.Sleep(100 * time.Millisecond)

		envelope, err = j.b.engine.GetPayloadV4(*res.PayloadID)
		if err != nil {
			j.logger.Error("failed to finish building L1 block", "err", err)
			return err
		}

		j.b.envelopes[envelope.ExecutionPayload.ParentHash] = envelope
	} else {
		j.logger.Warn("already had a block with that parent", "parent", j.head.Hash(), "number", j.head.Number.Uint64(), "fee_recipient", envelope.ExecutionPayload.FeeRecipient)

		j.logger.Warn("updating block hash", "pre", envelope.ExecutionPayload.BlockHash, "fee_recipient", envelope.ExecutionPayload.FeeRecipient, "txs", len(envelope.ExecutionPayload.Transactions))

		// modify gas limit so that we get a different block
		envelope.ExecutionPayload.GasLimit = envelope.ExecutionPayload.GasLimit + 100

		block, err := engine.ExecutableDataToBlockNoHash(*envelope.ExecutionPayload, make([]common.Hash, 0), &j.parentBeaconBlockRoot, make([][]byte, 0), j.b.geth.Backend.BlockChain().Config())
		if err != nil {
			j.logger.Error("failed to convert executable data to block", "err", err)
			return err
		}
		envelope.ExecutionPayload.BlockHash = block.Hash()

		j.logger.Warn("updated block hash", "post", envelope.ExecutionPayload.BlockHash, "fee_recipient", envelope.ExecutionPayload.FeeRecipient, "txs", len(envelope.ExecutionPayload.Transactions))
	}

	j.logger.Info("final envelope", "envelope", envelope)

	return nil
}

func (j *Job) Seal(ctx context.Context) (work.Block, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	envelope, ok := j.b.envelopes[j.head.Hash()]
	if !ok { // we haven't build a block with this parent yet, so we need to build one
		return nil, errors.New("no envelope found")
	}

	blobHashes := make([]common.Hash, 0) // must be non-nil even when empty, due to geth engine API checks
	if envelope.BlobsBundle != nil {
		for _, commitment := range envelope.BlobsBundle.Commitments {
			if len(commitment) != 48 {
				j.logger.Error("got malformed kzg commitment from engine", "commitment", commitment)
				break
			}
			blobHashes = append(blobHashes, opeth.KZGToVersionedHash(*(*[48]byte)(commitment)))
		}
		if len(blobHashes) != len(envelope.BlobsBundle.Commitments) {
			j.logger.Error("invalid or incomplete blob data", "collected", len(blobHashes), "engine", len(envelope.BlobsBundle.Commitments))
			return nil, errors.New("invalid or incomplete blob data")
		}
	}

	j.logger.Info("about to insert payload into the chain", "envelope-hash", envelope.ExecutionPayload.BlockHash, "txs", len(envelope.ExecutionPayload.Transactions))

	_, err := j.b.engine.NewPayloadV4(*envelope.ExecutionPayload, blobHashes, &j.parentBeaconBlockRoot, make([]hexutil.Bytes, 0))
	if err != nil {
		j.logger.Error("failed to insert built L1 block", "err", err)
		return nil, err
	}

	if envelope.BlobsBundle != nil {
		slot := (envelope.ExecutionPayload.Timestamp - j.b.geth.Backend.BlockChain().Genesis().Time()) / j.b.blockTime
		if j.b.beacon == nil {
			j.logger.Error("no blobs storage available")
			return nil, errors.New("no blobs storage available")
		}
		if err := j.b.beacon.StoreBlobsBundle(slot, envelope.BlobsBundle); err != nil {
			j.logger.Error("failed to persist blobs-bundle of block, not making block canonical now", "err", err)
			return nil, err
		}
	}

	j.logger.Info("about to forkchoice update", "safe", j.safe.Hash(), "finalized", j.finalized.Hash(), "head", envelope.ExecutionPayload.BlockHash)

	if _, err := j.b.engine.ForkchoiceUpdatedV3(engine.ForkchoiceStateV1{
		HeadBlockHash:      envelope.ExecutionPayload.BlockHash,
		SafeBlockHash:      j.safe.Hash(),
		FinalizedBlockHash: j.finalized.Hash(),
	}, nil); err != nil {
		j.logger.Error("failed to make built L1 block canonical", "err", err)
		return nil, err
	}

	j.b.withdrawalsIndex += uint64(len(envelope.ExecutionPayload.Withdrawals))

	j.logger.Info("incrementing withdrawals index", "index", j.b.withdrawalsIndex)

	j.envelope = *envelope

	return &FakePoSEnvelope{ExecutionPayloadEnvelope: j.envelope}, nil
}

func (job *Job) String() string {
	return job.id.String()
}

func (job *Job) Close() {
}

func (job *Job) IncludeTx(ctx context.Context, tx hexutil.Bytes) error {
	return errors.New("not implemented")
}

var _ work.BuildJob = (*Job)(nil)

func fakeBeaconBlockRoot(time uint64) common.Hash {
	var dat [8]byte
	binary.LittleEndian.PutUint64(dat[:], time)
	return crypto.Keccak256Hash(dat[:])
}

func randomWithdrawals(startIndex uint64) []*types.Withdrawal {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	withdrawals := make([]*types.Withdrawal, r.Intn(4))
	for i := 0; i < len(withdrawals); i++ {
		withdrawals[i] = &types.Withdrawal{
			Index:     startIndex + uint64(i),
			Validator: r.Uint64() % 100_000_000, // 100 million fake validators
			Address:   testutils.RandomAddress(r),
			Amount:    uint64(r.Intn(50_000_000_000) + 1),
		}
	}
	return withdrawals
}
