package monitor

import (
	"context"
	"testing"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// stubRPC implements the client.RPC interface for testing the read-only clients.
type stubRPC struct {
	calls  []string
	result interface{}
	err    error
}

func (s *stubRPC) CallContext(ctx context.Context, out interface{}, method string, args ...interface{}) error {
	s.calls = append(s.calls, method)
	if s.err != nil {
		return s.err
	}
	if b, ok := out.(*bool); ok {
		if v, ok := s.result.(bool); ok {
			*b = v
		}
	}
	return nil
}

func (s *stubRPC) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error { return nil }

func (s *stubRPC) Subscribe(ctx context.Context, namespace string, ch interface{}, args ...interface{}) (ethereum.Subscription, error) {
	return nil, nil
}

func (s *stubRPC) Close() {}

func TestFilterClientGetFailsafeEnabled(t *testing.T) {
	rpc := &stubRPC{result: true}
	fc := &FilterClient{rpc: rpc, filter: sources.NewInteropFilterClient(rpc), minSafety: safety.CrossUnsafe}
	enabled, err := fc.GetFailsafeEnabled(context.Background())
	require.NoError(t, err)
	require.True(t, enabled)
	require.Contains(t, rpc.calls, "admin_getFailsafeEnabled")
}

func TestFilterClientCheckMessage(t *testing.T) {
	rpc := &stubRPC{}
	fc := &FilterClient{rpc: rpc, filter: sources.NewInteropFilterClient(rpc), minSafety: safety.CrossUnsafe}
	msg := messages.Message{
		Identifier: messages.Identifier{ChainID: eth.ChainIDFromUInt64(1), BlockNumber: 10, LogIndex: 0, Timestamp: 1000, Origin: common.HexToAddress("0xabc")},
	}
	err := fc.CheckMessage(context.Background(), msg, eth.ChainIDFromUInt64(2), 1100)
	require.NoError(t, err)
	require.Contains(t, rpc.calls, "interop_checkAccessList")
}
