package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/jsonrpc"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

type recordedCall struct {
	msg  jsonrpc.Message
	resp *jsonrpc.Response
}

type capturingRecorder struct {
	mu       sync.Mutex
	outgoing []*recordedCall
	incoming []jsonrpc.Message
}

func (r *capturingRecorder) RecordIncoming(_ context.Context, msg jsonrpc.Message) jsonrpc.RecordDone {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.incoming = append(r.incoming, msg)
	return nil
}

func (r *capturingRecorder) RecordOutgoing(_ context.Context, msg jsonrpc.Message) jsonrpc.RecordDone {
	r.mu.Lock()
	defer r.mu.Unlock()
	call := &recordedCall{msg: msg}
	r.outgoing = append(r.outgoing, call)
	return func(_ context.Context, resp jsonrpc.Response) {
		r.mu.Lock()
		defer r.mu.Unlock()
		if call.resp != nil {
			panic("response recorded twice")
		}
		call.resp = &resp
	}
}

func (r *capturingRecorder) calls() []*recordedCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*recordedCall(nil), r.outgoing...)
}

func (r *capturingRecorder) notifications() []jsonrpc.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]jsonrpc.Message(nil), r.incoming...)
}

type echoResult struct {
	A int    `json:"a"`
	B string `json:"b"`
}

type testService struct {
	release chan struct{}
}

func (s *testService) Echo(a int, b string) echoResult {
	return echoResult{A: a, B: b}
}

func (s *testService) Nothing() {}

func (s *testService) Fail() error {
	return &jsonrpc.Error{Code: -39001, Message: "boom", Data: "details"}
}

func (s *testService) Block(ctx context.Context) error {
	select {
	case <-s.release:
	case <-ctx.Done():
	}
	return nil
}

func (s *testService) Count(ctx context.Context, n int) (*rpc.Subscription, error) {
	notifier, ok := rpc.NotifierFromContext(ctx)
	if !ok {
		return nil, rpc.ErrNotificationsUnsupported
	}
	sub := notifier.CreateSubscription()
	go func() {
		for i := range n {
			if err := notifier.Notify(sub.ID, i); err != nil {
				return
			}
		}
	}()
	return sub, nil
}

func newRecordingTestClient(t *testing.T) (client.RPC, *capturingRecorder) {
	srv := rpc.NewServer()
	svc := &testService{release: make(chan struct{})}
	require.NoError(t, srv.RegisterName("test", svc))
	rec := new(capturingRecorder)
	cl := client.NewBaseRPCClient(rpc.DialInProc(srv), client.WithRPCRecorder(rec))
	t.Cleanup(func() {
		close(svc.release)
		cl.Close()
		srv.Stop()
	})
	return cl, rec
}

func TestRecordingCall(t *testing.T) {
	cl, rec := newRecordingTestClient(t)
	ctx := context.Background()

	var res echoResult
	require.NoError(t, cl.CallContext(ctx, &res, "test_echo", 42, "hi"))
	require.Equal(t, echoResult{A: 42, B: "hi"}, res)

	require.NoError(t, cl.CallContext(ctx, nil, "test_nothing"))

	calls := rec.calls()
	require.Len(t, calls, 2)
	require.Equal(t, jsonrpc.Message{Method: "test_echo", Params: json.RawMessage(`[42,"hi"]`)}, calls[0].msg)
	require.Equal(t, &jsonrpc.Response{Result: json.RawMessage(`{"a":42,"b":"hi"}`)}, calls[0].resp)
	require.Equal(t, jsonrpc.Message{Method: "test_nothing"}, calls[1].msg)
	require.Equal(t, &jsonrpc.Response{Result: json.RawMessage(`null`)}, calls[1].resp)
}

func TestRecordingCallError(t *testing.T) {
	cl, rec := newRecordingTestClient(t)

	err := cl.CallContext(context.Background(), nil, "test_fail")
	var rpcErr rpc.Error
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, -39001, rpcErr.ErrorCode())
	require.Equal(t, "boom", rpcErr.Error())
	var dataErr rpc.DataError
	require.ErrorAs(t, err, &dataErr)
	require.Equal(t, "details", dataErr.ErrorData())

	err = cl.CallContext(context.Background(), nil, "test_unknown")
	require.ErrorAs(t, err, &rpcErr)

	calls := rec.calls()
	require.Len(t, calls, 2)
	require.Equal(t, &jsonrpc.Response{Error: &jsonrpc.Error{Code: -39001, Message: "boom", Data: "details"}}, calls[0].resp)
	require.NotNil(t, calls[1].resp)
	require.Equal(t, -32601, calls[1].resp.Error.Code)
}

func TestRecordingCallWithoutResponse(t *testing.T) {
	cl, rec := newRecordingTestClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := cl.CallContext(ctx, nil, "test_block")
	require.ErrorIs(t, err, context.DeadlineExceeded)

	calls := rec.calls()
	require.Len(t, calls, 1)
	require.Equal(t, "test_block", calls[0].msg.Method)
	require.Nil(t, calls[0].resp, "no response must be recorded without one arriving")
}

func TestRecordingBatchCall(t *testing.T) {
	cl, rec := newRecordingTestClient(t)

	var res echoResult
	batch := []rpc.BatchElem{
		{Method: "test_echo", Args: []any{7, "x"}, Result: &res},
		{Method: "test_fail", Result: new(any)},
	}
	require.NoError(t, cl.BatchCallContext(context.Background(), batch))
	require.NoError(t, batch[0].Error)
	require.Equal(t, echoResult{A: 7, B: "x"}, res)
	var rpcErr rpc.Error
	require.ErrorAs(t, batch[1].Error, &rpcErr)
	require.Equal(t, -39001, rpcErr.ErrorCode())

	calls := rec.calls()
	require.Len(t, calls, 2)
	require.Equal(t, jsonrpc.Message{Method: "test_echo", Params: json.RawMessage(`[7,"x"]`)}, calls[0].msg)
	require.Equal(t, &jsonrpc.Response{Result: json.RawMessage(`{"a":7,"b":"x"}`)}, calls[0].resp)
	require.Equal(t, jsonrpc.Message{Method: "test_fail"}, calls[1].msg)
	require.Equal(t, -39001, calls[1].resp.Error.Code)
}

func TestRecordingSubscriptionNotRecorded(t *testing.T) {
	cl, rec := newRecordingTestClient(t)

	ch := make(chan int)
	sub, err := cl.Subscribe(context.Background(), "test", ch, "count", 2)
	require.NoError(t, err)
	defer sub.Unsubscribe()
	for want := range 2 {
		select {
		case got := <-ch:
			require.Equal(t, want, got)
		case err := <-sub.Err():
			t.Fatalf("subscription failed: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for notification")
		}
	}
	require.Empty(t, rec.calls())
	require.Empty(t, rec.notifications())
}

func TestRecordingCallNonPointerResult(t *testing.T) {
	cl, rec := newRecordingTestClient(t)

	var res echoResult
	require.Error(t, cl.CallContext(context.Background(), res, "test_echo", 1, "a"))
	require.Empty(t, rec.calls(), "a rejected call must not be sent")
}

func TestRecordingCallNoResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s}`, req.ID)
	}))
	t.Cleanup(srv.Close)
	gethCl, err := rpc.DialHTTP(srv.URL)
	require.NoError(t, err)
	rec := new(capturingRecorder)
	cl := client.NewBaseRPCClient(gethCl, client.WithRPCRecorder(rec))
	t.Cleanup(cl.Close)

	err = cl.CallContext(context.Background(), new(string), "test_empty")
	require.ErrorIs(t, err, rpc.ErrNoResult)
	calls := rec.calls()
	require.Len(t, calls, 1)
	require.Equal(t, &jsonrpc.Response{}, calls[0].resp)
}

func TestRecordingBatchCallWithoutResponse(t *testing.T) {
	cl, rec := newRecordingTestClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	batch := []rpc.BatchElem{{Method: "test_echo", Args: []any{1, "a"}, Result: new(echoResult)}}
	require.ErrorIs(t, cl.BatchCallContext(ctx, batch), context.Canceled)

	calls := rec.calls()
	require.Len(t, calls, 1)
	require.Nil(t, calls[0].resp)
}

func TestNoRecordingWithoutMetrics(t *testing.T) {
	srv := rpc.NewServer()
	t.Cleanup(srv.Stop)
	cl := client.NewBaseRPCClient(rpc.DialInProc(srv),
		client.WithRPCRecorder(new(metrics.NoopRPCMetrics).NewRecorder("test")))
	t.Cleanup(cl.Close)
	require.IsType(t, &client.BaseRPCClient{}, cl, "no-op metrics must not install the recording wrapper")
}
