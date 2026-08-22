package apis

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// SupernodeQueryAPI is the minimal API surface the devstack DSL needs from op-supernode.
// It is intentionally small and can be expanded as needed.
type SupernodeQueryAPI interface {
	SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error)
	SyncStatus(ctx context.Context) (eth.SuperNodeSyncStatusResponse, error)
}

// SupernodeInteropTestAPI is the test-control surface on a supernode's interop
// verification activity: the handful of things acceptance tests need to do to a
// running verifier in order to observe it deterministically.
//
// It is deliberately shaped like an RPC service rather than like the Go object
// behind it. Every method takes a context and returns an error, and every value
// crossing it is plain serializable data, so a supernode running in another
// process can implement this by serving four methods while the in-process Go
// supernode implements it by calling itself. Nothing here hands out a pointer to
// a live object, which is what previously made this surface impossible for an
// out-of-process supernode to satisfy.
//
// Implementations are test-only. Production code paths must not use them.
type SupernodeInteropTestAPI interface {
	// PauseInterop asks the verifier to stop when it reaches timestamp ts, so a
	// test can hold verification still while it mutates the chains underneath.
	// The check is inclusive and forward-looking: a verifier already past ts
	// still stops. Passing 0 clears the pause, as ResumeInterop does.
	PauseInterop(ctx context.Context, ts uint64) error

	// ResumeInterop clears any pause, letting verification continue.
	ResumeInterop(ctx context.Context) error

	// InteropStatus reads the verifier's test-visible progress.
	InteropStatus(ctx context.Context) (eth.SupernodeInteropStatus, error)

	// InteropSealedBlocks reports how far one chain's interop logs DB extends.
	// It returns an error only if the chain is unknown to the supernode; an
	// empty logs DB is reported through the result.
	InteropSealedBlocks(ctx context.Context, chainID eth.ChainID) (eth.SupernodeSealedBlocks, error)
}
