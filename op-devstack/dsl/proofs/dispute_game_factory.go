package proofs

import (
	"encoding/binary"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	t          devtest.T
	require    *require.Assertions
	log        log.Logger
	l1Network  *dsl.L1Network
	ethClient  apis.EthClient
	dgf        *bindings.DisputeGameFactory
	addr       common.Address
	supervisor *dsl.Supervisor
	gameHelper *GameHelper
}

func NewDisputeGameFactory(t devtest.T, l1Network *dsl.L1Network, ethClient apis.EthClient, dgfAddr common.Address, supervisor *dsl.Supervisor) *DisputeGameFactory {
	dgf := bindings.NewDisputeGameFactory(bindings.WithClient(ethClient), bindings.WithTo(dgfAddr), bindings.WithTest(t))

	return &DisputeGameFactory{
		t:          t,
		require:    require.New(t),
		log:        t.Logger(),
		l1Network:  l1Network,
		dgf:        dgf,
		addr:       dgfAddr,
		supervisor: supervisor,
		ethClient:  ethClient,
	}
}

type GameCfg struct {
	allowFuture  bool
	allowUnsafe  bool
	rootClaimSet bool
	rootClaim    common.Hash
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
	gs := DeployGameHelper(f.t, eoa)
	f.gameHelper = gs
	return gs
}

func (f *DisputeGameFactory) GameCount() int64 {
	return contract.Read(f.dgf.GameCount()).Int64()
}

func (f *DisputeGameFactory) GameAtIndex(idx int64) *FaultDisputeGame {
	gameInfo := contract.Read(f.dgf.GameAtIndex(big.NewInt(idx)))
	game := bindings.NewFaultDisputeGame(bindings.WithClient(f.ethClient), bindings.WithTo(gameInfo.Proxy), bindings.WithTest(f.t))
	return NewFaultDisputeGame(f.t, f.require, gameInfo.Proxy, f.getGameHelper, game)
}

func (f *DisputeGameFactory) gameImpl(gameType challengerTypes.GameType) *FaultDisputeGame {
	implAddr := contract.Read(f.dgf.GameImpls(uint32(gameType)))
	game := bindings.NewFaultDisputeGame(bindings.WithClient(f.ethClient), bindings.WithTo(implAddr), bindings.WithTest(f.t))
	return NewFaultDisputeGame(f.t, f.require, implAddr, f.getGameHelper, game)
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
	proposalTimestamp := f.supervisor.FetchSyncStatus().SafeTimestamp

	return f.startSuperCannonGameOfType(eoa, proposalTimestamp, challengerTypes.SuperCannonGameType, opts...)
}

func (f *DisputeGameFactory) startSuperCannonGameOfType(eoa *dsl.EOA, timestamp uint64, gameType challengerTypes.GameType, opts ...GameOpt) *SuperFaultDisputeGame {
	cfg := NewGameCfg(opts...)
	extraData := f.createSuperGameExtraData(timestamp, cfg)
	rootClaim := cfg.rootClaim
	if !cfg.rootClaimSet {
		// Default to the correct root claim
		response := f.supervisor.FetchSuperRootAtTimestamp(timestamp)
		rootClaim = common.Hash(response.SuperRoot)
	}
	game, addr := f.createNewGame(eoa, gameType, rootClaim, extraData)

	return NewSuperFaultDisputeGame(f.t, f.require, addr, f.getGameHelper, game)
}

func (f *DisputeGameFactory) createSuperGameExtraData(timestamp uint64, cfg *GameCfg) []byte {
	if !cfg.allowFuture {
		f.supervisor.AwaitMinCrossSafeTimestamp(timestamp)
	}
	extraData := make([]byte, 32)
	binary.BigEndian.PutUint64(extraData[24:], timestamp)
	return extraData
}

func (f *DisputeGameFactory) createNewGame(eoa *dsl.EOA, gameType challengerTypes.GameType, claim common.Hash, extraData []byte) (*bindings.FaultDisputeGame, common.Address) {
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

func (f *DisputeGameFactory) initBond(gameType challengerTypes.GameType) eth.ETH {
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
