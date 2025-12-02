package dsl

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// RequireSupervisorChainHeads queries the supervisor actor for its sync status and asserts that
// the chain's LocalUnsafe, CrossUnsafe, LocalSafe, CrossSafe, and Finalized block IDs match the
// provided expected values. It fails the test if any assertion does not hold.
func RequireSupervisorChainHeads(t helpers.Testing, super *SupervisorActor, chain *Chain, unsafe, crossUnsafe, localSafe, safe, finalized eth.BlockID) {
	status, err := super.SyncStatus(t.Ctx())
	require.NoError(t, err)

	cStatus, ok := status.Chains[chain.ChainID]
	require.True(t, ok, "chain id not present in supervisor sync status %s", chain.ChainID)
	passed := true
	passed = passed && assert.Equal(t, unsafe, cStatus.LocalUnsafe.ID(), "unsafe for chain id: %s", chain.ChainID)
	passed = passed && assert.Equal(t, crossUnsafe, cStatus.CrossUnsafe, "cross unsafe for chain id: %s", chain.ChainID)
	passed = passed && assert.Equal(t, localSafe, cStatus.LocalSafe, "local safe for chain id: %s", chain.ChainID)
	passed = passed && assert.Equal(t, safe, cStatus.CrossSafe, "safe for chain id: %s", chain.ChainID)
	passed = passed && assert.Equal(t, finalized, cStatus.Finalized, "finalized for chain id: %s", chain.ChainID)
	require.True(t, passed, "One or more supervisor chain heads assertions failed")
}

// RequireUnsafeTimeOffset asserts that the difference between the sequencer's L2 unsafe block time
// and the chain's genesis L2 time matches the provided expected offset.
func RequireUnsafeTimeOffset(t helpers.Testing, c *Chain, timeOffset uint64) {
	require.GreaterOrEqual(t, c.Sequencer.L2Unsafe().Time, c.RollupCfg.Genesis.L2Time, "L2 unsafe timestamp is before rollup config genesis timestamp, look for misconfiguration")
	require.Equal(t, timeOffset, c.Sequencer.L2Unsafe().Time-c.RollupCfg.Genesis.L2Time)
}

// RequireL1Heads fetches the latest and finalized L1 block headers via the L1 miner client and
// asserts that their block numbers match the expected values, treating missing finalized blocks as zero.
func RequireL1Heads(t helpers.Testing, system *InteropDSL, latest, finalized uint64) {
	latestHeader, err := system.Actors.L1Miner.EthClient().BlockByNumber(t.Ctx(), big.NewInt(int64(rpc.LatestBlockNumber)))
	require.NoError(t, err)
	require.Equal(t, latest, latestHeader.Number().Uint64(), "L1 latest number does not match expectation")

	finalizedHeader, err := system.Actors.L1Miner.EthClient().BlockByNumber(t.Ctx(), big.NewInt(int64(rpc.FinalizedBlockNumber)))

	var finalizedActual uint64
	if err != nil {
		if err.Error() != "finalized block not found" {
			require.NoError(t, err)
		}
		finalizedActual = 0
	} else {
		finalizedActual = finalizedHeader.Number().Uint64()
	}

	require.Equal(t, finalized, finalizedActual, "L1 finalized number does not match expectation")
}

// SuperchainSyncStatusAsserter provides utilities to assert the synchronization status
// of multiple chains within the superchain context, tracking per-chain sync assertions.
type SuperchainSyncStatusAsserter struct {
	t              helpers.Testing
	ChainAsserters map[eth.ChainID]*ChainSyncStatusAsserter
}

// NewSuperchainSyncStatusAsserter constructs an asserter for each chain, initializing with the current sync status.
// If strict is true, any status change outside explicit assertions will cause failures.
func NewSuperchainSyncStatusAsserter(t helpers.Testing, system *InteropDSL, chains []*Chain, strict bool) *SuperchainSyncStatusAsserter {
	chainAsserters := make(map[eth.ChainID]*ChainSyncStatusAsserter, 0)
	for _, chain := range chains {
		chainAsserters[chain.ChainID] = NewChainSyncStatusAsserter(t, system, chain, strict)
	}

	return &SuperchainSyncStatusAsserter{
		t:              t,
		ChainAsserters: chainAsserters,
	}
}

// RequireAllSeqSyncStatuses runs a pre-check for all chains, executes the provided action,
// and then asserts post-action synchronization status updates per chain using given options.
func (a *SuperchainSyncStatusAsserter) RequireAllSeqSyncStatuses(action func(), options ...superchainAssertionOption) {
	cfg := &superchainAssertionConfig{
		chainAssertionOptions: make(map[eth.ChainID][]assertionOption),
	}
	for _, opt := range options {
		opt(cfg)
	}

	for _, chainAsserter := range a.ChainAsserters {
		chainAsserter.requireSeqSyncStatusPreCheck()
	}

	action()

	for _, chainAsserter := range a.ChainAsserters {
		chainCfg := &updateExpectedConfig{}
		for _, opt := range cfg.chainAssertionOptions[chainAsserter.chain.ChainID] {
			opt(chainCfg)
		}
		chainAsserter.requireSeqSyncStatusPostCheck(chainCfg)
	}
}

// RequireAllInitialSeqSyncStatuses asserts the initial synchronization status for all chains
// using the provided assertion options without executing any action.
func (a *SuperchainSyncStatusAsserter) RequireAllInitialSeqSyncStatuses(options ...assertionOption) {
	for _, chainAsserter := range a.ChainAsserters {
		chainAsserter.RequireInitialSeqSyncStatus(options...)
	}
}

// superchainAssertionConfig holds per-chain assertion options for superchain sync checks.
type superchainAssertionConfig struct {
	chainAssertionOptions map[eth.ChainID][]assertionOption
}

type superchainAssertionOption func(*superchainAssertionConfig)

// WithMapChainAssertions applies the given assertion options to all chains in the superchain configuration.
func WithMapChainAssertions(chainAssertions ...assertionOption) superchainAssertionOption {
	return func(cfg *superchainAssertionConfig) {
		for chainID := range cfg.chainAssertionOptions {
			cfg.chainAssertionOptions[chainID] = append(cfg.chainAssertionOptions[chainID], chainAssertions...)
		}
	}
}

// WithChainAssertions applies the given assertion options to the specified chain in the superchain configuration.
func WithChainAssertions(chain *Chain, chainAssertions ...assertionOption) superchainAssertionOption {
	return func(cfg *superchainAssertionConfig) {
		cfg.chainAssertionOptions[chain.ChainID] = append(cfg.chainAssertionOptions[chain.ChainID], chainAssertions...)
	}
}

// ChainSyncStatusAsserter tracks and asserts synchronization status changes for a single chain's sequencer.
type ChainSyncStatusAsserter struct {
	chain      *Chain
	t          helpers.Testing
	system     *InteropDSL
	PrevStatus *eth.SyncStatus
	strict     bool
}

// NewChainSyncStatusAsserter initializes a ChainSyncStatusAsserter capturing the current sync status and strict mode preference.
func NewChainSyncStatusAsserter(t helpers.Testing, system *InteropDSL, chain *Chain, strict bool) *ChainSyncStatusAsserter {
	return &ChainSyncStatusAsserter{
		t:          t,
		system:     system,
		chain:      chain,
		PrevStatus: chain.Sequencer.SyncStatus(),
		strict:     strict,
	}
}

// RequireInitialSeqSyncStatus asserts the current synchronization status using provided assertion options without running an action.
func (a *ChainSyncStatusAsserter) RequireInitialSeqSyncStatus(options ...assertionOption) {
	a.RequireSeqSyncStatus(func() {}, options...)
}

// requireSeqSyncStatusPreCheck verifies that sync status has not changed since last check in strict mode.
func (a *ChainSyncStatusAsserter) requireSeqSyncStatusPreCheck() {
	status := a.chain.Sequencer.SyncStatus()
	if a.strict {
		require.Equal(a.t, *a.PrevStatus, *status, "sync status changed since previous call to RequireSeqSyncStatus(), strict mode requires all sync status changes to be explicitly asserted. To fix, look for a previous action call outside a RequireSeqSyncStatus() call and wrap it in a RequireSeqSyncStatus() call.")
	}
}

// baseSyncStatusAssertions asserts the baseline ancestor-descendant relationships within the provided sync status.
// It verifies that each of CrossUnsafeL2, LocalSafeL2, SafeL2, and FinalizedL2 blocks in nextSyncStatus
// is a descendant of the UnsafeL2 block. Failures are reported via the testing interface, and the method
// returns true if all assertions succeed, false otherwise.
func (a *ChainSyncStatusAsserter) baseSyncStatusAssertions(nextSyncStatus *eth.SyncStatus) bool {
	result := true
	result = result && assert.True(a.t, AssertAncestorDescendantRelationship(a.t, a.chain, nextSyncStatus.CrossUnsafeL2.ID(), nextSyncStatus.UnsafeL2.ID()), "cross unsafe block id is not an ancestor of the next unsafe block id")
	result = result && assert.True(a.t, AssertAncestorDescendantRelationship(a.t, a.chain, nextSyncStatus.LocalSafeL2.ID(), nextSyncStatus.UnsafeL2.ID()), "local safe block id is not an ancestor of the next unsafe block id")
	result = result && assert.True(a.t, AssertAncestorDescendantRelationship(a.t, a.chain, nextSyncStatus.SafeL2.ID(), nextSyncStatus.UnsafeL2.ID()), "safe block id is not an ancestor of the next unsafe block id")
	result = result && assert.True(a.t, AssertAncestorDescendantRelationship(a.t, a.chain, nextSyncStatus.FinalizedL2.ID(), nextSyncStatus.UnsafeL2.ID()), "finalized block id is not an ancestor of the next unsafe block id")
	return result
}

// requireSeqSyncStatusPostCheck applies configured SyncStatusAssertions to the post-action status,
// fails the test if any assertion fails, and updates PrevStatus to the new status.
func (a *ChainSyncStatusAsserter) requireSeqSyncStatusPostCheck(cfg *updateExpectedConfig) {
	postStatus := a.chain.Sequencer.SyncStatus()

	result := a.baseSyncStatusAssertions(postStatus)

	for _, assertion := range []SyncStatusAssertion{cfg.unsafeBlockAssertion, cfg.crossUnsafeAssertion, cfg.localSafeAssertion, cfg.safeAssertion, cfg.finalizedAssertion} {
		if assertion != nil {
			result = result && assertion(a.t, a.PrevStatus, postStatus)
		}
	}
	require.True(a.t, result, "Failed to update and assert seq status")

	a.PrevStatus = postStatus
}

// RequireSeqSyncStatus runs a pre-check, executes the provided action, then applies the assertion options
// to verify expected synchronization status changes.
func (a *ChainSyncStatusAsserter) RequireSeqSyncStatus(action func(), options ...assertionOption) {
	a.requireSeqSyncStatusPreCheck()
	action()

	cfg := &updateExpectedConfig{}
	for _, opt := range options {
		opt(cfg)
	}
	a.requireSeqSyncStatusPostCheck(cfg)
}

// LogSyncStatus logs the current expected sync status values for the chain for debugging purposes.
func (a *ChainSyncStatusAsserter) LogSyncStatus() {
	a.t.Logf("chain %v expected status:\n  unsafe=%d\n  crossUnsafe=%d\n  localSafe=%d\n  safe=%d\n  finalized=%d",
		a.chain.ChainID,
		a.PrevStatus.UnsafeL2.Number,
		a.PrevStatus.CrossUnsafeL2.Number,
		a.PrevStatus.LocalSafeL2.Number,
		a.PrevStatus.SafeL2.Number,
		a.PrevStatus.FinalizedL2.Number,
	)
}

// RequireSupChainHeadsBySyncStatus asserts that the supervisor actor's chain heads match the asserter's PrevStatus values.
func (a *ChainSyncStatusAsserter) RequireSupChainHeadsBySyncStatus() {
	RequireSupervisorChainHeads(a.t, a.system.Actors.Supervisor, a.chain, a.PrevStatus.UnsafeL2.ID(), a.PrevStatus.CrossUnsafeL2.ID(), a.PrevStatus.LocalSafeL2.ID(), a.PrevStatus.SafeL2.ID(), a.PrevStatus.FinalizedL2.ID())
}

// assertionOption configures expected sync status assertions for a single chain.
type assertionOption func(*updateExpectedConfig)

// SyncStatusAssertion defines the signature for a function that asserts conditions between previous and next sync statuses.
type SyncStatusAssertion func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool

// updateExpectedConfig holds the SyncStatusAssertion functions for each sync status field.
type updateExpectedConfig struct {
	unsafeBlockAssertion SyncStatusAssertion
	crossUnsafeAssertion SyncStatusAssertion
	localSafeAssertion   SyncStatusAssertion
	safeAssertion        SyncStatusAssertion
	finalizedAssertion   SyncStatusAssertion
}

// WitUnsafeAdvancesTo returns an assertionOption that checks the next UnsafeL2.Number equals the expected value.
func WithUnsafeAdvancesTo(expectedUnsafeNumber uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.unsafeBlockAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			result := assert.Greater(t, nextSyncStatus.UnsafeL2.Number, prevSyncStatus.UnsafeL2.Number, "unsafe did not advance")
			return result && assert.Equal(t, expectedUnsafeNumber, nextSyncStatus.UnsafeL2.Number, "unsafe number did not match expectations")
		}
	}
}

// WithUnsafeAdvancesBy returns an assertionOption that checks UnsafeL2.Number advances by the given amount.
func WithUnsafeAdvancesBy(advancesBy uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.unsafeBlockAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			result := assert.Greater(t, nextSyncStatus.UnsafeL2.Number, prevSyncStatus.UnsafeL2.Number, "unsafe did not advance")
			return result && assert.Equal(t, prevSyncStatus.UnsafeL2.Number+advancesBy, nextSyncStatus.UnsafeL2.Number, "unsafe number did not advance as expected")
		}
	}
}

func WithUnsafeEquals(expectedUnsafeNumber uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.unsafeBlockAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			return assert.Equal(t, expectedUnsafeNumber, nextSyncStatus.UnsafeL2.Number, "unsafe number did not match expectations")
		}
	}
}

func WithCrossUnsafeAdvancesTo(expectedCrossUnsafeNumber uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.crossUnsafeAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			result := assert.Greater(t, nextSyncStatus.CrossUnsafeL2.Number, prevSyncStatus.CrossUnsafeL2.Number, "cross unsafe did not advance")
			return result && assert.Equal(t, expectedCrossUnsafeNumber, nextSyncStatus.CrossUnsafeL2.Number, "cross unsafe number did not match expectations")
		}
	}
}

func WithCrossUnsafeAdvancesBy(advancesBy uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.crossUnsafeAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			result := assert.Greater(t, nextSyncStatus.CrossUnsafeL2.Number, prevSyncStatus.CrossUnsafeL2.Number, "cross unsafe did not advance")
			return result && assert.Equal(t, prevSyncStatus.CrossUnsafeL2.Number+advancesBy, nextSyncStatus.CrossUnsafeL2.Number, "cross unsafe number did not advance as expected")
		}
	}
}

func WithCrossUnsafeEquals(expectedCrossUnsafeNumber uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.crossUnsafeAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			return assert.Equal(t, expectedCrossUnsafeNumber, nextSyncStatus.CrossUnsafeL2.Number, "cross unsafe number did not match expectations")
		}
	}
}

// WithLocalSafeAdvancesToUnsafe returns an assertionOption asserting that LocalSafe advances to the Unsafe block ID.
func WithLocalSafeAdvancesTo(expectedLocalSafeNumber uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.localSafeAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			result := assert.Greater(t, nextSyncStatus.LocalSafeL2.Number, prevSyncStatus.LocalSafeL2.Number, "local safe did not advance")
			return result && assert.Equal(t, expectedLocalSafeNumber, nextSyncStatus.LocalSafeL2.Number, "local safe number did not match expectations")
		}
	}
}

// WithLocalSafeAdvancesBy returns an assertionOption asserting that LocalSafe advances by the given amount.
func WithLocalSafeAdvancesBy(advancesBy uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.localSafeAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			result := assert.Greater(t, nextSyncStatus.LocalSafeL2.Number, prevSyncStatus.LocalSafeL2.Number, "local safe did not advance")
			return result && assert.Equal(t, prevSyncStatus.LocalSafeL2.Number+advancesBy, nextSyncStatus.LocalSafeL2.Number, "local safe number did not advance as expected")
		}
	}
}

func WithLocalSafeEquals(expectedLocalSafeNumber uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.localSafeAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			return assert.Equal(t, expectedLocalSafeNumber, nextSyncStatus.LocalSafeL2.Number, "local safe number did not match expectations")
		}
	}
}

// WithSafeAdvancesToUnsafe returns an assertionOption asserting that Safe advances to the Unsafe block ID.
func WithSafeAdvancesTo(expectedSafeNumber uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.safeAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			result := assert.Greater(t, nextSyncStatus.SafeL2.Number, prevSyncStatus.SafeL2.Number, "safe did not advance")
			return result && assert.Equal(t, expectedSafeNumber, nextSyncStatus.SafeL2.Number, "safe number did not match expectations")
		}
	}
}

func WithSafeAdvancesBy(advancesBy uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.safeAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			result := assert.Greater(t, nextSyncStatus.SafeL2.Number, prevSyncStatus.SafeL2.Number, "safe did not advance")
			return result && assert.Equal(t, prevSyncStatus.SafeL2.Number+advancesBy, nextSyncStatus.SafeL2.Number, "safe number did not advance as expected")
		}
	}
}

func WithSafeEquals(expectedSafeNumber uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.safeAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			return assert.Equal(t, expectedSafeNumber, nextSyncStatus.SafeL2.Number, "safe number did not match expectations")
		}
	}
}

// WithFinalizedAdvancesToUnsafe returns an assertionOption asserting that Finalized advances to the Unsafe block ID.
func WithFinalizedAdvancesTo(expectedFinalizedNumber uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.finalizedAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			result := assert.Greater(t, nextSyncStatus.FinalizedL2.Number, prevSyncStatus.FinalizedL2.Number, "finalized did not advance")
			return result && assert.Equal(t, expectedFinalizedNumber, nextSyncStatus.FinalizedL2.Number, "finalized number did not match expectations")
		}
	}
}

func WithFinalizedAdvancesBy(advancesBy uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.finalizedAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			result := assert.Greater(t, nextSyncStatus.FinalizedL2.Number, prevSyncStatus.FinalizedL2.Number, "finalized did not advance")
			return result && assert.Equal(t, prevSyncStatus.FinalizedL2.Number+advancesBy, nextSyncStatus.FinalizedL2.Number, "finalized number did not advance as expected")
		}
	}
}

func WithFinalizedEquals(expectedFinalizedNumber uint64) func(cfg *updateExpectedConfig) {
	return func(cfg *updateExpectedConfig) {
		cfg.finalizedAssertion = func(t assert.TestingT, prevSyncStatus, nextSyncStatus *eth.SyncStatus) bool {
			return assert.Equal(t, expectedFinalizedNumber, nextSyncStatus.FinalizedL2.Number, "finalized number did not match expectations")
		}
	}
}
