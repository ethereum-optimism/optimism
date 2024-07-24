package op_txpool

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/prometheus/client_golang/prometheus"

	"golang.org/x/time/rate"
)

var (
	authHeaderKey = "x-optimism-signature"

	// errs
	rateLimitErr             = &oprpc.JsonError{Message: "rate limited", Code: types.TransactionConditionalRejectedErrCode}
	endpointDisabledErr      = &oprpc.JsonError{Message: "endpoint disabled", Code: types.TransactionConditionalRejectedErrCode}
	missingConditionalErr    = &oprpc.JsonError{Message: "missing conditional", Code: types.TransactionConditionalRejectedErrCode}
	invalidAuthenticationErr = &oprpc.JsonError{Message: "invalid authentication", Code: types.TransactionConditionalRejectedErrCode}
	entrypointSupportErr     = &oprpc.JsonError{Message: "only 4337 Entrypoint contract support", Code: types.TransactionConditionalRejectedErrCode}
)

type ConditionalTxService struct {
	log log.Logger
	cfg *CLIConfig

	limiter             *rate.Limiter
	backends            map[string]client.RPC
	entrypointAddresses map[common.Address]bool

	costSummary prometheus.Summary
	requests    prometheus.Counter
	failures    *prometheus.CounterVec
}

func NewConditionalTxService(ctx context.Context, log log.Logger, m metrics.Factory, cfg *CLIConfig) (*ConditionalTxService, error) {
	backends := map[string]client.RPC{}
	for _, addr := range cfg.SendRawTransactionConditionalBackends {
		rpc, err := client.NewRPC(ctx, log, addr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial backend %s: %w", addr, err)
		}

		rpcMetrics := metrics.MakeRPCClientMetrics(addr, m)
		backends[addr] = client.NewInstrumentedRPC(rpc, &rpcMetrics)
	}

	limiter := rate.NewLimiter(types.TransactionConditionalMaxCost, int(cfg.SendRawTransactionConditionalRateLimit))
	entrypointAddresses := map[common.Address]bool{predeploys.EntryPoint_v060Addr: true, predeploys.EntryPoint_v070Addr: true}

	return &ConditionalTxService{
		log: log,
		cfg: cfg,

		limiter:             limiter,
		backends:            backends,
		entrypointAddresses: entrypointAddresses,

		costSummary: m.NewSummary(prometheus.SummaryOpts{
			Namespace: MetricsNameSpace,
			Name:      "txconditional_cost",
			Help:      "summary of cost observed by *accepted* conditional txs",
		}),
		requests: m.NewCounter(prometheus.CounterOpts{
			Namespace: MetricsNameSpace,
			Name:      "txconditional_requests",
			Help:      "number of conditional transaction requests",
		}),
		failures: m.NewCounterVec(prometheus.CounterOpts{
			Namespace: MetricsNameSpace,
			Name:      "txconditional_failures",
			Help:      "number of conditional transaction failures",
		}, []string{"err"}),
	}, nil
}

func (s *ConditionalTxService) SendRawTransactionConditional(ctx context.Context, txBytes hexutil.Bytes, cond *types.TransactionConditional) (common.Hash, error) {
	s.requests.Inc()
	if !s.cfg.SendRawTransactionConditionalEnabled {
		s.failures.WithLabelValues("disabled").Inc()
		return common.Hash{}, endpointDisabledErr
	}
	if cond == nil {
		s.failures.WithLabelValues("missing conditional").Inc()
		return common.Hash{}, missingConditionalErr
	}

	// Authenticate the request
	peerInfo := rpc.PeerInfoFromContext(ctx)
	authHeader := peerInfo.HTTP.Header.Get(authHeaderKey)
	if authHeader == "" {
		s.failures.WithLabelValues("missing auth").Inc()
		return common.Hash{}, invalidAuthenticationErr
	}
	authElems := strings.Split(authHeader, ":")
	if len(authElems) != 2 {
		s.failures.WithLabelValues("invalid auth header").Inc()
		return common.Hash{}, invalidAuthenticationErr
	}

	caller, signature := common.HexToAddress(authElems[0]), common.Hex2Bytes(authElems[1])
	sigPubKey, err := crypto.SigToPub(accounts.TextHash(peerInfo.HTTP.Body), signature)
	if err != nil {
		s.failures.WithLabelValues("invalid auth signature").Inc()
		return common.Hash{}, invalidAuthenticationErr
	}
	if caller != crypto.PubkeyToAddress(*sigPubKey) {
		s.failures.WithLabelValues("mismatch auth caller").Inc()
		return common.Hash{}, invalidAuthenticationErr
	}

	// Handle the request. For now, we do nothing with the authenticated signer
	hash, err := s.sendCondTx(ctx, caller, txBytes, cond)
	if err != nil {
		s.failures.WithLabelValues(err.Error()).Inc()
		s.log.Error("failed transaction conditional", "caller", caller.String(), "hash", hash.String(), "err", err)
	}
	return hash, err
}

func (s *ConditionalTxService) sendCondTx(ctx context.Context, caller common.Address, txBytes hexutil.Bytes, cond *types.TransactionConditional) (common.Hash, error) {
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		return common.Hash{}, fmt.Errorf("failed to unmarshal tx: %w", err)
	}

	txHash := tx.Hash()

	// external checks (tx target, conditional cost & validation)
	if tx.To() == nil || !s.entrypointAddresses[*tx.To()] {
		return txHash, entrypointSupportErr
	}
	if err := cond.Validate(); err != nil {
		return txHash, &oprpc.JsonError{
			Message: fmt.Sprintf("failed conditional validation: %s", err),
			Code:    types.TransactionConditionalRejectedErrCode,
		}
	}
	cost := cond.Cost()
	if cost > types.TransactionConditionalMaxCost {
		return txHash, &oprpc.JsonError{
			Message: fmt.Sprintf("conditional cost, %d, exceeded max: %d", cost, types.TransactionConditionalMaxCost),
			Code:    types.TransactionConditionalCostExceededMaxErrCode,
		}
	}

	// enforce rate limit on the cost to be observed
	ctxwt, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.limiter.WaitN(ctxwt, cost); err != nil {
		return txHash, rateLimitErr
	}

	s.costSummary.Observe(float64(cost))

	// Broadcast to the registered backends. If we observe a rejected conditional, we'll surface that to the
	// caller. Otherwise, we broadcast with best effort and will be alerted via metrics for unhealthy backends
	// 	NOTE: proxyd will feature this feature so in practice we wont actually need to fan-out here.
	s.log.Info("broadcasting conditional transaction", "caller", caller.String(), "hash", txHash.String())
	for addr, backend := range s.backends {
		if err := backend.CallContext(ctx, nil, "eth_sendRawTransactionConditional", txBytes, cond); err != nil {
			s.log.Error("error broadcasting to backend", "addr", addr, "err", err)
			if err, ok := err.(rpc.Error); ok && err.ErrorCode() == types.TransactionConditionalRejectedErrCode {
				return txHash, err
			}
		}
	}
	return txHash, nil
}
