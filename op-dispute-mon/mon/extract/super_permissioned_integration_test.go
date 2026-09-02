package extract

import (
	"context"
	"math/big"
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestExtractorChecksSuperPermissionedGame(t *testing.T) {
	gameAddr := common.Address{0xaa}
	blockHash := common.Hash{0xbb}
	l1Head := common.Hash{0xcc}
	l1HeadNum := uint64(200)
	l2SequenceNumber := uint64(1234)

	rpc := batchingTest.NewAbiBasedRpc(t, gameAddr, snapshots.LoadSuperPermissionedDisputeGameABI())
	caller := batching.NewMultiCaller(rpc, batching.DefaultBatchSize)
	block := rpcblock.ByHash(blockHash)
	rpc.SetResponse(gameAddr, "l1Head", block, nil, []any{l1Head})
	rpc.SetResponse(gameAddr, "l2SequenceNumber", block, nil, []any{new(big.Int).SetUint64(l2SequenceNumber)})
	rpc.SetResponse(gameAddr, "rootClaim", block, nil, []any{mockRootClaim})
	rpc.SetResponse(gameAddr, "status", block, nil, []any{uint8(gameTypes.GameStatusDefenderWon)})

	fetchGames := func(context.Context, common.Hash, uint64) ([]gameTypes.GameMetadata, error) {
		return []gameTypes.GameMetadata{
			{
				GameType:  uint32(gameTypes.SuperPermissionedGameType),
				Proxy:     gameAddr,
				Timestamp: uint64(time.Hour.Seconds()),
			},
		}, nil
	}
	provider := &stubSuperRootProvider{
		derivedFromL1BlockNum: l1HeadNum,
		superRoot:             mockRootClaim,
	}
	metrics := &stubOutputMetrics{}
	logger := testlog.Logger(t, log.LvlInfo)
	creator := NewGameCallerCreator(&mockCacheMetrics{}, caller)
	extractor := NewExtractor(
		logger,
		clock.NewDeterministicClock(time.Unix(48294294, 58)),
		new(stubGamesWaitingForRootSourceMetrics),
		creator.CreateContract,
		fetchGames,
		nil,
		nil,
		1,
		[]CommonEnricher{
			NewL1HeadBlockNumEnricher(&stubBlockFetcher{num: l1HeadNum}),
			NewSuperAgreementEnricher(logger, metrics, []SuperRootProvider{provider}, clock.NewDeterministicClock(time.Unix(9824924, 499))),
		},
		nil,
		nil,
		nil,
	)

	games, ignored, failed, err := extractor.Extract(context.Background(), blockHash, 0)
	require.NoError(t, err)
	require.Zero(t, ignored)
	require.Zero(t, failed)
	require.Len(t, games, 1)

	game, ok := games[0].(*monTypes.SuperPermissionedGameData)
	require.True(t, ok)
	require.Equal(t, uint32(gameTypes.SuperPermissionedGameType), game.GameType)
	require.Equal(t, l1Head, game.L1Head)
	require.Equal(t, l1HeadNum, game.L1HeadNum)
	require.Equal(t, l2SequenceNumber, game.L2SequenceNumber)
	require.Equal(t, gameTypes.GameStatusDefenderWon, game.Status)
	require.True(t, game.AgreeWithClaim)
	require.Equal(t, mockRootClaim, game.ExpectedRootClaim)
	require.Equal(t, l2SequenceNumber, provider.requestedTimestamp)
	require.NotZero(t, metrics.fetchTime)
}
