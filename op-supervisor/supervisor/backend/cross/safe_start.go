package cross

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-core/interop"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

type SafeStartDeps interface {
	Contains(chain eth.ChainID, query messages.ContainsQuery) (includedIn messages.BlockSeal, err error)

	CrossDerivedToSource(chainID eth.ChainID, derived eth.BlockID) (source messages.BlockSeal, err error)

	OpenBlock(chainID eth.ChainID, blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*messages.ExecutingMessage, err error)
}

// CrossSafeHazards checks if the given messages all exist and pass invariants.
// It returns a hazard-set: if any intra-block messaging happened,
// these hazard blocks have to be verified.
func CrossSafeHazards(d SafeStartDeps, linker depset.LinkChecker, logger log.Logger, chainID eth.ChainID, inL1Source eth.BlockID, candidate messages.BlockSeal) (*HazardSet, error) {
	safeDeps := &SafeHazardDeps{SafeStartDeps: d, inL1Source: inL1Source}
	return NewHazardSet(safeDeps, linker, logger, chainID, candidate)
}

type SafeHazardDeps struct {
	SafeStartDeps
	inL1Source eth.BlockID
}

func (d *SafeHazardDeps) IsCrossValidBlock(chainID eth.ChainID, derived eth.BlockID) error {
	initSource, err := d.CrossDerivedToSource(chainID, derived)
	if err != nil {
		return fmt.Errorf("non-cross-safe block %s: %w", derived, err)
	}
	if initSource.Number > d.inL1Source.Number {
		return fmt.Errorf("block %s derived from %s which is not in cross-safe scope %s: %w",
			derived, initSource, d.inL1Source, interop.ErrOutOfScope)
	}
	return nil
}
