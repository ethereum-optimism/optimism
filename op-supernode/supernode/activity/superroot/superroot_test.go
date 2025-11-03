package superroot

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockCC struct {
	valid    bool
	validErr error
	ref      eth.L2BlockRef
	refErr   error
	out      eth.Bytes32
	outErr   error
}

func (m *mockCC) Start(ctx context.Context) error  { return nil }
func (m *mockCC) Stop(ctx context.Context) error   { return nil }
func (m *mockCC) Pause(ctx context.Context) error  { return nil }
func (m *mockCC) Resume(ctx context.Context) error { return nil }
func (m *mockCC) VerifiedAtTimestamp(ctx context.Context, ts uint64) (bool, error) {
	return m.valid, m.validErr
}
func (m *mockCC) SafeAtTimestamp(ctx context.Context, ts uint64) (bool, error) {
	return m.valid, m.validErr
}
func (m *mockCC) BlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	return m.ref, m.refErr
}
func (m *mockCC) OutputRootAtL1(ctx context.Context, l1BlockNum uint64) (eth.Bytes32, error) {
	return m.out, m.outErr
}

func (m *mockCC) CurrentL1(ctx context.Context) (eth.BlockRef, error) {
	return eth.BlockRef{}, nil
}
func (m *mockCC) LastL1(ctx context.Context) (eth.BlockRef, error) { return eth.BlockRef{}, nil }
func (m *mockCC) VerifiedToTimestamp() (uint64, error)             { return 0, nil }
func (m *mockCC) VerifiedToTimestampWithL1(ctx context.Context, l1BlockNum uint64) (uint64, error) {
	return 0, nil
}
func (m *mockCC) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}

var _ cc.ChainContainer = (*mockCC)(nil)

func TestSuperroot_AtL1_Succeeds(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10):  &mockCC{valid: true},
		eth.ChainIDFromUInt64(420): &mockCC{valid: true},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	out, err := api.AtL1(context.Background(), 123)
	require.NoError(t, err)
	require.Len(t, out.CurrentL1s, 2)
}

func TestSuperroot_AtL1_PopulatesFields(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10):  &mockCC{valid: true, ref: eth.L2BlockRef{Number: 100, Time: 123}, out: eth.Bytes32{}},
		eth.ChainIDFromUInt64(420): &mockCC{valid: true, ref: eth.L2BlockRef{Number: 200, Time: 123}, out: eth.Bytes32{}},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	out, err := api.AtL1(context.Background(), 123)
	require.NoError(t, err)
	require.Len(t, out.CurrentL1s, 2)
}
