package extract

import (
	"context"
	"errors"
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestZKAgreementPolicy(t *testing.T) {
	root := common.Hash{0xaa}
	otherRoot := common.Hash{0xbb}
	tests := []struct {
		name              string
		providers         []SuperRootProvider
		claim             common.Hash
		wantErr           error
		agree             bool
		expected          common.Hash
		errors            int
		notFound          int
		outOfSync         int
		differentRoots    bool
		mixedAvailability bool
		fetchRecorded     bool
	}{
		{
			name:          "matching root",
			providers:     []SuperRootProvider{zkFound(101, root, 999)},
			claim:         root,
			agree:         true,
			expected:      root,
			fetchRecorded: true,
		},
		{
			name:          "mismatching root",
			providers:     []SuperRootProvider{zkFound(101, root, 999)},
			claim:         otherRoot,
			expected:      root,
			fetchRecorded: true,
		},
		{
			name:      "all not found",
			providers: []SuperRootProvider{zkNotFound(101), zkNotFound(102)},
			notFound:  2,
		},
		{
			name:              "mixed found and not found",
			providers:         []SuperRootProvider{zkFound(101, root, 0), zkNotFound(101)},
			claim:             root,
			notFound:          1,
			mixedAvailability: true,
			fetchRecorded:     true,
		},
		{
			name:           "different found roots",
			providers:      []SuperRootProvider{zkFound(101, root, 0), zkFound(101, otherRoot, 0)},
			claim:          root,
			differentRoots: true,
			fetchRecorded:  true,
		},
		{
			name:          "partial errors are excluded",
			providers:     []SuperRootProvider{zkError(errors.New("boom")), zkFound(101, root, 0)},
			claim:         root,
			agree:         true,
			expected:      root,
			errors:        1,
			fetchRecorded: true,
		},
		{
			name:          "out of sync is excluded",
			providers:     []SuperRootProvider{zkFound(100, otherRoot, 0), zkFound(101, root, 0)},
			claim:         root,
			agree:         true,
			expected:      root,
			outOfSync:     1,
			fetchRecorded: true,
		},
		{
			name:      "current L1 below head is out of sync",
			providers: []SuperRootProvider{zkFound(99, root, 0)},
			claim:     root,
			wantErr:   ErrAllSuperRootRpcsUnavailable,
			outOfSync: 1,
		},
		{
			name:      "all errors fail",
			providers: []SuperRootProvider{zkError(errors.New("boom"))},
			wantErr:   ErrAllSuperRootRpcsUnavailable,
			errors:    1,
		},
		{
			name:              "mixed availability subtracts out of sync",
			providers:         []SuperRootProvider{zkFound(101, root, 0), zkNotFound(101), zkFound(100, otherRoot, 0)},
			claim:             root,
			notFound:          1,
			outOfSync:         1,
			mixedAvailability: true,
			fetchRecorded:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := testlog.Logger(t, log.LvlDebug)
			metricer := &stubOutputMetrics{}
			enricher := NewZKAgreementEnricher(
				logger,
				metricer,
				test.providers,
				clock.NewDeterministicClock(time.Unix(1234, 0)),
			)
			game := &monTypes.CommonGameData{
				GameMetadata:       gameTypes.GameMetadata{GameType: uint32(gameTypes.ZKDisputeGameType)},
				L1HeadNum:          100,
				L2SequenceNumber:   44,
				RootClaim:          test.claim,
				NodeEndpointErrors: make(map[string]bool),
			}

			err := enricher.Enrich(context.Background(), rpcblock.Latest, nil, game)
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, test.agree, game.AgreeWithClaim)
			require.Equal(t, test.expected, game.ExpectedRootClaim)
			require.Equal(t, len(test.providers), game.NodeEndpointTotalCount)
			require.Equal(t, test.errors, game.NodeEndpointErrorCount)
			require.Len(t, game.NodeEndpointErrors, test.errors)
			require.Equal(t, test.notFound, game.NodeEndpointNotFoundCount)
			require.Equal(t, test.outOfSync, game.NodeEndpointOutOfSyncCount)
			require.Equal(t, test.differentRoots, game.NodeEndpointDifferentRoots)
			require.Equal(t, test.mixedAvailability, game.HasMixedAvailability())
			require.Zero(t, game.NodeEndpointSafeCount)
			require.Zero(t, game.NodeEndpointUnsafeCount)
			require.Equal(t, test.fetchRecorded, metricer.fetchTime != 0)
		})
	}
}

func TestZKAgreementRequiresProvider(t *testing.T) {
	enricher := NewZKAgreementEnricher(
		testlog.Logger(t, log.LvlDebug),
		&stubOutputMetrics{},
		nil,
		clock.NewDeterministicClock(time.Unix(1234, 0)),
	)
	game := &monTypes.CommonGameData{
		GameMetadata:       gameTypes.GameMetadata{GameType: uint32(gameTypes.ZKDisputeGameType)},
		NodeEndpointErrors: make(map[string]bool),
	}
	require.ErrorIs(t, enricher.Enrich(context.Background(), rpcblock.Latest, nil, game), ErrSuperRootRpcRequired)
}

type zkSuperRootProvider struct {
	response eth.SuperRootAtTimestampResponse
	err      error
}

func (p zkSuperRootProvider) SuperRootAtTimestamp(context.Context, uint64) (eth.SuperRootAtTimestampResponse, error) {
	return p.response, p.err
}

func zkFound(currentL1 uint64, root common.Hash, verifiedRequiredL1 uint64) SuperRootProvider {
	return zkSuperRootProvider{response: eth.SuperRootAtTimestampResponse{
		CurrentL1: eth.BlockID{Number: currentL1},
		Data: &eth.SuperRootResponseData{
			VerifiedRequiredL1: eth.BlockID{Number: verifiedRequiredL1},
			SuperRoot:          eth.Bytes32(root),
		},
	}}
}

func zkNotFound(currentL1 uint64) SuperRootProvider {
	return zkSuperRootProvider{response: eth.SuperRootAtTimestampResponse{
		CurrentL1: eth.BlockID{Number: currentL1},
	}}
}

func zkError(err error) SuperRootProvider {
	return zkSuperRootProvider{err: err}
}
