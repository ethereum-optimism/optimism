package ws

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
	"github.com/ethereum/go-ethereum/log"
)

// testEventTracker tracks events for testing without duplicating hub logic
type testEventTracker struct {
	clientRegistered   chan *Client
	clientUnregistered chan *Client
	messagesReceived   chan []byte
	shutdownComplete   chan struct{}
	clientCount        int32
}

func newTestEventTracker() *testEventTracker {
	return &testEventTracker{
		clientRegistered:   make(chan *Client, 10),
		clientUnregistered: make(chan *Client, 10),
		messagesReceived:   make(chan []byte, 100),
		shutdownComplete:   make(chan struct{}),
	}
}

func (t *testEventTracker) onClientRegistered(client *Client) {
	atomic.AddInt32(&t.clientCount, 1)
	select {
	case t.clientRegistered <- client:
	default:
	}
}

func (t *testEventTracker) onClientUnregistered(client *Client) {
	atomic.AddInt32(&t.clientCount, -1)
	select {
	case t.clientUnregistered <- client:
	default:
	}
}

func (t *testEventTracker) onMessageBroadcast(message []byte, successCount, dropCount int) {
	select {
	case t.messagesReceived <- message:
	default:
	}
}

func (t *testEventTracker) onShutdown() {
	close(t.shutdownComplete)
}

func (t *testEventTracker) getClientCount() int {
	return int(atomic.LoadInt32(&t.clientCount))
}

// testClient wraps Client with additional test functionality
type testClient struct {
	conn             *websocket.Conn
	messagesReceived chan []byte
	pingsReceived    chan struct{}
	pongsReceived    chan struct{}
	ctx              context.Context
	cancel           context.CancelFunc
}

func newTestClient(ctx context.Context, wsURL string) (*testClient, error) {
	ctx, cancel := context.WithCancel(ctx)
	tc := &testClient{
		messagesReceived: make(chan []byte, 100),
		pingsReceived:    make(chan struct{}, 10),
		pongsReceived:    make(chan struct{}, 10),
		ctx:              ctx,
		cancel:           cancel,
	}

	opts := &websocket.DialOptions{
		OnPingReceived: func(ctx context.Context, payload []byte) bool {
			select {
			case tc.pingsReceived <- struct{}{}:
			default:
			}
			return true // Send pong response
		},
		OnPongReceived: func(ctx context.Context, payload []byte) {
			select {
			case tc.pongsReceived <- struct{}{}:
			default:
			}
		},
	}

	conn, resp, err := websocket.Dial(ctx, wsURL, opts)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		cancel()
		return nil, err
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		_ = conn.CloseNow()
		cancel()
		return nil, errors.New("unexpected status code")
	}

	tc.conn = conn

	// Start message reader
	go tc.readMessages()

	return tc, nil
}

func (tc *testClient) readMessages() {
	defer func() {
		_ = tc.conn.CloseNow()
	}()
	defer tc.cancel()

	for {
		select {
		case <-tc.ctx.Done():
			return
		default:
			_, message, err := tc.conn.Read(tc.ctx)
			if err != nil {
				// Connection closed or other error - return immediately
				return
			}
			select {
			case tc.messagesReceived <- message:
			default:
				// Buffer full, drop message
			}
		}
	}
}

func (tc *testClient) Close() error {
	tc.cancel()
	return tc.conn.Close(websocket.StatusNormalClosure, "test complete")
}

func (tc *testClient) Ping(ctx context.Context) error {
	return tc.conn.Ping(ctx)
}

func (tc *testClient) Write(ctx context.Context, data []byte) error {
	return tc.conn.Write(ctx, websocket.MessageText, data)
}

// setupTestServer creates a test WebSocket server with event tracking.
func setupTestServer(t *testing.T) (*Handler, *testEventTracker, *httptest.Server, func()) {
	t.Helper()

	cfg := Config{
		WebsocketServerPort: 8080,
		RollupBoostWsURL:    "ws://mock-url",
	}

	logger := log.New("test", "websocket")
	isLeaderFn := func(ctx context.Context) bool { return true }

	handler := &Handler{
		cfg:          cfg,
		log:          logger,
		isLeaderFn:   isLeaderFn,
		metrics:      &metrics.NoopMetricsImpl{},
		pingInterval: defaultPingInterval,
		pongTimeout:  defaultPongTimeout,
	}

	// Create event tracker for testing
	tracker := newTestEventTracker()

	// Create hub with test callbacks
	callbacks := HubCallbacks{
		OnClientRegistered:   tracker.onClientRegistered,
		OnClientUnregistered: tracker.onClientUnregistered,
		OnMessageBroadcast:   tracker.onMessageBroadcast,
		OnShutdown:           tracker.onShutdown,
	}

	handler.hub = newHubWithCallbacks(handler.metrics, callbacks)
	go handler.hub.run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.handleWebSocket)
	server := httptest.NewServer(mux)

	cleanup := func() {
		select {
		case <-handler.hub.done:
		default:
			close(handler.hub.done)
		}
		server.Close()
	}

	return handler, tracker, server, cleanup
}

// waitForClientCount waits for the expected number of clients using events
func waitForClientCount(t *testing.T, tracker *testEventTracker, expected int, timeout time.Duration, msg string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for tracker.getClientCount() != expected {
		select {
		case <-ctx.Done():
			t.Fatalf("%s: timeout waiting for %d clients, got %d", msg, expected, tracker.getClientCount())
		case <-tracker.clientRegistered:
			// Client registered, check count
		case <-tracker.clientUnregistered:
			// Client unregistered, check count
		}
	}
}

// waitForMessage waits for a specific message with timeout
func waitForMessage(t *testing.T, client *testClient, expected string, timeout time.Duration, msg string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("%s: timeout waiting for message %q", msg, expected)
		case message := <-client.messagesReceived:
			if string(message) == expected {
				return // Found the expected message
			}
			// Continue waiting for the right message
		}
	}
}

// TestPingPongMechanism tests the actual ping/pong keepalive mechanism
func TestPingPongMechanism(t *testing.T) {
	_, tracker, server, cleanup := setupTestServer(t)
	defer cleanup()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test client
	client, err := newTestClient(ctx, wsURL)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	defer client.Close()

	// Wait for client registration
	waitForClientCount(t, tracker, 1, 2*time.Second, "Initial connection")

	// Send ping from client to server
	err = client.Ping(ctx)
	if err != nil {
		t.Fatalf("Failed to send ping: %v", err)
	}

	// Wait for pong response
	select {
	case <-time.After(3 * time.Second):
		t.Error("Timeout waiting for pong response")
	case <-client.pongsReceived:
		t.Log("Client received pong response")
	}

	// Verify client is still connected
	if tracker.getClientCount() != 1 {
		t.Errorf("Expected 1 client after ping/pong, got %d", tracker.getClientCount())
	}
}

// TestServerInitiatedPing tests that the server sends pings to clients
func TestServerInitiatedPing(t *testing.T) {
	handler, tracker, server, cleanup := setupTestServer(t)
	defer cleanup()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), handler.pingInterval+(5*time.Second))
	defer cancel()

	// Create test client
	client, err := newTestClient(ctx, wsURL)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	defer client.Close()

	// Wait for client registration
	waitForClientCount(t, tracker, 1, 2*time.Second, "Initial connection")

	// Wait for server to send a ping
	select {
	case <-time.After(handler.pingInterval + (2 * time.Second)):
		t.Error("Timeout waiting for server ping")
	case <-client.pingsReceived:
		t.Log("Client received ping from server")
	}
}

// TestClientTimeout tests what happens when a client doesn't respond
func TestClientTimeout(t *testing.T) {
	_, tracker, server, cleanup := setupTestServer(t)
	defer cleanup()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Create client connection but don't process messages
	conn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("Expected status code %d, got %d", http.StatusSwitchingProtocols, resp.StatusCode)
	}

	// Wait for client registration
	waitForClientCount(t, tracker, 1, 2*time.Second, "Initial connection")

	// Abruptly close the connection to simulate unresponsive client
	_ = conn.CloseNow()

	// Wait for client unregistration
	waitForClientCount(t, tracker, 0, 5*time.Second, "Client cleanup after timeout")
}

// TestMultipleClientsBroadcast tests broadcast functionality with multiple clients
func TestMultipleClientsBroadcast(t *testing.T) {
	handler, tracker, server, cleanup := setupTestServer(t)
	defer cleanup()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	numClients := 3
	var clients []*testClient

	// Connect multiple clients
	for i := 0; i < numClients; i++ {
		client, err := newTestClient(ctx, wsURL)
		if err != nil {
			t.Fatalf("Failed to create client %d: %v", i, err)
		}
		clients = append(clients, client)
	}

	// Clean up connections
	defer func() {
		for _, client := range clients {
			client.Close()
		}
	}()

	// Wait for all clients to be registered
	waitForClientCount(t, tracker, numClients, 5*time.Second, "All clients connected")

	// Send broadcast message
	testMessage := "Hello World!"
	handler.BroadcastMessage([]byte(testMessage))

	// Verify all clients received the message
	for i, client := range clients {
		waitForMessage(t, client, testMessage, 2*time.Second, fmt.Sprintf("Client %d message", i))
	}
}

// TestConcurrentConnections tests concurrent client connections and disconnections
func TestConcurrentConnections(t *testing.T) {
	_, tracker, server, cleanup := setupTestServer(t)
	defer cleanup()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	numClients := 10
	var wg sync.WaitGroup

	// Connect clients concurrently
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientIdx int) {
			defer wg.Done()

			client, err := newTestClient(ctx, wsURL)
			if err != nil {
				t.Errorf("Client %d connection failed: %v", clientIdx, err)
				return
			}
			defer client.Close()

			// Keep connection alive for a short time
			select {
			case <-ctx.Done():
			case <-time.After(100 * time.Millisecond):
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Wait for all clients to disconnect
	waitForClientCount(t, tracker, 0, 5*time.Second, "All clients disconnected")
}

// TestBroadcastWithSlowClient tests broadcast behavior when one client is slow
func TestBroadcastWithSlowClient(t *testing.T) {
	handler, tracker, server, cleanup := setupTestServer(t)
	defer cleanup()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create fast client
	fastClient, err := newTestClient(ctx, wsURL)
	if err != nil {
		t.Fatalf("Failed to create fast client: %v", err)
	}
	defer fastClient.Close()

	// Create slow client (don't read messages)
	slowConn, resp, err := websocket.Dial(ctx, wsURL, nil)
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("Failed to create slow client: %v", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("Expected status code %d, got %d", http.StatusSwitchingProtocols, resp.StatusCode)
	}
	defer func() {
		_ = slowConn.CloseNow()
	}()

	// Wait for both clients to be registered
	waitForClientCount(t, tracker, 2, 3*time.Second, "Both clients connected")

	// Send many messages to fill up the slow client's buffer
	for i := 0; i < 300; i++ {
		message := []byte(fmt.Sprintf("Large message to fill buffer %d", i))
		handler.BroadcastMessage(message)
	}

	// Fast client should still receive some messages
	select {
	case <-time.After(2 * time.Second):
		t.Error("Fast client didn't receive any messages")
	case <-fastClient.messagesReceived:
		t.Log("Fast client received messages despite slow client")
	}

	// Both clients should still be connected initially
	if tracker.getClientCount() != 2 {
		t.Logf("Expected 2 clients, got %d (slow client may have been cleaned up)", tracker.getClientCount())
	}
}

// TestHubShutdown tests graceful hub shutdown
func TestHubShutdown(t *testing.T) {
	_, tracker, server, cleanup := setupTestServer(t)
	defer cleanup()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect a client
	client, err := newTestClient(ctx, wsURL)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	defer client.Close()

	// Wait for client registration
	waitForClientCount(t, tracker, 1, 2*time.Second, "Client connected before shutdown")

	// Trigger shutdown - this is done by the cleanup function
	// But we can test it explicitly by calling the cleanup early
	cleanup()

	// Wait for shutdown completion
	select {
	case <-tracker.shutdownComplete:
		// Hub has completed shutdown
	case <-time.After(2 * time.Second):
		t.Fatal("Hub shutdown timed out")
	}

	// Verify clients were cleaned up
	if tracker.getClientCount() != 0 {
		t.Errorf("Expected 0 clients after shutdown, got %d", tracker.getClientCount())
	}

	t.Log("Hub shutdown completed successfully")
}

// TestUpstreamPingDetectsAndReconnectsOnSilentConnection verifies that
// pingUpstream detects a TCP-alive / application-silent upstream and
// triggers reconnection within pingInterval+pongTimeout, rather than waiting
// for a 30 s per-Read timeout.
//
// Before the fix, listenToRollupBoost used context.WithTimeout(ctx, 30s) on
// every Read call. coder/websocket destroys the TCP connection when a context
// deadline expires (coder/websocket#242), so the upstream link was reset every
// 30 s even when healthy, causing periodic flashblock gaps. The fix removes the
// read timeout and adds pingUpstream so that only genuinely stale connections
// trigger reconnection.
//
// This test fails without pingUpstream (old code) because a silent connection
// is not detected within the test window; it passes with pingUpstream because
// the pong timeout fires well within the deadline.
func TestUpstreamPingDetectsAndReconnectsOnSilentConnection(t *testing.T) {
	// Use short ping parameters so the test completes in ~400 ms.
	const testPingInterval = 200 * time.Millisecond
	const testPongTimeout = 100 * time.Millisecond

	var connectCount atomic.Int64

	// Upstream server: accepts connections and sends one flashblock message, but
	// never answers pings (simulates a TCP-alive / application-silent link).
	// CloseRead handles the WS close handshake so pingUpstream's conn.Close
	// completes promptly; OnPingReceived=false suppresses pong replies.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
			OnPingReceived:  func(_ context.Context, _ []byte) bool { return false },
		})
		if err != nil {
			return
		}
		defer conn.CloseNow()
		connectCount.Add(1)

		// Send one message so the first connection is productive.
		_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"slot":1}`))

		// CloseRead responds to close frames (so conn.Close in pingUpstream
		// can complete the handshake promptly) but pings go unanswered.
		closed := conn.CloseRead(r.Context())
		<-closed.Done()
	}))
	t.Cleanup(upstream.Close)

	wsURL := "ws" + strings.TrimPrefix(upstream.URL, "http")

	handler := &Handler{
		cfg: Config{
			WebsocketServerPort: 0,
			RollupBoostWsURL:    wsURL,
		},
		log:          log.New("test", "upstream-reconnect"),
		isLeaderFn:   func(context.Context) bool { return true },
		metrics:      &metrics.NoopMetricsImpl{},
		pingInterval: testPingInterval,
		pongTimeout:  testPongTimeout,
	}
	handler.hub = newHub(handler.metrics)
	go handler.hub.run()
	t.Cleanup(func() {
		select {
		case <-handler.hub.done:
		default:
			close(handler.hub.done)
		}
	})

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	go handler.listenToRollupBoost(ctx)

	// Expect ≥2 upstream connections: the initial dial plus one reconnect
	// triggered by the pong timeout (testPingInterval + testPongTimeout ≈ 300 ms).
	deadline := time.NewTimer(testPingInterval + testPongTimeout + 500*time.Millisecond)
	defer deadline.Stop()
	poll := time.NewTicker(20 * time.Millisecond)
	defer poll.Stop()

	for {
		select {
		case <-deadline.C:
			t.Fatalf(
				"upstream reconnect not observed: got %d connections (want ≥2); "+
					"without pingUpstream a silent connection is not detected until the 30 s read timeout",
				connectCount.Load(),
			)
		case <-poll.C:
			if connectCount.Load() >= 2 {
				return // reconnect observed — test passes
			}
		}
	}
}

// TestReadPumpClientRemainsRegisteredDuringPingCycles verifies that a
// subscribe-only client is not dropped by the server during normal operation.
//
// The old readPump implementation issued conn.Read with a 30 s timeout on each
// iteration. When that deadline expired, coder/websocket destroyed the TCP
// connection (coder/websocket#242), unregistering the client even though the
// connection was healthy. The new implementation uses conn.CloseRead, which has
// no per-call timeout and properly handles control frames without destroying
// the underlying connection.
//
// Note: this test validates correct behaviour across multiple ping cycles but
// does not fast-fail with the old 30 s hardcoded timeout (that would require
// waiting >30 s). See TestUpstreamPingDetectsAndReconnectsOnSilentConnection
// for the full root-cause regression test of the timeout-destroys-TCP bug.
func TestReadPumpClientRemainsRegisteredDuringPingCycles(t *testing.T) {
	// Run several server→client ping cycles quickly to exercise the
	// control-frame path through readPump / CloseRead.
	const testPingInterval = 50 * time.Millisecond
	const testPongTimeout = 25 * time.Millisecond

	handler, tracker, server, cleanup := setupTestServer(t)
	handler.pingInterval = testPingInterval
	handler.pongTimeout = testPongTimeout
	defer cleanup()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	client, err := newTestClient(ctx, wsURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer client.Close()

	waitForClientCount(t, tracker, 1, 2*time.Second, "initial connection")

	// Let 5 server-initiated ping/pong cycles complete.
	time.Sleep(5 * testPingInterval)

	if tracker.getClientCount() != 1 {
		t.Fatal("client was unexpectedly unregistered during idle ping cycles")
	}

	// Confirm the client can still receive broadcasts after the idle period.
	const msg = "post-idle-broadcast"
	handler.BroadcastMessage([]byte(msg))
	waitForMessage(t, client, msg, 2*time.Second, "broadcast after ping cycles")
}
