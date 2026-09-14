package conductor

import (
	"github.com/ethereum/go-ethereum/rpc"
)

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

// WithRPCAPIs registers additional namespaces on the conductor's RPC server,
// so embedders can expose their own endpoints on the same listener.
func WithRPCAPIs(apis ...rpc.API) Option {
	return func(oc *OpConductor) {
		oc.extraAPIs = append(oc.extraAPIs, apis...)
	}
}
