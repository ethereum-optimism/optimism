package bootstrap

import (
	"io"
	"log/slog"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// Runs each command's real action. Every Check() validates the interval first, so an out-of-range
// value is only rejected if the command copied the flag into its config. The empty RPC and key flags
// override the environment, so an accepted value always stops at the next check, before any RPC.
func TestReceiptQueryIntervalFlagReachesConfig(t *testing.T) {
	root, slogDefault := log.Root(), slog.Default()
	t.Cleanup(func() {
		log.SetDefault(root)
		slog.SetDefault(slogDefault)
	})

	tests := []struct {
		name    string
		flags   []cli.Flag
		action  cli.ActionFunc
		args    []string
		nextErr string
	}{
		{
			name:    "apply",
			flags:   deployer.ApplyFlags,
			action:  deployer.ApplyCLI(),
			args:    []string{"--deployment-target=live", "--workdir=" + t.TempDir()},
			nextErr: "l1 RPC URL must be specified for live deployment",
		},
		{
			name:    "continue",
			flags:   deployer.ContinueFlags,
			action:  deployer.ContinueCLI(),
			args:    []string{"--workdir=" + t.TempDir()},
			nextErr: "private key must be specified",
		},
		{name: "superchain", flags: SuperchainFlags, action: SuperchainCLI, nextErr: "l1RPCUrl must be specified"},
		{name: "implementations", flags: ImplementationsFlags, action: ImplementationsCLI, nextErr: "l1RPCUrl must be specified"},
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
				args := []string{"op-deployer", "cmd", "--l1-rpc-url=", "--private-key=", "--receipt-query-interval=" + interval}
				return app.RunContext(t.Context(), append(args, tt.args...))
			}
			for _, bad := range []string{"49ms", "2s", "-1s"} {
				require.ErrorContains(t, run(bad), "receipt query interval must be 0 or between 50ms and 1s")
			}
			for _, good := range []string{"0", "50ms", "1s"} {
				require.ErrorContains(t, run(good), tt.nextErr)
			}
		})
	}
}
