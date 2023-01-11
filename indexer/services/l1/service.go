package l1

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/bindings/legacy/scc"
	"github.com/ethereum-optimism/optimism/indexer/metrics"
	"github.com/ethereum-optimism/optimism/indexer/services"
	"github.com/ethereum-optimism/optimism/indexer/services/query"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/indexer/server"
	"github.com/ethereum-optimism/optimism/indexer/services/l1/bridge"

	_ "github.com/lib/pq"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/mux"
)

var logger = log.New("service", "l1")

// errNoChainID represents the error when the chain id is not provided
// and it cannot be remotely fetched
var errNoChainID = errors.New("no chain id provided")

var errNoNewBlocks = errors.New("no new blocks")

var ZeroAddress common.Address

// Driver is an interface for indexing deposits from l1.
type Driver interface {
	// Name is an identifier used to prefix logs for a particular service.
	Name() string
}

type ServiceConfig struct {
	Context            context.Context
	Metrics            *metrics.Metrics
	L1Client           *ethclient.Client
	RawL1Client        *rpc.Client
	ChainID            *big.Int
	AddressManager     services.AddressManager
	ConfDepth          uint64
	MaxHeaderBatchSize uint64
	StartBlockNumber   uint64
	DB                 *db.Database
	Bedrock            bool
}

type Service struct {
	cfg    ServiceConfig
	ctx    context.Context
	cancel func()

	bridges        map[string]bridge.Bridge
	portal         *bridge.Portal
	batchScanner   *scc.StateCommitmentChainFilterer
	latestHeader   uint64
	headerSelector *ConfirmedHeaderSelector
	l1Client       *ethclient.Client

	metrics    *metrics.Metrics
	tokenCache map[common.Address]*db.Token
	isBedrock  bool
	wg         sync.WaitGroup
}

type IndexerStatus struct {
	Synced  float64         `json:"synced"`
	Highest db.BlockLocator `json:"highest_block"`
}

func NewService(cfg ServiceConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(cfg.Context)

	// Handle restart logic

	logger.Info("Creating L1 Indexer")

	chainID, err := cfg.L1Client.ChainID(context.Background())
	if err != nil {
		cancel()
		return nil, err
	}
	if cfg.ChainID.Cmp(chainID) != 0 {
		cancel()
		return nil, fmt.Errorf("chain ID configured with %d but got %d", cfg.ChainID, chainID)
	}

	bridges, err := bridge.BridgesByChainID(cfg.ChainID, cfg.L1Client, cfg.AddressManager)
	if err != nil {
		cancel()
		return nil, err
	}

	var portal *bridge.Portal
	var batchScanner *scc.StateCommitmentChainFilterer
	if cfg.Bedrock {
		portal = bridge.NewPortal(cfg.AddressManager)
	} else {
		batchScanner, err = bridge.StateCommitmentChainScanner(cfg.L1Client, cfg.AddressManager)
		if err != nil {
			cancel()
			return nil, err
		}
	}

	logger.Info("Scanning bridges for deposits", "bridges", bridges)

	confirmedHeaderSelector, err := NewConfirmedHeaderSelector(HeaderSelectorConfig{
		ConfDepth:    cfg.ConfDepth,
		MaxBatchSize: cfg.MaxHeaderBatchSize,
	})

	if err != nil {
		cancel()
		return nil, err
	}

	service := &Service{
		cfg:            cfg,
		ctx:            ctx,
		cancel:         cancel,
		portal:         portal,
		bridges:        bridges,
		batchScanner:   batchScanner,
		headerSelector: confirmedHeaderSelector,
		metrics:        cfg.Metrics,
		tokenCache: map[common.Address]*db.Token{
			ZeroAddress: db.ETHL1Token,
		},
		isBedrock: cfg.Bedrock,
		l1Client:  cfg.L1Client,
	}
	service.wg.Add(1)
	return service, nil
}

func (s *Service) loop() {
	defer s.wg.Done()

	for {
		err := s.catchUp()
		if err == nil {
			break
		}
		if err == context.Canceled {
			return
		}

		logger.Error("error catching up to tip, trying again in a bit", "err", err)
		time.Sleep(10 * time.Second)
		continue
	}

	newHeads := make(chan *types.Header, 1000)
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			header, err := query.HeaderByNumberWithRetry(s.ctx, s.cfg.L1Client)
			if err != nil {
				logger.Error("error fetching header by number", "err", err)
				continue
			}
			newHeads <- header
		case header := <-newHeads:
			if header == nil {
				break
			}

			logger.Info("Received new header", "header", header.Hash)
			atomic.StoreUint64(&s.latestHeader, header.Number.Uint64())
			for {
				err := s.Update(header)
				if err != nil {
					if err != errNoNewBlocks {
						logger.Error("Unable to update indexer ", "err", err)
					}
					break
				}
			}
		case <-s.ctx.Done():
			logger.Info("service stopped")
			return
		}
	}
}

func (s *Service) Update(newHeader *types.Header) error {
	var lowest db.BlockLocator
	highestConfirmed, err := s.cfg.DB.GetHighestL1Block()
	if err != nil {
		return err
	}
	if highestConfirmed == nil {
		startHeader, err := s.l1Client.HeaderByNumber(s.ctx, new(big.Int).SetUint64(s.cfg.StartBlockNumber))
		if err != nil {
			return fmt.Errorf("error fetching header by number: %w", err)
		}
		highestConfirmed = &db.BlockLocator{
			Number: s.cfg.StartBlockNumber,
			Hash:   startHeader.Hash(),
		}
	}
	lowest = *highestConfirmed

	headers, err := s.headerSelector.NewHead(s.ctx, lowest.Number, newHeader, s.cfg.RawL1Client)
	if err != nil {
		return err
	}
	if len(headers) == 0 {
		return errNoNewBlocks
	}

	if lowest.Number+1 != headers[0].Number.Uint64() {
		logger.Error("Block number does not immediately follow ",
			"block", headers[0].Number.Uint64(), "hash", headers[0].Hash,
			"lowest_block", lowest.Number, "hash", lowest.Hash)
		return nil
	}

	if lowest.Number > 0 && lowest.Hash != headers[0].ParentHash {
		logger.Error("Parent hash does not connect to ",
			"block", headers[0].Number.Uint64(), "hash", headers[0].Hash,
			"lowest_block", lowest.Number, "hash", lowest.Hash)
		return nil
	}

	startHeight := headers[0].Number.Uint64()
	endHeight := headers[len(headers)-1].Number.Uint64()
	depositsByBlockHash := make(map[common.Hash][]db.Deposit)

	start := prometheus.NewTimer(s.metrics.UpdateDuration.WithLabelValues("l1"))
	defer func() {
		dur := start.ObserveDuration()
		logger.Info("updated index", "start_height", startHeight, "end_height", endHeight, "duration", dur)
	}()

	bridgeDepositsCh := make(chan bridge.DepositsMap, len(s.bridges))
	provenWithdrawalsCh := make(chan bridge.ProvenWithdrawalsMap, 1)
	finalizedWithdrawalsCh := make(chan bridge.FinalizedWithdrawalsMap, 1)
	errCh := make(chan error, len(s.bridges)+1)

	for _, bridgeImpl := range s.bridges {
		go func(b bridge.Bridge) {
			deposits, err := b.GetDepositsByBlockRange(s.ctx, startHeight, endHeight)
			if err != nil {
				errCh <- err
				return
			}
			bridgeDepositsCh <- deposits
		}(bridgeImpl)
	}

	if s.isBedrock {
		go func() {
			provenWithdrawals, err := s.portal.GetProvenWithdrawalsByBlockRange(s.ctx, startHeight, endHeight)
			if err != nil {
				errCh <- err
				return
			}
			provenWithdrawalsCh <- provenWithdrawals
		}()
		go func() {
			finalizedWithdrawals, err := s.portal.GetFinalizedWithdrawalsByBlockRange(s.ctx, startHeight, endHeight)
			if err != nil {
				errCh <- err
				return
			}
			finalizedWithdrawalsCh <- finalizedWithdrawals
		}()
	} else {
		provenWithdrawalsCh <- make(bridge.ProvenWithdrawalsMap)
		finalizedWithdrawalsCh <- make(bridge.FinalizedWithdrawalsMap)
	}

	var receives int
	for {
		select {
		case bridgeDeposits := <-bridgeDepositsCh:
			for blockHash, deposits := range bridgeDeposits {
				for _, deposit := range deposits {
					if err := s.cacheToken(deposit); err != nil {
						logger.Warn("error caching token", "err", err)
					}
				}

				depositsByBlockHash[blockHash] = append(depositsByBlockHash[blockHash], deposits...)
			}
		case err := <-errCh:
			return err
		}

		receives++
		if receives == len(s.bridges) {
			break
		}
	}

	provenWithdrawalsByBlockHash := <-provenWithdrawalsCh
	finalizedWithdrawalsByBlockHash := <-finalizedWithdrawalsCh

	var stateBatches map[common.Hash][]db.StateBatch
	if !s.isBedrock {
		stateBatches, err = QueryStateBatches(s.batchScanner, startHeight, endHeight, s.ctx)
		if err != nil {
			logger.Error("Error querying state batches", "err", err)
			return err
		}
	}

	for i, header := range headers {
		blockHash := header.Hash
		number := header.Number.Uint64()
		deposits := depositsByBlockHash[blockHash]
		batches := stateBatches[blockHash]
		provenWds := provenWithdrawalsByBlockHash[blockHash]
		finalizedWds := finalizedWithdrawalsByBlockHash[blockHash]

		// Always record block data in the last block
		// in the list of headers
		if len(deposits) == 0 && len(batches) == 0 && len(provenWds) == 0 && len(finalizedWds) == 0 && i != len(headers)-1 {
			continue
		}

		block := &db.IndexedL1Block{
			Hash:                 blockHash,
			ParentHash:           header.ParentHash,
			Number:               number,
			Timestamp:            header.Time,
			Deposits:             deposits,
			ProvenWithdrawals:    provenWds,
			FinalizedWithdrawals: finalizedWds,
		}

		err := s.cfg.DB.AddIndexedL1Block(block)
		if err != nil {
			logger.Error(
				"Unable to import ",
				"block", number,
				"hash", blockHash, "err", err,
				"block", block,
			)
			return err
		}

		err = s.cfg.DB.AddStateBatch(batches)
		if err != nil {
			logger.Error(
				"Unable to import state append batch",
				"block", number,
				"hash", blockHash, "err", err,
				"block", block,
			)
			return err
		}
		s.metrics.RecordStateBatches(len(batches))

		logger.Debug("Imported ",
			"block", number, "hash", blockHash, "deposits", len(block.Deposits))
		for _, deposit := range block.Deposits {
			token := s.tokenCache[deposit.L1Token]
			logger.Info(
				"indexed deposit",
				"tx_hash", deposit.TxHash,
				"symbol", token.Symbol,
				"amount", deposit.Amount,
			)
			s.metrics.RecordDeposit(deposit.L1Token)
		}
	}

	newHeaderNumber := newHeader.Number.Uint64()
	s.metrics.SetL1SyncHeight(endHeight)
	s.metrics.SetL1SyncPercent(endHeight, newHeaderNumber)
	latestHeaderNumber := headers[len(headers)-1].Number.Uint64()
	if latestHeaderNumber+s.cfg.ConfDepth-1 == newHeaderNumber {
		return errNoNewBlocks
	}
	return nil
}

func (s *Service) GetIndexerStatus(w http.ResponseWriter, r *http.Request) {
	highestBlock, err := s.cfg.DB.GetHighestL1Block()
	if err != nil {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var synced float64
	if s.latestHeader != 0 {
		synced = float64(highestBlock.Number) / float64(s.latestHeader)
	}

	status := &IndexerStatus{
		Synced:  synced,
		Highest: *highestBlock,
	}

	server.RespondWithJSON(w, http.StatusOK, status)
}

func (s *Service) GetDeposits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.ParseUint(limitStr, 10, 64)
	if err != nil && limitStr != "" {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if limit == 0 {
		limit = 10
	}

	offsetStr := r.URL.Query().Get("offset")
	offset, err := strconv.ParseUint(offsetStr, 10, 64)
	if err != nil && offsetStr != "" {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	page := db.PaginationParam{
		Limit:  limit,
		Offset: offset,
	}

	deposits, err := s.cfg.DB.GetDepositsByAddress(common.HexToAddress(vars["address"]), page)
	if err != nil {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondWithJSON(w, http.StatusOK, deposits)
}

func (s *Service) catchUp() error {
	realHead, err := query.HeaderByNumberWithRetry(s.ctx, s.cfg.L1Client)
	if err != nil {
		return err
	}
	realHeadNum := realHead.Number.Uint64()

	currHead, err := s.cfg.DB.GetHighestL1Block()
	if err != nil {
		return err
	}
	var currHeadNum uint64
	if currHead != nil {
		currHeadNum = currHead.Number
	}

	if realHeadNum-s.cfg.ConfDepth <= currHeadNum+s.cfg.MaxHeaderBatchSize {
		return nil
	}

	logger.Info("chain is far behind head, resyncing")
	s.metrics.SetL1CatchingUp(true)

	for realHeadNum-s.cfg.ConfDepth > currHeadNum+s.cfg.MaxHeaderBatchSize {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
			if err := s.Update(realHead); err != nil && err != errNoNewBlocks {
				return err
			}
			currHead, err := s.cfg.DB.GetHighestL1Block()
			if err != nil {
				return err
			}
			currHeadNum = currHead.Number
		}
	}

	logger.Info("indexer is close enough to tip, starting regular loop")
	s.metrics.SetL1CatchingUp(false)
	return nil
}

func (s *Service) cacheToken(deposit db.Deposit) error {
	if s.tokenCache[deposit.L1Token] != nil {
		return nil
	}

	token, err := s.cfg.DB.GetL1TokenByAddress(deposit.L1Token.String())
	if err != nil {
		return err
	}
	if token != nil {
		s.metrics.IncL1CachedTokensCount()
		s.tokenCache[deposit.L1Token] = token
		return nil
	}

	token, err = query.NewERC20(deposit.L1Token, s.cfg.L1Client)
	if err != nil {
		logger.Error("Error querying ERC20 token details",
			"l1_token", deposit.L1Token.String(), "err", err)
		token = &db.Token{
			Address: deposit.L1Token.String(),
		}
	}
	if err := s.cfg.DB.AddL1Token(deposit.L1Token.String(), token); err != nil {
		return err
	}
	s.tokenCache[deposit.L1Token] = token
	s.metrics.IncL1CachedTokensCount()
	return nil
}

func (s *Service) Start() error {
	if s.cfg.ChainID == nil {
		return errNoChainID
	}
	go s.loop()
	return nil
}

func (s *Service) Stop() {
	s.cancel()
	s.wg.Wait()
}
