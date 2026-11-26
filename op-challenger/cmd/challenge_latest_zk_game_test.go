package main

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestSelectZKGameToChallenge(t *testing.T) {
	owned := common.Address{0x01}
	other := common.Address{0x02}
	now := time.Unix(1_800_000_000, 0)
	open := now.Add(time.Hour)

	type zkGame struct {
		creator  common.Address
		status   contracts.ProposalStatus
		deadline time.Time
	}
	// run stubs the ZK games, at factory indices len(games)-1 down to 0 so they are ordered newest first, plus a
	// newer non-ZK game that must be skipped without any calls. It returns the selected index.
	run := func(t *testing.T, games ...zkGame) (uint64, error) {
		nonZK := gameTypes.GameMetadata{Index: uint64(len(games)), GameType: uint32(gameTypes.CannonGameType), Proxy: common.Address{0xff}}
		metadata := []gameTypes.GameMetadata{nonZK}
		stubRPC := batchingTest.NewAbiBasedRpc(t, common.Address{}, snapshots.LoadZKDisputeGameABI())
		for i, game := range games {
			addr := common.Address{0xa0, byte(i)}
			stubRPC.AddContract(addr, snapshots.LoadZKDisputeGameABI())
			stubRPC.SetResponse(addr, "gameCreator", rpcblock.Latest, nil, []interface{}{game.creator})
			stubRPC.SetResponse(addr, "totalBonds", rpcblock.Latest, nil, []interface{}{big.NewInt(0)})
			stubRPC.SetResponse(addr, "challengerBond", rpcblock.Latest, nil, []interface{}{big.NewInt(0)})
			stubRPC.SetResponse(addr, "claimData", rpcblock.Latest, nil, []interface{}{
				uint32(0), uint8(game.status), common.Address{}, common.Address{}, uint64(game.deadline.Unix()), common.Hash{},
			})
			stubRPC.SetResponse(addr, "l2SequenceNumber", rpcblock.Latest, nil, []interface{}{big.NewInt(0)})
			metadata = append(metadata, gameTypes.GameMetadata{
				Index:    uint64(len(games) - 1 - i),
				GameType: uint32(gameTypes.ZKDisputeGameType),
				Proxy:    addr,
			})
		}
		caller := batching.NewMultiCaller(stubRPC, batching.DefaultBatchSize)
		selected, game, err := selectZKGameToChallenge(context.Background(), testlog.Logger(t, log.LevelInfo), caller, metadata, []common.Address{owned}, now)
		if err != nil {
			return 0, err
		}
		require.Equal(t, selected.Proxy, game.Addr())
		return selected.Index, nil
	}

	t.Run("SkipsOtherCreators", func(t *testing.T) {
		index, err := run(t,
			zkGame{other, contracts.ProposalStatusUnchallenged, open},
			zkGame{owned, contracts.ProposalStatusUnchallenged, open},
			zkGame{owned, contracts.ProposalStatusUnchallenged, open},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), index)
	})

	t.Run("SkipsChallengedAndClosingGames", func(t *testing.T) {
		index, err := run(t,
			zkGame{owned, contracts.ProposalStatusChallenged, open},
			zkGame{owned, contracts.ProposalStatusUnchallengedAndValidProofProvided, open},
			zkGame{owned, contracts.ProposalStatusUnchallenged, now.Add(minChallengeTimeRemaining - time.Second)},
			zkGame{owned, contracts.ProposalStatusUnchallenged, now.Add(minChallengeTimeRemaining)},
		)
		require.NoError(t, err)
		require.Equal(t, uint64(0), index)
	})

	t.Run("NoEligibleGame", func(t *testing.T) {
		_, err := run(t,
			zkGame{other, contracts.ProposalStatusUnchallenged, open},
			zkGame{owned, contracts.ProposalStatusChallenged, open},
		)
		require.EqualError(t, err, "no challengeable ZK game from the given creators")
	})
}
