package rollup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/eth/gasprice"
)

// SyncService implements the main functionality around pulling in transactions
// and executing them. It can be configured to run in both sequencer mode and in
// verifier mode.
type SyncService struct {
	ctx                       context.Context
	cancel                    context.CancelFunc
	verifier                  bool
	db                        ethdb.Database
	scope                     event.SubscriptionScope
	txFeed                    event.Feed
	txLock                    sync.Mutex
	loopLock                  sync.Mutex
	enable                    bool
	eth1ChainId               uint64
	bc                        *core.BlockChain
	txpool                    *core.TxPool
	L1gpo                     *gasprice.L1Oracle
	client                    RollupClient
	syncing                   atomic.Value
	chainHeadSub              event.Subscription
	OVMContext                OVMContext
	pollInterval              time.Duration
	timestampRefreshThreshold time.Duration
	chainHeadCh               chan core.ChainHeadEvent
	syncType                  SyncType
}

// NewSyncService returns an initialized sync service.
func NewSyncService(ctx context.Context, cfg Config, txpool *core.TxPool, bc *core.BlockChain, db ethdb.Database) (*SyncService, error) {
	if bc == nil {
		return nil, errors.New("Must pass BlockChain to SyncService")
	}

	ctx, cancel := context.WithCancel(ctx)
	_ = cancel // satisfy govet

	if cfg.IsVerifier {
		log.Info("Running in verifier mode", "sync-type", cfg.SyncType.String())
	} else {
		log.Info("Running in sequencer mode", "sync-type", cfg.SyncType.String())
	}

	pollInterval := cfg.PollInterval
	if pollInterval == 0 {
		log.Info("Sanitizing poll interval to 15 seconds")
		pollInterval = time.Second * 15
	}
	timestampRefreshThreshold := cfg.TimestampRefreshThreshold
	if timestampRefreshThreshold == 0 {
		log.Info("Sanitizing timestamp refresh threshold to 5 minutes")
		timestampRefreshThreshold = time.Minute * 5
	}

	// Layer 2 chainid
	chainID := bc.Config().ChainID
	if chainID == nil {
		return nil, errors.New("Must configure with chain id")
	}
	// Initialize the rollup client
	client := NewClient(cfg.RollupClientHttp, chainID)
	log.Info("Configured rollup client", "url", cfg.RollupClientHttp, "chain-id", chainID.Uint64())
	service := SyncService{
		ctx:                       ctx,
		cancel:                    cancel,
		verifier:                  cfg.IsVerifier,
		enable:                    cfg.Eth1SyncServiceEnable,
		syncing:                   atomic.Value{},
		bc:                        bc,
		txpool:                    txpool,
		chainHeadCh:               make(chan core.ChainHeadEvent, 1),
		eth1ChainId:               cfg.Eth1ChainId,
		client:                    client,
		db:                        db,
		pollInterval:              pollInterval,
		timestampRefreshThreshold: timestampRefreshThreshold,
		syncType:                  cfg.SyncType,
	}

	// The chainHeadSub is used to synchronize the SyncService with the chain.
	// As the SyncService processes transactions, it waits until the transaction
	// is added to the chain. This synchronization is required for handling
	// reorgs and also favors safety over liveliness. If a transaction breaks
	// things downstream, it is expected that this channel will halt ingestion
	// of additional transactions by the SyncService.
	service.chainHeadSub = service.bc.SubscribeChainHeadEvent(service.chainHeadCh)

	// Initialize sync service setup if it is enabled. This code depends on
	// a remote server that indexes the layer one contracts. Place this
	// code behind this if statement so that this can run without the
	// requirement of the remote server being up.
	if service.enable {
		// Ensure that the rollup client can connect to a remote server
		// before starting.
		err := service.ensureClient()
		if err != nil {
			return nil, fmt.Errorf("Rollup client unable to connect: %w", err)
		}

		// Wait until the remote service is done syncing
		for {
			status, err := service.client.SyncStatus()
			if err != nil {
				log.Error("Cannot get sync status")
				continue
			}
			if !status.Syncing {
				break
			}
			log.Info("Still syncing", "index", status.CurrentTransactionIndex, "tip", status.HighestKnownTransactionIndex)
			time.Sleep(10 * time.Second)
		}

		// Initialize the latest L1 data here to make sure that
		// it happens before the RPC endpoints open up
		// Only do it if the sync service is enabled so that this
		// can be ran without needing to have a configured RollupClient
		err = service.initializeLatestL1(cfg.CanonicalTransactionChainDeployHeight)
		if err != nil {
			return nil, fmt.Errorf("Cannot initialize latest L1 data: %w", err)
		}

		// Log the OVMContext information on startup
		bn := service.GetLatestL1BlockNumber()
		ts := service.GetLatestL1Timestamp()
		log.Info("Initialized Latest L1 Info", "blocknumber", bn, "timestamp", ts)

		index := service.GetLatestIndex()
		queueIndex := service.GetLatestEnqueueIndex()
		verifiedIndex := service.GetLatestVerifiedIndex()
		block := service.bc.CurrentBlock()
		if block == nil {
			block = types.NewBlock(&types.Header{}, nil, nil, nil)
		}
		header := block.Header()
		log.Info("Initial Rollup State", "state", header.Root.Hex(), "index", stringify(index), "queue-index", stringify(queueIndex), "verified-index", verifiedIndex)

		// The sequencer needs to sync to the tip at start up
		// By setting the sync status to true, it will prevent RPC calls.
		// Be sure this is set to false later.
		if !service.verifier {
			service.setSyncStatus(true)
		}
	}
	return &service, nil
}

// ensureClient checks to make sure that the remote transaction source is
// available. It will return an error if it cannot connect via HTTP
func (s *SyncService) ensureClient() error {
	_, err := s.client.GetLatestEthContext()
	if err != nil {
		return fmt.Errorf("Cannot connect to data service: %w", err)
	}
	return nil
}

// Start initializes the SyncService.
func (s *SyncService) Start() error {
	if !s.enable {
		log.Info("Sync Service not initialized")
		return nil
	}
	log.Info("Initializing Sync Service", "eth1-chainid", s.eth1ChainId)

	if s.verifier {
		go s.VerifierLoop()
	} else {
		// The sequencer must sync the transactions to the tip and the
		// pending queue transactions on start before setting sync status
		// to false and opening up the RPC to accept transactions.
		err := s.syncTransactionsToTip(s.syncType)
		if err != nil {
			return fmt.Errorf("Cannot sync transactions to the tip: %w", err)
		}
		err = s.syncQueueToTip()
		if err != nil {
			log.Error("Sequencer cannot sync queue", "msg", err)
		}
		s.setSyncStatus(false)
		go s.SequencerLoop()
	}
	return nil
}

// initializeLatestL1 sets the initial values of the `L1BlockNumber`
// and `L1Timestamp` to the deploy height of the Canonical Transaction
// chain if the chain is empty, otherwise set it from the last
// transaction processed. This must complete before transactions
// are accepted via RPC when running as a sequencer.
func (s *SyncService) initializeLatestL1(ctcDeployHeight *big.Int) error {
	index := s.GetLatestIndex()
	if index == nil {
		if ctcDeployHeight == nil {
			return errors.New("Must configure with canonical transaction chain deploy height")
		}
		log.Info("Initializing initial OVM Context", "ctc-deploy-height", ctcDeployHeight.Uint64())
		context, err := s.client.GetEthContext(ctcDeployHeight.Uint64())
		if err != nil {
			return fmt.Errorf("Cannot fetch ctc deploy block at height %d: %w", ctcDeployHeight.Uint64(), err)
		}
		s.SetLatestL1Timestamp(context.Timestamp)
		s.SetLatestL1BlockNumber(context.BlockNumber)
	} else {
		log.Info("Found latest index", "index", *index)
		block := s.bc.GetBlockByNumber(*index - 1)
		if block == nil {
			block = s.bc.CurrentBlock()
			idx := block.Number().Uint64()
			if idx > *index {
				// This is recoverable with a reorg but should never happen
				return fmt.Errorf("Current block height greater than index")
			}
			s.SetLatestIndex(&idx)
			log.Info("Block not found, resetting index", "new", idx, "old", *index-1)
		}
		txs := block.Transactions()
		if len(txs) != 1 {
			log.Error("Unexpected number of transactions in block: %d", len(txs))
		}
		tx := txs[0]
		s.SetLatestL1Timestamp(tx.L1Timestamp())
		s.SetLatestL1BlockNumber(tx.L1BlockNumber().Uint64())
	}
	queueIndex := s.GetLatestEnqueueIndex()
	if queueIndex == nil {
		enqueue, err := s.client.GetLastConfirmedEnqueue()
		if err != nil {
			return fmt.Errorf("Cannot fetch last confirmed queue tx: %w", err)
		}
		// There are no enqueues yet
		if enqueue == nil {
			return nil
		}
		queueIndex = enqueue.GetMeta().QueueIndex
	}
	s.SetLatestEnqueueIndex(queueIndex)
	return nil
}

// setSyncStatus sets the `syncing` field as well as prevents
// any transactions from coming in via RPC.
// `syncing` should never be set directly outside of this function.
func (s *SyncService) setSyncStatus(status bool) {
	log.Info("Setting sync status", "status", status)
	s.syncing.Store(status)
}

// IsSyncing returns the syncing status of the syncservice.
// Returns false if not yet set.
func (s *SyncService) IsSyncing() bool {
	value := s.syncing.Load()
	val, ok := value.(bool)
	if !ok {
		return false
	}
	return val
}

// Stop will close the open channels and cancel the goroutines
// started by this service.
func (s *SyncService) Stop() error {
	s.chainHeadSub.Unsubscribe()
	close(s.chainHeadCh)
	s.scope.Close()

	if s.cancel != nil {
		defer s.cancel()
	}
	return nil
}

// VerifierLoop is the main loop for Verifier mode
func (s *SyncService) VerifierLoop() {
	log.Info("Starting Verifier Loop", "poll-interval", s.pollInterval, "timestamp-refresh-threshold", s.timestampRefreshThreshold)
	for {
		if err := s.verify(); err != nil {
			log.Error("Could not verify", "error", err)
		}
		time.Sleep(s.pollInterval)
	}
}

// verify is the main logic for the Verifier. The verifier logic is different
// depending on the SyncType.
func (s *SyncService) verify() error {
	switch s.syncType {
	case SyncTypeBatched:
		err := s.syncTransactionBatchesToTip()
		if err != nil {
			log.Error("Verifier cannot sync transaction batches to tip", "msg", err)
		}
	case SyncTypeSequenced:
		err := s.syncTransactionsToTip(SyncTypeSequenced)
		if err != nil {
			log.Error("Verifier cannot sync transactions with SyncTypeSequencer", "msg", err)
		}
	}
	return nil
}

// SequencerLoop is the polling loop that runs in sequencer mode. It sequences
// transactions and then updates the EthContext.
func (s *SyncService) SequencerLoop() {
	log.Info("Starting Sequencer Loop", "poll-interval", s.pollInterval, "timestamp-refresh-threshold", s.timestampRefreshThreshold)
	for {
		s.txLock.Lock()
		err := s.sequence()
		if err != nil {
			log.Error("Could not sequence", "error", err)
		}
		s.txLock.Unlock()

		if s.updateEthContext() != nil {
			log.Error("Could not update execution context", "error", err)
		}
		time.Sleep(s.pollInterval)
	}
}

// sequence is the main logic for the Sequencer. It will sync any `enqueue`
// transactions it has yet to sync and then pull in transaction batches to
// compare against the transactions it has in its local state. The sequencer
// should reorg based on the transaction batches that are posted because
// L1 is the source of truth. The sequencer concurrently accepts user
// transactions via the RPC.
func (s *SyncService) sequence() error {
	err := s.syncQueueToTip()
	if err != nil {
		log.Error("Sequencer cannot sync queue", "msg", err)
	}
	err = s.syncTransactionBatchesToTip()
	if err != nil {
		log.Error("Sequencer cannot sync transaction batches", "msg", err)
	}
	return nil
}

// Methods for safely accessing and storing the latest
// L1 blocknumber and timestamp. These are held in memory.

// GetLatestL1Timestamp returns the OVMContext timestamp
func (s *SyncService) GetLatestL1Timestamp() uint64 {
	return atomic.LoadUint64(&s.OVMContext.timestamp)
}

// GetLatestL1BlockNumber returns the OVMContext blocknumber
func (s *SyncService) GetLatestL1BlockNumber() uint64 {
	return atomic.LoadUint64(&s.OVMContext.blockNumber)
}

// SetLatestL1Timestamp will set the OVMContext timestamp
func (s *SyncService) SetLatestL1Timestamp(ts uint64) {
	atomic.StoreUint64(&s.OVMContext.timestamp, ts)
}

// SetLatestL1BlockNumber will set the OVMContext blocknumber
func (s *SyncService) SetLatestL1BlockNumber(bn uint64) {
	atomic.StoreUint64(&s.OVMContext.blockNumber, bn)
}

// GetLatestEnqueueIndex reads the last queue index processed
func (s *SyncService) GetLatestEnqueueIndex() *uint64 {
	return rawdb.ReadHeadQueueIndex(s.db)
}

// GetNextEnqueueIndex returns the next queue index to process
func (s *SyncService) GetNextEnqueueIndex() uint64 {
	latest := s.GetLatestEnqueueIndex()
	if latest == nil {
		return 0
	}
	return *latest + 1
}

// SetLatestEnqueueIndex writes the last queue index that was processed
func (s *SyncService) SetLatestEnqueueIndex(index *uint64) {
	if index != nil {
		rawdb.WriteHeadQueueIndex(s.db, *index)
	}
}

// GetLatestIndex reads the last CTC index that was processed
func (s *SyncService) GetLatestIndex() *uint64 {
	return rawdb.ReadHeadIndex(s.db)
}

// GetNextIndex reads the next CTC index to process
func (s *SyncService) GetNextIndex() uint64 {
	latest := s.GetLatestIndex()
	if latest == nil {
		return 0
	}
	return *latest + 1
}

// SetLatestIndex writes the last CTC index that was processed
func (s *SyncService) SetLatestIndex(index *uint64) {
	if index != nil {
		rawdb.WriteHeadIndex(s.db, *index)
	}
}

// GetLatestVerifiedIndex reads the last verified CTC index that was processed
// These are set by processing batches of transactions that were submitted to
// the Canonical Transaction Chain.
func (s *SyncService) GetLatestVerifiedIndex() *uint64 {
	return rawdb.ReadHeadVerifiedIndex(s.db)
}

// GetNextVerifiedIndex reads the next verified index
func (s *SyncService) GetNextVerifiedIndex() uint64 {
	index := s.GetLatestVerifiedIndex()
	if index == nil {
		return 0
	}
	return *index + 1
}

// SetLatestVerifiedIndex writes the last verified index that was processed
func (s *SyncService) SetLatestVerifiedIndex(index *uint64) {
	if index != nil {
		rawdb.WriteHeadVerifiedIndex(s.db, *index)
	}
}

// applyTransaction is a higher level API for applying a transaction
func (s *SyncService) applyTransaction(tx *types.Transaction) error {
	if tx.GetMeta().Index != nil {
		return s.applyIndexedTransaction(tx)
	}
	return s.applyTransactionToTip(tx)
}

// applyIndexedTransaction applys a transaction that has an index. This means
// that the source of the transaction was either a L1 batch or from the
// sequencer.
func (s *SyncService) applyIndexedTransaction(tx *types.Transaction) error {
	if tx == nil {
		return errors.New("Transaction is nil in applyIndexedTransaction")
	}
	index := tx.GetMeta().Index
	if index == nil {
		return errors.New("No index found in applyIndexedTransaction")
	}
	log.Trace("Applying indexed transaction", "index", *index)
	next := s.GetNextIndex()
	if *index == next {
		return s.applyTransactionToTip(tx)
	}
	if *index < next {
		return s.applyHistoricalTransaction(tx)
	}
	return fmt.Errorf("Received tx at index %d when looking for %d", *index, next)
}

// applyHistoricalTransaction will compare a historical transaction against what
// is locally indexed. This will trigger a reorg in the future
func (s *SyncService) applyHistoricalTransaction(tx *types.Transaction) error {
	if tx == nil {
		return errors.New("Transaction is nil in applyHistoricalTransaction")
	}
	index := tx.GetMeta().Index
	if index == nil {
		return errors.New("No index is found in applyHistoricalTransaction")
	}
	// Handle the off by one
	block := s.bc.GetBlockByNumber(*index + 1)
	if block == nil {
		return fmt.Errorf("Block %d is not found", *index+1)
	}
	txs := block.Transactions()
	if len(txs) != 1 {
		return fmt.Errorf("More than one transaction found in block %d", *index+1)
	}
	if !isCtcTxEqual(tx, txs[0]) {
		log.Error("Mismatched transaction", "index", index)
	} else {
		log.Debug("Batched transaction matches", "index", index, "hash", tx.Hash().Hex())
	}
	return nil
}

// applyTransactionToTip will do sanity checks on the transaction before
// applying it to the tip. It blocks until the transaction has been included in
// the chain.
func (s *SyncService) applyTransactionToTip(tx *types.Transaction) error {
	if tx == nil {
		return errors.New("nil transaction passed to applyTransactionToTip")
	}
	log.Trace("Applying transaction to tip")
	if tx.L1Timestamp() == 0 {
		ts := s.GetLatestL1Timestamp()
		bn := s.GetLatestL1BlockNumber()
		tx.SetL1Timestamp(ts)
		tx.SetL1BlockNumber(bn)
	} else if tx.L1Timestamp() > s.GetLatestL1Timestamp() {
		ts := tx.L1Timestamp()
		bn := tx.L1BlockNumber()
		s.SetLatestL1Timestamp(ts)
		s.SetLatestL1BlockNumber(bn.Uint64())
	} else if tx.L1Timestamp() < s.GetLatestL1Timestamp() {
		// TODO: this should force a reorg
		log.Error("Timestamp monotonicity violation", "hash", tx.Hash().Hex())
	}

	if tx.GetMeta().Index == nil {
		index := s.GetLatestIndex()
		if index == nil {
			tx.SetIndex(0)
		} else {
			tx.SetIndex(*index + 1)
		}
	}
	s.SetLatestIndex(tx.GetMeta().Index)
	if tx.GetMeta().QueueIndex != nil {
		s.SetLatestEnqueueIndex(tx.GetMeta().QueueIndex)
	}

	// This is a temporary fix for a bug in the SequencerEntrypoint. It will
	// be removed when the custom batch serialization is removed in favor of
	// batching RLP encoded transactions.
	tx = fixType(tx)

	txs := types.Transactions{tx}
	s.txFeed.Send(core.NewTxsEvent{Txs: txs})
	// Block until the transaction has been added to the chain
	log.Debug("Waiting for transaction to be added to chain", "hash", tx.Hash().Hex())
	<-s.chainHeadCh

	return nil
}

// applyBatchedTransaction applies transactions that were batched to layer one.
// The sequencer checks for batches over time to make sure that it does not
// deviate from the L1 state and this is the main method of transaction
// ingestion for the verifier.
func (s *SyncService) applyBatchedTransaction(tx *types.Transaction) error {
	if tx == nil {
		return errors.New("nil transaction passed into applyBatchedTransaction")
	}
	index := tx.GetMeta().Index
	if index == nil {
		return errors.New("No index found on transaction")
	}
	log.Trace("Applying batched transaction", "index", *index)
	err := s.applyIndexedTransaction(tx)
	if err != nil {
		return fmt.Errorf("Cannot apply batched transaction: %w", err)
	}
	s.SetLatestVerifiedIndex(index)
	return nil
}

// Higher level API for applying transactions. Should only be called for
// queue origin sequencer transactions, as the contracts on L1 manage the same
// validity checks that are done here.
func (s *SyncService) ValidateAndApplySequencerTransaction(tx *types.Transaction) error {
	if s.verifier {
		return errors.New("Verifier does not accept transactions out of band")
	}
	if tx == nil {
		return errors.New("nil transaction passed to ValidateAndApplySequencerTransaction")
	}

	s.txLock.Lock()
	defer s.txLock.Unlock()
	log.Trace("Sequencer transaction validation", "hash", tx.Hash().Hex())

	qo := tx.QueueOrigin()
	if qo == nil {
		return errors.New("invalid transaction with no queue origin")
	}
	if qo.Uint64() != uint64(types.QueueOriginSequencer) {
		return fmt.Errorf("invalid transaction with queue origin %d", qo.Uint64())
	}
	err := s.txpool.ValidateTx(tx)
	if err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	// Set the raw transaction data in the meta.
	txRaw, err := getRawTransaction(tx)
	if err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}
	meta := tx.GetMeta()
	newMeta := types.NewTransactionMeta(
		meta.L1BlockNumber,
		meta.L1Timestamp,
		meta.L1MessageSender,
		meta.SignatureHashType,
		types.QueueOrigin(meta.QueueOrigin.Uint64()),
		meta.Index,
		meta.QueueIndex,
		txRaw,
	)
	tx.SetTransactionMeta(newMeta)
	return s.applyTransaction(tx)
}

// syncTransactionsToTip will sync all of the transactions to the tip.
// The syncType determines the source of the transactions.
func (s *SyncService) syncTransactionsToTip(syncType SyncType) error {
	s.loopLock.Lock()
	defer s.loopLock.Unlock()

	for {
		latest, err := s.client.GetLatestTransaction(syncType)
		if err != nil {
			return fmt.Errorf("Cannot get latest transaction: %w", err)
		}
		if latest == nil {
			log.Info("No transactions to sync")
			return nil
		}
		latestIndex := latest.GetMeta().Index
		if latestIndex == nil {
			return errors.New("Latest index is nil")
		}
		nextIndex := s.GetNextIndex()
		log.Info("Syncing transactions to tip", "start", *latestIndex, "end", nextIndex)

		for i := nextIndex; i <= *latestIndex; i++ {
			tx, err := s.client.GetTransaction(i, syncType)
			if err != nil {
				log.Error("Cannot get latest transaction", "msg", err)
				time.Sleep(time.Second * 2)
				continue
			}
			if tx == nil {
				return fmt.Errorf("Transaction %d is nil", i)
			}
			err = s.applyTransaction(tx)
			if err != nil {
				return fmt.Errorf("Cannot apply transaction: %w", err)
			}
		}

		post, err := s.client.GetLatestTransaction(syncType)
		if err != nil {
			return fmt.Errorf("Cannot get latest transaction: %w", err)
		}
		postLatestIndex := post.GetMeta().Index
		if postLatestIndex == nil {
			return errors.New("Latest index is nil")
		}
		if *postLatestIndex == *latestIndex {
			return nil
		}
	}
}

// syncTransactionBatchesToTip will sync all of the transaction batches to the
// tip
func (s *SyncService) syncTransactionBatchesToTip() error {
	s.loopLock.Lock()
	defer s.loopLock.Unlock()
	log.Debug("Syncing transaction batches to tip")

	for {
		latest, _, err := s.client.GetLatestTransactionBatch()
		if err != nil {
			return fmt.Errorf("Cannot get latest transaction batch: %w", err)
		}
		if latest == nil {
			log.Info("No transaction batches to sync")
			return nil
		}
		latestIndex := latest.Index
		nextIndex := s.GetNextVerifiedIndex()

		for i := nextIndex; i <= latestIndex; i++ {
			log.Debug("Fetching transaction batch", "index", i)
			_, txs, err := s.client.GetTransactionBatch(i)
			if err != nil {
				return fmt.Errorf("Cannot get transaction batch: %w", err)
			}
			for _, tx := range txs {
				s.applyBatchedTransaction(tx)
			}
		}
		post, _, err := s.client.GetLatestTransactionBatch()
		if err != nil {
			return fmt.Errorf("Cannot get latest transaction batch: %w", err)
		}
		if post.Index == latest.Index {
			return nil
		}
	}
}

// syncQueueToTip will sync the `enqueue` transactions to the tip
// from the last known `enqueue` transaction
func (s *SyncService) syncQueueToTip() error {
	s.loopLock.Lock()
	defer s.loopLock.Unlock()

	for {
		latest, err := s.client.GetLatestEnqueue()
		if err != nil {
			return fmt.Errorf("Cannot get latest enqueue transaction: %w", err)
		}
		if latest == nil {
			log.Info("No enqueue transactions to sync")
			return nil
		}
		latestIndex := latest.GetMeta().QueueIndex
		if latestIndex == nil {
			return errors.New("Latest queue transaction has no queue index")
		}
		nextIndex := s.GetNextEnqueueIndex()
		log.Info("Syncing enqueue transactions to tip", "start", *latestIndex, "end", nextIndex)

		for i := nextIndex; i <= *latestIndex; i++ {
			tx, err := s.client.GetEnqueue(i)
			if err != nil {
				return fmt.Errorf("Canot get enqueue transaction; %w", err)
			}
			if tx == nil {
				return fmt.Errorf("Cannot get queue tx at index %d", i)
			}
			err = s.applyTransaction(tx)
			if err != nil {
				return fmt.Errorf("Cannot apply transaction: %w", err)
			}
		}
		post, err := s.client.GetLatestEnqueue()
		if err != nil {
			return fmt.Errorf("Cannot get latest transaction: %w", err)
		}
		postLatestIndex := post.GetMeta().QueueIndex
		if postLatestIndex == nil {
			return errors.New("Latest queue index is nil")
		}
		if *latestIndex == *postLatestIndex {
			return nil
		}
	}
}

// updateEthContext will update the OVM execution context's
// timestamp and blocknumber if enough time has passed since
// it was last updated. This is a sequencer only function.
func (s *SyncService) updateEthContext() error {
	context, err := s.client.GetLatestEthContext()
	if err != nil {
		return fmt.Errorf("Cannot get eth context: %w", err)
	}
	current := time.Unix(int64(s.GetLatestL1Timestamp()), 0)
	next := time.Unix(int64(context.Timestamp), 0)
	if next.Sub(current) > s.timestampRefreshThreshold {
		log.Info("Updating Eth Context", "timetamp", context.Timestamp, "blocknumber", context.BlockNumber)
		s.SetLatestL1BlockNumber(context.BlockNumber)
		s.SetLatestL1Timestamp(context.Timestamp)
	}
	return nil
}

// SubscribeNewTxsEvent registers a subscription of NewTxsEvent and
// starts sending event to the given channel.
func (s *SyncService) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return s.scope.Track(s.txFeed.Subscribe(ch))
}

// getRawTransaction will return the raw serialization of the transaction. This
// function will be deprecated in the near future when the batch serialization
// is RLP encoded transactions.
func getRawTransaction(tx *types.Transaction) ([]byte, error) {
	if tx == nil {
		return nil, errors.New("Cannot process nil transaction")
	}
	v, r, s := tx.RawSignatureValues()

	// V parameter here will include the chain ID, so we need to recover the original V. If the V
	// does not equal zero or one, we have an invalid parameter and need to throw an error.
	// This is technically a duplicate check because it happens inside of
	// `tx.AsMessage` as well.
	v = new(big.Int).SetUint64(v.Uint64() - 35 - 2*tx.ChainId().Uint64())
	if v.Uint64() != 0 && v.Uint64() != 1 {
		return nil, fmt.Errorf("invalid signature v parameter: %d", v.Uint64())
	}

	// Since we use a fixed encoding, we need to insert some placeholder address to represent that
	// the user wants to create a contract (in this case, the zero address).
	var target common.Address
	if tx.To() == nil {
		target = common.Address{}
	} else {
		target = *tx.To()
	}

	// Divide the gas price by one million to compress it
	// before it is send to the sequencer entrypoint. This is to save
	// space on calldata.
	gasPrice := new(big.Int).Div(tx.GasPrice(), new(big.Int).SetUint64(1000000))

	// Sequencer uses a custom encoding structure --
	// We originally receive sequencer transactions encoded in this way, but we decode them before
	// inserting into Geth so we can make transactions easily parseable. However, this means that
	// we need to re-encode the transactions before executing them.
	var data = new(bytes.Buffer)
	data.WriteByte(getSignatureType(tx))                         // 1 byte: 00 == EIP 155, 02 == ETH Sign Message
	data.Write(fillBytes(r, 32))                                 // 32 bytes: Signature `r` parameter
	data.Write(fillBytes(s, 32))                                 // 32 bytes: Signature `s` parameter
	data.Write(fillBytes(v, 1))                                  // 1 byte: Signature `v` parameter
	data.Write(fillBytes(new(big.Int).SetUint64(tx.Gas()), 3))   // 3 bytes: Gas limit
	data.Write(fillBytes(gasPrice, 3))                           // 3 bytes: Gas price
	data.Write(fillBytes(new(big.Int).SetUint64(tx.Nonce()), 3)) // 3 bytes: Nonce
	data.Write(target.Bytes())                                   // 20 bytes: Target address
	data.Write(tx.Data())

	return data.Bytes(), nil
}

// fillBytes is taken from a newer version of the golang standard library
func fillBytes(x *big.Int, size int) []byte {
	b := x.Bytes()
	switch {
	case len(b) > size:
		panic("math/big: value won't fit requested size")
	case len(b) == size:
		return b
	default:
		buf := make([]byte, size)
		copy(buf[size-len(b):], b)
		return buf
	}
}

// getSignatureType is a patch to fix a bug in the contracts. Will be deprecated
// with the move to RLP encoded transactions in batches.
func getSignatureType(tx *types.Transaction) uint8 {
	if tx.SignatureHashType() == 0 {
		return 0
	} else if tx.SignatureHashType() == 1 {
		return 2
	} else {
		return 1
	}
}

// This is a temporary fix to patch the enums being used in the raw data
func fixType(tx *types.Transaction) *types.Transaction {
	meta := tx.GetMeta()
	raw := meta.RawTransaction
	if len(raw) == 0 {
		log.Error("Transaction with no raw detected")
		return tx
	}
	if raw[0] == 0x00 {
		return tx
	} else if raw[0] == 0x01 {
		raw[0] = 0x02
	}
	queueOrigin := types.QueueOrigin(meta.QueueOrigin.Uint64())
	fixed := types.NewTransactionMeta(meta.L1BlockNumber, meta.L1Timestamp, meta.L1MessageSender, meta.SignatureHashType, queueOrigin, meta.Index, meta.QueueIndex, raw)
	tx.SetTransactionMeta(fixed)
	return tx
}

func stringify(i *uint64) string {
	if i == nil {
		return "<nil>"
	}
	return strconv.FormatUint(*i, 10)
}
