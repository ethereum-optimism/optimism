package sequencer

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/go/batch-submitter/bindings/ctc"
	"github.com/ethereum-optimism/optimism/go/batch-submitter/metrics"
	l2types "github.com/ethereum-optimism/optimism/l2geth/core/types"
	l2ethclient "github.com/ethereum-optimism/optimism/l2geth/ethclient"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const (
	appendSequencerBatchMethodName = "appendSequencerBatch"
)

var bigOne = new(big.Int).SetUint64(1)

type Config struct {
	Name        string
	L1Client    *ethclient.Client
	L2Client    *l2ethclient.Client
	BlockOffset uint64
	MaxTxSize   uint64
	CTCAddr     common.Address
	ChainID     *big.Int
	PrivKey     *ecdsa.PrivateKey
}

type Driver struct {
	cfg            Config
	ctcContract    *ctc.CanonicalTransactionChain
	rawCtcContract *bind.BoundContract
	walletAddr     common.Address
	ctcABI         *abi.ABI
	metrics        *metrics.Metrics
}

func NewDriver(cfg Config) (*Driver, error) {
	ctcContract, err := ctc.NewCanonicalTransactionChain(
		cfg.CTCAddr, cfg.L1Client,
	)
	if err != nil {
		return nil, err
	}

	parsed, err := abi.JSON(strings.NewReader(
		ctc.CanonicalTransactionChainABI,
	))
	if err != nil {
		return nil, err
	}

	ctcABI, err := ctc.CanonicalTransactionChainMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	rawCtcContract := bind.NewBoundContract(
		cfg.CTCAddr, parsed, cfg.L1Client, cfg.L1Client,
		cfg.L1Client,
	)

	walletAddr := crypto.PubkeyToAddress(cfg.PrivKey.PublicKey)

	return &Driver{
		cfg:            cfg,
		ctcContract:    ctcContract,
		rawCtcContract: rawCtcContract,
		walletAddr:     walletAddr,
		ctcABI:         ctcABI,
		metrics:        metrics.NewMetrics(cfg.Name),
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

	start, err := d.ctcContract.GetTotalElements(&bind.CallOpts{
		Pending: false,
		Context: ctx,
	})
	if err != nil {
		return nil, nil, err
	}
	start.Add(start, blockOffset)

	latestHeader, err := d.cfg.L2Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, err
	}

	// Add one because end is *exclusive*.
	end := new(big.Int).Add(latestHeader.Number, bigOne)

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

	name := d.cfg.Name

	log.Info(name+" submitting batch tx", "start", start, "end", end,
		"gasPrice", gasPrice)

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

	var batchElements = make([]BatchElement, 0, len(blocks))
	for _, block := range blocks {
		batchElements = append(batchElements, BatchElementFromBlock(block))
	}

	shouldStartAt := start.Uint64()
	batchParams, err := GenSequencerBatchParams(
		shouldStartAt, d.cfg.BlockOffset, batchElements,
	)
	if err != nil {
		return nil, err
	}

	log.Info(name+" batch params", "params", fmt.Sprintf("%#v", batchParams))

	batchArguments, err := batchParams.Serialize()
	if err != nil {
		return nil, err
	}

	appendSequencerBatchID := d.ctcABI.Methods[appendSequencerBatchMethodName].ID
	batchCallData := append(appendSequencerBatchID, batchArguments...)

	if uint64(len(batchCallData)) > d.cfg.MaxTxSize {
		panic("call data too large")
	}

	// Record the batch_tx_build_time.
	batchTxBuildTime := float64(time.Since(batchTxBuildStart) / time.Millisecond)
	d.metrics.BatchTxBuildTime.Set(batchTxBuildTime)
	d.metrics.NumTxPerBatch.Observe(float64(len(blocks)))

	log.Info(name+" batch call data", "data", hex.EncodeToString(batchCallData))

	opts, err := bind.NewKeyedTransactorWithChainID(
		d.cfg.PrivKey, d.cfg.ChainID,
	)
	if err != nil {
		return nil, err
	}
	opts.Nonce = nonce
	opts.Context = ctx
	opts.GasPrice = gasPrice

	return d.rawCtcContract.RawTransact(opts, batchCallData)
}
