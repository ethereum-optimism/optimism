package rollup

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum-optimism/optimism/l2geth/core"
	"github.com/ethereum-optimism/optimism/l2geth/core/state"
	"github.com/ethereum-optimism/optimism/l2geth/ethdb"
	"github.com/ethereum-optimism/optimism/l2geth/event"
	"github.com/ethereum-optimism/optimism/l2geth/log"

	"github.com/ethereum-optimism/optimism/l2geth/core/rawdb"
	"github.com/ethereum-optimism/optimism/l2geth/core/types"

	"github.com/ethereum-optimism/optimism/l2geth/eth/gasprice"
	"github.com/ethereum-optimism/optimism/l2geth/rollup/fees"
	"github.com/ethereum-optimism/optimism/l2geth/rollup/rcfg"
)

var (
	// errBadConfig is the error when the SyncService is started with invalid
	// configuration options
	errBadConfig = errors.New("bad config")
	// errShortRemoteTip is an error for when the remote tip is shorter than the
	// local tip
	errShortRemoteTip = errors.New("unexpected remote less than tip")
	// errZeroGasPriceTx is the error for when a user submits a transaction
	// with gas price zero and fees are currently enforced
	errZeroGasPriceTx = errors.New("cannot accept 0 gas price transaction")
	float1            = big.NewFloat(1)
)

// SyncService implements the main functionality around pulling in transactions
// and executing them. It can be configured to run in both sequencer mode and in
// verifier mode.
type SyncService struct {
	ctx                            context.Context
	cancel                         context.CancelFunc
	verifier                       bool
	db                             ethdb.Database
	scope                          event.SubscriptionScope
	txFeed                         event.Feed
	txLock                         sync.Mutex
	loopLock                       sync.Mutex
	enable                         bool
	bc                             *core.BlockChain
	txpool                         *core.TxPool
	RollupGpo                      *gasprice.RollupOracle
	client                         RollupClient
	syncing                        atomic.Value
	chainHeadSub                   event.Subscription
	OVMContext                     OVMContext
	pollInterval                   time.Duration
	timestampRefreshThreshold      time.Duration
	chainHeadCh                    chan core.ChainHeadEvent
	backend                        Backend
	gasPriceOracleOwnerAddress     common.Address
	gasPriceOracleOwnerAddressLock *sync.RWMutex
	enforceFees                    bool
	signer                         types.Signer
	feeThresholdUp                 *big.Float
	feeThresholdDown               *big.Float
}

// NewSyncService returns an initialized sync service
func NewSyncService(ctx context.Context, cfg Config, txpool *core.TxPool, bc *core.BlockChain, db ethdb.Database) (*SyncService, error) {
	if bc == nil {
		return nil, errors.New("Must pass BlockChain to SyncService")
	}

	ctx, cancel := context.WithCancel(ctx)
	_ = cancel // satisfy govet

	if cfg.IsVerifier {
		log.Info("Running in verifier mode", "sync-backend", cfg.Backend.String())
	} else {
		log.Info("Running in sequencer mode", "sync-backend", cfg.Backend.String())
		log.Info("Fees", "threshold-up", cfg.FeeThresholdUp, "threshold-down", cfg.FeeThresholdDown)
		log.Info("Enforce Fees", "set", cfg.EnforceFees)
	}

	pollInterval := cfg.PollInterval
	if pollInterval == 0 {
		log.Info("Sanitizing poll interval to 15 seconds")
		pollInterval = time.Second * 15
	}
	timestampRefreshThreshold := cfg.TimestampRefreshThreshold
	if timestampRefreshThreshold == 0 {
		log.Info("Sanitizing timestamp refresh threshold to 3 minutes")
		timestampRefreshThreshold = time.Minute * 3
	}

	// Layer 2 chainid
	chainID := bc.Config().ChainID
	if chainID == nil {
		return nil, errors.New("Must configure with chain id")
	}
	// Initialize the rollup client
	client := NewClient(cfg.RollupClientHttp, chainID)
	log.Info("Configured rollup client", "url", cfg.RollupClientHttp, "chain-id", chainID.Uint64(), "ctc-deploy-height", cfg.CanonicalTransactionChainDeployHeight)

	// Ensure sane values for the fee thresholds
	if cfg.FeeThresholdDown != nil {
		// The fee threshold down should be less than 1
		if cfg.FeeThresholdDown.Cmp(float1) != -1 {
			return nil, fmt.Errorf("%w: fee threshold down not lower than 1: %f", errBadConfig,
				cfg.FeeThresholdDown)
		}
	}
	if cfg.FeeThresholdUp != nil {
		// The fee threshold up should be greater than 1
		if cfg.FeeThresholdUp.Cmp(float1) != 1 {
			return nil, fmt.Errorf("%w: fee threshold up not larger than 1: %f", errBadConfig,
				cfg.FeeThresholdUp)
		}
	}

	service := SyncService{
		ctx:                            ctx,
		cancel:                         cancel,
		verifier:                       cfg.IsVerifier,
		enable:                         cfg.Eth1SyncServiceEnable,
		syncing:                        atomic.Value{},
		bc:                             bc,
		txpool:                         txpool,
		chainHeadCh:                    make(chan core.ChainHeadEvent, 1),
		client:                         client,
		db:                             db,
		pollInterval:                   pollInterval,
		timestampRefreshThreshold:      timestampRefreshThreshold,
		backend:                        cfg.Backend,
		gasPriceOracleOwnerAddress:     cfg.GasPriceOracleOwnerAddress,
		gasPriceOracleOwnerAddressLock: new(sync.RWMutex),
		enforceFees:                    cfg.EnforceFees,
		signer:                         types.NewEIP155Signer(chainID),
		feeThresholdDown:               cfg.FeeThresholdDown,
		feeThresholdUp:                 cfg.FeeThresholdUp,
	}

	// The chainHeadSub is used to synchronize the SyncService with the chain.
	// As the SyncService processes transactions, it waits until the transaction
	// is added to the chain. This synchronization is required for handling
	// reorgs and also favors safety over liveliness. If a transaction breaks
	// things downstream, it is expected that this channel will halt ingestion
	// of additional transactions by the SyncService.
	service.chainHeadSub = service.bc.SubscribeChainHeadEvent(service.chainHeadCh)

	// Initial sync service setup if it is enabled. This code depends on
	// a remote server that indexes the layer one contracts. Place this
	// code behind this if statement so that this can run without the
	// requirement of the remote server being up.
	if service.enable {
		// Ensure that the rollup client can connect to a remote server
		// before starting. Retry until it can connect.
		tEnsure := time.NewTicker(10 * time.Second)
		for ; true; <-tEnsure.C {
			err := service.ensureClient()
			if err != nil {
				log.Info("Cannot connect to upstream service", "msg", err)
			} else {
				log.Info("Connected to upstream service")
				tEnsure.Stop()
				break
			}
		}

		if !cfg.IsVerifier || cfg.Backend == BackendL2 {
			// Wait until the remote service is done syncing
			tStatus := time.NewTicker(10 * time.Second)
			for ; true; <-tStatus.C {
				status, err := service.client.SyncStatus(service.backend)
				if err != nil {
					log.Error("Cannot get sync status")
					continue
				}
				if !status.Syncing {
					tStatus.Stop()
					break
				}
				log.Info("Still syncing", "index", status.CurrentTransactionIndex, "tip", status.HighestKnownTransactionIndex)
			}
		}

		// Initialize the latest L1 data here to make sure that
		// it happens before the RPC endpoints open up
		// Only do it if the sync service is enabled so that this
		// can be ran without needing to have a configured RollupClient.
		err := service.initializeLatestL1(cfg.CanonicalTransactionChainDeployHeight)
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

// Start initializes the service
func (s *SyncService) Start() error {
	if !s.enable {
		log.Info("Running without syncing enabled")
		return nil
	}
	log.Info("Initializing Sync Service")
	if err := s.updateGasPriceOracleCache(nil); err != nil {
		return err
	}

	if s.verifier {
		go s.VerifierLoop()
	} else {
		go func() {
			if err := s.syncTransactionsToTip(); err != nil {
				log.Crit("Sequencer cannot sync transactions to tip", "err", err)
			}
			if err := s.syncQueueToTip(); err != nil {
				log.Crit("Sequencer cannot sync queue to tip", "err", err)
			}
			s.setSyncStatus(false)
			go s.SequencerLoop()
		}()
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
		// Recover from accidentally skipped batches if necessary.
		if s.verifier && s.backend == BackendL1 {
			tx, err := s.client.GetRawTransaction(*index, s.backend)
			if err != nil {
				return fmt.Errorf("Cannot fetch transaction from dtl at index %d: %w", *index, err)
			}

			oldbatchIndex := s.GetLatestBatchIndex()
			newBatchIndex := tx.Transaction.BatchIndex
			if tx.Transaction.BatchIndex > 0 {
				newBatchIndex -= 1
			}

			log.Info("Updating batch index", "old", oldbatchIndex, "new", newBatchIndex)
			s.SetLatestBatchIndex(&newBatchIndex)
		}

		log.Info("Found latest index", "index", *index)
		block := s.bc.GetBlockByNumber(*index + 1)
		if block == nil {
			block = s.bc.CurrentBlock()
			blockNum := block.Number().Uint64()
			if blockNum > *index {
				// This is recoverable with a reorg but should never happen
				return fmt.Errorf("Current block height greater than index")
			}
			var idx *uint64
			if blockNum > 0 {
				num := blockNum - 1
				idx = &num
			}
			s.SetLatestIndex(idx)
			log.Info("Block not found, resetting index", "new", stringify(idx), "old", *index)
		}
		txs := block.Transactions()
		if len(txs) != 1 {
			log.Error("Unexpected number of transactions in block", "count", len(txs))
			panic("Cannot recover OVM Context")
		}
		tx := txs[0]
		s.SetLatestL1Timestamp(tx.L1Timestamp())
		s.SetLatestL1BlockNumber(tx.L1BlockNumber().Uint64())
	}
	queueIndex := s.GetLatestEnqueueIndex()
	if queueIndex == nil {
		enqueue, err := s.client.GetLastConfirmedEnqueue()
		// There are no enqueues yet
		if errors.Is(err, errElementNotFound) {
			return nil
		}
		// Other unexpected error
		if err != nil {
			return fmt.Errorf("Cannot fetch last confirmed queue tx: %w", err)
		}
		// No error, the queue element was found
		queueIndex = enqueue.GetMeta().QueueIndex
	} else {
		log.Info("Found latest queue index", "queue-index", *queueIndex)
		// The queue index is defined. Work backwards from the tip
		// to make sure that the indexed queue index is the latest
		// enqueued transaction
		block := s.bc.CurrentBlock()
		for {
			// There are no blocks in the chain
			// This should never happen
			if block == nil {
				log.Warn("Found no genesis block when fixing queue index")
				break
			}
			num := block.Number().Uint64()
			// Handle the genesis block
			if num == 0 {
				log.Info("Hit genesis block when fixing queue index")
				queueIndex = nil
				break
			}
			txs := block.Transactions()
			// This should never happen
			if len(txs) != 1 {
				log.Warn("Found block with unexpected number of txs", "count", len(txs), "height", num)
				break
			}
			tx := txs[0]
			qi := tx.GetMeta().QueueIndex
			// When the queue index is set
			if qi != nil {
				if *qi == *queueIndex {
					log.Info("Found correct staring queue index", "queue-index", *qi)
				} else {
					log.Info("Found incorrect staring queue index, fixing", "old", *queueIndex, "new", *qi)
					queueIndex = qi
				}
				break
			}
			block = s.bc.GetBlockByNumber(num - 1)
		}
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
	log.Info("Stopping sync service")
	s.scope.Close()
	s.chainHeadSub.Unsubscribe()
	close(s.chainHeadCh)

	if s.cancel != nil {
		defer s.cancel()
	}
	return nil
}

// VerifierLoop is the main loop for Verifier mode
func (s *SyncService) VerifierLoop() {
	log.Info("Starting Verifier Loop", "poll-interval", s.pollInterval, "timestamp-refresh-threshold", s.timestampRefreshThreshold)
	t := time.NewTicker(s.pollInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			if err := s.verify(); err != nil {
				log.Error("Could not verify", "error", err)
			}
		case <-s.ctx.Done():
			return
		}
	}
}

// verify is the main logic for the Verifier. The verifier logic is different
// depending on the Backend
func (s *SyncService) verify() error {
	switch s.backend {
	case BackendL1:
		if err := s.syncBatchesToTip(); err != nil {
			return fmt.Errorf("Verifier cannot sync transaction batches to tip: %w", err)
		}
	case BackendL2:
		if err := s.syncTransactionsToTip(); err != nil {
			return fmt.Errorf("Verifier cannot sync transactions with BackendL2: %w", err)
		}
	}
	return nil
}

// SequencerLoop is the polling loop that runs in sequencer mode. It sequences
// transactions and then updates the EthContext.
func (s *SyncService) SequencerLoop() {
	log.Info("Starting Sequencer Loop", "poll-interval", s.pollInterval, "timestamp-refresh-threshold", s.timestampRefreshThreshold)
	t := time.NewTicker(s.pollInterval)
	defer t.Stop()
	for ; true; <-t.C {
		s.txLock.Lock()
		if err := s.sequence(); err != nil {
			log.Error("Could not sequence", "error", err)
		}
		s.txLock.Unlock()

		if err := s.updateL1BlockNumber(); err != nil {
			log.Error("Could not update execution context", "error", err)
		}
	}
}

// sequence is the main logic for the Sequencer. It will sync any `enqueue`
// transactions it has yet to sync and then pull in transaction batches to
// compare against the transactions it has in its local state. The sequencer
// should reorg based on the transaction batches that are posted because
// L1 is the source of truth. The sequencer concurrently accepts user
// transactions via the RPC. When reorg logic is enabled, this should
// also call `syncBatchesToTip`
func (s *SyncService) sequence() error {
	if err := s.syncQueueToTip(); err != nil {
		return fmt.Errorf("Sequencer cannot sequence queue: %w", err)
	}
	return nil
}

func (s *SyncService) syncQueueToTip() error {
	if err := s.syncToTip(s.syncQueue, s.client.GetLatestEnqueueIndex); err != nil {
		return fmt.Errorf("Cannot sync queue to tip: %w", err)
	}
	return nil
}

func (s *SyncService) syncBatchesToTip() error {
	if err := s.syncToTip(s.syncBatches, s.client.GetLatestTransactionBatchIndex); err != nil {
		return fmt.Errorf("Cannot sync transaction batches to tip: %w", err)
	}
	return nil
}

func (s *SyncService) syncTransactionsToTip() error {
	sync := func() (*uint64, error) {
		return s.syncTransactions(s.backend)
	}
	check := func() (*uint64, error) {
		return s.client.GetLatestTransactionIndex(s.backend)
	}
	if err := s.syncToTip(sync, check); err != nil {
		return fmt.Errorf("Verifier cannot sync transactions with backend %s: %w", s.backend.String(), err)
	}
	return nil
}

// updateL1GasPrice queries for the current L1 gas price and then stores it
// in the L1 Gas Price Oracle. This must be called over time to properly
// estimate the transaction fees that the sequencer should charge.
func (s *SyncService) updateL1GasPrice(statedb *state.StateDB) error {
	value, err := s.readGPOStorageSlot(statedb, rcfg.L1GasPriceSlot)
	if err != nil {
		return err
	}
	return s.RollupGpo.SetL1GasPrice(value)
}

// updateL2GasPrice accepts a state db and reads the gas price from the gas
// price oracle at the state that corresponds to the state db. If no state db
// is passed in, then the tip is used.
func (s *SyncService) updateL2GasPrice(statedb *state.StateDB) error {
	value, err := s.readGPOStorageSlot(statedb, rcfg.L2GasPriceSlot)
	if err != nil {
		return err
	}
	return s.RollupGpo.SetL2GasPrice(value)
}

// updateOverhead will update the overhead value from the OVM_GasPriceOracle
// in the local cache
func (s *SyncService) updateOverhead(statedb *state.StateDB) error {
	value, err := s.readGPOStorageSlot(statedb, rcfg.OverheadSlot)
	if err != nil {
		return err
	}
	return s.RollupGpo.SetOverhead(value)
}

// updateScalar will update the scalar value from the OVM_GasPriceOracle
// in the local cache
func (s *SyncService) updateScalar(statedb *state.StateDB) error {
	scalar, err := s.readGPOStorageSlot(statedb, rcfg.ScalarSlot)
	if err != nil {
		return err
	}
	decimals, err := s.readGPOStorageSlot(statedb, rcfg.DecimalsSlot)
	if err != nil {
		return err
	}
	return s.RollupGpo.SetScalar(scalar, decimals)
}

// cacheGasPriceOracleOwner accepts a statedb and caches the gas price oracle
// owner address locally
func (s *SyncService) cacheGasPriceOracleOwner(statedb *state.StateDB) error {
	s.gasPriceOracleOwnerAddressLock.Lock()
	defer s.gasPriceOracleOwnerAddressLock.Unlock()

	value, err := s.readGPOStorageSlot(statedb, rcfg.L2GasPriceOracleOwnerSlot)
	if err != nil {
		return err
	}
	s.gasPriceOracleOwnerAddress = common.BigToAddress(value)
	return nil
}

// readGPOStorageSlot is a helper function for reading storage
// slots from the OVM_GasPriceOracle
func (s *SyncService) readGPOStorageSlot(statedb *state.StateDB, hash common.Hash) (*big.Int, error) {
	var err error
	if statedb == nil {
		statedb, err = s.bc.State()
		if err != nil {
			return nil, err
		}
	}
	result := statedb.GetState(rcfg.L2GasPriceOracleAddress, hash)
	return result.Big(), nil
}

// updateGasPriceOracleCache caches the owner as well as updating the
// the L2 gas price from the OVM_GasPriceOracle.
// This should be sure to read all public variables from the
// OVM_GasPriceOracle
func (s *SyncService) updateGasPriceOracleCache(hash *common.Hash) error {
	var statedb *state.StateDB
	var err error
	if hash != nil {
		statedb, err = s.bc.StateAt(*hash)
	} else {
		statedb, err = s.bc.State()
	}
	if err != nil {
		return err
	}
	if err := s.cacheGasPriceOracleOwner(statedb); err != nil {
		return err
	}
	if err := s.updateL2GasPrice(statedb); err != nil {
		return err
	}
	if err := s.updateL1GasPrice(statedb); err != nil {
		return err
	}
	if err := s.updateOverhead(statedb); err != nil {
		return err
	}
	if err := s.updateScalar(statedb); err != nil {
		return err
	}
	return nil
}

// A thread safe getter for the gas price oracle owner address
func (s *SyncService) GasPriceOracleOwnerAddress() *common.Address {
	s.gasPriceOracleOwnerAddressLock.RLock()
	defer s.gasPriceOracleOwnerAddressLock.RUnlock()
	return &s.gasPriceOracleOwnerAddress
}

/// Update the execution context's timestamp and blocknumber
/// over time. This is only necessary for the sequencer.
func (s *SyncService) updateL1BlockNumber() error {
	context, err := s.client.GetLatestEthContext()
	if err != nil {
		return fmt.Errorf("Cannot get eth context: %w", err)
	}
	latest := s.GetLatestL1BlockNumber()
	if context.BlockNumber > latest {
		log.Info("Updating L1 block number", "blocknumber", context.BlockNumber)
		s.SetLatestL1BlockNumber(context.BlockNumber)
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

// GetLatestBatchIndex reads the last processed transaction batch
func (s *SyncService) GetLatestBatchIndex() *uint64 {
	return rawdb.ReadHeadBatchIndex(s.db)
}

// GetNextBatchIndex reads the index of the next transaction batch to process
func (s *SyncService) GetNextBatchIndex() uint64 {
	index := s.GetLatestBatchIndex()
	if index == nil {
		return 0
	}
	return *index + 1
}

// SetLatestBatchIndex writes the last index of the transaction batch that was processed
func (s *SyncService) SetLatestBatchIndex(index *uint64) {
	if index != nil {
		rawdb.WriteHeadBatchIndex(s.db, *index)
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
		log.Error("Mismatched transaction", "index", *index)
	} else {
		log.Debug("Historical transaction matches", "index", *index, "hash", tx.Hash().Hex())
	}
	return nil
}

// applyTransactionToTip will do sanity checks on the transaction before
// applying it to the tip. It blocks until the transaction has been included in
// the chain. It is assumed that validation around the index has already
// happened.
func (s *SyncService) applyTransactionToTip(tx *types.Transaction) error {
	if tx == nil {
		return errors.New("nil transaction passed to applyTransactionToTip")
	}
	// Queue Origin L1 to L2 transactions must have a timestamp that is set by
	// the L1 block that holds the transaction. This should never happen but is
	// a sanity check to prevent fraudulent execution.
	// No need to unlock here as the lock is only taken when its a queue origin
	// sequencer transaction.
	if tx.QueueOrigin() == types.QueueOriginL1ToL2 {
		if tx.L1Timestamp() == 0 {
			return fmt.Errorf("Queue origin L1 to L2 transaction without a timestamp: %s", tx.Hash().Hex())
		}
	}

	// If there is no L1 timestamp assigned to the transaction, then assign a
	// timestamp to it. The property that L1 to L2 transactions have the same
	// timestamp as the L1 block that it was included in is removed for better
	// UX. This functionality can be added back in during a future release. For
	// now, the sequencer will assign a timestamp to each transaction.
	ts := s.GetLatestL1Timestamp()
	bn := s.GetLatestL1BlockNumber()

	// The L1Timestamp is 0 for QueueOriginSequencer transactions when
	// running as the sequencer, the transactions are coming in via RPC.
	// This code path also runs for replicas/verifiers so any logic involving
	// `time.Now` can only run for the sequencer. All other nodes must listen
	// to what the sequencer says is the timestamp, otherwise there will be a
	// network split.
	// Note that it should never be possible for the timestamp to be set to
	// 0 when running as a verifier.
	shouldMalleateTimestamp := !s.verifier && tx.QueueOrigin() == types.QueueOriginL1ToL2
	if tx.L1Timestamp() == 0 || shouldMalleateTimestamp {
		// Get the latest known timestamp
		current := time.Unix(int64(ts), 0)
		// Get the current clocktime
		now := time.Now()
		// If enough time has passed, then assign the
		// transaction to have the timestamp now. Otherwise,
		// use the current timestamp
		if now.Sub(current) > s.timestampRefreshThreshold {
			current = now
		}
		log.Info("Updating latest timestamp", "timestamp", current, "unix", current.Unix())
		tx.SetL1Timestamp(uint64(current.Unix()))
	} else if tx.L1Timestamp() == 0 && s.verifier {
		// This should never happen
		log.Error("No tx timestamp found when running as verifier", "hash", tx.Hash().Hex())
	} else if tx.L1Timestamp() < ts {
		// This should never happen, but sometimes does
		log.Error("Timestamp monotonicity violation", "hash", tx.Hash().Hex(), "latest", ts, "tx", tx.L1Timestamp())
	}

	l1BlockNumber := tx.L1BlockNumber()
	// Set the L1 blocknumber
	if l1BlockNumber == nil {
		tx.SetL1BlockNumber(bn)
	} else if l1BlockNumber.Uint64() > bn {
		s.SetLatestL1BlockNumber(l1BlockNumber.Uint64())
	} else if l1BlockNumber.Uint64() < bn {
		// l1BlockNumber < latest l1BlockNumber
		// indicates an error
		log.Error("Blocknumber monotonicity violation", "hash", tx.Hash().Hex(),
			"new", l1BlockNumber.Uint64(), "old", bn)
	}

	// Store the latest timestamp value
	if tx.L1Timestamp() > ts {
		s.SetLatestL1Timestamp(tx.L1Timestamp())
	}

	index := s.GetLatestIndex()
	if tx.GetMeta().Index == nil {
		if index == nil {
			tx.SetIndex(0)
		} else {
			tx.SetIndex(*index + 1)
		}
	}

	// On restart, these values are repaired to handle
	// the case where the index is updated but the
	// transaction isn't yet added to the chain
	s.SetLatestIndex(tx.GetMeta().Index)
	if queueIndex := tx.GetMeta().QueueIndex; queueIndex != nil {
		s.SetLatestEnqueueIndex(queueIndex)
	}

	// The index was set above so it is safe to dereference
	log.Debug("Applying transaction to tip", "index", *tx.GetMeta().Index, "hash", tx.Hash().Hex(), "origin", tx.QueueOrigin().String())

	txs := types.Transactions{tx}
	errCh := make(chan error, 1)
	s.txFeed.Send(core.NewTxsEvent{
		Txs:   txs,
		ErrCh: errCh,
	})
	// Block until the transaction has been added to the chain
	log.Trace("Waiting for transaction to be added to chain", "hash", tx.Hash().Hex())

	select {
	case err := <-errCh:
		log.Error("Got error waiting for transaction to be added to chain", "msg", err)
		s.SetLatestL1Timestamp(ts)
		s.SetLatestL1BlockNumber(bn)
		s.SetLatestIndex(index)
		return err
	case <-s.chainHeadCh:
		// Update the cache when the transaction is from the owner
		// of the gas price oracle
		sender, _ := types.Sender(s.signer, tx)
		owner := s.GasPriceOracleOwnerAddress()
		if owner != nil && sender == *owner {
			if err := s.updateGasPriceOracleCache(nil); err != nil {
				s.SetLatestL1Timestamp(ts)
				s.SetLatestL1BlockNumber(bn)
				s.SetLatestIndex(index)
				return err
			}
		}
		return nil
	}
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

// verifyFee will verify that a valid fee is being paid.
func (s *SyncService) verifyFee(tx *types.Transaction) error {
	fee, err := fees.CalculateTotalFee(tx, s.RollupGpo)
	if err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	// Prevent transactions without enough balance from
	// being accepted by the chain but allow through 0
	// gas price transactions
	cost := tx.Value()
	if tx.GasPrice().Cmp(common.Big0) != 0 {
		cost = cost.Add(cost, fee)
	}
	state, err := s.bc.State()
	if err != nil {
		return err
	}
	from, err := types.Sender(s.signer, tx)
	if err != nil {
		return fmt.Errorf("invalid transaction: %w", core.ErrInvalidSender)
	}
	if state.GetBalance(from).Cmp(cost) < 0 {
		return fmt.Errorf("invalid transaction: %w", core.ErrInsufficientFunds)
	}

	if tx.GasPrice().Cmp(common.Big0) == 0 {
		// Allow 0 gas price transactions only if it is the owner of the gas
		// price oracle
		gpoOwner := s.GasPriceOracleOwnerAddress()
		if gpoOwner != nil {
			if from == *gpoOwner {
				return nil
			}
		}
		// Exit early if fees are enforced and the gasPrice is set to 0
		if s.enforceFees {
			return errZeroGasPriceTx
		}
		// If fees are not enforced and the gas price is 0, return early
		return nil
	}

	// Ensure that the user L2 gas price is high enough
	l2GasPrice, err := s.RollupGpo.SuggestL2GasPrice(context.Background())
	if err != nil {
		return err
	}

	// Reject user transactions that do not have large enough of a gas price.
	// Allow for a buffer in case the gas price changes in between the user
	// calling `eth_gasPrice` and submitting the transaction.
	opts := fees.PaysEnoughOpts{
		UserGasPrice:     tx.GasPrice(),
		ExpectedGasPrice: l2GasPrice,
		ThresholdUp:      s.feeThresholdUp,
		ThresholdDown:    s.feeThresholdDown,
	}

	// Check the error type and return the correct error message to the user
	if err := fees.PaysEnough(&opts); err != nil {
		if errors.Is(err, fees.ErrGasPriceTooLow) {
			return fmt.Errorf("%w: %d wei, use at least tx.gasPrice = %s wei",
				fees.ErrGasPriceTooLow, tx.GasPrice(), l2GasPrice)
		}
		if errors.Is(err, fees.ErrGasPriceTooHigh) {
			return fmt.Errorf("%w: %d wei, use at most tx.gasPrice = %s wei",
				fees.ErrGasPriceTooHigh, tx.GasPrice(), l2GasPrice)
		}
		return err
	}
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
	if err := s.verifyFee(tx); err != nil {
		return err
	}
	log.Trace("Sequencer transaction validation", "hash", tx.Hash().Hex())

	qo := tx.QueueOrigin()
	if qo != types.QueueOriginSequencer {
		return fmt.Errorf("invalid transaction with queue origin %s", qo.String())
	}
	if err := s.txpool.ValidateTx(tx); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}
	if err := s.applyTransaction(tx); err != nil {
		return err
	}
	return nil
}

// syncer represents a function that can sync remote items and then returns the
// index that it synced to as well as an error if it encountered one. It has
// side effects on the state and its functionality depends on the current state
type syncer func() (*uint64, error)

// rangeSyncer represents a function that syncs a range of items between its two
// arguments (inclusive)
type rangeSyncer func(uint64, uint64) error

// nextGetter is a type that represents a function that will return the next
// index
type nextGetter func() uint64

// indexGetter is a type that represents a function that returns an index and an
// error if there is a problem fetching the index. The different types of
// indices are canonical transaction chain indices, queue indices and batch
// indices. It does not induce side effects on state
type indexGetter func() (*uint64, error)

// isAtTip is a function that will determine if the local chain is at the tip
// of the remote datasource
func (s *SyncService) isAtTip(index *uint64, get indexGetter) (bool, error) {
	latest, err := get()
	if errors.Is(err, errElementNotFound) {
		if index == nil {
			return true, nil
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// There are no known enqueue transactions locally or remotely
	if latest == nil && index == nil {
		return true, nil
	}
	// Only one of the transactions are nil due to the check above so they
	// cannot be equal
	if latest == nil || index == nil {
		return false, nil
	}
	// The indices are equal
	if *latest == *index {
		return true, nil
	}
	// The local tip is greater than the remote tip. This should never happen
	if *latest < *index {
		return false, fmt.Errorf("is at tip mismatch: remote (%d) - local (%d): %w", *latest, *index, errShortRemoteTip)
	}
	// The indices are not equal
	return false, nil
}

// syncToTip is a function that can be used to sync to the tip of an ordered
// list of things. It is used to sync transactions, enqueue elements and batches
func (s *SyncService) syncToTip(sync syncer, getTip indexGetter) error {
	s.loopLock.Lock()
	defer s.loopLock.Unlock()

	for {
		index, err := sync()
		if errors.Is(err, errElementNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		isAtTip, err := s.isAtTip(index, getTip)
		if err != nil {
			return err
		}
		if isAtTip {
			return nil
		}
	}
}

// sync will sync a range of items
func (s *SyncService) sync(getLatest indexGetter, getNext nextGetter, syncer rangeSyncer) (*uint64, error) {
	latestIndex, err := getLatest()
	if err != nil {
		return nil, fmt.Errorf("Cannot sync: %w", err)
	}
	if latestIndex == nil {
		return nil, errors.New("Latest index is not defined")
	}

	nextIndex := getNext()
	if nextIndex == *latestIndex+1 {
		return latestIndex, nil
	}
	if err := syncer(nextIndex, *latestIndex); err != nil {
		return nil, err
	}
	return latestIndex, nil
}

// syncBatches will sync a range of batches from the current known tip to the
// remote tip.
func (s *SyncService) syncBatches() (*uint64, error) {
	index, err := s.sync(s.client.GetLatestTransactionBatchIndex, s.GetNextBatchIndex, s.syncTransactionBatchRange)
	if err != nil {
		return nil, fmt.Errorf("Cannot sync batches: %w", err)
	}
	return index, nil
}

// syncTransactionBatchRange will sync a range of batched transactions from
// start to end (inclusive)
func (s *SyncService) syncTransactionBatchRange(start, end uint64) error {
	log.Info("Syncing transaction batch range", "start", start, "end", end)
	for i := start; i <= end; i++ {
		log.Debug("Fetching transaction batch", "index", i)
		_, txs, err := s.client.GetTransactionBatch(i)
		if err != nil {
			return fmt.Errorf("Cannot get transaction batch: %w", err)
		}
		for _, tx := range txs {
			if err := s.applyBatchedTransaction(tx); err != nil {
				return fmt.Errorf("cannot apply batched transaction: %w", err)
			}
		}
		s.SetLatestBatchIndex(&i)
	}
	return nil
}

// syncQueue will sync from the local tip to the known tip of the remote
// enqueue transaction feed.
func (s *SyncService) syncQueue() (*uint64, error) {
	index, err := s.sync(s.client.GetLatestEnqueueIndex, s.GetNextEnqueueIndex, s.syncQueueTransactionRange)
	if err != nil {
		return nil, fmt.Errorf("Cannot sync queue: %w", err)
	}
	return index, nil
}

// syncQueueTransactionRange will apply a range of queue transactions from
// start to end (inclusive)
func (s *SyncService) syncQueueTransactionRange(start, end uint64) error {
	log.Info("Syncing enqueue transactions range", "start", start, "end", end)
	for i := start; i <= end; i++ {
		tx, err := s.client.GetEnqueue(i)
		if err != nil {
			return fmt.Errorf("Canot get enqueue transaction; %w", err)
		}
		if err := s.applyTransaction(tx); err != nil {
			return fmt.Errorf("Cannot apply transaction: %w", err)
		}
	}
	return nil
}

// syncTransactions will sync transactions to the remote tip based on the
// backend
func (s *SyncService) syncTransactions(backend Backend) (*uint64, error) {
	getLatest := func() (*uint64, error) {
		return s.client.GetLatestTransactionIndex(backend)
	}
	sync := func(start, end uint64) error {
		return s.syncTransactionRange(start, end, backend)
	}
	index, err := s.sync(getLatest, s.GetNextIndex, sync)
	if err != nil {
		return nil, fmt.Errorf("Cannot sync transactions with backend %s: %w", backend.String(), err)
	}
	return index, nil
}

// syncTransactionRange will sync a range of transactions from
// start to end (inclusive) from a specific Backend
func (s *SyncService) syncTransactionRange(start, end uint64, backend Backend) error {
	log.Info("Syncing transaction range", "start", start, "end", end, "backend", backend.String())
	for i := start; i <= end; i++ {
		tx, err := s.client.GetTransaction(i, backend)
		if err != nil {
			return fmt.Errorf("cannot fetch transaction %d: %w", i, err)
		}
		if err := s.applyTransaction(tx); err != nil {
			return fmt.Errorf("Cannot apply transaction: %w", err)
		}
	}
	return nil
}

// SubscribeNewTxsEvent registers a subscription of NewTxsEvent and
// starts sending event to the given channel.
func (s *SyncService) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return s.scope.Track(s.txFeed.Subscribe(ch))
}

func stringify(i *uint64) string {
	if i == nil {
		return "<nil>"
	}
	return strconv.FormatUint(*i, 10)
}

// IngestTransaction should only be called by trusted parties as it skips all
// validation and applies the transaction
func (s *SyncService) IngestTransaction(tx *types.Transaction) error {
	return s.applyTransaction(tx)
}
