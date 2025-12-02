package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/ethereum-optimism/optimism/op-conductor/client"
	clientmocks "github.com/ethereum-optimism/optimism/op-conductor/client/mocks"
	mocks "github.com/ethereum-optimism/optimism/op-conductor/health/mocks"
	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	p2pMocks "github.com/ethereum-optimism/optimism/op-node/p2p/mocks"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

const (
	unhealthyPeerCount      = 0
	minPeerCount            = 1
	healthyPeerCount        = 2
	blockTime               = 2
	interval                = 1
	minElP2pPeerCount       = 2
	healthyElP2pPeerCount   = 3
	unhealthyElP2pPeerCount = 1
)

type HealthMonitorTestSuite struct {
	suite.Suite

	log          log.Logger
	interval     uint64
	minPeerCount uint64
	rollupCfg    *rollup.Config

	minElP2pPeerCount uint64
}

func (s *HealthMonitorTestSuite) SetupSuite() {
	s.log = testlog.Logger(s.T(), log.LevelDebug)
	s.interval = interval
	s.minPeerCount = minPeerCount
	s.rollupCfg = &rollup.Config{
		BlockTime: blockTime,
	}
	s.minElP2pPeerCount = minElP2pPeerCount
}

func (s *HealthMonitorTestSuite) SetupMonitor(
	now, unsafeInterval, safeInterval uint64,
	mockRollupClient *testutils.MockRollupClient,
	mockP2P *p2pMocks.API,
	mockSupervisorHealthAPI SupervisorHealthAPI,
	elP2pClient client.ElP2PClient,
) *SequencerHealthMonitor {
	tp := &timeProvider{now: now}
	if mockP2P == nil {
		mockP2P = &p2pMocks.API{}
		ps1 := &apis.PeerStats{
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
		supervisor:     mockSupervisorHealthAPI,
	}
	if elP2pClient != nil {
		monitor.elP2p = &ElP2pHealthMonitor{
			log:          s.log,
			minPeerCount: s.minElP2pPeerCount,
			elP2pClient:  elP2pClient,
		}
	}
	err := monitor.Start(context.Background())
	s.NoError(err)
	return monitor
}

type monitorOpts func(*SequencerHealthMonitor)

// SetupMonitorWithRollupBoost creates a HealthMonitor that includes a RollupBoostHealthChecker
func (s *HealthMonitorTestSuite) SetupMonitorWithRollupBoost(
	now, unsafeInterval, safeInterval uint64,
	mockRollupClient *testutils.MockRollupClient,
	mockP2P *p2pMocks.API,
	mockRollupBoostHealthChecker *clientmocks.RollupBoostHealthChecker,
	elP2pClient client.ElP2PClient,
	opts ...monitorOpts,
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
	if mockRollupBoostHealthChecker != nil {
		monitor.rollupBoostHealthChecker = mockRollupBoostHealthChecker
	}
	if elP2pClient != nil {
		monitor.elP2p = &ElP2pHealthMonitor{
			log:          s.log,
			minPeerCount: s.minElP2pPeerCount,
			elP2pClient:  elP2pClient,
		}
	}
	for _, opt := range opts {
		opt(monitor)
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
	ps1 := &apis.PeerStats{
		Connected: unhealthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil).Times(1)

	monitor := s.SetupMonitor(now, 60, 60, rc, pc, nil, nil)

	healthUpdateCh := monitor.Subscribe()
	healthFailure := <-healthUpdateCh
	s.NotNil(healthFailure)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestUnhealthyLowElP2pPeerCount() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)
	rc.ExpectSyncStatus(ss1, nil)
	rc.ExpectSyncStatus(ss1, nil)

	healthyPc := &p2pMocks.API{}
	ps1 := &apis.PeerStats{
		Connected: healthyPeerCount,
	}
	healthyPc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil).Times(1)

	elP2pClient := &clientmocks.ElP2PClient{}
	elP2pClient.EXPECT().PeerCount(mock.Anything).Return(unhealthyElP2pPeerCount, nil).Times(1)

	monitor := s.SetupMonitor(now, 60, 60, rc, healthyPc, nil, elP2pClient)

	healthUpdateCh := monitor.Subscribe()
	healthFailure := <-healthUpdateCh
	s.NotNil(healthFailure)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestUnhealthyUnsafeHeadNotProgressing() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now, 5, now-8, 1)
	unsafeBlocksInterval := 10
	for i := 0; i < unsafeBlocksInterval+2; i++ {
		rc.ExpectSyncStatus(ss1, nil)
	}

	elP2pClient := &clientmocks.ElP2PClient{}
	elP2pClient.EXPECT().PeerCount(mock.Anything).Return(healthyElP2pPeerCount, nil)

	monitor := s.SetupMonitor(now, uint64(unsafeBlocksInterval), 60, rc, nil, nil, elP2pClient)
	healthUpdateCh := monitor.Subscribe()

	// once the unsafe interval is surpassed, we should expect "unsafe head is falling behind the unsafe interval"
	for i := 0; i < unsafeBlocksInterval+2; i++ {
		healthFailure := <-healthUpdateCh
		if i <= unsafeBlocksInterval {
			s.Nil(healthFailure)
			s.Equal(now, monitor.lastSeenUnsafeTime)
			s.Equal(uint64(5), monitor.lastSeenUnsafeNum)
		} else {
			s.NotNil(healthFailure)
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

	monitor := s.SetupMonitor(now, 60, 3, rc, nil, nil, nil)
	healthUpdateCh := monitor.Subscribe()

	for i := 0; i < 5; i++ {
		healthFailure := <-healthUpdateCh
		if i < 4 {
			s.Nil(healthFailure)
		} else {
			s.NotNil(healthFailure)
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

	elP2pClient := &clientmocks.ElP2PClient{}
	elP2pClient.EXPECT().PeerCount(mock.Anything).Return(healthyElP2pPeerCount, nil)

	rc := &testutils.MockRollupClient{}
	// although unsafe has lag of 20 seconds, it's within the configured unsafe interval
	// and it is advancing every block time, so it should be considered safe.
	rc.ExpectSyncStatus(mockSyncStatus(now-10, 1, now, 1), nil)
	rc.ExpectSyncStatus(mockSyncStatus(now-10, 1, now, 1), nil)
	rc.ExpectSyncStatus(mockSyncStatus(now-8, 2, now, 1), nil)
	// in this case now time is behind unsafe head time, this should still be considered healthy.
	rc.ExpectSyncStatus(mockSyncStatus(now+5, 2, now, 1), nil)

	monitor := s.SetupMonitor(now, 60, 60, rc, nil, nil, elP2pClient)
	healthUpdateCh := monitor.Subscribe()

	// confirm initial state
	s.Zero(monitor.lastSeenUnsafeNum)
	s.Zero(monitor.lastSeenUnsafeTime)

	// confirm state after first check
	healthFailure := <-healthUpdateCh
	s.Nil(healthFailure)
	lastSeenUnsafeTime := monitor.lastSeenUnsafeTime
	s.NotZero(monitor.lastSeenUnsafeTime)
	s.Equal(uint64(1), monitor.lastSeenUnsafeNum)

	healthFailure = <-healthUpdateCh
	s.Nil(healthFailure)
	s.Equal(lastSeenUnsafeTime, monitor.lastSeenUnsafeTime)
	s.Equal(uint64(1), monitor.lastSeenUnsafeNum)

	healthFailure = <-healthUpdateCh
	s.Nil(healthFailure)
	s.Equal(lastSeenUnsafeTime+2, monitor.lastSeenUnsafeTime)
	s.Equal(uint64(2), monitor.lastSeenUnsafeNum)

	healthFailure = <-healthUpdateCh
	s.Nil(healthFailure)
	s.Equal(lastSeenUnsafeTime+2, monitor.lastSeenUnsafeTime)
	s.Equal(uint64(2), monitor.lastSeenUnsafeNum)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestHealthySupervisor() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)
	rc.ExpectSyncStatus(ss1, nil)
	rc.ExpectSyncStatus(ss1, nil)

	su := &mocks.SupervisorHealthAPI{}
	su.EXPECT().SyncStatus(mock.Anything).Return(eth.SupervisorSyncStatus{}, nil).Times(1)

	monitor := s.SetupMonitor(now, 60, 60, rc, nil, su, nil)

	healthUpdateCh := monitor.Subscribe()
	healthFailure := <-healthUpdateCh
	s.Nil(healthFailure)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestUnhealthySupervisorConnectionDown() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)
	rc.ExpectSyncStatus(ss1, nil)
	rc.ExpectSyncStatus(ss1, nil)

	su := &mocks.SupervisorHealthAPI{}
	su.EXPECT().SyncStatus(mock.Anything).Return(eth.SupervisorSyncStatus{}, errors.New("supervisor connection down")).Times(1)

	monitor := s.SetupMonitor(now, 60, 60, rc, nil, su, nil)

	healthUpdateCh := monitor.Subscribe()
	healthFailure := <-healthUpdateCh
	s.NotNil(healthFailure)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestRollupBoostConnectionDown() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)
	rc.ExpectSyncStatus(ss1, nil)

	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	rbChecker := &clientmocks.RollupBoostHealthChecker{}
	rbChecker.EXPECT().Healthcheck(mock.Anything).Return(client.HealthStatus(""), errors.New("connection refused"))

	monitor := s.SetupMonitorWithRollupBoost(now, 60, 60, rc, pc, rbChecker, nil)

	healthUpdateCh := monitor.Subscribe()
	healthFailure := <-healthUpdateCh
	s.Equal(ErrRollupBoostConnectionDown, healthFailure)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestRollupBoostNotHealthy() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)
	rc.ExpectSyncStatus(ss1, nil)

	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	rbChecker := &clientmocks.RollupBoostHealthChecker{}
	rbChecker.EXPECT().Healthcheck(mock.Anything).Return(client.HealthStatusUnhealthy, nil)

	monitor := s.SetupMonitorWithRollupBoost(now, 60, 60, rc, pc, rbChecker, nil)

	healthUpdateCh := monitor.Subscribe()
	healthFailure := <-healthUpdateCh
	s.Equal(ErrRollupBoostNotHealthy, healthFailure)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestRollupBoostPartialStatus() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)
	rc.ExpectSyncStatus(ss1, nil)

	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	rbChecker := &clientmocks.RollupBoostHealthChecker{}
	rbChecker.EXPECT().Healthcheck(mock.Anything).Return(client.HealthStatusPartial, nil)

	monitor := s.SetupMonitorWithRollupBoost(now, 60, 60, rc, pc, rbChecker, nil)

	healthUpdateCh := monitor.Subscribe()
	healthFailure := <-healthUpdateCh
	s.Equal(ErrRollupBoostPartiallyHealthy, healthFailure)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestRollupBoostPartialStatusWithTolerance() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)

	for i := 0; i < 6; i++ {
		rc.ExpectSyncStatus(ss1, nil)
	}

	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	rbChecker := &clientmocks.RollupBoostHealthChecker{}
	rbChecker.EXPECT().Healthcheck(mock.Anything).Return(client.HealthStatusPartial, nil)

	toleranceLimit := uint64(2)
	toleranceIntervalSeconds := uint64(6)

	timeBoundedRotatingCounter, err := NewTimeBoundedRotatingCounter(toleranceIntervalSeconds)
	s.Nil(err)

	tp := &timeProvider{now: 1758792282}

	monitor := s.SetupMonitorWithRollupBoost(now, 60, 60, rc, pc, rbChecker, nil, func(shm *SequencerHealthMonitor) {
		timeBoundedRotatingCounter.timeProviderFn = tp.Now

		for i := 0; i < 999; i++ {
			timeBoundedRotatingCounter.temporalCache[int64(i)] = uint64(1)
		}

		shm.rollupBoostPartialHealthinessToleranceCounter = timeBoundedRotatingCounter
		shm.rollupBoostPartialHealthinessToleranceLimit = toleranceLimit
	})

	healthUpdateCh := monitor.Subscribe()

	s.Eventually(func() bool {
		return len(timeBoundedRotatingCounter.temporalCache) == 1000
	}, time.Second*3, time.Second*1)

	firstHealthStatus := <-healthUpdateCh
	secondHealthStatus := <-healthUpdateCh
	thirdHealthStatus := <-healthUpdateCh

	s.Nil(firstHealthStatus)
	s.Nil(secondHealthStatus)
	s.Equal(ErrRollupBoostPartiallyHealthy, thirdHealthStatus)

	tp.Now()

	fourthHealthStatus := <-healthUpdateCh
	fifthHealthStatus := <-healthUpdateCh
	sixthHealthStatus := <-healthUpdateCh

	s.Nil(fourthHealthStatus)
	s.Nil(fifthHealthStatus)
	s.Equal(ErrRollupBoostPartiallyHealthy, sixthHealthStatus)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestRollupBoostHealthy() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())
	numSecondsToWait := interval + 1

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)

	for i := 0; i < numSecondsToWait; i++ {
		rc.ExpectSyncStatus(ss1, nil)
	}

	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	rbChecker := &clientmocks.RollupBoostHealthChecker{}
	rbChecker.EXPECT().Healthcheck(mock.Anything).After(time.Duration(numSecondsToWait)*time.Second).Return(client.HealthStatusHealthy, nil)

	monitor := s.SetupMonitorWithRollupBoost(now, 60, 60, rc, pc, rbChecker, nil)

	healthUpdateCh := monitor.Subscribe()
	healthStatus := <-healthUpdateCh
	s.Nil(healthStatus)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestRollupBoostNilClient() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)
	rc.ExpectSyncStatus(ss1, nil)

	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	// No rollup boost health checker configured
	monitor := s.SetupMonitorWithRollupBoost(now, 60, 60, rc, pc, nil, nil)

	healthUpdateCh := monitor.Subscribe()
	healthStatus := <-healthUpdateCh
	s.Nil(healthStatus, "Health check should succeed with nil rollup boost health checker")

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestElP2pHealthy() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())
	numSecondsToWait := interval + 1

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)

	for i := 0; i < numSecondsToWait; i++ {
		rc.ExpectSyncStatus(ss1, nil)
	}

	rbChecker := &clientmocks.RollupBoostHealthChecker{}
	rbChecker.EXPECT().Healthcheck(mock.Anything).After(time.Duration(numSecondsToWait)*time.Second).Return(client.HealthStatusHealthy, nil)

	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	elP2pClient := &clientmocks.ElP2PClient{}
	elP2pClient.EXPECT().PeerCount(mock.Anything).Return(healthyElP2pPeerCount, nil)

	monitor := s.SetupMonitorWithRollupBoost(now, 60, 60, rc, pc, rbChecker, elP2pClient)

	healthUpdateCh := monitor.Subscribe()
	healthStatus := <-healthUpdateCh
	s.Nil(healthStatus)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestElP2pHealthyNilClient() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())
	numSecondsToWait := interval + 1

	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)

	for i := 0; i < numSecondsToWait; i++ {
		rc.ExpectSyncStatus(ss1, nil)
	}

	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	monitor := s.SetupMonitorWithRollupBoost(now, 60, 60, rc, pc, nil, nil)

	healthUpdateCh := monitor.Subscribe()
	healthStatus := <-healthUpdateCh
	s.Nil(healthStatus)

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
