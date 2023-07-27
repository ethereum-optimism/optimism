package integration_tests

import (
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// TestConcurrentWSPanic tests for a panic in the websocket proxy
// that occurred when messages were sent from the upstream to the
// client right after the client sent an invalid request.
func TestConcurrentWSPanic(t *testing.T) {
	var backendToProxyConn *websocket.Conn
	var setOnce sync.Once

	readyCh := make(chan struct{}, 1)
	quitC := make(chan struct{})

	// Pull out the backend -> proxyd conn so that we can spam it directly.
	// Use a sync.Once to make sure we only do that once, for the first
	// connection.
	backend := NewMockWSBackend(func(conn *websocket.Conn) {
		setOnce.Do(func() {
			backendToProxyConn = conn
			readyCh <- struct{}{}
		})
	}, nil, nil)
	defer backend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", backend.URL()))

	config := ReadConfig("ws")
	_, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	client, err := NewProxydWSClient("ws://127.0.0.1:8546", nil, nil)
	require.NoError(t, err)
	defer shutdown()

	// suppress tons of log messages
	oldHandler := log.Root().GetHandler()
	log.Root().SetHandler(log.DiscardHandler())
	defer func() {
		log.Root().SetHandler(oldHandler)
	}()

	<-readyCh

	var wg sync.WaitGroup
	wg.Add(2)
	// spam messages
	go func() {
		for {
			select {
			case <-quitC:
				wg.Done()
				return
			default:
				_ = backendToProxyConn.WriteMessage(websocket.TextMessage, []byte("garbage"))
			}
		}
	}()

	// spam invalid RPCs
	go func() {
		for {
			select {
			case <-quitC:
				wg.Done()
				return
			default:
				_ = client.WriteMessage(websocket.TextMessage, []byte("{\"id\": 1, \"method\": \"eth_foo\", \"params\": [\"newHeads\"]}"))
			}
		}
	}()

	// 1 second is enough to trigger the panic due to
	// concurrent write to websocket connection
	time.Sleep(time.Second)
	close(quitC)
	wg.Wait()
}

type backendHandler struct {
	msgCB   atomic.Value
	closeCB atomic.Value
}

func (b *backendHandler) MsgCB(conn *websocket.Conn, msgType int, data []byte) {
	cb := b.msgCB.Load()
	if cb == nil {
		return
	}
	cb.(MockWSBackendOnMessage)(conn, msgType, data)
}

func (b *backendHandler) SetMsgCB(cb MockWSBackendOnMessage) {
	b.msgCB.Store(cb)
}

func (b *backendHandler) CloseCB(conn *websocket.Conn, err error) {
	cb := b.closeCB.Load()
	if cb == nil {
		return
	}
	cb.(MockWSBackendOnClose)(conn, err)
}

func (b *backendHandler) SetCloseCB(cb MockWSBackendOnClose) {
	b.closeCB.Store(cb)
}

type clientHandler struct {
	msgCB atomic.Value
}

func (c *clientHandler) MsgCB(msgType int, data []byte) {
	cb := c.msgCB.Load().(ProxydWSClientOnMessage)
	if cb == nil {
		return
	}
	cb(msgType, data)
}

func (c *clientHandler) SetMsgCB(cb ProxydWSClientOnMessage) {
	c.msgCB.Store(cb)
}

func TestWS(t *testing.T) {
	backendHdlr := new(backendHandler)
	clientHdlr := new(clientHandler)

	backend := NewMockWSBackend(nil, func(conn *websocket.Conn, msgType int, data []byte) {
		backendHdlr.MsgCB(conn, msgType, data)
	}, func(conn *websocket.Conn, err error) {
		backendHdlr.CloseCB(conn, err)
	})
	defer backend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", backend.URL()))

	config := ReadConfig("ws")
	_, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	client, err := NewProxydWSClient("ws://127.0.0.1:8546", func(msgType int, data []byte) {
		clientHdlr.MsgCB(msgType, data)
	}, nil)
	defer client.HardClose()
	require.NoError(t, err)
	defer shutdown()

	tests := []struct {
		name       string
		backendRes string
		expRes     string
		clientReq  string
	}{
		{
			"ok response",
			"{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0xcd0c3e8af590364c09d0fa6a1210faf5\"}",
			"{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":\"0xcd0c3e8af590364c09d0fa6a1210faf5\"}",
			"{\"id\": 1, \"method\": \"eth_subscribe\", \"params\": [\"newHeads\"]}",
		},
		{
			"garbage backend response",
			"gibblegabble",
			"{\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32013,\"message\":\"backend returned an invalid response\"},\"id\":null}",
			"{\"id\": 1, \"method\": \"eth_subscribe\", \"params\": [\"newHeads\"]}",
		},
		{
			"blacklisted RPC",
			"}",
			"{\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32001,\"message\":\"rpc method is not whitelisted\"},\"id\":1}",
			"{\"id\": 1, \"method\": \"eth_whatever\", \"params\": []}",
		},
		{
			"garbage client request",
			"{}",
			"{\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32700,\"message\":\"parse error\"},\"id\":null}",
			"barf",
		},
		{
			"invalid client request",
			"{}",
			"{\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32700,\"message\":\"parse error\"},\"id\":null}",
			"{\"jsonrpc\": \"2.0\", \"method\": true}",
		},
		{
			"eth_accounts",
			"{}",
			"{\"jsonrpc\":\"2.0\",\"result\":[],\"id\":1}",
			"{\"jsonrpc\": \"2.0\", \"method\": \"eth_accounts\", \"id\": 1}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := time.NewTicker(10 * time.Second)
			doneCh := make(chan struct{}, 1)
			backendHdlr.SetMsgCB(func(conn *websocket.Conn, msgType int, data []byte) {
				require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(tt.backendRes)))
			})
			clientHdlr.SetMsgCB(func(msgType int, data []byte) {
				require.Equal(t, tt.expRes, string(data))
				doneCh <- struct{}{}
			})
			require.NoError(t, client.WriteMessage(
				websocket.TextMessage,
				[]byte(tt.clientReq),
			))
			select {
			case <-timeout.C:
				t.Fatalf("timed out")
			case <-doneCh:
				return
			}
		})
	}
}

func TestWSClientClosure(t *testing.T) {
	backendHdlr := new(backendHandler)
	clientHdlr := new(clientHandler)

	backend := NewMockWSBackend(nil, func(conn *websocket.Conn, msgType int, data []byte) {
		backendHdlr.MsgCB(conn, msgType, data)
	}, func(conn *websocket.Conn, err error) {
		backendHdlr.CloseCB(conn, err)
	})
	defer backend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", backend.URL()))

	config := ReadConfig("ws")
	_, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	for _, closeType := range []string{"soft", "hard"} {
		t.Run(closeType, func(t *testing.T) {
			client, err := NewProxydWSClient("ws://127.0.0.1:8546", func(msgType int, data []byte) {
				clientHdlr.MsgCB(msgType, data)
			}, nil)
			require.NoError(t, err)

			timeout := time.NewTicker(30 * time.Second)
			doneCh := make(chan struct{}, 1)
			backendHdlr.SetCloseCB(func(conn *websocket.Conn, err error) {
				doneCh <- struct{}{}
			})

			if closeType == "soft" {
				require.NoError(t, client.SoftClose())
			} else {
				client.HardClose()
			}

			select {
			case <-timeout.C:
				t.Fatalf("timed out")
			case <-doneCh:
				return
			}
		})
	}
}

func TestWSClientExceedReadLimit(t *testing.T) {
	backendHdlr := new(backendHandler)
	clientHdlr := new(clientHandler)

	backend := NewMockWSBackend(nil, func(conn *websocket.Conn, msgType int, data []byte) {
		backendHdlr.MsgCB(conn, msgType, data)
	}, func(conn *websocket.Conn, err error) {
		backendHdlr.CloseCB(conn, err)
	})
	defer backend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", backend.URL()))

	config := ReadConfig("ws")
	_, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	client, err := NewProxydWSClient("ws://127.0.0.1:8546", func(msgType int, data []byte) {
		clientHdlr.MsgCB(msgType, data)
	}, nil)
	require.NoError(t, err)

	closed := false
	originalHandler := client.conn.CloseHandler()
	client.conn.SetCloseHandler(func(code int, text string) error {
		closed = true
		return originalHandler(code, text)
	})

	backendHdlr.SetMsgCB(func(conn *websocket.Conn, msgType int, data []byte) {
		t.Fatalf("backend should not get the large message")
	})

	clientReq := "{\"id\": 1, \"method\": \"eth_subscribe\", \"params\": [\"" + strings.Repeat("barf", 256*opt.KiB+1) + "\"]}"
	err = client.WriteMessage(
		websocket.TextMessage,
		[]byte(clientReq),
	)
	require.Error(t, err)
	require.True(t, closed)

}
