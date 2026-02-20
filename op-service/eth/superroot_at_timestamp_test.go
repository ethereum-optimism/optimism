package eth

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

/*
Specification: SuperRootAtTimestampResponse Flattened Structure

This file tests the flattened SuperRootAtTimestampResponse structure where:
- Super, SuperRoot, and VerifiedRequiredL1 are top-level fields (moved from nested Data)
- Zero values indicate "not available":
  - VerifiedRequiredL1 == BlockID{} means verification incomplete (optimistic data)
  - SuperRoot == Bytes32{} means not all output roots were available
- The old Data field is removed
- JSON marshalling preserves backwards compatibility for the Super interface type
*/

func TestSuperRootAtTimestampResponse_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	chainA := ChainIDFromUInt64(1)
	chainB := ChainIDFromUInt64(2)
	timestamp := uint64(1000)

	t.Run("FullyPopulatedResponse", func(t *testing.T) {
		superV1 := NewSuperV1(timestamp,
			ChainIDAndOutput{ChainID: chainA, Output: Bytes32{0xa1}},
			ChainIDAndOutput{ChainID: chainB, Output: Bytes32{0xa2}},
		)

		original := SuperRootAtTimestampResponse{
			CurrentL1:                 BlockID{Hash: common.Hash{0x11}, Number: 100},
			CurrentSafeTimestamp:      500,
			CurrentFinalizedTimestamp: 400,
			ChainIDs:                  []ChainID{chainA, chainB},
			OptimisticAtTimestamp: map[ChainID]OutputWithRequiredL1{
				chainA: {
					Output:     &OutputResponse{OutputRoot: Bytes32{0xaa}},
					RequiredL1: BlockID{Hash: common.Hash{0x22}, Number: 90},
				},
			},
			Super:              superV1,
			SuperRoot:          SuperRoot(superV1),
			VerifiedRequiredL1: BlockID{Hash: common.Hash{0x33}, Number: 95},
		}

		// Marshal to JSON
		data, err := json.Marshal(original)
		require.NoError(t, err)

		// Unmarshal back
		var decoded SuperRootAtTimestampResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		// Verify all fields
		require.Equal(t, original.CurrentL1, decoded.CurrentL1)
		require.Equal(t, original.CurrentSafeTimestamp, decoded.CurrentSafeTimestamp)
		require.Equal(t, original.CurrentFinalizedTimestamp, decoded.CurrentFinalizedTimestamp)
		require.Equal(t, original.ChainIDs, decoded.ChainIDs)
		require.Equal(t, original.SuperRoot, decoded.SuperRoot)
		require.Equal(t, original.VerifiedRequiredL1, decoded.VerifiedRequiredL1)

		// Super is an interface, check the concrete type
		decodedSuper, ok := decoded.Super.(*SuperV1)
		require.True(t, ok, "Super should be *SuperV1")
		require.Equal(t, timestamp, decodedSuper.Timestamp)
		require.Len(t, decodedSuper.Chains, 2)
	})

	t.Run("ZeroVerifiedRequiredL1", func(t *testing.T) {
		// When verification is incomplete, VerifiedRequiredL1 is zero
		superV1 := NewSuperV1(timestamp,
			ChainIDAndOutput{ChainID: chainA, Output: Bytes32{0xa1}},
		)

		original := SuperRootAtTimestampResponse{
			CurrentL1:          BlockID{Hash: common.Hash{0x11}, Number: 100},
			ChainIDs:           []ChainID{chainA},
			Super:              superV1,
			SuperRoot:          SuperRoot(superV1),
			VerifiedRequiredL1: BlockID{}, // Zero - indicates optimistic data
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded SuperRootAtTimestampResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		require.Equal(t, BlockID{}, decoded.VerifiedRequiredL1)
		require.NotEqual(t, Bytes32{}, decoded.SuperRoot, "SuperRoot should still be populated")
	})

	t.Run("ZeroSuperRoot", func(t *testing.T) {
		// When output roots are not available, SuperRoot is zero
		original := SuperRootAtTimestampResponse{
			CurrentL1:          BlockID{Hash: common.Hash{0x11}, Number: 100},
			ChainIDs:           []ChainID{chainA},
			Super:              nil,
			SuperRoot:          Bytes32{}, // Zero - indicates no super root available
			VerifiedRequiredL1: BlockID{}, // Also zero since we can't verify
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded SuperRootAtTimestampResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		require.Equal(t, Bytes32{}, decoded.SuperRoot)
		require.Nil(t, decoded.Super)
	})

	t.Run("EmptyOptimisticMap", func(t *testing.T) {
		original := SuperRootAtTimestampResponse{
			CurrentL1:             BlockID{Hash: common.Hash{0x11}, Number: 100},
			ChainIDs:              []ChainID{},
			OptimisticAtTimestamp: map[ChainID]OutputWithRequiredL1{},
		}

		data, err := json.Marshal(original)
		require.NoError(t, err)

		var decoded SuperRootAtTimestampResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		require.Empty(t, decoded.OptimisticAtTimestamp)
	})
}

func TestSuperRootAtTimestampResponse_ZeroValueSemantics(t *testing.T) {
	t.Parallel()

	t.Run("IsVerified", func(t *testing.T) {
		// Helper to check if response indicates verified data
		isVerified := func(r SuperRootAtTimestampResponse) bool {
			return r.VerifiedRequiredL1 != BlockID{}
		}

		verified := SuperRootAtTimestampResponse{
			VerifiedRequiredL1: BlockID{Hash: common.Hash{0x11}, Number: 100},
		}
		require.True(t, isVerified(verified))

		unverified := SuperRootAtTimestampResponse{
			VerifiedRequiredL1: BlockID{},
		}
		require.False(t, isVerified(unverified))
	})

	t.Run("HasSuperRoot", func(t *testing.T) {
		// Helper to check if response has a computed super root
		hasSuperRoot := func(r SuperRootAtTimestampResponse) bool {
			return r.SuperRoot != Bytes32{}
		}

		withRoot := SuperRootAtTimestampResponse{
			SuperRoot: Bytes32{0x11},
		}
		require.True(t, hasSuperRoot(withRoot))

		withoutRoot := SuperRootAtTimestampResponse{
			SuperRoot: Bytes32{},
		}
		require.False(t, hasSuperRoot(withoutRoot))
	})
}
