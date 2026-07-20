package contracts

import (
	"context"
	"errors"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	batchingTest "github.com/ethereum-optimism/optimism/op-service/sources/batching/test"
	"github.com/ethereum-optimism/optimism/packages/contracts-bedrock/snapshots"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

var factoryAddr = common.Address{0xff, 0xff}
var proposerAddr = common.Address{0xaa, 0xbb}

type gameMetadata struct {
	GameType  uint32
	Timestamp time.Time
	Address   common.Address
	Proposer  common.Address
}

func TestHasProposedSince(t *testing.T) {
	cutOffTime := time.Unix(1000, 0)

	gameContractTypes := []struct {
		name string
		abi  *abi.ABI
	}{
		{"FaultDisputeGame", snapshots.LoadFaultDisputeGameABI()},
		{"SuperFaultDisputeGame", snapshots.LoadSuperFaultDisputeGameABI()},
		{"SuperPermissionedDisputeGame", snapshots.LoadSuperPermissionedDisputeGameABI()},
	}

	for _, contractType := range gameContractTypes {
		contractType := contractType
		t.Run("NoProposals-"+contractType.name, func(t *testing.T) {
			stubRpc, factory := setupDisputeGameFactoryTest(t)
			withClaims(stubRpc, contractType.abi)

			proposed, proposalTime, claim, err := factory.HasProposedSince(context.Background(), proposerAddr, cutOffTime, 0)
			require.NoError(t, err)
			require.False(t, proposed)
			require.Equal(t, time.Time{}, proposalTime)
			require.Equal(t, common.Hash{}, claim)
		})

		t.Run("NoMatchingProposal-"+contractType.name, func(t *testing.T) {
			stubRpc, factory := setupDisputeGameFactoryTest(t)
			withClaims(
				stubRpc,
				contractType.abi,
				gameMetadata{
					GameType:  0,
					Timestamp: time.Unix(1600, 0),
					Address:   common.Address{0x22},
					Proposer:  common.Address{0xee}, // Wrong proposer
				},
				gameMetadata{
					GameType:  1, // Wrong game type
					Timestamp: time.Unix(1700, 0),
					Address:   common.Address{0x33},
					Proposer:  proposerAddr,
				},
			)

			proposed, proposalTime, claim, err := factory.HasProposedSince(context.Background(), proposerAddr, cutOffTime, 0)
			require.NoError(t, err)
			require.False(t, proposed)
			require.Equal(t, time.Time{}, proposalTime)
			require.Equal(t, common.Hash{}, claim)
		})

		t.Run("MatchingProposalBeforeCutOff-"+contractType.name, func(t *testing.T) {
			stubRpc, factory := setupDisputeGameFactoryTest(t)
			withClaims(
				stubRpc,
				contractType.abi,
				gameMetadata{
					GameType:  0,
					Timestamp: time.Unix(999, 0),
					Address:   common.Address{0x11},
					Proposer:  proposerAddr,
				},
				gameMetadata{
					GameType:  0,
					Timestamp: time.Unix(1600, 0),
					Address:   common.Address{0x22},
					Proposer:  common.Address{0xee}, // Wrong proposer
				},
				gameMetadata{
					GameType:  1, // Wrong game type
					Timestamp: time.Unix(1700, 0),
					Address:   common.Address{0x33},
					Proposer:  proposerAddr,
				},
			)

			proposed, proposalTime, claim, err := factory.HasProposedSince(context.Background(), proposerAddr, cutOffTime, 0)
			require.NoError(t, err)
			require.False(t, proposed)
			require.Equal(t, time.Time{}, proposalTime)
			require.Equal(t, common.Hash{}, claim)
		})

		t.Run("MatchingProposalAtCutOff-"+contractType.name, func(t *testing.T) {
			stubRpc, factory := setupDisputeGameFactoryTest(t)
			withClaims(
				stubRpc,
				contractType.abi,
				gameMetadata{
					GameType:  0,
					Timestamp: cutOffTime,
					Address:   common.Address{0x11},
					Proposer:  proposerAddr,
				},
				gameMetadata{
					GameType:  0,
					Timestamp: time.Unix(1600, 0),
					Address:   common.Address{0x22},
					Proposer:  common.Address{0xee}, // Wrong proposer
				},
				gameMetadata{
					GameType:  1, // Wrong game type
					Timestamp: time.Unix(1700, 0),
					Address:   common.Address{0x33},
					Proposer:  proposerAddr,
				},
			)

			proposed, proposalTime, claim, err := factory.HasProposedSince(context.Background(), proposerAddr, cutOffTime, 0)
			require.NoError(t, err)
			require.True(t, proposed)
			require.Equal(t, cutOffTime, proposalTime)
			require.Equal(t, common.Hash{0xdd}, claim)
		})

		t.Run("MatchingProposalAfterCutOff-"+contractType.name, func(t *testing.T) {
			stubRpc, factory := setupDisputeGameFactoryTest(t)
			expectedProposalTime := time.Unix(1100, 0)
			withClaims(
				stubRpc,
				contractType.abi,
				gameMetadata{
					GameType:  0,
					Timestamp: expectedProposalTime,
					Address:   common.Address{0x11},
					Proposer:  proposerAddr,
				},
				gameMetadata{
					GameType:  0,
					Timestamp: time.Unix(1600, 0),
					Address:   common.Address{0x22},
					Proposer:  common.Address{0xee}, // Wrong proposer
				},
				gameMetadata{
					GameType:  1, // Wrong game type
					Timestamp: time.Unix(1700, 0),
					Address:   common.Address{0x33},
					Proposer:  proposerAddr,
				},
			)

			proposed, proposalTime, claim, err := factory.HasProposedSince(context.Background(), proposerAddr, cutOffTime, 0)
			require.NoError(t, err)
			require.True(t, proposed)
			require.Equal(t, expectedProposalTime, proposalTime)
			require.Equal(t, common.Hash{0xdd}, claim)
		})

		t.Run("MultipleMatchingProposalAfterCutOff-"+contractType.name, func(t *testing.T) {
			stubRpc, factory := setupDisputeGameFactoryTest(t)
			expectedProposalTime := time.Unix(1600, 0)
			withClaims(
				stubRpc,
				contractType.abi,
				gameMetadata{
					GameType:  0,
					Timestamp: time.Unix(1400, 0),
					Address:   common.Address{0x11},
					Proposer:  proposerAddr,
				},
				gameMetadata{
					GameType:  0,
					Timestamp: time.Unix(1500, 0),
					Address:   common.Address{0x22},
					Proposer:  proposerAddr,
				},
				gameMetadata{
					GameType:  0,
					Timestamp: expectedProposalTime,
					Address:   common.Address{0x33},
					Proposer:  proposerAddr,
				},
			)

			proposed, proposalTime, claim, err := factory.HasProposedSince(context.Background(), proposerAddr, cutOffTime, 0)
			require.NoError(t, err)
			require.True(t, proposed)
			// Should find the most recent proposal
			require.Equal(t, expectedProposalTime, proposalTime)
			require.Equal(t, common.Hash{0xdd}, claim)
		})
	}

	t.Run("HistoricalFaultDisputeGameWithoutGameCreator", func(t *testing.T) {
		stubRpc, factory := setupDisputeGameFactoryTest(t)
		expectedProposalTime := time.Unix(1100, 0)
		expectedClaim := common.Hash{0xcc}
		gameAddress := common.Address{0x44}

		stubRpc.SetResponse(factoryAddr, methodGameCount, rpcblock.Latest, nil, []interface{}{big.NewInt(1)})
		stubRpc.SetResponse(factoryAddr, methodGameAtIndex, rpcblock.Latest, []interface{}{big.NewInt(0)}, []interface{}{
			uint32(0),
			uint64(expectedProposalTime.Unix()),
			gameAddress,
		})

		historicalABI := snapshots.LoadFaultDisputeGameABI()
		// SetError packs the configured ABI outputs before returning the RPC error. The historical
		// contract rejects this selector, so model its response as output-less.
		gameCreator := historicalABI.Methods["gameCreator"]
		gameCreator.Outputs = nil
		historicalABI.Methods["gameCreator"] = gameCreator
		stubRpc.AddContract(gameAddress, historicalABI)
		stubRpc.SetError(gameAddress, "gameCreator", rpcblock.Latest, nil, errors.New("execution reverted: function selector was not recognized"))
		stubRpc.SetResponse(gameAddress, "rootClaim", rpcblock.Latest, nil, []interface{}{expectedClaim})
		stubRpc.SetResponse(gameAddress, "claimData", rpcblock.Latest, []interface{}{big.NewInt(0)}, []interface{}{
			uint32(math.MaxUint32),
			common.Address{},
			proposerAddr,
			big.NewInt(1000),
			expectedClaim,
			big.NewInt(1),
			big.NewInt(100),
		})

		proposed, proposalTime, claim, err := factory.HasProposedSince(context.Background(), proposerAddr, cutOffTime, 0)
		require.NoError(t, err)
		require.True(t, proposed)
		require.Equal(t, expectedProposalTime, proposalTime)
		require.Equal(t, expectedClaim, claim)
	})
	t.Run("SkipsNonMatchingGamesWithoutCallingThem", func(t *testing.T) {
		stubRpc, factory := setupDisputeGameFactoryTest(t)
		expectedProposalTime := time.Unix(1100, 0)
		expectedClaim := common.Hash{0xdd}
		matchingGame := gameMetadata{
			GameType:  0,
			Timestamp: expectedProposalTime,
			Address:   common.Address{0x51},
			Proposer:  proposerAddr,
		}
		withoutClaimData := gameMetadata{
			GameType:  5,
			Timestamp: time.Unix(1200, 0),
			Address:   common.Address{0x52},
		}
		incompatible := gameMetadata{
			GameType:  10,
			Timestamp: time.Unix(1300, 0),
			Address:   common.Address{0x53},
		}

		withGameList(stubRpc, matchingGame, withoutClaimData, incompatible)
		withGameMetadata(stubRpc, matchingGame.Address, snapshots.LoadFaultDisputeGameABI(), proposerAddr, expectedClaim)
		stubRpc.AddContract(withoutClaimData.Address, snapshots.LoadSuperPermissionedDisputeGameABI())
		stubRpc.AddContract(incompatible.Address, &abi.ABI{})

		proposed, proposalTime, claim, err := factory.HasProposedSince(context.Background(), proposerAddr, cutOffTime, 0)
		require.NoError(t, err)
		require.True(t, proposed)
		require.Equal(t, expectedProposalTime, proposalTime)
		require.Equal(t, expectedClaim, claim)
	})

	t.Run("MatchesSuperPermissionedGameType", func(t *testing.T) {
		stubRpc, factory := setupDisputeGameFactoryTest(t)
		expectedProposalTime := time.Unix(1100, 0)
		expectedClaim := common.Hash{0xee}
		game := gameMetadata{
			GameType:  5,
			Timestamp: expectedProposalTime,
			Address:   common.Address{0x54},
			Proposer:  proposerAddr,
		}

		withGameList(stubRpc, game)
		withGameMetadata(stubRpc, game.Address, snapshots.LoadSuperPermissionedDisputeGameABI(), proposerAddr, expectedClaim)

		proposed, proposalTime, claim, err := factory.HasProposedSince(context.Background(), proposerAddr, cutOffTime, 5)
		require.NoError(t, err)
		require.True(t, proposed)
		require.Equal(t, expectedProposalTime, proposalTime)
		require.Equal(t, expectedClaim, claim)
	})

	t.Run("StopsAtCutOffWithoutCallingGame", func(t *testing.T) {
		stubRpc, factory := setupDisputeGameFactoryTest(t)
		game := gameMetadata{
			GameType:  0,
			Timestamp: cutOffTime.Add(-time.Second),
			Address:   common.Address{0x55},
		}

		withGameList(stubRpc, game)

		proposed, proposalTime, claim, err := factory.HasProposedSince(context.Background(), proposerAddr, cutOffTime, 0)
		require.NoError(t, err)
		require.False(t, proposed)
		require.Equal(t, time.Time{}, proposalTime)
		require.Equal(t, common.Hash{}, claim)
	})

}

func TestProposalTx(t *testing.T) {
	stubRpc, factory := setupDisputeGameFactoryTest(t)
	gameType := uint32(123)
	outputRoot := common.Hash{0x01}
	l2BlockNum := common.BigToHash(big.NewInt(456)).Bytes()
	bond := big.NewInt(49284294829)
	stubRpc.SetResponse(factoryAddr, methodInitBonds, rpcblock.Latest, []interface{}{gameType}, []interface{}{bond})
	stubRpc.SetResponse(factoryAddr, methodCreateGame, rpcblock.Latest, []interface{}{gameType, outputRoot, l2BlockNum}, nil)
	tx, err := factory.ProposalTx(context.Background(), gameType, outputRoot, l2BlockNum)
	require.NoError(t, err)
	stubRpc.VerifyTxCandidate(tx)
	require.NotNil(t, tx.Value)
	require.Truef(t, bond.Cmp(tx.Value) == 0, "Expected bond %v but was %v", bond, tx.Value)
}

func withClaims(stubRpc *batchingTest.AbiBasedRpc, gameAbi *abi.ABI, games ...gameMetadata) {
	withGameList(stubRpc, games...)
	for _, game := range games {
		withGameMetadata(stubRpc, game.Address, gameAbi, game.Proposer, common.Hash{0xdd})
	}
}

func withGameList(stubRpc *batchingTest.AbiBasedRpc, games ...gameMetadata) {
	stubRpc.SetResponse(factoryAddr, methodGameCount, rpcblock.Latest, nil, []interface{}{big.NewInt(int64(len(games)))})
	for i, game := range games {
		stubRpc.SetResponse(factoryAddr, methodGameAtIndex, rpcblock.Latest, []interface{}{big.NewInt(int64(i))}, []interface{}{
			game.GameType,
			uint64(game.Timestamp.Unix()),
			game.Address,
		})
	}
}

func withGameMetadata(stubRpc *batchingTest.AbiBasedRpc, gameAddress common.Address, gameABI *abi.ABI, proposer common.Address, claim common.Hash) {
	stubRpc.AddContract(gameAddress, gameABI)
	stubRpc.SetResponse(gameAddress, "gameCreator", rpcblock.Latest, nil, []interface{}{proposer})
	stubRpc.SetResponse(gameAddress, "rootClaim", rpcblock.Latest, nil, []interface{}{claim})
}

func setupDisputeGameFactoryTest(t *testing.T) (*batchingTest.AbiBasedRpc, *DisputeGameFactory) {
	fdgAbi := snapshots.LoadDisputeGameFactoryABI()

	stubRpc := batchingTest.NewAbiBasedRpc(t, factoryAddr, fdgAbi)
	caller := batching.NewMultiCaller(stubRpc, batching.DefaultBatchSize)
	factory := NewDisputeGameFactory(factoryAddr, caller, time.Minute)
	return stubRpc, factory
}
