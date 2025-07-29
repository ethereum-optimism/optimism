package proofs

import (
	"encoding/binary"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	cTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
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
	supervisor *dsl.Supervisor
}

func NewDisputeGameFactory(t devtest.T, l1Network *dsl.L1Network, ethClient apis.EthClient, dgfAddr common.Address, supervisor *dsl.Supervisor) *DisputeGameFactory {
	dgf := bindings.NewDisputeGameFactory(bindings.WithClient(ethClient), bindings.WithTo(dgfAddr), bindings.WithTest(t))

	return &DisputeGameFactory{
		t:          t,
		require:    require.New(t),
		log:        t.Logger(),
		l1Network:  l1Network,
		dgf:        dgf,
		supervisor: supervisor,
		ethClient:  ethClient,
	}
}

type GameCfg struct {
	allowFuture bool
	allowUnsafe bool
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

func NewGameCfg(opts ...GameOpt) *GameCfg {
	cfg := &GameCfg{}
	for _, opt := range opts {
		opt.Apply(cfg)
	}
	return cfg
}

func (f *DisputeGameFactory) GameCount() int64 {
	return contract.Read(f.dgf.GameCount()).Int64()
}

func (f *DisputeGameFactory) GameAtIndex(idx int64) *FaultDisputeGame {
	gameInfo := contract.Read(f.dgf.GameAtIndex(big.NewInt(idx)))
	game := bindings.NewFaultDisputeGame(bindings.WithClient(f.ethClient), bindings.WithTo(gameInfo.Proxy), bindings.WithTest(f.t))
	return NewFaultDisputeGame(f.t, f.require, game)
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

func (f *DisputeGameFactory) StartSuperCannonGame(eoa *dsl.EOA, rootClaim common.Hash, opts ...GameOpt) *SuperFaultDisputeGame {
	proposalTimestamp := f.supervisor.FetchSyncStatus().SafeTimestamp

	gameType := uint32(cTypes.SuperCannonGameType)
	return f.startSuperCannonGameOfType(eoa, proposalTimestamp, rootClaim, gameType, opts...)
}

func (f *DisputeGameFactory) startSuperCannonGameOfType(eoa *dsl.EOA, timestamp uint64, rootClaim common.Hash, gameType uint32, opts ...GameOpt) *SuperFaultDisputeGame {
	cfg := NewGameCfg(opts...)
	extraData := f.createSuperGameExtraData(timestamp, cfg)
	game := f.createNewGame(eoa, gameType, rootClaim, extraData)

	return NewSuperFaultDisputeGame(f.t, f.require, game)
}

func (f *DisputeGameFactory) createSuperGameExtraData(timestamp uint64, cfg *GameCfg) []byte {
	if !cfg.allowFuture {
		f.supervisor.AwaitMinCrossSafeTimestamp(timestamp)
	}
	extraData := make([]byte, 32)
	binary.BigEndian.PutUint64(extraData[24:], timestamp)
	return extraData
}

func (f *DisputeGameFactory) createNewGame(eoa *dsl.EOA, gameType uint32, claim common.Hash, extraData []byte) *bindings.FaultDisputeGame {
	f.log.Info("Creating dispute game", "gameType", gameType, "claim", claim.Hex(), "extradata", common.Bytes2Hex(extraData))

	// Pull some metadata we need to construct a new game
	requiredBonds := contract.Read(f.dgf.InitBonds(gameType))

	receipt := contract.Write(eoa, f.dgf.Create(gameType, claim, extraData), txplan.WithValue(requiredBonds), txplan.WithGasRatio(2))
	f.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Extract logs from receipt
	f.require.Equal(2, len(receipt.Logs))
	createdLog, err := f.dgf.ParseDisputeGameCreated(receipt.Logs[1])
	f.require.NoError(err)

	gameAddr := createdLog.DisputeProxy
	log.Info("Dispute game created", "address", gameAddr.Hex())
	return bindings.NewFaultDisputeGame(bindings.WithClient(f.ethClient), bindings.WithTo(gameAddr), bindings.WithTest(f.t))
}
