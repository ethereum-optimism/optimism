package chain_container

import (
	"fmt"

	gethrpc "github.com/ethereum/go-ethereum/rpc"

	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

// ExtraRPCRoute is one additional JSON-RPC sub-route mounted on a chain's OWN RPC handler, so it is
// served at `<base>/<chainID>/<route>` — a sibling of the virtual node's own `<base>/<chainID>`.
//
// It exists because the shared router keys on the FIRST path segment only
// (resources.Router.ServeHTTP), so anything under a chain's prefix has to be mounted on that
// chain's handler rather than registered with the router. That is the monorepo's established
// sub-route idiom — op-faucet, op-sync-tester and op-test-sequencer all build their per-thing
// routes with oprpc.Handler.AddRPC + AddAPIToRPC — and it is what puts a module's endpoint at a
// DISTINCT route instead of adding a namespace to the node's own.
//
// A route inherits the chain's readiness gate, which is correct rather than incidental: a module
// mounted here reads the chain's own derived data, so while that chain's virtual node is down or
// paused the module has nothing new to say either, and the router's hold-then-503 is exactly what a
// consumer of a JSON-RPC route should see.
type ExtraRPCRoute struct {
	// Route is the sub-path, leading slash and no trailing slash, e.g. "/claimed".
	Route string
	// API is the namespace and service mounted on it.
	API gethrpc.API
}

// ChainContainerOption customises a chain container at construction.
//
// It is variadic on purpose: a container built with no options is byte-identical to one built
// before options existed, which is what lets an optional module be genuinely dormant.
type ChainContainerOption func(*simpleChainContainer)

// WithExtraRPCRoutes mounts additional JSON-RPC sub-routes on this chain's handler.
//
// The handler is rebuilt on every virtual-node (re)start, so the routes are re-registered each
// time; the service behind them must therefore outlive the handler, and must be safe for
// concurrent use. Nothing is registered when no route is supplied.
func WithExtraRPCRoutes(routes ...ExtraRPCRoute) ChainContainerOption {
	return func(c *simpleChainContainer) {
		c.extraRPCRoutes = append(c.extraRPCRoutes, routes...)
	}
}

// registerExtraRPCRoutes mounts the configured routes on a freshly-built per-chain handler.
//
// Registration failures are fatal. Continuing would leave the chain's root RPC healthy while a
// specifically configured safety route is absent, which is indistinguishable to its consumer from
// forgetting to enable the module.
func (c *simpleChainContainer) registerExtraRPCRoutes(h *oprpc.Handler) error {
	for _, r := range c.extraRPCRoutes {
		if err := h.AddRPC(r.Route); err != nil {
			return fmt.Errorf("create extra RPC route %q for chain %s: %w", r.Route, c.chainID, err)
		}
		if err := h.AddAPIToRPC(r.Route, r.API); err != nil {
			return fmt.Errorf("register namespace %q on extra RPC route %q for chain %s: %w",
				r.API.Namespace, r.Route, c.chainID, err)
		}
	}
	return nil
}
