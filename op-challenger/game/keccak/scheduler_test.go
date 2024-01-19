package keccak

import (
	"context"
	"math/big"
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
	preimage1 := types.LargePreimageMetaData{ // Incomplete so won't be verified
		LargePreimageIdent: types.LargePreimageIdent{
			Claimant: common.Address{0xab},
			UUID:     big.NewInt(111),
		},
	}
	preimage2 := types.LargePreimageMetaData{ // Already countered so won't be verified
		LargePreimageIdent: types.LargePreimageIdent{
			Claimant: common.Address{0xab},
			UUID:     big.NewInt(222),
		},
		Timestamp: 1234,
		Countered: true,
	}
	preimage3 := types.LargePreimageMetaData{
		LargePreimageIdent: types.LargePreimageIdent{
			Claimant: common.Address{0xdd},
			UUID:     big.NewInt(333),
		},
		Timestamp: 1234,
	}
	oracle := &stubOracle{
		images: []types.LargePreimageMetaData{preimage1, preimage2, preimage3},
	}
	verifier := &stubVerifier{}
	scheduler := NewLargePreimageScheduler(logger, []types.LargePreimageOracle{oracle}, verifier)
	scheduler.Start(ctx)
	defer scheduler.Close()
	err := scheduler.Schedule(common.Hash{0xaa}, 3)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return oracle.GetPreimagesCount() == 1
	}, 10*time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		verified := verifier.Verified()
		t.Logf("Verified preimages: %v", verified)
		return len(verified) == 1 && verified[0] == preimage3
	}, 10*time.Second, 10*time.Millisecond, "Did not verify preimage")
}

type stubOracle struct {
	m                 sync.Mutex
	addr              common.Address
	getPreimagesCount int
	images            []types.LargePreimageMetaData
}

func (s *stubOracle) Addr() common.Address {
	return s.addr
}

func (s *stubOracle) GetActivePreimages(_ context.Context, _ common.Hash) ([]types.LargePreimageMetaData, error) {
	s.m.Lock()
	defer s.m.Unlock()
	s.getPreimagesCount++
	return s.images, nil
}

func (s *stubOracle) GetPreimagesCount() int {
	s.m.Lock()
	defer s.m.Unlock()
	return s.getPreimagesCount
}

type stubVerifier struct {
	m        sync.Mutex
	verified []types.LargePreimageMetaData
}

func (s *stubVerifier) Verify(_ context.Context, _ types.LargePreimageOracle, image types.LargePreimageMetaData) {
	s.m.Lock()
	defer s.m.Unlock()
	s.verified = append(s.verified, image)
}

func (s *stubVerifier) Verified() []types.LargePreimageMetaData {
	s.m.Lock()
	defer s.m.Unlock()
	v := make([]types.LargePreimageMetaData, len(s.verified))
	copy(v, s.verified)
	return v
}
