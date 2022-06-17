package l2output

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/eth"
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
var supportedL2OutputVersion = eth.Bytes32{}

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

	// Determine the last committed L2 Block Number
	start, err := d.l2ooContract.LatestBlockNumber(callOpts)
	if err != nil {
		d.l.Error(name+" unable to get latest block number", "err", err)
		return nil, nil, err
	}
	start.Add(start, bigOne)

	// Next determine the L2 block that we need to commit
	nextBlockNumber, err := d.l2ooContract.NextBlockNumber(callOpts)
	if err != nil {
		d.l.Error(name+" unable to get next block number", "err", err)
		return nil, nil, err
	}
	latestHeader, err := d.cfg.L2Client.HeaderByNumber(ctx, nil)
	if err != nil {
		d.l.Error(name+" unable to retrieve latest header", "err", err)
		return nil, nil, err
	}
	currentBlockNumber := big.NewInt(latestHeader.Number.Int64())

	// If we do not have the new L2 Block number
	if currentBlockNumber.Cmp(nextBlockNumber) < 0 {
		d.l.Info(name+" submission interval has not elapsed",
			"currentBlockNumber", currentBlockNumber, "nextBlockNumber", nextBlockNumber)
		return start, start, nil
	}

	d.l.Info(name+" submission interval has elapsed",
		"currentBlockNumber", currentBlockNumber, "nextBlockNumber", nextBlockNumber)

	// Otherwise the submission interval has elapsed. Transform the next
	// expected timestamp into its L2 block number, and add one since end is
	// exclusive.
	end := new(big.Int).Add(nextBlockNumber, bigOne)

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

	numElements := new(big.Int).Sub(start, end).Uint64()
	d.l.Info(name+" checkpoint constructed", "start", start, "end", end,
		"nonce", nonce, "blocks_committed", numElements, "checkpoint_block", nextCheckpointBlock)

	l1Header, err := d.cfg.L1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error resolving checkpoint block: %v", err)
	}

	l2Header, err := d.cfg.L2Client.HeaderByNumber(ctx, nextCheckpointBlock)
	if err != nil {
		return nil, fmt.Errorf("error resolving checkpoint block: %v", err)
	}

	if l2Header.Number.Cmp(nextCheckpointBlock) != 0 {
		return nil, fmt.Errorf("invalid blockNumber: next blockNumber is %v, blockNumber of block is %v", nextCheckpointBlock, l2Header.Number)
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

	return d.l2ooContract.AppendL2Output(opts, l2OutputRoot, nextCheckpointBlock, l1Header.Hash(), l1Header.Number)
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

func (d *Driver) outputRootAtBlock(ctx context.Context, blockNum *big.Int) (eth.Bytes32, error) {
	output, err := d.cfg.RollupClient.OutputAtBlock(ctx, blockNum)
	if err != nil {
		return eth.Bytes32{}, err
	}
	if len(output) != 2 {
		return eth.Bytes32{}, fmt.Errorf("invalid outputAtBlock response")
	}
	if version := output[0]; version != supportedL2OutputVersion {
		return eth.Bytes32{}, fmt.Errorf("unsupported l2 output version")
	}
	return output[1], nil
}
