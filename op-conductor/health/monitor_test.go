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
	unhealthyPeerCount = 0
	minPeerCount       = 1
	healthyPeerCount   = 2
	blockTime          = 2
	interval           = 1
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
	s.interval = interval
	s.minPeerCount = minPeerCount
	s.rollupCfg = &rollup.Config{
		BlockTime: blockTime,
	}
}

func (s *HealthMonitorTestSuite) SetupMonitor(
	now, unsafeInterval, safeInterval uint64,
	mockRollupClient *testutils.MockRollupClient,
	mockP2P *p2pMocks.API,
	mockSupervisorHealthAPI SupervisorHealthAPI,
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
	err := monitor.Start(context.Background())
	s.NoError(err)
	return monitor
}

// SetupMonitorWithRollupBoost creates a HealthMonitor that includes a RollupBoostClient
func (s *HealthMonitorTestSuite) SetupMonitorWithRollupBoost(
	now, unsafeInterval, safeInterval uint64,
	mockRollupClient *testutils.MockRollupClient,
	mockP2P *p2pMocks.API,
	mockRollupBoost *clientmocks.RollupBoostClient,
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
		rb:             mockRollupBoost,
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

	monitor := s.SetupMonitor(now, 60, 60, rc, pc, nil)

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

	monitor := s.SetupMonitor(now, uint64(unsafeBlocksInterval), 60, rc, nil, nil)
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

	monitor := s.SetupMonitor(now, 60, 3, rc, nil, nil)
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

	rc := &testutils.MockRollupClient{}
	// although unsafe has lag of 20 seconds, it's within the configured unsafe interval
	// and it is advancing every block time, so it should be considered safe.
	rc.ExpectSyncStatus(mockSyncStatus(now-10, 1, now, 1), nil)
	rc.ExpectSyncStatus(mockSyncStatus(now-10, 1, now, 1), nil)
	rc.ExpectSyncStatus(mockSyncStatus(now-8, 2, now, 1), nil)
	// in this case now time is behind unsafe head time, this should still be considered healthy.
	rc.ExpectSyncStatus(mockSyncStatus(now+5, 2, now, 1), nil)

	monitor := s.SetupMonitor(now, 60, 60, rc, nil, nil)
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

	monitor := s.SetupMonitor(now, 60, 60, rc, nil, su)

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

	monitor := s.SetupMonitor(now, 60, 60, rc, nil, su)

	healthUpdateCh := monitor.Subscribe()
	healthFailure := <-healthUpdateCh
	s.NotNil(healthFailure)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestRollupBoostConnectionDown() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	// Setup healthy node conditions
	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)
	rc.ExpectSyncStatus(ss1, nil)

	// Setup healthy peer count
	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	// Setup rollup boost connection failure
	rb := &clientmocks.RollupBoostClient{}
	rb.EXPECT().Healthcheck(mock.Anything).Return(client.HealthStatus(""), errors.New("connection refused"))

	// Start monitor with all dependencies
	monitor := s.SetupMonitorWithRollupBoost(now, 60, 60, rc, pc, rb)

	// Check for connection down error
	healthUpdateCh := monitor.Subscribe()
	healthFailure := <-healthUpdateCh
	s.Equal(ErrRollupBoostConnectionDown, healthFailure)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestRollupBoostNotHealthy() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	// Setup healthy node conditions
	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)
	rc.ExpectSyncStatus(ss1, nil)

	// Setup healthy peer count
	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	// Setup unhealthy rollup boost
	rb := &clientmocks.RollupBoostClient{}
	rb.EXPECT().Healthcheck(mock.Anything).Return(client.HealthStatusUnhealthy, nil)

	// Start monitor with all dependencies
	monitor := s.SetupMonitorWithRollupBoost(now, 60, 60, rc, pc, rb)

	// Check for unhealthy status
	healthUpdateCh := monitor.Subscribe()
	healthFailure := <-healthUpdateCh
	s.Equal(ErrRollupBoostNotHealthy, healthFailure)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestRollupBoostPartialStatus() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	// Setup healthy node conditions
	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)
	rc.ExpectSyncStatus(ss1, nil)

	// Setup healthy peer count
	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	// Setup partial rollup boost status (treated as unhealthy)
	rb := &clientmocks.RollupBoostClient{}
	rb.EXPECT().Healthcheck(mock.Anything).Return(client.HealthStatusPartial, nil)

	// Start monitor with all dependencies
	monitor := s.SetupMonitorWithRollupBoost(now, 60, 60, rc, pc, rb)

	// Check for unhealthy status
	healthUpdateCh := monitor.Subscribe()
	healthFailure := <-healthUpdateCh
	s.Equal(ErrRollupBoostPartiallyHealthy, healthFailure)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestRollupBoostHealthy() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())
	numSecondsToWait := interval + 1

	// Setup healthy node conditions
	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)

	for i := 0; i < numSecondsToWait; i++ {
		rc.ExpectSyncStatus(ss1, nil)
	}

	// Setup healthy peer count
	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	// Setup healthy rollup boost
	rb := &clientmocks.RollupBoostClient{}
	// // Wait for longer than healthcheck interval before returning healthy status, to verify nothing breaks if rb is slow to respond
	rb.EXPECT().Healthcheck(mock.Anything).After(time.Duration(numSecondsToWait)*time.Second).Return(client.HealthStatusHealthy, nil)

	// Start monitor with all dependencies
	monitor := s.SetupMonitorWithRollupBoost(now, 60, 60, rc, pc, rb)

	// Should report healthy status
	healthUpdateCh := monitor.Subscribe()
	healthStatus := <-healthUpdateCh
	s.Nil(healthStatus)

	s.NoError(monitor.Stop())
}

func (s *HealthMonitorTestSuite) TestRollupBoostNilClient() {
	s.T().Parallel()
	now := uint64(time.Now().Unix())

	// Setup healthy node conditions
	rc := &testutils.MockRollupClient{}
	ss1 := mockSyncStatus(now-1, 1, now-3, 0)
	rc.ExpectSyncStatus(ss1, nil)

	// Setup healthy peer count
	pc := &p2pMocks.API{}
	ps1 := &p2p.PeerStats{
		Connected: healthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil)

	// Explicitly create a monitor with all other components but nil rollup boost client
	tp := &timeProvider{now: now}
	monitor := &SequencerHealthMonitor{
		log:            s.log,
		interval:       s.interval,
		metrics:        &metrics.NoopMetricsImpl{},
		healthUpdateCh: make(chan error),
		rollupCfg:      s.rollupCfg,
		unsafeInterval: 60,
		safeInterval:   60,
		safeEnabled:    true,
		minPeerCount:   s.minPeerCount,
		timeProviderFn: tp.Now,
		node:           rc,
		p2p:            pc,
		rb:             nil, // Explicitly set to nil
	}

	err := monitor.Start(context.Background())
	s.NoError(err)

	// Health check should succeed even with nil rb
	healthUpdateCh := monitor.Subscribe()
	healthStatus := <-healthUpdateCh
	s.Nil(healthStatus, "Health check should succeed with nil rollup boost client")

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
