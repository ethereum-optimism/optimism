package health

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/suite"

	"github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pMocks "github.com/ethereum-optimism/optimism/op-node/p2p/mocks"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

const (
	unhealthyPeerCount = 0
	minPeerCount       = 1
	healthyPeerCount   = 2
	blockTime          = 2
)

type HealthMonitorTestSuite struct {
	suite.Suite

	log          log.Logger
	rc           *testutils.MockRollupClient
	pc           *p2pMocks.API
	interval     uint64
	safeInterval uint64
	minPeerCount uint64
	rollupCfg    *rollup.Config
	monitor      HealthMonitor
}

func (s *HealthMonitorTestSuite) SetupSuite() {
	s.log = testlog.Logger(s.T(), log.LvlInfo)
	s.rc = &testutils.MockRollupClient{}
	s.pc = &p2pMocks.API{}
	s.interval = 1
	s.safeInterval = 5
	s.minPeerCount = minPeerCount
	s.rollupCfg = &rollup.Config{
		BlockTime: blockTime,
	}
}

func (s *HealthMonitorTestSuite) SetupTest() {
	s.monitor = NewSequencerHealthMonitor(s.log, s.interval, s.safeInterval, s.minPeerCount, s.rollupCfg, s.rc, s.pc)
	err := s.monitor.Start()
	s.NoError(err)
}

func (s *HealthMonitorTestSuite) TearDownTest() {
	err := s.monitor.Stop()
	s.NoError(err)
}

func (s *HealthMonitorTestSuite) TestUnhealthyLowPeerCount() {
	now := uint64(time.Now().Unix())
	ss1 := &eth.SyncStatus{
		UnsafeL2: eth.L2BlockRef{
			Time: now - 1,
		},
		SafeL2: eth.L2BlockRef{
			Time: now - 2,
		},
	}
	s.rc.ExpectSyncStatus(ss1, nil)

	ps1 := &p2p.PeerStats{
		Connected: unhealthyPeerCount,
	}
	s.pc.EXPECT().PeerStats(context.Background()).Return(ps1, nil).Times(1)

	healthUpdateCh := s.monitor.Subscribe()
	healthy := <-healthUpdateCh
	s.False(healthy)
}

func (s *HealthMonitorTestSuite) TestUnhealthyUnsafeHeadNotProgressing() {
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	s.pc.EXPECT().PeerStats(context.Background()).Return(ps1, nil).Times(3)

	now := uint64(time.Now().Unix())
	ss1 := &eth.SyncStatus{
		UnsafeL2: eth.L2BlockRef{
			Time: now - 1,
		},
		SafeL2: eth.L2BlockRef{
			Time: now - 2,
		},
	}
	s.rc.ExpectSyncStatus(ss1, nil)
	s.rc.ExpectSyncStatus(ss1, nil)
	s.rc.ExpectSyncStatus(ss1, nil)

	healthUpdateCh := s.monitor.Subscribe()
	for i := 0; i < 3; i++ {
		healthy := <-healthUpdateCh
		if i < 2 {
			s.True(healthy)
		} else {
			s.False(healthy)
		}
	}
}

func (s *HealthMonitorTestSuite) TestUnhealthySafeHeadNotProgressing() {
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	s.pc.EXPECT().PeerStats(context.Background()).Return(ps1, nil).Times(6)

	now := uint64(time.Now().Unix())
	syncStatusGenerator := func(unsafeTime uint64) *eth.SyncStatus {
		return &eth.SyncStatus{
			UnsafeL2: eth.L2BlockRef{
				Time: unsafeTime,
			},
			SafeL2: eth.L2BlockRef{
				Time: now,
			},
		}
	}
	s.rc.ExpectSyncStatus(syncStatusGenerator(now), nil)
	s.rc.ExpectSyncStatus(syncStatusGenerator(now), nil)
	s.rc.ExpectSyncStatus(syncStatusGenerator(now+2), nil)
	s.rc.ExpectSyncStatus(syncStatusGenerator(now+2), nil)
	s.rc.ExpectSyncStatus(syncStatusGenerator(now+4), nil)
	s.rc.ExpectSyncStatus(syncStatusGenerator(now+4), nil)

	healthUpdateCh := s.monitor.Subscribe()
	for i := 0; i < 6; i++ {
		healthy := <-healthUpdateCh
		if i < 5 {
			s.True(healthy)
		} else {
			s.False(healthy)
		}
	}
}

func TestHealthMonitor(t *testing.T) {
	suite.Run(t, new(HealthMonitorTestSuite))
}
