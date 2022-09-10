package heartbeat

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

const expHeartbeat = `{
	"version": "v1.2.3",
	"meta": "meta",
	"moniker": "yeet",
	"peerID": "1UiUfoobar",
	"chainID": 1234
}`

func TestBeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	reqCh := make(chan string, 2)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		reqCh <- string(body)
		r.Body.Close()
	}))
	defer s.Close()

	doneCh := make(chan struct{})
	go func() {
		_ = Beat(ctx, log.Root(), s.URL, &Payload{
			Version: "v1.2.3",
			Meta:    "meta",
			Moniker: "yeet",
			PeerID:  "1UiUfoobar",
			ChainID: 1234,
		})
		doneCh <- struct{}{}
	}()

	select {
	case hb := <-reqCh:
		require.JSONEq(t, expHeartbeat, hb)
		cancel()
		<-doneCh
	case <-ctx.Done():
		t.Fatalf("error: %v", ctx.Err())
	}
}
