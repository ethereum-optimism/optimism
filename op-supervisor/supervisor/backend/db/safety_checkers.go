package db

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/entrydb"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const (
	Unsafe    = "unsafe"
	Safe      = "safe"
	Finalized = "finalized"
)

// SafetyChecker is an interface for checking the safety of a log entry
// and updating the local head for a chain.
type SafetyChecker interface {
	LocalHeadForChain(chainID types.ChainID) entrydb.EntryIdx
	CrossHeadForChain(chainID types.ChainID) entrydb.EntryIdx
	Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) bool
	Update(chain types.ChainID, index entrydb.EntryIdx) heads.OperationFn
	Name() string
	SafetyLevel() types.SafetyLevel
}

// unsafeChecker is a SafetyChecker that uses the unsafe head as the view into the database
type unsafeChecker struct {
	chainsDB *ChainsDB
}

// safeChecker is a SafetyChecker that uses the safe head as the view into the database
type safeChecker struct {
	chainsDB *ChainsDB
}

// finalizedChecker is a SafetyChecker that uses the finalized head as the view into the database
type finalizedChecker struct {
	chainsDB *ChainsDB
}

// NewSafetyChecker creates a new SafetyChecker of the given type
func NewSafetyChecker(t types.SafetyLevel, chainsDB *ChainsDB) SafetyChecker {
	switch t {
	case Unsafe:
		return &unsafeChecker{
			chainsDB: chainsDB,
		}
	case Safe:
		return &safeChecker{
			chainsDB: chainsDB,
		}
	case Finalized:
		return &finalizedChecker{
			chainsDB: chainsDB,
		}
	default:
		panic("unknown safety checker type")
	}
}

// Name returns the safety checker type, using the same strings as the constants used in construction
func (c *unsafeChecker) Name() string {
	return Unsafe
}

func (c *safeChecker) Name() string {
	return Safe
}

func (c *finalizedChecker) Name() string {
	return Finalized
}

// LocalHeadForChain returns the local head for the given chain
// based on the type of SafetyChecker
func (c *unsafeChecker) LocalHeadForChain(chainID types.ChainID) entrydb.EntryIdx {
	heads := c.chainsDB.heads.Current().Get(chainID)
	return heads.Unsafe
}

func (c *safeChecker) LocalHeadForChain(chainID types.ChainID) entrydb.EntryIdx {
	heads := c.chainsDB.heads.Current().Get(chainID)
	return heads.LocalSafe
}

func (c *finalizedChecker) LocalHeadForChain(chainID types.ChainID) entrydb.EntryIdx {
	heads := c.chainsDB.heads.Current().Get(chainID)
	return heads.LocalFinalized
}

// CrossHeadForChain returns the x-head for the given chain
// based on the type of SafetyChecker
func (c *unsafeChecker) CrossHeadForChain(chainID types.ChainID) entrydb.EntryIdx {
	heads := c.chainsDB.heads.Current().Get(chainID)
	return heads.CrossUnsafe
}

func (c *safeChecker) CrossHeadForChain(chainID types.ChainID) entrydb.EntryIdx {
	heads := c.chainsDB.heads.Current().Get(chainID)
	return heads.CrossSafe
}

func (c *finalizedChecker) CrossHeadForChain(chainID types.ChainID) entrydb.EntryIdx {
	heads := c.chainsDB.heads.Current().Get(chainID)
	return heads.CrossFinalized
}

func (c *unsafeChecker) SafetyLevel() types.SafetyLevel {
	return types.CrossUnsafe
}

func (c *safeChecker) SafetyLevel() types.SafetyLevel {
	return types.CrossSafe
}

func (c *finalizedChecker) SafetyLevel() types.SafetyLevel {
	return types.CrossFinalized
}

// check checks if the log entry is safe, provided a local head for the chain
// it is used by the individual SafetyCheckers to determine if a log entry is safe
func check(
	chainsDB *ChainsDB,
	localHead entrydb.EntryIdx,
	chain types.ChainID,
	blockNum uint64,
	logIdx uint32,
	logHash common.Hash) bool {

	// for the Check to be valid, the log must:
	// exist at the blockNum and logIdx
	// have a hash that matches the provided hash (implicit in the Contains call), and
	// be less than or equal to the local head for the chain
	index, err := chainsDB.logDBs[chain].Contains(blockNum, logIdx, logHash)
	if err != nil {
		if errors.Is(err, logs.ErrFuture) {
			return false // TODO(#12031)
		}
		if errors.Is(err, logs.ErrConflict) {
			return false // TODO(#12031)
		}
		return false
	}
	return index <= localHead
}

// Check checks if the log entry is safe, provided a local head for the chain
// it passes on the local head this checker is concerned with, along with its view of the database
func (c *unsafeChecker) Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) bool {
	return check(c.chainsDB, c.LocalHeadForChain(chain), chain, blockNum, logIdx, logHash)
}
func (c *safeChecker) Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) bool {
	return check(c.chainsDB, c.LocalHeadForChain(chain), chain, blockNum, logIdx, logHash)
}
func (c *finalizedChecker) Check(chain types.ChainID, blockNum uint64, logIdx uint32, logHash common.Hash) bool {
	return check(c.chainsDB, c.LocalHeadForChain(chain), chain, blockNum, logIdx, logHash)
}

// Update creates an Operation that updates the x-head for the chain, given an index to set it to
func (c *unsafeChecker) Update(chain types.ChainID, index entrydb.EntryIdx) heads.OperationFn {
	return func(heads *heads.Heads) error {
		chainHeads := heads.Get(chain)
		chainHeads.CrossUnsafe = index
		heads.Put(chain, chainHeads)
		return nil
	}
}

func (c *safeChecker) Update(chain types.ChainID, index entrydb.EntryIdx) heads.OperationFn {
	return func(heads *heads.Heads) error {
		chainHeads := heads.Get(chain)
		chainHeads.CrossSafe = index
		heads.Put(chain, chainHeads)
		return nil
	}
}

func (c *finalizedChecker) Update(chain types.ChainID, index entrydb.EntryIdx) heads.OperationFn {
	return func(heads *heads.Heads) error {
		chainHeads := heads.Get(chain)
		chainHeads.CrossFinalized = index
		heads.Put(chain, chainHeads)
		return nil
	}
}
