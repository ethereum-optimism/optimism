package integration_tests

import (
	"bufio"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
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
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	client, err := NewProxydWSClient("ws://127.0.0.1:8546", nil, nil, nil)
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
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	client, err := NewProxydWSClient("ws://127.0.0.1:8546", nil, func(msgType int, data []byte) {
		clientHdlr.MsgCB(msgType, data)
	}, nil)
	defer client.HardClose()
	require.NoError(t, err)

	f, err := os.Open("testdata/ws_testdata.txt")
	require.NoError(t, err)
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan() // skip header
	for scanner.Scan() {
		record := strings.Split(scanner.Text(), "|")
		name, body, responseBody, expResponseBody := record[0], record[1], record[2], record[3]
		if expResponseBody == "" {
			expResponseBody = responseBody
		}
		require.NoError(t, err)
		t.Run(name, func(t *testing.T) {
			res := spamWSReqs(t, clientHdlr, backendHdlr, client, []byte(body), []byte(responseBody), 1)
			require.NoError(t, err)
			require.Equal(t, 1, res[expResponseBody])
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
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	for _, closeType := range []string{"soft", "hard"} {
		t.Run(closeType, func(t *testing.T) {
			client, err := NewProxydWSClient("ws://127.0.0.1:8546", nil, func(msgType int, data []byte) {
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

func TestWSClientMaxConns(t *testing.T) {
	backend := NewMockWSBackend(nil, nil, nil)
	defer backend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", backend.URL()))

	config := ReadConfig("ws")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	doneCh := make(chan struct{}, 1)
	_, err = NewProxydWSClient("ws://127.0.0.1:8546", nil, nil, nil)
	require.NoError(t, err)
	_, err = NewProxydWSClient("ws://127.0.0.1:8546", nil, nil, func(err error) {
		require.Contains(t, err.Error(), "unexpected EOF")
		doneCh <- struct{}{}
	})
	require.NoError(t, err)

	timeout := time.NewTicker(30 * time.Second)
	select {
	case <-timeout.C:
		t.Fatalf("timed out")
	case <-doneCh:
		return
	}
}

var sampleRequest = []byte("{\"jsonrpc\": \"2.0\", \"method\": \"eth_accounts\", \"id\": 1}")

func TestWSClientMaxRPSLimit(t *testing.T) {
	backendHdlr := new(backendHandler)
	clientHdlr := new(clientHandler)

	backend := NewMockWSBackend(nil, func(conn *websocket.Conn, msgType int, data []byte) {
		backendHdlr.MsgCB(conn, msgType, data)
	}, func(conn *websocket.Conn, err error) {
		backendHdlr.CloseCB(conn, err)
	})
	defer backend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", backend.URL()))

	config := ReadConfig("ws_frontend_rate_limit")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	t.Run("non-exempt over limit", func(t *testing.T) {
		client, err := NewProxydWSClient("ws://127.0.0.1:8546", nil, func(msgType int, data []byte) {
			clientHdlr.MsgCB(msgType, data)
		}, nil)
		defer client.HardClose()
		require.NoError(t, err)
		res := spamWSReqs(t, clientHdlr, backendHdlr, client, sampleRequest, []byte(""), 4)
		require.Equal(t, 3, res[invalidRateLimitResponse])
	})

	t.Run("exempt user agent over limit", func(t *testing.T) {
		h := make(http.Header)
		h.Set("User-Agent", "exempt_agent")
		client, err := NewProxydWSClient("ws://127.0.0.1:8546", h, func(msgType int, data []byte) {
			clientHdlr.MsgCB(msgType, data)
		}, nil)
		defer client.HardClose()
		require.NoError(t, err)
		res := spamWSReqs(t, clientHdlr, backendHdlr, client, sampleRequest, []byte(""), 4)
		require.Equal(t, 0, res[invalidRateLimitResponse])
	})

	t.Run("exempt origin over limit", func(t *testing.T) {
		h := make(http.Header)
		// In gorilla/websocket, the Origin header must be the same as the URL.
		// Otherwise, it will be rejected
		h.Set("Origin", "wss://127.0.0.1:8546")
		client, err := NewProxydWSClient("ws://127.0.0.1:8546", h, func(msgType int, data []byte) {
			clientHdlr.MsgCB(msgType, data)
		}, nil)
		defer client.HardClose()
		require.NoError(t, err)
		res := spamWSReqs(t, clientHdlr, backendHdlr, client, sampleRequest, []byte(""), 4)
		require.Equal(t, 0, res[invalidRateLimitResponse])
	})

	t.Run("multiple xff", func(t *testing.T) {
		h1 := make(http.Header)
		h1.Set("X-Forwarded-For", "1.1.1.1")
		h2 := make(http.Header)
		h2.Set("X-Forwarded-For", "2.2.2.2")
		client1, _ := NewProxydWSClient("ws://127.0.0.1:8546", h1, func(msgType int, data []byte) {
			clientHdlr.MsgCB(msgType, data)
		}, nil)
		defer client1.HardClose()
		client2, _ := NewProxydWSClient("ws://127.0.0.1:8546", h2, func(msgType int, data []byte) {
			clientHdlr.MsgCB(msgType, data)
		}, nil)
		defer client2.HardClose()
		res1 := spamWSReqs(t, clientHdlr, backendHdlr, client1, sampleRequest, []byte(""), 4)
		res2 := spamWSReqs(t, clientHdlr, backendHdlr, client2, sampleRequest, []byte(""), 4)
		require.Equal(t, 3, res1[invalidRateLimitResponse])
		require.Equal(t, 3, res2[invalidRateLimitResponse])
	})
}

func TestWSSenderRateLimitLimiting(t *testing.T) {
	backendHdlr := new(backendHandler)
	clientHdlr := new(clientHandler)

	backend := NewMockWSBackend(nil, func(conn *websocket.Conn, msgType int, data []byte) {
		backendHdlr.MsgCB(conn, msgType, data)
	}, func(conn *websocket.Conn, err error) {
		backendHdlr.CloseCB(conn, err)
	})
	defer backend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", backend.URL()))

	config := ReadConfig("ws_sender_rate_limit")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	// Two separate requests from the same sender
	// should be rate limited.
	client, err := NewProxydWSClient("ws://127.0.0.1:8546", nil, func(msgType int, data []byte) {
		clientHdlr.MsgCB(msgType, data)
	}, nil)
	defer client.HardClose()
	require.NoError(t, err)
	res := spamWSReqs(t, clientHdlr, backendHdlr, client, makeSendRawTransaction(txHex1), []byte(""), 4)
	require.Equal(t, 3, res[invalidSenderRateLimitResponse])

	// Clear the limiter.
	time.Sleep(1100 * time.Millisecond)

	// Two separate requests from different senders
	// should not be rate limited.
	res1 := spamWSReqs(t, clientHdlr, backendHdlr, client, makeSendRawTransaction(txHex1), []byte(""), 4)
	res2 := spamWSReqs(t, clientHdlr, backendHdlr, client, makeSendRawTransaction(txHex2), []byte(""), 4)
	require.Equal(t, 3, res1[invalidSenderRateLimitResponse])
	require.Equal(t, 3, res2[invalidSenderRateLimitResponse])
}

func spamWSReqs(t *testing.T, clientHdlr *clientHandler, backendHdlr *backendHandler, client *ProxydWSClient, request []byte, response []byte, n int) map[string]int {
	resCh := make(chan string)
	for i := 0; i < n; i++ {
		go func() {
			backendHdlr.SetMsgCB(func(conn *websocket.Conn, msgType int, data []byte) {
				require.NoError(t, conn.WriteMessage(websocket.TextMessage, response))
			})
			clientHdlr.SetMsgCB(func(msgType int, data []byte) {
				resCh <- string(data)
			})
			require.NoError(t, client.WriteMessage(
				websocket.TextMessage,
				[]byte(request),
			))
		}()
	}

	resMapping := make(map[string]int)
	for i := 0; i < n; i++ {
		res := <-resCh
		response := res
		resMapping[response]++
	}

	return resMapping
}
