package noopbuilder

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

var ErrNoBuild = errors.New("no building supported")

type Builder struct {
	id       seqtypes.BuilderID
	registry work.Jobs
}

var _ work.Builder = (*Builder)(nil)

func NewBuilder(id seqtypes.BuilderID, registry work.Jobs) *Builder {
	return &Builder{id: id, registry: registry}
}

func (n *Builder) NewJob(ctx context.Context, opts *seqtypes.BuildOpts) (work.BuildJob, error) {
	id := seqtypes.RandomJobID()
	job := &Job{
		id: id,
		unregister: func() {
			n.registry.UnregisterJob(id)
		},
	}
	if err := n.registry.RegisterJob(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (n *Builder) Close() error {
	return nil
}

func (n *Builder) String() string {
	return "noop-builder-" + n.id.String()
}

func (n *Builder) ID() seqtypes.BuilderID {
	return n.id
}
