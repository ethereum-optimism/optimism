package sequencer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

	"github.com/ethereum-optimism/optimism/op-batcher/db"
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
	Log  log.Logger
	Name string

	// API to submit txs to
	L1Client *ethclient.Client

	// API to hit for batch data
	RollupClient *rollupclient.RollupClient

	// Limit the size of txs
	MinL1TxSize uint64
	MaxL1TxSize uint64

	// Limit the amounts of blocks per channel
	MaxBlocksPerChannel uint64

	// Where to send the batch txs to.
	BatchInboxAddress common.Address

	// Persists progress of submitting block data, to avoid redoing any work
	HistoryDB db.HistoryDatabase

	// The batcher can decide to set it shorter than the actual timeout,
	//  since submitting continued channel data to L1 is not instantaneous.
	//  It's not worth it to work with nearly timed-out channels.
	ChannelTimeout uint64

	// Chain ID of the L1 chain to submit txs to.
	ChainID *big.Int

	// Private key to sign batch txs with
	PrivKey *ecdsa.PrivateKey
}

type Driver struct {
	cfg        Config
	walletAddr common.Address
	l          log.Logger

	currentResp *derive.BatcherChannelData
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

	// Clear prior best attempt at building data to output, if any.
	d.currentResp = nil

	history, err := d.cfg.HistoryDB.LoadHistory()
	if err != nil {
		return nil, nil, err
	}

	// prune the in-memory history copy, to see if we can drop stuck channels,
	// and resubmit involved L2 blocks if necessary
	history.Update(nil, d.cfg.ChannelTimeout, uint64(time.Now().Unix()))

	d.l.Info("Fetching batch data",
		"num_history", len(history.Channels),
		"min_tx_size", d.cfg.MinL1TxSize,
		"max_tx_size", d.cfg.MaxL1TxSize,
		"max_blocks_per_channel", d.cfg.MaxBlocksPerChannel)

	// TODO: the API needs to be less stateful. If we request to continue from an older frame number, we should be able to do so.
	// Since we sometimes drop the API response, and restart with the previous history.
	// E.g. if profitability is not enough, if there is a crash, etc.
	// Currently we rely on channel timeouts to get rid of the failed channel in such situation, to then start fresh again.
	resp, err := d.cfg.RollupClient.GetBatchBundle(
		ctx,
		&node.BatchBundleRequest{
			History:             history.Channels,
			MinSize:             hexutil.Uint64(d.cfg.MinL1TxSize),
			MaxSize:             hexutil.Uint64(d.cfg.MaxL1TxSize),
			MaxBlocksPerChannel: hexutil.Uint64(d.cfg.MaxBlocksPerChannel),
		},
	)
	if err != nil {
		return nil, nil, err
	}
	d.l.Info("Fetched batch data", "data_size", len(resp.Data), "channels", len(resp.Channels),
		"opened_blocks", resp.Meta.OpenedBlocks, "closed_blocks", resp.Meta.ClosedBlocks,
		"safe_head", resp.Meta.SafeHead, "unsafe_head", resp.Meta.UnsafeHead)

	// The main loop is robust / dumb: if we have not reached the unsafe head,
	// then we need to continue submitting txs.
	start := new(big.Int).SetUint64(resp.Meta.SafeHead.Number)
	end := new(big.Int).SetUint64(resp.Meta.UnsafeHead.Number)

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
		Data:      d.currentResp.Data,
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
	// Persist that we are working on certain channels
	if err := d.cfg.HistoryDB.Update(d.currentResp.Channels, d.cfg.ChannelTimeout, uint64(time.Now().Unix())); err != nil {
		return fmt.Errorf("failed to update history db: %v", err)
	}

	return d.cfg.L1Client.SendTransaction(ctx, tx)
}
