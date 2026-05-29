package conductor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigCheckRollupBoostAndNextMutuallyExclusive(t *testing.T) {
	cfg := &Config{
		ConsensusAddr:                 "127.0.0.1",
		ConsensusPort:                 9000,
		RaftServerID:                  "server-1",
		RaftStorageDir:                "/tmp/op-conductor",
		NodeRPC:                       "http://node.example",
		ExecutionRPC:                  "http://exec.example",
		RollupBoostEnabled:            true,
		RollupBoostNextEnabled:        true,
		RollupBoostNextHealthcheckURL: "http://rollupboost.example",
	}

	err := cfg.Check()
	require.Error(t, err)
	require.Contains(t, err.Error(), "only one of rollup-boost or rollup-boost next healthchecks can be enabled")
}

func TestConfigCheckReorgRecoveryRequiresExecutionWSURL(t *testing.T) {
	cfg := &Config{
		ConsensusAddr:        "127.0.0.1",
		ConsensusPort:        9000,
		RaftServerID:         "server-1",
		RaftStorageDir:       "/tmp/op-conductor",
		NodeRPC:              "http://node.example",
		ExecutionRPC:         "http://exec.example",
		ReorgRecoveryEnabled: true,
		// ExecutionWSURL intentionally empty.
	}

	err := cfg.Check()
	require.Error(t, err)
	require.Contains(t, err.Error(), "reorg-recovery enabled but missing execution WS URL")
}
