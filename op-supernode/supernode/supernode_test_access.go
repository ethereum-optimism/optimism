package supernode

// Test-only Supernode methods. Production code paths must not call these.

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
)

// InteropTestAPI returns the test-control surface for this supernode's interop
// verification activity, or nil if interop is not configured or has not started
// yet.
//
// The returned value is bound to the current Supernode instance, so tests that
// tear the supernode down must re-fetch it afterwards rather than hold on to
// one. That is also why this returns the interface rather than the activity
// itself: an out-of-process supernode answers the same interface over RPC, and
// nothing on the caller's side needs to know which it got.
func (s *Supernode) InteropTestAPI() apis.SupernodeInteropTestAPI {
	ia := s.interopActivity()
	if ia == nil {
		return nil
	}
	return interop.NewTestAPI(ia)
}

// interopActivity returns the registered interop activity, or nil if interop is
// not configured or has not started yet.
func (s *Supernode) interopActivity() *interop.Interop {
	s.activitiesMu.RLock()
	defer s.activitiesMu.RUnlock()
	for _, a := range s.activities {
		if ia, ok := a.(*interop.Interop); ok {
			return ia
		}
	}
	return nil
}
