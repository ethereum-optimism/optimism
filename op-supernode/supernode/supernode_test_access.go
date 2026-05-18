package supernode

// This file collects Supernode methods that expose test-only access to the
// interop activity. They must not be called by production code paths. Keeping
// them in one file makes the test-only surface easy to audit alongside
// interop/interop_test_access.go.

import (
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
)

// InteropActivity returns the single interop activity registered with the
// supernode, or nil if interop is not configured or has not started yet.
// The pointer is bound to the current Supernode instance; integration tests
// that tear the supernode down and back up (e.g. via the test harness's
// RestartWithFreshDataDir primitive) must re-fetch the activity after the
// restart.
func (s *Supernode) InteropActivity() *interop.Interop {
	s.activitiesMu.RLock()
	defer s.activitiesMu.RUnlock()
	for _, a := range s.activities {
		if ia, ok := a.(*interop.Interop); ok {
			return ia
		}
	}
	return nil
}
