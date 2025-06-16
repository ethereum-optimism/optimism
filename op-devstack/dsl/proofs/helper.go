package proofs

import (
	"encoding/binary"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	cTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl/contract"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

type DisputeGameFactoryHelper struct {
	t          devtest.T
	require    *require.Assertions
	log        log.Logger
	l1Network  *dsl.L1Network
	l2Network  *dsl.L2Network
	supervisor *dsl.Supervisor
	ethClient  apis.EthClient
}

func HelperFromInteropPreset(t devtest.T, sys *presets.SimpleInterop, l2Network *dsl.L2Network) *DisputeGameFactoryHelper {
	ethClient := sys.L1EL.EthClient()
	return &DisputeGameFactoryHelper{
		t:          t,
		require:    require.New(t),
		log:        t.Logger(),
		l1Network:  sys.L1Network,
		l2Network:  l2Network,
		supervisor: sys.Supervisor,
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

func (h *DisputeGameFactoryHelper) StartSuperCannonGame(eoa *dsl.EOA, rootClaim common.Hash, opts ...GameOpt) *SuperCannonGameHelper {
	block := h.l1Network.WaitForBlock()

	gameType := uint32(cTypes.SuperCannonGameType)
	return h.startSuperCannonGameOfType(eoa, block.Time, rootClaim, gameType, opts...)
}

func (h *DisputeGameFactoryHelper) startSuperCannonGameOfType(eoa *dsl.EOA, timestamp uint64, rootClaim common.Hash, gameType uint32, opts ...GameOpt) *SuperCannonGameHelper {
	cfg := NewGameCfg(opts...)
	extraData := h.createSuperGameExtraData(timestamp, cfg)
	game := h.createNewGame(eoa, gameType, rootClaim, extraData)

	return NewSuperCannonGameHelper(h.t, h.require, game)
}

func (h *DisputeGameFactoryHelper) createSuperGameExtraData(timestamp uint64, cfg *GameCfg) []byte {
	if !cfg.allowFuture {
		require.Eventually(h.t, func() bool {
			status := h.supervisor.FetchSyncStatus()
			return status.SafeTimestamp >= timestamp
		}, time.Minute, 5*time.Second, "Safe head did not reach proposal timestamp")
	}
	extraData := make([]byte, 32)
	binary.BigEndian.PutUint64(extraData[24:], timestamp)
	return extraData
}

func (h *DisputeGameFactoryHelper) createNewGame(eoa *dsl.EOA, gameType uint32, claim common.Hash, extraData []byte) *bindings.FaultDisputeGame {
	h.log.Info("Creating dispute game", "gameType", gameType, "claim", claim.Hex(), "extradata", common.Bytes2Hex(extraData))

	dgfAddr := h.l2Network.DisputeGameFactoryProxyAddr()
	dgf := bindings.NewDisputeGameFactory(bindings.WithClient(h.ethClient), bindings.WithTo(dgfAddr), bindings.WithTest(h.t))

	// Pull some metadata we need to construct a new game
	requiredBonds := contract.Read(dgf.InitBonds(gameType))

	receipt := contract.Write(eoa, dgf.Create(gameType, claim, extraData), txplan.WithValue(requiredBonds), txplan.WithGasRatio(2))
	h.require.Equal(types.ReceiptStatusSuccessful, receipt.Status)

	// Extract logs from receipt
	h.require.Equal(2, len(receipt.Logs))
	createdLog, err := dgf.ParseDisputeGameCreated(receipt.Logs[1])
	h.require.NoError(err)

	gameAddr := createdLog.DisputeProxy
	log.Info("Dispute game created", "address", gameAddr.Hex())
	return bindings.NewFaultDisputeGame(bindings.WithClient(h.ethClient), bindings.WithTo(gameAddr), bindings.WithTest(h.t))
}
