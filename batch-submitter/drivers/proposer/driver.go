package proposer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/batch-submitter/bindings/ctc"
	"github.com/ethereum-optimism/optimism/batch-submitter/bindings/scc"
	"github.com/ethereum-optimism/optimism/bss-core/drivers"
	"github.com/ethereum-optimism/optimism/bss-core/metrics"
	"github.com/ethereum-optimism/optimism/bss-core/txmgr"
	l2ethclient "github.com/ethereum-optimism/optimism/l2geth/ethclient"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// stateRootSize is the size in bytes of a state root.
const stateRootSize = 32

var bigOne = new(big.Int).SetUint64(1) //nolint:unused

type Config struct {
	Name                 string
	L1Client             *ethclient.Client
	L2Client             *l2ethclient.Client
	BlockOffset          uint64
	MaxStateRootElements uint64
	MinStateRootElements uint64
	SCCAddr              common.Address
	CTCAddr              common.Address
	ChainID              *big.Int
	PrivKey              *ecdsa.PrivateKey
}

type Driver struct {
	cfg            Config
	sccContract    *scc.StateCommitmentChain
	rawSccContract *bind.BoundContract
	ctcContract    *ctc.CanonicalTransactionChain
	walletAddr     common.Address
	metrics        *metrics.Base
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

	parsed, err := abi.JSON(strings.NewReader(
		scc.StateCommitmentChainABI,
	))
	if err != nil {
		return nil, err
	}

	rawSccContract := bind.NewBoundContract(
		cfg.SCCAddr, parsed, cfg.L1Client, cfg.L1Client, cfg.L1Client,
	)

	walletAddr := crypto.PubkeyToAddress(cfg.PrivKey.PublicKey)

	return &Driver{
		cfg:            cfg,
		sccContract:    sccContract,
		rawSccContract: rawSccContract,
		ctcContract:    ctcContract,
		walletAddr:     walletAddr,
		metrics:        metrics.NewBase("batch_submitter", cfg.Name),
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
func (d *Driver) Metrics() metrics.Metrics {
	return d.metrics
}

// ClearPendingTx a publishes a transaction at the next available nonce in order
// to clear any transactions in the mempool left over from a prior running
// instance of the batch submitter.
func (d *Driver) ClearPendingTx(
	ctx context.Context,
	txMgr txmgr.TxManager,
	l1Client *ethclient.Client,
) error {

	return drivers.ClearPendingTx(
		d.cfg.Name, ctx, txMgr, l1Client, d.walletAddr, d.cfg.PrivKey,
		d.cfg.ChainID,
	)
}

// GetBatchBlockRange returns the start and end L2 block heights that need to be
// processed. Note that the end value is *exclusive*, therefore if the returned
// values are identical nothing needs to be processed.
func (d *Driver) GetBatchBlockRange(
	ctx context.Context) (*big.Int, *big.Int, error) {

	blockOffset := new(big.Int).SetUint64(d.cfg.BlockOffset)

	start, err := d.sccContract.GetTotalElements(&bind.CallOpts{
		Pending: false,
		Context: ctx,
	})
	if err != nil {
		return nil, nil, err
	}
	start.Add(start, blockOffset)

	end, err := d.ctcContract.GetTotalElements(&bind.CallOpts{
		Pending: false,
		Context: ctx,
	})
	if err != nil {
		return nil, nil, err
	}
	end.Add(end, blockOffset)

	if start.Cmp(end) > 0 {
		return nil, nil, fmt.Errorf("invalid range, "+
			"end(%v) < start(%v)", end, start)
	}

	return start, end, nil
}

// CraftBatchTx transforms the L2 blocks between start and end into a batch
// transaction using the given nonce. A dummy gas price is used in the resulting
// transaction to use for size estimation.
//
// NOTE: This method SHOULD NOT publish the resulting transaction.
func (d *Driver) CraftBatchTx(
	ctx context.Context,
	start, end, nonce *big.Int,
) (*types.Transaction, error) {

	name := d.cfg.Name

	log.Info(name+" crafting batch tx", "start", start, "end", end,
		"nonce", nonce)

	var stateRoots [][stateRootSize]byte
	for i := new(big.Int).Set(start); i.Cmp(end) < 0; i.Add(i, bigOne) {
		// Consume state roots until reach our maximum tx size.
		if uint64(len(stateRoots)) > d.cfg.MaxStateRootElements {
			break
		}

		block, err := d.cfg.L2Client.BlockByNumber(ctx, i)
		if err != nil {
			return nil, err
		}

		stateRoots = append(stateRoots, block.Root())
	}

	// Abort if we don't have enough state roots to meet our minimum
	// requirement.
	if uint64(len(stateRoots)) < d.cfg.MinStateRootElements {
		log.Info(name+" number of state roots  below minimum",
			"num_state_roots", len(stateRoots),
			"min_state_roots", d.cfg.MinStateRootElements)
		return nil, nil
	}

	d.metrics.NumElementsPerBatch().Observe(float64(len(stateRoots)))

	log.Info(name+" batch constructed", "num_state_roots", len(stateRoots))

	opts, err := bind.NewKeyedTransactorWithChainID(
		d.cfg.PrivKey, d.cfg.ChainID,
	)
	if err != nil {
		return nil, err
	}
	opts.Context = ctx
	opts.Nonce = nonce
	opts.NoSend = true

	blockOffset := new(big.Int).SetUint64(d.cfg.BlockOffset)
	offsetStartsAtIndex := new(big.Int).Sub(start, blockOffset)

	tx, err := d.sccContract.AppendStateBatch(
		opts, stateRoots, offsetStartsAtIndex,
	)
	switch {
	case err == nil:
		return tx, nil

	// If the transaction failed because the backend does not support
	// eth_maxPriorityFeePerGas, fallback to using the default constant.
	// Currently Alchemy is the only backend provider that exposes this method,
	// so in the event their API is unreachable we can fallback to a degraded
	// mode of operation. This also applies to our test environments, as hardhat
	// doesn't support the query either.
	case drivers.IsMaxPriorityFeePerGasNotFoundError(err):
		log.Warn(d.cfg.Name + " eth_maxPriorityFeePerGas is unsupported " +
			"by current backend, using fallback gasTipCap")
		opts.GasTipCap = drivers.FallbackGasTipCap
		return d.sccContract.AppendStateBatch(
			opts, stateRoots, offsetStartsAtIndex,
		)

	default:
		return nil, err
	}
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

	finalTx, err := d.rawSccContract.RawTransact(opts, tx.Data())
	switch {
	case err == nil:
		return finalTx, nil

	// If the transaction failed because the backend does not support
	// eth_maxPriorityFeePerGas, fallback to using the default constant.
	// Currently Alchemy is the only backend provider that exposes this method,
	// so in the event their API is unreachable we can fallback to a degraded
	// mode of operation. This also applies to our test environments, as hardhat
	// doesn't support the query either.
	case drivers.IsMaxPriorityFeePerGasNotFoundError(err):
		log.Warn(d.cfg.Name + " eth_maxPriorityFeePerGas is unsupported " +
			"by current backend, using fallback gasTipCap")
		opts.GasTipCap = drivers.FallbackGasTipCap
		return d.rawSccContract.RawTransact(opts, tx.Data())

	default:
		return nil, err
	}
}

// SendTransaction injects a signed transaction into the pending pool for
// execution.
func (d *Driver) SendTransaction(
	ctx context.Context,
	tx *types.Transaction,
) error {
	return d.cfg.L1Client.SendTransaction(ctx, tx)
}
