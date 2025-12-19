package superroot

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockCC struct {
	verL2     eth.BlockID
	verL1     eth.BlockID
	optL2     eth.BlockID
	optL1     eth.BlockID
	output    eth.Bytes32
	currentL1 eth.BlockRef

	currentL1Err  error
	verifiedErr   error
	outputErr     error
	optimisticErr error
}

func (m *mockCC) Start(ctx context.Context) error  { return nil }
func (m *mockCC) Stop(ctx context.Context) error   { return nil }
func (m *mockCC) Pause(ctx context.Context) error  { return nil }
func (m *mockCC) Resume(ctx context.Context) error { return nil }

func (m *mockCC) SafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, nil
}
func (m *mockCC) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *mockCC) L1AtSafeHead(ctx context.Context, l2 eth.BlockID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}
func (m *mockCC) CurrentL1(ctx context.Context) (eth.BlockRef, error) {
	if m.currentL1Err != nil {
		return eth.BlockRef{}, m.currentL1Err
	}
	return m.currentL1, nil
}
func (m *mockCC) VerifiedAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	if m.verifiedErr != nil {
		return eth.BlockID{}, eth.BlockID{}, m.verifiedErr
	}
	return m.verL2, m.verL1, nil
}
func (m *mockCC) OptimisticAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	if m.optimisticErr != nil {
		return eth.BlockID{}, eth.BlockID{}, m.optimisticErr
	}
	return m.optL2, m.optL1, nil
}
func (m *mockCC) OutputRootAtL2BlockNumber(ctx context.Context, l2BlockNum uint64) (eth.Bytes32, error) {
	if m.outputErr != nil {
		return eth.Bytes32{}, m.outputErr
	}
	return m.output, nil
}
func (m *mockCC) OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputResponse, error) {
	if m.optimisticErr != nil {
		return nil, m.optimisticErr
	}
	// Return minimal output response; tests only assert presence/count
	return &eth.OutputResponse{}, nil
}

var _ cc.ChainContainer = (*mockCC)(nil)

func TestSuperroot_AtTimestamp_Succeeds(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verL2:     eth.BlockID{Number: 100},
			verL1:     eth.BlockID{Number: 1000},
			optL2:     eth.BlockID{Number: 100},
			optL1:     eth.BlockID{Number: 1000},
			output:    eth.Bytes32{},
			currentL1: eth.BlockRef{Number: 2000},
		},
		eth.ChainIDFromUInt64(420): &mockCC{
			verL2:     eth.BlockID{Number: 200},
			verL1:     eth.BlockID{Number: 1100},
			optL2:     eth.BlockID{Number: 200},
			optL1:     eth.BlockID{Number: 1100},
			output:    eth.Bytes32{},
			currentL1: eth.BlockRef{Number: 2100},
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	out, err := api.AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Len(t, out.OptimisticAtTimestamp, 2)
	// min values
	require.Equal(t, uint64(2000), out.CurrentL1.Number)
	require.Equal(t, uint64(1000), out.Data.VerifiedRequiredL1.Number)
	// With zero outputs, the superroot will be deterministic, just ensure it's set
	_ = out.Data.SuperRoot
}

func TestSuperroot_AtTimestamp_ComputesSuperRoot(t *testing.T) {
	t.Parallel()
	out1 := eth.Bytes32{1}
	out2 := eth.Bytes32{2}
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verL2:     eth.BlockID{Number: 100},
			verL1:     eth.BlockID{Number: 1000},
			optL2:     eth.BlockID{Number: 100},
			optL1:     eth.BlockID{Number: 1000},
			output:    out1,
			currentL1: eth.BlockRef{Number: 2000},
		},
		eth.ChainIDFromUInt64(420): &mockCC{
			verL2:     eth.BlockID{Number: 200},
			verL1:     eth.BlockID{Number: 1100},
			optL2:     eth.BlockID{Number: 200},
			optL1:     eth.BlockID{Number: 1100},
			output:    out2,
			currentL1: eth.BlockRef{Number: 2100},
		},
	}
	ts := uint64(123)
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	resp, err := api.AtTimestamp(context.Background(), ts)
	require.NoError(t, err)

	// Compute expected super root
	chainOutputs := []eth.ChainIDAndOutput{
		{ChainID: eth.ChainIDFromUInt64(10), Output: out1},
		{ChainID: eth.ChainIDFromUInt64(420), Output: out2},
	}
	expected := eth.SuperRoot(eth.NewSuperV1(ts, chainOutputs...))
	require.Equal(t, expected, resp.Data.SuperRoot)
}

func TestSuperroot_AtTimestamp_ErrorOnCurrentL1(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			currentL1Err: assertErr(),
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	_, err := api.AtTimestamp(context.Background(), 123)
	require.Error(t, err)
}

func TestSuperroot_AtTimestamp_ErrorOnVerifiedAt(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verifiedErr: assertErr(),
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	_, err := api.AtTimestamp(context.Background(), 123)
	require.Error(t, err)
}

func TestSuperroot_AtTimestamp_NotFoundOnVerifiedAt(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verifiedErr: fmt.Errorf("nope: %w", ethereum.NotFound),
		},
		eth.ChainIDFromUInt64(11): &mockCC{
			verL2:     eth.BlockID{Number: 200},
			verL1:     eth.BlockID{Number: 1100},
			optL2:     eth.BlockID{Number: 200},
			optL1:     eth.BlockID{Number: 1100},
			output:    eth.Bytes32{0x12},
			currentL1: eth.BlockRef{Number: 2100},
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	actual, err := api.AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Nil(t, actual.Data)
	require.NotContains(t, actual.OptimisticAtTimestamp, eth.ChainIDFromUInt64(10))
	require.Contains(t, actual.OptimisticAtTimestamp, eth.ChainIDFromUInt64(11))
}

func TestSuperroot_AtTimestamp_ErrorOnOutputRoot(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verL2:     eth.BlockID{Number: 100},
			outputErr: assertErr(),
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	_, err := api.AtTimestamp(context.Background(), 123)
	require.Error(t, err)
}

func TestSuperroot_AtTimestamp_ErrorOnOptimisticAt(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verL2:         eth.BlockID{Number: 100},
			output:        eth.Bytes32{1},
			optimisticErr: assertErr(),
		},
	}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	_, err := api.AtTimestamp(context.Background(), 123)
	require.Error(t, err)
}

func TestSuperroot_AtTimestamp_EmptyChains(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{}
	s := New(gethlog.New(), chains)
	api := &superrootAPI{s: s}
	out, err := api.AtTimestamp(context.Background(), 123)
	require.NoError(t, err)
	require.Len(t, out.OptimisticAtTimestamp, 0)
}

// assertErr returns a generic error instance used to signal mock failures.
func assertErr() error { return fmt.Errorf("mock error") }
