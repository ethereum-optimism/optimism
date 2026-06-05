package monitor

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// FilterChecker is the read-only interop-filter surface the observer needs.
type FilterChecker interface {
	// CheckMessage replays a single executing message as an access list to the filter.
	// A nil error means the filter considers the message valid at minSafety.
	CheckMessage(ctx context.Context, msg messages.Message, executingChain eth.ChainID, executingTimestamp uint64) error
	GetFailsafeEnabled(ctx context.Context) (bool, error)
	Close()
}

// FilterClient calls the op-interop-filter public RPC (read-only).
type FilterClient struct {
	client    client.RPC
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
	return &FilterClient{client: c, minSafety: minSafety, log: log}, nil
}

// CheckMessage builds the access-list for one executing message and calls interop_checkAccessList.
// A nil error means the filter considers the message valid at minSafety; a non-nil error is the
// filter's rejection (or a transport error).
func (fc *FilterClient) CheckMessage(ctx context.Context, msg messages.Message, executingChain eth.ChainID, executingTimestamp uint64) error {
	access := msg.Access()
	entries := messages.EncodeAccessList([]messages.Access{access})
	execDesc := messages.ExecutingDescriptor{ChainID: executingChain, Timestamp: executingTimestamp}
	return fc.client.CallContext(ctx, nil, "interop_checkAccessList", entries, fc.minSafety, execDesc)
}

func (fc *FilterClient) GetFailsafeEnabled(ctx context.Context) (bool, error) {
	var enabled bool
	err := fc.client.CallContext(ctx, &enabled, "admin_getFailsafeEnabled")
	return enabled, err
}

func (fc *FilterClient) Close() { fc.client.Close() }
