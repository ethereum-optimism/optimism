package verifier

import (
	"context"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-service/log"

	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/stretchr/testify/require"
)

// TestConfirmationDepth runs the rollup with both sequencer and verifier not immediately processing the tip of the chain.
func TestConfirmationDepth(t *testing.T) {
	op_e2e.InitParallel(t)

	cfg := e2esys.DefaultSystemConfig(t)
	cfg.DeployConfig.SequencerWindowSize = 4
	cfg.DeployConfig.MaxSequencerDrift = 10 * cfg.DeployConfig.L1BlockTime
	seqConfDepth := uint64(2)
	verConfDepth := uint64(5)
	cfg.Nodes["sequencer"].Driver.SequencerConfDepth = seqConfDepth
	cfg.Nodes["sequencer"].Driver.VerifierConfDepth = 0
	cfg.Nodes["verifier"].Driver.VerifierConfDepth = verConfDepth

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")

	log := testlog.Logger(t, log.LevelInfo)
	log.Info("genesis", "l2", sys.RollupConfig.Genesis.L2, "l1", sys.RollupConfig.Genesis.L1, "l2_time", sys.RollupConfig.Genesis.L2Time)

	l1Client := sys.NodeClient("l1")
	l2Seq := sys.NodeSourceClient("sequencer")
	l2Verif := sys.NodeSourceClient("verifier")

	// Wait enough time for the sequencer to submit a block with distance from L1 head, submit it,
	// and for the slower verifier to read a full sequence window and cover confirmation depth for reading and some margin
	<-time.After(time.Duration((cfg.DeployConfig.SequencerWindowSize+verConfDepth+3)*cfg.DeployConfig.L1BlockTime) * time.Second)

	// within a second, get both L1 and L2 verifier and sequencer block heads
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	l1Head, err := l1Client.HeaderByNumber(ctx, nil)
	require.NoError(t, err)
	l2SeqHead, err := l2Seq.InfoByLabel(ctx, eth.Unsafe)
	require.NoError(t, err)
	l2VerHead, err := l2Verif.InfoByLabel(ctx, eth.Unsafe)
	require.NoError(t, err)

	_, l2SeqL1Info, err := l2Seq.InfoAndFirstDepositByHash(ctx, l2SeqHead.Hash())
	require.NoError(t, err)
	seqInfo, err := derive.L1BlockInfoFromBytes(sys.RollupConfig, l2SeqHead.Time(), l2SeqL1Info.Data)
	require.NoError(t, err)
	require.LessOrEqual(t, seqInfo.Number+seqConfDepth, bigs.Uint64Strict(l1Head.Number), "the seq L2 head block should have an origin older than the L1 head block by at least the sequencer conf depth")

	_, l2VerL1Info, err := l2Verif.InfoAndFirstDepositByHash(ctx, l2VerHead.Hash())
	require.NoError(t, err)
	verInfo, err := derive.L1BlockInfoFromBytes(sys.RollupConfig, l2VerHead.Time(), l2VerL1Info.Data)
	require.NoError(t, err)
	require.LessOrEqual(t, verInfo.Number+verConfDepth, bigs.Uint64Strict(l1Head.Number), "the ver L2 head block should have an origin older than the L1 head block by at least the verifier conf depth")
}
