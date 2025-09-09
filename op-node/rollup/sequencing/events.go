package sequencing

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type BuildCancelEvent struct {
	Info  eth.PayloadInfo
	Force bool
}

func (ev BuildCancelEvent) String() string {
	return "build-cancel"
}

// BuildInvalidEvent is an internal engine event, to post-process upon invalid attributes.
// Not for temporary processing problems.
type BuildInvalidEvent struct {
	Attributes *derive.AttributesWithParent
	Err        error
}

func (ev BuildInvalidEvent) String() string {
	return "build-invalid"
}

type BuildSealEvent struct {
	Info         eth.PayloadInfo
	BuildStarted time.Time
	// if payload should be promoted to safe (must also be pending safe, see DerivedFrom)
	Concluding bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom eth.L1BlockRef
}

func (ev BuildSealEvent) String() string {
	return "build-seal"
}

// BuildSealedEvent is emitted by the engine when a payload finished building,
// but is not locally inserted as canonical block yet
type BuildSealedEvent struct {
	// if payload should be promoted to (local) safe (must also be pending safe, see DerivedFrom)
	Concluding bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom  eth.L1BlockRef
	BuildStarted time.Time

	Info     eth.PayloadInfo
	Envelope *eth.ExecutionPayloadEnvelope
	Ref      eth.L2BlockRef
}

func (ev BuildSealedEvent) String() string {
	return "build-sealed"
}

type BuildStartedEvent struct {
	Info eth.PayloadInfo

	BuildStarted time.Time

	Parent eth.L2BlockRef

	// if payload should be promoted to (local) safe (must also be pending safe, see DerivedFrom)
	Concluding bool
	// payload is promoted to pending-safe if non-zero
	DerivedFrom eth.L1BlockRef
}

func (ev BuildStartedEvent) String() string {
	return "build-started"
}

// SequencerActionEvent triggers the sequencer to start/seal a block, if active and ready to act.
// This event is used to prioritize sequencer work over derivation work,
// by emitting it before e.g. a derivation-pipeline step.
// A future sequencer in an async world may manage its own execution.
type SequencerActionEvent struct {
}

func (ev SequencerActionEvent) String() string {
	return "sequencer-action"
}
