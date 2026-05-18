package supernode

// Test-only Supernode methods. Production code paths must not call these.

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
