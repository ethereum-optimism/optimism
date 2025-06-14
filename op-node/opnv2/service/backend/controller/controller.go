package controller

import (
	"context"
	"iter"
	"maps"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/derive2"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/l1access"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/l1rewind"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/l2rewind"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/payloads"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rel"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// Controller keeps track of what is being offered by other modules,
// and tracks a set of requests of them.
// Whenever a new offer is made, the scheduler adapts its state, and schedules inspection.
// Whenever inspection finds something to do, new requests are made.
type Controller struct {
	rootCtx context.Context
	log     log.Logger

	cfgSet depset.RollupConfigSetV2
	depSet depset.DependencySet

	relStates      map[rel.ID]*RELState
	rwelStates     map[rwel.ID]*RWELState
	pipelineStates map[derive2.ID]*PipelineState
	chainDBStates  map[eth.ChainID]*ChainDBState
	payloadsStates map[eth.ChainID]*PayloadsState
	runCfgs        map[eth.ChainID]*RunCfgState

	l1AccessState *L1AccessState

	clock clock.Clock

	mu sync.RWMutex

	emitter event.Emitter
}

var _ State = (*Controller)(nil)

var _ event.AttachEmitter = (*Controller)(nil)
var _ event.Deriver = (*Controller)(nil)

func NewController(
	rootCtx context.Context,
	logger log.Logger,
	cfgSet depset.RollupConfigSetV2,
	depSet depset.DependencySet) *Controller {
	out := &Controller{
		rootCtx:        rootCtx,
		log:            logger,
		cfgSet:         cfgSet,
		depSet:         depSet,
		relStates:      make(map[rel.ID]*RELState),
		rwelStates:     make(map[rwel.ID]*RWELState),
		pipelineStates: make(map[derive2.ID]*PipelineState),
		chainDBStates:  make(map[eth.ChainID]*ChainDBState),
		payloadsStates: make(map[eth.ChainID]*PayloadsState),
		runCfgs:        make(map[eth.ChainID]*RunCfgState),
		clock:          clock.SystemClock,
	}
	return out
}

func (c *Controller) AttachEmitter(em event.Emitter) {
	c.emitter = em

	for _, chainID := range c.depSet.Chains() {
		c.chainDBStates[chainID] = NewChainDBState(c.rootCtx, c.emitter, chainID)
		c.payloadsStates[chainID] = NewPayloadsState(c.rootCtx, c.emitter, chainID)
		c.runCfgs[chainID] = NewRunCfgState(c.rootCtx, c.emitter, chainID)
	}
	c.l1AccessState = NewL1AccessState(c.rootCtx, c.emitter)
}

func (c *Controller) AddRWEL(id rwel.ID, chainID eth.ChainID) {
	c.rwelStates[id] = NewRWELState(c.rootCtx, c.emitter, chainID, id)
}

func (c *Controller) AddPipeline(id derive2.ID, chainID eth.ChainID) {
	c.pipelineStates[id] = NewPipelineState(c.rootCtx, c.emitter, chainID, id)
}

func (c *Controller) OnEvent(ctx context.Context, ev event.Event) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if id := rel.IDFromContext(ctx); id != rel.UnknownID {
		if state, ok := c.relStates[id]; ok {
			state.onUpdate(ev)
		}
	}

	if id := rwel.IDFromContext(ctx); id != rwel.UnknownID {
		if state, ok := c.rwelStates[id]; ok {
			state.onUpdate(ev, c.clock.Now())
		}
	}

	if id := derive2.IDFromContext(ctx); id != derive2.UnknownID {
		if state, ok := c.pipelineStates[id]; ok {
			state.onUpdate(ev, c.clock.Now())
		}
	}

	switch x := ev.(type) {
	// Incoming payloads
	case payloads.PayloadsUpdateEvent:
		state := c.payloadsStates[x.ChainID]
		state.min = x.Min
		state.max = x.Max
		state.count = x.Count
	// L1 updates
	case l1access.LatestL1UpdateEvent:
		if x.LatestL1 != c.l1AccessState.latestL1 &&
			x.LatestL1.ParentID() != c.l1AccessState.latestL1.ID() {
			// TODO reset polling state
			_ = x
		}
		c.l1AccessState.latestL1 = x.LatestL1
	case l1access.ConfirmedL1UpdateEvent:
		c.l1AccessState.confirmedL1 = x.ConfirmedL1
	case l1access.FinalizedL1UpdateEvent:
		c.l1AccessState.finalizedL1 = x.FinalizedL1
	// L2 rewinds
	case l2rewind.L2RewindCheckCompletedEvent:
		c.chainDBStates[x.ChainID].localUnsafe = x.LocalUnsafe
		// TODO handle cross-unsafe
	// L1 rewinds
	case l1rewind.L1RewindCheckCompletedEvent:
		c.chainDBStates[x.ChainID].localSafe = x.LocalSafe
		c.chainDBStates[x.ChainID].crossSafe = x.CrossSafe
	// block replacement
	case rwel.BuildReplacementBlockEvent:
		id := rwel.IDFromContext(ctx)
		s, ok := c.rwelStates[id]
		if ok {
			dbState, ok := c.chainDBStates[s.chainID]
			if ok && dbState.invalidated.Derived == x.Invalidated {
				dbState.replacementAttributes = x.Attributes
			}
		}
	// DB events
	case superevents.LocalUnsafeUpdateEvent:
		c.chainDBStates[x.ChainID].localUnsafe = types.BlockSealFromRef(x.NewLocalUnsafe)
	case superevents.CrossUnsafeUpdateEvent:
		c.chainDBStates[x.ChainID].crossUnsafe = x.NewCrossUnsafe
		c.chainDBStates[x.ChainID].crossUnsafeWork.ResetBackoff()
	case superevents.LocalSafeUpdateEvent:
		state := c.chainDBStates[x.ChainID]
		state.localSafe = x.NewLocalSafe
		if state.replacementComplete != nil {
			if state.replacementComplete.Replacement.Hash != x.NewLocalSafe.Derived.Hash {
				c.log.Warn("Inconsistent local-safe update reset block-replacement work")
			}
		}
		state.invalidated = types.DerivedBlockRefPair{}
		state.replacementAttributes = nil
		state.replacementComplete = nil
	case superevents.CrossSafeUpdateEvent:
		c.chainDBStates[x.ChainID].crossSafe = x.NewCrossSafe
		c.chainDBStates[x.ChainID].crossSafeWork.ResetBackoff()
	case superevents.FinalizedL2UpdateEvent:
		c.chainDBStates[x.ChainID].finalized = x.FinalizedL2
	case superevents.InvalidateLocalSafeEvent:
		c.chainDBStates[x.ChainID].invalidated = x.Candidate
		c.chainDBStates[x.ChainID].replacementAttributes = nil
		c.chainDBStates[x.ChainID].replacementComplete = nil
	case superevents.CrossSafeWorkErrEvent:
		c.chainDBStates[x.ChainID].crossSafeWork.DoBackoff(x.Err, c.clock.Now())
	case superevents.CrossUnsafeWorkErrEvent:
		c.chainDBStates[x.ChainID].crossUnsafeWork.DoBackoff(x.Err, c.clock.Now())
	default:
		return false
	}

	// check if the event was the desired response to a request
	if req := RequestFromContext(ctx); req != nil {
		if req.cond != nil && req.cond(ev) {
			// If this was the response we were awaiting, then mark the task as done
			c.log.Debug("Received response event", "req", req, "event", ev)
			req.cancel(context.Canceled)
			req.cond = nil
			return true
		}
	}

	return true
}

// Update is a function to inspect states, prioritize what needs to be done, and make new requests
func (c *Controller) Update() {
	for _, s := range c.rwelStates {
		s.maybePollLocalUnsafe()
		s.maybePollCrossSafe()
		s.maybePollFinalized()
	}
	for _, s := range c.rwelStates {
		s.maybeSyncFromL1()
	}
	for _, s := range c.rwelStates {
		s.maybeBackupSync()
	}
	for _, s := range c.pipelineStates {
		s.maybeDerive()
	}
	for _, s := range c.pipelineStates {
		s.maybeProcessAttributes()
	}
	for _, s := range c.pipelineStates {
		s.maybePrepareNextL1()
	}
	for _, s := range c.chainDBStates {
		s.maybeDBInit()
	}
	for _, s := range c.chainDBStates {
		s.maybeReplaceBlock()
	}
	for _, s := range c.chainDBStates {
		s.maybeCrossSafeUpdate()
		s.maybeCrossUnsafeUpdate()
	}
	for _, s := range c.chainDBStates {
		s.maybeDBFinalize()
	}
	for _, s := range c.chainDBStates {
		s.maybeCheckL1Origin()
	}
	for _, s := range c.chainDBStates {
		s.maybeCheckL1Source()
	}
	for _, s := range c.rwelStates {
		s.maybePromoteCrossSafe()
	}
	for _, s := range c.rwelStates {
		s.maybePromoteFinalized()
	}
	for _, s := range c.rwelStates {
		s.maybeELSync()
	}
	for _, s := range c.rwelStates {
		s.maybeIndex()
	}
	for _, s := range c.rwelStates {
		s.maybeNewBlock()
	}
	for _, s := range c.runCfgs {
		s.maybeUpdate()
	}
}

// TODO: do we want to iterate in sorted order for deterministic behavior?

func (c *Controller) Now() time.Time {
	return c.clock.Now()
}

func (c *Controller) IterPipelines(predicates ...Predicate[*PipelineState]) iter.Seq[*PipelineState] {
	return Filter(maps.Values(c.pipelineStates), predicates...)
}

func (c *Controller) IterDBs(predicates ...Predicate[*ChainDBState]) iter.Seq[*ChainDBState] {
	return Filter(maps.Values(c.chainDBStates), predicates...)
}

func (c *Controller) IterRWELs(predicates ...Predicate[*RWELState]) iter.Seq[*RWELState] {
	return Filter(maps.Values(c.rwelStates), predicates...)
}

func (c *Controller) IterRELs(predicates ...Predicate[*RELState]) iter.Seq[*RELState] {
	return Filter(maps.Values(c.relStates), predicates...)
}

func (c *Controller) IterRunCfgs(predicates ...Predicate[*RunCfgState]) iter.Seq[*RunCfgState] {
	return Filter(maps.Values(c.runCfgs), predicates...)
}

func (c *Controller) Payloads(chainID eth.ChainID) (state *PayloadsState, ok bool) {
	state, ok = c.payloadsStates[chainID]
	return
}

func (c *Controller) ChainDB(chainID eth.ChainID) (state *ChainDBState, ok bool) {
	state, ok = c.chainDBStates[chainID]
	return
}

func (c *Controller) L1State() *L1AccessState {
	return c.l1AccessState
}
