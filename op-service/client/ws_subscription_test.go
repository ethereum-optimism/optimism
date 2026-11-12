package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

const (
	testSubscribeMethod = "reth_subscribeTest"
	testSubscribeReply  = `{"jsonrpc":"2.0","id":1,"result":"0x1f"}`
)

func testNotification(result string) string {
	return `{"jsonrpc":"2.0","method":"` + testSubscribeMethod + `","params":{"subscription":"0x1f","result":` + result + `}}`
}

// wsSubscriptionServer accepts one websocket, requires the subscription request for
// testSubscribeMethod, writes messages in order and then stays silent until the client leaves.
func wsSubscriptionServer(t *testing.T, messages ...string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{CompressionMode: websocket.CompressionDisabled})
		if err != nil {
			return
		}
		defer func() { _ = conn.CloseNow() }()
		ctx := r.Context()
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var request struct {
			Method string `json:"method"`
		}
		if json.Unmarshal(data, &request) != nil || request.Method != testSubscribeMethod {
			_ = conn.Write(ctx, websocket.MessageText, []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`))
			return
		}
		for _, message := range messages {
			if conn.Write(ctx, websocket.MessageText, []byte(message)) != nil {
				return
			}
		}
		_, _, _ = conn.Read(ctx)
	}))
	t.Cleanup(srv.Close)
	return "ws" + srv.URL[len("http"):]
}

func subscribeTest(t *testing.T, url string, method string, readTimeout time.Duration) (*WSSubscription, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	sub, err := SubscribeWS(ctx, WSConfig{Log: newTestLogger(t), ReadTimeout: readTimeout, URL: url}, method)
	if sub != nil {
		t.Cleanup(sub.Close)
	}
	return sub, err
}

func receiveNotification(t *testing.T, sub *WSSubscription) string {
	t.Helper()
	select {
	case result, ok := <-sub.Notifications():
		require.True(t, ok, "subscription ended: %v", sub.Err())
		return string(result)
	case <-time.After(5 * time.Second):
		t.Fatal("no notification within 5s")
		return ""
	}
}

func awaitSubscriptionEnd(t *testing.T, sub *WSSubscription) error {
	t.Helper()
	for {
		select {
		case _, ok := <-sub.Notifications():
			if !ok {
				return sub.Err()
			}
		case <-time.After(5 * time.Second):
			t.Fatal("subscription did not end within 5s")
			return nil
		}
	}
}

func TestSubscribeWSDeliversNotificationsInOrder(t *testing.T) {
	url := wsSubscriptionServer(t, testNotification(`{"number":1}`), testSubscribeReply, testNotification(`{"number":2}`), testNotification(`{"number":3}`))
	sub, err := subscribeTest(t, url, testSubscribeMethod, time.Minute)
	require.NoError(t, err)

	for _, want := range []string{`{"number":1}`, `{"number":2}`, `{"number":3}`} {
		require.Equal(t, want, receiveNotification(t, sub))
	}
	require.NoError(t, sub.Err())
}

func TestSubscribeWSFailsOnErrorReply(t *testing.T) {
	url := wsSubscriptionServer(t)
	_, err := subscribeTest(t, url, "reth_subscribeUnknown", time.Minute)
	require.ErrorContains(t, err, "server replied with error -32601: Method not found")
}

func TestWSSubscriptionEndsOnceQuietForTheReadTimeout(t *testing.T) {
	url := wsSubscriptionServer(t, testSubscribeReply)
	sub, err := subscribeTest(t, url, testSubscribeMethod, 100*time.Millisecond)
	require.NoError(t, err)

	require.ErrorContains(t, awaitSubscriptionEnd(t, sub), "no message within 100ms")
}

func TestWSSubscriptionEndsClosedOnClose(t *testing.T) {
	url := wsSubscriptionServer(t, testSubscribeReply)
	sub, err := subscribeTest(t, url, testSubscribeMethod, time.Minute)
	require.NoError(t, err)

	sub.Close()
	require.ErrorIs(t, awaitSubscriptionEnd(t, sub), ErrWSSubscriptionClosed)
}

func TestDecodeWSSubscriptionMessage(t *testing.T) {
	for _, tc := range []struct {
		name    string
		data    string
		want    wsSubscriptionMessage
		wantErr string
	}{
		{name: "subscription reply", data: testSubscribeReply, want: wsSubscriptionMessage{}},
		{name: "notification", data: testNotification(`{"number":42}`), want: wsSubscriptionMessage{isNotification: true, result: json.RawMessage(`{"number":42}`)}},
		{name: "error reply", data: `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`, wantErr: "error -32601: Method not found"},
		{name: "notification without result", data: `{"jsonrpc":"2.0","method":"m","params":{"subscription":"0x1f"}}`, wantErr: "unexpected message"},
		{name: "malformed", data: `{"jsonrpc":`, wantErr: "undecodable message"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeWSSubscriptionMessage([]byte(tc.data))
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
