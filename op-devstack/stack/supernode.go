package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
)

type Supernode interface {
	Common
	QueryAPI() apis.SupernodeQueryAPI
}

// SupernodeTestControl is the narrow integration-test surface on a running
// supernode. Tests get direct access to the interop activity via
// InteropActivity; see op-supernode/supernode/activity/interop for the
// methods available on the returned pointer (PauseAt, Resume,
// BackfillAttempts, BackfillCompleted, ActivationTimestamp,
// VerificationStartTimestamp, FirstVerifiableTimestamp, FirstSealedBlock,
// LatestSealedBlock, ...).
type SupernodeTestControl interface {
	// InteropActivity returns the current interop activity, or nil if the
	// supernode is not running or interop is not configured. Callers must
	// not cache the pointer across RestartWithFreshDataDir, which tears the
	// supernode down and brings up a fresh instance.
	InteropActivity() *interop.Interop

	// RestartWithFreshDataDir stops the supernode, deletes its on-disk
	// data directory in full, and starts a fresh supernode against the same
	// chain containers, virtual nodes, and externally-visible RPC address.
	// Used by tests that need to exercise the cold-start path with no
	// prior verifiedDB / logsDB / safe_db state.
	RestartWithFreshDataDir() error
}
