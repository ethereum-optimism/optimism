package l2output

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-node/sources"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-node/eth"
)

var bigOne = big.NewInt(1)
var supportedL2OutputVersion = eth.Bytes32{}

type Config struct {
	Log  log.Logger
	Name string

	// L1Client is used to submit transactions to
	L1Client *ethclient.Client
	// RollupClient is used to retrieve output roots from
	RollupClient *sources.RollupClient

	// AllowNonFinalized enables the proposal of safe, but non-finalized L2 blocks.
	// The L1 block-hash embedded in the proposal TX is checked and should ensure the proposal
	// is never valid on an alternative L1 chain that would produce different L2 data.
	// This option is not necessary when higher proposal latency is acceptable and L1 is healthy.
	AllowNonFinalized bool

	// L2OOAddr is the L1 contract address of the L2 Output Oracle.
	L2OOAddr common.Address

	// ChainID is the L1 chain ID used for proposal transaction signing
	ChainID *big.Int

	// Privkey used for proposal transaction signing
	PrivKey *ecdsa.PrivateKey
}

type Driver struct {
	cfg             Config
	l2ooContract    *bindings.L2OutputOracle
	rawL2ooContract *bind.BoundContract
	walletAddr      common.Address
	l               log.Logger
}

func NewDriver(cfg Config) (*Driver, error) {
	l2ooContract, err := bindings.NewL2OutputOracle(cfg.L2OOAddr, cfg.L1Client)
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
	cfg.Log.Info("Configured driver", "wallet", walletAddr, "l2-output-contract", cfg.L2OOAddr)

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
func (d *Driver) GetBlockRange(ctx context.Context) (*big.Int, *big.Int, error) {
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
	status, err := d.cfg.RollupClient.SyncStatus(ctx)
	if err != nil {
		d.l.Error(name+" unable to get sync status", "err", err)
		return nil, nil, err
	}
	var currentBlockNumber *big.Int
	if d.cfg.AllowNonFinalized {
		currentBlockNumber = new(big.Int).SetUint64(status.SafeL2.Number)
	} else {
		currentBlockNumber = new(big.Int).SetUint64(status.FinalizedL2.Number)
	}

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
func (d *Driver) CraftTx(ctx context.Context, start, end, nonce *big.Int) (*types.Transaction, error) {
	name := d.cfg.Name

	d.l.Info(name+" crafting checkpoint tx", "start", start, "end", end, "nonce", nonce)

	// Fetch the final block in the range, as this is the only L2 output we need to submit.
	nextCheckpointBlock := new(big.Int).Sub(end, bigOne).Uint64()

	output, err := d.cfg.RollupClient.OutputAtBlock(ctx, nextCheckpointBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch output at block %d: %w", nextCheckpointBlock, err)
	}
	if output.Version != supportedL2OutputVersion {
		return nil, fmt.Errorf("unsupported l2 output version: %s", output.Version)
	}
	if output.BlockRef.Number != nextCheckpointBlock { // sanity check, e.g. in case of bad RPC caching
		return nil, fmt.Errorf("invalid blockNumber: next blockNumber is %v, blockNumber of block is %v", nextCheckpointBlock, output.BlockRef.Number)
	}

	// Always propose if it's part of the Finalized L2 chain. Or if allowed, if it's part of the safe L2 chain.
	if !(output.BlockRef.Number <= output.Status.FinalizedL2.Number || (d.cfg.AllowNonFinalized && output.BlockRef.Number <= output.Status.SafeL2.Number)) {
		d.l.Debug("not proposing yet, L2 block is not ready for proposal",
			"l2_proposal", output.BlockRef,
			"l2_safe", output.Status.SafeL2,
			"l2_finalized", output.Status.FinalizedL2,
			"allow_non_finalized", d.cfg.AllowNonFinalized)
		return nil, fmt.Errorf("output for L2 block %s is still unsafe", output.BlockRef)
	}

	opts, err := bind.NewKeyedTransactorWithChainID(d.cfg.PrivKey, d.cfg.ChainID)
	if err != nil {
		return nil, err
	}
	opts.Context = ctx
	opts.Nonce = nonce
	opts.NoSend = true

	// Note: the CurrentL1 is up to (and incl.) what the safe chain and finalized chain have been derived from,
	// and should be a quite recent L1 block (depends on L1 conf distance applied to rollup node).

	tx, err := d.l2ooContract.ProposeL2Output(
		opts,
		output.OutputRoot,
		new(big.Int).SetUint64(output.BlockRef.Number),
		output.Status.CurrentL1.Hash,
		new(big.Int).SetUint64(output.Status.CurrentL1.Number))
	if err != nil {
		return nil, err
	}

	numElements := new(big.Int).Sub(start, end).Uint64()
	d.l.Info(name+" proposal constructed",
		"start", start, "end", end,
		"nonce", nonce, "blocks_committed", numElements,
		"tx_hash", tx.Hash(),
		"output_version", output.Version,
		"output_root", output.OutputRoot,
		"output_block", output.BlockRef,
		"output_withdrawals_root", output.WithdrawalStorageRoot,
		"output_state_root", output.StateRoot,
		"current_l1", output.Status.CurrentL1,
		"safe_l2", output.Status.SafeL2,
		"finalized_l2", output.Status.FinalizedL2,
	)
	return tx, nil
}

// UpdateGasPrice signs an otherwise identical txn to the one provided but with
// updated gas prices sampled from the existing network conditions.
//
// NOTE: This method SHOULD NOT publish the resulting transaction.
func (d *Driver) UpdateGasPrice(ctx context.Context, tx *types.Transaction) (*types.Transaction, error) {
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

// SendTransaction injects a signed transaction into the pending pool for execution.
func (d *Driver) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	d.l.Info(d.cfg.Name+" sending transaction", "tx", tx.Hash())
	return d.cfg.L1Client.SendTransaction(ctx, tx)
}
