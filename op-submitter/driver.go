package submitter

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum-optimism/optimism/op-submitter/bindings"
	"github.com/ethereum-optimism/optimism/op-submitter/metrics"
	"github.com/ethereum-optimism/optimism/op-submitter/proposer/source"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrSubmitterNotRunning = errors.New("submitter is not running")
)

type L1Client interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	// CodeAt returns the code of the given account. This is needed to differentiate
	// between contract internal errors and the local chain being out of sync.
	CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error)

	// CallContract executes an Ethereum contract call with the specified data as the
	// input.
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

type L2NRContract interface {
	Version(*bind.CallOpts) (string, error)
	NextBlockNumber(*bind.CallOpts) (*big.Int, error)
}

type RollupClient interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

type DriverSetup struct {
	Log         log.Logger
	Metr        metrics.Metricer
	Cfg         SubmitterConfig
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

	l2nrContract L2NRContract
	l2nrABI      *abi.ABI

	running atomic.Bool
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

	return newExecutePayloadSubmitter(ctx, cancel, setup)
}

func newExecutePayloadSubmitter(ctx context.Context, cancel context.CancelFunc, setup DriverSetup) (*L2OutputSubmitter, error) {
	l2nrContract, err := bindings.NewL2NativeRollupCaller(*setup.Cfg.L2NativeRollupAddr, setup.L1Client)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create L2NR at address %s: %w", setup.Cfg.L2NativeRollupAddr, err)
	}

	cCtx, cCancel := context.WithTimeout(ctx, setup.Cfg.NetworkTimeout)
	defer cCancel()
	version, err := l2nrContract.Version(&bind.CallOpts{Context: cCtx})
	if err != nil {
		cancel()
		return nil, err
	}
	log.Info("Connected to L2NativeRollup", "address", setup.Cfg.L2NativeRollupAddr, "version", version)

	parsed, err := bindings.L2NativeRollupMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, err
	}

	return &L2OutputSubmitter{
		DriverSetup: setup,
		done:        make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

func (l *L2OutputSubmitter) StartL2OutputSubmitting() error {
	l.Log.Info("Starting Submitter")

	if !l.running.CompareAndSwap(false, true) {
		return errors.New("submitter is already running")
	}

	if l.Cfg.WaitNodeSync {
		err := l.waitNodeSync()
		if err != nil {
			return fmt.Errorf("error waiting for node sync: %w", err)
		}
	}

	l.wg.Add(1)
	go l.loop()

	l.Log.Info("Submitter started")
	return nil
}

func (l *L2OutputSubmitter) StopL2OutputSubmittingIfRunning() error {
	err := l.StopL2OutputSubmitting()
	if errors.Is(err, ErrSubmitterNotRunning) {
		return nil
	}
	return err
}

func (l *L2OutputSubmitter) StopL2OutputSubmitting() error {
	l.Log.Info("Stopping Submitter")

	if !l.running.CompareAndSwap(true, false) {
		return ErrSubmitterNotRunning
	}

	l.cancel()
	close(l.done)
	l.wg.Wait()

	l.Log.Info("Submitter stopped")
	return nil
}

// FetchL2NativeRollup gets the next output proposal for an 'execute' precompile submission.
// It queries the L2NR for the earliest next block number that should be proposed.
// It returns the payload to propose, and whether the proposal should be submitted at all.
// The passed context is expected to be a lifecycle context. A network timeout
// context will be derived from it.
func (l *L2OutputSubmitter) FetchExecutePayload(ctx context.Context) (source.Submission, bool, error) {
	if l.l2nrContract == nil {
		return source.Proposal{}, false, fmt.Errorf("L2NativeRollup contract not set, cannot fetch next output info")
	}

	cCtx, cancel := context.WithTimeout(ctx, l.Cfg.NetworkTimeout)
	defer cancel()
	callOpts := &bind.CallOpts{
		From:    l.Txmgr.From(),
		Context: cCtx,
	}
	nextCheckpointBlockBig, err := l.l2nrContract.NextBlockNumber(callOpts)
	if err != nil {
		return source.Proposal{}, false, fmt.Errorf("querying next block number: %w", err)
	}
	nextCheckpointBlock := nextCheckpointBlockBig.Uint64()
	// Fetch the current L2 heads
	currentBlockNumber, err := l.FetchCurrentBlockNumber(ctx)
	if err != nil {
		return source.Proposal{}, false, err
	}

	// Ensure that we do not submit a block in the future
	if currentBlockNumber < nextCheckpointBlock {
		l.Log.Debug("Submitter submission interval has not elapsed", "currentBlockNumber", currentBlockNumber, "nextBlockNumber", nextCheckpointBlock)
		return source.Proposal{}, false, nil
	}

	output, err := l.FetchOutput(ctx, nextCheckpointBlock)
	if err != nil {
		return source.Proposal{}, false, fmt.Errorf("fetching output: %w", err)
	}

	return output, true, nil
}

// FetchCurrentBlockNumber gets the current block number from the [L2OutputSubmitter]'s [RollupClient]. If the `AllowNonFinalized` configuration
// option is set, it will return the safe head block number, and if not, it will return the finalized head block number.
func (l *L2OutputSubmitter) FetchCurrentBlockNumber(ctx context.Context) (uint64, error) {
	status, err := l.ProposalSource.SyncStatus(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting sync status: %w", err)
	}

	return status.SafeL2, nil
}

func (l *L2OutputSubmitter) FetchOutput(ctx context.Context, block uint64) (source.Proposal, error) {
	output, err := l.ProposalSource.ProposalAtSequenceNum(ctx, block)
	if err != nil {
		return source.Proposal{}, fmt.Errorf("fetching output at block %d: %w", block, err)
	}
	if onum := output.SequenceNum; onum != block { // sanity check, e.g. in case of bad RPC caching
		return source.Proposal{}, fmt.Errorf("output block number %d mismatches requested %d", output.SequenceNum, block)
	}
	return output, nil
}

// ProposeL2OutputTxData creates the transaction data for the ProposeL2Output function
func (l *L2OutputSubmitter) ProposeL2OutputTxData(output source.Proposal) ([]byte, error) {
	return proposeL2OutputTxData(l.l2nrABI, output)
}

// proposeL2OutputTxData creates the transaction data for the ProposeL2Output function
func proposeL2OutputTxData(abi *abi.ABI, output source.Proposal) ([]byte, error) {
	return abi.Pack(
		"proposeL2Output",
		output.Root,
		new(big.Int).SetUint64(output.SequenceNum),
		output.CurrentL1.Hash,
		new(big.Int).SetUint64(output.CurrentL1.Number))
}

// sendTransaction creates & sends transactions through the underlying transaction manager.
func (l *L2OutputSubmitter) sendTransaction(ctx context.Context, output source.Proposal) error {
	l.Log.Info("Proposing output root", "output", output.Root, "block", output.SequenceNum)
	var receipt *types.Receipt
	data, err := l.ProposeL2OutputTxData(output)
	if err != nil {
		return err
	}
	receipt, err = l.Txmgr.Send(ctx, txmgr.TxCandidate{
		TxData:   data,
		To:       l.Cfg.L2NativeRollupAddr,
		GasLimit: 0,
	})
	if err != nil {
		return err
	}

	if receipt.Status == types.ReceiptStatusFailed {
		l.Log.Error("Submitter tx successfully published but reverted", "tx_hash", receipt.TxHash)
	} else {
		l.Log.Info("Submitter tx successfully published",
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
			var proposal source.Proposal
			var shouldPropose bool
			var err error
			proposal, shouldPropose, err = l.FetchExecutePayload(ctx)
			if err != nil {
				l.Log.Warn("Error getting proposal", "err", err)
				continue
			} else if !shouldPropose {
				// debug logging already in FetchExecutePayload
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

	return dial.WaitL1Sync(l.ctx, l.Log, l1head, time.Second*12, func(ctx context.Context) (eth.L1BlockRef, error) {
		status, err := l.ProposalSource.SyncStatus(ctx)
		if err != nil {
			return eth.L1BlockRef{}, err
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
