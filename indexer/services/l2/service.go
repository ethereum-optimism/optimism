package l2

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/metrics"
	"github.com/ethereum-optimism/optimism/indexer/server"
	"github.com/ethereum-optimism/optimism/indexer/services/query"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/indexer/db"
	"github.com/ethereum-optimism/optimism/indexer/services/l2/bridge"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
)

var logger = log.New("service", "l2")

// errNoChainID represents the error when the chain id is not provided
// and it cannot be remotely fetched
var errNoChainID = errors.New("no chain id provided")

// errWrongChainID represents the error when the configured chain id is not
// correct
var errWrongChainID = errors.New("wrong chain id provided")

var errNoNewBlocks = errors.New("no new blocks")

type ServiceConfig struct {
	Context  context.Context
	Metrics  *metrics.Metrics
	L2RPC    *rpc.Client
	L2Client *ethclient.Client
	ChainID  *big.Int

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
	latestHeader   uint64
	headerSelector *ConfirmedHeaderSelector

	metrics    *metrics.Metrics
	tokenCache map[common.Address]*db.Token
	wg         sync.WaitGroup
}

type IndexerStatus struct {
	Synced  float64         `json:"synced"`
	Highest db.BlockLocator `json:"highest_block"`
}

func NewService(cfg ServiceConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(cfg.Context)

	// Handle restart logic

	logger.Info("Creating L2 Indexer")

	chainID, err := cfg.L2Client.ChainID(context.Background())
	if err != nil {
		cancel()
		return nil, err
	}

	if cfg.ChainID != nil {
		if cfg.ChainID.Cmp(chainID) != 0 {
			cancel()
			return nil, fmt.Errorf("%w: configured with %d and got %d",
				errWrongChainID, cfg.ChainID, chainID)
		}
	} else {
		cfg.ChainID = chainID
	}

	bridges, err := bridge.BridgesByChainID(cfg.ChainID, cfg.L2Client, cfg.Bedrock)
	if err != nil {
		cancel()
		return nil, err
	}

	logger.Info("Scanning bridges for withdrawals", "bridges", bridges)

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
		bridges:        bridges,
		headerSelector: confirmedHeaderSelector,
		metrics:        cfg.Metrics,
		tokenCache: map[common.Address]*db.Token{
			predeploys.LegacyERC20ETHAddr: db.ETHL1Token,
		},
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
			header, err := query.HeaderByNumberWithRetry(s.ctx, s.cfg.L2Client)
			if err != nil {
				logger.Error("error fetching header by number", "err", err)
				continue
			}
			newHeads <- header
		case header := <-newHeads:
			logger.Info("Received new header", "header", header.Hash)
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
	var lowest = db.BlockLocator{
		Number: s.cfg.StartBlockNumber,
	}
	highestConfirmed, err := s.cfg.DB.GetHighestL2Block()
	if err != nil {
		return err
	}
	if highestConfirmed != nil {
		lowest = *highestConfirmed
	}

	headers, err := s.headerSelector.NewHead(s.ctx, lowest.Number, newHeader, s.cfg.L2RPC)
	if err != nil {
		return err
	}
	if len(headers) == 0 {
		return errNoNewBlocks
	}

	if lowest.Number+1 != headers[0].Number.Uint64() {
		logger.Error("Block number does not immediately follow ",
			"block", headers[0].Number.Uint64(), "hash", headers[0].Hash(),
			"lowest_block", lowest.Number, "hash", lowest.Hash)
		return nil
	}

	if lowest.Number > 0 && lowest.Hash != headers[0].ParentHash {
		logger.Error("Parent hash does not connect to ",
			"block", headers[0].Number.Uint64(), "hash", headers[0].Hash(),
			"lowest_block", lowest.Number, "hash", lowest.Hash)
		return nil
	}

	startHeight := headers[0].Number.Uint64()
	endHeight := headers[len(headers)-1].Number.Uint64()
	withdrawalsByBlockHash := make(map[common.Hash][]db.Withdrawal)

	start := prometheus.NewTimer(s.metrics.UpdateDuration.WithLabelValues("l2"))
	defer func() {
		dur := start.ObserveDuration()
		logger.Info("updated index", "start_height", startHeight, "end_height", endHeight, "duration", dur)
	}()

	bridgeWdsCh := make(chan bridge.WithdrawalsMap)
	errCh := make(chan error, len(s.bridges))

	for _, bridgeImpl := range s.bridges {
		go func(b bridge.Bridge) {
			wds, err := b.GetWithdrawalsByBlockRange(s.ctx, startHeight, endHeight)
			if err != nil {
				errCh <- err
				return
			}
			bridgeWdsCh <- wds
		}(bridgeImpl)
	}

	var receives int
	for {
		select {
		case bridgeWds := <-bridgeWdsCh:
			for blockHash, withdrawals := range bridgeWds {
				for _, wd := range withdrawals {
					if err := s.cacheToken(wd); err != nil {
						logger.Warn("error caching token", "err", err)
					}
				}

				withdrawalsByBlockHash[blockHash] = append(withdrawalsByBlockHash[blockHash], withdrawals...)
			}
		case err := <-errCh:
			return err
		}

		receives++
		if receives == len(s.bridges) {
			break
		}
	}

	for i, header := range headers {
		blockHash := header.Hash()
		number := header.Number.Uint64()
		withdrawals := withdrawalsByBlockHash[blockHash]

		if len(withdrawals) == 0 && i != len(headers)-1 {
			continue
		}

		block := &db.IndexedL2Block{
			Hash:        blockHash,
			ParentHash:  header.ParentHash,
			Number:      number,
			Timestamp:   header.Time,
			Withdrawals: withdrawals,
		}

		err := s.cfg.DB.AddIndexedL2Block(block)
		if err != nil {
			logger.Error(
				"Unable to import ",
				"block", number,
				"hash", blockHash,
				"err", err,
				"block", block,
			)
			return err
		}

		logger.Debug("Imported ",
			"block", number, "hash", blockHash, "withdrawals", len(block.Withdrawals))
		for _, withdrawal := range block.Withdrawals {
			token := s.tokenCache[withdrawal.L2Token]
			logger.Info(
				"indexed withdrawal ",
				"tx_hash", withdrawal.TxHash,
				"symbol", token.Symbol,
				"amount", withdrawal.Amount,
			)
			s.metrics.RecordWithdrawal(withdrawal.L2Token)
		}
	}

	newHeaderNumber := newHeader.Number.Uint64()
	s.metrics.SetL2SyncHeight(endHeight)
	s.metrics.SetL2SyncPercent(endHeight, newHeaderNumber)
	latestHeaderNumber := headers[len(headers)-1].Number.Uint64()
	if latestHeaderNumber+s.cfg.ConfDepth-1 == newHeaderNumber {
		return errNoNewBlocks
	}
	return nil
}

func (s *Service) GetIndexerStatus(w http.ResponseWriter, r *http.Request) {
	highestBlock, err := s.cfg.DB.GetHighestL2Block()
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

func (s *Service) GetWithdrawalBatch(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["hash"]
	if hash == "" {
		server.RespondWithError(w, http.StatusBadRequest, "must specify a hash")
		return
	}

	batch, err := s.cfg.DB.GetWithdrawalBatch(common.HexToHash(vars["hash"]))
	if err != nil {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondWithJSON(w, http.StatusOK, batch)
}

func (s *Service) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
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

	finalizationState := db.ParseFinalizationState(r.URL.Query().Get("finalized"))

	page := db.PaginationParam{
		Limit:  limit,
		Offset: offset,
	}

	withdrawals, err := s.cfg.DB.GetWithdrawalsByAddress(common.HexToAddress(vars["address"]), page, finalizationState)
	if err != nil {
		server.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.RespondWithJSON(w, http.StatusOK, withdrawals)
}

func (s *Service) catchUp() error {
	realHead, err := query.HeaderByNumberWithRetry(s.ctx, s.cfg.L2Client)
	if err != nil {
		return err
	}
	realHeadNum := realHead.Number.Uint64()

	currHead, err := s.cfg.DB.GetHighestL2Block()
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
	s.metrics.SetL2CatchingUp(true)

	for realHeadNum-s.cfg.ConfDepth > currHeadNum+s.cfg.MaxHeaderBatchSize {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
			if err := s.Update(realHead); err != nil && err != errNoNewBlocks {
				return err
			}
			currHead, err := s.cfg.DB.GetHighestL2Block()
			if err != nil {
				return err
			}
			currHeadNum = currHead.Number
		}
	}

	logger.Info("indexer is close enough to tip, starting regular loop")
	s.metrics.SetL2CatchingUp(false)
	return nil
}

func (s *Service) cacheToken(withdrawal db.Withdrawal) error {
	if s.tokenCache[withdrawal.L2Token] != nil {
		return nil
	}

	token, err := s.cfg.DB.GetL2TokenByAddress(withdrawal.L2Token.String())
	if err != nil {
		return err
	}
	if token != nil {
		s.metrics.IncL2CachedTokensCount()
		s.tokenCache[withdrawal.L2Token] = token
		return nil
	}
	token, err = query.NewERC20(withdrawal.L2Token, s.cfg.L2Client)
	if err != nil {
		logger.Error("Error querying ERC20 token details",
			"l2_token", withdrawal.L2Token.String(), "err", err)
		token = &db.Token{
			Address: withdrawal.L2Token.String(),
		}
	}
	if err := s.cfg.DB.AddL2Token(withdrawal.L2Token.String(), token); err != nil {
		return err
	}
	s.tokenCache[withdrawal.L2Token] = token
	s.metrics.IncL2CachedTokensCount()
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
