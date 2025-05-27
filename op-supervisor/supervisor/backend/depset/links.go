package depset

import "github.com/ethereum-optimism/optimism/op-service/eth"

type LinkChecker interface {
	// CanExecute determines if an executing message is valid w.r.t. chain and timestamp constraints.
	// I.e. if the chain may be executing messages at the given timestamp,
	// from the given chain at the given initiating timestamp.
	// I.e. this does not check a full message, it merely checks some linking constraints.
	CanExecute(execInChain eth.ChainID, execInTimestamp uint64, initChainID eth.ChainID, initTimestamp uint64) bool
}

// LinkerConfig represents the config dependencies that are needed
// to create the default LinkChecker implementation LinkCheckerImpl.
type LinkerConfig interface {
	ActivationConfig
	DependencySet
}

// LinkCheckerImpl implements a LinkChecker using the provided config
type LinkCheckerImpl struct {
	cfg LinkerConfig
}

func LinkerFromConfig(cfg LinkerConfig) *LinkCheckerImpl {
	return &LinkCheckerImpl{cfg: cfg}
}

func (lc *LinkCheckerImpl) CanExecute(execInChain eth.ChainID,
	execInTimestamp uint64, initChainID eth.ChainID, initTimestamp uint64) bool {
	// Check the chains exist in the dependency set.
	if !lc.cfg.HasChain(execInChain) {
		return false
	}
	if !lc.cfg.HasChain(initChainID) {
		return false
	}
	// Note: this function deliberately moves some business logic closer to the config.
	// What makes a valid/invalid link depends on the interop set, which can change over time.
	if !lc.cfg.IsInterop(execInChain, execInTimestamp) {
		return false
	}
	// Note: this does not cover the genesis block of a chain, but that block is empty anyway.
	if lc.cfg.IsInteropActivationBlock(execInChain, execInTimestamp) {
		return false
	}
	if !lc.cfg.IsInterop(initChainID, initTimestamp) {
		return false
	}
	// Note: this does not cover the genesis block of a chain, but that block is empty anyway.
	if lc.cfg.IsInteropActivationBlock(initChainID, initTimestamp) {
		return false
	}
	if initTimestamp > execInTimestamp {
		return false
	}
	window := lc.cfg.MessageExpiryWindow()
	expiresAt := initTimestamp + window
	if expiresAt < initTimestamp { // happens upon underflow
		return false
	}
	if expiresAt < execInTimestamp { // expiry check
		return false
	}
	return true
}

// LinkCheckFn is a function-type that implements LinkChecker, for testing and other special case definitions
type LinkCheckFn func(execInChain eth.ChainID, execInTimestamp uint64, initChainID eth.ChainID, initTimestamp uint64) bool

func (lFn LinkCheckFn) CanExecute(execInChain eth.ChainID, execInTimestamp uint64, initChainID eth.ChainID, initTimestamp uint64) bool {
	return lFn(execInChain, execInTimestamp, initChainID, initTimestamp)
}

var _ LinkChecker = (LinkCheckFn)(nil)
