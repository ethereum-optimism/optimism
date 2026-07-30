package extract

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	monTypes "github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestExtractorRoutesEverySupportedLifecycleGameType(t *testing.T) {
	games := make([]gameTypes.GameMetadata, 0, len(gameTypes.SupportedLifecycleGameTypes))
	faultCallers := make(map[common.Address]*mockGameCaller)
	zkCaller := validZKCaller()
	superPermissionedCaller := &routingSuperPermissionedCaller{}
	for i, gameType := range gameTypes.SupportedLifecycleGameTypes {
		metadata := gameTypes.GameMetadata{
			Index:    uint64(i),
			GameType: uint32(gameType),
			Proxy:    common.Address{byte(i + 1)},
		}
		games = append(games, metadata)
		switch gameType {
		case gameTypes.ZKDisputeGameType, gameTypes.SuperPermissionedGameType:
		default:
			faultCallers[metadata.Proxy] = &mockGameCaller{}
		}
	}
	commonEnricher := &recordingCommonEnricher{}
	faultEnricher := &recordingFaultEnricher{}
	zkAgreement := &recordingCommonEnricher{action: func(game *monTypes.CommonGameData) error {
		game.AgreeWithClaim = true
		game.ExpectedRootClaim = game.RootClaim
		return nil
	}}
	creator := func(_ context.Context, game gameTypes.GameMetadata) (GameCaller, error) {
		switch gameTypes.GameType(game.GameType) {
		case gameTypes.ZKDisputeGameType:
			return zkCaller, nil
		case gameTypes.SuperPermissionedGameType:
			return superPermissionedCaller, nil
		default:
			return faultCallers[game.Proxy], nil
		}
	}
	extractor := NewExtractor(
		testlog.Logger(t, log.LvlDebug),
		clock.NewDeterministicClock(time.Unix(1234, 0)),
		creator,
		func(context.Context, common.Hash, uint64) ([]gameTypes.GameMetadata, error) {
			return games, nil
		},
		nil,
		nil,
		5,
		[]CommonEnricher{commonEnricher},
		[]FaultEnricher{faultEnricher},
		zkAgreement,
	)

	extracted, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{0xaa}, 0)
	require.NoError(t, err)
	require.Zero(t, ignored)
	require.Zero(t, failed)
	require.Len(t, extracted, len(gameTypes.SupportedLifecycleGameTypes))
	require.Equal(t, len(gameTypes.SupportedLifecycleGameTypes), commonEnricher.Calls())
	require.Equal(t, len(gameTypes.SupportedLifecycleGameTypes)-2, faultEnricher.Calls())
	require.Equal(t, 1, zkAgreement.Calls())

	var faultCount, zkCount, superPermissionedCount int
	for _, game := range extracted {
		switch game.(type) {
		case *monTypes.FaultGameData:
			faultCount++
		case *monTypes.ZKGameData:
			zkCount++
		case *monTypes.SuperPermissionedGameData:
			superPermissionedCount++
		default:
			t.Fatalf("unexpected enriched game type %T", game)
		}
	}
	require.Equal(t, len(gameTypes.SupportedLifecycleGameTypes)-2, faultCount)
	require.Equal(t, 1, zkCount)
	require.Equal(t, 1, superPermissionedCount)
	require.Equal(t, 1, zkCaller.metadataCalls)
	require.Equal(t, 1, zkCaller.challengerMetadataCalls)
	require.Equal(t, 1, superPermissionedCaller.metadataCalls)
}

func TestExtractorValidatesPinnedZKState(t *testing.T) {
	tests := []struct {
		name         string
		configure    func(*routingZKCaller)
		parentStatus gameTypes.GameStatus
		wantErr      string
	}{
		{
			name: "root mismatch",
			configure: func(caller *routingZKCaller) {
				caller.challengerMetadata.ProposedRoot = common.Hash{0xee}
			},
			wantErr: "inconsistent ZK root claim",
		},
		{
			name: "sequence mismatch",
			configure: func(caller *routingZKCaller) {
				caller.challengerMetadata.L2SequenceNumber++
			},
			wantErr: "inconsistent ZK sequence number",
		},
		{
			name: "resolved proposal with live global status",
			configure: func(caller *routingZKCaller) {
				caller.challengerMetadata.ProposalStatus = contracts.ProposalStatusResolved
			},
			wantErr: "inconsistent ZK global status",
		},
		{
			name: "live proposal with terminal global status",
			configure: func(caller *routingZKCaller) {
				caller.metadata.Status = gameTypes.GameStatusDefenderWon
			},
			wantErr: "inconsistent ZK global status",
		},
		{
			name: "terminal child with live parent",
			configure: func(caller *routingZKCaller) {
				caller.metadata.Status = gameTypes.GameStatusDefenderWon
				caller.challengerMetadata.ProposalStatus = contracts.ProposalStatusResolved
				caller.challengerMetadata.ParentIndex = 7
			},
			parentStatus: gameTypes.GameStatusInProgress,
			wantErr:      "terminal ZK child has in-progress parent 7",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caller := validZKCaller()
			test.configure(caller)
			extractor, _ := newZKExtractor(t, caller, func(context.Context, uint64, rpcblock.Block) (gameTypes.GameStatus, error) {
				return test.parentStatus, nil
			}, &recordingCommonEnricher{})
			game, err := extractor.enrichGame(context.Background(), common.Hash{0xaa}, zkMetadata())
			require.Nil(t, game)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestExtractorZKParentSemantics(t *testing.T) {
	t.Run("max uint32 is the only no-parent sentinel", func(t *testing.T) {
		caller := validZKCaller()
		parentCalls := 0
		extractor, _ := newZKExtractor(t, caller, func(context.Context, uint64, rpcblock.Block) (gameTypes.GameStatus, error) {
			parentCalls++
			return gameTypes.GameStatusDefenderWon, nil
		}, &recordingCommonEnricher{})
		game, err := extractor.enrichGame(context.Background(), common.Hash{0xaa}, zkMetadata())
		require.NoError(t, err)
		zkGame := game.(*monTypes.ZKGameData)
		require.False(t, zkGame.HasParent)
		require.Equal(t, uint32(math.MaxUint32), zkGame.ParentIndex)
		require.Zero(t, parentCalls)
	})

	t.Run("index zero is a valid parent resolved by direct pinned lookup", func(t *testing.T) {
		caller := validZKCaller()
		caller.challengerMetadata.ParentIndex = 0
		var requestedIndex uint64
		var requestedBlock rpcblock.Block
		extractor, _ := newZKExtractor(t, caller, func(_ context.Context, index uint64, block rpcblock.Block) (gameTypes.GameStatus, error) {
			requestedIndex = index
			requestedBlock = block
			return gameTypes.GameStatusDefenderWon, nil
		}, &recordingCommonEnricher{})
		blockHash := common.Hash{0xaa}
		game, err := extractor.enrichGame(context.Background(), blockHash, zkMetadata())
		require.NoError(t, err)
		zkGame := game.(*monTypes.ZKGameData)
		require.True(t, zkGame.HasParent)
		require.Equal(t, uint32(0), zkGame.ParentIndex)
		require.Equal(t, gameTypes.GameStatusDefenderWon, zkGame.ParentStatus)
		require.Zero(t, requestedIndex)
		require.Equal(t, rpcblock.ByHash(blockHash), requestedBlock)
	})

	t.Run("parent outside scan is resolved by factory index", func(t *testing.T) {
		caller := validZKCaller()
		caller.challengerMetadata.ParentIndex = 1234
		var requestedIndex uint64
		extractor, _ := newZKExtractor(t, caller, func(_ context.Context, index uint64, _ rpcblock.Block) (gameTypes.GameStatus, error) {
			requestedIndex = index
			return gameTypes.GameStatusChallengerWon, nil
		}, &recordingCommonEnricher{})
		game, err := extractor.enrichGame(context.Background(), common.Hash{0xaa}, zkMetadata())
		require.NoError(t, err)
		require.Equal(t, uint64(1234), requestedIndex)
		require.Equal(t, gameTypes.GameStatusChallengerWon, game.(*monTypes.ZKGameData).ParentStatus)
	})
}

func TestExtractorZKCacheFallbackIsImmutable(t *testing.T) {
	for _, test := range []struct {
		name string
		fail func(*parentStatusSource, *recordingCommonEnricher)
	}{
		{
			name: "parent refresh failure",
			fail: func(parent *parentStatusSource, _ *recordingCommonEnricher) {
				parent.err = errors.New("parent unavailable")
			},
		},
		{
			name: "root source refresh failure",
			fail: func(_ *parentStatusSource, agreement *recordingCommonEnricher) {
				agreement.err = errors.New("root source unavailable")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			caller := validZKCaller()
			caller.challengerMetadata.ParentIndex = 7
			parent := &parentStatusSource{status: gameTypes.GameStatusDefenderWon}
			agreement := &recordingCommonEnricher{action: func(game *monTypes.CommonGameData) error {
				game.AgreeWithClaim = true
				game.ExpectedRootClaim = game.RootClaim
				return nil
			}}
			extractor, fetcher := newZKExtractor(t, caller, parent.Fetch, agreement)

			first, _, failed, err := extractor.Extract(context.Background(), common.Hash{0xaa}, 0)
			require.NoError(t, err)
			require.Zero(t, failed)
			require.Len(t, first, 1)
			snapshot := first[0].(*monTypes.ZKGameData)
			require.Equal(t, gameTypes.GameStatusDefenderWon, snapshot.ParentStatus)
			require.Equal(t, snapshot.RootClaim, snapshot.ExpectedRootClaim)

			test.fail(parent, agreement)
			fetcher.blockHash = common.Hash{0xbb}
			second, _, failed, err := extractor.Extract(context.Background(), fetcher.blockHash, 0)
			require.NoError(t, err)
			require.Equal(t, 1, failed)
			require.Len(t, second, 1)
			require.Same(t, snapshot, second[0])
			require.Equal(t, gameTypes.GameStatusDefenderWon, snapshot.ParentStatus)
			require.Equal(t, snapshot.RootClaim, snapshot.ExpectedRootClaim)
		})
	}
}

type routingZKCaller struct {
	metadata                contracts.GenericGameMetadata
	challengerMetadata      contracts.ChallengerMetadata
	metadataCalls           int
	challengerMetadataCalls int
}

func validZKCaller() *routingZKCaller {
	return &routingZKCaller{
		metadata: contracts.GenericGameMetadata{
			L1Head:        common.Hash{0x11},
			L2SequenceNum: 99,
			ProposedRoot:  common.Hash{0x22},
			Status:        gameTypes.GameStatusInProgress,
		},
		challengerMetadata: contracts.ChallengerMetadata{
			ParentIndex:      math.MaxUint32,
			ProposalStatus:   contracts.ProposalStatusUnchallenged,
			ProposedRoot:     common.Hash{0x22},
			L2SequenceNumber: 99,
		},
	}
}

func (c *routingZKCaller) GetMetadata(context.Context, rpcblock.Block) (contracts.GenericGameMetadata, error) {
	c.metadataCalls++
	return c.metadata, nil
}

func (c *routingZKCaller) GetChallengerMetadata(context.Context, rpcblock.Block) (contracts.ChallengerMetadata, error) {
	c.challengerMetadataCalls++
	return c.challengerMetadata, nil
}

func (*routingZKCaller) GetAnchorStateRegistry(context.Context, rpcblock.Block) (common.Address, error) {
	return common.Address{}, nil
}

type routingSuperPermissionedCaller struct {
	metadataCalls int
}

func (c *routingSuperPermissionedCaller) GetExtendedMetadata(context.Context, rpcblock.Block) (contracts.GameMetadata, error) {
	c.metadataCalls++
	return contracts.GameMetadata{Status: gameTypes.GameStatusDefenderWon}, nil
}

func (*routingSuperPermissionedCaller) GetAnchorStateRegistry(context.Context, rpcblock.Block) (common.Address, error) {
	return common.Address{}, nil
}

type recordingCommonEnricher struct {
	mu     sync.Mutex
	calls  int
	err    error
	action func(*monTypes.CommonGameData) error
}

func (e *recordingCommonEnricher) Enrich(_ context.Context, _ rpcblock.Block, _ GameCaller, game *monTypes.CommonGameData) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	if e.err != nil {
		return e.err
	}
	if e.action != nil {
		return e.action(game)
	}
	return nil
}

func (e *recordingCommonEnricher) Calls() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

type recordingFaultEnricher struct {
	mu    sync.Mutex
	calls int
}

func (e *recordingFaultEnricher) Enrich(context.Context, rpcblock.Block, FaultGameCaller, *monTypes.FaultGameData) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	return nil
}

func (e *recordingFaultEnricher) Calls() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

type parentStatusSource struct {
	status gameTypes.GameStatus
	err    error
}

func (s *parentStatusSource) Fetch(context.Context, uint64, rpcblock.Block) (gameTypes.GameStatus, error) {
	return s.status, s.err
}

type zkGameFetcher struct {
	blockHash common.Hash
}

func (f *zkGameFetcher) Fetch(context.Context, common.Hash, uint64) ([]gameTypes.GameMetadata, error) {
	return []gameTypes.GameMetadata{zkMetadata()}, nil
}

func zkMetadata() gameTypes.GameMetadata {
	return gameTypes.GameMetadata{
		Index:    55,
		GameType: uint32(gameTypes.ZKDisputeGameType),
		Proxy:    common.Address{0x55},
	}
}

func newZKExtractor(
	t *testing.T,
	caller *routingZKCaller,
	parent ParentGameStatusFetcher,
	agreement *recordingCommonEnricher,
) (*Extractor, *zkGameFetcher) {
	fetcher := &zkGameFetcher{}
	return NewExtractor(
		testlog.Logger(t, log.LvlDebug),
		clock.NewDeterministicClock(time.Unix(1234, 0)),
		func(context.Context, gameTypes.GameMetadata) (GameCaller, error) {
			return caller, nil
		},
		fetcher.Fetch,
		parent,
		nil,
		1,
		nil,
		nil,
		agreement,
	), fetcher
}
