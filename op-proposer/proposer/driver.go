package proposer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-proposer/bindings"
	"github.com/ethereum-optimism/optimism/op-proposer/contracts"
	"github.com/ethereum-optimism/optimism/op-proposer/metrics"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var (
	supportedL2OutputVersion = eth.Bytes32{}
	ErrProposerNotRunning    = errors.New("proposer is not running")
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

type L2OOContract interface {
	Version(*bind.CallOpts) (string, error)
	NextBlockNumber(*bind.CallOpts) (*big.Int, error)
}

type DGFContract interface {
	Version(ctx context.Context) (string, error)
	HasProposedSince(ctx context.Context, proposer common.Address, cutoff time.Time, gameType uint32) (bool, time.Time, error)
	ProposalTx(ctx context.Context, gameType uint32, outputRoot common.Hash, l2BlockNum uint64) (txmgr.TxCandidate, error)
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

	// RollupProvider's RollupClient() is used to retrieve output roots from
	RollupProvider dial.RollupProvider
}

// L2OutputSubmitter is responsible for proposing outputs
type L2OutputSubmitter struct {
	DriverSetup

	wg   sync.WaitGroup
	done chan struct{}

	ctx    context.Context
	cancel context.CancelFunc

	mutex   sync.Mutex
	running bool

	l2ooContract L2OOContract
	l2ooABI      *abi.ABI

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

	if setup.Cfg.L2OutputOracleAddr != nil {
		return newL2OOSubmitter(ctx, cancel, setup)
	} else if setup.Cfg.DisputeGameFactoryAddr != nil {
		return newDGFSubmitter(ctx, cancel, setup)
	} else {
		return nil, errors.New("neither the `L2OutputOracle` nor `DisputeGameFactory` addresses were provided")
	}
}

func newL2OOSubmitter(ctx context.Context, cancel context.CancelFunc, setup DriverSetup) (*L2OutputSubmitter, error) {
	l2ooContract, err := bindings.NewL2OutputOracleCaller(*setup.Cfg.L2OutputOracleAddr, setup.L1Client)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create L2OO at address %s: %w", setup.Cfg.L2OutputOracleAddr, err)
	}

	cCtx, cCancel := context.WithTimeout(ctx, setup.Cfg.NetworkTimeout)
	defer cCancel()
	version, err := l2ooContract.Version(&bind.CallOpts{Context: cCtx})
	if err != nil {
		cancel()
		return nil, err
	}
	log.Info("Connected to L2OutputOracle", "address", setup.Cfg.L2OutputOracleAddr, "version", version)

	parsed, err := bindings.L2OutputOracleMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, err
	}

	return &L2OutputSubmitter{
		DriverSetup: setup,
		done:        make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,

		l2ooContract: l2ooContract,
		l2ooABI:      parsed,
	}, nil
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

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.running {
		return errors.New("proposer is already running")
	}
	l.running = true

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

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if !l.running {
		return ErrProposerNotRunning
	}
	l.running = false

	l.cancel()
	close(l.done)
	l.wg.Wait()

	l.Log.Info("Proposer stopped")
	return nil
}

// FetchL2OOOutput gets the next output proposal for the L2OO.
// It queries the L2OO for the earliest next block number that should be proposed.
// It returns the output to propose, and whether the proposal should be submitted at all.
// The passed context is expected to be a lifecycle context. A network timeout
// context will be derived from it.
func (l *L2OutputSubmitter) FetchL2OOOutput(ctx context.Context) (*eth.OutputResponse, bool, error) {
	if l.l2ooContract == nil {
		return nil, false, fmt.Errorf("L2OutputOracle contract not set, cannot fetch next output info")
	}

	cCtx, cancel := context.WithTimeout(ctx, l.Cfg.NetworkTimeout)
	defer cancel()
	callOpts := &bind.CallOpts{
		From:    l.Txmgr.From(),
		Context: cCtx,
	}
	nextCheckpointBlockBig, err := l.l2ooContract.NextBlockNumber(callOpts)
	if err != nil {
		return nil, false, fmt.Errorf("querying next block number: %w", err)
	}
	nextCheckpointBlock := nextCheckpointBlockBig.Uint64()
	// Fetch the current L2 heads
	currentBlockNumber, err := l.FetchCurrentBlockNumber(ctx)
	if err != nil {
		return nil, false, err
	}

	// Ensure that we do not submit a block in the future
	if currentBlockNumber < nextCheckpointBlock {
		l.Log.Debug("Proposer submission interval has not elapsed", "currentBlockNumber", currentBlockNumber, "nextBlockNumber", nextCheckpointBlock)
		return nil, false, nil
	}

	output, err := l.FetchOutput(ctx, nextCheckpointBlock)
	if err != nil {
		return nil, false, fmt.Errorf("fetching output: %w", err)
	}

	// Always propose if it's part of the Finalized L2 chain. Or if allowed, if it's part of the safe L2 chain.
	if output.BlockRef.Number > output.Status.FinalizedL2.Number && (!l.Cfg.AllowNonFinalized || output.BlockRef.Number > output.Status.SafeL2.Number) {
		l.Log.Debug("Not proposing yet, L2 block is not ready for proposal",
			"l2_proposal", output.BlockRef,
			"l2_safe", output.Status.SafeL2,
			"l2_finalized", output.Status.FinalizedL2,
			"allow_non_finalized", l.Cfg.AllowNonFinalized)
		return output, false, nil
	}
	return output, true, nil
}

// FetchDGFOutput queries the DGF for the latest game and infers whether it is time to make another proposal
// If necessary, it gets the next output proposal for the DGF, and returns it along with
// a boolean for whether the proposal should be submitted at all.
// The passed context is expected to be a lifecycle context. A network timeout
// context will be derived from it.
func (l *L2OutputSubmitter) FetchDGFOutput(ctx context.Context) (*eth.OutputResponse, bool, error) {
	cutoff := time.Now().Add(-l.Cfg.ProposalInterval)
	proposedRecently, proposalTime, err := l.dgfContract.HasProposedSince(ctx, l.Txmgr.From(), cutoff, l.Cfg.DisputeGameType)
	if err != nil {
		return nil, false, fmt.Errorf("could not check for recent proposal: %w", err)
	}

	if proposedRecently {
		l.Log.Debug("Duration since last game not past proposal interval", "duration", time.Since(proposalTime))
		return nil, false, nil
	}
	l.Log.Info("No proposals found for at least proposal interval, submitting proposal now", "proposalInterval", l.Cfg.ProposalInterval)

	// Fetch the current L2 heads
	currentBlockNumber, err := l.FetchCurrentBlockNumber(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("could not fetch current block number: %w", err)
	}

	if currentBlockNumber == 0 {
		l.Log.Info("Skipping proposal for genesis block")
		return nil, false, nil
	}

	output, err := l.FetchOutput(ctx, currentBlockNumber)
	if err != nil {
		return nil, false, fmt.Errorf("could not fetch output at current block number %d: %w", currentBlockNumber, err)
	}

	return output, true, nil
}

// FetchCurrentBlockNumber gets the current block number from the [L2OutputSubmitter]'s [RollupClient]. If the `AllowNonFinalized` configuration
// option is set, it will return the safe head block number, and if not, it will return the finalized head block number.
func (l *L2OutputSubmitter) FetchCurrentBlockNumber(ctx context.Context) (uint64, error) {
	rollupClient, err := l.RollupProvider.RollupClient(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting rollup client: %w", err)
	}

	status, err := rollupClient.SyncStatus(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting sync status: %w", err)
	}

	// Use either the finalized or safe head depending on the config. Finalized head is default & safer.
	if l.Cfg.AllowNonFinalized {
		return status.SafeL2.Number, nil
	}
	return status.FinalizedL2.Number, nil
}

func (l *L2OutputSubmitter) FetchOutput(ctx context.Context, block uint64) (*eth.OutputResponse, error) {
	rollupClient, err := l.RollupProvider.RollupClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting rollup client: %w", err)
	}

	output, err := rollupClient.OutputAtBlock(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("fetching output at block %d: %w", block, err)
	}
	if output.Version != supportedL2OutputVersion {
		return nil, fmt.Errorf("unsupported l2 output version: %v, supported: %v", output.Version, supportedL2OutputVersion)
	}
	if onum := output.BlockRef.Number; onum != block { // sanity check, e.g. in case of bad RPC caching
		return nil, fmt.Errorf("output block number %d mismatches requested %d", output.BlockRef.Number, block)
	}
	return output, nil
}

// ProposeL2OutputTxData creates the transaction data for the ProposeL2Output function
func (l *L2OutputSubmitter) ProposeL2OutputTxData(output *eth.OutputResponse) ([]byte, error) {
	return proposeL2OutputTxData(l.l2ooABI, output)
}

// proposeL2OutputTxData creates the transaction data for the ProposeL2Output function
func proposeL2OutputTxData(abi *abi.ABI, output *eth.OutputResponse) ([]byte, error) {
	return abi.Pack(
		"proposeL2Output",
		output.OutputRoot,
		new(big.Int).SetUint64(output.BlockRef.Number),
		output.Status.CurrentL1.Hash,
		new(big.Int).SetUint64(output.Status.CurrentL1.Number))
}

func (l *L2OutputSubmitter) ProposeL2OutputDGFTxCandidate(ctx context.Context, output *eth.OutputResponse) (txmgr.TxCandidate, error) {
	cCtx, cancel := context.WithTimeout(ctx, l.Cfg.NetworkTimeout)
	defer cancel()
	return l.dgfContract.ProposalTx(cCtx, l.Cfg.DisputeGameType, common.Hash(output.OutputRoot), output.BlockRef.Number)
}

// We wait until l1head advances beyond blocknum. This is used to make sure proposal tx won't
// immediately fail when checking the l1 blockhash. Note that EstimateGas uses "latest" state to
// execute the transaction by default, meaning inside the call, the head block is considered
// "pending" instead of committed. In the case l1blocknum == l1head then, blockhash(l1blocknum)
// will produce a value of 0 within EstimateGas, and the call will fail when the contract checks
// that l1blockhash matches blockhash(l1blocknum).
func (l *L2OutputSubmitter) waitForL1Head(ctx context.Context, blockNum uint64) error {
	ticker := time.NewTicker(l.Cfg.PollInterval)
	defer ticker.Stop()
	l1head, err := l.Txmgr.BlockNumber(ctx)
	if err != nil {
		return err
	}
	for l1head <= blockNum {
		l.Log.Debug("Waiting for l1 head > l1blocknum1+1", "l1head", l1head, "l1blocknum", blockNum)
		select {
		case <-ticker.C:
			l1head, err = l.Txmgr.BlockNumber(ctx)
			if err != nil {
				return err
			}
		case <-l.done:
			return fmt.Errorf("L2OutputSubmitter is done()")
		}
	}
	return nil
}

// sendTransaction creates & sends transactions through the underlying transaction manager.
func (l *L2OutputSubmitter) sendTransaction(ctx context.Context, output *eth.OutputResponse) error {
	err := l.waitForL1Head(ctx, output.Status.HeadL1.Number+1)
	if err != nil {
		return err
	}

	l.Log.Info("Proposing output root", "output", output.OutputRoot, "block", output.BlockRef)
	var receipt *types.Receipt
	if l.Cfg.DisputeGameFactoryAddr != nil {
		candidate, err := l.ProposeL2OutputDGFTxCandidate(ctx, output)
		if err != nil {
			return err
		}
		receipt, err = l.Txmgr.Send(ctx, candidate)
		if err != nil {
			return err
		}
	} else {
		data, err := l.ProposeL2OutputTxData(output)
		if err != nil {
			return err
		}
		receipt, err = l.Txmgr.Send(ctx, txmgr.TxCandidate{
			TxData:   data,
			To:       l.Cfg.L2OutputOracleAddr,
			GasLimit: 0,
		})
		if err != nil {
			return err
		}
	}

	if receipt.Status == types.ReceiptStatusFailed {
		l.Log.Error("Proposer tx successfully published but reverted", "tx_hash", receipt.TxHash)
	} else {
		l.Log.Info("Proposer tx successfully published",
			"tx_hash", receipt.TxHash,
			"l1blocknum", output.Status.CurrentL1.Number,
			"l1blockhash", output.Status.CurrentL1.Hash)
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
			// retry logic is needed around output fetching here.
			var output *eth.OutputResponse
			var shouldPropose bool
			var err error
			if l.dgfContract == nil {
				output, shouldPropose, err = l.FetchL2OOOutput(ctx)
			} else {
				output, shouldPropose, err = l.FetchDGFOutput(ctx)
			}
			if err != nil {
				l.Log.Warn("Error getting output", "err", err)
				continue
			} else if !shouldPropose {
				// debug logging already in Fetch(DGF|L2OO)Output
				continue
			}

			l.proposeOutput(ctx, output)
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

	rollupClient, err := l.RollupProvider.RollupClient(l.ctx)
	if err != nil {
		return fmt.Errorf("failed to get rollup client: %w", err)
	}

	return dial.WaitRollupSync(l.ctx, l.Log, rollupClient, l1head, time.Second*12)
}

func (l *L2OutputSubmitter) proposeOutput(ctx context.Context, output *eth.OutputResponse) {
	cCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	if err := l.sendTransaction(cCtx, output); err != nil {
		l.Log.Error("Failed to send proposal transaction",
			"err", err,
			"l1blocknum", output.Status.CurrentL1.Number,
			"l1blockhash", output.Status.CurrentL1.Hash,
			"l1head", output.Status.HeadL1.Number)
		return
	}
	l.Metr.RecordL2BlocksProposed(output.BlockRef)
}
