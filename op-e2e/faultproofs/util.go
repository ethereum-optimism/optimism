package faultproofs

import (
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

type faultDisputeConfigOpts func(cfg *op_e2e.SystemConfig)

func withLargeBatches() faultDisputeConfigOpts {
	return func(cfg *op_e2e.SystemConfig) {
		maxTxDataSize := uint64(131072) // As per the Ethereum spec.
		// Allow the batcher to produce really huge calldata transactions.
		// Make the max deliberately bigger than the target but still with some padding below the actual limit
		cfg.BatcherTargetL1TxSizeBytes = maxTxDataSize - 5000
		cfg.BatcherMaxL1TxSizeBytes = maxTxDataSize - 1000
	}
}

func startFaultDisputeSystem(t *testing.T, opts ...faultDisputeConfigOpts) (*op_e2e.System, *ethclient.Client) {
	cfg := op_e2e.DefaultSystemConfig(t)
	delete(cfg.Nodes, "verifier")
	for _, opt := range opts {
		opt(&cfg)
	}
	cfg.DeployConfig.SequencerWindowSize = 4
	cfg.DeployConfig.FinalizationPeriodSeconds = 2
	cfg.SupportL1TimeTravel = true
	cfg.DeployConfig.L2OutputOracleSubmissionInterval = 1
	cfg.NonFinalizedProposals = true // Submit output proposals asap
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	return sys, sys.Clients["l1"]
}
