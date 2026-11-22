package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type Ensemble struct {
	// Endpoints is a no-op list of endpoints
	// to prevent repetitive endpoints in a yaml.
	// This list can all be declared at the top, and items can be referenced with yaml refs.
	Endpoints []string `yaml:"endpoints"`

	Builders   map[seqtypes.BuilderID]*BuilderEntry     `yaml:"builders"`
	Signers    map[seqtypes.SignerID]*SignerEntry       `yaml:"signers"`
	Committers map[seqtypes.CommitterID]*CommitterEntry `yaml:"committers"`
	Publishers map[seqtypes.PublisherID]*PublisherEntry `yaml:"publishers"`
	Sequencers map[seqtypes.SequencerID]*SequencerEntry `yaml:"sequencers"`
}

var _ work.Loader = (*Ensemble)(nil)

// Load is a short-cut to skip the config-loading phase, and use an existing config instead.
// This can be used by tests to plug in a config directly,
// without having to store it on disk somewhere.
func (c *Ensemble) Load(ctx context.Context) (work.Starter, error) {
	return c, nil
}

var _ work.Starter = (*Ensemble)(nil)

// Start sets up the configured group of builders.
// If any service fails to start, the services that did start are closed again.
func (c *Ensemble) Start(ctx context.Context, opts *work.StartOpts) (*work.Ensemble, error) {
	ensemble := new(work.Ensemble)
	if err := c.startInto(ctx, opts, ensemble); err != nil {
		return nil, errors.Join(err, ensemble.Close())
	}
	return ensemble, nil
}

// startInto starts the configured services and adds each one to the ensemble as soon as it has started,
// so that the caller can close them if a later service fails to start.
func (c *Ensemble) startInto(ctx context.Context, opts *work.StartOpts, ensemble *work.Ensemble) error {
	serviceOpts := &work.ServiceOpts{
		StartOpts: opts,
		Services:  ensemble,
	}
	if err := startAndAdd(ctx, c.Builders, serviceOpts, ensemble.AddBuilder); err != nil {
		return fmt.Errorf("failed to start builders: %w", err)
	}
	if err := startAndAdd(ctx, c.Signers, serviceOpts, ensemble.AddSigner); err != nil {
		return fmt.Errorf("failed to start signers: %w", err)
	}
	if err := startAndAdd(ctx, c.Committers, serviceOpts, ensemble.AddCommitter); err != nil {
		return fmt.Errorf("failed to start committers: %w", err)
	}
	if err := startAndAdd(ctx, c.Publishers, serviceOpts, ensemble.AddPublisher); err != nil {
		return fmt.Errorf("failed to start publishers: %w", err)
	}
	if err := startAndAdd(ctx, c.Sequencers, serviceOpts, ensemble.AddSequencer); err != nil {
		return fmt.Errorf("failed to start sequencers: %w", err)
	}
	return nil
}

type valueIface[K comparable, R any] interface {
	Start(ctx context.Context, id K, opts *work.ServiceOpts) (result R, err error)
}

type keyIface interface {
	comparable
	String() string
}

// startAndAdd is a util function to start entities from a configuration map, and add them to the ensemble.
func startAndAdd[K keyIface, R any, E valueIface[K, R]](
	ctx context.Context,
	entries map[K]E,
	opts *work.ServiceOpts,
	addFn func(v R) error) error {
	for id, conf := range entries {
		v, err := conf.Start(ctx, id, opts)
		if err != nil {
			return fmt.Errorf("failed to start %q: %w", id, err)
		}
		if err := addFn(v); err != nil {
			return fmt.Errorf("failed to add %q: %w", id, err)
		}
	}
	return nil
}
