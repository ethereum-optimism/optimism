package health

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
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
	interval     uint64
	minPeerCount uint64
	rollupCfg    *rollup.Config
}

func (s *HealthMonitorTestSuite) SetupSuite() {
	s.log = testlog.Logger(s.T(), log.LevelDebug)
	s.interval = 1
	s.minPeerCount = minPeerCount
	s.rollupCfg = &rollup.Config{
		BlockTime: blockTime,
	}
}

func (s *HealthMonitorTestSuite) SetupMonitor(
	now, unsafeInterval, safeInterval uint64,
	mockRollupClient *testutils.MockRollupClient,
	mockP2P *p2pMocks.API,
) *SequencerHealthMonitor {
	tp := &timeProvider{now: now}
	if mockP2P == nil {
		mockP2P = &p2pMocks.API{}
		ps1 := &p2p.PeerStats{
			Connected: healthyPeerCount,
		}
		mockP2P.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)
	}
	monitor := &SequencerHealthMonitor{
		log:            s.log,
		interval:       s.interval,
		metrics:        &metrics.NoopMetricsImpl{},
		healthUpdateCh: make(chan error),
		rollupCfg:      s.rollupCfg,
		unsafeInterval: unsafeInterval,
		safeInterval:   safeInterval,
		safeEnabled:    true,
		minPeerCount:   s.minPeerCount,
		timeProviderFn: tp.Now,
		node:           mockRollupClient,
		p2p:            mockP2P,
	}
	err := monitor.Start(context.Background())
	s.NoError(err)
	return monitor
}

func (s *HealthMonitorTestSuite) TestUnhealthyLowPeerCount() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)
	rc.ExpectSyncStatus(ss1, nil)
	rc.ExpectSyncStatus(ss1, nil)

	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: unhealthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil).Times(1)

	monitor := s.SetupMonitor(now, 60, 60, rc, pc)

	healthUpdateCh := monitor.Subscribe()
	healthy := <-healthUpdateCh
	s.NotNil(healthy)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestUnhealthyUnsafeHeadNotProgressing() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now, 5, now-8, 1)
	for i := 0; i < 5; i++ {
		rc.ExpectSyncStatus(ss1, nil)
	}

	monitor := s.SetupMonitor(now, 60, 60, rc, nil)
	healthUpdateCh := monitor.Subscribe()

	for i := 0; i < 5; i++ {
		healthy := <-healthUpdateCh
		if i < 4 {
			s.Nil(healthy)
			s.Equal(now, monitor.lastSeenUnsafeTime)
			s.Equal(uint64(5), monitor.lastSeenUnsafeNum)
		} else {
			s.NotNil(healthy)
		}
	}

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestUnhealthySafeHeadNotProgressing() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	rc.ExpectSyncStatus(mockSyncStatus(now, 1, now, 1), nil)
	rc.ExpectSyncStatus(mockSyncStatus(now, 1, now, 1), nil)
	rc.ExpectSyncStatus(mockSyncStatus(now+2, 2, now, 1), nil)
	rc.ExpectSyncStatus(mockSyncStatus(now+2, 2, now, 1), nil)
	rc.ExpectSyncStatus(mockSyncStatus(now+4, 3, now, 1), nil)
	rc.ExpectSyncStatus(mockSyncStatus(now+4, 3, now, 1), nil)

	monitor := s.SetupMonitor(now, 60, 3, rc, nil)
	healthUpdateCh := monitor.Subscribe()

	for i := 0; i < 5; i++ {
		healthy := <-healthUpdateCh
		if i < 4 {
			s.Nil(healthy)
		} else {
			s.NotNil(healthy)
		}
	}

	// test that the safeEnabled flag works
	monitor.safeEnabled = false
	rc.ExpectSyncStatus(mockSyncStatus(now+6, 4, now, 1), nil)
	rc.ExpectSyncStatus(mockSyncStatus(now+6, 4, now, 1), nil)
	healthy := <-healthUpdateCh
	s.Nil(healthy)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestHealthyWithUnsafeLag() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	// although unsafe has lag of 20 seconds, it's within the configured unsafe interval
	// and it is advancing every block time, so it should be considered safe.
	rc.ExpectSyncStatus(mockSyncStatus(now-10, 1, now, 1), nil)
	rc.ExpectSyncStatus(mockSyncStatus(now-10, 1, now, 1), nil)
	rc.ExpectSyncStatus(mockSyncStatus(now-8, 2, now, 1), nil)
	// in this case now time is behind unsafe head time, this should still be considered healthy.
	rc.ExpectSyncStatus(mockSyncStatus(now+5, 2, now, 1), nil)

	monitor := s.SetupMonitor(now, 60, 60, rc, nil)
	healthUpdateCh := monitor.Subscribe()

	// confirm initial state
	s.Zero(monitor.lastSeenUnsafeNum)
	s.Zero(monitor.lastSeenUnsafeTime)

	// confirm state after first check
	healthy := <-healthUpdateCh
	s.Nil(healthy)
	lastSeenUnsafeTime := monitor.lastSeenUnsafeTime
	s.NotZero(monitor.lastSeenUnsafeTime)
	s.Equal(uint64(1), monitor.lastSeenUnsafeNum)

	healthy = <-healthUpdateCh
	s.Nil(healthy)
	s.Equal(lastSeenUnsafeTime, monitor.lastSeenUnsafeTime)
	s.Equal(uint64(1), monitor.lastSeenUnsafeNum)

	healthy = <-healthUpdateCh
	s.Nil(healthy)
	s.Equal(lastSeenUnsafeTime+2, monitor.lastSeenUnsafeTime)
	s.Equal(uint64(2), monitor.lastSeenUnsafeNum)

	healthy = <-healthUpdateCh
	s.Nil(healthy)
	s.Equal(lastSeenUnsafeTime+2, monitor.lastSeenUnsafeTime)
	s.Equal(uint64(2), monitor.lastSeenUnsafeNum)

	s.NoError(monitor.Stop())
}

func mockSyncStatus(unsafeTime, unsafeNum, safeTime, safeNum uint64) *eth.SyncStatus {
	return &eth.SyncStatus{
		UnsafeL2: eth.L2BlockRef{
			Time:   unsafeTime,
			Number: unsafeNum,
		},
		SafeL2: eth.L2BlockRef{
			Time:   safeTime,
			Number: safeNum,
		},
	}
}

func TestHealthMonitor(t *testing.T) {
	suite.Run(t, new(HealthMonitorTestSuite))
}

type timeProvider struct {
	now uint64
}

func (tp *timeProvider) Now() uint64 {
	now := tp.now
	tp.now++
	return now
}
