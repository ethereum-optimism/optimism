package conductor

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	clientmocks "github.com/ethereum-optimism/optimism/op-conductor/client/mocks"
	consensusmocks "github.com/ethereum-optimism/optimism/op-conductor/consensus/mocks"
	"github.com/ethereum-optimism/optimism/op-conductor/health"
	healthmocks "github.com/ethereum-optimism/optimism/op-conductor/health/mocks"
	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func mockConfig(t *testing.T) Config {
	now := uint64(time.Now().Unix())
	return Config{
		ConsensusAddr:  "127.0.0.1",
		ConsensusPort:  0,
		RaftServerID:   "SequencerA",
		RaftStorageDir: "/tmp/raft",
		RaftBootstrap:  false,
		NodeRPC:        "http://node:8545",
		ExecutionRPC:   "http://geth:8545",
		Paused:         false,
		HealthCheck: HealthCheckConfig{
			Interval:       1,
			UnsafeInterval: 3,
			SafeInterval:   5,
			MinPeerCount:   1,
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
			ChannelTimeoutBedrock:   300,
			L1ChainID:               big.NewInt(1),
			L2ChainID:               big.NewInt(2),
			RegolithTime:            &now,
			CanyonTime:              &now,
			BatchInboxAddress:       [20]byte{1, 2},
			DepositContractAddress:  [20]byte{2, 3},
			L1SystemConfigAddress:   [20]byte{3, 4},
			ProtocolVersionsAddress: [20]byte{4, 5},
		},
		RPCEnableProxy: false,
	}
}

type OpConductorTestSuite struct {
	suite.Suite

	conductor      *OpConductor
	healthUpdateCh chan error
	leaderUpdateCh chan bool

	ctx     context.Context
	err     error
	log     log.Logger
	cfg     Config
	metrics metrics.Metricer
	version string
	ctrl    *clientmocks.SequencerControl
	cons    *consensusmocks.Consensus
	hmon    *healthmocks.HealthMonitor

	syncEnabled bool           // syncEnabled controls whether synchronization is enabled for test actions.
	next        chan struct{}  // next is used to signal when the next action in the test can proceed.
	wg          sync.WaitGroup // wg ensures that test actions are completed before moving on.
}

func (s *OpConductorTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.log = testlog.Logger(s.T(), log.LevelDebug)
	s.metrics = &metrics.NoopMetricsImpl{}
	s.cfg = mockConfig(s.T())
	s.version = "v0.0.1"
	s.next = make(chan struct{})
}

func (s *OpConductorTestSuite) SetupTest() {
	// initialize for every test so that method call count starts from 0
	s.ctrl = &clientmocks.SequencerControl{}
	s.cons = &consensusmocks.Consensus{}
	s.hmon = &healthmocks.HealthMonitor{}
	s.cons.EXPECT().ServerID().Return("SequencerA")

	conductor, err := NewOpConductor(s.ctx, &s.cfg, s.log, s.metrics, s.version, s.ctrl, s.cons, s.hmon)
	s.NoError(err)
	conductor.retryBackoff = func() time.Duration { return 0 } // disable retry backoff for tests
	s.conductor = conductor

	s.healthUpdateCh = make(chan error, 1)
	s.hmon.EXPECT().Start(mock.Anything).Return(nil)
	s.conductor.healthUpdateCh = s.healthUpdateCh

	s.leaderUpdateCh = make(chan bool, 1)
	s.conductor.leaderUpdateCh = s.leaderUpdateCh

	s.err = errors.New("error")
	s.syncEnabled = false   // default to no sync, turn it on by calling s.enableSynchronization()
	s.wg = sync.WaitGroup{} // create new wg for every test in case last test didn't finish the action loop during shutdown.
}

func (s *OpConductorTestSuite) TearDownTest() {
	s.hmon.EXPECT().Stop().Return(nil)
	s.cons.EXPECT().Shutdown().Return(nil)

	if s.syncEnabled {
		s.wg.Add(1)
		s.next <- struct{}{}
	}
	s.NoError(s.conductor.Stop(s.ctx))
	s.True(s.conductor.Stopped())
}

func (s *OpConductorTestSuite) startConductor() {
	err := s.conductor.Start(s.ctx)
	s.NoError(err)
	s.False(s.conductor.Stopped())
}

// enableSynchronization wraps conductor actionFn with extra synchronization logic
// so that we could control the execution of actionFn and observe the internal state transition in between.
func (s *OpConductorTestSuite) enableSynchronization() {
	s.syncEnabled = true
	s.conductor.loopActionFn = func() {
		<-s.next
		s.conductor.loopAction()
		s.wg.Done()
	}
	s.startConductor()
	s.executeAction()
}

func (s *OpConductorTestSuite) disableSynchronization() {
	s.syncEnabled = false
	s.startConductor()
}

func (s *OpConductorTestSuite) execute(fn func()) {
	s.wg.Add(1)
	if fn != nil {
		fn()
	}
	s.next <- struct{}{}
	s.wg.Wait()
}

func updateStatusAndExecuteAction[T any](s *OpConductorTestSuite, ch chan T, status T) {
	fn := func() {
		ch <- status
	}
	s.execute(fn) // this executes status update
	s.executeAction()
}

func (s *OpConductorTestSuite) updateLeaderStatusAndExecuteAction(status bool) {
	updateStatusAndExecuteAction(s, s.leaderUpdateCh, status)
}

func (s *OpConductorTestSuite) updateHealthStatusAndExecuteAction(status error) {
	updateStatusAndExecuteAction(s, s.healthUpdateCh, status)
}

func (s *OpConductorTestSuite) executeAction() {
	s.execute(nil)
}

// Scenario 1: pause -> resume -> stop
func (s *OpConductorTestSuite) TestControlLoop1() {
	s.disableSynchronization()

	// Pause
	err := s.conductor.Pause(s.ctx)
	s.NoError(err)
	s.True(s.conductor.Paused())

	// Send health update, make sure it can still be consumed.
	s.healthUpdateCh <- nil
	s.healthUpdateCh <- nil

	// Resume
	s.ctrl.EXPECT().SequencerActive(mock.Anything).Return(false, nil)
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
	s.disableSynchronization()

	// Pause
	err := s.conductor.Pause(s.ctx)
	s.NoError(err)
	s.True(s.conductor.Paused())

	// Pause again, this shouldn't block or cause any other issues
	err = s.conductor.Pause(s.ctx)
	s.NoError(err)
	s.True(s.conductor.Paused())

	// Resume
	s.ctrl.EXPECT().SequencerActive(mock.Anything).Return(false, nil)
	err = s.conductor.Resume(s.ctx)
	s.NoError(err)
	s.False(s.conductor.Paused())

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

// Scenario 3: pause -> stop
func (s *OpConductorTestSuite) TestControlLoop3() {
	s.disableSynchronization()

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

// In this test, we have a follower that is not healthy and not sequencing, it becomes leader through election.
// But since it does not have the same unsafe head as in consensus. We expect it to transfer leadership to another node.
// [follower, not healthy, not sequencing] -- become leader --> [leader, not healthy, not sequencing] -- transfer leadership --> [follower, not healthy, not sequencing]
func (s *OpConductorTestSuite) TestScenario1() {
	s.enableSynchronization()

	// set initial state
	s.conductor.leader.Store(false)
	s.conductor.healthy.Store(false)
	s.conductor.seqActive.Store(false)
	s.conductor.hcerr = health.ErrSequencerNotHealthy
	s.conductor.prevState = &state{
		leader:  false,
		healthy: false,
		active:  false,
	}

	// unsafe in consensus is different than unsafe in node.
	mockPayload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: 3,
			BlockHash:   [32]byte{4, 5, 6},
		},
	}
	mockBlockInfo := &testutils.MockBlockInfo{
		InfoNum:  1,
		InfoHash: [32]byte{1, 2, 3},
	}
	s.cons.EXPECT().TransferLeader().Return(nil)
	s.cons.EXPECT().LatestUnsafePayload().Return(mockPayload, nil).Times(1)
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).Return(mockBlockInfo, nil).Times(1)

	// become leader
	s.updateLeaderStatusAndExecuteAction(true)

	// expect to transfer leadership, go back to [follower, not healthy, not sequencing]
	s.False(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
	s.Equal(health.ErrSequencerNotHealthy, s.conductor.hcerr)
	s.Equal(&state{
		leader:  true,
		healthy: false,
		active:  false,
	}, s.conductor.prevState)
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 1)
}

// In this test, we have a follower that is not healthy and not sequencing, it becomes leader through election.
// But since it fails to compare the unsafe head to the value stored in consensus, we expect it to transfer leadership to another node.
// [follower, not healthy, not sequencing] -- become leader --> [leader, not healthy, not sequencing] -- transfer leadership --> [follower, not healthy, not sequencing]
func (s *OpConductorTestSuite) TestScenario1Err() {
	s.enableSynchronization()

	// set initial state
	s.conductor.leader.Store(false)
	s.conductor.healthy.Store(false)
	s.conductor.seqActive.Store(false)
	s.conductor.hcerr = health.ErrSequencerNotHealthy
	s.conductor.prevState = &state{
		leader:  false,
		healthy: false,
		active:  false,
	}

	s.cons.EXPECT().LatestUnsafePayload().Return(nil, errors.New("fake connection error")).Times(1)
	s.cons.EXPECT().TransferLeader().Return(nil)

	// become leader
	s.updateLeaderStatusAndExecuteAction(true)

	// expect to transfer leadership, go back to [follower, not healthy, not sequencing]
	s.False(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
	s.Equal(health.ErrSequencerNotHealthy, s.conductor.hcerr)
	s.Equal(&state{
		leader:  true,
		healthy: false,
		active:  false,
	}, s.conductor.prevState)
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 1)
}

// In this test, we have a follower that is not healthy and not sequencing. it becomes healthy and we expect it to stay as follower and not start sequencing.
// [follower, not healthy, not sequencing] -- become healthy --> [follower, healthy, not sequencing]
func (s *OpConductorTestSuite) TestScenario2() {
	s.enableSynchronization()

	// set initial state
	s.conductor.leader.Store(false)
	s.conductor.healthy.Store(false)
	s.conductor.seqActive.Store(false)

	// become healthy
	s.updateHealthStatusAndExecuteAction(nil)

	// expect to stay as follower, go to [follower, healthy, not sequencing]
	s.False(s.conductor.leader.Load())
	s.True(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
}

// In this test, we have a follower that is healthy and not sequencing, we send a leader update to it and expect it to start sequencing.
// [follower, healthy, not sequencing] -- become leader --> [leader, healthy, sequencing]
func (s *OpConductorTestSuite) TestScenario3() {
	s.enableSynchronization()

	mockPayload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: 1,
			Timestamp:   hexutil.Uint64(time.Now().Unix()),
			BlockHash:   [32]byte{1, 2, 3},
		},
	}

	mockBlockInfo := &testutils.MockBlockInfo{
		InfoNum:  1,
		InfoHash: [32]byte{1, 2, 3},
	}
	s.cons.EXPECT().LatestUnsafePayload().Return(mockPayload, nil).Times(1)
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).Return(mockBlockInfo, nil).Times(1)
	s.ctrl.EXPECT().StartSequencer(mock.Anything, mockPayload.ExecutionPayload.BlockHash).Return(nil).Times(1)

	// [follower, healthy, not sequencing]
	s.False(s.conductor.leader.Load())
	s.True(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())

	// become leader
	s.updateLeaderStatusAndExecuteAction(true)

	// [leader, healthy, sequencing]
	s.True(s.conductor.leader.Load())
	s.True(s.conductor.healthy.Load())
	s.True(s.conductor.seqActive.Load())
	s.ctrl.AssertCalled(s.T(), "StartSequencer", mock.Anything, mock.Anything)
	s.ctrl.AssertCalled(s.T(), "LatestUnsafeBlock", mock.Anything)
}

// This test setup is the same as Scenario 3, the difference is that scenario 3 is all happy case and in this test, we try to exhaust all the error cases.
// [follower, healthy, not sequencing] -- become leader, unsafe head does not match, retry, eventually succeed --> [leader, healthy, sequencing]
func (s *OpConductorTestSuite) TestScenario4() {
	s.enableSynchronization()

	// unsafe in consensus is 1 block ahead of unsafe in sequencer, we try to post the unsafe payload to sequencer and return error to allow retry
	// this is normal because the latest unsafe (in consensus) might not arrive at sequencer through p2p yet
	mockPayload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: 2,
			Timestamp:   hexutil.Uint64(time.Now().Unix()),
			BlockHash:   [32]byte{1, 2, 3},
		},
	}

	mockBlockInfo := &testutils.MockBlockInfo{
		InfoNum:  1,
		InfoHash: [32]byte{2, 3, 4},
	}
	s.cons.EXPECT().LatestUnsafePayload().Return(mockPayload, nil).Times(1)
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).Return(mockBlockInfo, nil).Times(1)
	s.ctrl.EXPECT().PostUnsafePayload(mock.Anything, mockPayload).Return(errors.New("simulated PostUnsafePayload failure")).Times(1)
	s.ctrl.EXPECT().StartSequencer(mock.Anything, mockBlockInfo.InfoHash).Return(nil).Times(1)

	s.updateLeaderStatusAndExecuteAction(true)

	// [leader, healthy, not sequencing]
	s.True(s.conductor.leader.Load())
	s.True(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
	s.cons.AssertNumberOfCalls(s.T(), "LatestUnsafePayload", 1)
	s.ctrl.AssertNumberOfCalls(s.T(), "LatestUnsafeBlock", 1)
	s.ctrl.AssertNumberOfCalls(s.T(), "PostUnsafePayload", 1)
	s.ctrl.AssertNotCalled(s.T(), "StartSequencer", mock.Anything, mock.Anything)

	s.cons.EXPECT().LatestUnsafePayload().Return(mockPayload, nil).Times(1)
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).Return(mockBlockInfo, nil).Times(1)
	s.ctrl.EXPECT().PostUnsafePayload(mock.Anything, mockPayload).Return(nil).Times(1)
	s.ctrl.EXPECT().StartSequencer(mock.Anything, mockPayload.ExecutionPayload.BlockHash).Return(nil).Times(1)

	s.executeAction()

	// [leader, healthy, sequencing]
	s.True(s.conductor.leader.Load())
	s.True(s.conductor.healthy.Load())
	s.True(s.conductor.seqActive.Load())
	s.cons.AssertNumberOfCalls(s.T(), "LatestUnsafePayload", 2)
	s.ctrl.AssertNumberOfCalls(s.T(), "LatestUnsafeBlock", 2)
	s.ctrl.AssertNumberOfCalls(s.T(), "PostUnsafePayload", 2)
	s.ctrl.AssertNumberOfCalls(s.T(), "StartSequencer", 1)
}

// In this test, we have a follower that is healthy and not sequencing, we send a unhealthy update to it and expect it to stay as follower and not start sequencing.
// [follower, healthy, not sequencing] -- become unhealthy --> [follower, not healthy, not sequencing]
func (s *OpConductorTestSuite) TestScenario5() {
	s.enableSynchronization()

	// set initial state
	s.conductor.leader.Store(false)
	s.conductor.healthy.Store(true)
	s.conductor.seqActive.Store(false)

	// become unhealthy
	s.updateHealthStatusAndExecuteAction(health.ErrSequencerNotHealthy)

	// expect to stay as follower, go to [follower, not healthy, not sequencing]
	s.False(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
}

// In this test, we have a leader that is healthy and sequencing, we send a leader update to it and expect it to stop sequencing.
// [leader, healthy, sequencing] -- step down as leader --> [follower, healthy, not sequencing]
func (s *OpConductorTestSuite) TestScenario6() {
	s.enableSynchronization()

	// set initial state
	s.conductor.leader.Store(true)
	s.conductor.healthy.Store(true)
	s.conductor.seqActive.Store(true)

	s.ctrl.EXPECT().StopSequencer(mock.Anything).Return(common.Hash{}, nil).Times(1)

	// step down as leader
	s.updateLeaderStatusAndExecuteAction(false)

	// expect to stay as follower, go to [follower, healthy, not sequencing]
	s.False(s.conductor.leader.Load())
	s.True(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
	s.ctrl.AssertCalled(s.T(), "StopSequencer", mock.Anything)
}

// In this test, we have a leader that is healthy and sequencing, we send a unhealthy update to it and expect it to stop sequencing and transfer leadership.
// 1. [leader, healthy, sequencing] -- become unhealthy -->
// 2. [leader, unhealthy, sequencing] -- stop sequencing, transfer leadership --> [follower, unhealthy, not sequencing]
func (s *OpConductorTestSuite) TestScenario7() {
	s.enableSynchronization()

	// set initial state
	s.conductor.leader.Store(true)
	s.conductor.healthy.Store(true)
	s.conductor.seqActive.Store(true)

	s.cons.EXPECT().TransferLeader().Return(nil).Times(1)
	s.ctrl.EXPECT().StopSequencer(mock.Anything).Return(common.Hash{}, nil).Times(1)

	// become unhealthy
	s.updateHealthStatusAndExecuteAction(health.ErrSequencerNotHealthy)

	// expect to step down as leader and stop sequencing
	s.False(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
	s.ctrl.AssertCalled(s.T(), "StopSequencer", mock.Anything)
	s.cons.AssertCalled(s.T(), "TransferLeader")
}

// In this test, we have a leader that is healthy and sequencing, we send a unhealthy update to it and expect it to stop sequencing and transfer leadership.
// However, the action we needed to take failed temporarily, so we expect it to retry until it succeeds.
// 1. [leader, healthy, sequencing] -- become unhealthy -->
// 2. [leader, unhealthy, sequencing] -- stop sequencing failed, transfer leadership failed, retry -->
// 3. [leader, unhealthy, sequencing] -- stop sequencing succeeded, transfer leadership failed, retry -->
// 4. [leader, unhealthy, not sequencing] -- transfer leadership succeeded -->
// 5. [follower, unhealthy, not sequencing]
func (s *OpConductorTestSuite) TestFailureAndRetry1() {
	s.enableSynchronization()

	// set initial state
	s.conductor.leader.Store(true)
	s.conductor.healthy.Store(true)
	s.conductor.seqActive.Store(true)
	s.conductor.prevState = &state{
		leader:  true,
		healthy: true,
		active:  true,
	}

	// step 1 & 2: become unhealthy, stop sequencing failed, transfer leadership failed
	s.cons.EXPECT().TransferLeader().Return(s.err).Times(1)
	s.ctrl.EXPECT().StopSequencer(mock.Anything).Return(common.Hash{}, s.err).Times(1)

	s.updateHealthStatusAndExecuteAction(health.ErrSequencerNotHealthy)

	s.True(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.True(s.conductor.seqActive.Load())
	s.Equal(health.ErrSequencerNotHealthy, s.conductor.hcerr)
	s.Equal(&state{
		leader:  true,
		healthy: true,
		active:  true,
	}, s.conductor.prevState)
	s.ctrl.AssertNumberOfCalls(s.T(), "StopSequencer", 1)
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 1)

	// step 3: [leader, unhealthy, sequencing] -- stop sequencing succeeded, transfer leadership failed, retry
	s.ctrl.EXPECT().StopSequencer(mock.Anything).Return(common.Hash{}, nil).Times(1)
	s.cons.EXPECT().TransferLeader().Return(s.err).Times(1)

	s.executeAction()

	s.True(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
	s.Equal(health.ErrSequencerNotHealthy, s.conductor.hcerr)
	s.Equal(&state{
		leader:  true,
		healthy: true,
		active:  true,
	}, s.conductor.prevState)
	s.ctrl.AssertNumberOfCalls(s.T(), "StopSequencer", 2)
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 2)

	// step 4: [leader, unhealthy, not sequencing] -- transfer leadership succeeded
	s.cons.EXPECT().TransferLeader().Return(nil).Times(1)

	s.executeAction()

	// [follower, unhealthy, not sequencing]
	s.False(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
	s.Equal(health.ErrSequencerNotHealthy, s.conductor.hcerr)
	s.Equal(&state{
		leader:  true,
		healthy: false,
		active:  false,
	}, s.conductor.prevState)
	s.ctrl.AssertNumberOfCalls(s.T(), "StopSequencer", 2)
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 3)
}

// In this test, we have a leader that is healthy and sequencing, we send a unhealthy update to it and expect it to stop sequencing and transfer leadership.
// However, the action we needed to take failed temporarily, so we expect it to retry until it succeeds.
// 1. [leader, healthy, sequencing] -- become unhealthy -->
// 2. [leader, unhealthy, sequencing] -- stop sequencing failed, transfer leadership succeeded, retry -->
// 3. [follower, unhealthy, sequencing] -- stop sequencing succeeded -->
// 4. [follower, unhealthy, not sequencing]
func (s *OpConductorTestSuite) TestFailureAndRetry2() {
	s.enableSynchronization()

	// set initial state
	s.conductor.leader.Store(true)
	s.conductor.healthy.Store(true)
	s.conductor.seqActive.Store(true)
	s.conductor.prevState = &state{
		leader:  true,
		healthy: true,
		active:  true,
	}

	// step 1 & 2: become unhealthy, stop sequencing failed, transfer leadership succeeded, retry
	s.cons.EXPECT().TransferLeader().Return(nil).Times(1)
	s.ctrl.EXPECT().StopSequencer(mock.Anything).Return(common.Hash{}, s.err).Times(1)

	s.updateHealthStatusAndExecuteAction(health.ErrSequencerNotHealthy)

	s.False(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.True(s.conductor.seqActive.Load())
	s.Equal(health.ErrSequencerNotHealthy, s.conductor.hcerr)
	s.Equal(&state{
		leader:  true,
		healthy: true,
		active:  true,
	}, s.conductor.prevState)
	s.ctrl.AssertNumberOfCalls(s.T(), "StopSequencer", 1)
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 1)

	// step 3: [follower, unhealthy, sequencing] -- stop sequencing succeeded
	s.ctrl.EXPECT().StopSequencer(mock.Anything).Return(common.Hash{}, nil).Times(1)

	s.executeAction()

	s.False(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
	s.Equal(&state{
		leader:  false,
		healthy: false,
		active:  true,
	}, s.conductor.prevState)
	s.ctrl.AssertNumberOfCalls(s.T(), "StopSequencer", 2)
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 1)
}

// In this test, we have a follower that is unhealthy (due to active sequencer not producing blocks)
// Then leadership transfer happened, and the follower became leader. We expect it to start sequencing and catch up eventually.
// 1. [follower, healthy, not sequencing] -- become unhealthy -->
// 2. [follower, unhealthy, not sequencing] -- gained leadership -->
// 3. [leader, unhealthy, not sequencing] -- start sequencing -->
// 4. [leader, unhealthy, sequencing] -> become healthy again -->
// 5. [leader, healthy, sequencing]
func (s *OpConductorTestSuite) TestFailureAndRetry3() {
	s.enableSynchronization()

	// set initial state, healthy follower
	s.conductor.leader.Store(false)
	s.conductor.healthy.Store(true)
	s.conductor.seqActive.Store(false)
	s.conductor.prevState = &state{
		leader:  false,
		healthy: true,
		active:  false,
	}

	s.log.Info("1. become unhealthy")
	s.updateHealthStatusAndExecuteAction(health.ErrSequencerNotHealthy)

	s.False(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
	s.Equal(&state{
		leader:  false,
		healthy: false,
		active:  false,
	}, s.conductor.prevState)

	s.log.Info("2 & 3. gained leadership, start sequencing")
	mockPayload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: 1,
			BlockHash:   [32]byte{1, 2, 3},
		},
	}
	mockBlockInfo := &testutils.MockBlockInfo{
		InfoNum:  1,
		InfoHash: [32]byte{1, 2, 3},
	}
	s.cons.EXPECT().LatestUnsafePayload().Return(mockPayload, nil).Times(2)
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).Return(mockBlockInfo, nil).Times(2)
	s.ctrl.EXPECT().StartSequencer(mock.Anything, mockBlockInfo.InfoHash).Return(nil).Times(1)

	s.updateLeaderStatusAndExecuteAction(true)

	s.True(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.True(s.conductor.seqActive.Load())
	s.Equal(&state{
		leader:  true,
		healthy: false,
		active:  false,
	}, s.conductor.prevState)
	s.cons.AssertNumberOfCalls(s.T(), "LatestUnsafePayload", 1)
	s.ctrl.AssertNumberOfCalls(s.T(), "LatestUnsafeBlock", 1)
	s.ctrl.AssertNumberOfCalls(s.T(), "StartSequencer", 1)

	s.log.Info("4. stay unhealthy for a bit while catching up")
	s.updateHealthStatusAndExecuteAction(health.ErrSequencerNotHealthy)

	s.True(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.True(s.conductor.seqActive.Load())
	s.Equal(&state{
		leader:  true,
		healthy: false,
		active:  false,
	}, s.conductor.prevState)

	s.log.Info("5. become healthy again")
	s.updateHealthStatusAndExecuteAction(nil)

	// need to use eventually here because starting from step 4, the loop is gonna queue an action and retry until it became healthy again.
	// use eventually here avoids the situation where health update is consumed after the action is executed.
	s.Eventually(func() bool {
		res := s.conductor.leader.Load() == true &&
			s.conductor.healthy.Load() == true &&
			s.conductor.seqActive.Load() == true &&
			s.conductor.prevState.Equal(&state{
				leader:  true,
				healthy: true,
				active:  true,
			})
		if !res {
			s.executeAction()
		}
		return res
	}, 2*time.Second, time.Millisecond)
}

// This test is similar to TestFailureAndRetry3, but the consensus payload is one block ahead of the new leader's unsafe head.
// Then leadership transfer happened, and the follower became leader. We expect it to start sequencing and catch up eventually.
// 1. [follower, healthy, not sequencing] -- become unhealthy -->
// 2. [follower, unhealthy, not sequencing] -- gained leadership -->
// 3. [leader, unhealthy, not sequencing] -- start sequencing -->
// 4. [leader, unhealthy, sequencing] -> become healthy again -->
// 5. [leader, healthy, sequencing]
func (s *OpConductorTestSuite) TestFailureAndRetry4() {
	s.enableSynchronization()

	// set initial state, healthy follower
	s.conductor.leader.Store(false)
	s.conductor.healthy.Store(true)
	s.conductor.seqActive.Store(false)
	s.conductor.prevState = &state{
		leader:  false,
		healthy: true,
		active:  false,
	}

	s.log.Info("1. become unhealthy")
	s.updateHealthStatusAndExecuteAction(health.ErrSequencerNotHealthy)

	s.False(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
	s.Equal(&state{
		leader:  false,
		healthy: false,
		active:  false,
	}, s.conductor.prevState)

	s.log.Info("2 & 3. gained leadership, post unsafe payload and start sequencing")
	mockPayload := &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: 2,
			BlockHash:   [32]byte{4, 5, 6},
		},
	}
	mockBlockInfo := &testutils.MockBlockInfo{
		InfoNum:  1,
		InfoHash: [32]byte{1, 2, 3},
	}
	s.cons.EXPECT().LatestUnsafePayload().Return(mockPayload, nil).Times(2)
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).Return(mockBlockInfo, nil).Times(2)
	s.ctrl.EXPECT().PostUnsafePayload(mock.Anything, mockPayload).Return(nil).Times(1)
	s.ctrl.EXPECT().StartSequencer(mock.Anything, mockPayload.ExecutionPayload.BlockHash).Return(nil).Times(1)

	s.updateLeaderStatusAndExecuteAction(true)

	s.True(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.True(s.conductor.seqActive.Load())
	s.Equal(&state{
		leader:  true,
		healthy: false,
		active:  false,
	}, s.conductor.prevState)
	s.cons.AssertNumberOfCalls(s.T(), "LatestUnsafePayload", 1)
	s.ctrl.AssertNumberOfCalls(s.T(), "LatestUnsafeBlock", 1)
	s.ctrl.AssertNumberOfCalls(s.T(), "PostUnsafePayload", 1)
	s.ctrl.AssertNumberOfCalls(s.T(), "StartSequencer", 1)

	s.log.Info("4. stay unhealthy for a bit while catching up")
	s.updateHealthStatusAndExecuteAction(health.ErrSequencerNotHealthy)

	s.True(s.conductor.leader.Load())
	s.False(s.conductor.healthy.Load())
	s.True(s.conductor.seqActive.Load())
	s.Equal(&state{
		leader:  true,
		healthy: false,
		active:  false,
	}, s.conductor.prevState)

	s.log.Info("5. become healthy again")
	s.updateHealthStatusAndExecuteAction(nil)

	// need to use eventually here because starting from step 4, the loop is gonna queue an action and retry until it became healthy again.
	// use eventually here avoids the situation where health update is consumed after the action is executed.
	s.Eventually(func() bool {
		res := s.conductor.leader.Load() == true &&
			s.conductor.healthy.Load() == true &&
			s.conductor.seqActive.Load() == true &&
			s.conductor.prevState.Equal(&state{
				leader:  true,
				healthy: true,
				active:  true,
			})
		if !res {
			s.executeAction()
		}
		return res
	}, 2*time.Second, 100*time.Millisecond)
}

func (s *OpConductorTestSuite) TestConductorRestart() {
	// set initial state
	s.conductor.leader.Store(false)
	s.conductor.healthy.Store(true)
	s.conductor.seqActive.Store(true)
	s.ctrl.EXPECT().StopSequencer(mock.Anything).Return(common.Hash{}, nil).Times(1)

	s.enableSynchronization()

	// expect to stay as follower, go to [follower, healthy, not sequencing]
	s.False(s.conductor.leader.Load())
	s.True(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
	s.ctrl.AssertCalled(s.T(), "StopSequencer", mock.Anything)
}

func (s *OpConductorTestSuite) TestHandleInitError() {
	// This will cause an error in the init function, which should cause the conductor to stop successfully without issues.
	_, err := New(s.ctx, &s.cfg, s.log, s.version)
	_, ok := err.(*multierror.Error)
	// error should not be a multierror, this means that init failed, but Stop() succeeded, which is what we expect.
	s.False(ok)
}

// TestRollupBoostHealthFailure tests that OpConductor correctly handles rollup boost health failures
func (s *OpConductorTestSuite) TestRollupBoostHealthFailure() {
	s.enableSynchronization()

	// set initial state as a leader that is healthy and sequencing
	s.conductor.leader.Store(true)
	s.conductor.healthy.Store(true)
	s.conductor.seqActive.Store(true)
	s.conductor.prevState = &state{
		leader:  true,
		healthy: true,
		active:  true,
	}

	// Setup expectations - leader with unhealthy rollup boost should stop sequencing and transfer leadership
	s.ctrl.EXPECT().StopSequencer(mock.Anything).Return(common.Hash{}, nil).Times(1)
	s.cons.EXPECT().TransferLeader().Return(nil).Times(1)

	// Simulate a rollup boost health failure
	s.updateHealthStatusAndExecuteAction(health.ErrRollupBoostNotHealthy)

	// Verify the OpConductor transitions to follower state and stops sequencing
	s.False(s.conductor.leader.Load(), "Should transition to follower")
	s.False(s.conductor.healthy.Load(), "Should be marked as unhealthy")
	s.False(s.conductor.seqActive.Load(), "Sequencer should be stopped")
	s.Equal(health.ErrRollupBoostNotHealthy, s.conductor.hcerr, "Error should be stored")

	// Verify method calls
	s.ctrl.AssertNumberOfCalls(s.T(), "StopSequencer", 1)
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 1)
}

// TestRollupBoostConnectionDown tests that OpConductor correctly handles rollup boost connection failures
func (s *OpConductorTestSuite) TestRollupBoostConnectionDown() {
	s.enableSynchronization()

	// set initial state as a leader that is healthy and sequencing
	s.conductor.leader.Store(true)
	s.conductor.healthy.Store(true)
	s.conductor.seqActive.Store(true)
	s.conductor.prevState = &state{
		leader:  true,
		healthy: true,
		active:  true,
	}

	// Setup expectations - leader with rollup boost connection down should stop sequencing and transfer leadership
	s.ctrl.EXPECT().StopSequencer(mock.Anything).Return(common.Hash{}, nil).Times(1)
	s.cons.EXPECT().TransferLeader().Return(nil).Times(1)

	// Simulate a rollup boost connection failure
	s.updateHealthStatusAndExecuteAction(health.ErrRollupBoostConnectionDown)

	// Verify the OpConductor transitions to follower state and stops sequencing
	s.False(s.conductor.leader.Load(), "Should transition to follower")
	s.False(s.conductor.healthy.Load(), "Should be marked as unhealthy")
	s.False(s.conductor.seqActive.Load(), "Sequencer should be stopped")
	s.Equal(health.ErrRollupBoostConnectionDown, s.conductor.hcerr, "Error should be stored")

	// Verify method calls
	s.ctrl.AssertNumberOfCalls(s.T(), "StopSequencer", 1)
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 1)
}

func TestControlLoop(t *testing.T) {
	suite.Run(t, new(OpConductorTestSuite))
}

// TestSupervisorConnectionDown tests that OpConductor correctly handles supervisor connection failures
func (s *OpConductorTestSuite) TestSupervisorConnectionDown() {
	s.enableSynchronization()

	// set initial state as a leader that is healthy and sequencing
	s.conductor.leader.Store(true)
	s.conductor.healthy.Store(true)
	s.conductor.seqActive.Store(true)
	s.conductor.prevState = &state{
		leader:  true,
		healthy: true,
		active:  true,
	}

	// Setup expectations - leader with supervisor connection down should stop sequencing and transfer leadership
	s.ctrl.EXPECT().StopSequencer(mock.Anything).Return(common.Hash{}, nil).Times(1)
	s.cons.EXPECT().TransferLeader().Return(nil).Times(1)

	// Simulate a supervisor connection failure
	s.updateHealthStatusAndExecuteAction(health.ErrSupervisorConnectionDown)

	// Verify the OpConductor transitions to follower state and stops sequencing
	s.False(s.conductor.leader.Load(), "Should transition to follower")
	s.False(s.conductor.healthy.Load(), "Should be marked as unhealthy")
	s.False(s.conductor.seqActive.Load(), "Sequencer should be stopped")
	s.Equal(health.ErrSupervisorConnectionDown, s.conductor.hcerr, "Error should be stored")

	// Verify method calls
	s.ctrl.AssertNumberOfCalls(s.T(), "StopSequencer", 1)
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 1)
}

// TestFlashblocksHandlerIntegration tests that the flashblocks handler is properly initialized and started
func (s *OpConductorTestSuite) TestFlashblocksHandlerIntegration() {
	// Use a random available port to avoid conflicts
	listener, err := net.Listen("tcp", "localhost:0")
	s.NoError(err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Channels for coordination without timing dependencies
	testCtx, testCancel := context.WithCancel(context.Background())
	defer testCancel()

	serverConnected := make(chan struct{})
	clientConnected := make(chan struct{})
	messagesSent := make(chan struct{})

	// Use sync.Once to prevent double-closing channels
	var serverConnectedOnce, messagesSentOnce sync.Once

	// Create a test HTTP server for rollup boost WebSocket using coder/websocket
	rollupBoostServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Accept the WebSocket connection using coder/websocket
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			CompressionMode: websocket.CompressionDisabled,
		})
		if err != nil {
			s.T().Logf("Failed to accept WebSocket connection: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "test complete")

		// Signal that connection is established (only once)
		serverConnectedOnce.Do(func() {
			close(serverConnected)
		})

		// Wait for client to connect before sending messages
		select {
		case <-clientConnected:
			// Client is connected, proceed with sending messages
		case <-testCtx.Done():
			return
		}

		// Send test messages and signal completion
		messages := []string{"Hello", "World", "Test"}
		for _, msg := range messages {
			err := conn.Write(testCtx, websocket.MessageText, []byte(msg))
			if err != nil {
				s.T().Logf("Failed to write message: %v", err)
				return // Connection closed
			}
		}

		// Signal messages sent (only once)
		messagesSentOnce.Do(func() {
			close(messagesSent)
		})

		// Keep connection alive by reading until context is cancelled
		for {
			select {
			case <-testCtx.Done():
				return
			default:
				// Read with timeout to avoid blocking indefinitely
				readCtx, cancel := context.WithTimeout(testCtx, 100*time.Millisecond)
				_, _, err := conn.Read(readCtx)
				cancel()

				if err != nil {
					// Expected on timeout or connection close
					if errors.Is(err, context.DeadlineExceeded) {
						continue // Timeout is expected, continue loop
					}
					return // Other errors mean connection is closed
				}
			}
		}
	}))
	defer rollupBoostServer.Close()

	// Convert HTTP URL to WebSocket URL for rollup boost
	rollupBoostWsURL := strings.Replace(rollupBoostServer.URL, "http", "ws", 1)

	// Create a copy of the config to avoid modifying the shared config object
	testCfg := s.cfg
	testCfg.RollupBoostWsURL = rollupBoostWsURL
	testCfg.WebsocketServerPort = port

	// Create a new conductor with the updated config
	conductor, err := NewOpConductor(s.ctx, &testCfg, s.log, s.metrics, s.version, s.ctrl, s.cons, s.hmon)
	s.NoError(err)

	// Set up mock expectation for Leader() calls - the flashblocks handler checks leadership
	// before forwarding messages, so we need to mock this to return true
	s.cons.EXPECT().Leader().Return(true)

	// Start the conductor, which should initialize and start the flashblocks handler
	s.hmon.EXPECT().Start(mock.Anything).Return(nil)
	err = conductor.Start(s.ctx)
	s.NoError(err)

	// Wait for conductor to be ready using its internal state
	s.NotNil(conductor.flashblocksHandler, "flashblocks handler should be initialized")

	// Wait for rollup boost server connection (event-driven, not time-based)
	select {
	case <-serverConnected:
		// Connection established
	case <-time.After(5 * time.Second):
		s.Fail("Timeout waiting for rollup boost server connection")
	}

	// Connect to the WebSocket server BEFORE messages are sent
	wsURL := fmt.Sprintf("ws://localhost:%d/ws", testCfg.WebsocketServerPort)

	// Create connection context
	connCtx, connCancel := context.WithTimeout(testCtx, 3*time.Second)
	defer connCancel()

	var client *websocket.Conn
	var resp *http.Response

	// Simple retry loop with context timeout
	for {
		select {
		case <-connCtx.Done():
			s.Fail("Failed to connect to WebSocket server within timeout")
		default:
			client, resp, err = websocket.Dial(connCtx, wsURL, nil)
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			if err == nil && resp.StatusCode == http.StatusSwitchingProtocols {
				goto connected
			}
			// Brief pause before retry
			select {
			case <-connCtx.Done():
				s.Failf("Failed to connect to WebSocket server", "Last error: %v", err)
			case <-time.After(10 * time.Millisecond):
				// Continue loop
			}
		}
	}

connected:
	defer client.Close(websocket.StatusNormalClosure, "test complete")

	// Signal that client is connected so rollup boost server can send messages
	close(clientConnected)

	// Wait for messages to be sent (event-driven)
	select {
	case <-messagesSent:
		// Messages sent
	case <-time.After(2 * time.Second):
		s.Fail("Timeout waiting for messages to be sent")
	}

	// Wait for and verify we receive messages from rollup boost (event-driven)
	expectedMessages := []string{"Hello", "World", "Test"}
	receivedMessages := make([]string, 0, len(expectedMessages))

	// Read messages with timeout
	readCtx, readCancel := context.WithTimeout(testCtx, 3*time.Second)
	defer readCancel()

	for len(receivedMessages) < len(expectedMessages) {
		_, message, err := client.Read(readCtx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				s.Failf("Timeout waiting for messages", "Received %d/%d messages: %v",
					len(receivedMessages), len(expectedMessages), receivedMessages)
			} else {
				s.Failf("Error reading messages", "Error: %v", err)
			}
			break
		}
		receivedMessages = append(receivedMessages, string(message))
	}

	// Verify we received the expected messages
	s.Equal(len(expectedMessages), len(receivedMessages), "Should receive all expected messages")
	for i, expected := range expectedMessages {
		if i < len(receivedMessages) {
			s.Equal(expected, receivedMessages[i], "Message content should match")
		}
	}
	s.T().Log("Successfully received all messages from rollup boost via op-conductor")

	// Stop the conductor, which should also stop the flashblocks handler
	s.hmon.EXPECT().Stop().Return(nil)
	s.cons.EXPECT().Shutdown().Return(nil)
	err = conductor.Stop(s.ctx)
	s.NoError(err)

	// Verify that the conductor is stopped
	s.True(conductor.Stopped())
}

// TestRollupBoostPartialFailure tests that OpConductor correctly handles rollup boost partial health failures.
// This test verifies that when a leader is unhealthy and actively sequencing due to ErrRollupBoostPartiallyHealthy,
// it should stop sequencing and transfer leadership instead of waiting for health recovery.
// Scenario: [leader, unhealthy, active] with prevState [leader, unhealthy, inactive] and ErrRollupBoostPartiallyHealthy
// Expected: Stop sequencing and transfer leadership (not wait for recovery)
func (s *OpConductorTestSuite) TestRollupBoostPartialFailure() {
	s.enableSynchronization()

	// Set initial state: leader is unhealthy and actively sequencing
	// Previous state was [leader, unhealthy, inactive] - this simulates the scenario where
	// the leader started sequencing during a network stall but rollup boost is partially healthy
	s.conductor.leader.Store(true)
	s.conductor.healthy.Store(false)
	s.conductor.seqActive.Store(true)
	s.conductor.prevState = &state{
		leader:  true,
		healthy: false,
		active:  false,
	}
	s.conductor.cfg.RollupBoostEnabled = true

	// Setup expectations - with ErrRollupBoostPartiallyHealthy, conductor should NOT wait for recovery
	// Instead, it should stop sequencing and transfer leadership to another node
	s.ctrl.EXPECT().StopSequencer(mock.Anything).Return(common.Hash{}, nil).Times(1)
	s.cons.EXPECT().TransferLeader().Return(nil).Times(1)

	// Trigger the health update with rollup boost partial failure
	s.updateHealthStatusAndExecuteAction(health.ErrRollupBoostPartiallyHealthy)

	// Verify the conductor stops sequencing and transfers leadership instead of waiting for recovery
	s.False(s.conductor.leader.Load(), "Should transfer leadership to another node")
	s.False(s.conductor.healthy.Load(), "Should remain marked as unhealthy")
	s.False(s.conductor.seqActive.Load(), "Should stop sequencing")
	s.Equal(health.ErrRollupBoostPartiallyHealthy, s.conductor.hcerr, "Should store the rollup boost error")

	// Verify the expected actions were taken
	s.ctrl.AssertNumberOfCalls(s.T(), "StopSequencer", 1)
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 1)
}
