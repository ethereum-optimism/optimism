package superroot

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockCC struct {
	valid    bool
	validErr error
	ref      eth.L2BlockRef
	refErr   error
}

func (m *mockCC) Start(ctx context.Context) error  { return nil }
func (m *mockCC) Stop(ctx context.Context) error   { return nil }
func (m *mockCC) Pause(ctx context.Context) error  { return nil }
func (m *mockCC) Resume(ctx context.Context) error { return nil }
func (m *mockCC) FullyValidAt(ctx context.Context, ts uint64) (bool, error) {
	return m.valid, m.validErr
}
func (m *mockCC) BlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	return m.ref, m.refErr
}

var _ cc.ChainContainer = (*mockCC)(nil)

func TestSuperroot_ErrIfAnyChainNotValid(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10):  &mockCC{valid: true},
		eth.ChainIDFromUInt64(420): &mockCC{valid: false},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	_, err := api.AtTimestamp(context.Background(), hexutil.Uint64(123))
	require.Error(t, err)
}

func TestSuperroot_ReturnsStringWithBlockRefs(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10):  &mockCC{valid: true, ref: eth.L2BlockRef{Number: 100, Time: 123}},
		eth.ChainIDFromUInt64(420): &mockCC{valid: true, ref: eth.L2BlockRef{Number: 200, Time: 123}},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	out, err := api.AtTimestamp(context.Background(), hexutil.Uint64(123))
	require.NoError(t, err)
	require.Contains(t, out, "Chain 10:")
	require.Contains(t, out, "Chain 420:")
}
