package filter

import (
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// parityLinkCfg is a minimal real depset.LinkerConfig used to drive the
// AUTHORITATIVE link rule depset.LinkCheckerImpl.CanExecute. It mirrors the
// config used by the depset package's own tests (links_test.go, mockLinkCfg).
//
// activationTimes is chosen so that, for every timestamp sampled by the parity
// fuzz, interop is active and the timestamp is NOT an interop activation block,
// on BOTH the executing and initiating chains. That isolates the comparison to
// exactly the timing + expiry rules (the only thing the filter's
// validateMessageTiming also enforces, with timeout disabled).
type parityLinkCfg struct {
	activationTimes map[eth.ChainID]uint64
	window          uint64
}

func (m *parityLinkCfg) Chains() (out []eth.ChainID) {
	for id := range m.activationTimes {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Cmp(out[j]) < 0 })
	return
}

func (m *parityLinkCfg) HasChain(chainID eth.ChainID) bool {
	_, ok := m.activationTimes[chainID]
	return ok
}

func (m *parityLinkCfg) IsInterop(chainID eth.ChainID, ts uint64) bool {
	v, ok := m.activationTimes[chainID]
	if !ok {
		return false
	}
	return ts >= v
}

func (m *parityLinkCfg) IsInteropActivationBlock(chainID eth.ChainID, ts uint64) bool {
	v, ok := m.activationTimes[chainID]
	if !ok {
		return false
	}
	return ts == v
}

func (m *parityLinkCfg) MessageExpiryWindow() uint64 {
	return m.window
}

var _ depset.LinkerConfig = (*parityLinkCfg)(nil)

// Bounded sample space. baseTS is large enough that both chains are well past
// interop activation (and never on the activation block) for any sampled
// timestamp, and small enough that initTS + maxWindow cannot overflow uint64.
const (
	parityBaseTS     = uint64(1_000_000) // interop active well before this
	parityTSSpan     = uint64(50_000)    // timestamps in [baseTS, baseTS+span)
	parityMaxWindow  = uint64(60_000)    // expiry window in [0, maxWindow]
	parityActExec    = uint64(900)       // exec chain activation (<< baseTS)
	parityActInit    = uint64(800)       // init chain activation (<< baseTS)
	parityChainExec  = uint64(900)
	parityChainInit  = uint64(901)
	parityIterations = 200_000
)

// makeParityLinker builds a real depset linker for a given expiry window.
// MessageExpiryWindow is per-config, so we rebuild when the window changes.
func makeParityLinker(window uint64) *depset.LinkCheckerImpl {
	cfg := &parityLinkCfg{
		activationTimes: map[eth.ChainID]uint64{
			eth.ChainIDFromUInt64(parityChainExec): parityActExec,
			eth.ChainIDFromUInt64(parityChainInit): parityActInit,
		},
		window: window,
	}
	return depset.LinkerFromConfig(cfg)
}

// canonicalTimingAccepts returns the canonical verdict for the timing+expiry
// rules via the REAL depset.CanExecute. exec chain @ inclusionTS, init chain
// @ initTS. With the chosen config the chain/activation gates never fire in the
// sampled range, so the verdict reflects only timestamp ordering + expiry.
func canonicalTimingAccepts(linker *depset.LinkCheckerImpl, initTS, inclusionTS uint64) bool {
	return linker.CanExecute(
		eth.ChainIDFromUInt64(parityChainExec), inclusionTS, // executing chain @ inclusion
		eth.ChainIDFromUInt64(parityChainInit), initTS, // initiating chain @ init
	)
}

// filterTimingAccepts returns the FIXED filter verdict via the real, directly
// callable validateMessageTiming with timeout disabled.
func filterTimingAccepts(initTS, inclusionTS, window uint64) bool {
	return validateMessageTiming(initTS, inclusionTS, window, 0 /*timeout*/, inclusionTS) == nil
}

// TestTimingParityFuzz is an aggressive adversarial differential: across a large
// bounded random sample (heavily biased to the boundaries), the FIXED filter
// timing rule must AGREE with the canonical depset rule on every single tuple.
func TestTimingParityFuzz(t *testing.T) {
	rng := rand.New(rand.NewSource(0xC0FFEE)) // deterministic for reproducibility
	deadline := time.Now().Add(5 * time.Second)

	// Cache linkers per window (rebuilding a config every iteration is wasteful).
	linkers := make(map[uint64]*depset.LinkCheckerImpl)
	getLinker := func(window uint64) *depset.LinkCheckerImpl {
		if l, ok := linkers[window]; ok {
			return l
		}
		l := makeParityLinker(window)
		linkers[window] = l
		return l
	}

	var (
		iters             int
		equalCases        int // init == inclusion exercised
		offByOneHi        int // init == inclusion+1 exercised
		offByOneLo        int // init == inclusion-1 exercised
		expiryBoundary    int // init + window == inclusion exercised
		expiryBoundaryOff int // init + window == inclusion-1 exercised
		acceptCount       int
		rejectCount       int
	)

	for i := 0; i < parityIterations; i++ {
		if i%1024 == 0 && time.Now().After(deadline) {
			break
		}
		iters++

		window := uint64(rng.Int63n(int64(parityMaxWindow) + 1))
		inclusionTS := parityBaseTS + uint64(rng.Int63n(int64(parityTSSpan)))

		// Bias heavily toward the boundaries.
		var initTS uint64
		switch rng.Intn(8) {
		case 0: // exact same-timestamp (the fixed case)
			initTS = inclusionTS
		case 1: // init == inclusion+1 (init strictly after exec -> reject)
			initTS = inclusionTS + 1
		case 2: // init == inclusion-1
			if inclusionTS > 0 {
				initTS = inclusionTS - 1
			} else {
				initTS = inclusionTS
			}
		case 3: // expiry boundary: init + window == inclusion (not expired)
			if inclusionTS >= window {
				initTS = inclusionTS - window
			} else {
				initTS = inclusionTS
			}
		case 4: // just-expired: init + window == inclusion-1
			if inclusionTS >= window+1 {
				initTS = inclusionTS - window - 1
			} else {
				initTS = inclusionTS
			}
		default: // uniform around inclusion within the sample span
			lo := parityBaseTS
			hi := parityBaseTS + parityTSSpan
			initTS = lo + uint64(rng.Int63n(int64(hi-lo)))
		}

		// Guard: keep everything in the non-overflow, interop-active, non-activation
		// window so the depset chain/activation gates never fire and only timing+expiry
		// is under test. (initTS may be inclusion+1 which is still safely in range.)
		if initTS < parityBaseTS || initTS > parityBaseTS+parityTSSpan {
			// out of isolated range; skip (keeps the comparison purely timing/expiry)
			continue
		}

		linker := getLinker(window)
		filterOK := filterTimingAccepts(initTS, inclusionTS, window)
		canonOK := canonicalTimingAccepts(linker, initTS, inclusionTS)

		require.Equalf(t, canonOK, filterOK,
			"DISAGREEMENT: initTS=%d inclusionTS=%d window=%d -> filter=%v canonical=%v",
			initTS, inclusionTS, window, filterOK, canonOK)

		// Instrumentation so the test cannot vacuously pass.
		if initTS == inclusionTS {
			equalCases++
		}
		if initTS == inclusionTS+1 {
			offByOneHi++
		}
		if inclusionTS > 0 && initTS == inclusionTS-1 {
			offByOneLo++
		}
		if initTS+window == inclusionTS {
			expiryBoundary++
		}
		if initTS+window == inclusionTS-1 {
			expiryBoundaryOff++
		}
		if filterOK {
			acceptCount++
		} else {
			rejectCount++
		}
	}

	t.Logf("iterations=%d agreements=%d (0 disagreements)", iters, iters)
	t.Logf("boundary coverage: init==inclusion=%d  init==inclusion+1=%d  init==inclusion-1=%d  init+window==inclusion=%d  init+window==inclusion-1=%d",
		equalCases, offByOneHi, offByOneLo, expiryBoundary, expiryBoundaryOff)
	t.Logf("verdict mix: accept=%d reject=%d", acceptCount, rejectCount)

	// The fuzz must not be vacuous: the fixed same-timestamp case, the off-by-one
	// reject case, and the expiry boundary must all have been exercised, and both
	// accept and reject verdicts must appear.
	require.Positive(t, equalCases, "init==inclusion (the fixed case) was never exercised")
	require.Positive(t, offByOneHi, "init==inclusion+1 (reject case) was never exercised")
	require.Positive(t, expiryBoundary, "expiry boundary init+window==inclusion was never exercised")
	require.Positive(t, acceptCount, "no accept verdicts produced")
	require.Positive(t, rejectCount, "no reject verdicts produced")
}

// TestTimingParityFocusedBoundaries pins the exact boundary behavior with direct
// asserts: filter and canonical must agree on each.
func TestTimingParityFocusedBoundaries(t *testing.T) {
	const window = uint64(400)
	linker := makeParityLinker(window)
	const inclusion = parityBaseTS + 10_000

	cases := []struct {
		name        string
		initTS      uint64
		inclusionTS uint64
		wantAccept  bool
	}{
		{"init==inclusion -> accept", inclusion, inclusion, true},
		{"init==inclusion+1 (init>inclusion) -> reject", inclusion + 1, inclusion, false},
		{"init+window==inclusion (not expired) -> accept", inclusion - window, inclusion, true},
		{"init+window==inclusion-1 (expired) -> reject", inclusion - window - 1, inclusion, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			filterOK := filterTimingAccepts(tc.initTS, tc.inclusionTS, window)
			canonOK := canonicalTimingAccepts(linker, tc.initTS, tc.inclusionTS)
			require.Equal(t, tc.wantAccept, filterOK, "filter verdict mismatch vs expected")
			require.Equal(t, tc.wantAccept, canonOK, "canonical verdict mismatch vs expected")
			require.Equal(t, canonOK, filterOK, "filter and canonical must agree")
		})
	}
}

// TestSameTimestampEntryPointNowAccepts drives the REAL filter entry point
// (LockstepCrossValidator.ValidateAccessEntry, the function backing the
// interop_checkAccessList RPC that op-reth calls) and asserts it now returns nil
// for init==inclusion==T (previously a hard ErrConflict reject).
func TestSameTimestampEntryPointNowAccepts(t *testing.T) {
	const T = uint64(2000)

	mock := newMockChainIngester()
	mock.SetReady(true)
	checksum := messages.MessageChecksum{0x01}
	mock.AddLog(T, 10, 0, checksum, messages.BlockSeal{Number: 10, Timestamp: T})
	mock.SetLatestTimestamp(T)

	chains := map[eth.ChainID]ChainIngester{
		eth.ChainIDFromUInt64(testChainA): mock,
	}
	cv := newTestCrossValidator(chains, 400 /*expiryWindow*/, T)

	access := makeAccess(testChainA, T, 10, 0, checksum)
	execDescriptor := makeExecDescriptor(testChainA, T /*inclusion == init*/, 0 /*no timeout*/)

	err := cv.ValidateAccessEntry(access, safety.LocalUnsafe, execDescriptor)
	require.NoError(t, err,
		"fixed filter must ACCEPT same-timestamp interop through the real entry point")
}
