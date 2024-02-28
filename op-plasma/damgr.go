package plasma

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ErrPendingChallenge is return when data is not available but can still be challenged/resolved
// so derivation should halt temporarily.
var ErrPendingChallenge = errors.New("not found, pending challenge")

// ErrExpiredChallenge is returned when a challenge was not resolved and derivation should skip this input.
var ErrExpiredChallenge = errors.New("challenge expired")

// ErrMissingPastWindow is returned when the input data is MIA and cannot be challenged.
// This is a protocol fatal error.
var ErrMissingPastWindow = errors.New("data missing past window")

// L1Fetcher is the required interface for syncing the DA challenge contract state.
type L1Fetcher interface {
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
	L1BlockRefByNumber(context.Context, uint64) (eth.L1BlockRef, error)
}

// DAStorage interface for calling the DA storage server.
type DAStorage interface {
	GetInput(ctx context.Context, key []byte) ([]byte, error)
	SetInput(ctx context.Context, img []byte) ([]byte, error)
}

// Config is the relevant subset of rollup config for plasma DA.
type Config struct {
	// Required for filtering contract events
	DAChallengeContractAddress common.Address
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
	l1      L1Fetcher

	state *State

	// the latest l1 block we synced challenge contract events from
	origin eth.BlockID
	// the latest recorded finalized head as per the challenge contract
	finalizedHead eth.L1BlockRef

	finalizedHeadSignalFunc eth.HeadSignalFn
}

// NewPlasmaDA creates a new PlasmaDA instance with the given log and CLIConfig.
func NewPlasmaDA(log log.Logger, cli CLIConfig, cfg Config, l1f L1Fetcher, metrics Metricer) *DA {
	return NewPlasmaDAWithStorage(log, cfg, cli.NewDAClient(), l1f, metrics)
}

// NewPlasmaDAWithStorage creates a new PlasmaDA instance with the given log and DAStorage interface.
func NewPlasmaDAWithStorage(log log.Logger, cfg Config, storage DAStorage, l1f L1Fetcher, metrics Metricer) *DA {
	return &DA{
		log:     log,
		cfg:     cfg,
		storage: storage,
		l1:      l1f,
		metrics: metrics,
		state:   NewState(log, metrics),
	}
}

// OnFinalizedHeadSignal sets the callback function to be called when the finalized head is updated.
// This will signal to the engine queue that will set the proper L2 block as finalized.
func (d *DA) OnFinalizedHeadSignal(f eth.HeadSignalFn) {
	d.finalizedHeadSignalFunc = f
}

// GetInput returns the input data for the given commitment bytes. blockNumber is required to lookup
// the challenge status in the DataAvailabilityChallenge L1 contract.
func (d *DA) GetInput(ctx context.Context, commitment []byte, blockId eth.BlockID) (eth.Data, error) {
	// If the challenge head is ahead in the case of a pipeline reset or stall, we might have synced a
	// challenge event for this commitment. Otherwise we mark the commitment as part of the canonical
	// chain so potential future challenge events can be selected.
	ch := d.state.GetOrTrackChallenge(commitment, blockId.Number, d.cfg.ChallengeWindow)

	// Fetch the input from the DA storage.
	data, err := d.storage.GetInput(ctx, commitment)

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
			if err := d.LookAhead(ctx); err != nil {
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
		// data not found in storage, return from challenge resolved input
		resolvedInput, err := d.state.GetResolvedInput(commitment)
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
				if err := d.LookAhead(ctx); err != nil {
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
func (d *DA) AdvanceL1Origin(ctx context.Context, block eth.BlockID) error {
	// do not repeat for the same origin
	if block.Number <= d.origin.Number {
		return nil
	}
	// sync challenges for the given block ID
	if err := d.LoadChallengeEvents(ctx, block); err != nil {
		return err
	}
	// advance challenge window, computing the finalized head
	bn, err := d.state.ExpireChallenges(block.Number)
	if err != nil {
		return err
	}

	if bn > d.finalizedHead.Number {
		ref, err := d.l1.L1BlockRefByNumber(ctx, bn)
		if err != nil {
			return err
		}
		d.metrics.RecordChallengesHead("finalized", bn)

		// if we get a greater finalized head, signal to the engine queue
		if d.finalizedHeadSignalFunc != nil {
			d.finalizedHeadSignalFunc(ctx, ref)

		}
		// prune old state
		d.state.Prune(bn)
		d.finalizedHead = ref

	}
	d.origin = block
	d.metrics.RecordChallengesHead("latest", d.origin.Number)

	d.log.Info("processed plasma l1 origin", "origin", block, "next-finalized", bn, "finalized", d.finalizedHead.Number)
	return nil
}

// LoadChallengeEvents fetches the l1 block receipts and updates the challenge status
func (d *DA) LoadChallengeEvents(ctx context.Context, block eth.BlockID) error {
	//cached with deposits events call so not expensive
	_, receipts, err := d.l1.FetchReceipts(ctx, block.Hash)
	if err != nil {
		return err
	}
	d.log.Info("updating challenges", "epoch", block.Number, "numReceipts", len(receipts))
	for i, rec := range receipts {
		if rec.Status != types.ReceiptStatusSuccessful {
			continue
		}
		for j, log := range rec.Logs {
			if log.Address == d.cfg.DAChallengeContractAddress && len(log.Topics) > 0 && log.Topics[0] == ChallengeStatusEventABIHash {
				event, err := DecodeChallengeStatusEvent(log)
				if err != nil {
					d.log.Error("failed to decode challenge event", "block", block.Number, "tx", i, "log", j, "err", err)
					continue
				}
				d.log.Info("decoded challenge status event", "block", block.Number, "tx", i, "log", j, "event", event)
				comm, err := DecodeKeccak256(event.ChallengedCommitment)
				if err != nil {
					d.log.Error("failed to decode commitment", "block", block.Number, "tx", i, "err", err)
					continue
				}

				bn := event.ChallengedBlockNumber.Uint64()
				// if we are not tracking the commitment from processing the l1 origin in derivation,
				// i.e. someone challenged garbage data, this challenge is invalid.
				if !d.state.IsTracking(comm.Encode(), bn) {
					d.log.Warn("skipping invalid challenge", "block", bn)
					continue
				}
				switch ChallengeStatus(event.Status) {
				case ChallengeResolved:
					// cached with input resolution call so not expensive
					_, txs, err := d.l1.InfoAndTxsByHash(ctx, block.Hash)
					if err != nil {
						d.log.Error("failed to fetch l1 block", "block", block.Number, "err", err)
						continue
					}
					tx := txs[i]
					// txs and receipts must be in the same order
					if tx.Hash() != rec.TxHash {
						d.log.Error("tx hash mismatch", "block", block.Number, "tx", i, "log", j, "txHash", tx.Hash(), "receiptTxHash", rec.TxHash)
						continue
					}
					input, err := DecodeResolvedInput(tx.Data())
					if err != nil {
						d.log.Error("failed to decode resolved input", "block", block.Number, "tx", i, "err", err)
						continue
					}
					if err := comm.Verify(input); err != nil {
						d.log.Error("failed to verify commitment", "block", block.Number, "tx", i, "err", err)
						continue
					}
					d.log.Debug("resolved input", "block", block.Number, "tx", i)
					d.state.SetResolvedChallenge(comm.Encode(), input, log.BlockNumber)
				case ChallengeActive:
					d.state.SetActiveChallenge(comm.Encode(), log.BlockNumber, d.cfg.ResolveWindow)
				default:
					d.log.Warn("skipping unknown challenge status", "block", block.Number, "tx", i, "log", j, "status", event.Status)
				}
			}
		}

	}
	return nil
}

// LookAhead increments the challenges head and process the new block if it exists.
// It is only used if the derivation pipeline stalls and we need to wait for a challenge to be resolved
// to get the next input.
func (d *DA) LookAhead(ctx context.Context) error {
	blkRef, err := d.l1.L1BlockRefByNumber(ctx, d.origin.Number+1)
	if errors.Is(err, ethereum.NotFound) {
		return io.EOF
	}
	if err != nil {
		d.log.Error("failed to fetch l1 head", "err", err)
		return err
	}
	return d.AdvanceL1Origin(ctx, blkRef.ID())
}

var (
	ChallengeStatusEventName    = "ChallengeStatusChanged"
	ChallengeStatusEventABI     = "ChallengeStatusChanged(uint256,bytes,uint8)"
	ChallengeStatusEventABIHash = crypto.Keccak256Hash([]byte(ChallengeStatusEventABI))
)

// State getter for inspecting
func (d *DA) State() *State {
	return d.state
}

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
	dacAbi, _ := bindings.DataAvailabilityChallengeMetaData.GetAbi()

	args := make(map[string]interface{})
	err := dacAbi.Methods["resolve"].Inputs.UnpackIntoMap(args, data[4:])
	if err != nil {
		return nil, err
	}
	rd := args["resolveData"].([]byte)
	if rd == nil {
		return nil, fmt.Errorf("invalid resolve data")
	}
	return rd, nil
}
