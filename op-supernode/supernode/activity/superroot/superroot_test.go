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
	out      eth.Bytes32
	outErr   error
}

func (m *mockCC) Start(ctx context.Context) error  { return nil }
func (m *mockCC) Stop(ctx context.Context) error   { return nil }
func (m *mockCC) Pause(ctx context.Context) error  { return nil }
func (m *mockCC) Resume(ctx context.Context) error { return nil }
func (m *mockCC) SafeAtTimestamp(ctx context.Context, ts uint64) (bool, error) {
	return m.valid, m.validErr
}
func (m *mockCC) VerifiedAtTimestamp(ctx context.Context, ts uint64) (bool, error) {
	return m.valid, m.validErr
}
func (m *mockCC) BlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	return m.ref, m.refErr
}
func (m *mockCC) OutputRootAtTimestamp(ctx context.Context, ts uint64) (eth.Bytes32, error) {
	return m.out, m.outErr
}
func (m *mockCC) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
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

func TestSuperroot_ComputesSuperRootResponse(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10):  &mockCC{valid: true, ref: eth.L2BlockRef{Number: 100, Time: 123}, out: eth.Bytes32{}},
		eth.ChainIDFromUInt64(420): &mockCC{valid: true, ref: eth.L2BlockRef{Number: 200, Time: 123}, out: eth.Bytes32{}},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	out, err := api.AtTimestamp(context.Background(), hexutil.Uint64(123))
	require.NoError(t, err)
	require.Equal(t, uint64(123), out.Timestamp)
	require.Equal(t, eth.SuperRootVersionV1, out.Version)
	require.Len(t, out.Chains, 2)
}

func TestSuperroot_DeterministicOrderingAndHash(t *testing.T) {
	t.Parallel()
	// Provide outputs for 3 chains intentionally inserted in non-sorted order
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(420):  &mockCC{valid: true, out: eth.Bytes32{0x02}},
		eth.ChainIDFromUInt64(10):   &mockCC{valid: true, out: eth.Bytes32{0x01}},
		eth.ChainIDFromUInt64(8453): &mockCC{valid: true, out: eth.Bytes32{0x03}},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}

	out1, err := api.AtTimestamp(context.Background(), hexutil.Uint64(123))
	require.NoError(t, err)

	// Call again to ensure stability
	out2, err := api.AtTimestamp(context.Background(), hexutil.Uint64(123))
	require.NoError(t, err)

	// Chains should be sorted by ChainID
	require.Equal(t, []eth.ChainID{
		eth.ChainIDFromUInt64(10), eth.ChainIDFromUInt64(420), eth.ChainIDFromUInt64(8453),
	}, []eth.ChainID{out1.Chains[0].ChainID, out1.Chains[1].ChainID, out1.Chains[2].ChainID})

	// Hashes should be deterministic across calls
	require.Equal(t, out1.SuperRoot, out2.SuperRoot)

	// And equal to the manually constructed super-v1 hash (sorting inside constructor)
	expected := eth.SuperRoot(eth.NewSuperV1(123,
		eth.ChainIDAndOutput{ChainID: eth.ChainIDFromUInt64(10), Output: eth.Bytes32{0x01}},
		eth.ChainIDAndOutput{ChainID: eth.ChainIDFromUInt64(420), Output: eth.Bytes32{0x02}},
		eth.ChainIDAndOutput{ChainID: eth.ChainIDFromUInt64(8453), Output: eth.Bytes32{0x03}},
	))
	require.Equal(t, expected, out1.SuperRoot)
}
