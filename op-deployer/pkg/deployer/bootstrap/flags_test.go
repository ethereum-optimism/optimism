package bootstrap

import (
	"io"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// Runs each command's real action. Every Check() validates the interval first, so an out-of-range
// value is only rejected if the command copied the flag into its config. Nothing here reaches an RPC.
func TestReceiptQueryIntervalFlagReachesConfig(t *testing.T) {
	const rejected = "receipt query interval must be"
	tests := []struct {
		name   string
		flags  []cli.Flag
		action cli.ActionFunc
		args   []string
	}{
		{name: "apply", flags: deployer.ApplyFlags, action: deployer.ApplyCLI()},
		{name: "apply genesis", flags: deployer.ApplyFlags, action: deployer.ApplyCLI(), args: []string{"--deployment-target=genesis"}},
		{name: "apply forge", flags: deployer.ApplyFlags, action: deployer.ApplyCLI(), args: []string{"--use-forge"}},
		{name: "continue", flags: deployer.ContinueFlags, action: deployer.ContinueCLI()},
		{name: "superchain", flags: SuperchainFlags, action: SuperchainCLI},
		{name: "superchain forge", flags: SuperchainFlags, action: SuperchainCLI, args: []string{"--use-forge"}},
		{name: "implementations", flags: ImplementationsFlags, action: ImplementationsCLI},
		{name: "implementations forge", flags: ImplementationsFlags, action: ImplementationsCLI, args: []string{"--use-forge"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := func(interval string) error {
				app := &cli.App{
					Writer:    io.Discard,
					ErrWriter: io.Discard,
					Flags:     deployer.GlobalFlags,
					Commands:  []*cli.Command{{Name: "cmd", Flags: tt.flags, Action: tt.action}},
				}
				return app.Run(append([]string{"op-deployer", "cmd", "--receipt-query-interval=" + interval}, tt.args...))
			}
			for _, bad := range []string{"9ms", "13s", "-1s"} {
				require.ErrorContains(t, run(bad), rejected)
			}
			// These fail on the next check instead, since no other flags are set.
			for _, good := range []string{"0", "10ms", "12s"} {
				err := run(good)
				require.Error(t, err)
				require.NotContains(t, err.Error(), rejected)
			}
		})
	}
}
