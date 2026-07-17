package sysgo

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core"
)

type L2CLNode interface {
	stack.Lifecycle
	UserRPC() string
	InteropRPC() (endpoint string, jwtSecret eth.Bytes32)
}

// L2CLRole describes the responsibility of a consensus-layer node slot.
type L2CLRole string

const (
	L2CLRoleSequencer L2CLRole = "sequencer"
	L2CLRoleVerifier  L2CLRole = "verifier"
)

// L2CLLaunchContext is the product-neutral input required to launch an
// externally implemented L2 consensus client. It exposes endpoints and
// immutable network configuration instead of concrete sysgo backends.
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
	Config       L2CLConfig

	// RegisterMetrics may be nil when the preset does not collect component
	// metrics. External clients may call it with ordinary Prometheus targets.
	RegisterMetrics func(serviceName string, endpoints ...PrometheusMetricsTarget)
}

// L2CLFactory may provide an external implementation for an individual L2 CL
// slot. Returning handled=false preserves the preset's ordinary op-node/Kona
// selection for that slot, enabling mixed-client topologies without a global
// registry or process environment selector.
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

	IsSequencer  bool
	IndexingMode bool

	// EnableReqRespSync is the flag to enable/disable req-resp sync.
	EnableReqRespSync bool

	// UseReqRespSync controls whether to use the req-resp sync protocol. EnableReqRespSync == false && UseReqRespSync == true is not allowed, and node will fail to start.
	UseReqRespSync bool

	// NoDiscovery is the flag to enable/disable discovery
	NoDiscovery bool

	FollowSource string

	// OpConL1FinalizedGuard overrides op-con-node's --l1-finalized-guard
	// ("required" or "disabled"). Empty falls back to the launcher default
	// (the DEVSTACK_OPCON_L1_FINALIZED_GUARD env, itself defaulting to "disabled").
	// Set "required" only against a finalizing L1, so op-con-node tracks L1 finality
	// and advances its L2 finalized head. Ignored by non-op-con CL kinds.
	OpConL1FinalizedGuard string

	// OpConSequencer routes this node's SEQUENCER slot to op-con-node when the
	// devstack CL kind is op-con-node. By default sequencer slots stay on
	// op-node (op-con-node was verifier-only before it grew a sequencing mode);
	// tests that exercise op-con-node's sequencer opt in per-test with
	// L2CLOpConSequencer(). Ignored for verifier slots and non-op-con CL kinds.
	OpConSequencer bool

	// OpConPayloadRingBlocks overrides the op-con-node sequencer's signed
	// replay-ring depth (--sequencer-payload-ring-blocks) when > 0. Bootstrap
	// tests shrink it so a late joiner falls below the signed horizon on a
	// short devnet chain. Ignored for non-op-con CL kinds and verifier slots.
	OpConPayloadRingBlocks int

	// OpConUnsafePayloadWS makes an op-con-node verifier follow a sequencing
	// op-con-node's signed-payload websocket multicast at this ws:// URL
	// (--follow, the push analog of --follow-el): subscribe,
	// verify each block's unsafe-block signature against the expected signer,
	// and ingest accepted blocks. Ignored by non-op-con CL kinds.
	OpConUnsafePayloadWS string

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

func L2CLIndexing() L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.IndexingMode = true
	})
}

func L2CLFollowSource(source string) L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.FollowSource = source
	})
}

// L2CLOpConSequencer opts the sequencer slot into op-con-node when the devstack
// CL kind is op-con-node (DEVSTACK_L2CL_KIND=op-con-node). With any other CL
// kind this is a no-op, so tests using it remain valid ordinary suite additions.
func L2CLOpConSequencer() L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.OpConSequencer = true
	})
}

// L2CLOpConUnsafePayloadWS points an op-con-node verifier at a sequencing
// op-con-node's signed-payload websocket multicast (--follow). No-op
// for other CL kinds.
func L2CLOpConUnsafePayloadWS(wsURL string) L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.OpConUnsafePayloadWS = wsURL
	})
}

// L2CLOpConPayloadRingBlocks shrinks/overrides the op-con-node sequencer's
// signed replay-ring depth (--sequencer-payload-ring-blocks). No-op for other
// CL kinds.
func L2CLOpConPayloadRingBlocks(blocks int) L2CLOption {
	return L2CLOptionFn(func(p devtest.T, _ ComponentTarget, cfg *L2CLConfig) {
		cfg.OpConPayloadRingBlocks = blocks
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
		IndexingMode:      false,
		EnableReqRespSync: true,
		UseReqRespSync:    false,
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
