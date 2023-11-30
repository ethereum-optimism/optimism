package faultproofs

import (
	"context"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	l2oo2 "github.com/ethereum-optimism/optimism/op-e2e/e2eutils/l2oo"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func startFaultDisputeSystem(t *testing.T) (*op_e2e.System, *ethclient.Client) {
	cfg := op_e2e.DefaultSystemConfig(t)
	delete(cfg.Nodes, "verifier")
	cfg.DeployConfig.SequencerWindowSize = 4
	cfg.DeployConfig.FinalizationPeriodSeconds = 2
	cfg.SupportL1TimeTravel = true
	cfg.DeployConfig.L2OutputOracleSubmissionInterval = 1
	cfg.NonFinalizedProposals = true // Submit output proposals asap
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	return sys, sys.Clients["l1"]
}

// setupDisputeGameForInvalidOutputRoot sets up an L2 chain with at least one valid output root followed by an invalid output root.
// A cannon dispute game is started to dispute the invalid output root with the correct root claim provided.
// An honest challenger is run to defend the root claim (ie disagree with the invalid output root).
func setupDisputeGameForInvalidOutputRoot(t *testing.T, outputRoot common.Hash) (*op_e2e.System, *ethclient.Client, *disputegame.CannonGameHelper, *disputegame.HonestHelper) {
	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)

	l2oo := l2oo2.NewL2OOHelper(t, sys.Cfg.L1Deployments, l1Client, sys.Cfg.Secrets.Proposer, sys.RollupConfig)

	// Wait for one valid output root to be submitted
	l2oo.WaitForProposals(ctx, 1)

	err := sys.L2OutputSubmitter.Driver().StopL2OutputSubmitting()
	require.NoError(t, err)
	sys.L2OutputSubmitter = nil

	// Submit an invalid output root
	l2oo.PublishNextOutput(ctx, outputRoot)

	l1Endpoint := sys.NodeEndpoint("l1")
	l2Endpoint := sys.NodeEndpoint("sequencer")

	// Dispute the new output root by creating a new game with the correct cannon trace.
	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys.Cfg.L1Deployments, l1Client)
	game, correctTrace := disputeGameFactory.StartCannonGameWithCorrectRoot(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Endpoint, l2Endpoint,
		challenger.WithPrivKey(sys.Cfg.Secrets.Mallory),
	)
	require.NotNil(t, game)

	// Start the honest challenger
	game.StartChallenger(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Endpoint, l2Endpoint, "Defender",
		challenger.WithPrivKey(sys.Cfg.Secrets.Mallory),
	)
	return sys, l1Client, game, correctTrace
}
