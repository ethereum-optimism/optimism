package monitor

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// FilterChecker is the read-only interop-filter surface the observer needs.
type FilterChecker interface {
	// CheckMessage replays a single executing message as an access list to the filter.
	// A nil error means the filter considers the message valid at minSafety.
	CheckMessage(ctx context.Context, msg messages.Message, executingChain eth.ChainID, executingTimestamp uint64) error
	GetFailsafeEnabled(ctx context.Context) (bool, error)
	Close()
}

// FilterClient calls the op-interop-filter public RPC (read-only). It delegates the
// access-list check to the canonical sources.InteropFilterClient so the monitor stays
// in sync with the filter's RPC signature.
type FilterClient struct {
	rpc       client.RPC
	filter    *sources.InteropFilterClient
	minSafety safety.Level
	log       log.Logger
}

var _ FilterChecker = (*FilterClient)(nil)

func NewFilterClient(endpoint string, minSafety safety.Level, log log.Logger) (*FilterClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("interop-filter endpoint not configured")
	}
	c, err := client.NewRPC(context.Background(), log, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create interop-filter client: %w", err)
	}
	return &FilterClient{rpc: c, filter: sources.NewInteropFilterClient(c), minSafety: minSafety, log: log}, nil
}

// CheckMessage builds the access-list for one executing message and delegates to the
// canonical interop-filter client's CheckAccessList. A nil error means the filter
// considers the message valid at minSafety; a non-nil error is the filter's rejection
// (or a transport error).
func (fc *FilterClient) CheckMessage(ctx context.Context, msg messages.Message, executingChain eth.ChainID, executingTimestamp uint64) error {
	access := msg.Access()
	entries := messages.EncodeAccessList([]messages.Access{access})
	execDesc := messages.ExecutingDescriptor{ChainID: executingChain, Timestamp: executingTimestamp}
	return fc.filter.CheckAccessList(ctx, entries, fc.minSafety, execDesc)
}

// GetFailsafeEnabled reads the filter's failsafe state via admin_getFailsafeEnabled.
// This is an admin-namespace method, distinct from the interop_* query API the
// canonical filter client wraps, so it is called directly here.
func (fc *FilterClient) GetFailsafeEnabled(ctx context.Context) (bool, error) {
	var enabled bool
	err := fc.rpc.CallContext(ctx, &enabled, "admin_getFailsafeEnabled")
	return enabled, err
}

func (fc *FilterClient) Close() { fc.rpc.Close() }
