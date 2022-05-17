package sequencer

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-batcher/db"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-proposer/rollupclient"
	"github.com/ethereum-optimism/optimism/op-proposer/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

type Config struct {
	Log               log.Logger
	Name              string
	L1Client          *ethclient.Client
	L2Client          *ethclient.Client
	RollupClient      *rollupclient.RollupClient
	MinL1TxSize       uint64
	MaxL1TxSize       uint64
	BatchInboxAddress common.Address
	HistoryDB         db.HistoryDatabase
	ChainID           *big.Int
	PrivKey           *ecdsa.PrivateKey
}

type Driver struct {
	cfg        Config
	walletAddr common.Address
	l          log.Logger

	currentBatch *node.BatchBundleResponse
}

func NewDriver(cfg Config) (*Driver, error) {
	walletAddr := crypto.PubkeyToAddress(cfg.PrivKey.PublicKey)

	return &Driver{
		cfg:        cfg,
		walletAddr: walletAddr,
		l:          cfg.Log,
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
	ctx context.Context,
) (*big.Int, *big.Int, error) {

	// Clear prior batch, if any.
	d.currentBatch = nil

	history, err := d.cfg.HistoryDB.LoadHistory()
	if err != nil {
		return nil, nil, err
	}

	latestBlockID := history.LatestID()
	ancestors := history.Ancestors()

	d.l.Info("Fetching bundle",
		"latest_number", latestBlockID.Number,
		"lastest_hash", latestBlockID.Hash,
		"num_ancestors", len(ancestors),
		"min_tx_size", d.cfg.MinL1TxSize,
		"max_tx_size", d.cfg.MaxL1TxSize)

	batchResp, err := d.cfg.RollupClient.GetBatchBundle(
		ctx,
		&node.BatchBundleRequest{
			L2History: ancestors,
			MinSize:   hexutil.Uint64(d.cfg.MinL1TxSize),
			MaxSize:   hexutil.Uint64(d.cfg.MaxL1TxSize),
		},
	)
	if err != nil {
		return nil, nil, err
	}

	// Bundle is not available yet, return the next expected block number.
	if batchResp == nil {
		start64 := latestBlockID.Number + 1
		start := big.NewInt(int64(start64))
		return start, start, nil
	}

	// There is nothing to be done if the rollup returns a last block hash equal
	// to the previous block hash. Return identical start and end block heights
	// to signal that there is no work to be done.
	start := big.NewInt(int64(batchResp.PrevL2BlockNum) + 1)
	if batchResp.LastL2BlockHash == batchResp.PrevL2BlockHash {
		return start, start, nil
	}

	if batchResp.PrevL2BlockHash != latestBlockID.Hash {
		d.l.Warn("Reorg", "rpc_prev_block_hash", batchResp.PrevL2BlockHash,
			"db_prev_block_hash", latestBlockID.Hash)
	}

	// If the bundle is empty, this implies that all blocks in the range were
	// empty blocks. Simply commit the new head and return that there is no work
	// to be done.
	if len(batchResp.Bundle) == 0 {
		err = d.cfg.HistoryDB.AppendEntry(eth.BlockID{
			Number: uint64(batchResp.LastL2BlockNum),
			Hash:   batchResp.LastL2BlockHash,
		})
		if err != nil {
			return nil, nil, err
		}

		next := big.NewInt(int64(batchResp.LastL2BlockNum + 1))
		return next, next, nil
	}

	d.currentBatch = batchResp
	end := big.NewInt(int64(batchResp.LastL2BlockNum + 1))

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

	gasTipCap, err := d.cfg.L1Client.SuggestGasTipCap(ctx)
	if err != nil {
		// TODO(conner): handle fallback
		return nil, err
	}

	head, err := d.cfg.L1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	gasFeeCap := txmgr.CalcGasFeeCap(head.BaseFee, gasTipCap)

	rawTx := &types.DynamicFeeTx{
		ChainID:   d.cfg.ChainID,
		Nonce:     nonce.Uint64(),
		To:        &d.cfg.BatchInboxAddress,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Data:      d.currentBatch.Bundle,
	}

	gas, err := core.IntrinsicGas(rawTx.Data, nil, false, true, true)
	if err != nil {
		return nil, err
	}
	rawTx.Gas = gas

	return types.SignNewTx(
		d.cfg.PrivKey, types.LatestSignerForChainID(d.cfg.ChainID), rawTx,
	)
}

// UpdateGasPrice signs an otherwise identical txn to the one provided but with
// updated gas prices sampled from the existing network conditions.
//
// NOTE: Thie method SHOULD NOT publish the resulting transaction.
func (d *Driver) UpdateGasPrice(
	ctx context.Context,
	tx *types.Transaction,
) (*types.Transaction, error) {

	gasTipCap, err := d.cfg.L1Client.SuggestGasTipCap(ctx)
	if err != nil {
		// TODO(conner): handle fallback
		return nil, err
	}

	head, err := d.cfg.L1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	gasFeeCap := txmgr.CalcGasFeeCap(head.BaseFee, gasTipCap)

	rawTx := &types.DynamicFeeTx{
		ChainID:   d.cfg.ChainID,
		Nonce:     tx.Nonce(),
		To:        tx.To(),
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       tx.Gas(),
		Data:      tx.Data(),
	}

	return types.SignNewTx(
		d.cfg.PrivKey, types.LatestSignerForChainID(d.cfg.ChainID), rawTx,
	)
}

// SendTransaction injects a signed transaction into the pending pool for
// execution.
func (d *Driver) SendTransaction(
	ctx context.Context,
	tx *types.Transaction,
) error {

	err := d.cfg.HistoryDB.AppendEntry(eth.BlockID{
		Number: uint64(d.currentBatch.LastL2BlockNum),
		Hash:   d.currentBatch.LastL2BlockHash,
	})
	if err != nil {
		return err
	}

	return d.cfg.L1Client.SendTransaction(ctx, tx)
}
