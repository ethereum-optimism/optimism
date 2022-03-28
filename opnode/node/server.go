package node

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// TODO(inphi): add metrics

type rpcServer struct {
	endpoint   string
	api        *nodeAPI
	httpServer *http.Server
	appVersion string
	listenAddr net.Addr
	log        log.Logger
}

func newRPCServer(ctx context.Context, addr string, port int, l2Client l2EthClient, withdrawalContractAddress common.Address, log log.Logger, appVersion string) (*rpcServer, error) {
	api := newNodeAPI(l2Client, withdrawalContractAddress, log.New("rpc", "node"))
	endpoint := fmt.Sprintf("%s:%d", addr, port)
	r := &rpcServer{
		endpoint:   endpoint,
		api:        api,
		appVersion: appVersion,
		log:        log,
	}
	return r, nil
}

func (s *rpcServer) Start() error {
	apis := []rpc.API{{
		Namespace:     "optimism",
		Service:       s.api,
		Public:        true,
		Authenticated: false,
	}}
	srv := rpc.NewServer()
	if err := node.RegisterApis(apis, nil, srv, true); err != nil {
		return err
	}

	host := strings.Split(s.endpoint, ":")[0]
	nodeHandler := node.NewHTTPHandlerStack(srv, nil, []string{host}, nil)

	mux := http.NewServeMux()
	mux.Handle("/", nodeHandler)
	mux.HandleFunc("/healthz", healthzHandler(s.appVersion))

	listener, err := net.Listen("tcp", s.endpoint)
	if err != nil {
		return err
	}
	s.listenAddr = listener.Addr()

	s.httpServer = &http.Server{Handler: mux}
	go func() {
		if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) { // todo improve error handling
			s.log.Error("http server failed", "err", err)
		}
	}()
	return nil
}

func (r *rpcServer) Stop() {
	_ = r.httpServer.Shutdown(context.Background())
}

func (r *rpcServer) Addr() net.Addr {
	return r.listenAddr
}

func healthzHandler(appVersion string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(appVersion))
	}
}

type l2EthClientImpl struct {
	l2RPCClient *rpc.Client
}

func (c *l2EthClientImpl) GetBlockHeader(ctx context.Context, blockTag string) (*types.Header, error) {
	var head *types.Header
	err := c.l2RPCClient.CallContext(ctx, &head, "eth_getBlockByNumber", blockTag, false)
	return head, err
}

func (c *l2EthClientImpl) GetProof(ctx context.Context, address common.Address, blockTag string) (*AccountResult, error) {
	var getProofResponse *AccountResult
	err := c.l2RPCClient.CallContext(ctx, &getProofResponse, "eth_getProof", address, []common.Hash{}, blockTag)
	if err == nil && getProofResponse == nil {
		err = ethereum.NotFound
	}
	return getProofResponse, err
}
