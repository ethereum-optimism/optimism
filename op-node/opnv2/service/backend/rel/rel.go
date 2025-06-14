package rel

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type L2Source interface {
	apis.L2EthClient
}

// REL is a Read-only Execution Layer node
type REL struct {

	// TODO latest L2 head
	// TODO last seen status

	chainID  eth.ChainID
	l2Source L2Source
	cfg      *rollup.Config

	id  ID
	ctx context.Context

	emitter event.Emitter
}

var _ event.AttachEmitter = (*REL)(nil)
var _ event.Deriver = (*REL)(nil)

func NewREL(rootCtx context.Context, id ID, l2Source L2Source, cfg *rollup.Config) *REL {
	return &REL{
		chainID:  eth.ChainIDFromBig(cfg.L2ChainID),
		l2Source: l2Source,
		cfg:      cfg,
		id:       id,
		ctx:      WithID(rootCtx, id),
	}
}

func (r *REL) AttachEmitter(em event.Emitter) {
	r.emitter = em
}

func (r *REL) ID() ID {
	return r.id
}

func (r *REL) ChainID() eth.ChainID {
	return r.chainID
}

// IsSyncedTo answers if the REL can serve things before the given block number
func (r *REL) IsSyncedTo(num uint64) bool {
	return false
}

func (r *REL) OnEvent(ctx context.Context, ev event.Event) bool {
	if IDFromContext(ctx) != r.id { // If the event is not for us, ignore it
		return false
	}

	// TODO: retrieve context from ev, and filter if the event is directed at us.

	// TODO: handle requests to consolidate attributes

	// TODO: handle requests to poll for new L2 unsafe head

	return false
}

type LocalUnsafeUpdateEvent struct {
	LocalUnsafe eth.L2BlockRef
}

func (ev LocalUnsafeUpdateEvent) String() string {
	return "REL-local-unsafe-update"
}

type CrossSafeUpdateEvent struct {
	CrossSafe eth.L2BlockRef
}

func (ev CrossSafeUpdateEvent) String() string {
	return "REL-cross-safe-update"
}

type FinalizedUpdateEvent struct {
	Finalized eth.L2BlockRef
}

func (ev FinalizedUpdateEvent) String() string {
	return "REL-finalized-update"
}
