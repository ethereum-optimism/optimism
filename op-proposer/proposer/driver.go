package proposer

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	_ "net/http/pprof"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-proposer/bindings"
	"github.com/ethereum-optimism/optimism/op-proposer/metrics"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
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

type RollupClient interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
	OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error)
}

type DriverSetup struct {
	Log      log.Logger
	Metr     metrics.Metricer
	Cfg      ProposerConfig
	Txmgr    txmgr.TxManager
	L1Client L1Client

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

	l2ooContract *bindings.L2OutputOracleCaller
	l2ooABI      *abi.ABI

	dgfContract *bindings.DisputeGameFactoryCaller
	dgfABI      *abi.ABI
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
	dgfCaller, err := bindings.NewDisputeGameFactoryCaller(*setup.Cfg.DisputeGameFactoryAddr, setup.L1Client)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create DGF at address %s: %w", setup.Cfg.DisputeGameFactoryAddr, err)
	}

	cCtx, cCancel := context.WithTimeout(ctx, setup.Cfg.NetworkTimeout)
	defer cCancel()
	version, err := dgfCaller.Version(&bind.CallOpts{Context: cCtx})
	if err != nil {
		cancel()
		return nil, err
	}
	log.Info("Connected to DisputeGameFactory", "address", setup.Cfg.DisputeGameFactoryAddr, "version", version)

	parsed, err := bindings.DisputeGameFactoryMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, err
	}

	return &L2OutputSubmitter{
		DriverSetup: setup,
		done:        make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,

		dgfContract: dgfCaller,
		dgfABI:      parsed,
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

// FetchNextOutputInfo gets the block number of the next proposal.
// It returns: the next block number, if the proposal should be made, error
func (l *L2OutputSubmitter) FetchNextOutputInfo(ctx context.Context) (*eth.OutputResponse, bool, error) {
	if l.l2ooContract == nil {
		return nil, false, fmt.Errorf("L2OutputOracle contract not set, cannot fetch next output info")
	}

	cCtx, cancel := context.WithTimeout(ctx, l.Cfg.NetworkTimeout)
	defer cancel()
	callOpts := &bind.CallOpts{
		From:    l.Txmgr.From(),
		Context: cCtx,
	}
	nextCheckpointBlock, err := l.l2ooContract.NextBlockNumber(callOpts)
	if err != nil {
		l.Log.Error("proposer unable to get next block number", "err", err)
		return nil, false, err
	}
	// Fetch the current L2 heads
	currentBlockNumber, err := l.FetchCurrentBlockNumber(ctx)
	if err != nil {
		return nil, false, err
	}

	// Ensure that we do not submit a block in the future
	if currentBlockNumber.Cmp(nextCheckpointBlock) < 0 {
		l.Log.Debug("proposer submission interval has not elapsed", "currentBlockNumber", currentBlockNumber, "nextBlockNumber", nextCheckpointBlock)
		return nil, false, nil
	}

	return l.FetchOutput(ctx, nextCheckpointBlock)
}

// FetchCurrentBlockNumber gets the current block number from the [L2OutputSubmitter]'s [RollupClient]. If the `AllowNonFinalized` configuration
// option is set, it will return the safe head block number, and if not, it will return the finalized head block number.
func (l *L2OutputSubmitter) FetchCurrentBlockNumber(ctx context.Context) (*big.Int, error) {
	rollupClient, err := l.RollupProvider.RollupClient(ctx)
	if err != nil {
		l.Log.Error("proposer unable to get rollup client", "err", err)
		return nil, err
	}

	cCtx, cancel := context.WithTimeout(ctx, l.Cfg.NetworkTimeout)
	defer cancel()

	status, err := rollupClient.SyncStatus(cCtx)
	if err != nil {
		l.Log.Error("proposer unable to get sync status", "err", err)
		return nil, err
	}

	// Use either the finalized or safe head depending on the config. Finalized head is default & safer.
	var currentBlockNumber *big.Int
	if l.Cfg.AllowNonFinalized {
		currentBlockNumber = new(big.Int).SetUint64(status.SafeL2.Number)
	} else {
		currentBlockNumber = new(big.Int).SetUint64(status.FinalizedL2.Number)
	}
	return currentBlockNumber, nil
}

func (l *L2OutputSubmitter) FetchOutput(ctx context.Context, block *big.Int) (*eth.OutputResponse, bool, error) {
	rollupClient, err := l.RollupProvider.RollupClient(ctx)
	if err != nil {
		l.Log.Error("proposer unable to get rollup client", "err", err)
		return nil, false, err
	}

	cCtx, cancel := context.WithTimeout(ctx, l.Cfg.NetworkTimeout)
	defer cancel()

	output, err := rollupClient.OutputAtBlock(cCtx, block.Uint64())
	if err != nil {
		l.Log.Error("failed to fetch output at block", "block", block, "err", err)
		return nil, false, err
	}
	if output.Version != supportedL2OutputVersion {
		l.Log.Error("unsupported l2 output version", "output_version", output.Version, "supported_version", supportedL2OutputVersion)
		return nil, false, errors.New("unsupported l2 output version")
	}
	if output.BlockRef.Number != block.Uint64() { // sanity check, e.g. in case of bad RPC caching
		l.Log.Error("invalid blockNumber", "next_block", block, "output_block", output.BlockRef.Number)
		return nil, false, errors.New("invalid blockNumber")
	}

	// Always propose if it's part of the Finalized L2 chain. Or if allowed, if it's part of the safe L2 chain.
	if output.BlockRef.Number > output.Status.FinalizedL2.Number && (!l.Cfg.AllowNonFinalized || output.BlockRef.Number > output.Status.SafeL2.Number) {
		l.Log.Debug("not proposing yet, L2 block is not ready for proposal",
			"l2_proposal", output.BlockRef,
			"l2_safe", output.Status.SafeL2,
			"l2_finalized", output.Status.FinalizedL2,
			"allow_non_finalized", l.Cfg.AllowNonFinalized)
		return nil, false, nil
	}
	return output, true, nil
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

func (l *L2OutputSubmitter) ProposeL2OutputDGFTxData(output *eth.OutputResponse) ([]byte, *big.Int, error) {
	bond, err := l.dgfContract.InitBonds(&bind.CallOpts{}, l.Cfg.DisputeGameType)
	if err != nil {
		return nil, nil, err
	}
	data, err := proposeL2OutputDGFTxData(l.dgfABI, l.Cfg.DisputeGameType, output)
	if err != nil {
		return nil, nil, err
	}
	return data, bond, err
}

// proposeL2OutputDGFTxData creates the transaction data for the DisputeGameFactory's `create` function
func proposeL2OutputDGFTxData(abi *abi.ABI, gameType uint32, output *eth.OutputResponse) ([]byte, error) {
	return abi.Pack("create", gameType, output.OutputRoot, math.U256Bytes(new(big.Int).SetUint64(output.BlockRef.Number)))
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
		l.Log.Debug("waiting for l1 head > l1blocknum1+1", "l1head", l1head, "l1blocknum", blockNum)
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

	var receipt *types.Receipt
	if l.Cfg.DisputeGameFactoryAddr != nil {
		data, bond, err := l.ProposeL2OutputDGFTxData(output)
		if err != nil {
			return err
		}
		receipt, err = l.Txmgr.Send(ctx, txmgr.TxCandidate{
			TxData:   data,
			To:       l.Cfg.DisputeGameFactoryAddr,
			GasLimit: 0,
			Value:    bond,
		})
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
		l.Log.Error("proposer tx successfully published but reverted", "tx_hash", receipt.TxHash)
	} else {
		l.Log.Info("proposer tx successfully published",
			"tx_hash", receipt.TxHash,
			"l1blocknum", output.Status.CurrentL1.Number,
			"l1blockhash", output.Status.CurrentL1.Hash)
	}
	return nil
}

// loop is responsible for creating & submitting the next outputs
func (l *L2OutputSubmitter) loop() {
	defer l.wg.Done()
	ctx := l.ctx

	if l.Cfg.WaitNodeSync {
		err := l.waitNodeSync()
		if err != nil {
			l.Log.Error("Error waiting for node sync", "err", err)
			return
		}
	}

	if l.dgfContract == nil {
		l.loopL2OO(ctx)
	} else {
		l.loopDGF(ctx)
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

func (l *L2OutputSubmitter) loopL2OO(ctx context.Context) {
	ticker := time.NewTicker(l.Cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			output, shouldPropose, err := l.FetchNextOutputInfo(ctx)
			if err != nil || !shouldPropose {
				break
			}

			l.proposeOutput(ctx, output)
		case <-l.done:
			return
		}
	}
}

func (l *L2OutputSubmitter) loopDGF(ctx context.Context) {
	ticker := time.NewTicker(l.Cfg.ProposalInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			blockNumber, err := l.FetchCurrentBlockNumber(ctx)
			if err != nil {
				break
			}

			output, shouldPropose, err := l.FetchOutput(ctx, blockNumber)
			if err != nil || !shouldPropose {
				break
			}

			l.proposeOutput(ctx, output)
		case <-l.done:
			return
		}
	}
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
