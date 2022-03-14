package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	bsscore "github.com/ethereum-optimism/optimism/go/bss-core"
	"github.com/ethereum-optimism/optimism/go/bss-core/dial"
	"github.com/ethereum-optimism/optimism/go/bss-core/drivers"
	"github.com/ethereum-optimism/optimism/go/bss-core/txmgr"
	"github.com/ethereum-optimism/optimism/go/teleportr/bindings/deposit"
	"github.com/ethereum-optimism/optimism/go/teleportr/db"
	"github.com/ethereum-optimism/optimism/go/teleportr/flags"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/cors"
	"github.com/urfave/cli"
)

type ContextKey string

const (
	ContextKeyReqID ContextKey = "req_id"
)

func Main(gitVersion string) func(*cli.Context) error {
	return func(cliCtx *cli.Context) error {
		cfg, err := NewConfig(cliCtx)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		depositAddr, err := bsscore.ParseAddress(cfg.DepositAddress)
		if err != nil {
			return err
		}

		l1Client, err := dial.L1EthClientWithTimeout(
			ctx, cfg.L1EthRpc, cfg.DisableHTTP2,
		)
		if err != nil {
			return err
		}
		defer l1Client.Close()

		depositContract, err := deposit.NewTeleportrDeposit(
			depositAddr, l1Client,
		)
		if err != nil {
			return err
		}

		// TODO(conner): make read-only
		database, err := db.Open(db.Config{
			Host:      cfg.PostgresHost,
			Port:      uint16(cfg.PostgresPort),
			User:      cfg.PostgresUser,
			Password:  cfg.PostgresPassword,
			DBName:    cfg.PostgresDBName,
			EnableSSL: cfg.PostgresEnableSSL,
		})
		if err != nil {
			return err
		}
		defer database.Close()

		server := NewServer(
			ctx,
			l1Client,
			database,
			depositAddr,
			depositContract,
			cfg.NumConfirmations,
		)

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = server.ListenAndServe(cfg.Hostname, cfg.Port)
		}()

		interruptChannel := make(chan os.Signal, 1)
		signal.Notify(interruptChannel, []os.Signal{
			os.Interrupt,
			os.Kill,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		}...)

		select {
		case <-interruptChannel:
			_ = server.httpServer.Shutdown(ctx)
			time.AfterFunc(defaultTimeout, func() {
				cancel()
				_ = server.httpServer.Close()
			})
			wg.Wait()
		case <-ctx.Done():
		}

		return nil
	}
}

type Config struct {
	Hostname          string
	Port              uint16
	L1EthRpc          string
	DepositAddress    string
	NumConfirmations  uint64
	PostgresHost      string
	PostgresPort      uint16
	PostgresUser      string
	PostgresPassword  string
	PostgresDBName    string
	PostgresEnableSSL bool
	DisableHTTP2      bool
}

func NewConfig(ctx *cli.Context) (Config, error) {
	return Config{
		Hostname:          ctx.GlobalString(flags.APIHostnameFlag.Name),
		Port:              uint16(ctx.GlobalUint64(flags.APIPortFlag.Name)),
		L1EthRpc:          ctx.GlobalString(flags.L1EthRpcFlag.Name),
		DepositAddress:    ctx.GlobalString(flags.DepositAddressFlag.Name),
		NumConfirmations:  ctx.GlobalUint64(flags.NumDepositConfirmationsFlag.Name),
		PostgresHost:      ctx.GlobalString(flags.PostgresHostFlag.Name),
		PostgresPort:      uint16(ctx.GlobalUint64(flags.PostgresPortFlag.Name)),
		PostgresUser:      ctx.GlobalString(flags.PostgresUserFlag.Name),
		PostgresPassword:  ctx.GlobalString(flags.PostgresPasswordFlag.Name),
		PostgresDBName:    ctx.GlobalString(flags.PostgresDBNameFlag.Name),
		PostgresEnableSSL: ctx.GlobalBool(flags.PostgresEnableSSLFlag.Name),
	}, nil
}

const (
	ContentTypeHeader = "Content-Type"
	ContentTypeJSON   = "application/json"

	defaultTimeout = 10 * time.Second
)

type Server struct {
	ctx              context.Context
	l1Client         *ethclient.Client
	database         *db.Database
	depositAddr      common.Address
	depositContract  *deposit.TeleportrDeposit
	numConfirmations uint64

	httpServer *http.Server
}

func NewServer(
	ctx context.Context,
	l1Client *ethclient.Client,
	database *db.Database,
	depositAddr common.Address,
	depositContract *deposit.TeleportrDeposit,
	numConfirmations uint64,
) *Server {

	if numConfirmations == 0 {
		panic("NumConfirmations cannot be zero")
	}

	return &Server{
		ctx:              ctx,
		l1Client:         l1Client,
		database:         database,
		depositAddr:      depositAddr,
		depositContract:  depositContract,
		numConfirmations: numConfirmations,
	}
}

func (s *Server) ListenAndServe(host string, port uint16) error {
	handler := mux.NewRouter()
	handler.HandleFunc("/healthz", HandleHealthz).Methods("GET")
	handler.HandleFunc(
		"/status",
		instrumentedErrorHandler(s.HandleStatus),
	).Methods("GET")
	handler.HandleFunc(
		"/estimate/{addr:0x[0-9a-fA-F]{40}}/{amount:[0-9]{1,80}}",
		instrumentedErrorHandler(s.HandleEstimate),
	).Methods("GET")
	handler.HandleFunc(
		"/track/{txhash:0x[0-9a-fA-F]{64}}",
		instrumentedErrorHandler(s.HandleTrack),
	).Methods("GET")
	handler.HandleFunc(
		"/history/{addr:0x[0-9a-fA-F]{40}}",
		instrumentedErrorHandler(s.HandleHistory),
	).Methods("GET")
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})
	addr := fmt.Sprintf("%s:%d", host, port)
	s.httpServer = &http.Server{
		Handler: c.Handler(handler),
		Addr:    addr,
		BaseContext: func(_ net.Listener) context.Context {
			return s.ctx
		},
	}
	log.Info("Starting HTTP server", "addr", addr)
	return s.httpServer.ListenAndServe()
}

func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}

type StatusResponse struct {
	CurrentBalanceWei   string `json:"current_balance_wei"`
	MaximumBalanceWei   string `json:"maximum_balance_wei"`
	MinDepositAmountWei string `json:"min_deposit_amount_wei"`
	MaxDepositAmountWei string `json:"max_deposit_amount_wei"`
	IsAvailable         bool   `json:"is_available"`
}

func (s *Server) HandleStatus(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {

	maxBalance, err := s.depositContract.MaxBalance(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		rpcErrorsTotal.WithLabelValues("max_balance").Inc()
		return err
	}

	minDepositAmount, err := s.depositContract.MinDepositAmount(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		rpcErrorsTotal.WithLabelValues("min_deposit_amount").Inc()
		return err
	}

	maxDepositAmount, err := s.depositContract.MaxDepositAmount(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		rpcErrorsTotal.WithLabelValues("max_deposit_amount").Inc()
		return err
	}

	curBalance, err := s.l1Client.BalanceAt(ctx, s.depositAddr, nil)
	if err != nil {
		rpcErrorsTotal.WithLabelValues("balance_at").Inc()
		return err
	}

	balanceAfterMaxDeposit := new(big.Int).Add(
		curBalance, maxDepositAmount,
	)
	isAvailable := maxBalance.Cmp(balanceAfterMaxDeposit) >= 0

	resp := StatusResponse{
		CurrentBalanceWei:   curBalance.String(),
		MaximumBalanceWei:   maxBalance.String(),
		MinDepositAmountWei: minDepositAmount.String(),
		MaxDepositAmountWei: maxDepositAmount.String(),
		IsAvailable:         isAvailable,
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	w.Header().Set(ContentTypeHeader, ContentTypeJSON)
	_, err = w.Write(jsonResp)
	return err
}

type EstimateResponse struct {
	BaseFee     string `json:"base_fee"`
	GasTipCap   string `json:"gas_tip_cap"`
	GasFeeCap   string `json:"gas_fee_cap"`
	GasEstimate string `json:"gas_estimate"`
}

func (s *Server) HandleEstimate(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {

	vars := mux.Vars(r)
	addressStr, ok := vars["addr"]
	if !ok {
		return StatusError{
			Err:  errors.New("missing address parameter"),
			Code: http.StatusBadRequest,
		}
	}
	address, err := bsscore.ParseAddress(addressStr)
	if err != nil {
		return StatusError{
			Err:  err,
			Code: http.StatusBadRequest,
		}
	}

	amountStr, ok := vars["amount"]
	if !ok {
		return StatusError{
			Err:  errors.New("missing amount parameter"),
			Code: http.StatusBadRequest,
		}
	}
	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		return StatusError{
			Err:  errors.New("unable to parse amount"),
			Code: http.StatusBadRequest,
		}
	}

	gasTipCap, err := s.l1Client.SuggestGasTipCap(ctx)
	if err != nil {
		rpcErrorsTotal.WithLabelValues("suggest_gas_tip_cap").Inc()
		// If the request failed because the backend does not support
		// eth_maxPriorityFeePerGas, fallback to using the default constant.
		// Currently Alchemy is the only backend provider that exposes this
		// method, so in the event their API is unreachable we can fallback to a
		// degraded mode of operation. This also applies to our test
		// environments, as hardhat doesn't support the query either.
		if !drivers.IsMaxPriorityFeePerGasNotFoundError(err) {
			return err
		}
		gasTipCap = drivers.FallbackGasTipCap
	}

	header, err := s.l1Client.HeaderByNumber(ctx, nil)
	if err != nil {
		rpcErrorsTotal.WithLabelValues("header_by_number").Inc()
		return err
	}

	gasFeeCap := txmgr.CalcGasFeeCap(header.BaseFee, gasTipCap)

	gasUsed, err := s.l1Client.EstimateGas(ctx, ethereum.CallMsg{
		From:      address,
		To:        &s.depositAddr,
		Gas:       0,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Value:     amount,
		Data:      nil,
	})
	if err != nil {
		rpcErrorsTotal.WithLabelValues("estimate_gas").Inc()
		return err
	}

	resp := EstimateResponse{
		BaseFee:     header.BaseFee.String(),
		GasTipCap:   gasTipCap.String(),
		GasFeeCap:   gasFeeCap.String(),
		GasEstimate: new(big.Int).SetUint64(gasUsed).String(),
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	w.Header().Set(ContentTypeHeader, ContentTypeJSON)
	_, err = w.Write(jsonResp)
	return err
}

type RPCTeleport struct {
	ID             string           `json:"id"`
	Address        string           `json:"address"`
	AmountWei      string           `json:"amount_wei"`
	TxHash         string           `json:"tx_hash"`
	BlockNumber    string           `json:"block_number"`
	BlockTimestamp string           `json:"block_timestamp_unix"`
	Disbursement   *RPCDisbursement `json:"disbursement"`
}

func makeRPCTeleport(teleport *db.Teleport) RPCTeleport {
	rpcTeleport := RPCTeleport{
		ID:             strconv.FormatUint(teleport.ID, 10),
		Address:        teleport.Address.String(),
		AmountWei:      teleport.Amount.String(),
		TxHash:         teleport.Deposit.TxnHash.String(),
		BlockNumber:    strconv.FormatUint(teleport.Deposit.BlockNumber, 10),
		BlockTimestamp: strconv.FormatInt(teleport.Deposit.BlockTimestamp.Unix(), 10),
	}
	if teleport.Disbursement != nil {
		rpcTeleport.Disbursement = &RPCDisbursement{
			TxHash:         teleport.Disbursement.TxnHash.String(),
			BlockNumber:    strconv.FormatUint(teleport.Disbursement.BlockNumber, 10),
			BlockTimestamp: strconv.FormatInt(teleport.Disbursement.BlockTimestamp.Unix(), 10),
			Success:        teleport.Disbursement.Success,
		}
	}
	return rpcTeleport
}

type RPCDisbursement struct {
	TxHash         string `json:"tx_hash"`
	BlockNumber    string `json:"block_number"`
	BlockTimestamp string `json:"block_timestamp_unix"`
	Success        bool   `json:"success"`
}

type TrackResponse struct {
	CurrentBlockNumber     string      `json:"current_block_number"`
	ConfirmationsRequired  string      `json:"confirmations_required"`
	ConfirmationsRemaining string      `json:"confirmations_remaining"`
	Teleport               RPCTeleport `json:"teleport"`
}

func (s *Server) HandleTrack(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {

	vars := mux.Vars(r)
	txHashStr, ok := vars["txhash"]
	if !ok {
		return StatusError{
			Err:  errors.New("missing txhash parameter"),
			Code: http.StatusBadRequest,
		}
	}
	txHash := common.HexToHash(txHashStr)

	blockNumber, err := s.l1Client.BlockNumber(ctx)
	if err != nil {
		rpcErrorsTotal.WithLabelValues("block_number").Inc()
		return err
	}

	teleport, err := s.database.LoadTeleportByDepositHash(txHash)
	if err != nil {
		databaseErrorsTotal.WithLabelValues("load_teleport_by_deposit_hash").Inc()
		return err
	}

	if teleport == nil {
		return StatusError{
			Code: http.StatusNotFound,
		}
	}

	var confsRemaining uint64
	if teleport.Deposit.BlockNumber+s.numConfirmations > blockNumber+1 {
		confsRemaining = blockNumber + 1 -
			(teleport.Deposit.BlockNumber + s.numConfirmations)
	}

	resp := TrackResponse{
		CurrentBlockNumber:     strconv.FormatUint(blockNumber, 10),
		ConfirmationsRequired:  strconv.FormatUint(s.numConfirmations, 10),
		ConfirmationsRemaining: strconv.FormatUint(confsRemaining, 10),
		Teleport:               makeRPCTeleport(teleport),
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	w.Header().Set(ContentTypeHeader, ContentTypeJSON)
	_, err = w.Write(jsonResp)
	return err
}

type HistoryResponse struct {
	Teleports []RPCTeleport `json:"teleports"`
}

func (s *Server) HandleHistory(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) error {

	vars := mux.Vars(r)
	addrStr, ok := vars["addr"]
	if !ok {
		return StatusError{
			Err:  errors.New("missing addr parameter"),
			Code: http.StatusBadRequest,
		}
	}
	addr := common.HexToAddress(addrStr)

	teleports, err := s.database.LoadTeleportsByAddress(addr)
	if err != nil {
		databaseErrorsTotal.WithLabelValues("load_teleports_by_address").Inc()
		return err
	}

	rpcTeleports := make([]RPCTeleport, 0, len(teleports))
	for _, teleport := range teleports {
		rpcTeleports = append(rpcTeleports, makeRPCTeleport(&teleport))
	}

	resp := HistoryResponse{
		Teleports: rpcTeleports,
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	w.Header().Set(ContentTypeHeader, ContentTypeJSON)
	_, err = w.Write(jsonResp)
	return err
}

type Error interface {
	error
	Status() int
}

type StatusError struct {
	Code int
	Err  error
}

func (se StatusError) Error() string {
	if se.Err != nil {
		msg := se.Err.Error()
		if msg != "" {
			return msg
		}
	}
	return http.StatusText(se.Code)
}

func (se StatusError) Status() int {
	return se.Code
}

func instrumentedErrorHandler(
	h func(context.Context, http.ResponseWriter, *http.Request) error,
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		rpcRequestsTotal.Inc()

		ctx, cancel := populateContext(w, r)
		defer cancel()

		reqID := GetReqID(ctx)

		log.Info("HTTP request",
			"req_id", reqID,
			"path", r.URL.Path,
			"user_agent", r.UserAgent())

		respTimer := prometheus.NewTimer(httpRequestDurationSumm)
		err := h(ctx, w, r)
		elapsed := respTimer.ObserveDuration()

		var statusCode int
		switch e := err.(type) {
		case nil:
			statusCode = 200
			log.Info("HTTP success",
				"req_id", reqID,
				"elapsed", elapsed)

		case Error:
			statusCode = e.Status()
			log.Warn("HTTP error",
				"req_id", reqID,
				"elapsed", elapsed,
				"status", statusCode,
				"err", e.Error())
			http.Error(w, e.Error(), statusCode)

		default:
			statusCode = http.StatusInternalServerError
			log.Warn("HTTP internal error",
				"req_id", reqID,
				"elapsed", elapsed,
				"status", statusCode,
				"err", err)
			http.Error(w, http.StatusText(statusCode), statusCode)
		}

		httpResponseCodesTotal.WithLabelValues(strconv.Itoa(statusCode)).Inc()
	}
}

func populateContext(
	w http.ResponseWriter,
	r *http.Request,
) (context.Context, func()) {

	ctx := context.WithValue(r.Context(), ContextKeyReqID, uuid.NewString())
	return context.WithTimeout(ctx, defaultTimeout)
}

func GetReqID(ctx context.Context) string {
	if reqID, ok := ctx.Value(ContextKeyReqID).(string); ok {
		return reqID
	}
	return ""
}
