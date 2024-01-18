package keccak

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestScheduleNextCheck(t *testing.T) {
	ctx := context.Background()
	logger := testlog.Logger(t, log.LvlInfo)
	oracle := &stubOracle{}
	scheduler := NewLargePreimageScheduler(logger, []types.LargePreimageOracle{oracle})
	scheduler.Start(ctx)
	defer scheduler.Close()
	err := scheduler.Schedule(common.Hash{0xaa}, 3)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return oracle.GetPreimagesCount() == 1
	}, 10*time.Second, 10*time.Millisecond)
}

type stubOracle struct {
	m                 sync.Mutex
	addr              common.Address
	getPreimagesCount int
}

func (s *stubOracle) Addr() common.Address {
	return s.addr
}

func (s *stubOracle) GetActivePreimages(_ context.Context, _ common.Hash) ([]types.LargePreimageMetaData, error) {
	s.m.Lock()
	defer s.m.Unlock()
	s.getPreimagesCount++
	return nil, nil
}

func (s *stubOracle) GetPreimagesCount() int {
	s.m.Lock()
	defer s.m.Unlock()
	return s.getPreimagesCount
}
