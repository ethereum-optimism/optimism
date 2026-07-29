package keccak

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var stubChallengePeriod = uint64(3600)

func TestScheduleNextCheck(t *testing.T) {
	ctx := context.Background()
	currentTimestamp := uint64(1240)
	logger := testlog.Logger(t, log.LevelInfo)
	preimage1 := keccakTypes.LargePreimageMetaData{ // Incomplete so won't be verified
		LargePreimageIdent: keccakTypes.LargePreimageIdent{
			Claimant: common.Address{0xab},
			UUID:     big.NewInt(111),
		},
	}
	preimage2 := keccakTypes.LargePreimageMetaData{ // Already countered so won't be verified
		LargePreimageIdent: keccakTypes.LargePreimageIdent{
			Claimant: common.Address{0xab},
			UUID:     big.NewInt(222),
		},
		Timestamp: currentTimestamp - 10,
		Countered: true,
	}
	preimage3 := keccakTypes.LargePreimageMetaData{
		LargePreimageIdent: keccakTypes.LargePreimageIdent{
			Claimant: common.Address{0xdd},
			UUID:     big.NewInt(333),
		},
		Timestamp: currentTimestamp - 10,
	}
	oracle := &stubOracle{
		images: []keccakTypes.LargePreimageMetaData{preimage1, preimage2, preimage3},
	}
	cl := clock.NewDeterministicClock(time.Unix(int64(currentTimestamp), 0))
	challenger := &stubChallenger{}
	scheduler := NewLargePreimageScheduler(logger, metrics.NoopMetrics, cl, OracleSourceArray{oracle}, challenger)
	scheduler.Start(ctx)
	defer scheduler.Close()
	err := scheduler.Schedule(common.Hash{0xaa}, 3)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return oracle.GetPreimagesCount() == 1
	}, 10*time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		verified := challenger.Checked()
		t.Logf("Checked preimages: %v", verified)
		return len(verified) == 1 && verified[0] == preimage3
	}, 10*time.Second, 10*time.Millisecond, "Did not verify preimage")
}

type stubOracle struct {
	m                 sync.Mutex
	addr              common.Address
	getPreimagesCount int
	images            []keccakTypes.LargePreimageMetaData
	treeRoots         map[keccakTypes.LargePreimageIdent]common.Hash
}

func (s *stubOracle) ChallengePeriod(_ context.Context) (uint64, error) {
	return stubChallengePeriod, nil
}

func (s *stubOracle) GetInputDataBlocks(_ context.Context, _ rpcblock.Block, _ keccakTypes.LargePreimageIdent) ([]uint64, error) {
	panic("not supported")
}

func (s *stubOracle) DecodeInputData(_ []byte) (*big.Int, keccakTypes.InputData, error) {
	panic("not supported")
}

func (s *stubOracle) Addr() common.Address {
	return s.addr
}

func (s *stubOracle) GetActivePreimages(_ context.Context, _ common.Hash) ([]keccakTypes.LargePreimageMetaData, error) {
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

func (s *stubOracle) ChallengeTx(_ keccakTypes.LargePreimageIdent, _ keccakTypes.Challenge) (txmgr.TxCandidate, error) {
	panic("not supported")
}

func (s *stubOracle) GetProposalTreeRoot(_ context.Context, _ rpcblock.Block, ident keccakTypes.LargePreimageIdent) (common.Hash, error) {
	root, ok := s.treeRoots[ident]
	if ok {
		return root, nil
	}
	return common.Hash{}, errors.New("unknown tree root")
}

type stubChallenger struct {
	m       sync.Mutex
	checked []keccakTypes.LargePreimageMetaData
}

func (s *stubChallenger) Challenge(_ context.Context, _ common.Hash, _ Oracle, preimages []keccakTypes.LargePreimageMetaData) error {
	s.m.Lock()
	defer s.m.Unlock()
	s.checked = append(s.checked, preimages...)
	return nil
}

func (s *stubChallenger) Checked() []keccakTypes.LargePreimageMetaData {
	s.m.Lock()
	defer s.m.Unlock()
	v := make([]keccakTypes.LargePreimageMetaData, len(s.checked))
	copy(v, s.checked)
	return v
}

type OracleSourceArray []keccakTypes.LargePreimageOracle

func (o OracleSourceArray) Oracles() []keccakTypes.LargePreimageOracle {
	return o
}
