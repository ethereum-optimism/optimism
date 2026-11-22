package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/metrics"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/builders/noopbuilder"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/committers/noopcommitter"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/publishers/nooppublisher"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/sequencers/noopseq"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/signers/noopsigner"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

func TestEnsemble_Start(t *testing.T) {
	startOpts := func(t *testing.T) *work.StartOpts {
		return &work.StartOpts{
			Log:     testlog.Logger(t, log.LevelError),
			Metrics: &metrics.NoopMetrics{},
		}
	}
	t.Run("empty", func(t *testing.T) {
		v := new(Ensemble)
		out, err := v.Start(context.Background(), nil)
		require.NoError(t, err)
		require.Empty(t, out.Builders())
		require.Empty(t, out.Signers())
		require.Empty(t, out.Committers())
		require.Empty(t, out.Publishers())
		require.Empty(t, out.Sequencers())
	})
	t.Run("noops", func(t *testing.T) {
		v := &Ensemble{
			Endpoints: nil,
			Builders: map[seqtypes.BuilderID]*BuilderEntry{
				"noop-builder": {
					Noop: &noopbuilder.Config{},
				},
			},
			Signers: map[seqtypes.SignerID]*SignerEntry{
				"noop-signer": {
					Noop: &noopsigner.Config{},
				},
			},
			Committers: map[seqtypes.CommitterID]*CommitterEntry{
				"noop-committer": {
					Noop: &noopcommitter.Config{},
				},
			},
			Publishers: map[seqtypes.PublisherID]*PublisherEntry{
				"noop-publisher": {
					Noop: &nooppublisher.Config{},
				},
			},
			Sequencers: map[seqtypes.SequencerID]*SequencerEntry{
				"noop-sequencer": {
					Noop: &noopseq.Config{},
				},
			},
		}
		out, err := v.Start(context.Background(), startOpts(t))
		require.NoError(t, err)
		require.Len(t, out.Builders(), 1)
		require.Len(t, out.Signers(), 1)
		require.Len(t, out.Committers(), 1)
		require.Len(t, out.Publishers(), 1)
		require.Len(t, out.Sequencers(), 1)
	})
	t.Run("builder fails", func(t *testing.T) {
		v := &Ensemble{
			Builders: map[seqtypes.BuilderID]*BuilderEntry{
				"bad-builder": {}, // no kind configured
			},
		}
		out, err := v.Start(context.Background(), startOpts(t))
		require.ErrorIs(t, err, seqtypes.ErrUnknownKind)
		require.ErrorContains(t, err, "failed to start builders")
		require.Nil(t, out)
	})
	t.Run("later service fails", func(t *testing.T) {
		v := &Ensemble{
			Builders: map[seqtypes.BuilderID]*BuilderEntry{
				"noop-builder": {
					Noop: &noopbuilder.Config{},
				},
			},
			Signers: map[seqtypes.SignerID]*SignerEntry{
				"bad-signer": {}, // no kind configured
			},
		}
		out, err := v.Start(context.Background(), startOpts(t))
		require.ErrorIs(t, err, seqtypes.ErrUnknownKind)
		require.ErrorContains(t, err, "failed to start signers")
		require.Nil(t, out)
	})
}
