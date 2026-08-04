package extract

import (
	"context"
	"errors"
	"math"
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

func TestExtractorZKSnapshotValidation(t *testing.T) {
	t.Run("root game", func(t *testing.T) {
		caller := validZKCaller()
		extractor, trace := newZKExtractor(t, caller, parentStatus(gameTypes.GameStatusDefenderWon), &testZKAgreement{})
		blockHash := common.Hash{0xaa}

		game, err := extractor.enrichGame(t.Context(), blockHash, zkMetadata())
		require.NoError(t, err)
		zk := game.(*monTypes.ZKGameData)
		require.Equal(t, caller.metadata.L1Head, zk.L1Head)
		require.Equal(t, uint64(100), zk.L1HeadNum)
		require.Equal(t, caller.metadata.L2SequenceNum, zk.L2SequenceNumber)
		require.Equal(t, caller.challenger.Deadline, zk.Deadline)
		require.Equal(t, caller.anchorStateRegistry, zk.AnchorStateRegistry)
		require.Equal(t, uint32(math.MaxUint32), zk.ParentIndex)
		require.Nil(t, zk.ParentStatus)
		require.Equal(t, []string{"metadata", "l1-head", "agreement", "challenger", "anchor"}, *trace)
		require.Equal(t, []rpcblock.Block{
			rpcblock.ByHash(blockHash), rpcblock.ByHash(blockHash), rpcblock.ByHash(blockHash),
		}, caller.blocks)
	})

	t.Run("child game pins parent lookup", func(t *testing.T) {
		caller := validZKCaller()
		caller.challenger.ParentIndex = 0
		var requestedIndex uint64
		var requestedBlock rpcblock.Block
		extractor, trace := newZKExtractor(t, caller, func(_ context.Context, index uint64, block rpcblock.Block) (gameTypes.GameStatus, error) {
			requestedIndex = index
			requestedBlock = block
			return gameTypes.GameStatusDefenderWon, nil
		}, &testZKAgreement{})
		blockHash := common.Hash{0xbb}

		game, err := extractor.enrichGame(t.Context(), blockHash, zkMetadata())
		require.NoError(t, err)
		zk := game.(*monTypes.ZKGameData)
		require.Equal(t, gameTypes.GameStatusDefenderWon, *zk.ParentStatus)
		require.Zero(t, requestedIndex)
		require.Equal(t, rpcblock.ByHash(blockHash), requestedBlock)
		require.Equal(t, []string{"metadata", "l1-head", "agreement", "challenger", "anchor", "parent"}, *trace)
	})

	tests := []struct {
		name      string
		configure func(*testZKCaller)
		parent    ParentGameStatusFetcher
		wantErr   string
	}{
		{name: "root mismatch", configure: func(c *testZKCaller) { c.challenger.ProposedRoot = common.Hash{0xee} }, wantErr: "inconsistent ZK root claim"},
		{name: "sequence mismatch", configure: func(c *testZKCaller) { c.challenger.L2SequenceNumber++ }, wantErr: "inconsistent ZK sequence number"},
		{name: "unknown proposal status", configure: func(c *testZKCaller) { c.challenger.ProposalStatus = contracts.ProposalStatus(255) }, wantErr: "invalid proposal status"},
		{name: "resolved proposal with live global status", configure: func(c *testZKCaller) { c.challenger.ProposalStatus = contracts.ProposalStatusResolved }, wantErr: "inconsistent ZK global status"},
		{name: "live proposal with terminal global status", configure: func(c *testZKCaller) { c.metadata.Status = gameTypes.GameStatusDefenderWon }, wantErr: "inconsistent ZK global status"},
		{name: "invalid participants", configure: func(c *testZKCaller) { c.challenger.Challenger = common.Address{0x01} }, wantErr: "invalid ZK participants"},
		{name: "anchor read", configure: func(c *testZKCaller) { c.anchorErr = errors.New("anchor unavailable") }, wantErr: "failed to fetch ZK anchor state registry"},
		{name: "zero anchor", configure: func(c *testZKCaller) { c.anchorStateRegistry = common.Address{} }, wantErr: "anchor state registry is zero"},
		{
			name: "parent read",
			configure: func(c *testZKCaller) {
				c.challenger.ParentIndex = 7
			},
			parent: func(context.Context, uint64, rpcblock.Block) (gameTypes.GameStatus, error) {
				return 0, errors.New("parent unavailable")
			},
			wantErr: "failed to fetch ZK parent game 7 status",
		},
		{
			name: "terminal child with live parent",
			configure: func(c *testZKCaller) {
				c.metadata.Status = gameTypes.GameStatusDefenderWon
				c.challenger.ProposalStatus = contracts.ProposalStatusResolved
				c.challenger.ParentIndex = 7
			},
			parent:  parentStatus(gameTypes.GameStatusInProgress),
			wantErr: "terminal ZK child has in-progress parent 7",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caller := validZKCaller()
			test.configure(caller)
			parent := test.parent
			if parent == nil {
				parent = parentStatus(gameTypes.GameStatusDefenderWon)
			}
			extractor, _ := newZKExtractor(t, caller, parent, &testZKAgreement{})

			game, err := extractor.enrichGame(t.Context(), common.Hash{0xaa}, zkMetadata())
			require.Nil(t, game)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestExtractorRejectsZKCallerWithoutCapabilities(t *testing.T) {
	caller := &anchorOnlyCaller{}
	extractor := NewExtractor(
		testlog.Logger(t, log.LvlDebug),
		clock.NewDeterministicClock(time.Unix(1234, 0)),
		func(context.Context, gameTypes.GameMetadata) (GameCaller, error) { return caller, nil },
		func(context.Context, common.Hash, uint64) ([]gameTypes.GameMetadata, error) { return nil, nil },
		parentStatus(gameTypes.GameStatusDefenderWon),
		nil,
		1,
		nil,
		nil,
		&testZKAgreement{},
	)

	game, err := extractor.enrichGame(t.Context(), common.Hash{0xaa}, zkMetadata())
	require.Nil(t, game)
	require.ErrorContains(t, err, "does not support ZK game extraction")
}

func TestCommonEnrichersSkipZKOwnedReads(t *testing.T) {
	caller := &anchorOnlyCaller{}
	game := &monTypes.CommonGameData{GameMetadata: gameTypes.GameMetadata{GameType: uint32(gameTypes.ZKDisputeGameType)}}
	logger := testlog.Logger(t, log.LvlDebug)

	require.NoError(t, NewAnchorStateRegistryEnricher(logger).Enrich(t.Context(), rpcblock.Latest, caller, game))
	require.Zero(t, caller.calls)
	require.NoError(t, NewSuperAgreementEnricher(logger, &stubOutputMetrics{}, nil, clock.SystemClock).Enrich(t.Context(), rpcblock.Latest, caller, game))
}

func TestExtractorZKLagAndCache(t *testing.T) {
	caller := validZKCaller()
	agreement := &testZKAgreement{err: gameTypes.ErrNotInSync}
	extractor, trace := newZKExtractor(t, caller, parentStatus(gameTypes.GameStatusDefenderWon), agreement)

	games, ignored, failed, err := extractor.Extract(t.Context(), common.Hash{0xaa}, 0)
	require.NoError(t, err)
	require.Empty(t, games)
	require.Zero(t, ignored)
	require.Zero(t, failed)
	require.Equal(t, []string{"metadata", "l1-head", "agreement"}, *trace)

	*trace = nil
	agreement.err = nil
	agreement.action = func(game *monTypes.ZKGameData) {
		game.AgreeWithClaim = true
		game.ExpectedRootClaim = game.RootClaim
	}
	games, _, failed, err = extractor.Extract(t.Context(), common.Hash{0xbb}, 0)
	require.NoError(t, err)
	require.Zero(t, failed)
	require.Len(t, games, 1)
	snapshot := games[0]
	original := snapshot.(*monTypes.ZKGameData)
	originalExpectedRoot := original.ExpectedRootClaim

	*trace = nil
	agreement.action = func(game *monTypes.ZKGameData) {
		game.AgreeWithClaim = false
		game.ExpectedRootClaim = common.Hash{0xff}
		game.NodeEndpointOutOfSyncCount = 3
	}
	caller.anchorErr = errors.New("anchor unavailable")
	games, _, failed, err = extractor.Extract(t.Context(), common.Hash{0xbc}, 0)
	require.NoError(t, err)
	require.Equal(t, 1, failed)
	require.Same(t, snapshot, games[0])
	require.True(t, original.AgreeWithClaim)
	require.Equal(t, originalExpectedRoot, original.ExpectedRootClaim)
	require.Zero(t, original.NodeEndpointOutOfSyncCount)
	require.Equal(t, []string{"metadata", "l1-head", "agreement", "challenger", "anchor"}, *trace)
	caller.anchorErr = nil

	*trace = nil
	agreement.err = gameTypes.ErrNotInSync
	games, _, failed, err = extractor.Extract(t.Context(), common.Hash{0xcc}, 0)
	require.NoError(t, err)
	require.Zero(t, failed)
	require.Equal(t, []monTypes.EnrichedGame{snapshot}, games)
	require.Equal(t, []string{"metadata", "l1-head", "agreement"}, *trace)

	*trace = nil
	agreement.err = errors.New("root source unavailable")
	games, _, failed, err = extractor.Extract(t.Context(), common.Hash{0xdd}, 0)
	require.NoError(t, err)
	require.Equal(t, 1, failed)
	require.Equal(t, []monTypes.EnrichedGame{snapshot}, games)
	require.Same(t, snapshot, games[0])
}

func TestValidateZKStatus(t *testing.T) {
	tests := []struct {
		name     string
		global   gameTypes.GameStatus
		proposal contracts.ProposalStatus
		valid    bool
	}{
		{name: "unchallenged live", global: gameTypes.GameStatusInProgress, proposal: contracts.ProposalStatusUnchallenged, valid: true},
		{name: "challenged live", global: gameTypes.GameStatusInProgress, proposal: contracts.ProposalStatusChallenged, valid: true},
		{name: "unchallenged proof live", global: gameTypes.GameStatusInProgress, proposal: contracts.ProposalStatusUnchallengedAndValidProofProvided, valid: true},
		{name: "challenged proof live", global: gameTypes.GameStatusInProgress, proposal: contracts.ProposalStatusChallengedAndValidProofProvided, valid: true},
		{name: "resolved defender", global: gameTypes.GameStatusDefenderWon, proposal: contracts.ProposalStatusResolved, valid: true},
		{name: "resolved challenger", global: gameTypes.GameStatusChallengerWon, proposal: contracts.ProposalStatusResolved, valid: true},
		{name: "resolved live", global: gameTypes.GameStatusInProgress, proposal: contracts.ProposalStatusResolved},
		{name: "unresolved terminal", global: gameTypes.GameStatusDefenderWon, proposal: contracts.ProposalStatusUnchallenged},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateZKStatus(test.global, test.proposal)
			require.Equal(t, test.valid, err == nil)
		})
	}
}

func TestValidateZKParticipants(t *testing.T) {
	a := common.Address{0x01}
	tests := []struct {
		name       string
		status     contracts.ProposalStatus
		challenger common.Address
		prover     common.Address
		valid      bool
	}{
		{name: "unchallenged", status: contracts.ProposalStatusUnchallenged, valid: true},
		{name: "challenged", status: contracts.ProposalStatusChallenged, challenger: a, valid: true},
		{name: "unchallenged proof", status: contracts.ProposalStatusUnchallengedAndValidProofProvided, prover: a, valid: true},
		{name: "challenged proof", status: contracts.ProposalStatusChallengedAndValidProofProvided, challenger: a, prover: a, valid: true},
		{name: "resolved empty", status: contracts.ProposalStatusResolved, valid: true},
		{name: "resolved challenged", status: contracts.ProposalStatusResolved, challenger: a, valid: true},
		{name: "resolved proved", status: contracts.ProposalStatusResolved, prover: a, valid: true},
		{name: "resolved challenged and proved", status: contracts.ProposalStatusResolved, challenger: a, prover: a, valid: true},
		{name: "unchallenged with challenger", status: contracts.ProposalStatusUnchallenged, challenger: a},
		{name: "challenged without challenger", status: contracts.ProposalStatusChallenged},
		{name: "challenged with prover", status: contracts.ProposalStatusChallenged, challenger: a, prover: a},
		{name: "unchallenged proof without prover", status: contracts.ProposalStatusUnchallengedAndValidProofProvided},
		{name: "unchallenged proof with challenger", status: contracts.ProposalStatusUnchallengedAndValidProofProvided, challenger: a, prover: a},
		{name: "challenged proof without challenger", status: contracts.ProposalStatusChallengedAndValidProofProvided, prover: a},
		{name: "challenged proof without prover", status: contracts.ProposalStatusChallengedAndValidProofProvided, challenger: a},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateZKParticipants(test.status, test.challenger, test.prover)
			require.Equal(t, test.valid, err == nil)
		})
	}
}

type testZKCaller struct {
	metadata            contracts.GenericGameMetadata
	challenger          contracts.ChallengerMetadata
	anchorStateRegistry common.Address
	anchorErr           error
	trace               *[]string
	blocks              []rpcblock.Block
}

func validZKCaller() *testZKCaller {
	return &testZKCaller{
		metadata: contracts.GenericGameMetadata{
			L1Head:        common.Hash{0x11},
			L2SequenceNum: 99,
			ProposedRoot:  common.Hash{0x22},
			Status:        gameTypes.GameStatusInProgress,
		},
		challenger: contracts.ChallengerMetadata{
			ParentIndex:      math.MaxUint32,
			ProposalStatus:   contracts.ProposalStatusUnchallenged,
			ProposedRoot:     common.Hash{0x22},
			L2SequenceNumber: 99,
			Deadline:         time.Unix(5678, 0),
		},
		anchorStateRegistry: common.Address{0xab},
	}
}

func (c *testZKCaller) GetMetadata(_ context.Context, block rpcblock.Block) (contracts.GenericGameMetadata, error) {
	*c.trace = append(*c.trace, "metadata")
	c.blocks = append(c.blocks, block)
	return c.metadata, nil
}

func (c *testZKCaller) GetChallengerMetadata(_ context.Context, block rpcblock.Block) (contracts.ChallengerMetadata, error) {
	*c.trace = append(*c.trace, "challenger")
	c.blocks = append(c.blocks, block)
	return c.challenger, nil
}

func (c *testZKCaller) GetAnchorStateRegistry(_ context.Context, block rpcblock.Block) (common.Address, error) {
	*c.trace = append(*c.trace, "anchor")
	c.blocks = append(c.blocks, block)
	return c.anchorStateRegistry, c.anchorErr
}

type testZKAgreement struct {
	trace  *[]string
	err    error
	action func(*monTypes.ZKGameData)
}

func (e *testZKAgreement) Enrich(_ context.Context, _ rpcblock.Block, _ ZKGameCaller, game *monTypes.ZKGameData) error {
	*e.trace = append(*e.trace, "agreement")
	if e.err != nil {
		return e.err
	}
	if e.action != nil {
		e.action(game)
	}
	return nil
}

type testCommonEnricher struct {
	trace *[]string
}

func (e *testCommonEnricher) Enrich(_ context.Context, _ rpcblock.Block, _ GameCaller, game *monTypes.CommonGameData) error {
	*e.trace = append(*e.trace, "l1-head")
	game.L1HeadNum = 100
	return nil
}

type anchorOnlyCaller struct {
	calls int
}

func (c *anchorOnlyCaller) GetAnchorStateRegistry(context.Context, rpcblock.Block) (common.Address, error) {
	c.calls++
	return common.Address{0xab}, nil
}

func newZKExtractor(t *testing.T, caller *testZKCaller, parent ParentGameStatusFetcher, agreement *testZKAgreement) (*Extractor, *[]string) {
	t.Helper()
	trace := new([]string)
	caller.trace = trace
	agreement.trace = trace
	tracedParent := func(ctx context.Context, index uint64, block rpcblock.Block) (gameTypes.GameStatus, error) {
		*trace = append(*trace, "parent")
		return parent(ctx, index, block)
	}
	return NewExtractor(
		testlog.Logger(t, log.LvlDebug),
		clock.NewDeterministicClock(time.Unix(1234, 0)),
		func(context.Context, gameTypes.GameMetadata) (GameCaller, error) { return caller, nil },
		func(context.Context, common.Hash, uint64) ([]gameTypes.GameMetadata, error) {
			return []gameTypes.GameMetadata{zkMetadata()}, nil
		},
		tracedParent,
		nil,
		1,
		[]CommonEnricher{&testCommonEnricher{trace: trace}},
		nil,
		agreement,
	), trace
}

func zkMetadata() gameTypes.GameMetadata {
	return gameTypes.GameMetadata{Index: 9, GameType: uint32(gameTypes.ZKDisputeGameType), Proxy: common.Address{0x99}, Timestamp: 123}
}

func parentStatus(status gameTypes.GameStatus) ParentGameStatusFetcher {
	return func(context.Context, uint64, rpcblock.Block) (gameTypes.GameStatus, error) { return status, nil }
}
