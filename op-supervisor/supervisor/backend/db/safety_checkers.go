package db

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/heads"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	backendTypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/types"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const (
	Unsafe    = "unsafe"
	Safe      = "safe"
	Finalized = "finalized"
)

// SafetyChecker is an interface for checking the safety of a log entry
// it maintains a consistent view between local and cross chain for a given safety level
type SafetyChecker interface {
	LocalHead(chainID types.ChainID) heads.HeadPointer
	CrossHead(chainID types.ChainID) heads.HeadPointer
	CheckLocal(chain types.ChainID, blockNum uint64, logIdx uint32, logHash backendTypes.TruncatedHash) error
	CheckCross(chain types.ChainID, blockNum uint64, logIdx uint32, logHash backendTypes.TruncatedHash) error
	UpdateLocal(chain types.ChainID, pointer heads.HeadPointer) error
	UpdateCross(chain types.ChainID, pointer heads.HeadPointer) error
	String() string
	LocalSafetyLevel() types.SafetyLevel
	CrossSafetyLevel() types.SafetyLevel
}

// NewSafetyChecker creates a new SafetyChecker of the given type
func NewSafetyChecker(t types.SafetyLevel, chainsDB *ChainsDB) SafetyChecker {
	return NewChecker(t, chainsDB)
}

// check checks if the log entry is safe, provided a local head for the chain
// it is used by the individual SafetyCheckers to determine if a log entry is safe
func check(
	chainsDB *ChainsDB,
	head heads.HeadPointer,
	chain types.ChainID,
	blockNum uint64,
	logIdx uint32,
	logHash backendTypes.TruncatedHash) error {

	// for the Check to be valid, the log must:
	// 1. have the expected logHash at the indicated blockNum and logIdx
	_, err := chainsDB.logDBs[chain].Contains(blockNum, logIdx, logHash)
	if err != nil {
		return err
	}
	// 2. be within the range of the given head
	if !head.WithinRange(blockNum, logIdx) {
		return logs.ErrFuture
	}
	return nil
}

// checker is a composition of accessor and update functions for a given safety level.
// they implement the SafetyChecker interface.
// checkers can be made with NewChecker.
type checker struct {
	chains      *ChainsDB
	localSafety types.SafetyLevel
	crossSafety types.SafetyLevel
	updateCross func(chain types.ChainID, pointer heads.HeadPointer) error
	updateLocal func(chain types.ChainID, pointer heads.HeadPointer) error
	localHead   func(chain types.ChainID) heads.HeadPointer
	crossHead   func(chain types.ChainID) heads.HeadPointer
	checkCross  func(chain types.ChainID, blockNum uint64, logIdx uint32, logHash backendTypes.TruncatedHash) error
	checkLocal  func(chain types.ChainID, blockNum uint64, logIdx uint32, logHash backendTypes.TruncatedHash) error
}

func (c *checker) String() string {
	return fmt.Sprintf("%s+%s", c.localSafety.String(), c.crossSafety.String())
}

func (c *checker) LocalSafetyLevel() types.SafetyLevel {
	return c.localSafety
}

func (c *checker) CrossSafetyLevel() types.SafetyLevel {
	return c.crossSafety
}

func (c *checker) UpdateCross(chain types.ChainID, pointer heads.HeadPointer) error {
	return c.updateCross(chain, pointer)
}
func (c *checker) UpdateLocal(chain types.ChainID, pointer heads.HeadPointer) error {
	return c.updateLocal(chain, pointer)
}
func (c *checker) LocalHead(chain types.ChainID) heads.HeadPointer {
	return c.localHead(chain)
}
func (c *checker) CrossHead(chain types.ChainID) heads.HeadPointer {
	return c.crossHead(chain)
}
func (c *checker) CheckCross(chain types.ChainID, blockNum uint64, logIdx uint32, logHash backendTypes.TruncatedHash) error {
	return c.checkCross(chain, blockNum, logIdx, logHash)
}
func (c *checker) CheckLocal(chain types.ChainID, blockNum uint64, logIdx uint32, logHash backendTypes.TruncatedHash) error {
	return c.checkLocal(chain, blockNum, logIdx, logHash)
}

func NewChecker(t types.SafetyLevel, c *ChainsDB) *checker {
	// checkWith creates a function which takes a chain-getter and returns a function that returns the head for the chain
	checkWith := func(getHead func(chain types.ChainID) heads.HeadPointer) func(chain types.ChainID, blockNum uint64, logIdx uint32, logHash backendTypes.TruncatedHash) error {
		return func(chain types.ChainID, blockNum uint64, logIdx uint32, logHash backendTypes.TruncatedHash) error {
			return check(c, getHead(chain), chain, blockNum, logIdx, logHash)
		}
	}
	switch t {
	case Unsafe:
		return &checker{
			chains:      c,
			localSafety: types.Unsafe,
			crossSafety: types.CrossUnsafe,
			updateCross: c.heads.UpdateCrossUnsafe,
			updateLocal: c.heads.UpdateLocalUnsafe,
			crossHead:   c.heads.CrossUnsafe,
			localHead:   c.heads.LocalUnsafe,
			checkCross:  checkWith(c.heads.CrossUnsafe),
			checkLocal:  checkWith(c.heads.LocalUnsafe),
		}
	case Safe:
		return &checker{
			chains:      c,
			localSafety: types.Safe,
			crossSafety: types.CrossSafe,
			updateCross: c.heads.UpdateCrossSafe,
			updateLocal: c.heads.UpdateLocalSafe,
			crossHead:   c.heads.CrossSafe,
			localHead:   c.heads.LocalSafe,
			checkCross:  checkWith(c.heads.CrossSafe),
			checkLocal:  checkWith(c.heads.LocalSafe),
		}
	case Finalized:
		return &checker{
			chains:      c,
			localSafety: types.Finalized,
			crossSafety: types.CrossFinalized,
			updateCross: c.heads.UpdateCrossFinalized,
			updateLocal: c.heads.UpdateLocalFinalized,
			crossHead:   c.heads.CrossFinalized,
			localHead:   c.heads.LocalFinalized,
			checkCross:  checkWith(c.heads.CrossFinalized),
			checkLocal:  checkWith(c.heads.LocalFinalized),
		}
	}
	return &checker{}
}
