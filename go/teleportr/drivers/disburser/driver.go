package disburser

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/go/bss-core/metrics"
	"github.com/ethereum-optimism/optimism/go/bss-core/txmgr"
	"github.com/ethereum-optimism/optimism/go/teleportr/bindings/deposit"
	"github.com/ethereum-optimism/optimism/go/teleportr/bindings/disburse"
	"github.com/ethereum-optimism/optimism/go/teleportr/db"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

// DisbursementSuccessTopic is the topic hash for DisbursementSuccess events.
var DisbursementSuccessTopic = common.HexToHash(
	"0xeaa22fd2d7b875476355b32cf719794faf9d91b66e73bc6375a053cace9caaee",
)

// DisbursementFailedTopic is the topic hash for DisbursementFailed events.
var DisbursementFailedTopic = common.HexToHash(
	"0x9b478c095979d3d3a7d602ffd9ee1f0843204d853558ae0882c8fcc0a5bc78cf",
)

type Config struct {
	Name                 string
	L1Client             *ethclient.Client
	L2Client             *ethclient.Client
	Database             *db.Database
	MaxTxSize            uint64
	NumConfirmations     uint64
	DeployBlockNumber    uint64
	FilterQueryMaxBlocks uint64
	DepositAddr          common.Address
	DisburserAddr        common.Address
	ChainID              *big.Int
	PrivKey              *ecdsa.PrivateKey
}

type Driver struct {
	cfg                  Config
	depositContract      *deposit.TeleportrDeposit
	disburserContract    *disburse.TeleportrDisburser
	rawDisburserContract *bind.BoundContract
	walletAddr           common.Address
	metrics              *Metrics

	currentDepositIDs []uint64
}

func NewDriver(cfg Config) (*Driver, error) {
	if cfg.NumConfirmations == 0 {
		panic("NumConfirmations cannot be zero")
	}
	if cfg.FilterQueryMaxBlocks == 0 {
		panic("FilterQueryMaxBlocks cannot be zero")
	}

	depositContract, err := deposit.NewTeleportrDeposit(
		cfg.DepositAddr, cfg.L1Client,
	)
	if err != nil {
		return nil, err
	}

	disburserContract, err := disburse.NewTeleportrDisburser(
		cfg.DisburserAddr, cfg.L2Client,
	)
	if err != nil {
		return nil, err
	}

	parsed, err := abi.JSON(strings.NewReader(
		disburse.TeleportrDisburserMetaData.ABI,
	))
	if err != nil {
		return nil, err
	}

	rawDisburserContract := bind.NewBoundContract(
		cfg.DisburserAddr, parsed, cfg.L2Client, cfg.L2Client, cfg.L2Client,
	)

	walletAddr := crypto.PubkeyToAddress(cfg.PrivKey.PublicKey)

	return &Driver{
		cfg:                  cfg,
		depositContract:      depositContract,
		disburserContract:    disburserContract,
		rawDisburserContract: rawDisburserContract,
		walletAddr:           walletAddr,
		metrics:              NewMetrics(cfg.Name),
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
// instance. When publishing to L2 there is no mempool so the transaction can't
// get stuck, thus the behavior is unimplemented.
func (d *Driver) ClearPendingTx(
	ctx context.Context,
	txMgr txmgr.TxManager,
	l1Client *ethclient.Client,
) error {

	return nil
}

// GetBatchBlockRange returns the start and end L2 block heights that need to be
// processed. Note that the end value is *exclusive*, therefore if the returned
// values are identical nothing needs to be processed.
func (d *Driver) GetBatchBlockRange(
	ctx context.Context) (*big.Int, *big.Int, error) {

	// Update balance metrics on each iteration.
	d.updateBalanceMetrics(ctx)

	// Clear the current deposit IDs from any prior iteration.
	d.currentDepositIDs = nil

	// Before proceeding, process the outcomes of any transactions we've
	// published in the past. This handles both the restart case, as well as
	// post processing after a txn is published.
	err := d.processPendingTxs(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Next, load the last disbursement ID claimed by postgres and the contract.
	lastDisbursementID, err := d.latestDisbursementID()
	if err != nil {
		return nil, nil, err
	}
	if lastDisbursementID != nil {
		d.metrics.PostgresLastDisbursedID.Set(float64(*lastDisbursementID))
	}

	startID, err := d.disburserContract.TotalDisbursements(
		&bind.CallOpts{},
	)
	if err != nil {
		return nil, nil, err
	}
	startID64 := startID.Uint64()
	d.metrics.ContractNextDisbursementID.Set(float64(startID64))

	// Do a quick sanity check that that the database and contract are in sync.
	d.logDatabaseContractMismatch(lastDisbursementID, startID64)

	// Now, proceed to ingest any new deposits by inspect L1 events from the
	// deposit contract, using the last processed block in postgres as a lower
	// bound.
	blockNumber, err := d.cfg.L1Client.BlockNumber(ctx)
	if err != nil {
		return nil, nil, err
	}
	lastProcessedBlock, err := d.lastProcessedBlock()
	if err != nil {
		return nil, nil, err
	}

	err = d.ingestDeposits(ctx, blockNumber, lastProcessedBlock)
	if err != nil {
		return nil, nil, err
	}

	// After successfully ingesting deposits, check to see if there are any
	// now-confirmed deposits that we can attempt to disburse.
	confirmedDeposits, err := d.loadConfirmedDepositsInRange(
		blockNumber, startID64, math.MaxUint64,
	)
	if err != nil {
		return nil, nil, err
	}

	if len(confirmedDeposits) == 0 {
		return startID, startID, nil
	}

	// Compute the end fo the range as the last confirmed deposit plus one.
	endID64 := confirmedDeposits[len(confirmedDeposits)-1].ID + 1
	endID := new(big.Int).SetUint64(endID64)

	return startID, endID, nil
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

	blockNumber, err := d.cfg.L1Client.BlockNumber(ctx)
	if err != nil {
		return nil, err
	}

	confirmedDeposits, err := d.loadConfirmedDepositsInRange(
		blockNumber, start.Uint64(), end.Uint64(),
	)
	if err != nil {
		return nil, err
	}

	var disbursements []disburse.TeleportrDisburserDisbursement
	var depositIDs []uint64
	value := new(big.Int)
	for _, deposit := range confirmedDeposits {
		disbursement := disburse.TeleportrDisburserDisbursement{
			Amount: deposit.Amount,
			Addr:   deposit.Address,
		}
		disbursements = append(disbursements, disbursement)
		depositIDs = append(depositIDs, deposit.ID)
		value = value.Add(value, deposit.Amount)
	}

	log.Info(name+" crafting batch tx", "start", start, "end", end,
		"nonce", nonce)

	d.metrics.NumElementsPerBatch().Observe(float64(len(disbursements)))

	log.Info(name+" batch constructed", "num_disbursements", len(disbursements))

	gasPrice, err := d.cfg.L2Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	opts, err := bind.NewKeyedTransactorWithChainID(
		d.cfg.PrivKey, d.cfg.ChainID,
	)
	if err != nil {
		return nil, err
	}
	opts.Context = ctx
	opts.Nonce = nonce
	opts.GasPrice = gasPrice
	opts.NoSend = true
	opts.Value = value

	tx, err := d.disburserContract.Disburse(opts, start, disbursements)
	if err != nil {
		return nil, err
	}

	d.currentDepositIDs = depositIDs

	return tx, nil
}

// UpdateGasPrice signs an otherwise identical txn to the one provided but with
// updated gas prices sampled from the existing network conditions.
//
// NOTE: Thie method SHOULD NOT publish the resulting transaction.
func (d *Driver) UpdateGasPrice(
	ctx context.Context,
	tx *types.Transaction,
) (*types.Transaction, error) {

	gasPrice, err := d.cfg.L1Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	opts, err := bind.NewKeyedTransactorWithChainID(
		d.cfg.PrivKey, d.cfg.ChainID,
	)
	if err != nil {
		return nil, err
	}
	opts.Context = ctx
	opts.Nonce = new(big.Int).SetUint64(tx.Nonce())
	opts.GasPrice = gasPrice
	opts.Value = tx.Value()
	opts.NoSend = true

	return d.rawDisburserContract.RawTransact(opts, tx.Data())
}

// SendTransaction injects a signed transaction into the pending pool for
// execution.
func (d *Driver) SendTransaction(
	ctx context.Context,
	tx *types.Transaction,
) error {

	txHash := tx.Hash()
	startID := d.currentDepositIDs[0]
	endID := d.currentDepositIDs[len(d.currentDepositIDs)-1] + 1

	// Record the pending transaction hash so that we can recover if we crash
	// after publishing.
	err := d.upsertPendingTx(db.PendingTx{
		TxHash:  txHash,
		StartID: startID,
		EndID:   endID,
	})
	if err != nil {
		return err
	}

	return d.cfg.L2Client.SendTransaction(ctx, tx)
}

// processPendingTxs is a helper method which updates Postgres with the effects
// of a published disbursement tx. This handles both startup recovery as well as
// normal operation after a transaction is published.
func (d *Driver) processPendingTxs(ctx context.Context) error {
	pendingTxs, err := d.listPendingTxs()
	if err != nil {
		return err
	}

	// Nothing to do. This can happen on first startup, or if GetBatchBlockRange
	// was called before shutdown without sending a transaction.
	if len(pendingTxs) == 0 {
		return nil
	}

	// Fetch the receipt for the pending transaction that confirmed. In practice
	// there should only be one, but if there isn't we will return an error here
	// to process any others on subsequent calls.
	var receipt *types.Receipt
	var pendingTx db.PendingTx
	for _, pendingTx = range pendingTxs {
		r, err := d.cfg.L2Client.TransactionReceipt(ctx, pendingTx.TxHash)
		if err == ethereum.NotFound {
			continue
		} else if err != nil {
			return err
		}

		// Also skip any reverted transactions.
		if r.Status != 1 {
			continue
		}

		receipt = r
		break
	}

	// Backend is reporting not knowing any of the transactions, try again
	// later.
	if receipt == nil {
		return errors.New("unable to find receipt for any pending tx")
	}

	// Useing the block number, load the header so that we can get accurate
	// timestamps for the disbursements.
	header, err := d.cfg.L2Client.HeaderByNumber(ctx, receipt.BlockNumber)
	if err != nil {
		return err
	}
	blockTimestamp := time.Unix(int64(header.Time), 0)

	var successfulDisbursements int
	var failedDisbursements int
	var failedUpserts uint64
	for _, event := range receipt.Logs {
		// Extract the deposit ID from the second topic if this is a
		// success/fail event.
		var depositID uint64
		var success bool
		switch event.Topics[0] {
		case DisbursementSuccessTopic:
			depositID = new(big.Int).SetBytes(event.Topics[1][:]).Uint64()
			success = true
			successfulDisbursements++
		case DisbursementFailedTopic:
			depositID = new(big.Int).SetBytes(event.Topics[1][:]).Uint64()
			success = false
			failedDisbursements++
		default:
			continue
		}

		err = d.cfg.Database.UpsertDisbursement(
			depositID,
			receipt.TxHash,
			receipt.BlockNumber.Uint64(),
			blockTimestamp,
			success,
		)
		if err != nil {
			failedUpserts++
			log.Warn("Unable to mark disbursement success",
				"depositId", depositID,
				"txHash", receipt.TxHash,
				"blockNumber", receipt.BlockNumber,
				"blockTimestamp", blockTimestamp,
				"err", err)
			continue
		}

		log.Info("Disbursement marked success",
			"depositId", depositID,
			"txHash", receipt.TxHash,
			"blockNumber", receipt.BlockNumber,
			"blockTimestamp", blockTimestamp)
	}

	d.metrics.SuccessfulDisbursements.Add(float64(successfulDisbursements))
	d.metrics.FailedDisbursements.Add(float64(failedDisbursements))
	d.metrics.FailedDatabaseMethods.With(DBMethodUpsertDisbursement).
		Inc()

	// We have completed our post-processing once all of the disbursements are
	// written without failures.
	if failedUpserts > 0 {
		return errors.New("failed to upsert all disbursements successfully")
	}

	// If we upserted all disbursements successfully, remove any pending txs
	// with the same start/end id.
	err = d.deletePendingTx(pendingTx.StartID, pendingTx.EndID)
	if err != nil {
		return err
	}

	// Sanity check that this leaves our pending tx table empty.
	pendingTxs, err = d.listPendingTxs()
	if err != nil {
		return err
	}

	// If not, return an error so that subsequent calls to GetBatchBlockRange
	// will attempt to process other pending txs before continuing.
	if len(pendingTxs) != 0 {
		return errors.New("pending txs remain in database")
	}

	return nil
}

// ingestDeposits is a preprocessing step done each time we attempt to compute a
// new block range. This method scans for any missing or recent logs from the
// contract, and upserts any new deposits into the database.
func (d *Driver) ingestDeposits(
	ctx context.Context,
	blockNumber uint64,
	lastProcessedBlockNumber *uint64,
) error {

	filterStartBlockNumber := FindFilterStartBlockNumber(
		FilterStartBlockNumberParams{
			BlockNumber:              blockNumber,
			NumConfirmations:         d.cfg.NumConfirmations,
			DeployBlockNumber:        d.cfg.DeployBlockNumber,
			LastProcessedBlockNumber: lastProcessedBlockNumber,
		},
	)

	// Fetch deposit events in block ranges capped by the FilterQueryMaxBlocks
	// parameter.
	maxBlocks := d.cfg.FilterQueryMaxBlocks
	for start := filterStartBlockNumber; start < blockNumber+1; start += maxBlocks {
		var end = start + maxBlocks
		if end > blockNumber {
			end = blockNumber
		}

		opts := &bind.FilterOpts{
			Start:   start,
			End:     &end,
			Context: ctx,
		}
		events, err := d.depositContract.FilterEtherReceived(opts, nil, nil, nil)
		if err != nil {
			return err
		}
		defer events.Close()

		var deposits []db.Deposit
		for events.Next() {
			event := events.Event

			header, err := d.cfg.L1Client.HeaderByNumber(
				ctx, big.NewInt(int64(event.Raw.BlockNumber)),
			)
			if err != nil {
				return err
			}

			deposits = append(deposits, db.Deposit{
				ID:      event.DepositId.Uint64(),
				Address: event.Emitter,
				Amount:  event.Amount,
				ConfirmationInfo: db.ConfirmationInfo{
					TxnHash:        event.Raw.TxHash,
					BlockNumber:    event.Raw.BlockNumber,
					BlockTimestamp: time.Unix(int64(header.Time), 0),
				},
			})
		}
		err = events.Error()
		if err != nil {
			return err
		}

		err = d.upsertDeposits(deposits, end)
		if err != nil {
			return err
		}
	}

	return nil
}

// loadConfirmedDeposits retrieves the list of confirmed deposits with IDs
// in the range [startID, endID).
func (d *Driver) loadConfirmedDepositsInRange(
	blockNumber uint64,
	startID uint64,
	endID uint64,
) ([]db.Deposit, error) {

	confirmedDeposits, err := d.confirmedDeposits(blockNumber)
	if err != nil {
		return nil, err
	}

	// On the off chance that we failed to record a disbursement, filter out any
	// which are lower than what the disbursement contract says is the next
	// disbursement. Note that it is possible for the last disbursement id to
	// match the contract, but still have lingering, incomplete disbursements
	// before that.
	var filteredDeposits = make([]db.Deposit, 0, len(confirmedDeposits))
	var missingDisbursements int
	for _, deposit := range confirmedDeposits {
		switch {

		// If the deposit ID is less than our start ID, this indicates that we
		// are missing a disbursement for this deposit even though the contract
		// beleves it was disbursed.
		case deposit.ID < startID:
			log.Warn("Filtering deposit with missing disbursement",
				"deposit_id", deposit.ID)
			missingDisbursements++
			continue

		// This is mostly a defensive measure, to ensure that we return the
		// exact range that was sanity checked in GetBatchBlockRange, which can
		// change as a result of the block number increasing.
		case deposit.ID >= endID:
			continue
		}

		filteredDeposits = append(filteredDeposits, deposit)
	}
	d.metrics.MissingDisbursements.Set(float64(missingDisbursements))

	if len(filteredDeposits) == 0 {
		return nil, nil
	}

	// Ensure the next confirmed deposit matches what the contract expects.
	if startID != filteredDeposits[0].ID {
		panic("confirmed deposits start is not contiguous")
	}

	// Ensure that the slice has contiguous deposit ids. This is done by
	// checking that the final deposit id is equal to:
	//   start + len(confirmedDeposits).
	// The id is the primary key so there cannot be duplicates, and they are
	// returned in sorted order.
	lastDepositID := filteredDeposits[len(filteredDeposits)-1].ID
	if startID+uint64(len(filteredDeposits)) != lastDepositID+1 {
		panic("confirmed deposits are not continguous")
	}

	return filteredDeposits, nil
}

// logDatabaseContractMismatch records any instances of our database
// desynchronizing from the disrburser contract. This method panics in
// irrecoverable cases of desynchronization.
func (d *Driver) logDatabaseContractMismatch(
	lastDisbursementID *uint64,
	contractNextID uint64,
) {

	switch {

	// Database indicates we have done a disbursement.
	case lastDisbursementID != nil:
		switch {
		// The last recorded disbursement is behind what the contract believes.
		case *lastDisbursementID+1 < contractNextID:
			log.Warn("Recorded disbursements behind contract",
				"last_disbursement_id", *lastDisbursementID,
				"contract_next_id", contractNextID)
			d.metrics.DepositIDMismatch.Inc()

		// The last recorded disbursement is ahead of what the contract believes.
		// This should NEVER happen unless the sequencer blows up and loses
		// state. Exit so that the the problem can be surfaced loudly.
		case *lastDisbursementID+1 > contractNextID:
			log.Error("Recorded disbursements ahead of contract",
				"last_disbursement_id", *lastDisbursementID,
				"contract_next_id", contractNextID)
			panic("Recorded disbursements ahead contract")

		// Databse and contract are in sync.
		default:
			d.metrics.DepositIDMismatch.Set(0.0)
		}

	// Database indicates we have not done a disbursement, but contract does.
	case contractNextID != 0:
		// The contract shows that is has disbursed, but we don't have a
		// recording of it.
		log.Warn("Recorded disbursements behind contract",
			"last_disbursement_id", nil,
			"contract_next_id", contractNextID)
		d.metrics.DepositIDMismatch.Inc()

	// Database and contract indicate we have not done a disbursement.
	default:
		d.metrics.DepositIDMismatch.Set(0.0)
	}
}

func (d *Driver) upsertDeposits(deposits []db.Deposit, end uint64) error {
	err := d.cfg.Database.UpsertDeposits(deposits, end)
	if err != nil {
		d.metrics.FailedDatabaseMethods.With(DBMethodUpsertDeposits).Inc()
		return err
	}
	return nil
}

func (d *Driver) confirmedDeposits(blockNumber uint64) ([]db.Deposit, error) {
	confirmedDeposits, err := d.cfg.Database.ConfirmedDeposits(
		blockNumber, d.cfg.NumConfirmations,
	)
	if err != nil {
		d.metrics.FailedDatabaseMethods.With(DBMethodConfirmedDeposits).Inc()
		return nil, err
	}
	return confirmedDeposits, nil
}

func (d *Driver) lastProcessedBlock() (*uint64, error) {
	lastProcessedBlock, err := d.cfg.Database.LastProcessedBlock()
	if err != nil {
		d.metrics.FailedDatabaseMethods.With(DBMethodLastProcessedBlock).Inc()
		return nil, err
	}
	return lastProcessedBlock, nil
}

func (d *Driver) upsertPendingTx(pendingTx db.PendingTx) error {
	err := d.cfg.Database.UpsertPendingTx(pendingTx)
	if err != nil {
		d.metrics.FailedDatabaseMethods.With(DBMethodUpsertPendingTx).Inc()
		return err
	}
	return nil
}

func (d *Driver) listPendingTxs() ([]db.PendingTx, error) {
	pendingTxs, err := d.cfg.Database.ListPendingTxs()
	if err != nil {
		d.metrics.FailedDatabaseMethods.With(DBMethodListPendingTxs).Inc()
		return nil, err
	}
	return pendingTxs, nil
}

func (d *Driver) latestDisbursementID() (*uint64, error) {
	lastDisbursementID, err := d.cfg.Database.LatestDisbursementID()
	if err != nil {
		d.metrics.FailedDatabaseMethods.With(DBMethodLatestDisbursementID).Inc()
		return nil, err
	}
	return lastDisbursementID, nil
}

func (d *Driver) deletePendingTx(startID, endID uint64) error {
	err := d.cfg.Database.DeletePendingTx(startID, endID)
	if err != nil {
		d.metrics.FailedDatabaseMethods.With(DBMethodDeletePendingTx).Inc()
		return err
	}
	return nil
}

func (d *Driver) updateBalanceMetrics(ctx context.Context) {
	disburserBal, err := d.cfg.L2Client.BalanceAt(ctx, d.walletAddr, nil)
	if err != nil {
		log.Error("Error getting disburser wallet balance", "err", err)
		disburserBal = big.NewInt(0)
	}

	depositBal, err := d.cfg.L1Client.BalanceAt(ctx, d.cfg.DepositAddr, nil)
	if err != nil {
		log.Error("Error getting deposit contract balance", "err", err)
		depositBal = big.NewInt(0)
	}

	d.metrics.DisburserBalance.Set(float64(disburserBal.Uint64()))
	d.metrics.DepositContractBalance.Set(float64(depositBal.Uint64()))
}
