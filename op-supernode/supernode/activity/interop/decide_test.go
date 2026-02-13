package interop

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCheckPreconditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		obs  RoundObservation
		want *Decision
	}{
		{
			name: "pause when paused",
			obs: RoundObservation{
				Paused:       true,
				ChainsReady:  true,
				L1Consistent: true,
			},
			want: ptrDecision(DecisionWait),
		},
		{
			name: "wait when chains not ready",
			obs: RoundObservation{
				ChainsReady: false,
			},
			want: ptrDecision(DecisionWait),
		},
		{
			name: "rewind when L1 inconsistent",
			obs: RoundObservation{
				ChainsReady:  true,
				L1Consistent: false,
			},
			want: ptrDecision(DecisionRewind),
		},
		{
			name: "proceed when preconditions are satisfied",
			obs: RoundObservation{
				ChainsReady:  true,
				L1Consistent: true,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := checkPreconditions(tt.obs)
			if tt.want == nil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, *tt.want, got.Decision)
		})
	}
}

func TestDecideVerifiedResult(t *testing.T) {
	t.Parallel()

	ts := uint64(1000)
	validResult := Result{
		Timestamp:   ts + 1,
		L1Inclusion: eth.BlockID{Hash: common.HexToHash("0xl1"), Number: 50},
		L2Heads: map[eth.ChainID]eth.BlockID{
			eth.ChainIDFromUInt64(1): {Hash: common.HexToHash("0xa"), Number: 100},
		},
	}
	invalidResult := Result{
		Timestamp:   ts + 1,
		L1Inclusion: eth.BlockID{Hash: common.HexToHash("0xl1"), Number: 50},
		L2Heads: map[eth.ChainID]eth.BlockID{
			eth.ChainIDFromUInt64(1): {Hash: common.HexToHash("0xa"), Number: 100},
		},
		InvalidHeads: map[eth.ChainID]eth.BlockID{
			eth.ChainIDFromUInt64(2): {Hash: common.HexToHash("0xbad"), Number: 200},
		},
	}

	tests := []struct {
		name     string
		verified Result
		want     Decision
	}{
		{
			name:     "wait when verification result is empty",
			verified: Result{},
			want:     DecisionWait,
		},
		{
			name:     "invalidate on invalid verification",
			verified: invalidResult,
			want:     DecisionInvalidate,
		},
		{
			name:     "advance on valid verification",
			verified: validResult,
			want:     DecisionAdvance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := decideVerifiedResult(RoundObservation{}, tt.verified)
			require.Equal(t, tt.want, got.Decision, "unexpected decision")

			if tt.want == DecisionInvalidate {
				require.NotEmpty(t, got.Result.InvalidHeads, "invalidate should carry invalid heads")
			}
			if tt.want == DecisionAdvance {
				require.False(t, got.Result.IsEmpty(), "advance should carry result")
				require.True(t, got.Result.IsValid(), "advance result should be valid")
			}
		})
	}
}

func ptrDecision(d Decision) *Decision {
	return &d
}
