package p2p

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

type stubScoreBookUpdate struct {
	id   peer.ID
	diff store.ScoreDiff
}
type stubScoreBook struct {
	err     error
	scores  map[peer.ID]store.PeerScores
	updates chan stubScoreBookUpdate
}

func (s *stubScoreBook) GetPeerScores(id peer.ID) (store.PeerScores, error) {
	if s.err != nil {
		return store.PeerScores{}, s.err
	}
	scores, ok := s.scores[id]
	if !ok {
		return store.PeerScores{}, nil
	}
	return scores, nil
}

func (s *stubScoreBook) SetScore(id peer.ID, diff store.ScoreDiff) (store.PeerScores, error) {
	s.updates <- stubScoreBookUpdate{id, diff}
	return s.GetPeerScores(id)
}

type appScoreTestData struct {
	ctx       context.Context
	logger    log.Logger
	clock     *clock.DeterministicClock
	peers     []peer.ID
	scorebook *stubScoreBook
}

func (a *appScoreTestData) WaitForNextScoreBookUpdate(t *testing.T) stubScoreBookUpdate {
	ctx, cancelFunc := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancelFunc()
	select {
	case update := <-a.scorebook.updates:
		return update
	case <-ctx.Done():
		t.Fatal("Did not receive expected scorebook update")
		return stubScoreBookUpdate{}
	}
}

func setupPeerApplicationScorerTest(t *testing.T, params *ApplicationScoreParams) (*appScoreTestData, *peerApplicationScorer) {
	data := &appScoreTestData{
		ctx:    context.Background(),
		logger: testlog.Logger(t, log.LvlInfo),
		clock:  clock.NewDeterministicClock(time.UnixMilli(1000)),
		peers:  []peer.ID{},
		scorebook: &stubScoreBook{
			scores:  make(map[peer.ID]store.PeerScores),
			updates: make(chan stubScoreBookUpdate, 10),
		},
	}
	appScorer := newPeerApplicationScorer(data.ctx, data.logger, data.clock, params, data.scorebook, func() []peer.ID {
		return data.peers
	})
	return data, appScorer
}

func TestIncrementValidResponses(t *testing.T) {
	data, appScorer := setupPeerApplicationScorerTest(t, &ApplicationScoreParams{
		ValidResponseCap: 10,
	})

	appScorer.onValidResponse("aaa")
	require.Len(t, data.scorebook.updates, 1)
	update := <-data.scorebook.updates
	require.Equal(t, stubScoreBookUpdate{peer.ID("aaa"), store.IncrementValidResponses{Cap: 10}}, update)
}

func TestIncrementErrorResponses(t *testing.T) {
	data, appScorer := setupPeerApplicationScorerTest(t, &ApplicationScoreParams{
		ErrorResponseCap: 10,
	})

	appScorer.onResponseError("aaa")
	require.Len(t, data.scorebook.updates, 1)
	update := <-data.scorebook.updates
	require.Equal(t, stubScoreBookUpdate{peer.ID("aaa"), store.IncrementErrorResponses{Cap: 10}}, update)
}

func TestIncrementRejectedPayloads(t *testing.T) {
	data, appScorer := setupPeerApplicationScorerTest(t, &ApplicationScoreParams{
		RejectedPayloadCap: 10,
	})

	appScorer.onRejectedPayload("aaa")
	require.Len(t, data.scorebook.updates, 1)
	update := <-data.scorebook.updates
	require.Equal(t, stubScoreBookUpdate{peer.ID("aaa"), store.IncrementRejectedPayloads{Cap: 10}}, update)
}

func TestApplicationScore(t *testing.T) {
	data, appScorer := setupPeerApplicationScorerTest(t, &ApplicationScoreParams{
		ValidResponseWeight:   0.8,
		ErrorResponseWeight:   0.6,
		RejectedPayloadWeight: 0.4,
	})

	peerScore := store.PeerScores{
		ReqResp: store.ReqRespScores{
			ValidResponses:   1,
			ErrorResponses:   2,
			RejectedPayloads: 3,
		},
	}
	data.scorebook.scores["aaa"] = peerScore
	score := appScorer.ApplicationScore("aaa")
	require.Equal(t, 1*0.8+2*0.6+3*0.4, score)
}

func TestApplicationScoreZeroWhenScoreDoesNotLoad(t *testing.T) {
	data, appScorer := setupPeerApplicationScorerTest(t, &ApplicationScoreParams{})

	data.scorebook.err = errors.New("boom")
	score := appScorer.ApplicationScore("aaa")
	require.Zero(t, score)
}

func TestDecayScoresAfterDecayInterval(t *testing.T) {
	params := &ApplicationScoreParams{
		ValidResponseDecay:   0.8,
		ErrorResponseDecay:   0.7,
		RejectedPayloadDecay: 0.3,
		DecayToZero:          0.1,
		DecayInterval:        90 * time.Second,
	}
	data, appScorer := setupPeerApplicationScorerTest(t, params)
	data.peers = []peer.ID{"aaa", "bbb"}

	expectedDecay := &store.DecayApplicationScores{
		ValidResponseDecay:   0.8,
		ErrorResponseDecay:   0.7,
		RejectedPayloadDecay: 0.3,
		DecayToZero:          0.1,
	}

	appScorer.start()
	defer appScorer.stop()

	data.clock.WaitForNewPendingTaskWithTimeout(30 * time.Second)

	data.clock.AdvanceTime(params.DecayInterval)

	require.Equal(t, stubScoreBookUpdate{id: "aaa", diff: expectedDecay}, data.WaitForNextScoreBookUpdate(t))
	require.Equal(t, stubScoreBookUpdate{id: "bbb", diff: expectedDecay}, data.WaitForNextScoreBookUpdate(t))
}
