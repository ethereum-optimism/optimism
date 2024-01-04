package conductor

import (
	"context"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	clientmocks "github.com/ethereum-optimism/optimism/op-conductor/client/mocks"
	consensusmocks "github.com/ethereum-optimism/optimism/op-conductor/consensus/mocks"
	healthmocks "github.com/ethereum-optimism/optimism/op-conductor/health/mocks"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func mockConfig(t *testing.T) Config {
	now := uint64(time.Now().Unix())
	dir, err := os.MkdirTemp("/tmp", "")
	require.NoError(t, err)
	return Config{
		ConsensusAddr:  "127.0.0.1",
		ConsensusPort:  50050,
		RaftServerID:   "SequencerA",
		RaftStorageDir: dir,
		RaftBootstrap:  false,
		NodeRPC:        "http://node:8545",
		ExecutionRPC:   "http://geth:8545",
		HealthCheck: HealthCheckConfig{
			Interval:     1,
			SafeInterval: 5,
			MinPeerCount: 1,
		},
		RollupCfg: rollup.Config{
			Genesis: rollup.Genesis{
				L1: eth.BlockID{
					Hash:   [32]byte{1, 2},
					Number: 100,
				},
				L2: eth.BlockID{
					Hash:   [32]byte{2, 3},
					Number: 0,
				},
				L2Time: now,
				SystemConfig: eth.SystemConfig{
					BatcherAddr: [20]byte{1},
					Overhead:    [32]byte{1},
					Scalar:      [32]byte{1},
					GasLimit:    30000000,
				},
			},
			BlockTime:               2,
			MaxSequencerDrift:       600,
			SeqWindowSize:           3600,
			ChannelTimeout:          300,
			L1ChainID:               big.NewInt(1),
			L2ChainID:               big.NewInt(2),
			CanyonTime:              &now,
			BatchInboxAddress:       [20]byte{1, 2},
			DepositContractAddress:  [20]byte{2, 3},
			L1SystemConfigAddress:   [20]byte{3, 4},
			ProtocolVersionsAddress: [20]byte{4, 5},
		},
	}
}

type OpConductorTestSuite struct {
	suite.Suite

	conductor *OpConductor

	healthUpdateCh chan bool
	leaderUpdateCh chan bool

	ctx     context.Context
	log     log.Logger
	cfg     Config
	version string
	ctrl    *clientmocks.SequencerControl
	cons    *consensusmocks.Consensus
	hmon    *healthmocks.HealthMonitor
}

func (s *OpConductorTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.log = testlog.Logger(s.T(), log.LvlDebug)
	s.cfg = mockConfig(s.T())
	s.version = "v0.0.1"
	s.ctrl = &clientmocks.SequencerControl{}
	s.cons = &consensusmocks.Consensus{}
	s.hmon = &healthmocks.HealthMonitor{}

	s.cons.EXPECT().ServerID().Return("SequencerA")
}

func (s *OpConductorTestSuite) SetupTest() {
	conductor, err := NewOpConductor(s.ctx, &s.cfg, s.log, s.version, s.ctrl, s.cons, s.hmon)
	s.NoError(err)
	s.conductor = conductor

	s.healthUpdateCh = make(chan bool)
	s.hmon.EXPECT().Start().Return(nil)
	s.hmon.EXPECT().Subscribe().Return(s.healthUpdateCh)

	s.leaderUpdateCh = make(chan bool)
	s.cons.EXPECT().LeaderCh().Return(s.leaderUpdateCh)

	err = s.conductor.Start(s.ctx)
	s.NoError(err)
	s.False(s.conductor.Stopped())
}

// Scenario 1: pause -> resume -> stop
func (s *OpConductorTestSuite) TestControlLoop1() {
	// Pause
	err := s.conductor.Pause(s.ctx)
	s.NoError(err)
	s.True(s.conductor.Paused())

	// Send health update, make sure it can still be consumed.
	s.healthUpdateCh <- true

	// Resume
	err = s.conductor.Resume(s.ctx)
	s.NoError(err)
	s.False(s.conductor.Paused())

	// Stop
	s.hmon.EXPECT().Stop().Return(nil)
	s.cons.EXPECT().Shutdown().Return(nil)
	err = s.conductor.Stop(s.ctx)
	s.NoError(err)
	s.True(s.conductor.Stopped())
}

// Scenario 2: pause -> pause -> resume -> resume
func (s *OpConductorTestSuite) TestControlLoop2() {
	// Pause
	err := s.conductor.Pause(s.ctx)
	s.NoError(err)
	s.True(s.conductor.Paused())

	// Pause again, this shouldn't block or cause any other issues
	err = s.conductor.Pause(s.ctx)
	s.NoError(err)
	s.True(s.conductor.Paused())

	// Resume
	err = s.conductor.Resume(s.ctx)
	s.NoError(err)
	s.False(s.conductor.Paused())

	// Resume
	err = s.conductor.Resume(s.ctx)
	s.NoError(err)
	s.False(s.conductor.Paused())
}

// Scenario 3: pause -> stop
func (s *OpConductorTestSuite) TestControlLoop3() {
	// Pause
	err := s.conductor.Pause(s.ctx)
	s.NoError(err)
	s.True(s.conductor.Paused())

	// Stop
	s.hmon.EXPECT().Stop().Return(nil)
	s.cons.EXPECT().Shutdown().Return(nil)
	err = s.conductor.Stop(s.ctx)
	s.NoError(err)
	s.True(s.conductor.Stopped())
}

func TestHealthMonitor(t *testing.T) {
	suite.Run(t, new(OpConductorTestSuite))
}
