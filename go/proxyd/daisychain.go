package proxyd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
)

var (
	// Represents the chain id for the Optimism L2 mainnet
	MainnetChainId = big.NewInt(10)
	// Represents the chain id for the Optimism L2 public testnet
	KovanChainId = big.NewInt(69)
)

// maxEpoch represents the maximum epoch
// supported by the daisychain. If the RequestOptions
// is sent with an epoch larger than this, it is an
// error
const maxEpoch = uint(6)

// DaisyChainServer is the main struct representing
// the daisy chain server. It has a reference to
// each possible backend
type DaisyChainServer struct {
	rpcServer          *http.Server
	maxBodySize        int64
	authenticatedPaths map[string]string
	epoch1             *Backend
	epoch2             *Backend
	epoch3             *Backend
	epoch4             *Backend
	epoch5             *Backend
	epoch6             *Backend
	bedrockCutoffBlock *big.Int
	l2ChainID          *big.Int
	upgrader           *websocket.Upgrader
}

// RequestOptions represents the optional
// additional RPC parameter than can be sent
// by a user to the daisychain server. The epoch
// represents the backend that the request should be
// sent to.
type RequestOptions struct {
	Epoch uint `json:"epoch"`
}

// NewDaisyChainServer will create a daisychain server
func NewDaisyChainServer(
	backends map[string]*Backend,
	maxBodySize int64,
	authenticatedPaths map[string]string,
	l2ChainID *big.Int,
	bedrockCutoffBlock *big.Int,
) *DaisyChainServer {
	srv := DaisyChainServer{
		epoch1:             backends["epoch1"],
		epoch2:             backends["epoch2"],
		epoch3:             backends["epoch3"],
		epoch4:             backends["epoch4"],
		epoch5:             backends["epoch5"],
		epoch6:             backends["epoch6"],
		maxBodySize:        maxBodySize,
		authenticatedPaths: authenticatedPaths,
		bedrockCutoffBlock: bedrockCutoffBlock,
		l2ChainID:          l2ChainID,
		upgrader: &websocket.Upgrader{
			HandshakeTimeout: 5 * time.Second,
		},
	}
	return &srv
}

// StartDaisyChain will start a daisychain server
// and is the entrypoint to the application
func StartDaisyChain(config *Config) (func(), error) {
	if err := config.ValidateDaisyChainBackends(); err != nil {
		return func() {}, err
	}

	lim := NewLocalRateLimiter()
	_, backendsByName, err := config.BuildBackends(lim)
	if err != nil {
		return func() {}, err
	}

	resolvedAuth, err := config.ResolveAuth()
	if err != nil {
		return func() {}, err
	}

	// parse the config
	srv := NewDaisyChainServer(
		backendsByName,
		config.Server.MaxBodySizeBytes,
		resolvedAuth,
		config.Eth.L2ChainID,
		config.Eth.BedrockCutoffBlock,
	)

	l2ChainID, err := srv.checkChainIDs()
	if err != nil {
		return func() {}, err
	}
	if srv.l2ChainID == nil {
		srv.l2ChainID = l2ChainID
	}

	if srv.l2ChainID.Cmp(MainnetChainId) == 0 {
		log.Info("running on mainnet")
		// TODO(tynes): check and set the mainnet cutoff block
		// srv.bedrockCutoffBlock = ...
	}
	if srv.l2ChainID.Cmp(KovanChainId) == 0 {
		log.Info("running on kovan")
		// TODO(tynes): check and set the kovan cutoff block
		// srv.bedrockCutoffBlock = ...
	}

	if srv.bedrockCutoffBlock == nil {
		log.Info("bedrock cutoff block not detected, all requests routing to latest")
		srv.bedrockCutoffBlock = new(big.Int)
	} else {
		log.Info("bedrock cutoff block", "number", srv.bedrockCutoffBlock)
	}

	if config.Metrics.Enabled {
		addr := fmt.Sprintf("%s:%d", config.Metrics.Host, config.Metrics.Port)
		log.Info("starting metrics server", "addr", addr)
		go func() {
			if err := http.ListenAndServe(addr, promhttp.Handler()); err != nil {
				log.Error("error starting metrics server", "err", err)
			}
		}()
	}

	// To allow integration tests to cleanly come up, wait
	// 10ms to give the below goroutines enough time to
	// encounter an error creating their servers
	errTimer := time.NewTimer(10 * time.Millisecond)

	if config.Server.RPCPort != 0 {
		go func() {
			if err := srv.RPCListenAndServe(config.Server.RPCHost, config.Server.RPCPort); err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					log.Info("RPC server shut down")
					return
				}
				log.Crit("error starting RPC server", "err", err)
			}
		}()
	}

	<-errTimer.C
	log.Info("started daisychain")

	return func() {
		log.Info("shutting down daisychain")
		srv.Shutdown()
		log.Info("goodbye")
	}, nil
}

// checkChainIDs will check that each configured backend returns
// the same chain id
func (s *DaisyChainServer) checkChainIDs() (*big.Int, error) {
	// send a chain id request to each node to ensure they are on the same chain
	req, _ := ParseRPCReq([]byte(`{"id":"1","jsonrpc":"2.0","method":"eth_chainId","params":[]}`))
	chainIds := []*hexutil.Big{}
	for _, backend := range s.Backends() {
		res, _ := backend.Forward(context.Background(), req)
		str, ok := res.Result.(string)
		if !ok {
			return nil, errors.New("cannot fetch chainid on start")
		}
		chainId := new(hexutil.Big)
		err := chainId.UnmarshalText([]byte(str))
		if err != nil {
			return nil, err
		}
		chainIds = append(chainIds, chainId)
	}

	if len(chainIds) == 0 {
		panic("cannot fetch remote chain id")
	}
	chainId := chainIds[0].ToInt()
	for _, id := range chainIds {
		if id.ToInt().Cmp(chainId) != 0 {
			log.Crit("mismatched chain ids detected", "chain-id", chainId, "other", id)
		}
	}
	log.Info("detected chain id", "value", chainId)
	if s.l2ChainID != nil {
		if s.l2ChainID.Cmp(chainId) != 0 {
			return nil, fmt.Errorf("mismatched chainids: expected %d, got %d", s.l2ChainID, chainId)
		}
	}
	return chainId, nil
}

// HandleRPC is the top level RPC handler for the daisychain.
// All RPC requests are routed through this handler.
func (s *DaisyChainServer) HandleRPC(w http.ResponseWriter, r *http.Request) {
	ctx := populateContext(w, r, s.authenticatedPaths)
	if ctx == nil {
		return
	}

	doRequest := func(ctx context.Context, req *RPCReq) (*RPCRes, bool) {
		argType, ok := argTypes[req.Method]
		if !ok {
			return NewRPCErrorRes(req.ID, ErrParseErr), false
		}

		values, err := parsePositionalArguments(req.Params, argType)
		if err != nil {
			return NewRPCErrorRes(req.ID, fmt.Errorf("%s: %w", ErrParseErr, err)), false
		}

		argument, ok := parseRequestOptions(values)
		if !ok {
			return NewRPCErrorRes(req.ID, ErrParseErr), false
		}

		req, err = trimRequestOptions(req, values)
		if err != nil {
			return NewRPCErrorRes(req.ID, ErrParseErr), false
		}

		// TODO(tynes): perhaps a better approach is to attempt to classify the request
		// up front and then have an enum in a switch statement
		var res *RPCRes
		// Check to see if the request is meant for the latest node
		// first. This will be the most commonly used case, so check first.
		if s.isLatestEpochsRPC(argument) {
			res = s.handleLatestEpochsRPC(ctx, req, values)
		} else if s.isHashBasedRPC(values) {
			// Check to see if a hash was passed in the rpc params
			// and fall back to each node, looking for the hash
			// TODO: If we want users to be able to pass a hash
			// and not need to be explicit with the epoch and have
			// it "just work" then this needs to be checked first
			res = s.handleHashTaggedRPC(ctx, req)
		} else {
			// If an epoch was passed then route based on the epoch
			res = s.handleEpochRPC(ctx, req, argument)
		}
		return res, false
	}

	handleRPC(ctx, w, r, s.maxBodySize, doRequest)
}

// isLatestEpochsRPC will check to see if the request
func (s *DaisyChainServer) isLatestEpochsRPC(opts *RequestOptions) bool {
	if opts == nil {
		return true
	}
	if opts.Epoch == 5 || opts.Epoch == 6 {
		return true
	}
	return false
}

// Backends returns a list of all of the configured
// backends in descending order.
func (s *DaisyChainServer) Backends() []*Backend {
	backends := []*Backend{}
	if s.epoch6 != nil {
		backends = append(backends, s.epoch6)
	}
	if s.epoch5 != nil {
		backends = append(backends, s.epoch5)
	}
	if s.epoch4 != nil {
		backends = append(backends, s.epoch4)
	}
	if s.epoch3 != nil {
		backends = append(backends, s.epoch3)
	}
	if s.epoch2 != nil {
		backends = append(backends, s.epoch2)
	}
	if s.epoch1 != nil {
		backends = append(backends, s.epoch1)
	}
	return backends
}

// handleLatestEpochsRPC will send a request to the post regenesis nodes.
// This needs to be able to determine if the request should be sent to
// the bedrock node or the pre bedrock node based on the bedrockCutoffBlock.
// It also needs to handle string based blocktags.
// TODO: handle eth_getLogs across the cutoff point
func (s *DaisyChainServer) handleLatestEpochsRPC(ctx context.Context, req *RPCReq, values []reflect.Value) *RPCRes {
	backend := s.epoch6
	if num, ok := s.isNumberBasedRPC(values); ok {
		// TODO(tynes): can get away without using big math here
		// TODO(tynes): behavior for pending, latest and earliest?
		// "pending" is -1
		// "latest" is -2
		// "earliest" is 0
		if num.Cmp(common.Big0) > 1 && num.Cmp(s.bedrockCutoffBlock) < 1 {
			backend = s.epoch5
		}
	}

	if backend == nil {
		log.Trace("attempting to query unconfigured backend")
		return NewRPCErrorRes(req.ID, ErrInternal)
	}
	res, _ := backend.Forward(ctx, req)
	return res
}

// handleEpochRPC will determine the epoch that the user specified and
// send the request to the appropriate backend
func (s *DaisyChainServer) handleEpochRPC(ctx context.Context, req *RPCReq, argument *RequestOptions) *RPCRes {
	if argument == nil {
		log.Trace("cannot process nil request options")
		return NewRPCErrorRes(req.ID, ErrInternal)
	}

	var backend *Backend
	switch argument.Epoch {
	case 6:
		backend = s.epoch6
	case 5:
		backend = s.epoch5
	case 4:
		backend = s.epoch4
	case 3:
		backend = s.epoch3
	case 2:
		backend = s.epoch2
	case 1:
		backend = s.epoch1
	default:
		return NewRPCErrorRes(req.ID, ErrInternal)
	}

	// This should never happen
	if backend == nil {
		log.Trace("attempting to query unconfigured backend")
		return NewRPCErrorRes(req.ID, ErrInternal)
	}

	res, err := backend.Forward(ctx, req)
	if err != nil {
		return NewRPCErrorRes(req.ID, err)
	}
	return res
}

// isHashBasedRPC will check to see if any of the params are
// a hash so that each backend can be attempted in order
func (s *DaisyChainServer) isHashBasedRPC(values []reflect.Value) bool {
	for _, value := range values {
		iface := value.Interface()
		if param, ok := iface.(rpc.BlockNumberOrHash); ok {
			if _, ok := param.Hash(); ok {
				return true
			}
		}
		if _, ok := iface.(common.Hash); ok {
			return true
		}
	}
	return false
}

// RPCListenAndServe and start the RPC server
func (s *DaisyChainServer) RPCListenAndServe(host string, port int) error {
	hdlr := mux.NewRouter()
	hdlr.HandleFunc("/healthz", s.HandleHealthz).Methods("GET")
	hdlr.HandleFunc("/", s.HandleRPC).Methods("POST")
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
	})
	addr := fmt.Sprintf("%s:%d", host, port)
	s.rpcServer = &http.Server{
		Handler: instrumentedHdlr(c.Handler(hdlr)),
		Addr:    addr,
	}
	log.Info("starting HTTP server", "addr", addr)
	return s.rpcServer.ListenAndServe()
}

// Shutdown will shut down the server
func (s *DaisyChainServer) Shutdown() {
	if s.rpcServer != nil {
		_ = s.rpcServer.Shutdown(context.Background())
	}
}

// HandleHealthz is the health handler for infrastructure
// monitoring
func (s *DaisyChainServer) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}

// isNumberBasedRPC checks to see if the RPC request is number based.
// This is important for determining which of the post regenesis era
// RPC nodes to send the request to. If the number is less than the
// bedrockCutoffBlock, then it should go to the first post regenesis
// node. If the number is greater than the bedrockCutoffBlock, then
// it should go to the bedrock node.
func (s *DaisyChainServer) isNumberBasedRPC(values []reflect.Value) (*big.Int, bool) {
	for _, value := range values {
		iface := value.Interface()
		if param, ok := iface.(rpc.BlockNumberOrHash); ok {
			if num, ok := param.Number(); ok {
				return new(big.Int).SetInt64(num.Int64()), true
			}
		}
		if num, ok := iface.(rpc.BlockNumber); ok {
			return new(big.Int).SetInt64(num.Int64()), true
		}
	}
	return nil, false
}

// handleHashTaggedRPC will try each backend until it finds
// a backend with a successful response. The requests should be tried
// in descending order
func (s *DaisyChainServer) handleHashTaggedRPC(ctx context.Context, req *RPCReq) *RPCRes {
	var res *RPCRes
	for i, backend := range s.Backends() {
		log.Trace("trying hash tagged request", "backend", backend, "index", i, "method", req.Method)
		res, _ = backend.Forward(ctx, req)
		if !res.IsError() {
			break
		}
	}
	return res
}

// trimRequestOptions will remove the optional RequestOptions from the
// RPC request and mutate the RPC request params so that it can be
// safely forwarded to a backend.
func trimRequestOptions(req *RPCReq, values []reflect.Value) (*RPCReq, error) {
	raw, err := json.Marshal(values[0 : len(values)-1])
	if err != nil {
		return nil, err
	}
	req.Params = raw
	return req, nil
}

// parseRequestOptions parses the daisychain RequestOptions from
// the request. The RequestOptions are optional, meaning that they
// can be `nil`. Each Ethereum RPC method is extended to accept an
// optional RequestOptions that can be passed in to determine which
// backend chain to submit the request to and is removed before
// actually forwarding the request.
func parseRequestOptions(values []reflect.Value) (*RequestOptions, bool) {
	requestOpts := values[len(values)-1]
	argument, ok := requestOpts.Interface().(*RequestOptions)
	if !ok {
		return nil, false
	}
	if argument == nil {
		return nil, true
	}
	if argument.Epoch > maxEpoch {
		return nil, false
	}
	return argument, true
}
