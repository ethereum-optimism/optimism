package sysgo

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum/go-ethereum/core"
)

type L2CLNode interface {
	stack.Lifecycle
	UserRPC() string
}

// L2CLRole describes the responsibility of a consensus-layer node slot.
type L2CLRole string

const (
	L2CLRoleSequencer L2CLRole = "sequencer"
	L2CLRoleVerifier  L2CLRole = "verifier"
)

// L2CLLaunchContext is the product-neutral input required to launch an
// externally implemented L2 consensus client. It contains stable endpoints
// and immutable network configuration, rather than concrete sysgo backends.
type L2CLLaunchContext struct {
	Target ComponentTarget
	Role   L2CLRole

	L1UserRPC   string
	L1BeaconRPC string
	L2UserRPC   string
	L2EngineRPC string
	L2JWTPath   string

	L1Genesis    *core.Genesis
	L2Genesis    *core.Genesis
	RollupConfig *rollup.Config
	FollowSource string

	// UnsafeSource* identifies the complete EL selected as a verifier's
	// unsafe source when the runtime owns that EL.
	UnsafeSourceUserRPC   string
	UnsafeSourceEngineRPC string
	UnsafeSourceJWTPath   string

	Config        L2CLConfig
	DependencySet depset.DependencySet

	// RegisterMetrics is nil in classic and staged runtimes because they do not
	// own a metrics collector. External clients must treat nil as registration
	// being unavailable; their Prometheus endpoint may still be scraped directly.
	RegisterMetrics func(serviceName string, endpoints ...PrometheusMetricsTarget)
}

// L2CLFactory may provide an external implementation for an individual L2 CL
// slot. For a handled slot, the factory must return a non-nil lifecycle/RPC
// handle whose UserRPC address is stable. The factory owns readiness and the
// external process lifecycle, including starting the process and registering
// cleanup with devtest.T. Launch may be deferred until the factory has enough
// slots to start one shared process. The runtime deliberately does not call
// Start, wait for readiness, or register Stop. Supplying a factory may require
// endpoint providers to start before CL selection and resolves L2CLOptions
// before CreateL2CL, even when the factory declines. A decline must return
// (nil, false) without starting resources; the runtime immediately selects its
// fallback. Returning handled=false preserves the configured fallback client
// kind, not factory-free start order or option side effects.
type L2CLFactory interface {
	CreateL2CL(t devtest.T, ctx L2CLLaunchContext) (node L2CLNode, handled bool)
}

type L2CLFactoryFn func(t devtest.T, ctx L2CLLaunchContext) (node L2CLNode, handled bool)

var _ L2CLFactory = L2CLFactoryFn(nil)

func (fn L2CLFactoryFn) CreateL2CL(t devtest.T, ctx L2CLLaunchContext) (L2CLNode, bool) {
	return fn(t, ctx)
}

type L2CLConfig struct {
	// SyncMode to run, if this is a sequencer
	SequencerSyncMode nodeSync.Mode
	// SyncMode to run, if this is a verifier
	VerifierSyncMode nodeSync.Mode

	// SafeDBPath is the path to the safe DB to use. Disabled if empty.
	SafeDBPath string

	IsSequencer bool
	// SequencerStopped starts a sequencer without producing blocks until its
	// admin start method is called.
	SequencerStopped bool

	// EnableReqRespSync enables/disables the req-resp sync server (serving payloads to peers).
	EnableReqRespSync bool

	// NoDiscovery is the flag to enable/disable discovery
	NoDiscovery bool

	FollowSource string

	// OffsetELSafe retracts safe and finalized from the EL-sync tip by floor(OffsetELSafe / L2BlockTime) blocks.
	OffsetELSafe time.Duration

	// SequencerMaxSafeLag caps the gap between unsafe and safe L2 heads;
	// the sequencer stalls block production when the gap is exceeded. 0 disables.
	SequencerMaxSafeLag uint64
}

func L2CLSequencer() L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.IsSequencer = true
	})
}

func L2CLSequencerStopped() L2CLOption {
	return L2CLOptionFn(func(_ devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.SequencerStopped = true
	})
}

func L2CLFollowSource(source string) L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.FollowSource = source
	})
}

// L2CLSequencerMaxSafeLag sets the sequencer max-safe-lag. The sequencer stalls
// block production when the unsafe/safe gap exceeds this value. 0 disables.
func L2CLSequencerMaxSafeLag(maxSafeLag uint64) L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.SequencerMaxSafeLag = maxSafeLag
	})
}

func DefaultL2CLConfig() *L2CLConfig {
	return &L2CLConfig{
		SequencerSyncMode: nodeSync.CLSync,
		VerifierSyncMode:  nodeSync.CLSync,
		SafeDBPath:        "",
		IsSequencer:       false,
		EnableReqRespSync: true,
		NoDiscovery:       false,
		FollowSource:      "",
	}
}

type L2CLOption interface {
	Apply(p devtest.T, target ComponentTarget, cfg *L2CLConfig)
}

type L2CLOptionFn func(p devtest.T, target ComponentTarget, cfg *L2CLConfig)

var _ L2CLOption = L2CLOptionFn(nil)

func (fn L2CLOptionFn) Apply(p devtest.T, target ComponentTarget, cfg *L2CLConfig) {
	fn(p, target, cfg)
}

// L2CLOptionBundle a list of multiple L2CLOption, to all be applied in order.
type L2CLOptionBundle []L2CLOption

var _ L2CLOption = L2CLOptionBundle(nil)

func (l L2CLOptionBundle) Apply(p devtest.T, target ComponentTarget, cfg *L2CLConfig) {
	for _, opt := range l {
		p.Require().NotNil(opt, "cannot Apply nil L2CLOption")
		opt.Apply(p, target, cfg)
	}
}
