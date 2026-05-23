package cross

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"

	"github.com/ethereum-optimism/optimism/op-core/interop"
	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
)

func TestHazardSafeFrontierChecks(t *testing.T) {
	t.Run("empty hazards", func(t *testing.T) {
		sfcd := &mockSafeFrontierCheckDeps{}
		l1Source := eth.BlockID{}
		hazards := map[eth.ChainID]messages.BlockSeal{}
		// when there are no hazards,
		// no work is done, and no error is returned
		err := HazardSafeFrontierChecks(sfcd, l1Source, NewHazardSetFromEntries(hazards))
		require.NoError(t, err)
	})
	t.Run("initSource in scope", func(t *testing.T) {
		sfcd := &mockSafeFrontierCheckDeps{}
		sfcd.crossSourceFn = func() (messages.BlockSeal, error) {
			return messages.BlockSeal{Number: 1}, nil
		}
		l1Source := eth.BlockID{Number: 2}
		hazards := map[eth.ChainID]messages.BlockSeal{eth.ChainIDFromUInt64(123): {}}
		// when there is one hazard, and CrossSource returns a BlockSeal within scope
		// (ie the hazard's block number is less than or equal to the source block number),
		// no error is returned
		err := HazardSafeFrontierChecks(sfcd, l1Source, NewHazardSetFromEntries(hazards))
		require.NoError(t, err)
	})
	t.Run("initSource out of scope", func(t *testing.T) {
		sfcd := &mockSafeFrontierCheckDeps{}
		sfcd.crossSourceFn = func() (messages.BlockSeal, error) {
			return messages.BlockSeal{Number: 3}, nil
		}
		l1Source := eth.BlockID{Number: 2}
		hazards := map[eth.ChainID]messages.BlockSeal{eth.ChainIDFromUInt64(123): {}}
		// when there is one hazard, and CrossSource returns a BlockSeal out of scope
		// (ie the hazard's block number is greater than the source block number),
		// an error is returned as a ErrOutOfScope
		err := HazardSafeFrontierChecks(sfcd, l1Source, NewHazardSetFromEntries(hazards))
		require.ErrorIs(t, err, interop.ErrOutOfScope)
	})
	t.Run("errFuture: candidate cross safe failure", func(t *testing.T) {
		sfcd := &mockSafeFrontierCheckDeps{}
		sfcd.crossSourceFn = func() (messages.BlockSeal, error) {
			return messages.BlockSeal{Number: 3}, interop.ErrFuture
		}
		sfcd.candidateCrossSafeFn = func() (candidate types.DerivedBlockRefPair, err error) {
			return types.DerivedBlockRefPair{
					Source:  eth.BlockRef{},
					Derived: eth.BlockRef{Number: 3, Hash: common.BytesToHash([]byte{0x01})},
				},
				errors.New("some error")
		}
		l1Source := eth.BlockID{}
		hazards := map[eth.ChainID]messages.BlockSeal{eth.ChainIDFromUInt64(123): {}}
		// when there is one hazard, and CrossSource returns an ErrFuture,
		// and CandidateCrossSafe returns an error,
		// the error from CandidateCrossSafe is returned
		err := HazardSafeFrontierChecks(sfcd, l1Source, NewHazardSetFromEntries(hazards))
		require.ErrorContains(t, err, "some error")
	})
	t.Run("errFuture: expected block does not match candidate", func(t *testing.T) {
		sfcd := &mockSafeFrontierCheckDeps{}
		sfcd.crossSourceFn = func() (messages.BlockSeal, error) {
			return messages.BlockSeal{}, interop.ErrFuture
		}
		sfcd.candidateCrossSafeFn = func() (candidate types.DerivedBlockRefPair, err error) {
			return types.DerivedBlockRefPair{
				Source:  eth.BlockRef{},
				Derived: eth.BlockRef{Number: 3, Hash: common.BytesToHash([]byte{0x01})},
			}, nil
		}
		l1Source := eth.BlockID{}
		hazards := map[eth.ChainID]messages.BlockSeal{eth.ChainIDFromUInt64(123): {Number: 3, Hash: common.BytesToHash([]byte{0x02})}}
		// when there is one hazard, and CrossSource returns an ErrFuture,
		// and CandidateCrossSafe returns a candidate that does not match the hazard,
		// (ie the candidate's block number is the same as the hazard's block number, but the hashes are different),
		// an error is returned as a ErrConflict
		err := HazardSafeFrontierChecks(sfcd, l1Source, NewHazardSetFromEntries(hazards))
		require.ErrorIs(t, err, interop.ErrConflict)
	})
	t.Run("errFuture: local-safe hazard out of scope", func(t *testing.T) {
		sfcd := &mockSafeFrontierCheckDeps{}
		sfcd.crossSourceFn = func() (messages.BlockSeal, error) {
			return messages.BlockSeal{}, interop.ErrFuture
		}
		sfcd.candidateCrossSafeFn = func() (candidate types.DerivedBlockRefPair, err error) {
			return types.DerivedBlockRefPair{
					Source:  eth.BlockRef{Number: 9},
					Derived: eth.BlockRef{},
				},
				nil
		}
		l1Source := eth.BlockID{Number: 8}
		hazards := map[eth.ChainID]messages.BlockSeal{eth.ChainIDFromUInt64(123): {Number: 3, Hash: common.BytesToHash([]byte{0x02})}}
		// when there is one hazard, and CrossSource returns an ErrFuture,
		// and the initSource is out of scope,
		// an error is returned as a ErrOutOfScope
		err := HazardSafeFrontierChecks(sfcd, l1Source, NewHazardSetFromEntries(hazards))
		require.ErrorIs(t, err, interop.ErrOutOfScope)
	})
	t.Run("CrossSource Error", func(t *testing.T) {
		sfcd := &mockSafeFrontierCheckDeps{}
		sfcd.crossSourceFn = func() (messages.BlockSeal, error) {
			return messages.BlockSeal{}, errors.New("some error")
		}
		sfcd.candidateCrossSafeFn = func() (candidate types.DerivedBlockRefPair, err error) {
			return types.DerivedBlockRefPair{
				Source:  eth.BlockRef{Number: 9},
				Derived: eth.BlockRef{},
			}, nil
		}
		l1Source := eth.BlockID{Number: 8}
		hazards := map[eth.ChainID]messages.BlockSeal{eth.ChainIDFromUInt64(123): {Number: 3, Hash: common.BytesToHash([]byte{0x02})}}
		// when there is one hazard, and CrossSource returns an ErrFuture,
		// and the initSource is out of scope,
		// an error is returned as a ErrOutOfScope
		err := HazardSafeFrontierChecks(sfcd, l1Source, NewHazardSetFromEntries(hazards))
		require.ErrorContains(t, err, "some error")
	})
	t.Run("Hazard Chain Out of Scope is translated to ErrFuture", func(t *testing.T) {
		sfcd := &mockSafeFrontierCheckDeps{}
		sfcd.crossSourceFn = func() (messages.BlockSeal, error) {
			return messages.BlockSeal{}, interop.ErrFuture
		}
		sfcd.candidateCrossSafeFn = func() (candidate types.DerivedBlockRefPair, err error) {
			return types.DerivedBlockRefPair{
				Source:  eth.BlockRef{Number: 9},
				Derived: eth.BlockRef{},
			}, interop.ErrOutOfScope
		}
		l1Source := eth.BlockID{Number: 8}
		hazards := map[eth.ChainID]messages.BlockSeal{eth.ChainIDFromUInt64(123): {Number: 3, Hash: common.BytesToHash([]byte{0x02})}}
		// when there is one hazard, and CrossSource returns an ErrFuture,
		// and the initSource is out of scope,
		// an error is returned as a ErrOutOfScope
		err := HazardSafeFrontierChecks(sfcd, l1Source, NewHazardSetFromEntries(hazards))
		require.ErrorIs(t, err, interop.ErrFuture)
	})
}

type mockSafeFrontierCheckDeps struct {
	candidateCrossSafeFn func() (candidate types.DerivedBlockRefPair, err error)
	crossSourceFn        func() (source messages.BlockSeal, err error)
}

func (m *mockSafeFrontierCheckDeps) CandidateCrossSafe(chain eth.ChainID) (candidate types.DerivedBlockRefPair, err error) {
	if m.candidateCrossSafeFn != nil {
		return m.candidateCrossSafeFn()
	}
	return types.DerivedBlockRefPair{}, nil
}

func (m *mockSafeFrontierCheckDeps) CrossDerivedToSource(chainID eth.ChainID, derived eth.BlockID) (source messages.BlockSeal, err error) {
	if m.crossSourceFn != nil {
		return m.crossSourceFn()
	}
	return messages.BlockSeal{}, nil
}
