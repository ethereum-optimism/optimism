package fullseq

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/HashKeyChain/verse/op-service/eth"
	"github.com/HashKeyChain/verse/op-service/testlog"
	"github.com/HashKeyChain/verse/op-test-sequencer/metrics"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work/builders/noopbuilder"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work/committers/noopcommitter"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work/publishers/nooppublisher"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/backend/work/signers/noopsigner"
	"github.com/HashKeyChain/verse/op-test-sequencer/sequencer/seqtypes"
)

func TestConfig(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	builderID := seqtypes.BuilderID("test-builder")
	signerID := seqtypes.SignerID("test-signer")
	committerID := seqtypes.CommitterID("test-committer")
	publisherID := seqtypes.PublisherID("test-publisher")
	sequencerID := seqtypes.SequencerID("test-sequencer")
	ensemble := &work.Ensemble{}
	jobs := work.NewJobRegistry()
	require.NoError(t, ensemble.AddBuilder(noopbuilder.NewBuilder(builderID, jobs)))
	require.NoError(t, ensemble.AddSigner(noopsigner.NewSigner(signerID, logger)))
	require.NoError(t, ensemble.AddCommitter(noopcommitter.NewCommitter(committerID, logger)))
	require.NoError(t, ensemble.AddPublisher(nooppublisher.NewPublisher(publisherID, logger)))
	cfg := &Config{
		ChainID:             eth.ChainIDFromUInt64(1),
		Builder:             builderID,
		Signer:              signerID,
		Committer:           committerID,
		Publisher:           publisherID,
		SequencerConfDepth:  2,
		SequencerEnabled:    true,
		SequencerStopped:    true,
		SequencerMaxSafeLag: 10,
	}
	opts := &work.ServiceOpts{
		StartOpts: &work.StartOpts{
			Log:     logger,
			Metrics: &metrics.NoopMetrics{},
			Jobs:    jobs,
		},
		Services: ensemble,
	}
	seq, err := cfg.Start(context.Background(), sequencerID, opts)
	require.NoError(t, err)
	require.NoError(t, ensemble.AddSequencer(seq))
}
