package l2

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l2bridge"
	"github.com/ethereum-optimism/optimism/go/indexer/db"
	"github.com/ethereum-optimism/optimism/go/indexer/metrics"
	ethereum "github.com/ethereum-optimism/optimism/l2geth"
	"github.com/ethereum-optimism/optimism/l2geth/accounts/abi/bind"
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

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

type Backend interface {
	bind.ContractBackend
	HeaderBackend

	SubscribeNewHead(context.Context, chan<- *types.Header) (ethereum.Subscription, error)
	TransactionByHash(context.Context, common.Hash) (*types.Transaction, bool, error)
}

var (
	// weiToGwei is the conversion rate from wei to gwei.
	weiToGwei = new(big.Float).SetFloat64(1e-18)
)

func uint64ToBytes(i uint64) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], i)
	return buf[:]
}

func bytesToUint64(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// Merge function to add two uint64 numbers
func add(existing, new []byte) []byte {
	return uint64ToBytes(bytesToUint64(existing) + bytesToUint64(new))
}

func weiToGwei64(wei *big.Int) float64 {
	gwei := new(big.Float).SetInt(wei)
	gwei.Mul(gwei, weiToGwei)
	gwei64, _ := gwei.Float64()
	return gwei64
}

// Driver is an interface for indexing deposits from l1.
type Driver interface {
	// Name is an identifier used to prefix logs for a particular service.
	Name() string

	// Metrics returns the subservice telemetry object.
	Metrics() *metrics.Metrics
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

	l2BridgeContract *l2bridge.L2StandardBridgeFilterer
	backend          Backend
	headerSelector   HeaderSelector

	metrics *metrics.Metrics

	wg sync.WaitGroup
}

type IndexerStatus struct {
	Synced  float64           `json:"synced"`
	Highest db.L2BlockLocator `json:"highest_block"`
}

func NewService(cfg ServiceConfig) (*Service, error) {
	ctx, cancel := context.WithCancel(cfg.Context)

	contract, err := l2bridge.NewL2StandardBridgeFilterer(cfg.L2StandardBridgeAddress, cfg.L2Client)
	if err != nil {
		return nil, err
	}

	// Handle restart logic

	logger.Info("Creating L2 Indexer")

	chainID, err := cfg.L2Client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	if cfg.ChainID != nil {
		if cfg.ChainID.Cmp(chainID) != 0 {
			return nil, fmt.Errorf("%w: configured with %d and got %d",
				errWrongChainID, cfg.ChainID, chainID)
		}
	} else {
		cfg.ChainID = chainID
	}

	confirmedHeaderSelector, err := NewConfirmedHeaderSelector(HeaderSelectorConfig{
		ConfDepth:    cfg.ConfDepth,
		MaxBatchSize: cfg.MaxHeaderBatchSize,
	})

	if err != nil {
		return nil, err
	}

	return &Service{
		cfg:              cfg,
		ctx:              ctx,
		cancel:           cancel,
		l2BridgeContract: contract,
		headerSelector:   confirmedHeaderSelector,
		backend:          cfg.L2Client,
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

func (s *Service) fetchBlockEventIterator(start, end uint64) (
	*l2bridge.L2StandardBridgeWithdrawalInitiatedIterator, error) {

	const NUM_RETRIES = 5
	var err error
	for retry := 0; retry < NUM_RETRIES; retry++ {
		ctxt, cancel := context.WithTimeout(s.ctx, DefaultConnectionTimeout)

		var iter *l2bridge.L2StandardBridgeWithdrawalInitiatedIterator
		iter, err = s.l2BridgeContract.FilterWithdrawalInitiated(&bind.FilterOpts{
			Start:   start,
			End:     &end,
			Context: ctxt,
		}, nil, nil, nil)
		if err != nil {
			logger.Error("Unable to query withdrawal events for block range ",
				"start", start, "end", end, "error", err)
			cancel()
			continue
		}
		cancel()
		return iter, nil
	}
	return nil, err
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

	headers := s.headerSelector.NewHead(s.ctx, lowest.Number, newHeader, s.backend)
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

	iter, err := s.fetchBlockEventIterator(startHeight, endHeight)
	if err != nil {
		return err
	}

	withdrawalsByBlockhash := make(map[common.Hash][]db.Withdrawal)
	for iter.Next() {
		withdrawalsByBlockhash[iter.Event.Raw.BlockHash] = append(
			withdrawalsByBlockhash[iter.Event.Raw.BlockHash], db.Withdrawal{
				TxHash:      iter.Event.Raw.TxHash,
				L1Token:     iter.Event.L1Token,
				L2Token:     iter.Event.L2Token,
				FromAddress: iter.Event.From,
				ToAddress:   iter.Event.To,
				Amount:      iter.Event.Amount,
				Data:        iter.Event.Data,
				LogIndex:    iter.Event.Raw.Index,
			})
	}
	if err := iter.Error(); err != nil {
		return err
	}

	// Index L1 ERC20 tokens
	for _, withdrawals := range withdrawalsByBlockhash {
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

	latestHeader, err := s.cfg.L2Client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	synced := float64(highestBlock.Number) / float64(latestHeader.Number.Int64())

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
			header, err := s.cfg.L2Client.HeaderByNumber(ctx, nil)
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
	realHead, err := s.cfg.L2Client.HeaderByNumber(context.Background(), nil)
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
