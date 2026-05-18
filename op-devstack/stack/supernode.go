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
// VerificationStartTimestamp, FirstVerifiableTimestamp, FirstSealedBlock,
// LatestSealedBlock, ...).
type InteropTestControl interface {
	// InteropActivity returns the current interop activity, or nil if the
	// supernode is stopped or interop is not configured. Do not cache the
	// pointer across RestartWithFreshDataDir.
	InteropActivity() *interop.Interop

	// RestartWithFreshDataDir stops the supernode, deletes its on-disk
	// data directory, and starts a fresh supernode against the same chain
	// containers, virtual nodes, and externally-visible RPC address.
	RestartWithFreshDataDir() error
}
