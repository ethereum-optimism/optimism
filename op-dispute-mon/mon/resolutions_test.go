package mon

import (
	"testing"
	"time"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-dispute-mon/mon/types"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestResolutionMonitor_CheckResolutions(t *testing.T) {
	r, cl, m := newTestResolutionMonitor(t)
	games := newTestGames(uint64(cl.Now().Unix()))
	r.CheckResolutions(games)

	require.Equal(t, 1, m.calls[true][true])
	require.Equal(t, 1, m.calls[true][false])
	require.Equal(t, 1, m.calls[false][true])
	require.Equal(t, 1, m.calls[false][false])
}

func newTestResolutionMonitor(t *testing.T) (*ResolutionMonitor, *clock.DeterministicClock, *stubResolutionMetrics) {
	logger := testlog.Logger(t, log.LvlInfo)
	cl := clock.NewDeterministicClock(time.Unix(int64(time.Hour.Seconds()), 0))
	metrics := &stubResolutionMetrics{}
	return NewResolutionMonitor(logger, metrics, cl), cl, metrics
}

type stubResolutionMetrics struct {
	calls map[bool]map[bool]int // completed -> max duration reached -> call count
}

func (s *stubResolutionMetrics) RecordGameResolutionStatus(complete bool, maxDurationReached bool, count int) {
	if s.calls == nil {
		s.calls = make(map[bool]map[bool]int)
		s.calls[true] = make(map[bool]int)
		s.calls[false] = make(map[bool]int)
	}
	s.calls[complete][maxDurationReached] += count
}

func newTestGames(duration uint64) []*types.EnrichedGameData {
	newTestGame := func(duration uint64, status gameTypes.GameStatus) *types.EnrichedGameData {
		return &types.EnrichedGameData{MaxClockDuration: duration, Status: status}
	}
	return []*types.EnrichedGameData{
		newTestGame(duration, gameTypes.GameStatusInProgress),
		newTestGame(duration*10, gameTypes.GameStatusInProgress),
		newTestGame(duration, gameTypes.GameStatusDefenderWon),
		newTestGame(duration*10, gameTypes.GameStatusChallengerWon),
	}
}
