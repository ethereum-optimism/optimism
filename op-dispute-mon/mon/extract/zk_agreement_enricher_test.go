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
		{name: "matching root", providers: []SuperRootProvider{zkFound(101, root)}, claim: root, agree: true, expected: root, fetchRecorded: true},
		{name: "mismatching root", providers: []SuperRootProvider{zkFound(101, root)}, claim: otherRoot, expected: root, fetchRecorded: true},
		{name: "all behind", providers: []SuperRootProvider{zkFound(100, root), zkFound(99, otherRoot)}, wantErr: gameTypes.ErrNotInSync, outOfSync: 2},
		{name: "all errors", providers: []SuperRootProvider{zkError(errors.New("first")), zkError(errors.New("second"))}, wantErr: ErrAllSuperRootRpcsUnavailable, errors: 2},
		{name: "behind and error", providers: []SuperRootProvider{zkFound(100, root), zkError(errors.New("boom"))}, wantErr: ErrAllSuperRootRpcsUnavailable, errors: 1, outOfSync: 1},
		{name: "usable and error", providers: []SuperRootProvider{zkError(errors.New("boom")), zkFound(101, root)}, claim: root, agree: true, expected: root, errors: 1, fetchRecorded: true},
		{name: "usable and behind", providers: []SuperRootProvider{zkFound(100, otherRoot), zkFound(101, root)}, claim: root, agree: true, expected: root, outOfSync: 1, fetchRecorded: true},
		{name: "all not found", providers: []SuperRootProvider{zkNotFound(101), zkNotFound(102)}, notFound: 2},
		{name: "found and not found", providers: []SuperRootProvider{zkFound(101, root), zkNotFound(101)}, claim: root, expected: root, notFound: 1, mixedAvailability: true, fetchRecorded: true},
		{name: "conflicting roots use first provider", providers: []SuperRootProvider{zkFound(101, root), zkFound(101, otherRoot)}, claim: root, expected: root, differentRoots: true, fetchRecorded: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metricer := &stubOutputMetrics{}
			enricher := NewZKAgreementEnricher(
				testlog.Logger(t, log.LvlDebug),
				metricer,
				test.providers,
				clock.NewDeterministicClock(time.Unix(1234, 0)),
			)
			game := newZKAgreementGame(test.claim)

			err := enricher.Enrich(t.Context(), rpcblock.Latest, nil, game)
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

func TestZKAgreementCancellationPreventsMutation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	root := common.Hash{0xaa}
	provider := zkSuperRootProvider{response: zkResponse(101, &root), action: cancel}
	metricer := &stubOutputMetrics{}
	enricher := NewZKAgreementEnricher(
		testlog.Logger(t, log.LvlDebug),
		metricer,
		[]SuperRootProvider{provider},
		clock.NewDeterministicClock(time.Unix(1234, 0)),
	)
	game := newZKAgreementGame(root)

	require.ErrorIs(t, enricher.Enrich(ctx, rpcblock.Latest, nil, game), context.Canceled)
	require.Zero(t, game.NodeEndpointTotalCount)
	require.Empty(t, game.NodeEndpointErrors)
	require.Zero(t, metricer.fetchTime)
}

func TestZKAgreementRequiresProvider(t *testing.T) {
	enricher := NewZKAgreementEnricher(
		testlog.Logger(t, log.LvlDebug),
		&stubOutputMetrics{},
		nil,
		clock.NewDeterministicClock(time.Unix(1234, 0)),
	)
	require.ErrorIs(t, enricher.Enrich(t.Context(), rpcblock.Latest, nil, newZKAgreementGame(common.Hash{})), ErrSuperRootRpcRequired)
}

func newZKAgreementGame(claim common.Hash) *monTypes.ZKGameData {
	return &monTypes.ZKGameData{CommonGameData: monTypes.CommonGameData{
		GameMetadata:       gameTypes.GameMetadata{GameType: uint32(gameTypes.ZKDisputeGameType)},
		L1HeadNum:          100,
		L2SequenceNumber:   44,
		RootClaim:          claim,
		NodeEndpointErrors: make(map[string]bool),
	}}
}

type zkSuperRootProvider struct {
	response eth.SuperRootAtTimestampResponse
	err      error
	action   func()
}

func (p zkSuperRootProvider) SuperRootAtTimestamp(context.Context, uint64) (eth.SuperRootAtTimestampResponse, error) {
	if p.action != nil {
		p.action()
	}
	return p.response, p.err
}

func zkResponse(currentL1 uint64, root *common.Hash) eth.SuperRootAtTimestampResponse {
	response := eth.SuperRootAtTimestampResponse{CurrentL1: eth.BlockID{Number: currentL1}}
	if root != nil {
		response.Data = &eth.SuperRootResponseData{SuperRoot: eth.Bytes32(*root)}
	}
	return response
}

func zkFound(currentL1 uint64, root common.Hash) SuperRootProvider {
	return zkSuperRootProvider{response: zkResponse(currentL1, &root)}
}

func zkNotFound(currentL1 uint64) SuperRootProvider {
	return zkSuperRootProvider{response: zkResponse(currentL1, nil)}
}

func zkError(err error) SuperRootProvider {
	return zkSuperRootProvider{err: err}
}
