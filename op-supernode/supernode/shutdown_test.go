package supernode

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// newTestSupernode builds a minimal Supernode wired with a real HTTP server,
// a real metrics server, and the given activities. Both servers bind to
// 127.0.0.1:0 so there are no port conflicts.
func newTestSupernode(t *testing.T, acts []activity.Activity) *Supernode {
	t.Helper()
	log := gethlog.New()

	router := resources.NewRouter(log, resources.RouterConfig{})
	httpSrv := httputil.NewHTTPServer("127.0.0.1:0", router)
	metrics := resources.NewMetricsService(log, "127.0.0.1", 0, http.NewServeMux())

	return &Supernode{
		log:        log,
		version:    "test",
		chains:     nil,
		activities: acts,
		httpServer: httpSrv,
		rpcRouter:  router,
		metrics:    metrics,
	}
}

// TestCleanShutdown starts a supernode with multiple services running — a real HTTP
// server, a real metrics server, a mock activity.
// It then calls Stop() and asserts it returns within a reasonable deadline.
func TestCleanShutdown(t *testing.T) {
	t.Parallel()

	const (
		// How long the slow activity's goroutine sleeps after context cancellation.
		activityLinger = 200 * time.Millisecond

		// How long Stop() is allowed to take in total.
		// Generous enough for a real graceful shutdown, tight enough to catch a hang.
		stopDeadline = 2 * time.Second
	)

	s := newTestSupernode(t, []activity.Activity{&mockRunnable{}})

	// Cancel the start context after a brief window so Start() exits promptly.
	startCtx, cancelStart := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStart()
	require.NoError(t, s.Start(startCtx))

	// Run Stop() in a goroutine so we can race it against the deadline.
	stopCtx, cancelStop := context.WithTimeout(context.Background(), stopDeadline)
	defer cancelStop()

	stopDone := make(chan error, 1)
	go func() { stopDone <- s.Stop(context.Background()) }()

	select {
	case err := <-stopDone:
		require.NoError(t, err)
	case <-stopCtx.Done():
		t.Fatalf("Stop() did not return within %s — supernode hung on shutdown ", stopDeadline)
	}
}
