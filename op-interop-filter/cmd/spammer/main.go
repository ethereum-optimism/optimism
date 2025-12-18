package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

var (
	Version   = "v0.0.1"
	GitCommit = ""
	GitDate   = ""
)

var (
	L2RPCFlag = &cli.StringFlag{
		Name:     "l2-rpc",
		Usage:    "L2 RPC endpoint URL",
		Required: true,
		EnvVars:  []string{"L2_RPC"},
	}
	FilterRPCFlag = &cli.StringFlag{
		Name:     "filter-rpc",
		Usage:    "Filter service RPC endpoint URL",
		Required: true,
		EnvVars:  []string{"FILTER_RPC"},
	}
	ChainIDFlag = &cli.Uint64Flag{
		Name:     "chain-id",
		Usage:    "Chain ID to test",
		Required: true,
		EnvVars:  []string{"CHAIN_ID"},
	}
	NumQueriesFlag = &cli.IntFlag{
		Name:    "num-queries",
		Usage:   "Number of queries to run (0 = run forever)",
		Value:   100,
		EnvVars: []string{"NUM_QUERIES"},
	}
	BlockRangeFlag = &cli.IntFlag{
		Name:    "block-range",
		Usage:   "Number of recent blocks to sample from",
		Value:   100,
		EnvVars: []string{"BLOCK_RANGE"},
	}
	QueryIntervalFlag = &cli.StringFlag{
		Name:    "query-interval",
		Usage:   "Interval between queries (e.g., 100ms, 1s)",
		Value:   "500ms",
		EnvVars: []string{"QUERY_INTERVAL"},
	}
	MetricsPortFlag = &cli.IntFlag{
		Name:    "metrics.port",
		Usage:   "Port for Prometheus metrics server",
		Value:   7301,
		EnvVars: []string{"METRICS_PORT"},
	}
	MetricsEnabledFlag = &cli.BoolFlag{
		Name:    "metrics.enabled",
		Usage:   "Enable Prometheus metrics",
		Value:   true,
		EnvVars: []string{"METRICS_ENABLED"},
	}
	SafetyLevelFlag = &cli.StringFlag{
		Name:    "safety-level",
		Usage:   "Safety level to use for queries (unsafe or cross-unsafe)",
		Value:   "cross-unsafe",
		EnvVars: []string{"SAFETY_LEVEL"},
	}
	JWTSecretFlag = &cli.StringFlag{
		Name:    "jwt-secret",
		Usage:   "Path to JWT secret for authenticated admin RPC endpoints (optional)",
		EnvVars: []string{"JWT_SECRET"},
	}
)

// Metrics for the spammer
type SpammerMetrics struct {
	queriesTotal      *prometheus.CounterVec
	queryLatency      *prometheus.HistogramVec
	errorsTotal       prometheus.Counter
	uptime            prometheus.Gauge
	currentBlockRange prometheus.Gauge
}

// RecentQuery stores info about a recent query for debugging
type RecentQuery struct {
	Timestamp   string      `json:"timestamp"`
	ChainID     eth.ChainID `json:"chain_id"`
	BlockNumber uint64      `json:"block_number"`
	TxHash      string      `json:"tx_hash"`
	LogIndex    uint        `json:"log_index"`
	Address     string      `json:"address"`
	QueryType   string      `json:"query_type"` // "valid" or "invalid"
	Result      string      `json:"result"`     // "accepted" or "rejected"
}

// RecentQueries is a thread-safe ring buffer of recent queries
type RecentQueries struct {
	mu      sync.RWMutex
	queries []RecentQuery
	maxSize int
}

func NewRecentQueries(size int) *RecentQueries {
	return &RecentQueries{
		queries: make([]RecentQuery, 0, size),
		maxSize: size,
	}
}

func (r *RecentQueries) Add(q RecentQuery) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.queries) >= r.maxSize {
		// Remove oldest
		r.queries = r.queries[1:]
	}
	r.queries = append(r.queries, q)
}

func (r *RecentQueries) Get() []RecentQuery {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Return in reverse order (newest first)
	result := make([]RecentQuery, len(r.queries))
	for i, q := range r.queries {
		result[len(r.queries)-1-i] = q
	}
	return result
}

func newSpammerMetrics(reg prometheus.Registerer) *SpammerMetrics {
	m := &SpammerMetrics{
		queriesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "filter_spammer",
			Name:      "queries_total",
			Help:      "Total number of queries sent",
		}, []string{"type", "result"}),
		queryLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "filter_spammer",
			Name:      "query_latency_seconds",
			Help:      "Latency of filter queries",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		}, []string{"type"}),
		errorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "filter_spammer",
			Name:      "errors_total",
			Help:      "Total number of unexpected errors",
		}),
		uptime: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "filter_spammer",
			Name:      "up",
			Help:      "1 if spammer is running",
		}),
		currentBlockRange: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "filter_spammer",
			Name:      "block_range",
			Help:      "Number of blocks in sample range",
		}),
	}

	reg.MustRegister(m.queriesTotal)
	reg.MustRegister(m.queryLatency)
	reg.MustRegister(m.errorsTotal)
	reg.MustRegister(m.uptime)
	reg.MustRegister(m.currentBlockRange)

	return m
}

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Name = "filter-spammer"
	app.Usage = "Spam the interop filter service with queries to validate its behavior"
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Flags = cliapp.ProtectFlags(append([]cli.Flag{
		L2RPCFlag,
		FilterRPCFlag,
		ChainIDFlag,
		NumQueriesFlag,
		BlockRangeFlag,
		QueryIntervalFlag,
		MetricsPortFlag,
		MetricsEnabledFlag,
		SafetyLevelFlag,
		JWTSecretFlag,
	}, oplog.CLIFlags("SPAMMER")...))
	app.Action = run

	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Crit("Application failed", "err", err)
	}
}

func run(cliCtx *cli.Context) error {
	logger := oplog.NewLogger(os.Stdout, oplog.ReadCLIConfig(cliCtx))

	l2RPC := cliCtx.String(L2RPCFlag.Name)
	filterRPC := cliCtx.String(FilterRPCFlag.Name)
	chainID := cliCtx.Uint64(ChainIDFlag.Name)
	numQueries := cliCtx.Int(NumQueriesFlag.Name)
	blockRange := cliCtx.Int(BlockRangeFlag.Name)
	queryIntervalStr := cliCtx.String(QueryIntervalFlag.Name)
	metricsPort := cliCtx.Int(MetricsPortFlag.Name)
	metricsEnabled := cliCtx.Bool(MetricsEnabledFlag.Name)

	queryInterval, err := time.ParseDuration(queryIntervalStr)
	if err != nil {
		return fmt.Errorf("invalid query-interval: %w", err)
	}

	safetyLevelStr := cliCtx.String(SafetyLevelFlag.Name)
	var safetyLevel suptypes.SafetyLevel
	switch safetyLevelStr {
	case "unsafe", "local-unsafe":
		safetyLevel = suptypes.LocalUnsafe
	case "cross-unsafe":
		safetyLevel = suptypes.CrossUnsafe
	default:
		return fmt.Errorf("invalid safety-level %q: must be 'unsafe' or 'cross-unsafe'", safetyLevelStr)
	}

	ctx := cliCtx.Context

	// Setup metrics and recent queries tracking
	var metrics *SpammerMetrics
	recentQueries := NewRecentQueries(10) // Track last 10 queries

	if metricsEnabled {
		reg := prometheus.NewRegistry()
		metrics = newSpammerMetrics(reg)
		metrics.uptime.Set(1)
		metrics.currentBlockRange.Set(float64(blockRange))

		// Start metrics server with /recent endpoint
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
		mux.HandleFunc("/recent", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(recentQueries.Get())
		})
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", metricsPort),
			Handler: mux,
		}
		go func() {
			logger.Info("Starting metrics server", "port", metricsPort)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("Metrics server error", "err", err)
			}
		}()
		defer func() { _ = server.Shutdown(context.Background()) }()
	}

	logger.Info("Starting filter spammer",
		"l2RPC", l2RPC,
		"filterRPC", filterRPC,
		"chainID", chainID,
		"numQueries", numQueries,
		"blockRange", blockRange,
		"queryInterval", queryInterval,
		"safetyLevel", safetyLevel,
		"metricsEnabled", metricsEnabled,
		"metricsPort", metricsPort,
	)

	// Connect to L2 RPC
	l2Client, err := client.NewRPC(ctx, logger, l2RPC)
	if err != nil {
		return fmt.Errorf("failed to connect to L2 RPC: %w", err)
	}
	defer l2Client.Close()

	ethClient, err := sources.NewEthClient(l2Client, logger, nil, &sources.EthClientConfig{
		MaxRequestsPerBatch:   20,
		MaxConcurrentRequests: 10,
		TrustRPC:              true,
		MustBePostMerge:       false,
		RPCProviderKind:       sources.RPCKindBasic,
		ReceiptsCacheSize:     100,
		TransactionsCacheSize: 100,
		HeadersCacheSize:      100,
		PayloadsCacheSize:     100,
		BlockRefsCacheSize:    100,
	})
	if err != nil {
		return fmt.Errorf("failed to create eth client: %w", err)
	}
	defer ethClient.Close()

	// Connect to filter RPC (public supervisor API)
	filterClient, err := rpc.DialContext(ctx, filterRPC)
	if err != nil {
		return fmt.Errorf("failed to connect to filter RPC: %w", err)
	}
	defer filterClient.Close()

	// Optionally connect to admin RPC with JWT if secret is provided
	jwtSecretPath := cliCtx.String(JWTSecretFlag.Name)
	var adminClient *rpc.Client
	if jwtSecretPath != "" {
		// Load JWT secret
		jwtSecret, err := oprpc.ObtainJWTSecret(logger, jwtSecretPath, false)
		if err != nil {
			return fmt.Errorf("failed to load JWT secret: %w", err)
		}

		// Connect to admin endpoint with JWT authentication
		adminEndpoint := filterRPC + "/admin"
		adminClient, err = rpc.DialOptions(ctx, adminEndpoint,
			rpc.WithHTTPAuth(node.NewJWTAuth(jwtSecret)))
		if err != nil {
			return fmt.Errorf("failed to connect to admin RPC: %w", err)
		}
		defer adminClient.Close()

		// Check filter is ready (not in failsafe) using admin API
		var failsafe bool
		if err := adminClient.CallContext(ctx, &failsafe, "admin_getFailsafeEnabled"); err != nil {
			logger.Warn("Could not check failsafe status via admin API", "err", err)
		} else {
			if failsafe {
				return errors.New("filter service is in failsafe mode")
			}
			logger.Info("Filter service responding", "failsafe", failsafe)
		}
	} else {
		logger.Info("No JWT secret provided, skipping admin API check")
	}

	// Wait for backfill to complete by doing a test query
	// The filter returns "service not ready, backfill in progress" until ready
	logger.Info("Waiting for filter backfill to complete...")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try a dummy checkAccessList call with empty access list
		// If backfill is in progress, it will return "backfill in progress"
		// If ready, it will succeed (empty list = valid)
		var result any
		err := filterClient.CallContext(ctx, &result, "supervisor_checkAccessList", []any{}, "finalized", map[string]any{
			"chainID":   fmt.Sprintf("0x%x", chainID),
			"timestamp": "0x0",
			"blockNum":  "0x0",
		})
		if err == nil {
			logger.Info("Filter backfill complete, ready to spam")
			break
		}
		errStr := err.Error()
		if strings.Contains(errStr, "backfill") || strings.Contains(errStr, "uninitialized") {
			logger.Debug("Backfill still in progress, waiting...", "err", errStr)
			time.Sleep(2 * time.Second)
			continue
		}
		// Other errors mean backfill is done but query failed for other reason
		logger.Info("Filter backfill complete, ready to spam", "testErr", errStr)
		break
	}

	// Get current head
	head, err := ethClient.InfoByLabel(ctx, eth.Unsafe)
	if err != nil {
		return fmt.Errorf("failed to get head: %w", err)
	}
	headNum := head.NumberU64()
	logger.Info("Current L2 head", "block", headNum)

	// Calculate block range to sample from (avoid underflow)
	var startBlock uint64
	if headNum > uint64(blockRange) {
		startBlock = headNum - uint64(blockRange)
	} else {
		startBlock = 1
	}

	spammer := &Spammer{
		logger:        logger,
		ethClient:     ethClient,
		filterClient:  filterClient,
		chainID:       eth.ChainIDFromUInt64(chainID),
		startBlock:    startBlock,
		endBlock:      headNum,
		safetyLevel:   safetyLevel,
		metrics:       metrics,
		recentQueries: recentQueries,
	}

	// Run queries
	ticker := time.NewTicker(queryInterval)
	defer ticker.Stop()

	validQueries := 0
	invalidQueries := 0
	errorCount := 0

	for i := 0; numQueries == 0 || i < numQueries; i++ {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down", "validQueries", validQueries, "invalidQueries", invalidQueries, "errors", errorCount)
			return nil
		case <-ticker.C:
			// Alternate between valid and invalid queries
			if i%2 == 0 {
				if err := spammer.RunValidQuery(ctx); err != nil {
					logger.Error("Valid query failed unexpectedly", "err", err, "query", i)
					errorCount++
					if metrics != nil {
						metrics.errorsTotal.Inc()
					}
					if errorCount > 10 {
						return fmt.Errorf("too many errors (%d): last error: %w", errorCount, err)
					}
				} else {
					validQueries++
					logger.Debug("Valid query passed", "query", i)
				}
			} else {
				if err := spammer.RunInvalidQuery(ctx); err != nil {
					logger.Error("Invalid query test failed", "err", err, "query", i)
					errorCount++
					if metrics != nil {
						metrics.errorsTotal.Inc()
					}
					if errorCount > 10 {
						return fmt.Errorf("too many errors (%d): last error: %w", errorCount, err)
					}
				} else {
					invalidQueries++
					logger.Debug("Invalid query rejected as expected", "query", i)
				}
			}

			if (i+1)%20 == 0 {
				logger.Info("Progress", "queries", i+1, "valid", validQueries, "invalid", invalidQueries, "errors", errorCount)
			}
		}
	}

	logger.Info("Spammer completed successfully",
		"validQueries", validQueries,
		"invalidQueries", invalidQueries,
		"errors", errorCount,
	)

	if errorCount > 0 {
		return fmt.Errorf("completed with %d errors", errorCount)
	}
	return nil
}

// Spammer handles the spam testing logic
type Spammer struct {
	logger        log.Logger
	ethClient     *sources.EthClient
	filterClient  *rpc.Client
	chainID       eth.ChainID
	startBlock    uint64
	endBlock      uint64
	safetyLevel   suptypes.SafetyLevel
	metrics       *SpammerMetrics
	recentQueries *RecentQueries
}

const maxQueryRetries = 100

// RunValidQuery fetches a random log and verifies the filter accepts it
func (s *Spammer) RunValidQuery(ctx context.Context) error {
	for attempt := 0; attempt < maxQueryRetries; attempt++ {
		err := s.tryValidQuery(ctx)
		if err == nil {
			return nil
		}
		// Check if this is a retryable error
		if !strings.Contains(err.Error(), "retry:") {
			return err
		}
		// Retryable error - try again
		s.logger.Debug("Retrying valid query", "attempt", attempt+1, "reason", err)
	}
	return fmt.Errorf("failed to find suitable block after %d attempts", maxQueryRetries)
}

// tryValidQuery attempts a single valid query, returns error with "retry:" prefix if retryable
func (s *Spammer) tryValidQuery(ctx context.Context) error {
	start := time.Now()

	// Pick a random block
	blockNum := s.startBlock + uint64(rand.Int63n(int64(s.endBlock-s.startBlock+1)))

	// Get block info
	block, err := s.ethClient.InfoByNumber(ctx, blockNum)
	if err != nil {
		return fmt.Errorf("failed to get block %d: %w", blockNum, err)
	}

	// Get receipts
	_, receipts, err := s.ethClient.FetchReceipts(ctx, block.Hash())
	if err != nil {
		return fmt.Errorf("failed to get receipts for block %d: %w", blockNum, err)
	}

	// Find a block with logs
	var foundLog bool
	for _, receipt := range receipts {
		if len(receipt.Logs) == 0 {
			continue
		}

		// Pick a random log from this receipt
		logIdx := rand.Intn(len(receipt.Logs))
		log := receipt.Logs[logIdx]

		// Compute the log hash
		logHash := LogToLogHash(log)

		// Create access entry
		access := suptypes.ChecksumArgs{
			BlockNumber: blockNum,
			LogIndex:    uint32(log.Index),
			Timestamp:   block.Time(),
			ChainID:     s.chainID,
			LogHash:     logHash,
		}.Access()

		// Encode and query
		entries := suptypes.EncodeAccessList([]suptypes.Access{access})

		// Use a timestamp after the initiating message's timestamp
		// Same-timestamp execution is invalid per spec
		execDesc := suptypes.ExecutingDescriptor{
			ChainID:   s.chainID,
			Timestamp: block.Time() + 1, // Execute at least 1 second later
		}

		err := s.filterClient.CallContext(ctx, nil, "supervisor_checkAccessList", entries, s.safetyLevel, execDesc)

		// Record metrics
		if s.metrics != nil {
			s.metrics.queryLatency.WithLabelValues("valid").Observe(time.Since(start).Seconds())
		}

		// Determine result
		result := "accepted"
		if err != nil {
			result = "rejected"
			if s.metrics != nil {
				s.metrics.queriesTotal.WithLabelValues("valid", "rejected").Inc()
			}
		} else {
			if s.metrics != nil {
				s.metrics.queriesTotal.WithLabelValues("valid", "accepted").Inc()
			}
		}

		// Record to recent queries
		if s.recentQueries != nil {
			s.recentQueries.Add(RecentQuery{
				Timestamp:   time.Now().Format("15:04:05"),
				ChainID:     s.chainID,
				BlockNumber: blockNum,
				TxHash:      receipt.TxHash.Hex(),
				LogIndex:    log.Index,
				Address:     log.Address.Hex(),
				QueryType:   "valid",
				Result:      result,
			})
		}

		if err != nil {
			// Check if this is a "skipped data" error (block outside filter's range)
			// This means the block is too old for the filter to search, not a real error
			if strings.Contains(err.Error(), "skipped data") {
				return fmt.Errorf("retry: block %d outside filter range", blockNum)
			}
			return fmt.Errorf("valid query rejected: block=%d logIdx=%d err=%w", blockNum, log.Index, err)
		}

		foundLog = true
		break
	}

	if !foundLog {
		// Block had no logs, try again with a different block
		return fmt.Errorf("retry: block %d has no logs", blockNum)
	}

	return nil
}

// RunInvalidQuery creates an invalid query and verifies the filter rejects it
func (s *Spammer) RunInvalidQuery(ctx context.Context) error {
	start := time.Now()

	// Pick a random block
	blockNum := s.startBlock + uint64(rand.Int63n(int64(s.endBlock-s.startBlock+1)))

	// Get block info
	block, err := s.ethClient.InfoByNumber(ctx, blockNum)
	if err != nil {
		return fmt.Errorf("failed to get block %d: %w", blockNum, err)
	}

	// Get receipts
	_, receipts, err := s.ethClient.FetchReceipts(ctx, block.Hash())
	if err != nil {
		return fmt.Errorf("failed to get receipts for block %d: %w", blockNum, err)
	}

	// Find a block with logs
	var foundLog bool
	for _, receipt := range receipts {
		if len(receipt.Logs) == 0 {
			continue
		}

		// Pick a random log from this receipt
		logIdx := rand.Intn(len(receipt.Logs))
		log := receipt.Logs[logIdx]

		// Compute the log hash
		logHash := LogToLogHash(log)

		// Create access entry with WRONG checksum (flip a byte)
		access := suptypes.ChecksumArgs{
			BlockNumber: blockNum,
			LogIndex:    uint32(log.Index),
			Timestamp:   block.Time(),
			ChainID:     s.chainID,
			LogHash:     logHash,
		}.Access()

		// Corrupt the checksum (flip byte at index 10, preserving prefix byte at 0)
		access.Checksum[10] ^= 0xFF

		// Encode and query
		entries := suptypes.EncodeAccessList([]suptypes.Access{access})

		// Use a timestamp after the initiating message's timestamp
		// Same-timestamp execution is invalid per spec
		execDesc := suptypes.ExecutingDescriptor{
			ChainID:   s.chainID,
			Timestamp: block.Time() + 1, // Execute at least 1 second later
		}

		err := s.filterClient.CallContext(ctx, nil, "supervisor_checkAccessList", entries, s.safetyLevel, execDesc)

		// Record metrics
		if s.metrics != nil {
			s.metrics.queryLatency.WithLabelValues("invalid").Observe(time.Since(start).Seconds())
		}

		// Determine result
		result := "rejected"
		if err == nil {
			result = "accepted"
			if s.metrics != nil {
				s.metrics.queriesTotal.WithLabelValues("invalid", "accepted").Inc()
			}
		} else {
			// Error expected - query was correctly rejected
			if s.metrics != nil {
				s.metrics.queriesTotal.WithLabelValues("invalid", "rejected").Inc()
			}
		}

		// Record to recent queries
		if s.recentQueries != nil {
			s.recentQueries.Add(RecentQuery{
				Timestamp:   time.Now().Format("15:04:05"),
				ChainID:     s.chainID,
				BlockNumber: blockNum,
				TxHash:      receipt.TxHash.Hex(),
				LogIndex:    log.Index,
				Address:     log.Address.Hex(),
				QueryType:   "invalid",
				Result:      result,
			})
		}

		if err == nil {
			return fmt.Errorf("invalid query was accepted: block=%d logIdx=%d", blockNum, log.Index)
		}

		foundLog = true
		break
	}

	if !foundLog {
		// Block had no logs, try again with a different block
		return s.RunInvalidQuery(ctx)
	}

	return nil
}

// LogToLogHash computes the log hash used in LogsDB
// This matches processors.LogToLogHash
func LogToLogHash(l *gethtypes.Log) common.Hash {
	// Compute payload hash from topics and data
	msg := make([]byte, 0)
	for _, topic := range l.Topics {
		msg = append(msg, topic.Bytes()...)
	}
	msg = append(msg, l.Data...)
	payloadHash := crypto.Keccak256Hash(msg)

	// Compute log hash
	return suptypes.PayloadHashToLogHash(payloadHash, l.Address)
}
