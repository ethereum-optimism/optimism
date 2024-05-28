package plasma

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-plasma/bindings"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ErrPendingChallenge is returned when data is not available but can still be challenged/resolved
// so derivation should halt temporarily.
var ErrPendingChallenge = errors.New("not found, pending challenge")

// ErrExpiredChallenge is returned when a challenge was not resolved and derivation should skip this input.
var ErrExpiredChallenge = errors.New("challenge expired")

// ErrMissingPastWindow is returned when the input data is MIA and cannot be challenged.
// This is a protocol fatal error.
var ErrMissingPastWindow = errors.New("data missing past window")

// ErrInvalidChallenge is returned when a challenge event does is decoded but does not
// relate to the actual chain commitments.
var ErrInvalidChallenge = errors.New("invalid challenge")

// L1Fetcher is the required interface for syncing the DA challenge contract state.
type L1Fetcher interface {
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
	L1BlockRefByNumber(context.Context, uint64) (eth.L1BlockRef, error)
}

// DAStorage interface for calling the DA storage server.
type DAStorage interface {
	GetInput(ctx context.Context, key CommitmentData) ([]byte, error)
	SetInput(ctx context.Context, img []byte) (CommitmentData, error)
}

// HeadSignalFn is the callback function to accept head-signals without a context.
type HeadSignalFn func(eth.L1BlockRef)

// Config is the relevant subset of rollup config for plasma DA.
type Config struct {
	// Required for filtering contract events
	DAChallengeContractAddress common.Address
	// Allowed CommitmentType
	CommitmentType CommitmentType
	// The number of l1 blocks after the input is committed during which one can challenge.
	ChallengeWindow uint64
	// The number of l1 blocks after a commitment is challenged during which one can resolve.
	ResolveWindow uint64
}

type DA struct {
	log     log.Logger
	cfg     Config
	metrics Metricer

	storage DAStorage

	// the DA state keeps track of all the commitments and their challenge status.
	state *State

	// the latest l1 block we synced challenge contract events from
	origin eth.BlockID
	// the latest recorded finalized head as per the challenge contract
	finalizedHead eth.L1BlockRef
	// the latest recorded finalized head as per the l1 finalization signal
	l1FinalizedHead eth.L1BlockRef
	// flag the reset function we are resetting because of an expired challenge
	resetting bool

	finalizedHeadSignalFunc HeadSignalFn
}

// NewPlasmaDA creates a new PlasmaDA instance with the given log and CLIConfig.
func NewPlasmaDA(log log.Logger, cli CLIConfig, cfg Config, metrics Metricer) *DA {
	return NewPlasmaDAWithStorage(log, cfg, cli.NewDAClient(), metrics)
}

// NewPlasmaDAWithStorage creates a new PlasmaDA instance with the given log and DAStorage interface.
func NewPlasmaDAWithStorage(log log.Logger, cfg Config, storage DAStorage, metrics Metricer) *DA {
	return &DA{
		log:     log,
		cfg:     cfg,
		storage: storage,
		metrics: metrics,
		state:   NewState(log, metrics),
	}
}

// NewPlasmaDAWithState creates a plasma storage from initial state used for testing in isolation.
// We pass the L1Fetcher to each method so it is kept in sync with the conf depth of the pipeline.
func NewPlasmaDAWithState(log log.Logger, cfg Config, storage DAStorage, metrics Metricer, state *State) *DA {
	return &DA{
		log:     log,
		cfg:     cfg,
		storage: storage,
		metrics: metrics,
		state:   state,
	}
}

// OnFinalizedHeadSignal sets the callback function to be called when the finalized head is updated.
// This will signal to the engine queue that will set the proper L2 block as finalized.
func (d *DA) OnFinalizedHeadSignal(f HeadSignalFn) {
	d.finalizedHeadSignalFunc = f
}

// Finalize takes the L1 finality signal, compares the plasma finalized block and forwards the finality
// signal to the engine queue based on whichever is most behind.
func (d *DA) Finalize(l1Finalized eth.L1BlockRef) {
	ref := d.finalizedHead
	d.log.Info("received l1 finalized signal, forwarding to engine queue", "l1", l1Finalized, "plasma", ref)
	// if the l1 finalized head is behind it is the finalized head
	if l1Finalized.Number < d.finalizedHead.Number {
		ref = l1Finalized
	}
	// prune finalized state
	d.state.Prune(ref.Number)

	if d.finalizedHeadSignalFunc == nil {
		d.log.Warn("finalized head signal function not set")
		return
	}

	// signal the engine queue
	d.finalizedHeadSignalFunc(ref)
}

// LookAhead increments the challenges origin and process the new block if it exists.
// It is used when the derivation pipeline stalls due to missing data and we need to continue
// syncing challenge events until the challenge is resolved or expires.
func (d *DA) LookAhead(ctx context.Context, l1 L1Fetcher) error {
	blkRef, err := l1.L1BlockRefByNumber(ctx, d.origin.Number+1)
	// temporary error, will do a backoff
	if err != nil {
		return err
	}
	return d.AdvanceL1Origin(ctx, l1, blkRef.ID())
}

// Reset the challenge event derivation origin in case of L1 reorg
func (d *DA) Reset(ctx context.Context, base eth.L1BlockRef, baseCfg eth.SystemConfig) error {
	// resetting due to expired challenge, do not clear state.
	// If the DA source returns ErrReset, the pipeline is forced to reset by the rollup driver.
	// In that case the Reset function will be called immediately, BEFORE the pipeline can
	// call any further stage to step. Thus the state will NOT be cleared if the reset originates
	// from this stage of the pipeline.
	if d.resetting {
		d.resetting = false
	} else {
		// resetting due to L1 reorg, clear state
		d.origin = base.ID()
		d.state.Reset()
	}
	return io.EOF
}

// GetInput returns the input data for the given commitment bytes. blockNumber is required to lookup
// the challenge status in the DataAvailabilityChallenge L1 contract.
func (d *DA) GetInput(ctx context.Context, l1 L1Fetcher, comm CommitmentData, blockId eth.BlockID) (eth.Data, error) {
	// If it's not the right commitment type, report it as an expired commitment in order to skip it
	if d.cfg.CommitmentType != comm.CommitmentType() {
		return nil, fmt.Errorf("invalid commitment type; expected: %v, got: %v: %w", d.cfg.CommitmentType, comm.CommitmentType(), ErrExpiredChallenge)
	}
	// If the challenge head is ahead in the case of a pipeline reset or stall, we might have synced a
	// challenge event for this commitment. Otherwise we mark the commitment as part of the canonical
	// chain so potential future challenge events can be selected.
	ch := d.state.GetOrTrackChallenge(comm.Encode(), blockId.Number, d.cfg.ChallengeWindow)

	// Fetch the input from the DA storage.
	data, err := d.storage.GetInput(ctx, comm)

	// data is not found in storage but may be available if the challenge was resolved.
	notFound := errors.Is(ErrNotFound, err)

	if err != nil && !notFound {
		d.log.Error("failed to get preimage", "err", err)
		// the storage client request failed for some other reason
		// in which case derivation pipeline should be retried
		return nil, err
	}

	switch ch.challengeStatus {
	case ChallengeActive:
		if d.isExpired(ch.expiresAt) {
			// this challenge has expired, this input must be skipped
			return nil, ErrExpiredChallenge
		} else if notFound {
			// data is missing and a challenge is active, we must wait for the challenge to resolve
			// hence we continue syncing new origins to sync the new challenge events.
			if err := d.LookAhead(ctx, l1); err != nil {
				return nil, err
			}
			return nil, ErrPendingChallenge
		}
	case ChallengeExpired:
		// challenge was marked as expired, skip
		return nil, ErrExpiredChallenge
	case ChallengeResolved:
		// challenge was resolved, data is available in storage, return directly
		if !notFound {
			return data, nil
		}
		// Generic Commitments don't resolve from L1 so if we still can't find the data with out of luck
		if comm.CommitmentType() == GenericCommitmentType {
			return nil, ErrMissingPastWindow
		}
		// data not found in storage, return from challenge resolved input
		resolvedInput, err := d.state.GetResolvedInput(comm.Encode())
		if err != nil {
			return nil, err
		}
		return resolvedInput, nil
	default:
		if notFound {
			if d.isExpired(ch.expiresAt) {
				// we're past the challenge window and the data is not available
				return nil, ErrMissingPastWindow
			} else {
				// continue syncing challenges hoping it eventually is challenged and resolved
				if err := d.LookAhead(ctx, l1); err != nil {
					return nil, err
				}
				return nil, ErrPendingChallenge
			}
		}
	}

	return data, nil
}

// isExpired returns whether the given expiration block number is lower or equal to the current head
func (d *DA) isExpired(bn uint64) bool {
	return d.origin.Number >= bn
}

// AdvanceL1Origin syncs any challenge events included in the l1 block, expires any active challenges
// after the new resolveWindow, computes and signals the new finalized head and sets the l1 block
// as the new head for tracking challenges. If forwards an error if any new challenge have expired to
// trigger a derivation reset.
func (d *DA) AdvanceL1Origin(ctx context.Context, l1 L1Fetcher, block eth.BlockID) error {
	// do not repeat for the same origin
	if block.Number <= d.origin.Number {
		return nil
	}
	// sync challenges for the given block ID
	if err := d.LoadChallengeEvents(ctx, l1, block); err != nil {
		return err
	}
	// advance challenge window, computing the finalized head
	bn, err := d.state.ExpireChallenges(block.Number)
	if err != nil {
		// warn the reset function not to clear the state
		d.resetting = true
		return err
	}

	// finalized head signal is called only when the finalized head number increases
	// and the l1 finalized head ahead of the DA finalized head.
	if bn > d.finalizedHead.Number {
		ref, err := l1.L1BlockRefByNumber(ctx, bn)
		if err != nil {
			return err
		}
		d.metrics.RecordChallengesHead("finalized", bn)

		// keep track of finalized had so it can be picked up by the
		// l1 finalization signal
		d.finalizedHead = ref
	}
	d.origin = block
	d.metrics.RecordChallengesHead("latest", d.origin.Number)

	d.log.Info("processed plasma l1 origin", "origin", block, "next-finalized", bn, "finalized", d.finalizedHead.Number, "l1-finalize", d.l1FinalizedHead.Number)
	return nil
}

// LoadChallengeEvents fetches the l1 block receipts and updates the challenge status
func (d *DA) LoadChallengeEvents(ctx context.Context, l1 L1Fetcher, block eth.BlockID) error {
	// filter any challenge event logs in the block
	logs, err := d.fetchChallengeLogs(ctx, l1, block)
	if err != nil {
		return err
	}

	for _, log := range logs {
		i := log.TxIndex
		status, comm, err := d.decodeChallengeStatus(log)
		if err != nil {
			d.log.Error("failed to decode challenge event", "block", block.Number, "tx", i, "log", log.Index, "err", err)
			continue
		}
		switch status {
		case ChallengeResolved:
			// cached with input resolution call so not expensive
			_, txs, err := l1.InfoAndTxsByHash(ctx, block.Hash)
			if err != nil {
				d.log.Error("failed to fetch l1 block", "block", block.Number, "err", err)
				continue
			}
			// avoid panic in black swan case of faulty rpc
			if uint(len(txs)) <= i {
				d.log.Error("tx/receipt mismatch in InfoAndTxsByHash")
				continue
			}
			// select the transaction corresponding to the receipt
			tx := txs[i]
			// txs and receipts must be in the same order
			if tx.Hash() != log.TxHash {
				d.log.Error("tx hash mismatch", "block", block.Number, "txIdx", i, "log", log.Index, "txHash", tx.Hash(), "receiptTxHash", log.TxHash)
				continue
			}
			var input []byte
			if d.cfg.CommitmentType == Keccak256CommitmentType {
				// Decode the input from resolver tx calldata
				input, err = DecodeResolvedInput(tx.Data())
				if err != nil {
					d.log.Error("failed to decode resolved input", "block", block.Number, "txIdx", i, "err", err)
					continue
				}
				if err := comm.Verify(input); err != nil {
					d.log.Error("failed to verify commitment", "block", block.Number, "txIdx", i, "err", err)
					continue
				}
			}
			d.log.Info("challenge resolved", "block", block, "txIdx", i, "comm", comm.Encode())
			d.state.SetResolvedChallenge(comm.Encode(), input, log.BlockNumber)
		case ChallengeActive:
			d.log.Info("detected new active challenge", "block", block, "comm", comm.Encode())
			d.state.SetActiveChallenge(comm.Encode(), log.BlockNumber, d.cfg.ResolveWindow)
		default:
			d.log.Warn("skipping unknown challenge status", "block", block.Number, "tx", i, "log", log.Index, "status", status, "comm", comm.Encode())
		}
	}
	return nil
}

// fetchChallengeLogs returns logs for challenge events if any for the given block
func (d *DA) fetchChallengeLogs(ctx context.Context, l1 L1Fetcher, block eth.BlockID) ([]*types.Log, error) {
	var logs []*types.Log
	// Don't look at the challenge contract if there is no challenge contract.
	if d.cfg.CommitmentType == GenericCommitmentType {
		return logs, nil
	}
	//cached with deposits events call so not expensive
	_, receipts, err := l1.FetchReceipts(ctx, block.Hash)
	if err != nil {
		return nil, err
	}
	d.log.Info("loading challenges", "epoch", block.Number, "numReceipts", len(receipts))
	for _, rec := range receipts {
		// skip error logs
		if rec.Status != types.ReceiptStatusSuccessful {
			continue
		}
		for _, log := range rec.Logs {
			if log.Address == d.cfg.DAChallengeContractAddress && len(log.Topics) > 0 && log.Topics[0] == ChallengeStatusEventABIHash {
				logs = append(logs, log)
			}
		}
	}

	return logs, nil
}

// decodeChallengeStatus decodes and validates a challenge event from a transaction log, returning the associated commitment bytes.
func (d *DA) decodeChallengeStatus(log *types.Log) (ChallengeStatus, CommitmentData, error) {
	event, err := DecodeChallengeStatusEvent(log)
	if err != nil {
		return 0, nil, err
	}
	comm, err := DecodeCommitmentData(event.ChallengedCommitment)
	if err != nil {
		return 0, nil, err
	}
	d.log.Debug("decoded challenge status event", "log", log, "event", event, "comm", fmt.Sprintf("%x", comm.Encode()))

	bn := event.ChallengedBlockNumber.Uint64()
	// IsTracking just validates whether the commitment was challenged for the correct block number
	// if it has been loaded from the batcher inbox before. Spam commitments will be tracked but
	// ignored and evicted unless derivation encounters the commitment.
	if !d.state.IsTracking(comm.Encode(), bn) {
		return 0, nil, fmt.Errorf("%w: %x at block %d", ErrInvalidChallenge, comm.Encode(), bn)
	}
	return ChallengeStatus(event.Status), comm, nil
}

var (
	ChallengeStatusEventName    = "ChallengeStatusChanged"
	ChallengeStatusEventABI     = "ChallengeStatusChanged(uint256,bytes,uint8)"
	ChallengeStatusEventABIHash = crypto.Keccak256Hash([]byte(ChallengeStatusEventABI))
)

// DecodeChallengeStatusEvent decodes the challenge status event from the log data and the indexed challenged
// hash and block number from the topics.
func DecodeChallengeStatusEvent(log *types.Log) (*bindings.DataAvailabilityChallengeChallengeStatusChanged, error) {
	// abi lazy loaded, cached after decoded once
	dacAbi, err := bindings.DataAvailabilityChallengeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	var event bindings.DataAvailabilityChallengeChallengeStatusChanged
	err = dacAbi.UnpackIntoInterface(&event, ChallengeStatusEventName, log.Data)
	if err != nil {
		return nil, err
	}
	var indexed abi.Arguments
	for _, arg := range dacAbi.Events[ChallengeStatusEventName].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(&event, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	return &event, nil
}

// DecodeResolvedInput decodes the preimage bytes from the tx input data.
func DecodeResolvedInput(data []byte) ([]byte, error) {
	dacAbi, err := bindings.DataAvailabilityChallengeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	args := make(map[string]interface{})
	err = dacAbi.Methods["resolve"].Inputs.UnpackIntoMap(args, data[4:])
	if err != nil {
		return nil, err
	}
	rd, ok := args["resolveData"].([]byte)
	if !ok || len(rd) == 0 {
		return nil, fmt.Errorf("invalid resolve data")
	}
	return rd, nil
}
