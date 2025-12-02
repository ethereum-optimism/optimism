package proofs

import (
	"context"
	"encoding/binary"
	"math/big"
	"net/url"
	"path"
	"time"

	challengerConfig "github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/prestates"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safetyTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	challengerTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
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
	supervisor    *dsl.Supervisor
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
	supervisor *dsl.Supervisor,
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
		supervisor:    supervisor,
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

func (f *DisputeGameFactory) GameImpl(gameType gameTypes.GameType) *FaultDisputeGame {
	implAddr := contract.Read(f.dgf.GameImpls(uint32(gameType)))
	game := bindings.NewFaultDisputeGame(bindings.WithClient(f.ethClient), bindings.WithTo(implAddr), bindings.WithTest(f.t))
	return NewFaultDisputeGame(f.t, f.require, implAddr, f.getGameHelper, f.honestTraceForGame, game)
}

func (f *DisputeGameFactory) GameArgs(gameType gameTypes.GameType) []byte {
	return contract.Read(f.dgf.GameArgs(uint32(gameType)))
}

func (f *DisputeGameFactory) WaitForGame() *FaultDisputeGame {
	initialCount := f.GameCount()
	f.t.Require().Eventually(func() bool {
		gameCount := f.GameCount()
		check := gameCount > initialCount
		f.t.Logf("waiting for new game. current=%d new=%d", initialCount, gameCount)
		return check
	}, time.Minute*10, time.Second*5)

	return f.GameAtIndex(initialCount)
}

func (f *DisputeGameFactory) StartSuperCannonGame(eoa *dsl.EOA, opts ...GameOpt) *SuperFaultDisputeGame {
	f.require.NotNil(f.supervisor, "supervisor is required to start super games")

	return f.startSuperCannonGameOfType(eoa, gameTypes.SuperCannonGameType, opts...)
}

func (f *DisputeGameFactory) startSuperCannonGameOfType(eoa *dsl.EOA, gameType gameTypes.GameType, opts ...GameOpt) *SuperFaultDisputeGame {
	cfg := NewGameCfg(opts...)
	if len(cfg.superOutputRoots) != 0 && cfg.rootClaimSet {
		f.t.Error("cannot set both super output roots and root claim in super game")
		f.t.FailNow()
	}
	timestamp := cfg.l2SequenceNumber
	if !cfg.l2SequenceNumberSet {
		timestamp = f.supervisor.FetchSyncStatus().SafeTimestamp
	}
	extraData := f.createSuperGameExtraData(timestamp, cfg)
	rootClaim := cfg.rootClaim
	if !cfg.rootClaimSet {
		rootClaim = crypto.Keccak256Hash(extraData)
	}
	game, addr := f.createNewGame(eoa, gameType, rootClaim, extraData)

	return NewSuperFaultDisputeGame(f.t, f.require, addr, f.getGameHelper, game)
}

func (f *DisputeGameFactory) createSuperGameExtraData(timestamp uint64, cfg *GameCfg) []byte {
	f.require.NotNil(f.supervisor, "supervisor is required create super games")
	if !cfg.allowFuture {
		f.supervisor.AwaitMinCrossSafeTimestamp(timestamp)
	}
	super, err := f.supervisor.FetchSuperRootAtTimestamp(timestamp).ToSuper()
	f.require.NoError(err, "Failed to fetch super root for timestamp %v", timestamp)

	superV1, ok := super.(*eth.SuperV1)
	f.require.Truef(ok, "Unsupported super type %T", super)
	if len(cfg.superOutputRoots) != 0 {
		f.require.Len(cfg.superOutputRoots, len(superV1.Chains), "Super output roots length mismatch")
		for i := range superV1.Chains {
			superV1.Chains[i].Output = cfg.superOutputRoots[i]
		}
	}
	extraData := superV1.Marshal()
	return extraData
}

func (f *DisputeGameFactory) StartCannonGame(eoa *dsl.EOA, opts ...GameOpt) *FaultDisputeGame {
	return f.startOutputRootGameOfType(eoa, gameTypes.CannonGameType, f.honestTraceForGame, opts...)
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
	case gameTypes.CannonGameType:
		return f.honestOutputCannonTrace(
			game,
			f.challengerCfg.CannonAbsolutePreStateBaseURL,
			f.challengerCfg.CannonAbsolutePreState,
			f.challengerCfg.Cannon,
			vm.NewOpProgramServerExecutor(f.log),
		)
	case gameTypes.CannonKonaGameType:
		return f.honestOutputCannonTrace(
			game,
			f.challengerCfg.CannonKonaAbsolutePreStateBaseURL,
			f.challengerCfg.CannonKonaAbsolutePreState,
			f.challengerCfg.CannonKona,
			vm.NewKonaExecutor(),
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
		f.l2CL.Reached(safetyTypes.LocalSafe, blockNum, 30)
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
	info, err := p.client.InfoByNumber(ctx, blockNum.Uint64())
	if err != nil {
		return nil, err
	}
	return info.Header(), nil
}
