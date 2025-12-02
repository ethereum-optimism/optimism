package supernode

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	rpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// mock runnable activity
type mockRunnable struct {
	started int
	stopped int
}

func (m *mockRunnable) Start(ctx context.Context) error {
	m.started++
	<-ctx.Done()
	return ctx.Err()
}
func (m *mockRunnable) Stop(ctx context.Context) error { m.stopped++; return nil }

// ensure it satisfies both Activity and RunnableActivity
var _ activity.Activity = (*mockRunnable)(nil)
var _ activity.RunnableActivity = (*mockRunnable)(nil)

// plain marker-only activity
type plainActivity struct{}

var _ activity.Activity = (*plainActivity)(nil)

// Start is implemented, but no Stop, so this is not runnable
func (p *plainActivity) Start() { panic("plain activity should not be started") }

// rpc activity
type rpcSvc struct{}

func (s *rpcSvc) Echo(_ context.Context) (string, error) { return "ok", nil }

type rpcAct struct{}

func (a *rpcAct) RPCNamespace() string    { return "act" }
func (a *rpcAct) RPCService() interface{} { return &rpcSvc{} }

var _ activity.Activity = (*rpcAct)(nil)
var _ activity.RPCActivity = (*rpcAct)(nil)

func TestRunnableActivityGating(t *testing.T) {
	t.Parallel()
	run := &mockRunnable{}
	plain := &plainActivity{}

	s := &Supernode{
		log:        gethlog.New(),
		version:    "test",
		chains:     nil,
		activities: []activity.Activity{run, plain},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() { _ = s.Start(ctx); close(done) }()

	<-done // wait until context canceled and Start exits

	require.Equal(t, 1, run.started, "runnable activity should be started exactly once")
	require.Equal(t, 0, run.stopped, "Stop is invoked during Stop(), not here")

	// now stop and ensure Stop was called on runnable activity
	err := s.Stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, run.stopped, "runnable activity should be stopped exactly once")
}

func TestRPCActivityRegistration(t *testing.T) {
	t.Parallel()
	s := &Supernode{
		log:        gethlog.New(),
		version:    "test",
		activities: []activity.Activity{&rpcAct{}},
	}
	// mount root RPC handler
	s.rootRPC = rpc.NewHandler("test")

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	// start to trigger RPC registration
	go func() { _ = s.Start(ctx) }()

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		body := map[string]any{"jsonrpc": "2.0", "id": 1, "method": "act_echo", "params": []any{}}
		raw, _ := json.Marshal(body)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		s.rootRPC.ServeHTTP(rec, req)
		if rec.Code == http.StatusOK {
			var resp struct {
				Result string `json:"result"`
				Error  any    `json:"error"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &resp)
			if resp.Error == nil && resp.Result == "ok" {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("RPC method not available in time, last response: %s", rec.Body.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
}
