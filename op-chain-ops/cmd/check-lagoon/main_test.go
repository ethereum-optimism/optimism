package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestSDMConfigLoadsFromNestedTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "check-lagoon.toml")
	require.NoError(t, os.WriteFile(path, []byte(`
[sdm]
l2 = "http://producer"
l2-verifier = "http://verifier"
rollup-rpc = "http://rollup"
account = "abcd"
batch-size = 17
slot-count = 23
opt-in = false
timeout = "45s"
`), 0o600))

	app := newApp()
	app.Writer = io.Discard
	app.ErrWriter = io.Discard
	var observed bool
	for _, command := range app.Commands {
		if command.Name != "sdm" {
			continue
		}
		for _, subcommand := range command.Subcommands {
			if subcommand.Name != "all" {
				continue
			}
			subcommand.Action = func(ctx *cli.Context) error {
				observed = true
				require.Equal(t, "http://producer", ctx.String(SDMEndpointL2.Name))
				require.Equal(t, "http://verifier", ctx.String(SDMVerifierL2.Name))
				require.Equal(t, "http://rollup", ctx.String(SDMRollupRPC.Name))
				require.Equal(t, "abcd", ctx.String(SDMAccountKey.Name))
				require.Equal(t, 17, ctx.Int(SDMBatchSize.Name))
				require.Equal(t, uint64(23), ctx.Uint64(SDMSlotCount.Name))
				require.False(t, ctx.Bool(SDMOptIn.Name))
				require.Equal(t, 45*time.Second, ctx.Duration(SDMTimeout.Name))
				return nil
			}
		}
	}
	require.NoError(t, app.Run([]string{"check-lagoon", "sdm", "all", "--config", path}))
	require.True(t, observed)
}

func TestCombinedLagoonConfigLoadsBothWorkloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "check-lagoon.toml")
	require.NoError(t, os.WriteFile(path, []byte(`
l2-a = "http://chain-a"
l2-b = "http://chain-b"
account = "interop-key"

[sdm]
account = "sdm-key"
rollup-rpc-a = "http://rollup-a"
rollup-rpc-b = "http://rollup-b"
contract-a = "0x0000000000000000000000000000000000000001"
contract-b = "0x0000000000000000000000000000000000000002"
batch-size = 15
`), 0o600))

	app := newApp()
	app.Writer = io.Discard
	app.ErrWriter = io.Discard
	var observed bool
	for _, command := range app.Commands {
		if command.Name != "all" {
			continue
		}
		command.Action = func(ctx *cli.Context) error {
			observed = true
			require.Equal(t, "http://chain-a", ctx.String(EndpointL2A.Name))
			require.Equal(t, "http://chain-b", ctx.String(EndpointL2B.Name))
			require.Equal(t, "interop-key", ctx.String(AccountKey.Name))
			require.Equal(t, "sdm-key", ctx.String(AllSDMAccountKey.Name))
			require.Equal(t, "http://rollup-a", ctx.String(AllSDMRollupRPCA.Name))
			require.Equal(t, "http://rollup-b", ctx.String(AllSDMRollupRPCB.Name))
			require.Equal(t, 15, ctx.Int(SDMBatchSize.Name))
			return nil
		}
	}
	require.NoError(t, app.Run([]string{"check-lagoon", "all", "--config", path}))
	require.True(t, observed)
}

func TestCombinedLagoonRejectsSharedSender(t *testing.T) {
	const key = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	app := newApp()
	app.Writer = io.Discard
	app.ErrWriter = io.Discard
	err := app.Run([]string{
		"check-lagoon", "all",
		"--account", key,
		"--sdm-account", key,
		"--sdm-rollup-rpc-a", "http://rollup-a",
		"--sdm-rollup-rpc-b", "http://rollup-b",
	})
	require.ErrorContains(t, err, "must differ")
}

func TestSDMCLIOverridesConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "check-lagoon.toml")
	require.NoError(t, os.WriteFile(path, []byte("[sdm]\nbatch-size = 17\n"), 0o600))

	app := newApp()
	app.Writer = io.Discard
	app.ErrWriter = io.Discard
	for _, command := range app.Commands {
		if command.Name != "sdm" {
			continue
		}
		for _, subcommand := range command.Subcommands {
			if subcommand.Name == "all" {
				subcommand.Action = func(ctx *cli.Context) error {
					require.Equal(t, 19, ctx.Int(SDMBatchSize.Name))
					return nil
				}
			}
		}
	}
	require.NoError(t, app.Run([]string{"check-lagoon", "sdm", "all", "--config", path, "--batch-size", "19"}))
}
