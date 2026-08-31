package claimfollow

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// API is the served surface, and it is one method wide.
//
// The follow protocol is `optimism_syncStatus` and nothing else — verified against both op-node's
// follow source and kona's — so this type deliberately does not grow the rest of the optimism
// namespace. A route that answered outputAtBlock or safeHeadAtL1Block would be impersonating a node
// with state it does not have; the private chain's own LightCL, sitting in front of this, serves
// the real optimism namespace to every other consumer from the claim-driven safe head this feeds
// it.
//
// It is mounted at the chain's SIBLING route (`<base>/<chainID>/claimed`) and never on the chain's
// own route. The distinction is load-bearing: `<base>/<chainID>` is the RENDERING chain's honest
// public view and must stay that, and a consumer pointed at the wrong one of the two gets an
// obviously different answer instead of plausible-looking refs of the wrong chain — which a
// sequencing LightCL would force-reset onto.
type API struct {
	m *Module
}

// NewAPI wraps a module as the "optimism" namespace service.
func NewAPI(m *Module) *API { return &API{m: m} }

// SyncStatus serves optimism_syncStatus. See Module.SyncStatus for the field population, and the
// package comment for why an error before the first claim is the right answer rather than a gap.
func (a *API) SyncStatus(_ context.Context) (*eth.SyncStatus, error) { return a.m.SyncStatus() }
