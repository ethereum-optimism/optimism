package proposer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/go/batch-submitter/bindings/ctc"
	"github.com/ethereum-optimism/optimism/go/batch-submitter/bindings/scc"
	"github.com/ethereum-optimism/optimism/go/batch-submitter/metrics"
	l2types "github.com/ethereum-optimism/optimism/l2geth/core/types"
	l2ethclient "github.com/ethereum-optimism/optimism/l2geth/ethclient"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var bigOne = new(big.Int).SetUint64(1) //nolint:unused

type Config struct {
	Name        string
	L1Client    *ethclient.Client
	L2Client    *l2ethclient.Client
	BlockOffset uint64
	MaxTxSize   uint64
	SCCAddr     common.Address
	CTCAddr     common.Address
	ChainID     *big.Int
	PrivKey     *ecdsa.PrivateKey
}

type Driver struct {
	cfg         Config
	sccContract *scc.StateCommitmentChain
	ctcContract *ctc.CanonicalTransactionChain
	walletAddr  common.Address
	metrics     *metrics.Metrics
}

func NewDriver(cfg Config) (*Driver, error) {
	sccContract, err := scc.NewStateCommitmentChain(
		cfg.SCCAddr, cfg.L1Client,
	)
	if err != nil {
		return nil, err
	}

	ctcContract, err := ctc.NewCanonicalTransactionChain(
		cfg.CTCAddr, cfg.L1Client,
	)
	if err != nil {
		return nil, err
	}

	walletAddr := crypto.PubkeyToAddress(cfg.PrivKey.PublicKey)

	return &Driver{
		cfg:         cfg,
		sccContract: sccContract,
		ctcContract: ctcContract,
		walletAddr:  walletAddr,
		metrics:     metrics.NewMetrics(cfg.Name),
	}, nil
}

// Name is an identifier used to prefix logs for a particular service.
func (d *Driver) Name() string {
	return d.cfg.Name
}

// WalletAddr is the wallet address used to pay for batch transaction fees.
func (d *Driver) WalletAddr() common.Address {
	return d.walletAddr
}

// Metrics returns the subservice telemetry object.
func (d *Driver) Metrics() *metrics.Metrics {
	return d.metrics
}

// GetBatchBlockRange returns the start and end L2 block heights that need to be
// processed. Note that the end value is *exclusive*, therefore if the returned
// values are identical nothing needs to be processed.
func (d *Driver) GetBatchBlockRange(
	ctx context.Context) (*big.Int, *big.Int, error) {

	blockOffset := new(big.Int).SetUint64(d.cfg.BlockOffset)
	maxBatchSize := new(big.Int).SetUint64(1)

	start, err := d.sccContract.GetTotalElements(&bind.CallOpts{
		Pending: false,
		Context: ctx,
	})
	if err != nil {
		return nil, nil, err
	}
	start.Add(start, blockOffset)

	totalElements, err := d.ctcContract.GetTotalElements(&bind.CallOpts{
		Pending: false,
		Context: ctx,
	})
	if err != nil {
		return nil, nil, err
	}
	totalElements.Add(totalElements, blockOffset)

	// Take min(start + blockOffset + maxBatchSize, totalElements).
	end := new(big.Int).Add(start, maxBatchSize)
	if totalElements.Cmp(end) < 0 {
		end.Set(totalElements)
	}

	if start.Cmp(end) > 0 {
		return nil, nil, fmt.Errorf("invalid range, "+
			"end(%v) < start(%v)", end, start)
	}

	return start, end, nil
}

// SubmitBatchTx transforms the L2 blocks between start and end into a batch
// transaction using the given nonce and gasPrice. The final transaction is
// published and returned to the call.
func (d *Driver) SubmitBatchTx(
	ctx context.Context,
	start, end, nonce, gasPrice *big.Int) (*types.Transaction, error) {

	batchTxBuildStart := time.Now()

	var blocks []*l2types.Block
	for i := new(big.Int).Set(start); i.Cmp(end) < 0; i.Add(i, bigOne) {
		block, err := d.cfg.L2Client.BlockByNumber(ctx, i)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, block)

		// TODO(conner): remove when moving to multiple blocks
		break //nolint
	}

	var stateRoots = make([][32]byte, 0, len(blocks))
	for _, block := range blocks {
		stateRoots = append(stateRoots, block.Root())
	}

	batchTxBuildTime := float64(time.Since(batchTxBuildStart) / time.Millisecond)
	d.metrics.BatchTxBuildTime.Set(batchTxBuildTime)
	d.metrics.NumTxPerBatch.Observe(float64(len(blocks)))

	opts, err := bind.NewKeyedTransactorWithChainID(
		d.cfg.PrivKey, d.cfg.ChainID,
	)
	if err != nil {
		return nil, err
	}
	opts.Nonce = nonce
	opts.Context = ctx
	opts.GasPrice = gasPrice

	blockOffset := new(big.Int).SetUint64(d.cfg.BlockOffset)
	offsetStartsAtIndex := new(big.Int).Sub(start, blockOffset)

	return d.sccContract.AppendStateBatch(opts, stateRoots, offsetStartsAtIndex)
}
