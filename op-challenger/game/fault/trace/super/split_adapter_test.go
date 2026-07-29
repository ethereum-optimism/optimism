package super

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var creatorError = errors.New("captured args")

func TestSplitAdapter(t *testing.T) {
	depth := types.Depth(30)
	prestateTimestamp := uint64(150)
	poststateTimestamp := uint64(200)

	t.Run("FromAbsolutePrestate", func(t *testing.T) {
		creator, rootProvider, adapter := setupSplitAdapterTest(t, depth, prestateTimestamp, poststateTimestamp)
		expectedSuper := eth.NewSuperV1(prestateTimestamp)
		rootProvider.AddAtTimestamp(prestateTimestamp, eth.SuperRootAtTimestampResponse{
			Data: &eth.SuperRootResponseData{Super: expectedSuper},
		})
		postClaim := types.Claim{
			ClaimData: types.ClaimData{
				Value:    common.Hash{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
				Position: types.NewPosition(depth, big.NewInt(0)),
			},
		}
		expectedClaimInfo := ClaimInfo{
			AgreedPrestate: expectedSuper.Marshal(),
			Claim:          postClaim.Value,
		}
		_, err := adapter(context.Background(), depth, types.Claim{}, postClaim)
		require.ErrorIs(t, err, creatorError)
		require.Equal(t, split.CreateLocalContext(types.Claim{}, postClaim), creator.localContext)
		require.Equal(t, expectedClaimInfo, creator.claimInfo)
	})

	t.Run("AfterClaimedBlock", func(t *testing.T) {
		creator, rootProvider, adapter := setupSplitAdapterTest(t, depth, prestateTimestamp, poststateTimestamp)
		l1Head := eth.BlockID{}
		expectedSuper := eth.NewSuperV1(poststateTimestamp)
		rootProvider.AddAtTimestamp(poststateTimestamp, eth.SuperRootAtTimestampResponse{
			CurrentL1: inSyncCurrentL1(l1Head),
			Data:      &eth.SuperRootResponseData{Super: expectedSuper},
		})
		preClaim := types.Claim{
			ClaimData: types.ClaimData{
				Value:    common.Hash{0x11},
				Position: types.NewPosition(depth, big.NewInt(999_999)),
			},
		}
		postClaim := types.Claim{
			ClaimData: types.ClaimData{
				Value:    common.Hash{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
				Position: types.NewPosition(depth, big.NewInt(1_000_000)),
			},
		}
		expectedClaimInfo := ClaimInfo{
			AgreedPrestate: expectedSuper.Marshal(),
			Claim:          postClaim.Value,
		}
		_, err := adapter(context.Background(), depth, preClaim, postClaim)
		require.ErrorIs(t, err, creatorError)
		require.Equal(t, split.CreateLocalContext(preClaim, postClaim), creator.localContext)
		require.Equal(t, expectedClaimInfo, creator.claimInfo)
	})

	t.Run("MiddleOfTimestampTransition", func(t *testing.T) {
		creator, rootProvider, adapter := setupSplitAdapterTest(t, depth, prestateTimestamp, poststateTimestamp)
		l1Head := eth.BlockID{}
		prevSuper := eth.NewSuperV1(prestateTimestamp)
		rootProvider.AddAtTimestamp(prestateTimestamp, eth.SuperRootAtTimestampResponse{
			CurrentL1: inSyncCurrentL1(l1Head),
			Data:      &eth.SuperRootResponseData{Super: prevSuper},
		})
		rootProvider.AddAtTimestamp(prestateTimestamp+1, eth.SuperRootAtTimestampResponse{
			CurrentL1: inSyncCurrentL1(l1Head),
			Data:      &eth.SuperRootResponseData{Super: eth.NewSuperV1(prestateTimestamp + 1)},
		})
		preClaim := types.Claim{
			ClaimData: types.ClaimData{
				Value:    common.Hash{0x11},
				Position: types.NewPosition(depth, big.NewInt(2)),
			},
		}
		postClaim := types.Claim{
			ClaimData: types.ClaimData{
				Value:    common.Hash{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
				Position: types.NewPosition(depth, big.NewInt(3)),
			},
		}
		expectedPrestate := eth.TransitionState{
			SuperRoot: prevSuper.Marshal(),
			Step:      3,
		}
		expectedClaimInfo := ClaimInfo{
			AgreedPrestate: expectedPrestate.Marshal(),
			Claim:          postClaim.Value,
		}
		_, err := adapter(context.Background(), depth, preClaim, postClaim)
		require.ErrorIs(t, err, creatorError)
		require.Equal(t, split.CreateLocalContext(preClaim, postClaim), creator.localContext)
		require.Equal(t, expectedClaimInfo, creator.claimInfo)
	})
}

func setupSplitAdapterTest(t *testing.T, depth types.Depth, prestateTimestamp uint64, poststateTimestamp uint64) (*capturingCreator, *stubSuperNodeRootProvider, split.ProviderCreator) {
	creator := &capturingCreator{}
	rootProvider := &stubSuperNodeRootProvider{}
	prestateProvider := NewSuperNodePrestateProvider(rootProvider, prestateTimestamp)
	traceProvider := NewSuperNodeTraceProvider(testlog.Logger(t, log.LvlInfo), prestateProvider, rootProvider, eth.BlockID{}, depth, prestateTimestamp, poststateTimestamp)
	adapter := SuperRootSplitAdapter(traceProvider, creator.Create)
	return creator, rootProvider, adapter
}

type capturingCreator struct {
	localContext common.Hash
	claimInfo    ClaimInfo
}

func (c *capturingCreator) Create(_ context.Context, localContext common.Hash, _ types.Depth, claimInfo ClaimInfo) (types.TraceProvider, error) {
	c.localContext = localContext
	c.claimInfo = claimInfo
	return nil, creatorError
}
