package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// cliInitIntent creates an intent using the CLI init command
func cliInitIntent(t *testing.T, runner *CLITestRunner, l1ChainID uint64, l2ChainIDs []common.Hash) (*state.Intent, *state.State) {
	workDir := runner.GetWorkDir()

	chainIDStrings := make([]string, len(l2ChainIDs))
	for i, id := range l2ChainIDs {
		chainIDStrings[i] = new(uint256.Int).SetBytes32(id[:]).String()
	}
	l2ChainIDsStr := strings.Join(chainIDStrings, ",")

	runner.ExpectSuccess(t, []string{
		"init",
		"--l1-chain-id", fmt.Sprintf("%d", l1ChainID),
		"--l2-chain-ids", l2ChainIDsStr,
		"--intent-type", "custom",
		"--workdir", workDir,
	}, nil)

	intent, err := pipeline.ReadIntent(workDir)
	require.NoError(t, err)

	st, err := pipeline.ReadState(workDir)
	require.NoError(t, err)

	return intent, st
}
