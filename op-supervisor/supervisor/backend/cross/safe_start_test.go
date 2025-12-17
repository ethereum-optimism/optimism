package cross

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestCrossSafeHazards(t *testing.T) {
	t.Run("empty execMsgs", func(t *testing.T) {
		ssd := &mockSafeStartDeps{}
		chainID := eth.ChainIDFromUInt64(123)
		inL1Source := eth.BlockID{}
		candidate := types.BlockSeal{}
		// when there are no execMsgs,
		// no work is done, and no error is returned
		hazards, err := CrossSafeHazards(ssd, linkerAny{}, newTestLogger(t), chainID, inL1Source, candidate)
		require.NoError(t, err)
		require.Empty(t, hazards.Entries())
	})
	t.Run("executing msg does bad link", func(t *testing.T) {
		ssd := &mockSafeStartDeps{}
		done := false
		linker := depset.LinkCheckFn(func(execInChain eth.ChainID, execInTimestamp uint64, initChainID eth.ChainID, initTimestamp uint64) bool {
			done = true
			require.Equal(t, execInChain, eth.ChainIDFromUInt64(123))
			require.Equal(t, execInTimestamp, uint64(9000))
			require.Equal(t, initChainID, eth.ChainIDFromUInt64(0xbad))
			require.Equal(t, initTimestamp, uint64(42))
			// doesn't matter what is bad, we checked we have all the inputs for all possible cases.
			return false
		})
		chainID := eth.ChainIDFromUInt64(123)
		inL1Source := eth.BlockID{}
		candidate := types.BlockSeal{Timestamp: uint64(9000)}
		ssd.openBlockFn = newOpenBlockFn(&types.ExecutingMessage{
			ChainID:   eth.ChainIDFromUInt64(0xbad),
			Timestamp: 42,
		})
		hazards, err := CrossSafeHazards(ssd, linker, newTestLogger(t), chainID, inL1Source, candidate)
		require.ErrorIs(t, err, types.ErrConflict)
		require.Empty(t, hazards)
		require.True(t, done)
	})
	t.Run("timestamp is greater than candidate", func(t *testing.T) {
		ssd := &mockSafeStartDeps{}
		chainID := eth.ChainIDFromUInt64(123)
		inL1Source := eth.BlockID{}
		candidate := types.BlockSeal{Timestamp: 2}
		ssd.candidate = candidate
		em1 := &types.ExecutingMessage{ChainID: chainID, Timestamp: 10}
		ssd.openBlockFn = newOpenBlockFn(em1)
		// when there is one execMsg, and the timestamp is greater than the candidate,
		// an error is returned
		hazards, err := CrossSafeHazards(ssd, linkerAny{}, newTestLogger(t), chainID, inL1Source, candidate)
		require.ErrorContains(t, err, "breaks timestamp invariant")
		require.Empty(t, hazards)
	})
	t.Run("timestamp is equal, Check returns error", func(t *testing.T) {
		ssd := &mockSafeStartDeps{}
		ssd.checkFn = func() (includedIn types.BlockSeal, err error) {
			return types.BlockSeal{}, errors.New("some error")
		}
		chainID := eth.ChainIDFromUInt64(123)
		inL1Source := eth.BlockID{}
		candidate := types.BlockSeal{Timestamp: 2}
		ssd.candidate = candidate
		em1 := &types.ExecutingMessage{ChainID: chainID, Timestamp: 2}
		ssd.openBlockFn = newOpenBlockFn(em1)
		// when there is one execMsg, and the timestamp is equal to the candidate,
		// and check returns an error,
		// that error is returned
		hazards, err := CrossSafeHazards(ssd, linkerAny{}, newTestLogger(t), chainID, inL1Source, candidate)
		require.ErrorContains(t, err, "some error")
		require.Empty(t, hazards)
	})
	t.Run("timestamp is equal, same hazard twice", func(t *testing.T) {
		ssd := &mockSafeStartDeps{}
		sampleBlockSeal := types.BlockSeal{Number: 3, Hash: common.BytesToHash([]byte{0x02}), Timestamp: 2}
		ssd.checkFn = func() (includedIn types.BlockSeal, err error) {
			return sampleBlockSeal, nil
		}
		chainID := eth.ChainIDFromUInt64(123)
		inL1Source := eth.BlockID{}
		candidate := types.BlockSeal{Number: 0, Hash: common.Hash{}, Timestamp: 2}
		ssd.candidate = candidate
		em1 := &types.ExecutingMessage{ChainID: chainID, Timestamp: 2}
		em2 := &types.ExecutingMessage{ChainID: chainID, Timestamp: 2}
		ssd.openBlockFn = func(chainID eth.ChainID, blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*types.ExecutingMessage, err error) {
			if blockNum == candidate.Number {
				return eth.BlockRef{
					Hash:   candidate.Hash,
					Time:   candidate.Timestamp,
					Number: candidate.Number,
				}, 2, map[uint32]*types.ExecutingMessage{0: em1, 1: em2}, nil
			}
			return eth.BlockRef{
				Hash:   sampleBlockSeal.Hash,
				Time:   sampleBlockSeal.Timestamp,
				Number: sampleBlockSeal.Number,
			}, 2, map[uint32]*types.ExecutingMessage{0: em1, 1: em2}, nil
		}
		// when there are two execMsgs, and both are equal time to the candidate,
		// and check returns the same includedIn for both
		// they load the hazards once, and return no error
		hazards, err := CrossSafeHazards(ssd, linkerAny{}, newTestLogger(t), chainID, inL1Source, candidate)
		require.NoError(t, err)
		require.Equal(t, map[eth.ChainID]types.BlockSeal{chainID: sampleBlockSeal}, hazards.Entries())
	})
	t.Run("timestamp is equal, different hazards", func(t *testing.T) {
		logger := newTestLogger(t)
		ssd := &mockSafeStartDeps{}
		// set the check function to return a different BlockSeal for the second call
		sampleBlockSeal := types.BlockSeal{Number: 3, Hash: common.BytesToHash([]byte{0x02}), Timestamp: 2}
		sampleBlockSeal2 := types.BlockSeal{Number: 333, Hash: common.BytesToHash([]byte{0x22}), Timestamp: 2}
		calls := 0
		ssd.checkFn = func() (includedIn types.BlockSeal, err error) {
			defer func() { calls++ }()
			if calls == 0 {
				return sampleBlockSeal, nil
			}
			return sampleBlockSeal2, nil
		}
		chainID := eth.ChainIDFromUInt64(123)
		inL1Source := eth.BlockID{}
		candidate := types.BlockSeal{Timestamp: 2}
		ssd.candidate = candidate
		em1 := &types.ExecutingMessage{ChainID: chainID, Timestamp: 2}
		em2 := &types.ExecutingMessage{ChainID: chainID, Timestamp: 2}
		ssd.openBlockFn = newOpenBlockFn(em1, em2)
		// when there are two execMsgs, and both are equal time to the candidate,
		// and check returns different includedIn for the two,
		// an error is returned
		hazards, err := CrossSafeHazards(ssd, linkerAny{}, logger, chainID, inL1Source, candidate)
		require.ErrorContains(t, err, "but already depend on")
		require.Empty(t, hazards)
	})
	t.Run("timestamp is less, check returns error", func(t *testing.T) {
		logger := newTestLogger(t)
		ssd := &mockSafeStartDeps{}
		ssd.checkFn = func() (includedIn types.BlockSeal, err error) {
			return types.BlockSeal{}, errors.New("some error")
		}
		chainID := eth.ChainIDFromUInt64(123)
		inL1Source := eth.BlockID{}
		candidate := types.BlockSeal{Timestamp: 2}
		ssd.candidate = candidate
		em1 := &types.ExecutingMessage{ChainID: chainID, Timestamp: 1}
		ssd.openBlockFn = newOpenBlockFn(em1)
		// when there is one execMsg, and the timestamp is less than the candidate,
		// and check returns an error,
		// that error is returned
		hazards, err := CrossSafeHazards(ssd, linkerAny{}, logger, chainID, inL1Source, candidate)
		require.ErrorContains(t, err, "some error")
		require.Empty(t, hazards)
	})
	t.Run("timestamp is less, DerivedToSource returns error", func(t *testing.T) {
		logger := newTestLogger(t)
		ssd := &mockSafeStartDeps{}
		sampleBlockSeal := types.BlockSeal{Number: 3, Hash: common.BytesToHash([]byte{0x02}), Timestamp: 2}
		ssd.checkFn = func() (includedIn types.BlockSeal, err error) {
			return sampleBlockSeal, nil
		}
		ssd.derivedToSrcFn = func() (source types.BlockSeal, err error) {
			return types.BlockSeal{}, errors.New("some error")
		}
		chainID := eth.ChainIDFromUInt64(123)
		inL1Source := eth.BlockID{}
		candidate := types.BlockSeal{Timestamp: 2}
		ssd.candidate = candidate
		em1 := &types.ExecutingMessage{ChainID: chainID, Timestamp: 1}
		ssd.openBlockFn = newOpenBlockFn(em1)
		// when there is one execMsg, and the timestamp is less than the candidate,
		// and DerivedToSource returns aan error,
		// that error is returned
		hazards, err := CrossSafeHazards(ssd, linkerAny{}, logger, chainID, inL1Source, candidate)
		require.ErrorContains(t, err, "some error")
		require.Empty(t, hazards)
	})
	t.Run("timestamp is less, DerivedToSource Number is greater", func(t *testing.T) {
		logger := newTestLogger(t)
		ssd := &mockSafeStartDeps{}
		sampleBlockSeal := types.BlockSeal{Number: 3, Hash: common.BytesToHash([]byte{0x02})}
		ssd.checkFn = func() (includedIn types.BlockSeal, err error) {
			return sampleBlockSeal, nil
		}
		sampleSource := types.BlockSeal{Number: 4, Hash: common.BytesToHash([]byte{0x03})}
		ssd.derivedToSrcFn = func() (source types.BlockSeal, err error) {
			return sampleSource, nil
		}
		chainID := eth.ChainIDFromUInt64(123)
		inL1Source := eth.BlockID{}
		candidate := types.BlockSeal{Timestamp: 2}
		ssd.candidate = candidate
		em1 := &types.ExecutingMessage{ChainID: chainID, Timestamp: 1}
		ssd.openBlockFn = newOpenBlockFn(em1)
		// when there is one execMsg, and the timestamp is less than the candidate,
		// and DerivedToSource returns a BlockSeal with a greater Number than the inL1Source,
		// an error is returned as a ErrOutOfScope
		hazards, err := CrossSafeHazards(ssd, linkerAny{}, logger, chainID, inL1Source, candidate)
		require.ErrorIs(t, err, types.ErrOutOfScope)
		require.Empty(t, hazards)
	})
	t.Run("timestamp is less, DerivedToSource Number less", func(t *testing.T) {
		logger := newTestLogger(t)
		ssd := &mockSafeStartDeps{}
		// Don't need checkFn since we want no hazards
		sampleSource := types.BlockSeal{Number: 1, Hash: common.BytesToHash([]byte{0x03}), Timestamp: 3}
		ssd.derivedToSrcFn = func() (source types.BlockSeal, err error) {
			return sampleSource, nil
		}
		chainID := eth.ChainIDFromUInt64(123)
		inL1Source := eth.BlockID{Number: 10}
		candidate := types.BlockSeal{Timestamp: 2}
		ssd.candidate = candidate
		em1 := &types.ExecutingMessage{ChainID: chainID, Timestamp: 1}
		ssd.openBlockFn = newOpenBlockFn(em1)
		// when there is one execMsg, and the timestamp is less than the candidate,
		// and DerivedToSource returns a BlockSeal with a smaller Number than the inL1Source,
		// no error is returned
		hazards, err := CrossSafeHazards(ssd, linkerAny{}, logger, chainID, inL1Source, candidate)
		require.NoError(t, err)
		require.Empty(t, hazards.Entries())
	})
	t.Run("timestamp is less, DerivedToSource Number equal", func(t *testing.T) {
		logger := newTestLogger(t)
		ssd := &mockSafeStartDeps{}
		// Don't need checkFn since we want no hazards
		ssd.derivedToSrcFn = func() (source types.BlockSeal, err error) {
			return types.BlockSeal{Number: 1, Hash: common.BytesToHash([]byte{0x03}), Timestamp: 1}, nil
		}
		chainID := eth.ChainIDFromUInt64(123)
		inL1Source := eth.BlockID{Number: 1}
		candidate := types.BlockSeal{Timestamp: 2}
		ssd.candidate = candidate
		em1 := &types.ExecutingMessage{ChainID: chainID, Timestamp: 1}
		ssd.openBlockFn = newOpenBlockFn(em1)
		// when there is one execMsg, and the timestamp is less than the candidate,
		// and DerivedToSource returns a BlockSeal with a equal to the Number of inL1Source,
		// no error is returned
		hazards, err := CrossSafeHazards(ssd, linkerAny{}, logger, chainID, inL1Source, candidate)
		require.NoError(t, err)
		require.Empty(t, hazards.Entries())
	})
	t.Run("message close to expiry", func(t *testing.T) {
		logger := newTestLogger(t)
		ssd := &mockSafeStartDeps{}
		chainID := eth.ChainIDFromUInt64(123)
		inL1Source := eth.BlockID{Number: 1}
		candidate := types.BlockSeal{Timestamp: 11}
		em1 := &types.ExecutingMessage{ChainID: chainID, Timestamp: 1}
		ssd.openBlockFn = newOpenBlockFn(em1)
		linker := depset.LinkCheckFn(func(execInChain eth.ChainID, execInTimestamp uint64, initChainID eth.ChainID, initTimestamp uint64) bool {
			require.Equal(t, uint64(11), execInTimestamp)
			require.Equal(t, uint64(1), initTimestamp)
			return true
		})
		// when there is one execMsg, and the timestamp is less than the candidate,
		// and DerivedToSource returns a BlockSeal with a equal to the Number of inL1Source,
		// no error is returned
		hazards, err := CrossSafeHazards(ssd, linker, logger, chainID, inL1Source, candidate)
		require.NoError(t, err)
		require.Empty(t, hazards.Entries())
	})
}

type mockSafeStartDeps struct {
	checkFn        func() (includedIn types.BlockSeal, err error)
	derivedToSrcFn func() (source types.BlockSeal, err error)
	openBlockFn    func(chainID eth.ChainID, blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*types.ExecutingMessage, err error)
	candidate      types.BlockSeal
}

func (m *mockSafeStartDeps) Contains(chain eth.ChainID, query types.ContainsQuery) (includedIn types.BlockSeal, err error) {
	if m.checkFn != nil {
		return m.checkFn()
	}
	return types.BlockSeal{}, nil
}

func (m *mockSafeStartDeps) CrossDerivedToSource(chainID eth.ChainID, derived eth.BlockID) (source types.BlockSeal, err error) {
	if m.derivedToSrcFn != nil {
		return m.derivedToSrcFn()
	}
	return types.BlockSeal{}, nil
}

func (m *mockSafeStartDeps) OpenBlock(chainID eth.ChainID, blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*types.ExecutingMessage, err error) {
	if m.openBlockFn != nil {
		return m.openBlockFn(chainID, blockNum)
	}
	execMsgs = make(map[uint32]*types.ExecutingMessage)
	return eth.BlockRef{Time: m.candidate.Timestamp}, uint32(len(execMsgs)), execMsgs, nil
}
