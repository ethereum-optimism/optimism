package conductor

import (
	"context"
	"errors"
	"fmt"
	"math/big"
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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	clientmocks "github.com/ethereum-optimism/optimism/op-conductor/client/mocks"
	consensusmocks "github.com/ethereum-optimism/optimism/op-conductor/consensus/mocks"
	"github.com/ethereum-optimism/optimism/op-conductor/health"
	healthmocks "github.com/ethereum-optimism/optimism/op-conductor/health/mocks"
	"github.com/ethereum-optimism/optimism/op-conductor/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
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
			BlockTime:              2,
			MaxSequencerDrift:      600,
			SeqWindowSize:          3600,
			ChannelTimeoutBedrock:  300,
			L1ChainID:              big.NewInt(1),
			L2ChainID:              big.NewInt(2),
			RegolithTime:           &now,
			CanyonTime:             &now,
			BatchInboxAddress:      [20]byte{1, 2},
			DepositContractAddress: [20]byte{2, 3},
			L1SystemConfigAddress:  [20]byte{3, 4},
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
	// Poll for op-node adopting a posted payload quickly, and give up quickly, so tests that
	// exercise a node which never catches up do not wait on the production timeout.
	conductor.unsafeHeadCatchUpInterval = time.Millisecond
	conductor.startSequencerRetryInterval = time.Millisecond
	conductor.unsafeHeadCatchUpTimeout = 50 * time.Millisecond
	conductor.unhealthyUnsafeHeadCatchUpTimeout = 50 * time.Millisecond
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
		// Drop the retry the last failing action queued. Teardown hands the loop exactly one turn
		// and then relies on it being inside loopAction's select when Stop cancels shutdownCtx: a
		// pending action consumes that turn instead, leaving the loop parked on the next hand-off
		// while Stop waits for it to exit.
		select {
		case <-s.conductor.actionCh:
		default:
		}
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

	// unsafe in consensus is 2 blocks ahead of unsafe in node, and consensus no longer retains
	// the missing payloads, so op-node cannot be brought up to the consensus head here.
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
	s.cons.EXPECT().UnsafePayloadsAfter(uint64(1)).Return(nil).Times(1)
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).Return(mockBlockInfo, nil)
	s.ctrl.EXPECT().PostUnsafePayload(mock.Anything, mockPayload).Return(nil).Times(1)

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
	s.ctrl.AssertNotCalled(s.T(), "StartSequencer", mock.Anything, mock.Anything)
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
	s.cons.EXPECT().UnsafePayloadsAfter(uint64(1)).Return([]*eth.ExecutionPayloadEnvelope{mockPayload}).Times(1)
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).Return(mockBlockInfo, nil).Times(1)
	s.ctrl.EXPECT().PostUnsafePayload(mock.Anything, mockPayload).Return(errors.New("simulated PostUnsafePayload failure")).Times(1)

	s.updateLeaderStatusAndExecuteAction(true)

	// [leader, healthy, not sequencing]
	s.True(s.conductor.leader.Load())
	s.True(s.conductor.healthy.Load())
	s.False(s.conductor.seqActive.Load())
	s.cons.AssertNumberOfCalls(s.T(), "LatestUnsafePayload", 1)
	s.ctrl.AssertNumberOfCalls(s.T(), "LatestUnsafeBlock", 1)
	s.ctrl.AssertNumberOfCalls(s.T(), "PostUnsafePayload", 1)
	s.ctrl.AssertNotCalled(s.T(), "StartSequencer", mock.Anything, mock.Anything)

	// The post succeeds on retry, and the sequencer is only started once op-node reports it has
	// adopted the posted head: starting at a head op-node has not adopted is rejected by op-node.
	node := newFakeOpNode(mockBlockInfo.InfoNum, mockBlockInfo.InfoHash)
	s.cons.EXPECT().LatestUnsafePayload().Return(mockPayload, nil).Times(1)
	s.cons.EXPECT().UnsafePayloadsAfter(uint64(1)).Return([]*eth.ExecutionPayloadEnvelope{mockPayload}).Times(1)
	s.expectOpNode(node)

	s.executeAction()

	// [leader, healthy, sequencing]
	s.True(s.conductor.leader.Load())
	s.True(s.conductor.healthy.Load())
	s.True(s.conductor.seqActive.Load())
	s.cons.AssertNumberOfCalls(s.T(), "LatestUnsafePayload", 2)
	s.ctrl.AssertNumberOfCalls(s.T(), "LatestUnsafeBlock", 3)
	s.ctrl.AssertNumberOfCalls(s.T(), "PostUnsafePayload", 2)
	s.ctrl.AssertNumberOfCalls(s.T(), "StartSequencer", 1)
	s.True(node.started, "op-node must have accepted the start at the head it adopted")
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
	node := newFakeOpNode(1, [32]byte{1, 2, 3})
	s.cons.EXPECT().LatestUnsafePayload().Return(mockPayload, nil)
	s.cons.EXPECT().UnsafePayloadsAfter(uint64(1)).Return([]*eth.ExecutionPayloadEnvelope{mockPayload})
	s.expectOpNode(node)

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
	// Two reads: the initial comparison against consensus, then the poll that observes op-node
	// adopting the posted payload.
	s.ctrl.AssertNumberOfCalls(s.T(), "LatestUnsafeBlock", 2)
	s.ctrl.AssertNumberOfCalls(s.T(), "PostUnsafePayload", 1)
	s.ctrl.AssertNumberOfCalls(s.T(), "StartSequencer", 1)
	s.Equal(uint64(2), node.head.InfoNum)

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
	// error should not be a joined error, this means that init failed, but Stop() succeeded, which is what we expect.
	type multiUnwrap interface{ Unwrap() []error }
	_, ok := err.(multiUnwrap)
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

// TestFlashblocksHandlerIntegration tests that the flashblocks handler is properly initialized and started
func (s *OpConductorTestSuite) TestFlashblocksHandlerIntegration() {

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
	// Bind port dynamically via handler (use port 0)
	testCfg.WebsocketServerPort = 0

	// Create a new conductor with the updated config
	conductor, err := NewOpConductor(s.ctx, &testCfg, s.log, s.metrics, s.version, s.ctrl, s.cons, s.hmon)
	s.NoError(err)

	// Set up mock expectation for Leader() calls - the flashblocks handler checks leadership
	// before forwarding messages, so we need to mock this to return true
	s.cons.EXPECT().Leader().Return(true)

	// Start the conductor, which should initialize and start the flashblocks handler
	s.hmon.EXPECT().Start(mock.Anything).Return(nil)

	s.NotNil(conductor.flashblocksHandler, "flashblocks handler should be initialized before starting the conductor")
	err = conductor.Start(s.ctx)
	s.NoError(err)

	boundPort := conductor.flashblocksHandler.BoundPort()
	s.NotZero(boundPort, "bound port should be non-zero")

	// Wait for rollup boost server connection (event-driven, not time-based)
	select {
	case <-serverConnected:
		// Connection established
	case <-time.After(5 * time.Second):
		s.Fail("Timeout waiting for rollup boost server connection")
	}

	// Connect to the WebSocket server BEFORE messages are sent
	wsURL := fmt.Sprintf("ws://127.0.0.1:%d/ws", boundPort)

	// Create connection context
	connCtx, connCancel := context.WithTimeout(testCtx, 10*time.Second)
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
			case <-time.After(25 * time.Millisecond):
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
	readCtx, readCancel := context.WithTimeout(testCtx, 10*time.Second)
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

// fakeOpNode models the two op-node behaviours that a conductor handing over sequencing has to
// respect: a posted payload is not adopted the instant PostUnsafePayload returns, and op-node
// refuses to start sequencing at a head it has not adopted.
//
// Adoption advances on any interaction with op-node other than a post, so a conductor that waits
// converges however it chooses to observe op-node, but one that starts sequencing straight after
// posting does not.
type fakeOpNode struct {
	head    *testutils.MockBlockInfo
	pending []*eth.ExecutionPayloadEnvelope
	started bool
	// adoptPerInteraction is how many posted payloads op-node adopts per interaction: 0 models a
	// node that never catches up, and more than 1 a node that can step over the head the
	// conductor is waiting for between two observations.
	adoptPerInteraction int
	// startLag rejects this many starts at the already-adopted head, modelling op-node's
	// sequencer learning the head from a forkchoice update that trails the execution client.
	startLag int
}

func newFakeOpNode(number uint64, hash common.Hash) *fakeOpNode {
	return &fakeOpNode{
		head:                &testutils.MockBlockInfo{InfoNum: number, InfoHash: hash},
		adoptPerInteraction: 1,
	}
}

func (n *fakeOpNode) adopt() {
	for range n.adoptPerInteraction {
		if len(n.pending) == 0 || uint64(n.pending[0].ExecutionPayload.BlockNumber) != n.head.InfoNum+1 {
			return
		}
		next := n.pending[0]
		n.pending = n.pending[1:]
		n.head = &testutils.MockBlockInfo{
			InfoNum:  uint64(next.ExecutionPayload.BlockNumber),
			InfoHash: next.ExecutionPayload.BlockHash,
		}
	}
}

func (n *fakeOpNode) latestUnsafeBlock(context.Context) (eth.BlockInfo, error) {
	n.adopt()
	return n.head, nil
}

func (n *fakeOpNode) postUnsafePayload(_ context.Context, payload *eth.ExecutionPayloadEnvelope) error {
	n.pending = append(n.pending, payload)
	return nil
}

func (n *fakeOpNode) startSequencer(_ context.Context, head common.Hash) error {
	n.adopt()
	sequencerHead := n.head
	if n.startLag > 0 {
		n.startLag--
		sequencerHead = &testutils.MockBlockInfo{InfoNum: n.head.InfoNum - 1, InfoHash: common.HexToHash("0x5741e")}
	}
	if head != sequencerHead.InfoHash {
		// Mirrors sequencing.Sequencer.Start, which compares against the head it has adopted.
		return fmt.Errorf("%w: head %s:%d, received %s", driver.ErrUnsafeHeadMismatch, sequencerHead.InfoHash, sequencerHead.InfoNum, head)
	}
	n.started = true
	return nil
}

func (s *OpConductorTestSuite) expectOpNode(node *fakeOpNode) {
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).RunAndReturn(node.latestUnsafeBlock)
	s.ctrl.EXPECT().PostUnsafePayload(mock.Anything, mock.Anything).RunAndReturn(node.postUnsafePayload)
	s.ctrl.EXPECT().StartSequencer(mock.Anything, mock.Anything).RunAndReturn(node.startSequencer)
}

func unsafePayload(number uint64) *eth.ExecutionPayloadEnvelope {
	return &eth.ExecutionPayloadEnvelope{
		ExecutionPayload: &eth.ExecutionPayload{
			BlockNumber: eth.Uint64Quantity(number),
			Timestamp:   hexutil.Uint64(time.Now().Unix()),
			BlockHash:   common.BigToHash(new(big.Int).SetUint64(number)),
		},
	}
}

// A new leader whose op-node is one block behind must not be told to start sequencing until
// op-node has actually adopted the payload the conductor just posted. Starting straight after
// PostUnsafePayload loses the race and costs a full action retry backoff, which is what leaves
// the sequencer idle long enough to have to build several blocks at once when it does start.
func (s *OpConductorTestSuite) TestStartSequencerWaitsForPostedPayloadToBeAdopted() {
	s.enableSynchronization()

	consensusHead := unsafePayload(12)
	node := newFakeOpNode(11, unsafePayload(11).ExecutionPayload.BlockHash)
	s.cons.EXPECT().LatestUnsafePayload().Return(consensusHead, nil)
	s.cons.EXPECT().UnsafePayloadsAfter(uint64(11)).Return([]*eth.ExecutionPayloadEnvelope{consensusHead})
	s.expectOpNode(node)

	s.updateLeaderStatusAndExecuteAction(true)

	s.True(s.conductor.seqActive.Load(), "sequencing must be active after a single action")
	s.True(node.started)
	s.Equal(uint64(12), node.head.InfoNum)
	s.ctrl.AssertNumberOfCalls(s.T(), "StartSequencer", 1)
}

// A new leader two or more blocks behind the consensus unsafe head used to be unrecoverable: the
// payload post was skipped, startSequencer returned unsafe head mismatch, and the action loop
// re-queued the identical failing action forever with no sequencer anywhere in the cluster.
// Consensus retains the payloads, so they are replayed and the leader converges.
func (s *OpConductorTestSuite) TestStartSequencerRecoversFromMultiBlockGap() {
	s.enableSynchronization()

	missing := []*eth.ExecutionPayloadEnvelope{unsafePayload(13), unsafePayload(14)}
	node := newFakeOpNode(12, unsafePayload(12).ExecutionPayload.BlockHash)
	s.cons.EXPECT().LatestUnsafePayload().Return(missing[1], nil)
	s.cons.EXPECT().UnsafePayloadsAfter(uint64(12)).Return(missing)
	s.expectOpNode(node)

	s.updateLeaderStatusAndExecuteAction(true)

	s.True(s.conductor.seqActive.Load(), "sequencing must be active after a single action")
	s.True(node.started)
	s.Equal(uint64(14), node.head.InfoNum)
	s.ctrl.AssertNumberOfCalls(s.T(), "PostUnsafePayload", 2)
	s.ctrl.AssertNumberOfCalls(s.T(), "StartSequencer", 1)
}

// The execution client reporting the consensus unsafe head is not proof that op-node's sequencer
// can start there: it learns the adopted head from a forkchoice update that can trail by an event
// hop. That rejection must be retried in place, not turned into an action retry that idles the
// cluster for a randomised backoff.
func (s *OpConductorTestSuite) TestStartSequencerRetriesWhileOpNodeSequencerTrailsTheEL() {
	s.enableSynchronization()

	consensusHead := unsafePayload(12)
	node := newFakeOpNode(12, consensusHead.ExecutionPayload.BlockHash)
	node.startLag = 2
	s.cons.EXPECT().LatestUnsafePayload().Return(consensusHead, nil)
	s.expectOpNode(node)

	s.updateLeaderStatusAndExecuteAction(true)

	s.True(s.conductor.seqActive.Load(), "sequencing must be active after a single action")
	s.True(node.started)
	s.ctrl.AssertNumberOfCalls(s.T(), "StartSequencer", 3)
	s.ctrl.AssertNotCalled(s.T(), "PostUnsafePayload", mock.Anything, mock.Anything)
}

// When the local op-node cannot be brought up to the consensus unsafe head at all, the cluster
// must still converge on one sequencer. A healthy leader that keeps failing to start hands
// leadership to a member that may be caught up instead of retrying itself forever.
func (s *OpConductorTestSuite) TestStartSequencerTransfersLeadershipWhenItCannotConverge() {
	s.enableSynchronization()

	consensusHead := unsafePayload(14)
	node := newFakeOpNode(12, unsafePayload(12).ExecutionPayload.BlockHash)
	node.adoptPerInteraction = 0 // op-node never adopts what it is sent
	s.cons.EXPECT().LatestUnsafePayload().Return(consensusHead, nil)
	s.cons.EXPECT().UnsafePayloadsAfter(uint64(12)).Return([]*eth.ExecutionPayloadEnvelope{unsafePayload(13), consensusHead})
	s.expectOpNode(node)
	s.cons.EXPECT().TransferLeader().Return(nil).Times(1)

	s.updateLeaderStatusAndExecuteAction(true)
	s.False(s.conductor.seqActive.Load())
	s.True(s.conductor.leader.Load(), "must not give up leadership on the first failure")

	for range maxFailedStarts - 1 {
		s.executeAction()
	}

	s.False(s.conductor.leader.Load(), "leadership must be handed over once the leader cannot start")
	s.False(s.conductor.seqActive.Load())
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 1)
	s.False(node.started)
}

// A transfer that fails must be retried on the next action. Zeroing the failed-start counter
// before the transfer left a leader that could not hand over needing another maxFailedStarts
// failed starts, and so several more block times with no sequencer in the cluster, before trying
// again.
func (s *OpConductorTestSuite) TestStartSequencerRetriesLeadershipTransferAfterItFails() {
	s.enableSynchronization()

	consensusHead := unsafePayload(14)
	node := newFakeOpNode(13, unsafePayload(13).ExecutionPayload.BlockHash)
	node.adoptPerInteraction = 0
	s.cons.EXPECT().LatestUnsafePayload().Return(consensusHead, nil)
	s.cons.EXPECT().UnsafePayloadsAfter(uint64(13)).Return([]*eth.ExecutionPayloadEnvelope{consensusHead})
	s.expectOpNode(node)
	s.cons.EXPECT().TransferLeader().Return(errors.New("no healthy voter to transfer to")).Times(1)

	s.updateLeaderStatusAndExecuteAction(true)
	for range maxFailedStarts - 1 {
		s.executeAction()
	}
	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 1)
	s.True(s.conductor.leader.Load(), "a failed transfer leaves this server the leader")

	s.cons.EXPECT().TransferLeader().Return(nil).Times(1)
	s.executeAction()

	s.cons.AssertNumberOfCalls(s.T(), "TransferLeader", 2)
	s.False(s.conductor.leader.Load())
}

// allowLeadershipTransfer permits TransferLeader and reports whether it was called, so that a
// regression fails on an assertion rather than on an unexpected mock call.
func (s *OpConductorTestSuite) allowLeadershipTransfer() *bool {
	transferred := new(bool)
	s.cons.EXPECT().TransferLeader().RunAndReturn(func() error {
		*transferred = true
		return nil
	}).Maybe()
	return transferred
}

// retryFailingActions drives the action loop the given number of times, which only works while
// every action keeps failing and so keeps queueing the next one. It aborts as soon as leadership is
// transferred: that is the assertion, and it is also the point past which a successful action would
// leave the next iteration with nothing to run.
func (s *OpConductorTestSuite) retryFailingActions(transferred *bool, actions int) {
	for range actions {
		s.Require().False(*transferred, "leadership must not be transferred")
		s.executeAction()
	}
	s.Require().False(*transferred, "leadership must not be transferred")
}

// op-node being level with or ahead of the head consensus committed is not a local problem: a
// sealed block reaches consensus before it is gossiped, so nothing can put one member's op-node
// ahead of consensus without putting every member's there. Round-robin transfer would then cycle
// the whole cluster, each member sequencer-less in turn, so this retries in place instead.
func (s *OpConductorTestSuite) TestStartSequencerDoesNotTransferLeadershipWhenNodeIsAheadOfConsensus() {
	s.enableSynchronization()

	consensusHead := unsafePayload(14)
	node := newFakeOpNode(15, unsafePayload(15).ExecutionPayload.BlockHash)
	s.cons.EXPECT().LatestUnsafePayload().Return(consensusHead, nil)
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).RunAndReturn(node.latestUnsafeBlock)
	transferred := s.allowLeadershipTransfer()

	s.updateLeaderStatusAndExecuteAction(true)
	s.retryFailingActions(transferred, 2*maxFailedStarts)

	s.False(s.conductor.seqActive.Load())
	s.True(s.conductor.leader.Load(), "leadership must stay put: every member sees the same thing")
	s.ctrl.AssertNotCalled(s.T(), "PostUnsafePayload", mock.Anything, mock.Anything)
	s.ctrl.AssertNotCalled(s.T(), "StartSequencer", mock.Anything, mock.Anything)
}

// A cluster that has committed nothing to consensus yet — freshly bootstrapped, or re-bootstrapped
// — is in the same state on every member, so handing leadership over cannot help and only costs
// the cluster a sequencer-less window per member.
func (s *OpConductorTestSuite) TestStartSequencerDoesNotTransferLeadershipWithoutAConsensusHead() {
	s.enableSynchronization()

	s.cons.EXPECT().LatestUnsafePayload().Return(nil, nil)
	transferred := s.allowLeadershipTransfer()

	s.updateLeaderStatusAndExecuteAction(true)
	s.retryFailingActions(transferred, 2*maxFailedStarts)

	s.False(s.conductor.seqActive.Load())
	s.True(s.conductor.leader.Load())
}

// A start abandoned because this conductor is shutting down must not hand leadership over on its
// way out.
func (s *OpConductorTestSuite) TestStartSequencerDoesNotTransferLeadershipOnCancelledContext() {
	s.enableSynchronization()

	s.cons.EXPECT().LatestUnsafePayload().Return(unsafePayload(14), nil)
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).Return(nil, context.Canceled)
	transferred := s.allowLeadershipTransfer()

	s.updateLeaderStatusAndExecuteAction(true)
	s.retryFailingActions(transferred, 2*maxFailedStarts)

	s.False(s.conductor.seqActive.Load())
	s.True(s.conductor.leader.Load())
}

// The retained history is read without a raft barrier while the consensus head comes from a
// barrier read, so it can already run past the head this attempt is waiting for. Posting past the
// target steps op-node's head over it, and the wait compares hashes exactly, so it would then
// never see a match.
func (s *OpConductorTestSuite) TestStartSequencerDoesNotReplayPastTheConsensusHead() {
	s.enableSynchronization()

	consensusHead := unsafePayload(13)
	node := newFakeOpNode(11, unsafePayload(11).ExecutionPayload.BlockHash)
	node.adoptPerInteraction = 3 // op-node can adopt everything it holds between two reads
	s.cons.EXPECT().LatestUnsafePayload().Return(consensusHead, nil)
	s.cons.EXPECT().UnsafePayloadsAfter(uint64(11)).Return([]*eth.ExecutionPayloadEnvelope{
		unsafePayload(12), consensusHead, unsafePayload(14),
	})
	s.expectOpNode(node)

	s.updateLeaderStatusAndExecuteAction(true)

	s.True(s.conductor.seqActive.Load(), "sequencing must be active after a single action")
	s.True(node.started)
	s.Equal(uint64(13), node.head.InfoNum, "op-node must not be pushed past the head being waited for")
	s.ctrl.AssertNumberOfCalls(s.T(), "PostUnsafePayload", 2)
}

// The payloads are already posted and the window still has time left, so one failed head read must
// not abandon the attempt — let alone count towards handing leadership over.
func (s *OpConductorTestSuite) TestStartSequencerToleratesATransientHeadReadFailure() {
	s.enableSynchronization()

	consensusHead := unsafePayload(12)
	node := newFakeOpNode(11, unsafePayload(11).ExecutionPayload.BlockHash)
	s.cons.EXPECT().LatestUnsafePayload().Return(consensusHead, nil)
	s.cons.EXPECT().UnsafePayloadsAfter(uint64(11)).Return([]*eth.ExecutionPayloadEnvelope{consensusHead})
	s.ctrl.EXPECT().PostUnsafePayload(mock.Anything, mock.Anything).RunAndReturn(node.postUnsafePayload)
	s.ctrl.EXPECT().StartSequencer(mock.Anything, mock.Anything).RunAndReturn(node.startSequencer)

	reads := 0
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).RunAndReturn(func(ctx context.Context) (eth.BlockInfo, error) {
		reads++
		if reads == 2 {
			return nil, errors.New("simulated EL read failure")
		}
		return node.latestUnsafeBlock(ctx)
	})

	s.updateLeaderStatusAndExecuteAction(true)

	s.True(s.conductor.seqActive.Load(), "one failed read must not abandon the attempt")
	s.True(node.started)
	s.cons.AssertNotCalled(s.T(), "TransferLeader")
}

// A start rejected because op-node's sequencer has not caught up with its execution client is
// retried on its own, slower interval: StartSequencer calls back into this conductor and then takes
// the sequencer's own lock, so it must not be hammered at the rate of a cheap head read.
func (s *OpConductorTestSuite) TestStartSequencerRetryUsesItsOwnInterval() {
	s.enableSynchronization()

	s.conductor.unsafeHeadCatchUpInterval = 0
	s.conductor.startSequencerRetryInterval = 20 * time.Millisecond
	s.conductor.unsafeHeadCatchUpTimeout = 5 * time.Second

	consensusHead := unsafePayload(12)
	node := newFakeOpNode(12, consensusHead.ExecutionPayload.BlockHash)
	node.startLag = 3
	s.cons.EXPECT().LatestUnsafePayload().Return(consensusHead, nil)
	s.expectOpNode(node)

	start := time.Now()
	s.updateLeaderStatusAndExecuteAction(true)

	s.True(s.conductor.seqActive.Load())
	s.GreaterOrEqual(time.Since(start), 3*s.conductor.startSequencerRetryInterval,
		"each rejected start must wait the start-retry interval, not the head-poll interval")
	s.ctrl.AssertNumberOfCalls(s.T(), "StartSequencer", 4)
}

// An unhealthy leader's fallback is to hand leadership over, and its own op-node is the likely
// reason a catch-up is slow, so it must not sit on the single action loop for the budget a healthy
// leader would spend waiting on it.
func (s *OpConductorTestSuite) TestStartSequencerBudgetsAnUnhealthyLeaderTighter() {
	s.enableSynchronization() // leaves the action loop parked, so this drives startSequencer itself

	s.conductor.unsafeHeadCatchUpTimeout = time.Minute
	s.conductor.unhealthyUnsafeHeadCatchUpTimeout = 20 * time.Millisecond

	consensusHead := unsafePayload(14)
	node := newFakeOpNode(13, unsafePayload(13).ExecutionPayload.BlockHash)
	node.adoptPerInteraction = 0 // op-node never adopts what it is sent
	s.cons.EXPECT().LatestUnsafePayload().Return(consensusHead, nil)
	s.cons.EXPECT().UnsafePayloadsAfter(uint64(13)).Return([]*eth.ExecutionPayloadEnvelope{consensusHead})
	s.expectOpNode(node)

	s.conductor.healthy.Store(false)
	start := time.Now()
	err := s.conductor.startSequencer()
	elapsed := time.Since(start)

	s.ErrorIs(err, ErrUnsafeHeadMismatch)
	s.Less(elapsed, 10*time.Second, "an unhealthy leader must not wait out the healthy budget")
	s.GreaterOrEqual(elapsed, s.conductor.unhealthyUnsafeHeadCatchUpTimeout/2,
		"but it must still give op-node its own budget")
}

// Catching up and getting the start accepted are two halves of the same wait and must share one
// budget: action() is a single loop, so two budgets in a row is twice as long with leadership and
// health updates queued behind it.
func (s *OpConductorTestSuite) TestStartSequencerSharesOneBudgetWithTheCatchUp() {
	s.enableSynchronization() // leaves the action loop parked, so this drives startSequencer itself

	const budget = 400 * time.Millisecond
	s.conductor.unsafeHeadCatchUpTimeout = budget
	s.conductor.unsafeHeadCatchUpInterval = 5 * time.Millisecond
	s.conductor.startSequencerRetryInterval = 5 * time.Millisecond

	consensusHead := unsafePayload(12)
	behind := &testutils.MockBlockInfo{InfoNum: 11, InfoHash: unsafePayload(11).ExecutionPayload.BlockHash}
	caughtUp := &testutils.MockBlockInfo{InfoNum: 12, InfoHash: consensusHead.ExecutionPayload.BlockHash}
	// op-node's execution client spends most of the budget adopting the payload, and op-node's
	// sequencer never catches up with it afterwards. The start gets what is left of the budget.
	adopted := time.Now().Add(280 * time.Millisecond)

	s.cons.EXPECT().LatestUnsafePayload().Return(consensusHead, nil)
	s.cons.EXPECT().UnsafePayloadsAfter(uint64(11)).Return([]*eth.ExecutionPayloadEnvelope{consensusHead})
	s.ctrl.EXPECT().PostUnsafePayload(mock.Anything, mock.Anything).Return(nil)
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).RunAndReturn(func(context.Context) (eth.BlockInfo, error) {
		if time.Now().Before(adopted) {
			return behind, nil
		}
		return caughtUp, nil
	})
	s.ctrl.EXPECT().StartSequencer(mock.Anything, mock.Anything).
		Return(fmt.Errorf("%w: head 11, received 12", driver.ErrUnsafeHeadMismatch))

	start := time.Now()
	err := s.conductor.startSequencer()

	s.ErrorIs(err, ErrUnsafeHeadMismatch)
	s.Less(time.Since(start), budget+budget/8, "catch-up and start must share one budget, not take one each")
}

// The select guards honour shutdownCtx but an RPC in flight does not, and startSequencer makes
// dozens of them, so the attempt needs a deadline of its own or an unresponsive op-node wedges the
// action loop.
func (s *OpConductorTestSuite) TestStartSequencerBoundsItsRPCs() {
	s.enableSynchronization() // leaves the action loop parked, so this drives startSequencer itself

	s.conductor.unsafeHeadCatchUpTimeout = 50 * time.Millisecond
	s.cons.EXPECT().LatestUnsafePayload().Return(unsafePayload(12), nil)
	s.ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).RunAndReturn(func(ctx context.Context) (eth.BlockInfo, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	done := make(chan error, 1)
	go func() { done <- s.conductor.startSequencer() }()

	select {
	case err := <-done:
		s.ErrorIs(err, context.DeadlineExceeded)
	case <-time.After(30 * time.Second):
		s.Fail("startSequencer left an RPC unbounded")
	}
}

// Those RPCs also have to be cancellable, so Stop() cannot block behind one: stopSequencer already
// passes shutdownCtx for the same reason, and issue #22094's other half is an unbounded shutdown.
//
// Outside the suite: cancelling shutdownCtx retires the control loop, which the suite's action
// synchronisation needs alive.
func TestStartSequencerRPCsFollowShutdown(t *testing.T) {
	cfg := mockConfig(t)
	ctrl := &clientmocks.SequencerControl{}
	cons := &consensusmocks.Consensus{}
	cons.EXPECT().ServerID().Return("SequencerA").Maybe()
	cons.EXPECT().LatestUnsafePayload().Return(unsafePayload(12), nil)
	ctrl.EXPECT().LatestUnsafeBlock(mock.Anything).RunAndReturn(func(ctx context.Context) (eth.BlockInfo, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	oc, err := NewOpConductor(context.Background(), &cfg, testlog.Logger(t, log.LevelDebug),
		&metrics.NoopMetricsImpl{}, "v0.0.1", ctrl, cons, &healthmocks.HealthMonitor{})
	require.NoError(t, err)
	// Start() would also bring up the RPC server and the control loop; only the shutdown context
	// matters here.
	oc.shutdownCtx, oc.shutdownCancel = context.WithCancel(context.Background())
	oc.healthy.Store(true)
	oc.unsafeHeadCatchUpTimeout = time.Hour

	done := make(chan error, 1)
	go func() { done <- oc.startSequencer() }()
	oc.shutdownCancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(30 * time.Second):
		require.Fail(t, "startSequencer did not derive its context from shutdownCtx")
	}
}
