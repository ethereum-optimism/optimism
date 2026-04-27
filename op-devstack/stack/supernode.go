package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
)

type Supernode interface {
	Common
	QueryAPI() apis.SupernodeQueryAPI
}

// InteropTestControl is the narrow integration-test surface on a running
// supernode. Tests get direct access to the interop activity via
// InteropActivity; see op-supernode/supernode/activity/interop for the
// methods available on the returned pointer (PauseAt, Resume,
// BackfillAttempts, BackfillCompleted, ActivationTimestamp,
// BackfillEndTimestamp, FirstVerifiableTimestamp, FirstSealedBlock,
// LatestSealedBlock, ...).
type InteropTestControl interface {
	// InteropActivity returns the current interop activity, or nil if the
	// supernode is not running or interop is not configured. Callers must
	// not cache the pointer across RestartInteropActivity, which swaps the
	// activity for a fresh instance.
	InteropActivity() *interop.Interop

	// RestartInteropActivity stops the running interop activity, optionally
	// wipes its on-disk logs DBs, and launches a fresh instance against the
	// still-running supernode (HTTP server, chain containers, and all other
	// activities remain up).
	RestartInteropActivity(wipeLogsDBs bool) error
}
