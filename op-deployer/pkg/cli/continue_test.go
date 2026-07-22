package cli

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/stretchr/testify/require"
)

func TestContinueCommandRegistered(t *testing.T) {
	app := NewApp("v0.0.0")
	command := app.Command("continue")
	require.NotNil(t, command)
	require.Equal(t, "broadcasts and validates an already-prepared chain deployment", command.Usage)

	flags := make(map[string]bool)
	for _, cliFlag := range command.Flags {
		for _, name := range cliFlag.Names() {
			flags[name] = true
		}
	}
	require.False(t, flags[deployer.UseForgeFlagName])
	require.False(t, flags[deployer.DeploymentTargetFlag.Name])
}
