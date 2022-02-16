package l2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum-optimism/optimism/go/indexer/metrics"
	"github.com/ethereum-optimism/optimism/go/indexer/services/l2/bridge"
	"github.com/ethereum-optimism/optimism/l2geth/common"
	l2common "github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum-optimism/optimism/l2geth/core/types"
	l2ethclient "github.com/ethereum-optimism/optimism/l2geth/ethclient"
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

// clientRetryInterval is the interval to wait between retrying client API
// calls.
var clientRetryInterval = 5 * time.Second

// HeaderByNumberWithRetry retries the given func until it succeeds, waiting
// for clientRetryInterval duration after every call.
func HeaderByNumberWithRetry(ctx context.Context,
	client *l2ethclient.Client) (*types.Header, error) {
	for {
		res, err := client.HeaderByNumber(ctx, nil)
		switch err {
		case nil:
			return res, err
		default:
			log.Error("Error fetching header", "err", err)
			break
		}
		time.Sleep(clientRetryInterval)
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

type ServiceConfig struct {
	Context                 context.Context
	L2Client                *l2ethclient.Client
	ChainID                 *big.Int
	L2StandardBridgeAddress l2common.Address
	ConfDepth               uint64
	MaxHeaderBatchSize      uint64
	StartBlockNumber        uint64
	StartBlockHash          string
	DB                      *db.Database
}

type Service struct {
	cfg    ServiceConfig
	ctx    context.Context
	cancel func()

	bridges        map[string]bridge.Bridge
	latestHeader   uint64
	headerSelector *ConfirmedHeaderSelector

	metrics *metrics.Metrics

	wg sync.WaitGroup
}

type IndexerStatus struct {
	Synced  float64           `json:"synced"`
	Highest db.L2BlockLocator `json:"highest_block"`
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

	bridges, err := bridge.BridgesByChainID(cfg.ChainID, cfg.L2Client, ctx)
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

	return &Service{
		cfg:            cfg,
		ctx:            ctx,
		cancel:         cancel,
		bridges:        bridges,
		headerSelector: confirmedHeaderSelector,
	}, nil
}

func (s *Service) Loop(ctx context.Context) {
	if err := s.catchUp(ctx); err != nil {
		if err == context.Canceled {
			return
		}

		logger.Error("error catching up to tip, trying to subscribe anyway", "err", err)
	}

	newHeads := make(chan *types.Header, 1000)
	go s.subscribeNewHeads(ctx, newHeads)

	for {
		select {
		case header := <-newHeads:
			logger.Info("Received new header", "header", header.Hash)
			for {
				err := s.Update(header)
				if err != nil && err != errNoNewBlocks {
					logger.Error("Unable to update indexer ", "err", err)
				}
				break
			}
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *Service) Update(newHeader *types.Header) error {
	var lowest = db.L2BlockLocator{
		Number: s.cfg.StartBlockNumber,
		Hash:   common.HexToHash(s.cfg.StartBlockHash),
	}
	highestConfirmed, err := s.cfg.DB.GetHighestL2Block()
	if err != nil {
		return err
	}
	if highestConfirmed != nil {
		lowest = *highestConfirmed
	}

	headers := s.headerSelector.NewHead(s.ctx, lowest.Number, newHeader, s.cfg.L2Client)
	if len(headers) == 0 {
		return errNoNewBlocks
	}

	if lowest.Number+1 != headers[0].Number.Uint64() {
		logger.Error("Block number does not immediately follow ",
			"block", headers[0].Number.Uint64(), "hash", headers[0].Hash(),
			"lowest_block", lowest.Number, "hash", lowest.Hash)
		return nil
	}

	if lowest.Hash != headers[0].ParentHash {
		logger.Error("Parent hash does not connect to ",
			"block", headers[0].Number.Uint64(), "hash", headers[0].Hash(),
			"lowest_block", lowest.Number, "hash", lowest.Hash)
		return nil
	}

	startHeight := headers[0].Number.Uint64()
	endHeight := headers[len(headers)-1].Number.Uint64()
	withdrawalsByBlockhash := make(map[common.Hash][]db.Withdrawal)

	for _, bridgeImpl := range s.bridges {
		bridgeWithdrawals, err := bridgeImpl.GetWithdrawalsByBlockRange(startHeight, endHeight)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		// ERC20 withdrawals l2_token needs to be indexed before they can be
		// inserted, because l2_token is a foreign key to the token metadata
		switch bridgeImpl.(type) {
		case *bridge.StandardBridge:
			// Index L2 ERC20 tokens
			for _, withdrawals := range bridgeWithdrawals {
				for _, withdrawal := range withdrawals {
					token, err := s.cfg.DB.GetL2TokenByAddress(withdrawal.L2Token.String())
					if err != nil {
						return err
					}
					if token != nil {
						continue
					}
					token, err = QueryERC20(withdrawal.L2Token, s.cfg.L2Client)
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
				}
			}
		}

		for blockHash, withdrawals := range bridgeWithdrawals {
			withdrawalsByBlockhash[blockHash] = append(withdrawalsByBlockhash[blockHash], withdrawals...)
		}
	}

	for _, header := range headers {
		blockHash := header.Hash()
		number := header.Number.Uint64()
		withdrawals := withdrawalsByBlockhash[blockHash]

		block := &db.IndexedL2Block{
			Hash:        blockHash,
			ParentHash:  header.ParentHash,
			Number:      number,
			Timestamp:   header.Time,
			Withdrawals: withdrawals,
		}

		err := s.cfg.DB.AddIndexedL2Block(block)
		if err != nil {
			logger.Error("Unable to import ",
				"block", number, "hash", blockHash, "err", err, "block", block)
			return err
		}

		logger.Debug("Imported ",
			"block", number, "hash", blockHash, "withdrawals", len(block.Withdrawals))
		for _, withdrawal := range block.Withdrawals {
			logger.Info("Indexed withdrawal ", "tx_hash", withdrawal.TxHash)
		}
	}

	latestHeaderNumber := headers[len(headers)-1].Number.Uint64()
	newHeaderNumber := newHeader.Number.Uint64()
	if latestHeaderNumber+s.cfg.ConfDepth-1 == newHeaderNumber {
		return errNoNewBlocks
	}
	return nil
}

func (s *Service) GetIndexerStatus(w http.ResponseWriter, r *http.Request) {
	highestBlock, err := s.cfg.DB.GetHighestL2Block()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
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

	respondWithJSON(w, http.StatusOK, status)
}

func (s *Service) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.ParseUint(limitStr, 10, 64)
	if err != nil && limitStr != "" {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if limit == 0 {
		limit = 10
	}

	offsetStr := r.URL.Query().Get("offset")
	offset, err := strconv.ParseUint(offsetStr, 10, 64)
	if err != nil && offsetStr != "" {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	page := db.PaginationParam{
		Limit:  uint64(limit),
		Offset: uint64(offset),
	}

	withdrawals, err := s.cfg.DB.GetWithdrawalsByAddress(common.HexToAddress(vars["address"]), page)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, withdrawals)
}

func (s *Service) subscribeNewHeads(ctx context.Context, heads chan *types.Header) {
	tick := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-tick.C:
			header, err := HeaderByNumberWithRetry(ctx, s.cfg.L2Client)
			if err != nil {
				logger.Error("error fetching header by number", "err", err)
			}
			heads <- header
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) catchUp(ctx context.Context) error {
	realHead, err := HeaderByNumberWithRetry(ctx, s.cfg.L2Client)
	if err != nil {
		return err
	}

	currHead, err := s.cfg.DB.GetHighestL2Block()
	if err != nil {
		return err
	}

	realHeadNum := realHead.Number.Uint64()
	var currHeadNum uint64
	if currHead != nil {
		currHeadNum = currHead.Number
	}

	if realHeadNum-s.cfg.ConfDepth <= currHeadNum+s.cfg.MaxHeaderBatchSize {
		return nil
	}

	logger.Info("chain is far behind head, resyncing")
	for realHeadNum-s.cfg.ConfDepth > currHeadNum+s.cfg.MaxHeaderBatchSize {
		select {
		case <-ctx.Done():
			return context.Canceled
		default:
			if err := s.Update(realHead); err != nil {
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
	return nil
}

func (s *Service) Start() error {
	if s.cfg.ChainID == nil {
		return errNoChainID
	}
	s.wg.Add(1)
	go s.Loop(s.ctx)
	return nil
}

func (s *Service) Stop() error {
	s.cancel()
	s.wg.Wait()
	if err := s.cfg.DB.Close(); err != nil {
		return err
	}
	return nil
}
