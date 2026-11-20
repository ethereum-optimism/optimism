package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestRPCCapabilityFlags(t *testing.T) {
	for _, command := range [][]string{{"all"}, {"sdm", "all"}, {"sdm", "block"}, {"sdm", "verifier"}, {"sdm", "rpc"}} {
		t.Run(command[len(command)-1], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "options.toml")
			require.NoError(t, os.WriteFile(path, []byte("[sdm]\nrpc-js = true\nrpc-ots = true\n"), 0o600))
			for _, test := range []struct {
				args    []string
				js, ots bool
			}{
				{nil, false, false},
				{[]string{"--sdm.rpc-js"}, true, false},
				{[]string{"--sdm.rpc-ots"}, false, true},
				{[]string{"--sdm-rpc-js", "--sdm-rpc-ots"}, true, true},
				{[]string{"--config", path}, true, true},
				{[]string{"--config", path, "--sdm.rpc-js=false", "--sdm.rpc-ots=false"}, false, false},
			} {
				app := newApp()
				app.Writer = io.Discard
				app.ErrWriter = io.Discard
				seen := false
				action := func(c *cli.Context) error {
					seen = true
					require.Equal(t, test.js, c.Bool(SDMRPCJS.Name))
					require.Equal(t, test.ots, c.Bool(SDMRPCOTS.Name))
					if len(command) == 2 {
						cfg, cleanup, err := resolveSDMConfig(context.Background(), log.New(), c, false)
						require.NoError(t, err)
						defer cleanup()
						require.Equal(t, test.js, cfg.RPCOptions.RequireJS)
						require.Equal(t, test.ots, cfg.RPCOptions.CheckOTS)
						require.Equal(t, test.js || test.ots, cfg.CheckRPC, "explicit capability flags enable replay checks")
					}
					return nil
				}
				for _, cmd := range app.Commands {
					if cmd.Name != command[0] {
						continue
					}
					if len(command) == 1 {
						cmd.Action = action
						continue
					}
					for _, sub := range cmd.Subcommands {
						if sub.Name == command[1] {
							sub.Action = action
						}
					}
				}
				args := append([]string{"check-lagoon"}, command...)
				args = append(args, test.args...)
				require.NoError(t, app.Run(args))
				require.True(t, seen)
			}
		})
	}
}

func TestRPCCapabilityEnvironment(t *testing.T) {
	t.Setenv("CHECK_LAGOON_SDM_RPC_JS", "true")
	t.Setenv("CHECK_LAGOON_SDM_RPC_OTS", "true")
	app := newApp()
	app.Writer = io.Discard
	app.ErrWriter = io.Discard
	seen := false
	for _, command := range app.Commands {
		if command.Name == "all" {
			command.Action = func(c *cli.Context) error {
				seen = true
				require.True(t, c.Bool(SDMRPCJS.Name))
				require.True(t, c.Bool(SDMRPCOTS.Name))
				return nil
			}
		}
	}
	require.NoError(t, app.Run([]string{"check-lagoon", "all"}))
	require.True(t, seen)
}
