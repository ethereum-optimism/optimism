package ws

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/log"
)

func TestStopWaitsForRollupBoostListener(t *testing.T) {
	connected := make(chan struct{})
	closed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept WebSocket connection: %v", err)
			return
		}
		close(connected)
		defer close(closed)
		defer conn.CloseNow()

		_, _, _ = conn.Read(r.Context())
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, err := opclient.DialWS(t.Context(), opclient.WSConfig{
		URL:         wsURL,
		DialTimeout: time.Second,
		MaxAttempts: 1,
		Log:         log.New(),
	})
	if err != nil {
		t.Fatalf("dial WebSocket server: %v", err)
	}

	handler := &Handler{
		cfg:                    Config{RollupBoostWsURL: wsURL},
		log:                    log.New(),
		isLeaderFn:             func(context.Context) bool { return true },
		metrics:                &metrics.NoopMetricsImpl{},
		initialRollupBoostConn: conn,
	}
	handler.rollupBoostCtx, handler.rollupBoostWsCancel = context.WithCancel(t.Context())
	handler.listenerWG.Add(1)
	go handler.listenToRollupBoost(handler.rollupBoostCtx, handler.initialRollupBoostConn)

	select {
	case <-connected:
	case <-time.After(time.Second):
		t.Fatal("listener did not connect to WebSocket server")
	}

	handler.Stop()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("rollup boost connection was not closed")
	}
}
