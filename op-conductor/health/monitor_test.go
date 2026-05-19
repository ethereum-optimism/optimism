package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ethereum-optimism/optimism/op-conductor/client"
	clientmocks "github.com/ethereum-optimism/optimism/op-conductor/client/mocks"
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

func newSyncStatusMonitor(t *testing.T, now, unsafeInterval, safeInterval uint64, mockRollupClient *testutils.MockRollupClient) (*SequencerHealthMonitor, *timeProvider) {
	t.Helper()
	tp := &timeProvider{now: now}
	monitor := &SequencerHealthMonitor{
		log:            testlog.Logger(t, log.LevelDebug),
		rollupCfg:      &rollup.Config{BlockTime: blockTime},
		unsafeInterval: unsafeInterval,
		safeInterval:   safeInterval,
		safeEnabled:    true,
		timeProviderFn: tp.Now,
		node:           mockRollupClient,
	}
	return monitor, tp
}

func newPeerCountMonitor(t *testing.T, minPeerCount uint64, mockP2P *p2pMocks.API) *SequencerHealthMonitor {
	t.Helper()
	return &SequencerHealthMonitor{
		log:          testlog.Logger(t, log.LevelDebug),
		minPeerCount: minPeerCount,
		p2p:          mockP2P,
	}
}

func TestUnsafeHeadCatchingUpStaysHealthy(t *testing.T) {
	now := uint64(time.Now().Unix())
	rc := &testutils.MockRollupClient{}
	polls := []struct {
		now        uint64
		unsafeTime uint64
		unsafeNum  uint64
	}{
		{now: now, unsafeTime: now - 2, unsafeNum: 5},
		{now: now + 2, unsafeTime: now, unsafeNum: 6},
		{now: now + 22, unsafeTime: now + 2, unsafeNum: 7},
		{now: now + 24, unsafeTime: now + 4, unsafeNum: 8},
		{now: now + 26, unsafeTime: now + 8, unsafeNum: 9},
		{now: now + 28, unsafeTime: now + 14, unsafeNum: 10},
		{now: now + 30, unsafeTime: now + 20, unsafeNum: 11},
		{now: now + 32, unsafeTime: now + 28, unsafeNum: 12},
	}
	for _, poll := range polls {
		rc.ExpectSyncStatus(mockSyncStatus(poll.unsafeTime, poll.unsafeNum, poll.now, poll.unsafeNum), nil)
	}

	monitor, tp := newSyncStatusMonitor(t, now, 10, 60, rc)
	for _, poll := range polls {
		tp.now = poll.now
		require.NoError(t, monitor.checkNodeSyncStatus(context.Background()))
	}
}

func TestStoppedSequencerStillMarkedUnhealthy(t *testing.T) {
	now := uint64(time.Now().Unix())
	rc := &testutils.MockRollupClient{}
	polls := []struct {
		now uint64
		err error
	}{
		{now: now},
		{now: now + 2},
		{now: now + 22},
		{now: now + 24},
		{now: now + 26},
		{now: now + 28},
		{now: now + 30, err: ErrSequencerNotHealthy},
	}
	for _, poll := range polls {
		rc.ExpectSyncStatus(mockSyncStatus(now, 6, poll.now, 6), nil)
	}

	monitor, tp := newSyncStatusMonitor(t, now, 10, 60, rc)
	for _, poll := range polls {
		tp.now = poll.now
		require.Equal(t, poll.err, monitor.checkNodeSyncStatus(context.Background()))
	}
}

func TestLagAboveCeilingMarkedUnhealthy(t *testing.T) {
	now := uint64(time.Now().Unix())
	rc := &testutils.MockRollupClient{}
	polls := []struct {
		now        uint64
		unsafeTime uint64
		unsafeNum  uint64
		err        error
	}{
		{now: now, unsafeTime: now, unsafeNum: 5},
		{now: now + 2, unsafeTime: now + 2, unsafeNum: 6},
		{now: now + 202, unsafeTime: now + 2, unsafeNum: 7},
		{now: now + 204, unsafeTime: now + 14, unsafeNum: 8},
		{now: now + 206, unsafeTime: now + 26, unsafeNum: 9},
		{now: now + 208, unsafeTime: now + 38, unsafeNum: 10},
		{now: now + 210, unsafeTime: now + 50, unsafeNum: 11, err: ErrSequencerNotHealthy},
	}
	for _, poll := range polls {
		rc.ExpectSyncStatus(mockSyncStatus(poll.unsafeTime, poll.unsafeNum, poll.now, poll.unsafeNum), nil)
	}

	monitor, tp := newSyncStatusMonitor(t, now, 10, 60, rc)
	for _, poll := range polls {
		tp.now = poll.now
		require.Equal(t, poll.err, monitor.checkNodeSyncStatus(context.Background()))
	}
}

func TestTransientSyncStatusFailureAfterHealthyPollStaysHealthy(t *testing.T) {
	now := uint64(time.Now().Unix())
	rc := &testutils.MockRollupClient{}
	syncErr := errors.New("optimism_syncStatus unavailable")

	rc.ExpectSyncStatus(mockSyncStatus(now, 5, now, 5), nil)
	for i := uint64(0); i < recoveringWindowSize; i++ {
		rc.ExpectSyncStatus(nil, syncErr)
	}

	monitor, _ := newSyncStatusMonitor(t, now, 10, 60, rc)
	require.NoError(t, monitor.checkNodeSyncStatus(context.Background()))
	for i := uint64(1); i < recoveringWindowSize; i++ {
		require.NoError(t, monitor.checkNodeSyncStatus(context.Background()))
	}
	require.Equal(t, ErrSequencerConnectionDown, monitor.checkNodeSyncStatus(context.Background()))
}

func TestInitialSyncStatusFailureToleratedUntilWindowFails(t *testing.T) {
	now := uint64(time.Now().Unix())
	rc := &testutils.MockRollupClient{}
	for i := uint64(0); i < recoveringWindowSize; i++ {
		rc.ExpectSyncStatus(nil, errors.New("optimism_syncStatus unavailable"))
	}

	monitor, _ := newSyncStatusMonitor(t, now, 10, 60, rc)
	for i := uint64(1); i < recoveringWindowSize; i++ {
		require.NoError(t, monitor.checkNodeSyncStatus(context.Background()))
	}
	require.Equal(t, ErrSequencerConnectionDown, monitor.checkNodeSyncStatus(context.Background()))
}

func TestMixedSyncStatusFailureWindowStaysHealthy(t *testing.T) {
	now := uint64(time.Now().Unix())
	rc := &testutils.MockRollupClient{}
	syncErr := errors.New("optimism_syncStatus unavailable")
	polls := []struct {
		status *eth.SyncStatus
		err    error
	}{
		{err: syncErr},
		{status: mockSyncStatus(now, 5, now, 5)},
		{err: syncErr},
		{status: mockSyncStatus(now, 6, now, 6)},
		{err: syncErr},
		{err: syncErr},
		{status: mockSyncStatus(now, 7, now, 7)},
	}
	for _, poll := range polls {
		rc.ExpectSyncStatus(poll.status, poll.err)
	}

	monitor, _ := newSyncStatusMonitor(t, now, 10, 60, rc)
	for range polls {
		require.NoError(t, monitor.checkNodeSyncStatus(context.Background()))
	}
}

func TestSingleSyncStatusSuccessDoesNotResetWindow(t *testing.T) {
	now := uint64(time.Now().Unix())
	rc := &testutils.MockRollupClient{}
	syncErr := errors.New("optimism_syncStatus unavailable")
	for i := uint64(1); i < recoveringWindowSize; i++ {
		rc.ExpectSyncStatus(nil, syncErr)
	}
	rc.ExpectSyncStatus(mockSyncStatus(now, 5, now, 5), nil)
	for i := uint64(1); i < recoveringWindowSize; i++ {
		rc.ExpectSyncStatus(nil, syncErr)
	}
	for i := uint64(0); i < recoveringWindowSize; i++ {
		rc.ExpectSyncStatus(mockSyncStatus(now, 6+i, now, 6+i), nil)
	}

	monitor, _ := newSyncStatusMonitor(t, now, 10, 60, rc)
	for i := uint64(1); i < recoveringWindowSize; i++ {
		require.NoError(t, monitor.checkNodeSyncStatus(context.Background()))
	}
	require.NoError(t, monitor.checkNodeSyncStatus(context.Background()), "single success after failures should still be inconclusive")
	for i := uint64(1); i < recoveringWindowSize; i++ {
		require.NoError(t, monitor.checkNodeSyncStatus(context.Background()))
	}
	for i := uint64(1); i < recoveringWindowSize; i++ {
		require.NoError(t, monitor.checkNodeSyncStatus(context.Background()), "recovering successes should not immediately restore Success state")
	}
	require.NoError(t, monitor.checkNodeSyncStatus(context.Background()), "full window of successes should restore Success state")
}

func TestPollsInRecoveryNotResetOnSecondRegression(t *testing.T) {
	now := uint64(time.Now().Unix())
	rc := &testutils.MockRollupClient{}
	polls := []struct {
		now        uint64
		unsafeTime uint64
		unsafeNum  uint64
		err        error
	}{
		{now: now, unsafeTime: now, unsafeNum: 5},
		{now: now + 2, unsafeTime: now + 2, unsafeNum: 6},
		{now: now + 17, unsafeTime: now + 2, unsafeNum: 7},
		{now: now + 19, unsafeTime: now + 7, unsafeNum: 8},
		{now: now + 21, unsafeTime: now + 3, unsafeNum: 9},
		{now: now + 23, unsafeTime: now + 3, unsafeNum: 9},
		{now: now + 25, unsafeTime: now + 3, unsafeNum: 9},
		{now: now + 27, unsafeTime: now + 3, unsafeNum: 9, err: ErrSequencerNotHealthy},
	}
	for _, poll := range polls {
		rc.ExpectSyncStatus(mockSyncStatus(poll.unsafeTime, poll.unsafeNum, poll.now, poll.unsafeNum), nil)
	}

	monitor, tp := newSyncStatusMonitor(t, now, 10, 60, rc)
	for _, poll := range polls {
		tp.now = poll.now
		require.Equal(t, poll.err, monitor.checkNodeSyncStatus(context.Background()))
	}
	require.Equal(t, uint64(15), monitor.initialLagInRecovery)
	require.Equal(t, uint64(12), monitor.recoveryWindowStartLag)
	require.Equal(t, uint64(6), monitor.pollsInRecovery)
	require.Equal(t, uint64(recoveringWindowSize), monitor.pollsInRecoveryWindow)
}

func TestUnsafeHeadRecoveryPlateauAboveUnsafeIntervalMarkedUnhealthy(t *testing.T) {
	now := uint64(time.Now().Unix())
	rc := &testutils.MockRollupClient{}
	polls := []struct {
		now       uint64
		unsafeLag uint64
		unsafeNum uint64
		err       error
	}{
		{now: now, unsafeLag: 25, unsafeNum: 5},
		{now: now + 2, unsafeLag: 24, unsafeNum: 6},
		{now: now + 4, unsafeLag: 24, unsafeNum: 7},
		{now: now + 6, unsafeLag: 24, unsafeNum: 8},
		{now: now + 8, unsafeLag: 24, unsafeNum: 9},
		{now: now + 10, unsafeLag: 24, unsafeNum: 10, err: ErrSequencerNotHealthy},
	}
	for _, poll := range polls {
		rc.ExpectSyncStatus(mockSyncStatus(poll.now-poll.unsafeLag, poll.unsafeNum, poll.now, poll.unsafeNum), nil)
	}

	monitor, tp := newSyncStatusMonitor(t, now, 10, 60, rc)
	for _, poll := range polls {
		tp.now = poll.now
		require.Equal(t, poll.err, monitor.checkNodeSyncStatus(context.Background()))
	}
}

func (s *HealthMonitorTestSuite) TestUnhealthyLowPeerCount() {
	s.T().Parallel()

	pc := &p2pMocks.API{}
	ps1 := &apis.PeerStats{
		Connected: unhealthyPeerCount,
	}
	pc.EXPECT().PeerStats(mock.Anything).Return(ps1, nil).Times(recoveringWindowSize)

	monitor := newPeerCountMonitor(s.T(), s.minPeerCount, pc)
	for i := uint64(1); i < recoveringWindowSize; i++ {
		s.NoError(monitor.checkNodePeerCount(context.Background()))
	}
	s.Equal(ErrSequencerNotHealthy, monitor.checkNodePeerCount(context.Background()))
	pc.AssertExpectations(s.T())
}

func TestPeerStatsRPCErrorWindow(t *testing.T) {
	pc := &p2pMocks.API{}
	pc.EXPECT().PeerStats(mock.Anything).Return(nil, errors.New("p2p unavailable")).Times(recoveringWindowSize)

	monitor := newPeerCountMonitor(t, minPeerCount, pc)
	for i := uint64(1); i < recoveringWindowSize; i++ {
		require.NoError(t, monitor.checkNodePeerCount(context.Background()))
	}
	require.Equal(t, ErrSequencerConnectionDown, monitor.checkNodePeerCount(context.Background()))
	pc.AssertExpectations(t)
}

func TestPeerCountMixedSuccessFailureWindowStaysHealthy(t *testing.T) {
	pc := &p2pMocks.API{}
	pc.EXPECT().PeerStats(mock.Anything).Return(nil, errors.New("p2p unavailable")).Once()
	pc.EXPECT().PeerStats(mock.Anything).Return(&apis.PeerStats{Connected: healthyPeerCount}, nil).Once()
	pc.EXPECT().PeerStats(mock.Anything).Return(&apis.PeerStats{Connected: unhealthyPeerCount}, nil).Once()
	pc.EXPECT().PeerStats(mock.Anything).Return(&apis.PeerStats{Connected: healthyPeerCount}, nil).Once()
	pc.EXPECT().PeerStats(mock.Anything).Return(nil, errors.New("p2p unavailable")).Once()

	monitor := newPeerCountMonitor(t, minPeerCount, pc)
	for i := uint64(0); i < recoveringWindowSize; i++ {
		require.NoError(t, monitor.checkNodePeerCount(context.Background()))
	}
	pc.AssertExpectations(t)
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

	monitor := s.SetupMonitor(now, 60, 60, rc, healthyPc, elP2pClient)

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
	unsafeBlocksInterval := uint64(10)
	for i := uint64(0); i < unsafeBlocksInterval+recoveringWindowSize+1; i++ {
		rc.ExpectSyncStatus(ss1, nil)
	}

	monitor, tp := newSyncStatusMonitor(s.T(), now, unsafeBlocksInterval, 60, rc)
	for i := uint64(0); i < unsafeBlocksInterval+recoveringWindowSize+1; i++ {
		tp.now = now + i
		healthFailure := monitor.checkNodeSyncStatus(context.Background())
		if i < unsafeBlocksInterval+recoveringWindowSize {
			s.Nil(healthFailure)
			s.Equal(now, monitor.lastSeenUnsafeTime)
			s.Equal(uint64(5), monitor.lastSeenUnsafeNum)
		} else {
			s.Equal(ErrSequencerNotHealthy, healthFailure)
		}
	}
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

	monitor := s.SetupMonitor(now, 60, 60, rc, nil, elP2pClient)
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
