package stack

import "github.com/ethereum-optimism/optimism/op-service/apis"

type Supernode interface {
	Common
	QueryAPI() apis.SupernodeQueryAPI
}

// InteropTestControl provides integration test control methods for the interop activity.
// This interface is for integration test control only.
type InteropTestControl interface {
	// PauseInteropActivity pauses the interop activity at the given timestamp.
	// When the interop activity attempts to process this timestamp, it returns early.
	// This function is for integration test control only.
	PauseInteropActivity(ts uint64)

	// ResumeInteropActivity clears any pause on the interop activity, allowing normal processing.
	// This function is for integration test control only.
	ResumeInteropActivity()
}
