package l2output

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/l2"
	"github.com/ethereum-optimism/optimism/op-proposer/rollupclient"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

var bigOne = big.NewInt(1)
var supportedL2OutputVersion = l2.Bytes32{}

type Config struct {
	Log          log.Logger
	Name         string
	L1Client     *ethclient.Client
	L2Client     *ethclient.Client
	RollupClient *rollupclient.RollupClient
	L2OOAddr     common.Address
	ChainID      *big.Int
	PrivKey      *ecdsa.PrivateKey
}

type Driver struct {
	cfg             Config
	l2ooContract    *bindings.L2OutputOracle
	rawL2ooContract *bind.BoundContract
	walletAddr      common.Address
	l               log.Logger
}

func NewDriver(cfg Config) (*Driver, error) {
	l2ooContract, err := bindings.NewL2OutputOracle(
		cfg.L2OOAddr, cfg.L1Client,
	)
	if err != nil {
		return nil, err
	}

	parsed, err := abi.JSON(strings.NewReader(
		bindings.L2OutputOracleMetaData.ABI,
	))
	if err != nil {
		return nil, err
	}

	rawL2ooContract := bind.NewBoundContract(
		cfg.L2OOAddr, parsed, cfg.L1Client, cfg.L1Client, cfg.L1Client,
	)

	walletAddr := crypto.PubkeyToAddress(cfg.PrivKey.PublicKey)
	log.Info("Configured driver", "wallet", walletAddr, "l2-output-contract", cfg.L2OOAddr)

	return &Driver{
		cfg:             cfg,
		l2ooContract:    l2ooContract,
		rawL2ooContract: rawL2ooContract,
		walletAddr:      walletAddr,
		l:               cfg.Log,
	}, nil
}

// Name is an identifier used to prefix logs for a particular service.
func (d *Driver) Name() string {
	return d.cfg.Name
}

// WalletAddr is the wallet address used to pay for transaction fees.
func (d *Driver) WalletAddr() common.Address {
	return d.walletAddr
}

// GetBlockRange returns the start and end L2 block heights that need to be
// processed. Note that the end value is *exclusive*, therefore if the returned
// values are identical nothing needs to be processed.
func (d *Driver) GetBlockRange(
	ctx context.Context) (*big.Int, *big.Int, error) {

	name := d.cfg.Name

	callOpts := &bind.CallOpts{
		Pending: false,
		Context: ctx,
	}

	// Determine the next uncommitted L2 block number. We do so by transforming
	// the timestamp of the latest committed L2 block into its block number and
	// adding one.
	l2ooTimestamp, err := d.l2ooContract.LatestBlockTimestamp(callOpts)
	if err != nil {
		d.l.Error(name+" unable to get latest block timestamp", "err", err)
		return nil, nil, err
	}
	start, err := d.l2ooContract.ComputeL2BlockNumber(callOpts, l2ooTimestamp)
	if err != nil {
		d.l.Error(name+" unable to compute latest l2 block number", "err", err)
		return nil, nil, err
	}
	start.Add(start, bigOne)

	// Next we need to obtain the current timestamp and the next timestamp at
	// which we will need to submit an L2 output. The former is done by simply
	// adding the submission interval to the latest committed block's timestamp;
	// the latter inspects the timestamp of the latest block.
	nextTimestamp, err := d.l2ooContract.NextTimestamp(callOpts)
	if err != nil {
		d.l.Error(name+" unable to get next block timestamp", "err", err)
		return nil, nil, err
	}
	latestHeader, err := d.cfg.L1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		d.l.Error(name+" unable to retrieve latest header", "err", err)
		return nil, nil, err
	}
	currentTimestamp := big.NewInt(int64(latestHeader.Time))

	// If the submission window has yet to elapsed, we must wait before
	// submitting our L2 output commitment. Return start as the end value which
	// will signal that there is no work to be done.
	if currentTimestamp.Cmp(nextTimestamp) < 0 {
		d.l.Info(name+" submission interval has not elapsed",
			"currentTimestamp", currentTimestamp, "nextTimestamp", nextTimestamp)
		return start, start, nil
	}

	d.l.Info(name+" submission interval has elapsed",
		"currentTimestamp", currentTimestamp, "nextTimestamp", nextTimestamp)

	// Otherwise the submission interval has elapsed. Transform the next
	// expected timestamp into its L2 block number, and add one since end is
	// exclusive.
	end, err := d.l2ooContract.ComputeL2BlockNumber(callOpts, nextTimestamp)
	if err != nil {
		d.l.Error(name+" unable to compute next l2 block number", "err", err)
		return nil, nil, err
	}
	end.Add(end, bigOne)

	return start, end, nil
}

// CraftTx transforms the L2 blocks between start and end into a transaction
// using the given nonce.
//
// NOTE: This method SHOULD NOT publish the resulting transaction.
func (d *Driver) CraftTx(
	ctx context.Context,
	start, end, nonce *big.Int,
) (*types.Transaction, error) {

	name := d.cfg.Name

	d.l.Info(name+" crafting checkpoint tx", "start", start, "end", end,
		"nonce", nonce)

	// Fetch the final block in the range, as this is the only L2 output we need
	// to submit.
	nextCheckpointBlock := new(big.Int).Sub(end, bigOne)

	l2OutputRoot, err := d.outputRootAtBlock(ctx, nextCheckpointBlock)
	if err != nil {
		return nil, err
	}

	// Fetch the next expected timestamp that we will submit along with the
	// L2Output.
	callOpts := &bind.CallOpts{
		Pending: false,
		Context: ctx,
	}
	timestamp, err := d.l2ooContract.NextTimestamp(callOpts)
	if err != nil {
		return nil, err
	}

	// Sanity check that we are submitting against the same expected timestamp.
	expCheckpointBlock, err := d.l2ooContract.ComputeL2BlockNumber(
		callOpts, timestamp,
	)
	if err != nil {
		return nil, err
	}
	if nextCheckpointBlock.Cmp(expCheckpointBlock) != 0 {
		return nil, fmt.Errorf("expected next checkpoint block to be %d, "+
			"found %d", nextCheckpointBlock.Uint64(),
			expCheckpointBlock.Uint64())
	}

	numElements := new(big.Int).Sub(start, end).Uint64()
	d.l.Info(name+" checkpoint constructed", "start", start, "end", end,
		"nonce", nonce, "blocks_committed", numElements, "checkpoint_block", nextCheckpointBlock)

	header, err := d.cfg.L1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error resolving checkpoint block: %v", err)
	}

	l2Header, err := d.cfg.L2Client.HeaderByNumber(ctx, nextCheckpointBlock)
	if err != nil {
		return nil, fmt.Errorf("error resolving checkpoint block: %v", err)
	}

	if l2Header.Time != timestamp.Uint64() {
		return nil, fmt.Errorf("invalid timestamp: next timestamp is %v, timestamp of block is %v", timestamp, l2Header.Time)
	}

	opts, err := bind.NewKeyedTransactorWithChainID(
		d.cfg.PrivKey, d.cfg.ChainID,
	)
	if err != nil {
		return nil, err
	}
	opts.Context = ctx
	opts.Nonce = nonce
	opts.NoSend = true

	return d.l2ooContract.AppendL2Output(opts, l2OutputRoot, timestamp, header.Hash(), header.Number)
}

// UpdateGasPrice signs an otherwise identical txn to the one provided but with
// updated gas prices sampled from the existing network conditions.
//
// NOTE: Thie method SHOULD NOT publish the resulting transaction.
func (d *Driver) UpdateGasPrice(
	ctx context.Context,
	tx *types.Transaction,
) (*types.Transaction, error) {

	opts, err := bind.NewKeyedTransactorWithChainID(
		d.cfg.PrivKey, d.cfg.ChainID,
	)
	if err != nil {
		return nil, err
	}
	opts.Context = ctx
	opts.Nonce = new(big.Int).SetUint64(tx.Nonce())
	opts.NoSend = true

	return d.rawL2ooContract.RawTransact(opts, tx.Data())
}

// SendTransaction injects a signed transaction into the pending pool for
// execution.
func (d *Driver) SendTransaction(
	ctx context.Context,
	tx *types.Transaction,
) error {

	return d.cfg.L1Client.SendTransaction(ctx, tx)
}

func (d *Driver) outputRootAtBlock(ctx context.Context, blockNum *big.Int) (l2.Bytes32, error) {
	output, err := d.cfg.RollupClient.OutputAtBlock(ctx, blockNum)
	if err != nil {
		return l2.Bytes32{}, err
	}
	if len(output) != 2 {
		return l2.Bytes32{}, fmt.Errorf("invalid outputAtBlock response")
	}
	if version := output[0]; version != supportedL2OutputVersion {
		return l2.Bytes32{}, fmt.Errorf("unsupported l2 output version")
	}
	return output[1], nil
}
