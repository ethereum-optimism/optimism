package stack

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
)

type Supernode interface {
	Common
	ClientRPC() client.RPC
	QueryAPI() apis.SupernodeQueryAPI
	UserRPC() string
}

// SupernodeTestControl is the integration-test surface on a running interop verifier, which is
// op-supernode in process or lokahi in a process of its own.
type SupernodeTestControl interface {
	// PauseInteropAt makes the verifier stop before it verifies timestamp ts, until
	// ResumeInterop. It fails when the verifier is stopped or interop is not configured.
	PauseInteropAt(ctx context.Context, ts uint64) error

	// ResumeInterop clears a pause, so the verifier continues.
	ResumeInterop(ctx context.Context) error

	// RestartWithFreshDataDir stops the verifier, deletes its on-disk
	// data directory, and starts it again with the same chains and the same
	// externally-visible RPC address.
	RestartWithFreshDataDir() error

	// Stop halts the verifier while preserving its data directory and RPC
	// address; Start brings it back up. Used by sync tests that need to halt
	// the verifier, mutate external state, and resume.
	Stop()
	Start()
}

// SupernodeInternals reaches into an in-process op-supernode. Only op-supernode has it: a test
// that needs it tests op-supernode's own mechanics.
type SupernodeInternals interface {
	// InteropActivity returns the current interop activity, or nil if the
	// supernode is stopped or interop is not configured. Do not cache the
	// pointer across RestartWithFreshDataDir.
	InteropActivity() *interop.Interop
}
