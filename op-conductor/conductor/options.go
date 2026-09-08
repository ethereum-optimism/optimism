package conductor

import (
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

// defaultPolicyInterval is how often a leader re-consults its LeadershipPolicy
// when no other event has triggered the control loop.
const defaultPolicyInterval = 30 * time.Second

// Option customizes an OpConductor at construction time.
type Option func(*OpConductor)

// WithTransferStrategy overrides how transfer targets are chosen when this
// conductor gives up leadership. It takes precedence over the strategy implied
// by the configuration flags.
func WithTransferStrategy(s TransferStrategy) Option {
	return func(oc *OpConductor) {
		oc.transferStrategy = s
	}
}

// WithLeadershipPolicy installs a policy that lets a healthy leader voluntarily
// give up leadership. The policy is consulted whenever the control loop runs
// while this conductor is a healthy active leader, and additionally on the given
// interval so that changes elsewhere in the cluster are eventually noticed.
// A non-positive interval falls back to the default.
//
// Without this option no policy goroutine is started and the conductor behaves
// exactly as it does today.
func WithLeadershipPolicy(p LeadershipPolicy, interval time.Duration) Option {
	return func(oc *OpConductor) {
		oc.leadershipPolicy = p
		if interval <= 0 {
			interval = defaultPolicyInterval
		}
		oc.policyInterval = interval
	}
}

// WithRPCAPIs registers additional namespaces on the conductor's RPC server,
// so embedders can expose their own endpoints on the same listener.
func WithRPCAPIs(apis ...rpc.API) Option {
	return func(oc *OpConductor) {
		oc.extraAPIs = append(oc.extraAPIs, apis...)
	}
}
