package proofs

import (
	"context"
	"encoding/binary"
	"math"
	"math/big"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	challengerConfig "github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/prestates"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/super"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txintent/contractio"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

type DisputeGameFactory struct {
	t             devtest.T
	require       *require.Assertions
	log           log.Logger
	l1Network     *dsl.L1Network
	ethClient     apis.EthClient
	dgf           *bindings.DisputeGameFactory
	addr          common.Address
	l2CL          *dsl.L2CLNode
	l2EL          *dsl.L2ELNode
	superNode     dsl.SuperRootSource
	l1Proposer    *dsl.EOA
	gameHelper    *GameHelper
	challengerCfg *challengerConfig.Config

	honestTraces map[common.Address]challengerTypes.TraceAccessor
}

func NewDisputeGameFactory(
	t devtest.T,
	l1Network *dsl.L1Network,
	ethClient apis.EthClient,
	dgfAddr common.Address,
	l2CL *dsl.L2CLNode,
	l2EL *dsl.L2ELNode,
	superNode dsl.SuperRootSource,
	l1Proposer *dsl.EOA,
	challengerCfg *challengerConfig.Config,
) *DisputeGameFactory {
	dgf := bindings.NewDisputeGameFactory(bindings.WithClient(ethClient), bindings.WithTo(dgfAddr), bindings.WithTest(t))

	return &DisputeGameFactory{
		t:             t,
		require:       require.New(t),
		log:           t.Logger(),
		l1Network:     l1Network,
		dgf:           dgf,
		addr:          dgfAddr,
		l2CL:          l2CL,
		l2EL:          l2EL,
		superNode:     superNode,
		l1Proposer:    l1Proposer,
		ethClient:     ethClient,
		challengerCfg: challengerCfg,

		honestTraces: make(map[common.Address]challengerTypes.TraceAccessor),
	}
}

type GameCfg struct {
	allowFuture         bool
	allowUnsafe         bool
	l2SequenceNumber    uint64
	l2SequenceNumberSet bool
	rootClaimSet        bool
	rootClaim           common.Hash
	superOutputRoots    []eth.Bytes32
	zkParentIndex       *uint32
}
type GameOpt interface {
	Apply(cfg *GameCfg)
}
type gameOptFn func(c *GameCfg)

func (g gameOptFn) Apply(cfg *GameCfg) {
	g(cfg)
}

func WithUnsafeProposal() GameOpt {
	return gameOptFn(func(c *GameCfg) {
		c.allowUnsafe = true
	})
}

func WithFutureProposal() GameOpt {
	return gameOptFn(func(c *GameCfg) {
		c.allowFuture = true
	})
}

func WithRootClaim(claim common.Hash) GameOpt {
	return gameOptFn(func(c *GameCfg) {
		c.rootClaim = claim
		c.rootClaimSet = true
	})
}

func WithL2SequenceNumber(seqNum uint64) GameOpt {
	return gameOptFn(func(c *GameCfg) {
		c.l2SequenceNumber = seqNum
		c.l2SequenceNumberSet = true
	})
}

// WithSuperRootFrom sets the output roots to use in a super root game.
// The length of outputRoots must match the number of chains in the super root.
func WithSuperRootFrom(outputRoots ...eth.Bytes32) GameOpt {
	return gameOptFn(func(c *GameCfg) {
		c.superOutputRoots = outputRoots
	})
}

// WithZKParent links a ZK game to an earlier ZK game in the factory. Without
// this option, StartZKGame uses the uint32 max sentinel for a root game.
func WithZKParent(parentIndex uint32) GameOpt {
	return gameOptFn(func(c *GameCfg) {
		c.zkParentIndex = &parentIndex
	})
}

func NewGameCfg(opts ...GameOpt) *GameCfg {
	cfg := &GameCfg{}
	for _, opt := range opts {
		opt.Apply(cfg)
	}
	return cfg
}

func (f *DisputeGameFactory) Address() common.Address {
	return f.addr
}

func (f *DisputeGameFactory) getGameHelper(eoa *dsl.EOA) *GameHelper {
	if f.gameHelper != nil {
		return f.gameHelper
	}
	gs := DeployGameHelper(f.t, eoa, f.honestTraceForGame)
	f.gameHelper = gs
	return gs
}

func (f *DisputeGameFactory) GameCount() int64 {
	return contract.Read(f.dgf.GameCount()).Int64()
}

func (f *DisputeGameFactory) GameAtIndex(idx int64) *FaultDisputeGame {
	gameInfo := contract.Read(f.dgf.GameAtIndex(big.NewInt(idx)))
	game := bindings.NewFaultDisputeGame(bindings.WithClient(f.ethClient), bindings.WithTo(gameInfo.Proxy), bindings.WithTest(f.t))
	return NewFaultDisputeGame(f.t, f.require, gameInfo.Proxy, f.getGameHelper, f.honestTraceForGame, game)
}

func (f *DisputeGameFactory) SuperGameAtIndex(idx int64) *SuperFaultDisputeGame {
	gameInfo := contract.Read(f.dgf.GameAtIndex(big.NewInt(idx)))
	gameType := gameTypes.GameType(gameInfo.GameType)
	f.require.Truef(
		gameType == gameTypes.SuperPermissionedGameType ||
			gameType == gameTypes.SuperCannonKonaGameType,
		"game at index %d is not a supported super game: %v",
		idx,
		gameType,
	)
	game := bindings.NewFaultDisputeGame(bindings.WithClient(f.ethClient), bindings.WithTo(gameInfo.Proxy), bindings.WithTest(f.t))
	return NewSuperFaultDisputeGame(f.t, f.require, gameInfo.Proxy, f.getGameHelper, f.honestTraceForGame, game)
}

func (f *DisputeGameFactory) ZKGameAtIndex(idx uint32) *ZKGame {
	gameInfo := contract.Read(f.dgf.GameAtIndex(big.NewInt(int64(idx))))
	f.require.Equalf(gameTypes.ZKDisputeGameType, gameTypes.GameType(gameInfo.GameType),
		"game at index %d is not a ZK dispute game", idx)
	return newZKGame(f.t, f.require, f.ethClient, gameInfo.Proxy).withFactoryIndex(idx)
}

func (f *DisputeGameFactory) GameImpl(gameType gameTypes.GameType) *FaultDisputeGame {
	implAddr := contract.Read(f.dgf.GameImpls(uint32(gameType)))
	game := bindings.NewFaultDisputeGame(bindings.WithClient(f.ethClient), bindings.WithTo(implAddr), bindings.WithTest(f.t))
	return NewFaultDisputeGame(f.t, f.require, implAddr, f.getGameHelper, f.honestTraceForGame, game)
}

func (f *DisputeGameFactory) VerifyGameImplPresent(gameType gameTypes.GameType) {
	implAddr := contract.Read(f.dgf.GameImpls(uint32(gameType)))
	f.require.NotEqualf(common.Address{}, implAddr,
		"expected DisputeGameFactory to have an implementation for %s (%d)", gameType, uint32(gameType))
}

func (f *DisputeGameFactory) VerifyGameImplAbsent(gameType gameTypes.GameType) {
	implAddr := contract.Read(f.dgf.GameImpls(uint32(gameType)))
	f.require.Equalf(common.Address{}, implAddr,
		"expected DisputeGameFactory to have no implementation for %s (%d), got %s", gameType, uint32(gameType), implAddr)
}

func (f *DisputeGameFactory) GameArgs(gameType gameTypes.GameType) []byte {
	return contract.Read(f.dgf.GameArgs(uint32(gameType)))
}

func (f *DisputeGameFactory) WaitForGame() *FaultDisputeGame {
	initialCount := f.GameCount()
	timedCtx, cancel := context.WithTimeout(f.t.Ctx(), 10*time.Minute)
	defer cancel()
	var lastReadErr error
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		count, readErr := contractio.Read(f.dgf.GameCount(), timedCtx)
		lastReadErr = readErr
		if readErr != nil {
			f.log.Debug("Game count unavailable while waiting for new game", "current", initialCount, "err", readErr)
			return false, nil
		}
		gameCount := count.Int64()
		f.t.Logf("waiting for new game. current=%d new=%d", initialCount, gameCount)
		return gameCount > initialCount, nil
	})
	f.require.NoErrorf(err, "dispute game factory did not create a new game beyond count %d; last read error: %v", initialCount, lastReadErr)
	return f.GameAtIndex(initialCount)
}

// WaitForZKGameAtIndex waits until the factory holds a ZK dispute game at the
// given index, then returns that game.
func (f *DisputeGameFactory) WaitForZKGameAtIndex(idx int64) *ZKGame {
	f.require.GreaterOrEqual(idx, int64(0), "game index must not be negative")
	f.require.Less(idx, int64(math.MaxUint32), "game index must fit in uint32")
	timedCtx, cancel := context.WithTimeout(f.t.Ctx(), 2*time.Minute)
	defer cancel()
	var lastReadErr error
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		count, readErr := contractio.Read(f.dgf.GameCount(), timedCtx)
		lastReadErr = readErr
		if readErr != nil {
			f.log.Debug("Game count unavailable while waiting for ZK game", "index", idx, "err", readErr)
			return false, nil
		}
		f.log.Info("Waiting for ZK game", "index", idx, "count", count)
		if count.Int64() <= idx {
			return false, nil
		}
		gameInfo, readErr := contractio.Read(f.dgf.GameAtIndex(big.NewInt(idx)), timedCtx)
		lastReadErr = readErr
		if readErr != nil {
			f.log.Debug("Game info unavailable while waiting for ZK game", "index", idx, "err", readErr)
			return false, nil
		}
		return gameTypes.GameType(gameInfo.GameType) == gameTypes.ZKDisputeGameType, nil
	})
	f.require.NoErrorf(err, "dispute game factory did not have a ZK game at index %d; last read error: %v", idx, lastReadErr)
	return f.ZKGameAtIndex(uint32(idx))
}

func (f *DisputeGameFactory) StartSuperCannonKonaGame(eoa *dsl.EOA, opts ...GameOpt) *SuperFaultDisputeGame {
	f.require.NotNil(f.superNode, "super node is required to start super games")

	return f.startSuperGameOfType(eoa, gameTypes.SuperCannonKonaGameType, opts...)
}

func (f *DisputeGameFactory) StartSuperPermissionedGame(opts ...GameOpt) *SuperFaultDisputeGame {
	f.require.NotNil(f.l1Proposer, "proposer EOA is required to start a super permissioned game")
	f.require.NotNil(f.superNode, "super node is required to start super games")
	return f.startSuperGameOfType(f.l1Proposer, gameTypes.SuperPermissionedGameType, opts...)
}

func (f *DisputeGameFactory) StartZKGame(eoa *dsl.EOA, opts ...GameOpt) *ZKGame {
	f.require.NotNil(f.superNode, "super node is required to start ZK games")
	cfg := NewGameCfg(opts...)
	f.require.False(cfg.rootClaimSet, "ZK root claim is derived from the super-root proof and cannot be overridden")

	timestamp := cfg.l2SequenceNumber
	if !cfg.l2SequenceNumberSet {
		minSequence := f.zkAnchorSequenceNumber()
		if cfg.zkParentIndex != nil {
			minSequence = f.ZKGameAtIndex(*cfg.zkParentIndex).L2SequenceNumber()
		}
		timestamp = f.waitForSafeSuperRootAfter(minSequence)
	}

	superRootProof := f.createSuperGameExtraData(timestamp, cfg)
	parentIndex := uint32(math.MaxUint32)
	if cfg.zkParentIndex != nil {
		parentIndex = *cfg.zkParentIndex
	}
	extraData := make([]byte, 4+len(superRootProof))
	binary.BigEndian.PutUint32(extraData[:4], parentIndex)
	copy(extraData[4:], superRootProof)
	rootClaim := crypto.Keccak256Hash(superRootProof)

	gameIndex := f.GameCount()
	f.require.GreaterOrEqual(gameIndex, int64(0), "ZK game index must not be negative")
	f.require.LessOrEqual(gameIndex, int64(math.MaxUint32), "ZK game index must fit in uint32")
	_, addr := f.createNewGame(eoa, gameTypes.ZKDisputeGameType, rootClaim, extraData)
	return newZKGame(f.t, f.require, f.ethClient, addr).withFactoryIndex(uint32(gameIndex))
}

// WaitForSafeSuperRootAfter returns a verified super-root timestamp and a copy
// of its per-chain output roots. Tests can mutate the copy to model a faulty
// proposer without changing supernode-owned response data.
func (f *DisputeGameFactory) WaitForSafeSuperRootAfter(sequence uint64) (uint64, []eth.Bytes32) {
	f.require.NotNil(f.superNode, "super node is required to read safe super roots")
	timestamp := f.waitForSafeSuperRootAfter(sequence)

	resp := f.superNode.SuperRootAtTimestamp(timestamp)
	f.require.NotNil(resp.Data, "super root data must be present at timestamp %d", timestamp)
	superV1, ok := resp.Data.Super.(*eth.SuperV1)
	f.require.Truef(ok, "unsupported super type %T", resp.Data.Super)
	outputs := make([]eth.Bytes32, len(superV1.Chains))
	for i := range superV1.Chains {
		outputs[i] = superV1.Chains[i].Output
	}
	return timestamp, outputs
}

func (f *DisputeGameFactory) waitForSafeSuperRootAfter(sequence uint64) uint64 {
	var timestamp uint64
	f.t.Require().Eventually(func() bool {
		timestamp = f.safeTimestamp()
		return timestamp > sequence
	}, 2*time.Minute, time.Second, "safe super-root timestamp did not advance beyond the requested sequence")
	return timestamp
}

func (f *DisputeGameFactory) zkAnchorSequenceNumber() uint64 {
	impl := f.ZKGameImpl()
	registry := bindings.NewBindings[bindings.AnchorStateRegistry](
		bindings.WithClient(f.ethClient),
		bindings.WithTo(impl.Args.AnchorStateRegistry),
		bindings.WithTest(f.t),
	)
	anchor := contract.Read(registry.GetAnchorRoot())
	return bigs.Uint64Strict(anchor.L2SequenceNumber)
}

func (f *DisputeGameFactory) startSuperGameOfType(eoa *dsl.EOA, gameType gameTypes.GameType, opts ...GameOpt) *SuperFaultDisputeGame {
	cfg := NewGameCfg(opts...)
	if len(cfg.superOutputRoots) != 0 && cfg.rootClaimSet {
		f.t.Error("cannot set both super output roots and root claim in super game")
		f.t.FailNow()
	}
	timestamp := cfg.l2SequenceNumber
	if !cfg.l2SequenceNumberSet {
		timestamp = f.safeTimestamp()
	}
	extraData := f.createSuperGameExtraData(timestamp, cfg)
	rootClaim := cfg.rootClaim
	if !cfg.rootClaimSet {
		rootClaim = crypto.Keccak256Hash(extraData)
	}
	game, addr := f.createNewGame(eoa, gameType, rootClaim, extraData)

	return NewSuperFaultDisputeGame(f.t, f.require, addr, f.getGameHelper, f.honestTraceForGame, game)
}

func (f *DisputeGameFactory) createSuperGameExtraData(timestamp uint64, cfg *GameCfg) []byte {
	f.require.NotNil(f.superNode, "super node is required create super games")
	// A future proposal commits to a timestamp the node has not reached, so no super root exists there
	// yet. Model it by stamping the current safe super root's dependency set at the requested timestamp.
	queryTimestamp := timestamp
	if cfg.allowFuture {
		queryTimestamp = f.safeTimestamp()
	}
	resp := f.awaitMinVerifiedTimestamp(queryTimestamp)
	f.require.NotNil(resp.Data, "Super root data must be present at timestamp %v", queryTimestamp)
	superV1, ok := resp.Data.Super.(*eth.SuperV1)
	f.require.Truef(ok, "unsupported super type %T", resp.Data.Super)
	superV1.Timestamp = timestamp
	if len(cfg.superOutputRoots) != 0 {
		f.require.Len(cfg.superOutputRoots, len(superV1.Chains), "Super output roots length mismatch")
		for i := range superV1.Chains {
			superV1.Chains[i].Output = cfg.superOutputRoots[i]
		}
	}
	extraData := superV1.Marshal()
	return extraData
}

func (f *DisputeGameFactory) awaitMinVerifiedTimestamp(timestamp uint64) eth.SuperRootAtTimestampResponse {
	ctx, cancel := context.WithTimeout(f.t.Ctx(), 2*time.Minute)
	defer cancel()
	resp, err := awaitMinVerifiedTimestamp(ctx, timestamp, time.Second, f.superNode.QueryAPI().SuperRootAtTimestamp)
	f.require.NoErrorf(err, "super root at timestamp %d was not verified in time", timestamp)
	return resp
}

type superRootAtTimestampFn func(context.Context, uint64) (eth.SuperRootAtTimestampResponse, error)

type superRootAtTimestampReadyFn func(eth.SuperRootAtTimestampResponse) bool

func awaitSuperRootAtTimestamp(
	ctx context.Context,
	timestamp uint64,
	pollInterval time.Duration,
	query superRootAtTimestampFn,
	ready superRootAtTimestampReadyFn,
) (eth.SuperRootAtTimestampResponse, error) {
	var resp eth.SuperRootAtTimestampResponse
	err := wait.For(ctx, pollInterval, func() (bool, error) {
		queried, err := query(ctx, timestamp)
		if err != nil || !ready(queried) {
			return false, nil
		}
		resp = queried
		return true, nil
	})
	return resp, err
}

func awaitMinVerifiedTimestamp(
	ctx context.Context,
	timestamp uint64,
	pollInterval time.Duration,
	query superRootAtTimestampFn,
) (eth.SuperRootAtTimestampResponse, error) {
	return awaitSuperRootAtTimestamp(ctx, timestamp, pollInterval, query, func(resp eth.SuperRootAtTimestampResponse) bool {
		return resp.Data != nil
	})
}

func (f *DisputeGameFactory) StartCannonKonaGame(eoa *dsl.EOA, opts ...GameOpt) *FaultDisputeGame {
	return f.startOutputRootGameOfType(eoa, gameTypes.CannonKonaGameType, f.honestTraceForGame, opts...)
}

func (f *DisputeGameFactory) honestTraceForGame(game *FaultDisputeGame) challengerTypes.TraceAccessor {
	if existing, ok := f.honestTraces[game.Address]; ok {
		return existing
	}
	f.require.NotNil(f.challengerCfg, "Challenger config is required to create honest trace")
	switch game.GameType() {
	case gameTypes.CannonKonaGameType:
		return f.honestOutputCannonTrace(
			game,
			f.challengerCfg.CannonKonaAbsolutePreStateBaseURL,
			f.challengerCfg.CannonKonaAbsolutePreState,
			f.challengerCfg.CannonKona,
			vm.NewKonaExecutor(),
		)
	case gameTypes.SuperCannonKonaGameType:
		return f.honestSuperCannonTrace(
			game,
			f.challengerCfg.CannonKonaAbsolutePreStateBaseURL,
			f.challengerCfg.CannonKonaAbsolutePreState,
			f.challengerCfg.CannonKona,
			vm.NewKonaSuperExecutor(),
		)
	default:
		f.require.Truef(false, "Honest trace not supported for game type %v", game.GameType())
		return nil
	}
}

func (f *DisputeGameFactory) honestOutputCannonTrace(
	game *FaultDisputeGame,
	prestateBaseUrl *url.URL,
	prestateFile string,
	vmConfig vm.Config,
	serverExecutor vm.OracleServerExecutor,
) challengerTypes.TraceAccessor {
	logger := f.t.Logger().New("role", "honestTrace")
	prestateBlock := game.StartingL2SequenceNumber()
	rollupClient := f.l2CL.Escape().RollupAPI()
	prestateProvider := outputs.NewPrestateProvider(rollupClient, prestateBlock)
	l1HeadHash := game.L1Head()
	l1Head, err := f.ethClient.BlockRefByHash(f.t.Ctx(), l1HeadHash)
	f.require.NoError(err, "Failed to fetch L1 Head")

	prestateSource := prestates.NewPrestateSource(
		prestateBaseUrl,
		prestateFile,
		path.Join(f.challengerCfg.Datadir, "test-prestates"),
		cannon.NewStateConverter(vmConfig),
	)
	prestatePath, err := prestateSource.PrestatePath(f.t.Ctx(), game.absolutePrestate())
	f.require.NoError(err, "Failed to get prestate path")
	l2ElClient := f.l2EL.Escape().L2EthClient()
	accessor, err := outputs.NewOutputCannonTraceAccessor(
		logger,
		metrics.NoopMetrics,
		vmConfig,
		serverExecutor,
		&ethClientHeaderProvider{client: l2ElClient},
		prestateProvider,
		prestatePath,
		rollupClient,
		f.t.TempDir(),
		l1Head.ID(),
		game.SplitDepth(),
		prestateBlock,
		game.L2SequenceNumber(),
	)
	f.require.NoError(err, "Failed to create trace accessor")
	f.honestTraces[game.Address] = accessor
	return accessor
}

func (f *DisputeGameFactory) honestSuperCannonTrace(
	game *FaultDisputeGame,
	prestateBaseUrl *url.URL,
	prestateFile string,
	vmConfig vm.Config,
	serverExecutor vm.OracleServerExecutor,
) challengerTypes.TraceAccessor {
	logger := f.t.Logger().New("role", "honestSuperTrace")
	f.require.NotNil(f.superNode, "SuperNode is required to create honest super trace")

	prestateTimestamp := game.StartingL2SequenceNumber()
	poststateTimestamp := game.L2SequenceNumber()

	l1HeadHash := game.L1Head()
	l1Head, err := f.ethClient.BlockRefByHash(f.t.Ctx(), l1HeadHash)
	f.require.NoError(err, "Failed to fetch L1 Head")

	prestateProvider := super.NewSuperNodePrestateProvider(f.superNode.QueryAPI(), prestateTimestamp)

	vmPrestateSource := prestates.NewPrestateSource(
		prestateBaseUrl,
		prestateFile,
		path.Join(f.challengerCfg.Datadir, "test-prestates"),
		cannon.NewStateConverter(vmConfig),
	)
	vmPrestatePath, err := vmPrestateSource.PrestatePath(f.t.Ctx(), game.absolutePrestate())
	f.require.NoError(err, "Failed to get prestate path")

	accessor, err := super.NewSuperCannonTraceAccessor(
		logger,
		metrics.NoopMetrics,
		vmConfig,
		serverExecutor,
		prestateProvider,
		f.superNode.QueryAPI(),
		vmPrestatePath,
		path.Join(f.challengerCfg.Datadir, "test-prestates"),
		l1Head.ID(),
		game.SplitDepth(),
		prestateTimestamp,
		poststateTimestamp,
	)
	f.require.NoError(err, "Failed to create super cannon trace accessor")

	f.honestTraces[game.Address] = accessor
	return accessor
}

func (f *DisputeGameFactory) startOutputRootGameOfType(
	eoa *dsl.EOA,
	gameType gameTypes.GameType,
	honestTraceProvider func(game *FaultDisputeGame) challengerTypes.TraceAccessor,
	opts ...GameOpt) *FaultDisputeGame {
	cfg := NewGameCfg(opts...)
	blockNum := cfg.l2SequenceNumber
	if !cfg.l2SequenceNumberSet {
		blockNum = f.l2CL.SafeL2BlockRef().Number
	}
	extraData := f.createOutputGameExtraData(blockNum, cfg)
	rootClaim := cfg.rootClaim
	if !cfg.rootClaimSet {
		// Default to correct root claim
		response, err := f.l2CL.Escape().RollupAPI().OutputAtBlock(f.t.Ctx(), blockNum)
		f.require.NoErrorf(err, "Failed to get output root at block %v", blockNum)
		rootClaim = common.Hash(response.OutputRoot)
	}
	game, addr := f.createNewGame(eoa, gameType, rootClaim, extraData)
	return NewFaultDisputeGame(f.t, f.require, addr, f.getGameHelper, honestTraceProvider, game)
}

func (f *DisputeGameFactory) createOutputGameExtraData(blockNum uint64, cfg *GameCfg) []byte {
	f.require.NotNil(f.l2CL, "L2 CL is required create output games")
	if !cfg.allowFuture {
		f.l2CL.Reached(safety.LocalSafe, blockNum, 30)
	}
	extraData := make([]byte, 32)
	binary.BigEndian.PutUint64(extraData[24:], blockNum)
	return extraData
}

func (f *DisputeGameFactory) createNewGame(eoa *dsl.EOA, gameType gameTypes.GameType, claim common.Hash, extraData []byte) (*bindings.FaultDisputeGame, common.Address) {
	f.log.Info("Creating dispute game", "gameType", gameType, "claim", claim.Hex(), "extradata", common.Bytes2Hex(extraData))

	// Pull some metadata we need to construct a new game
	requiredBonds := f.initBond(gameType)

	receipt := contract.Write(eoa, f.dgf.Create(uint32(gameType), claim, extraData), txplan.WithValue(requiredBonds), txplan.WithGasRatio(2))
	f.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Extract logs from receipt
	f.require.Equal(2, len(receipt.Logs))
	createdLog, err := f.dgf.ParseDisputeGameCreated(receipt.Logs[1])
	f.require.NoError(err)

	gameAddr := createdLog.DisputeProxy
	log.Info("Dispute game created", "address", gameAddr.Hex())
	return bindings.NewFaultDisputeGame(bindings.WithClient(f.ethClient), bindings.WithTo(gameAddr), bindings.WithTest(f.t)), gameAddr
}

func (f *DisputeGameFactory) initBond(gameType gameTypes.GameType) eth.ETH {
	return eth.WeiBig(contract.Read(f.dgf.InitBonds(uint32(gameType))))
}

func (f *DisputeGameFactory) CreateHelperEOA(eoa *dsl.EOA) *GameHelperEOA {
	helper := f.getGameHelper(eoa)
	eoaHelper := helper.AuthEOA(eoa)
	return &GameHelperEOA{
		helper: eoaHelper,
		EOA:    eoa,
	}
}

// safeTimestamp retrieves the current safe timestamp from the supernode.
func (f *DisputeGameFactory) safeTimestamp() uint64 {
	ctx, cancel := context.WithTimeout(f.t.Ctx(), 2*time.Minute)
	defer cancel()
	timestamp, err := safeTimestamp(ctx, uint64(time.Now().Unix()), time.Second, f.superNode.QueryAPI().SuperRootAtTimestamp)
	f.require.NoError(err, "Failed to fetch super root at timestamp")
	return timestamp
}

func safeTimestamp(
	ctx context.Context,
	timestamp uint64,
	pollInterval time.Duration,
	query superRootAtTimestampFn,
) (uint64, error) {
	resp, err := awaitSuperRootAtTimestamp(ctx, timestamp, pollInterval, query, func(eth.SuperRootAtTimestampResponse) bool {
		return true
	})
	return resp.CurrentSafeTimestamp, err
}

// RunFPP runs the fault proof program between the two supplied timestamps. Currently only supports kona-interop.
func (f *DisputeGameFactory) RunFPP(startTimestamp uint64, endTimestamp uint64) {
	f.require.NotNil(f.superNode, "super node is required to run FPP")
	f.require.NotNil(f.challengerCfg, "challenger config is required to run FPP")

	splitDepth := f.GameImpl(gameTypes.SuperCannonKonaGameType).SplitDepth()

	// Use the current L1 head that the super node has processed. Otherwise the trace provider will fail because the node is not sufficiently up to date.
	superRootResp, err := f.superNode.QueryAPI().SuperRootAtTimestamp(f.t.Ctx(), endTimestamp)
	f.require.NoError(err, "Failed to fetch super root at timestamp")
	l1Head := superRootResp.CurrentL1
	// SuperRootAtTimestamp's CurrentL1 names the block currently being processed.
	// The trace provider's gate requires supernode CurrentL1 > l1Head, so wait
	// until the supernode advances past this block before invoking it.
	f.superNode.AwaitFullyProcessedL1(l1Head.Number)

	prestateProvider := super.NewSuperNodePrestateProvider(f.superNode.QueryAPI(), startTimestamp)
	traceProvider := super.NewSuperNodeTraceProvider(
		f.log.New("role", "fpp-trace"),
		prestateProvider,
		f.superNode.QueryAPI(),
		eth.BlockID{Hash: l1Head.Hash, Number: l1Head.Number},
		splitDepth,
		startTimestamp,
		endTimestamp,
	)

	tmpDir := f.t.TempDir()

	// Starting prestate is the aboslutePrestate
	absolutePrestate, err := prestateProvider.AbsolutePreState(f.t.Ctx())
	f.require.NoError(err, "Failed to get absolute prestate")
	agreedPrestate := absolutePrestate.Marshal()

	// Iterate through valid claims at splitDepth (the leaves of the top game) to get a few steps past the endTimestamp
	for i := uint64(0); i < (endTimestamp-startTimestamp)*super.StepsPerTimestamp+3; i++ {
		pos := challengerTypes.NewPosition(splitDepth, new(big.Int).SetUint64(i))

		timestamp, step, err := traceProvider.ComputeStep(pos)
		f.require.NoError(err, "Failed to compute step")

		// Create LocalGameInputs using the previous claim (or anchor state) as agreed and current as disputed
		f.log.Info("Getting preimage bytes at position", "position", pos, "timestamp", timestamp, "step", step, "i", i)
		claimedPreimage, err := traceProvider.GetPreimageBytes(f.t.Ctx(), pos)
		f.require.NoError(err, "Failed to get claim at position %v", pos)
		inputs := utils.LocalGameInputs{
			L1Head:           l1Head.Hash,
			AgreedPreState:   agreedPrestate,
			L2Claim:          crypto.Keccak256Hash(claimedPreimage),
			L2SequenceNumber: new(big.Int).SetUint64(endTimestamp),
		}

		f.log.Info("Created LocalGameInputs for FPP",
			"index", pos.IndexAtDepth(),
			"l1Head", inputs.L1Head,
			"l2Claim", inputs.L2Claim,
			"startTimestamp", startTimestamp,
			"endTimestamp", endTimestamp,
			"timestamp", timestamp,
			"step", step,
			"invalidTransition", eth.InvalidTransition,
			"invalidTransitionHash", eth.InvalidTransitionHash,
		)

		runFPPForStep(f, tmpDir, inputs)

		// This claim becomes the agreed prestate for the next iteration
		agreedPrestate = claimedPreimage
	}
}

// runFPPForStep executes the native kona interop client using the LocalGameInputs and requires the claim to be successfully validated.
func runFPPForStep(f *DisputeGameFactory, tmpDir string, inputs utils.LocalGameInputs) {
	executor := vm.NewNativeKonaSuperExecutor()
	oracleCommand, err := executor.OracleCommand(f.challengerCfg.CannonKona, tmpDir, inputs)
	f.require.NoError(err, "Failed to create command")
	f.log.Info("Executing FPP", "command", oracleCommand)
	exePath, err := filepath.Abs(oracleCommand[0])
	f.require.NoError(err, "Failed to get absolute path to executable")
	cmd := exec.Command(exePath, oracleCommand[1:]...)
	cmd.Dir = tmpDir
	log := f.log.New("role", "fpp-trace")
	cmd.Stdout = &mipsevm.LoggingWriter{Log: log}
	cmd.Stderr = &mipsevm.LoggingWriter{Log: log}
	cmd.Env = append(append(cmd.Env, os.Environ()...), "NO_COLOR=1")
	err = cmd.Run()
	f.require.NoError(err, "Failed to execute game")
}

type GameHelperEOA struct {
	helper *GameHelper
	EOA    *dsl.EOA
}

func (a *GameHelperEOA) PerformMoves(game *FaultDisputeGame, moves ...GameHelperMove) []*Claim {
	return a.helper.PerformMoves(a.EOA, game, moves)
}

func (a *GameHelperEOA) Address() common.Address {
	return a.EOA.Address()
}

// ethClientHeaderProvider is an adapter for the L1Client interface used in op-node and devstack to
// the HeaderProvider interface used in challenger
type ethClientHeaderProvider struct {
	client apis.EthClient
}

func (p *ethClientHeaderProvider) HeaderByNumber(ctx context.Context, blockNum *big.Int) (*types.Header, error) {
	return p.client.HeaderByNumber(ctx, bigs.Uint64Strict(blockNum))
}
