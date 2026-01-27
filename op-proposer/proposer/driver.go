package proposer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-proposer/contracts"
	"github.com/ethereum-optimism/optimism/op-proposer/metrics"
	"github.com/ethereum-optimism/optimism/op-proposer/proposer/source"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var ErrProposerNotRunning = errors.New("proposer is not running")

type L1Client interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	// CodeAt returns the code of the given account. This is needed to differentiate
	// between contract internal errors and the local chain being out of sync.
	CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error)

	// CallContract executes an Ethereum contract call with the specified data as the
	// input.
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

type DGFContract interface {
	Version(ctx context.Context) (string, error)
	HasProposedSince(ctx context.Context, proposer common.Address, cutoff time.Time, gameType uint32) (bool, time.Time, common.Hash, error)
	ProposalTx(ctx context.Context, gameType uint32, outputRoot common.Hash, extraData []byte) (txmgr.TxCandidate, error)
}

type RollupClient interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

type DriverSetup struct {
	Log         log.Logger
	Metr        metrics.Metricer
	Cfg         ProposerConfig
	Txmgr       txmgr.TxManager
	L1Client    L1Client
	Multicaller *batching.MultiCaller

	// ProposalSource retrieves the proposal data to submit
	ProposalSource source.ProposalSource
}

// L2OutputSubmitter is responsible for proposing outputs
type L2OutputSubmitter struct {
	DriverSetup

	wg   sync.WaitGroup
	done chan struct{}

	ctx    context.Context
	cancel context.CancelFunc

	running atomic.Bool

	dgfContract DGFContract
}

// NewL2OutputSubmitter creates a new L2 Output Submitter
func NewL2OutputSubmitter(setup DriverSetup) (_ *L2OutputSubmitter, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	// The above context is long-lived, and passed to the `L2OutputSubmitter` instance. This context is closed by
	// `StopL2OutputSubmitting`, but if this function returns an error or panics, we want to ensure that the context
	// doesn't leak.
	defer func() {
		if err != nil || recover() != nil {
			cancel()
		}
	}()

	if setup.Cfg.DisputeGameFactoryAddr == nil {
		return nil, errors.New("missing DisputeGameFactory address")
	}

	return newDGFSubmitter(ctx, cancel, setup)
}

func newDGFSubmitter(ctx context.Context, cancel context.CancelFunc, setup DriverSetup) (*L2OutputSubmitter, error) {
	dgfCaller := contracts.NewDisputeGameFactory(*setup.Cfg.DisputeGameFactoryAddr, setup.Multicaller, setup.Cfg.NetworkTimeout)

	version, err := dgfCaller.Version(ctx)
	if err != nil {
		cancel()
		return nil, err
	}
	log.Info("Connected to DisputeGameFactory", "address", setup.Cfg.DisputeGameFactoryAddr, "version", version)

	return &L2OutputSubmitter{
		DriverSetup: setup,
		done:        make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,

		dgfContract: dgfCaller,
	}, nil
}

func (l *L2OutputSubmitter) StartL2OutputSubmitting() error {
	l.Log.Info("Starting Proposer")

	if !l.running.CompareAndSwap(false, true) {
		return errors.New("proposer is already running")
	}

	if l.Cfg.WaitNodeSync {
		err := l.waitNodeSync()
		if err != nil {
			return fmt.Errorf("error waiting for node sync: %w", err)
		}
	}

	l.wg.Add(1)
	go l.loop()

	l.Log.Info("Proposer started")
	return nil
}

func (l *L2OutputSubmitter) StopL2OutputSubmittingIfRunning() error {
	err := l.StopL2OutputSubmitting()
	if errors.Is(err, ErrProposerNotRunning) {
		return nil
	}
	return err
}

func (l *L2OutputSubmitter) StopL2OutputSubmitting() error {
	l.Log.Info("Stopping Proposer")

	if !l.running.CompareAndSwap(true, false) {
		return ErrProposerNotRunning
	}

	l.cancel()
	close(l.done)
	l.wg.Wait()

	l.Log.Info("Proposer stopped")
	return nil
}

// FetchDGFOutput queries the DGF for the latest game and infers whether it is time to make another proposal
// If necessary, it gets the next output proposal for the DGF, and returns it along with
// a boolean for whether the proposal should be submitted at all.
// The passed context is expected to be a lifecycle context. A network timeout
// context will be derived from it.
func (l *L2OutputSubmitter) FetchDGFOutput(ctx context.Context) (source.Proposal, bool, error) {
	cutoff := time.Now().Add(-l.Cfg.ProposalInterval)
	proposedRecently, proposalTime, claim, err := l.dgfContract.HasProposedSince(ctx, l.Txmgr.From(), cutoff, l.Cfg.DisputeGameType)
	if err != nil {
		return source.Proposal{}, false, fmt.Errorf("could not check for recent proposal: %w", err)
	}

	if proposedRecently {
		l.Log.Debug("Duration since last game not past proposal interval", "duration", time.Since(proposalTime))
		return source.Proposal{}, false, nil
	}

	// Fetch the current L2 heads
	currentBlockNumber, err := l.FetchCurrentBlockNumber(ctx)
	if err != nil {
		return source.Proposal{}, false, fmt.Errorf("could not fetch current block number: %w", err)
	}

	if currentBlockNumber == 0 {
		l.Log.Info("Skipping proposal for genesis block")
		return source.Proposal{}, false, nil
	}

	output, err := l.FetchOutput(ctx, currentBlockNumber)
	if err != nil {
		return source.Proposal{}, false, fmt.Errorf("could not fetch output at current block number %d: %w", currentBlockNumber, err)
	}

	if claim == output.Root {
		l.Log.Debug("Skipping proposal: output root unchanged since last proposed game", "last_proposed_root", claim, "output_root", output.Root)
		return source.Proposal{}, false, nil
	}

	l.Log.Info("No proposals found for at least proposal interval, submitting proposal now", "proposalInterval", l.Cfg.ProposalInterval)

	return output, true, nil
}

// FetchCurrentBlockNumber gets the current block number from the [L2OutputSubmitter]'s [RollupClient]. If the `AllowNonFinalized` configuration
// option is set, it will return the safe head block number, and if not, it will return the finalized head block number.
func (l *L2OutputSubmitter) FetchCurrentBlockNumber(ctx context.Context) (uint64, error) {
	status, err := l.ProposalSource.SyncStatus(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting sync status: %w", err)
	}

	// Use either the finalized or safe head depending on the config. Finalized head is default & safer.
	if l.Cfg.AllowNonFinalized {
		return status.SafeL2, nil
	}
	return status.FinalizedL2, nil
}

func (l *L2OutputSubmitter) FetchOutput(ctx context.Context, block uint64) (source.Proposal, error) {
	proposal, err := l.ProposalSource.ProposalAtSequenceNum(ctx, block)
	if err != nil {
		return source.Proposal{}, fmt.Errorf("fetching proposal at block %d: %w", block, err)
	}
	if onum := proposal.SequenceNum; onum != block && !proposal.IsSuperRootProposal() { // sanity check, e.g. in case of bad RPC caching
		return source.Proposal{}, fmt.Errorf("proposal block number %d mismatches requested %d", proposal.SequenceNum, block)
	}
	return proposal, nil
}

func (l *L2OutputSubmitter) ProposeL2OutputDGFTxCandidate(ctx context.Context, output source.Proposal) (txmgr.TxCandidate, error) {
	cCtx, cancel := context.WithTimeout(ctx, l.Cfg.NetworkTimeout)
	defer cancel()
	return l.dgfContract.ProposalTx(cCtx, l.Cfg.DisputeGameType, output.Root, output.ExtraData())
}

// sendTransaction creates & sends transactions through the underlying transaction manager.
func (l *L2OutputSubmitter) sendTransaction(ctx context.Context, output source.Proposal) error {
	l.Log.Info("Proposing output root", "output", output.Root, "sequenceNum", output.SequenceNum, "extraData", output.ExtraData())

	candidate, err := l.ProposeL2OutputDGFTxCandidate(ctx, output)
	if err != nil {
		return fmt.Errorf("failed to create DGF tx candidate: %w", err)
	}

	receipt, err := l.Txmgr.Send(ctx, candidate)
	if err != nil {
		return fmt.Errorf("failed to send proposal tx: %w", err)
	}

	if receipt.Status == types.ReceiptStatusFailed {
		l.Log.Error("Proposer tx successfully published but reverted", "tx_hash", receipt.TxHash)
	} else {
		l.Log.Info("Proposer tx successfully published",
			"tx_hash", receipt.TxHash,
			"l1blocknum", output.CurrentL1.Number,
			"l1blockhash", output.CurrentL1.Hash)
	}
	return nil
}

// loop is responsible for creating & submitting the next outputs
// The loop regularly polls the L2 chain to infer whether to make the next proposal.
func (l *L2OutputSubmitter) loop() {
	defer l.wg.Done()
	defer l.Log.Info("loop returning")
	ctx := l.ctx
	ticker := time.NewTicker(l.Cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// prioritize quit signal
			select {
			case <-l.done:
				return
			default:
			}

			// A note on retrying: the outer ticker already runs on a short
			// poll interval, which has a default value of 6 seconds. So no
			// retry logic is needed around proposal fetching here.
			proposal, shouldPropose, err := l.FetchDGFOutput(ctx)
			if err != nil {
				l.Log.Warn("Error getting proposal", "err", err)
				continue
			} else if !shouldPropose {
				// debug logging already in FetchDGFOutput
				continue
			}

			l.proposeOutput(ctx, proposal)
		case <-l.done:
			return
		}
	}
}

func (l *L2OutputSubmitter) waitNodeSync() error {
	cCtx, cancel := context.WithTimeout(l.ctx, l.Cfg.NetworkTimeout)
	defer cancel()

	l1head, err := l.Txmgr.BlockNumber(cCtx)
	if err != nil {
		return fmt.Errorf("failed to retrieve current L1 block number: %w", err)
	}

	return dial.WaitL1Sync(l.ctx, l.Log, l1head, time.Second*12, func(ctx context.Context) (eth.BlockID, error) {
		status, err := l.ProposalSource.SyncStatus(ctx)
		if err != nil {
			return eth.BlockID{}, err
		}
		return status.CurrentL1, nil
	})
}

func (l *L2OutputSubmitter) proposeOutput(ctx context.Context, output source.Proposal) {
	cCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	if err := l.sendTransaction(cCtx, output); err != nil {
		logCtx := []interface{}{
			"err", err,
			"l1blocknum", output.CurrentL1.Number,
			"l1blockhash", output.CurrentL1.Hash,
		}
		// Add legacy data only if available
		if output.Legacy.HeadL1 != (eth.L1BlockRef{}) {
			logCtx = append(logCtx, "l1head", output.Legacy.HeadL1.Number)
		}
		l.Log.Error("Failed to send proposal transaction", logCtx...)
		return
	}
	l.Metr.RecordL2Proposal(output.SequenceNum)
	if output.Legacy.BlockRef != (eth.L2BlockRef{}) {
		// Record legacy metrics when available
		l.Metr.RecordL2BlocksProposed(output.Legacy.BlockRef)
	}
}
