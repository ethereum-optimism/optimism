package faultproofs

import (
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
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
	// Allow the batcher to produce really huge calldata transactions.
	cfg.BatcherTargetL1TxSizeBytes = 130072 // A bit under the max tx size as per Ethereum spec
	cfg.BatcherMaxL1TxSizeBytes = 131072    // The absolute limit as per Ethereum spec
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	return sys, sys.Clients["l1"]
}
