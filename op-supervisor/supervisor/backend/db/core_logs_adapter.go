package db

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	interoptypes "github.com/ethereum-optimism/optimism/op-core/interop/types"
	"github.com/ethereum-optimism/optimism/op-core/persistence/dberrors"
	corelogs "github.com/ethereum-optimism/optimism/op-core/persistence/logsdb"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/reads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// mapCoreErr maps core interop-type errors to supervisor types errors to preserve semantics.
func mapCoreErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, dberrors.ErrFuture):
		return types.ErrFuture
	case errors.Is(err, dberrors.ErrConflict):
		return types.ErrConflict
	case errors.Is(err, dberrors.ErrSkipped):
		return types.ErrSkipped
	case errors.Is(err, dberrors.ErrDataCorruption):
		return types.ErrDataCorruption
	case errors.Is(err, dberrors.ErrPreviousToFirst):
		return types.ErrPreviousToFirst
	default:
		return err
	}
}

// coreLogDBAdapter wraps a Core logs DB to satisfy the Supervisor LogStorage interface.
type coreLogDBAdapter struct {
	inner *corelogs.DB
}

func NewCoreLogDBAdapter(db *corelogs.DB) *coreLogDBAdapter {
	return &coreLogDBAdapter{inner: db}
}

func (a *coreLogDBAdapter) Close() error { return a.inner.Close() }

func (a *coreLogDBAdapter) IsEmpty() bool { return a.inner.IsEmpty() }

func (a *coreLogDBAdapter) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *types.ExecutingMessage) error {
	var em *interoptypes.ExecutingMessage
	if execMsg != nil {
		em = &interoptypes.ExecutingMessage{
			ChainID:   execMsg.ChainID,
			BlockNum:  execMsg.BlockNum,
			LogIdx:    execMsg.LogIdx,
			Timestamp: execMsg.Timestamp,
			Checksum:  interoptypes.MessageChecksum(execMsg.Checksum),
		}
	}
	return a.inner.AddLog(logHash, parentBlock, logIdx, em)
}

func (a *coreLogDBAdapter) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	return a.inner.SealBlock(parentHash, block, timestamp)
}

// invalidatorAdapter adapts Supervisor reads.Invalidator to Core reads.Invalidator
type invalidatorAdapter struct {
	inner reads.Invalidator
}

func (ia invalidatorAdapter) TryInvalidate(rule corelogs.InvalidationRule) (func(), error) {
	switch r := rule.(type) {
	case corelogs.DerivedInvalidation:
		return ia.inner.TryInvalidate(reads.DerivedInvalidation{Timestamp: r.Timestamp})
	case corelogs.SourceInvalidation:
		return ia.inner.TryInvalidate(reads.SourceInvalidation{Number: r.Number})
	case corelogs.InvalidationRules:
		var rules reads.InvalidationRules
		for _, sub := range r {
			switch s := sub.(type) {
			case corelogs.DerivedInvalidation:
				rules = append(rules, reads.DerivedInvalidation{Timestamp: s.Timestamp})
			case corelogs.SourceInvalidation:
				rules = append(rules, reads.SourceInvalidation{Number: s.Number})
			}
		}
		return ia.inner.TryInvalidate(rules)
	default:
		// Unknown rule type; do nothing
		return func() {}, nil
	}
}

func (a *coreLogDBAdapter) Rewind(inv reads.Invalidator, newHead eth.BlockID) error {
	if err := a.inner.Rewind(invalidatorAdapter{inner: inv}, newHead); err != nil {
		return mapCoreErr(err)
	}
	return nil
}

func (a *coreLogDBAdapter) LatestSealedBlock() (id eth.BlockID, ok bool) {
	return a.inner.LatestSealedBlock()
}

func (a *coreLogDBAdapter) FindSealedBlock(number uint64) (block types.BlockSeal, err error) {
	bs, err := a.inner.FindSealedBlock(number)
	if err != nil {
		return types.BlockSeal{}, mapCoreErr(err)
	}
	return types.BlockSeal{Hash: bs.Hash, Number: bs.Number, Timestamp: bs.Timestamp}, nil
}

func (a *coreLogDBAdapter) Contains(query types.ContainsQuery) (includedIn types.BlockSeal, err error) {
	cq := interoptypes.ContainsQuery{
		Timestamp: query.Timestamp,
		BlockNum:  query.BlockNum,
		LogIdx:    query.LogIdx,
		Checksum:  interoptypes.MessageChecksum(query.Checksum),
	}
	res, err := a.inner.Contains(cq)
	if err != nil {
		return types.BlockSeal{}, mapCoreErr(err)
	}
	return types.BlockSeal{Hash: res.Hash, Number: res.Number, Timestamp: res.Timestamp}, nil
}

func (a *coreLogDBAdapter) IteratorStartingAt(sealedNum uint64, logsSince uint32) (corelogs.Iterator, error) {
	return a.inner.IteratorStartingAt(sealedNum, logsSince)
}

func (a *coreLogDBAdapter) OpenBlock(blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*types.ExecutingMessage, err error) {
	ref, logCount, coreMsgs, err := a.inner.OpenBlock(blockNum)
	if err != nil {
		return eth.BlockRef{}, 0, nil, mapCoreErr(err)
	}
	var out map[uint32]*types.ExecutingMessage
	if coreMsgs != nil {
		out = make(map[uint32]*types.ExecutingMessage, len(coreMsgs))
		for idx, m := range coreMsgs {
			out[idx] = &types.ExecutingMessage{
				ChainID:   m.ChainID,
				BlockNum:  m.BlockNum,
				LogIdx:    m.LogIdx,
				Timestamp: m.Timestamp,
				Checksum:  types.MessageChecksum(m.Checksum),
			}
		}
	}
	return ref, logCount, out, nil
}
