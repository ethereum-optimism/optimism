package geth

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	opeth "github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

type Beacon interface {
	StoreBlobsBundle(slot uint64, bundle *engine.BlobsBundle) error
}

// fakePoS is a testing-only utility to attach to Geth,
// to build a fake proof-of-stake L1 chain with fixed block time and basic lagging safe/finalized blocks.
type FakePoS struct {
	clock     clock.Clock
	eth       Backend
	log       log.Logger
	blockTime uint64

	withdrawalsIndex uint64

	finalizedDistance uint64
	safeDistance      uint64

	engineAPI EngineAPI
	sub       ethereum.Subscription

	beacon Beacon

	config *params.ChainConfig

	pauseMu      sync.Mutex
	pauseAt      uint64
	pauseReached chan struct{}
	paused       bool
	resumeCh     chan struct{}
}

type Backend interface {
	// HeaderByNumber is assumed to behave the same as go-ethereum/ethclient.Client.HeaderByNumber.
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
}

type EngineAPI interface {
	ForkchoiceUpdatedV4(context.Context, engine.ForkchoiceStateV1, *engine.PayloadAttributes, *types.CustodyBitmap) (engine.ForkChoiceResponse, error)
	ForkchoiceUpdatedV3(context.Context, engine.ForkchoiceStateV1, *engine.PayloadAttributes) (engine.ForkChoiceResponse, error)
	ForkchoiceUpdatedV2(context.Context, engine.ForkchoiceStateV1, *engine.PayloadAttributes) (engine.ForkChoiceResponse, error)

	GetPayloadV6(engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error)
	GetPayloadV5(engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error)
	GetPayloadV4(engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error)
	GetPayloadV3(engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error)
	GetPayloadV2(engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error)

	NewPayloadV5(context.Context, engine.ExecutableData, []common.Hash, *common.Hash, []hexutil.Bytes) (engine.PayloadStatusV1, error)
	NewPayloadV4(context.Context, engine.ExecutableData, []common.Hash, *common.Hash, []hexutil.Bytes) (engine.PayloadStatusV1, error)
	NewPayloadV3(context.Context, engine.ExecutableData, []common.Hash, *common.Hash) (engine.PayloadStatusV1, error)
	NewPayloadV2(context.Context, engine.ExecutableData) (engine.PayloadStatusV1, error)
}

func NewFakePoS(backend Backend, engineAPI EngineAPI, c clock.Clock, logger log.Logger, blockTime uint64, finalizedDistance uint64, beacon Beacon, config *params.ChainConfig) *FakePoS {
	return &FakePoS{
		clock:             c,
		eth:               backend,
		log:               logger,
		blockTime:         blockTime,
		finalizedDistance: finalizedDistance,
		safeDistance:      10,
		engineAPI:         engineAPI,
		beacon:            beacon,
		config:            config,
		resumeCh:          make(chan struct{}, 1),
	}
}

// PauseAtBlock configures the block producer to pause immediately after block
// number is made canonical. The returned channel closes when the pause has
// taken effect.
func (f *FakePoS) PauseAtBlock(number uint64) (<-chan struct{}, error) {
	f.pauseMu.Lock()
	defer f.pauseMu.Unlock()
	if f.pauseAt != 0 {
		return nil, fmt.Errorf("L1 block producer already configured to pause at block %d", f.pauseAt)
	}

	head, err := f.eth.HeaderByNumber(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("get L1 head before configuring pause: %w", err)
	}
	headNumber := bigs.Uint64Strict(head.Number)
	if headNumber >= number {
		return nil, fmt.Errorf("cannot pause at L1 block %d: current head is %d", number, headNumber)
	}

	f.pauseAt = number
	f.pauseReached = make(chan struct{})
	return f.pauseReached, nil
}

// Resume resumes block production after a configured pause has taken effect.
func (f *FakePoS) Resume() error {
	f.pauseMu.Lock()
	if !f.paused {
		f.pauseMu.Unlock()
		return errors.New("L1 block producer is not paused")
	}
	f.paused = false
	f.pauseMu.Unlock()
	f.resumeCh <- struct{}{}
	return nil
}

// pauseIfConfigured returns true if the subscription context ended while the
// block producer was paused.
func (f *FakePoS) pauseIfConfigured(ctx context.Context, number uint64) bool {
	f.pauseMu.Lock()
	if f.pauseAt != number {
		f.pauseMu.Unlock()
		return false
	}
	reached := f.pauseReached
	f.pauseAt = 0
	f.pauseReached = nil
	f.paused = true
	f.pauseMu.Unlock()
	close(reached)

	select {
	case <-f.resumeCh:
		return false
	case <-ctx.Done():
		return true
	}
}

func (f *FakePoS) FakeBeaconBlockRoot(time uint64) common.Hash {
	var dat [8]byte
	binary.LittleEndian.PutUint64(dat[:], time)
	return crypto.Keccak256Hash(dat[:])
}

func (f *FakePoS) Start() error {
	if advancing, ok := f.clock.(*clock.AdvancingClock); ok {
		advancing.Start()
	}
	withdrawalsRNG := rand.New(rand.NewSource(450368975843)) // avoid generating the same address as any test
	genesisHeader, err := f.eth.HeaderByNumber(context.Background(), new(big.Int))
	if err != nil {
		return fmt.Errorf("get genesis header: %w", err)
	}
	f.sub = event.NewSubscription(func(quit <-chan struct{}) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			<-quit
			cancel()
		}()

		// poll every half a second: enough to catch up with any block time when ticks are missed
		t := f.clock.NewTicker(time.Second / 2)
		for {
			select {
			case now := <-t.Ch():
				head, err := f.eth.HeaderByNumber(context.Background(), nil)
				if err != nil {
					f.log.Warn("Failed to obtain latest header", "err", err)
					continue
				}
				finalized, err := f.eth.HeaderByNumber(context.Background(), big.NewInt(int64(rpc.FinalizedBlockNumber)))
				if err != nil {
					finalized = genesisHeader // fallback to genesis if nothing is finalized
				}
				safe, err := f.eth.HeaderByNumber(context.Background(), big.NewInt(int64(rpc.SafeBlockNumber)))
				if err != nil { // fallback to finalized if nothing is safe
					safe = finalized
				}
				if bigs.Uint64Strict(head.Number) > f.finalizedDistance { // progress finalized block, if we can
					finalized, err = f.eth.HeaderByNumber(context.Background(), new(big.Int).SetUint64(bigs.Uint64Strict(head.Number)-f.finalizedDistance))
					if err != nil {
						f.log.Warn("Failed to finalized header", "err", err)
						continue
					}
				}
				if bigs.Uint64Strict(head.Number) > f.safeDistance { // progress safe block, if we can
					safe, err = f.eth.HeaderByNumber(context.Background(), new(big.Int).SetUint64(bigs.Uint64Strict(head.Number)-f.safeDistance))
					if err != nil {
						f.log.Warn("Failed to safe header", "err", err)
						continue
					}
				}
				// start building the block as soon as we are past the current head time
				if head.Time >= uint64(now.Unix()) {
					continue
				}
				newBlockTime := head.Time + f.blockTime
				if time.Unix(int64(newBlockTime), 0).Add(5 * time.Minute).Before(f.clock.Now()) {
					// We're a long way behind, let's skip some blocks...
					newBlockTime = uint64(f.clock.Now().Unix())
				}
				// create some random withdrawals
				withdrawals := make([]*types.Withdrawal, withdrawalsRNG.Intn(4))
				for i := 0; i < len(withdrawals); i++ {
					withdrawals[i] = &types.Withdrawal{
						Index:     f.withdrawalsIndex + uint64(i),
						Validator: withdrawalsRNG.Uint64() % 100_000_000, // 100 million fake validators
						Address:   testutils.RandomAddress(withdrawalsRNG),
						// in gwei, consensus-layer quirk. withdraw non-zero value up to 50 ETH
						Amount: uint64(withdrawalsRNG.Intn(50_000_000_000) + 1),
					}
				}
				attrs := &engine.PayloadAttributes{
					Timestamp:             newBlockTime,
					Random:                common.Hash{},
					SuggestedFeeRecipient: head.Coinbase,
					Withdrawals:           withdrawals,
				}
				parentBeaconBlockRoot := f.FakeBeaconBlockRoot(head.Time) // parent beacon block root
				nextHeight := new(big.Int).SetUint64(bigs.Uint64Strict(head.Number) + 1)
				isCancun := f.config.IsCancun(nextHeight, newBlockTime)
				isPrague := f.config.IsPrague(nextHeight, newBlockTime)
				isOsaka := f.config.IsOsaka(nextHeight, newBlockTime)
				isAmsterdam := f.config.IsAmsterdam(nextHeight, newBlockTime)
				if isCancun {
					attrs.BeaconRoot = &parentBeaconBlockRoot
				}
				if isAmsterdam {
					slotNumber := (newBlockTime - genesisHeader.Time) / f.blockTime
					attrs.SlotNumber = &slotNumber
					targetGasLimit := head.GasLimit
					attrs.TargetGasLimit = &targetGasLimit
				}
				fcState := engine.ForkchoiceStateV1{
					HeadBlockHash:      head.Hash(),
					SafeBlockHash:      safe.Hash(),
					FinalizedBlockHash: finalized.Hash(),
				}
				var res engine.ForkChoiceResponse
				if isAmsterdam {
					// FakePoS does not model custody-column availability. EIP-8070 makes the
					// custody bitmap optional, so custody behavior needs separate CL/DA coverage.
					res, err = f.engineAPI.ForkchoiceUpdatedV4(ctx, fcState, attrs, nil)
				} else if isCancun {
					res, err = f.engineAPI.ForkchoiceUpdatedV3(ctx, fcState, attrs)
				} else {
					res, err = f.engineAPI.ForkchoiceUpdatedV2(ctx, fcState, attrs)
				}
				if err != nil {
					f.log.Error("failed to start building L1 block", "err", err)
					continue
				}
				if err := ValidatePayloadStatus("start-building forkchoice update", res.PayloadStatus); err != nil {
					f.log.Error("failed to start building L1 block", "err", err)
					continue
				}
				if res.PayloadID == nil {
					f.log.Error("failed to start block building", "res", res)
					continue
				}
				// wait with sealing, if we are not behind already
				delay := time.Unix(int64(newBlockTime), 0).Sub(f.clock.Now())
				tim := f.clock.NewTimer(delay)
				select {
				case <-tim.Ch():
					// no-op
				case <-ctx.Done():
					tim.Stop()
					return nil
				}
				var envelope *engine.ExecutionPayloadEnvelope
				if isAmsterdam {
					envelope, err = f.engineAPI.GetPayloadV6(*res.PayloadID)
				} else if isOsaka {
					envelope, err = f.engineAPI.GetPayloadV5(*res.PayloadID)
				} else if isPrague {
					envelope, err = f.engineAPI.GetPayloadV4(*res.PayloadID)
				} else if isCancun {
					envelope, err = f.engineAPI.GetPayloadV3(*res.PayloadID)
				} else {
					envelope, err = f.engineAPI.GetPayloadV2(*res.PayloadID)
				}
				if err != nil {
					f.log.Error("failed to finish building L1 block", "err", err)
					continue
				}
				if isAmsterdam {
					EnsureAmsterdamBlockAccessList(envelope.ExecutionPayload)
				}

				blobHashes := make([]common.Hash, 0) // must be non-nil even when empty, due to geth engine API checks
				if envelope.BlobsBundle != nil {
					for _, commitment := range envelope.BlobsBundle.Commitments {
						if len(commitment) != 48 {
							f.log.Error("got malformed kzg commitment from engine", "commitment", commitment)
							break
						}
						blobHashes = append(blobHashes, opeth.KZGToVersionedHash(*(*[48]byte)(commitment)))
					}
					if len(blobHashes) != len(envelope.BlobsBundle.Commitments) {
						f.log.Error("invalid or incomplete blob data", "collected", len(blobHashes), "engine", len(envelope.BlobsBundle.Commitments))
						continue
					}
				}

				var payloadStatus engine.PayloadStatusV1
				if isAmsterdam {
					payloadStatus, err = f.engineAPI.NewPayloadV5(context.Background(), *envelope.ExecutionPayload, blobHashes, &parentBeaconBlockRoot, make([]hexutil.Bytes, 0))
				} else if isPrague {
					payloadStatus, err = f.engineAPI.NewPayloadV4(context.Background(), *envelope.ExecutionPayload, blobHashes, &parentBeaconBlockRoot, make([]hexutil.Bytes, 0))
				} else if isCancun {
					payloadStatus, err = f.engineAPI.NewPayloadV3(context.Background(), *envelope.ExecutionPayload, blobHashes, &parentBeaconBlockRoot)
				} else {
					payloadStatus, err = f.engineAPI.NewPayloadV2(context.Background(), *envelope.ExecutionPayload)
				}
				if err != nil {
					f.log.Error("failed to insert built L1 block", "err", err)
					continue
				}
				if err := ValidatePayloadStatus("new payload", payloadStatus); err != nil {
					f.log.Error("failed to insert built L1 block", "err", err)
					continue
				}

				if envelope.BlobsBundle != nil {
					slot := (envelope.ExecutionPayload.Timestamp - genesisHeader.Time) / f.blockTime
					if f.beacon == nil {
						f.log.Error("no blobs storage available")
						continue
					}
					if err := f.beacon.StoreBlobsBundle(slot, envelope.BlobsBundle); err != nil {
						f.log.Error("failed to persist blobs-bundle of block, not making block canonical now", "err", err)
						continue
					}
				}
				fcState = engine.ForkchoiceStateV1{
					HeadBlockHash:      envelope.ExecutionPayload.BlockHash,
					SafeBlockHash:      safe.Hash(),
					FinalizedBlockHash: finalized.Hash(),
				}
				var fcRes engine.ForkChoiceResponse
				if isAmsterdam {
					fcRes, err = f.engineAPI.ForkchoiceUpdatedV4(ctx, fcState, nil, nil)
				} else {
					fcRes, err = f.engineAPI.ForkchoiceUpdatedV3(ctx, fcState, nil)
				}
				if err != nil {
					f.log.Error("failed to make built L1 block canonical", "err", err)
					continue
				}
				if err := ValidatePayloadStatus("canonicalizing forkchoice update", fcRes.PayloadStatus); err != nil {
					f.log.Error("failed to make built L1 block canonical", "err", err)
					continue
				}
				// Increment global withdrawals index in the CL.
				// The EL doesn't really care about the value,
				// but it's nice to mock something consistent with the CL specs.
				f.withdrawalsIndex += uint64(len(withdrawals))
				if f.pauseIfConfigured(ctx, bigs.Uint64Strict(nextHeight)) {
					return nil
				}
			case <-ctx.Done():
				return nil
			}
		}
	})
	return nil
}

func (f *FakePoS) Stop() error {
	if f.sub == nil || f.clock == nil {
		return errors.New("fakePoS not started, but stop was called")
	}
	f.sub.Unsubscribe()
	if advancing, ok := f.clock.(*clock.AdvancingClock); ok {
		advancing.Stop()
	}
	return nil
}
