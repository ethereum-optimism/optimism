package extract

import (
	"context"
	"errors"
	"math"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
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
	commonEnricher := &recordingCommonEnricher{action: func(game *monTypes.CommonGameData) error {
		game.L1HeadNum = 123
		return nil
	}}
	faultEnricher := &recordingFaultEnricher{}
	bondEnricher := &recordingBondEnricher{}
	zkAgreement := &recordingZKEnricher{action: func(game *monTypes.ZKGameData) error {
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
	extractor, err := NewExtractor(
		testlog.Logger(t, log.LvlDebug),
		clock.NewDeterministicClock(time.Unix(1234, 0)),
		creator,
		func(context.Context, common.Hash, uint64) ([]gameTypes.GameMetadata, error) {
			return games, nil
		},
		func(context.Context, uint64, rpcblock.Block) (gameTypes.GameStatus, error) {
			return gameTypes.GameStatusDefenderWon, nil
		},
		nil,
		5,
		[]CommonEnricher{commonEnricher},
		[]FaultEnricher{faultEnricher},
		bondEnricher,
		zkAgreement,
	)
	require.NoError(t, err)

	extracted, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{0xaa}, 0)
	require.NoError(t, err)
	require.Zero(t, ignored)
	require.Zero(t, failed)
	require.Len(t, extracted, len(gameTypes.SupportedLifecycleGameTypes))
	require.Equal(t, len(gameTypes.SupportedLifecycleGameTypes), commonEnricher.Calls())
	require.Equal(t, len(gameTypes.SupportedLifecycleGameTypes)-2, faultEnricher.Calls())
	require.Equal(t, len(gameTypes.SupportedLifecycleGameTypes)-1, bondEnricher.Calls())
	for _, l1Head := range faultEnricher.L1Heads() {
		require.Equal(t, uint64(123), l1Head)
	}
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
			}, &recordingZKEnricher{})
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
		}, &recordingZKEnricher{})
		game, err := extractor.enrichGame(context.Background(), common.Hash{0xaa}, zkMetadata())
		require.NoError(t, err)
		zkGame := game.(*monTypes.ZKGameData)
		require.Nil(t, zkGame.ParentStatus)
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
		}, &recordingZKEnricher{})
		blockHash := common.Hash{0xaa}
		game, err := extractor.enrichGame(context.Background(), blockHash, zkMetadata())
		require.NoError(t, err)
		zkGame := game.(*monTypes.ZKGameData)
		require.Equal(t, uint32(0), zkGame.ParentIndex)
		require.Equal(t, gameTypes.GameStatusDefenderWon, *zkGame.ParentStatus)
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
		}, &recordingZKEnricher{})
		game, err := extractor.enrichGame(context.Background(), common.Hash{0xaa}, zkMetadata())
		require.NoError(t, err)
		require.Equal(t, uint64(1234), requestedIndex)
		require.Equal(t, gameTypes.GameStatusChallengerWon, *game.(*monTypes.ZKGameData).ParentStatus)
	})
}

func TestExtractorZKCacheFallbackIsImmutable(t *testing.T) {
	for _, test := range []struct {
		name string
		fail func(*parentStatusSource, *recordingZKEnricher)
	}{
		{
			name: "parent refresh failure",
			fail: func(parent *parentStatusSource, _ *recordingZKEnricher) {
				parent.err = errors.New("parent unavailable")
			},
		},
		{
			name: "root source refresh failure",
			fail: func(_ *parentStatusSource, agreement *recordingZKEnricher) {
				agreement.err = errors.New("root source unavailable")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			caller := validZKCaller()
			caller.challengerMetadata.ParentIndex = 7
			parent := &parentStatusSource{status: gameTypes.GameStatusDefenderWon}
			agreement := &recordingZKEnricher{action: func(game *monTypes.ZKGameData) error {
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
			require.Equal(t, gameTypes.GameStatusDefenderWon, *snapshot.ParentStatus)
			require.Equal(t, snapshot.RootClaim, snapshot.ExpectedRootClaim)

			test.fail(parent, agreement)
			fetcher.blockHash = common.Hash{0xbb}
			second, _, failed, err := extractor.Extract(context.Background(), fetcher.blockHash, 0)
			require.NoError(t, err)
			require.Equal(t, 1, failed)
			require.Len(t, second, 1)
			require.Same(t, snapshot, second[0])
			require.Equal(t, gameTypes.GameStatusDefenderWon, *snapshot.ParentStatus)
			require.Equal(t, snapshot.RootClaim, snapshot.ExpectedRootClaim)
		})
	}
}

func TestExtractorSkipsOutOfSyncZKWithoutFailure(t *testing.T) {
	caller := validZKCaller()
	agreement := &recordingZKEnricher{err: gameTypes.ErrNotInSync}
	extractor, fetcher := newZKExtractor(t, caller, func(context.Context, uint64, rpcblock.Block) (gameTypes.GameStatus, error) {
		return gameTypes.GameStatusDefenderWon, nil
	}, agreement)

	games, ignored, failed, err := extractor.Extract(context.Background(), common.Hash{0xaa}, 0)
	require.NoError(t, err)
	require.Empty(t, games)
	require.Zero(t, ignored)
	require.Zero(t, failed)
	require.Zero(t, caller.bondMetadataCalls)

	agreement.err = nil
	agreement.action = func(game *monTypes.ZKGameData) error {
		game.AgreeWithClaim = true
		game.ExpectedRootClaim = game.RootClaim
		return nil
	}
	games, _, failed, err = extractor.Extract(context.Background(), common.Hash{0xbb}, 0)
	require.NoError(t, err)
	require.Zero(t, failed)
	require.Len(t, games, 1)
	require.Equal(t, 1, caller.bondMetadataCalls)
	snapshot := games[0]

	agreement.err = gameTypes.ErrNotInSync
	fetcher.blockHash = common.Hash{0xcc}
	games, _, failed, err = extractor.Extract(context.Background(), fetcher.blockHash, 0)
	require.NoError(t, err)
	require.Zero(t, failed)
	require.Equal(t, []monTypes.EnrichedGame{snapshot}, games)
	require.Equal(t, 1, caller.bondMetadataCalls)
}

type routingZKCaller struct {
	metadata                contracts.GenericGameMetadata
	challengerMetadata      contracts.ChallengerMetadata
	bondMetadata            contracts.ZKBondMetadata
	metadataCalls           int
	challengerMetadataCalls int
	bondMetadataCalls       int
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
			ParentIndex:    math.MaxUint32,
			ProposalStatus: contracts.ProposalStatusUnchallenged,
			ProposedRoot:   common.Hash{0x22},
		},
		bondMetadata: contracts.ZKBondMetadata{
			GameCreator:    common.Address{0xc1},
			TotalBonds:     big.NewInt(100),
			ChallengerBond: big.NewInt(30),
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

func (c *routingZKCaller) GetBondMetadata(context.Context, rpcblock.Block) (contracts.ZKBondMetadata, error) {
	c.bondMetadataCalls++
	return c.bondMetadata, nil
}

func (*routingZKCaller) GetCredits(context.Context, rpcblock.Block, ...common.Address) ([]*big.Int, error) {
	return nil, nil
}

func (*routingZKCaller) GetBondDistributionMode(context.Context, rpcblock.Block) (faultTypes.BondDistributionMode, error) {
	return faultTypes.UndecidedDistributionMode, nil
}

func (*routingZKCaller) GetWithdrawals(context.Context, rpcblock.Block, ...common.Address) ([]*contracts.WithdrawalRequest, error) {
	return nil, nil
}

func (*routingZKCaller) GetBalanceAndDelay(context.Context, rpcblock.Block) (*big.Int, time.Duration, common.Address, error) {
	return big.NewInt(0), 0, common.Address{}, nil
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

type recordingZKEnricher struct {
	mu     sync.Mutex
	calls  int
	err    error
	action func(*monTypes.ZKGameData) error
}

type recordingBondEnricher struct {
	mu    sync.Mutex
	calls int
}

func (e *recordingBondEnricher) Enrich(context.Context, rpcblock.Block, BondGameCaller, monTypes.BondedGame) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	return nil
}

func (e *recordingBondEnricher) Calls() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

func (e *recordingZKEnricher) Enrich(_ context.Context, _ rpcblock.Block, _ ZKGameCaller, game *monTypes.ZKGameData) error {
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

func (e *recordingZKEnricher) Calls() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
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
	mu      sync.Mutex
	calls   int
	l1Heads []uint64
}

func (e *recordingFaultEnricher) Enrich(_ context.Context, _ rpcblock.Block, _ FaultGameCaller, game *monTypes.FaultGameData) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	e.l1Heads = append(e.l1Heads, game.L1HeadNum)
	return nil
}

func (e *recordingFaultEnricher) Calls() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.calls
}

func (e *recordingFaultEnricher) L1Heads() []uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]uint64(nil), e.l1Heads...)
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
	agreement *recordingZKEnricher,
) (*Extractor, *zkGameFetcher) {
	fetcher := &zkGameFetcher{}
	extractor, err := NewExtractor(
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
		&recordingBondEnricher{},
		agreement,
	)
	require.NoError(t, err)
	return extractor, fetcher
}
