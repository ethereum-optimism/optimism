package interopsmoke

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestValidateInvalidMessageOptions(t *testing.T) {
	for _, tc := range []struct {
		name               string
		blocks, txPerBlock uint
		wantErr            bool
	}{
		{"valid", 2, 3, false},
		{"zero blocks", 0, 1, true},
		{"zero tx per block", 1, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateInvalidMessageOptions(tc.blocks, tc.txPerBlock); (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestInvalidDirections(t *testing.T) {
	env := &smokeEnv{
		userA: &remoteUser{chain: &remoteChain{name: "L2A"}},
		userB: &remoteUser{chain: &remoteChain{name: "L2B"}},
	}
	for _, tc := range []struct {
		direction string
		wantNames []string
		wantErr   bool
	}{
		{direction: directionBoth, wantNames: []string{"A->B", "B->A"}},
		{direction: "", wantNames: []string{"A->B", "B->A"}},
		{direction: directionAToB, wantNames: []string{"A->B"}},
		{direction: directionBToA, wantNames: []string{"B->A"}},
		{direction: "sideways", wantErr: true},
	} {
		t.Run(tc.direction, func(t *testing.T) {
			env.direction = tc.direction
			dirs, err := invalidDirections(env)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if len(dirs) != len(tc.wantNames) {
				t.Fatalf("got %d directions, want %d", len(dirs), len(tc.wantNames))
			}
			for i, want := range tc.wantNames {
				if dirs[i].name != want {
					t.Fatalf("direction %d = %s, want %s", i, dirs[i].name, want)
				}
			}
		})
	}
}

func TestFirstLogFrom(t *testing.T) {
	messenger := common.HexToAddress("0x4200000000000000000000000000000000000023")
	eventLogger := common.HexToAddress("0x1111111111111111111111111111111111111111")
	logs := []*types.Log{
		{Address: messenger},
		{Address: eventLogger},
		{Address: eventLogger},
	}

	if got := firstLogFrom(logs, eventLogger); got != 1 {
		t.Fatalf("first log from EventLogger = %d, want 1", got)
	}
	if got := firstLogFrom(logs, messenger); got != 0 {
		t.Fatalf("first log from messenger = %d, want 0", got)
	}
	if got := firstLogFrom(logs, common.Address{}); got != -1 {
		t.Fatalf("absent origin = %d, want -1", got)
	}
	if got := firstLogFrom(nil, eventLogger); got != -1 {
		t.Fatalf("no logs = %d, want -1", got)
	}
}

// stubEthClient answers BlockRefByLabel from refs; every other method nil-panics.
type stubEthClient struct {
	apis.EthClient
	refs func() (eth.BlockRef, error)
}

func (s *stubEthClient) BlockRefByLabel(context.Context, eth.BlockLabel) (eth.BlockRef, error) {
	return s.refs()
}

func TestWaitForHead(t *testing.T) {
	atTime := func(minTime uint64) func(eth.BlockRef) bool {
		return func(head eth.BlockRef) bool { return head.Time >= minTime }
	}

	t.Run("retries a failed head lookup", func(t *testing.T) {
		calls := 0
		chain := &remoteChain{name: "chainA", ethClient: &stubEthClient{refs: func() (eth.BlockRef, error) {
			calls++
			if calls == 1 {
				return eth.BlockRef{}, errors.New("transient rpc failure")
			}
			return eth.BlockRef{Number: 2, Time: 20}, nil
		}}}

		require.NoError(t, waitForHead(context.Background(), chain, "timestamp >= 20", atTime(20)))
		require.Equal(t, 2, calls, "should have retried past the failure")
	})

	t.Run("returns the context error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		chain := &remoteChain{name: "chainA", ethClient: &stubEthClient{refs: func() (eth.BlockRef, error) {
			cancel()
			return eth.BlockRef{}, errors.New("transient rpc failure")
		}}}

		require.ErrorIs(t, waitForHead(ctx, chain, "timestamp >= 20", atTime(20)), context.Canceled)
	})
}
