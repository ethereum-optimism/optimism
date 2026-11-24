package client_test

import (
	"context"
	"encoding/binary"
	"errors"
	"log/slog"
	"math/big"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

const (
	// gethWebsocketLimit is go-ethereum's default websocket message size limit for RPC
	// clients (wsDefaultReadLimit in its rpc package).
	gethWebsocketLimit = 32 << 20
	// responseSize is the size of the responses in TestResponseSizeParity: just above
	// gethWebsocketLimit, the limit that op-service's dial removes.
	responseSize = gethWebsocketLimit + 1<<20

	// testPrivateKey is the first well-known development account, for txmgr's signer.
	testPrivateKey = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

	// callTimeout bounds the dial and call in each case of TestResponseSizeParity. It is
	// longer than client.RPC's default call timeout, so slow runs, such as with -race, pass.
	callTimeout = time.Minute
)

// ethService answers eth_call with a result of the size given in the call data, and
// eth_chainId, which txmgr calls when it starts.
type ethService struct{}

// callArgs is the part of an eth_call transaction object that ethService reads.
type callArgs struct {
	Input hexutil.Bytes `json:"input"`
}

// ChainId serves eth_chainId, which txmgr needs to start.
func (ethService) ChainId() *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(1))
}

// Call serves eth_call: it returns as many zero bytes as the call data asks for (see
// requestResultSize).
func (ethService) Call(args callArgs, _ string) (hexutil.Bytes, error) {
	if len(args.Input) != 8 {
		return nil, errors.New("call data must be the result size as 8 bytes")
	}
	return make(hexutil.Bytes, binary.BigEndian.Uint64(args.Input)), nil
}

// startServer serves ethService over HTTP and websocket, and returns both URLs.
func startServer(t *testing.T) (httpURL, wsURL string) {
	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("eth", ethService{}))
	t.Cleanup(srv.Stop)
	httpSrv := httptest.NewServer(srv)
	t.Cleanup(httpSrv.Close)
	wsSrv := httptest.NewServer(srv.WebsocketHandler([]string{"*"}))
	t.Cleanup(wsSrv.Close)
	return httpSrv.URL, "ws" + strings.TrimPrefix(wsSrv.URL, "http")
}

// callFunc makes an eth_call to ethService whose response is about size bytes.
type callFunc func(ctx context.Context, size int) error

// requestResultSize returns 8 bytes of call data that ask ethService for a response of about
// size bytes: the result is size/2 bytes, which hex encoding doubles.
func requestResultSize(size int) []byte {
	return binary.BigEndian.AppendUint64(nil, uint64(size/2))
}

// rpcCaller makes sized calls through an op-service client.RPC, and checks that the whole
// result arrived.
func rpcCaller(cl client.RPC) callFunc {
	return func(ctx context.Context, size int) error {
		var res hexutil.Bytes
		err := cl.CallContext(ctx, &res, "eth_call", callArgs{Input: requestResultSize(size)}, "latest")
		if err == nil && len(res) != size/2 {
			return errors.New("result has the wrong size")
		}
		return err
	}
}

// ethCaller is the eth_call method shared by ethclient.Client and txmgr's backend.
type ethCaller interface {
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

// ethClientCaller makes sized calls through an ethCaller, and checks that the whole result
// arrived.
func ethClientCaller(cl ethCaller) callFunc {
	return func(ctx context.Context, size int) error {
		res, err := cl.CallContract(ctx, ethereum.CallMsg{Data: requestResultSize(size)}, nil)
		if err == nil && len(res) != size/2 {
			return errors.New("result has the wrong size")
		}
		return err
	}
}

// dialers are the ways op-service dials an RPC endpoint. Each dials url and returns a
// callFunc on the connection.
var dialers = []struct {
	name string
	dial func(t *testing.T, ctx context.Context, lgr log.Logger, url string) callFunc
}{
	{"NewRPC", func(t *testing.T, ctx context.Context, lgr log.Logger, url string) callFunc {
		cl, err := client.NewRPC(ctx, lgr, url, client.WithCallTimeout(callTimeout))
		require.NoError(t, err)
		t.Cleanup(cl.Close)
		return rpcCaller(cl)
	}},
	{"NewRPC lazy", func(t *testing.T, ctx context.Context, lgr log.Logger, url string) callFunc {
		cl, err := client.NewRPC(ctx, lgr, url, client.WithLazyDial(), client.WithCallTimeout(callTimeout))
		require.NoError(t, err)
		t.Cleanup(cl.Close)
		return rpcCaller(cl)
	}},
	{"dial.DialEthClientWithTimeout", func(t *testing.T, ctx context.Context, lgr log.Logger, url string) callFunc {
		cl, err := dial.DialEthClientWithTimeout(ctx, dial.DefaultDialTimeout, lgr, url)
		require.NoError(t, err)
		t.Cleanup(cl.Close)
		return ethClientCaller(cl)
	}},
	{"txmgr.NewConfig", func(t *testing.T, ctx context.Context, lgr log.Logger, url string) callFunc {
		cliCfg := txmgr.NewCLIConfig(url, txmgr.DefaultBatcherFlagValues)
		cliCfg.PrivateKey = testPrivateKey
		cfg, err := txmgr.NewConfig(cliCfg, lgr)
		require.NoError(t, err)
		t.Cleanup(cfg.Backend.Close)
		return ethClientCaller(cfg.Backend)
	}},
}

// TestResponseSizeParity checks that responses above geth's default websocket limit arrive
// over websocket as they do over HTTP, however op-service dials the endpoint.
func TestResponseSizeParity(t *testing.T) {
	httpURL, wsURL := startServer(t)
	transports := []struct{ name, url string }{{"http", httpURL}, {"ws", wsURL}}
	for _, d := range dialers {
		t.Run(d.name, func(t *testing.T) {
			for _, tr := range transports {
				t.Run(tr.name, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
					defer cancel()
					call := d.dial(t, ctx, testlog.Logger(t, slog.LevelInfo), tr.url)
					require.NoError(t, call(ctx, responseSize))
				})
			}
		})
	}
}

// TestWebsocketLimitOverride checks a caller-supplied limit replaces the default and is enforced.
func TestWebsocketLimitOverride(t *testing.T) {
	_, wsURL := startServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cl, err := client.NewRPC(ctx, testlog.Logger(t, slog.LevelInfo), wsURL,
		client.WithGethRPCOptions(rpc.WithWebsocketMessageSizeLimit(1024)))
	require.NoError(t, err)
	t.Cleanup(cl.Close)
	call := rpcCaller(cl)
	require.NoError(t, call(ctx, 512))
	require.ErrorContains(t, call(ctx, 2048), "read limit exceeded")
}
