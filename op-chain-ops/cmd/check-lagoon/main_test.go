package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-chain-ops/cmd/check-lagoon/sdmcheck"
)

func TestSDMConfigLoadsFromNestedTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "check-lagoon.toml")
	// Root keys must not leak into the same-named [sdm] keys.
	require.NoError(t, os.WriteFile(path, []byte(`
account = "interop-key"
json = true

[sdm]
l2 = "http://producer"
l2-verifier = "http://verifier"
rollup-rpc = "http://rollup"
account = "abcd"
opt-in = true
json = false
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
				require.True(t, ctx.Bool(SDMOptIn.Name))
				require.False(t, ctx.Bool(SDMJSON.Name))
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
			require.Equal(t, "sdm-key", ctx.String(SDMAccountKey.Name))
			require.Equal(t, "http://rollup-a", ctx.String(AllSDMRollupRPCA.Name))
			require.Equal(t, "http://rollup-b", ctx.String(AllSDMRollupRPCB.Name))
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
	require.NoError(t, os.WriteFile(path, []byte("[sdm]\nl2 = \"http://from-config\"\n"), 0o600))

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
					require.Equal(t, "http://from-flag", ctx.String(SDMEndpointL2.Name))
					return nil
				}
			}
		}
	}
	require.NoError(t, app.Run([]string{"check-lagoon", "sdm", "all", "--config", path, "--sdm-l2", "http://from-flag"}))
}

func TestSDMOptInDefaultsOff(t *testing.T) {
	app := newApp()
	app.Writer = io.Discard
	app.ErrWriter = io.Discard
	var observed int
	check := func(ctx *cli.Context) error {
		observed++
		require.False(t, ctx.Bool(SDMOptIn.Name))
		return nil
	}
	for _, command := range app.Commands {
		if command.Name == "all" {
			command.Action = check
		}
		if command.Name == "sdm" {
			for _, subcommand := range command.Subcommands {
				if subcommand.Name == "all" {
					subcommand.Action = check
				}
			}
		}
	}
	require.NoError(t, app.Run([]string{"check-lagoon", "all"}))
	require.NoError(t, app.Run([]string{"check-lagoon", "sdm", "all"}))
	require.Equal(t, 2, observed)
}

func TestSDMBlockCommandsRejectWorkloadFlags(t *testing.T) {
	for _, subcommand := range []string{"block", "verifier"} {
		app := newApp()
		app.Writer = io.Discard
		app.ErrWriter = io.Discard
		err := app.Run([]string{"check-lagoon", "sdm", subcommand, "--sdm.account", "abcd"})
		require.ErrorContains(t, err, "flag provided but not defined", subcommand)
	}
}

func TestRunLagoonLegsFailureDoesNotCancelOtherLegs(t *testing.T) {
	interopFailed := make(chan struct{})
	// Each SDM leg outlives the Interop failure and passes only if ctx is still live.
	sdmLeg := func(ctx context.Context) (*sdmcheck.Result, error) {
		<-interopFailed
		// Gives a cancel-on-first-error implementation time to cancel ctx; the
		// sleep only affects how reliably a regression is caught, never a pass.
		time.Sleep(10 * time.Millisecond)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return &sdmcheck.Result{RPCURL: "sdm"}, nil
	}
	var result lagoonAllResult
	err := runLagoonLegs(context.Background(), lagoonLegs{
		interop: func(context.Context) error {
			defer close(interopFailed)
			return errors.New("relay failed")
		},
		sdmA: sdmLeg,
		sdmB: sdmLeg,
	}, &result)

	require.ErrorContains(t, err, "Interop smoke: relay failed")
	require.False(t, result.InteropPassed)
	require.Equal(t, "relay failed", result.InteropError)
	require.NotNil(t, result.SDMChainA)
	require.Empty(t, result.SDMChainAError)
	require.NotNil(t, result.SDMChainB)
	require.Empty(t, result.SDMChainBError)
}

func TestRunLagoonLegsJoinsEveryFailure(t *testing.T) {
	var result lagoonAllResult
	err := runLagoonLegs(context.Background(), lagoonLegs{
		interop: func(context.Context) error { return nil },
		sdmA:    func(context.Context) (*sdmcheck.Result, error) { return nil, errors.New("no PostExec tx") },
		sdmB: func(context.Context) (*sdmcheck.Result, error) {
			return &sdmcheck.Result{RPCURL: "b"}, errors.New("verifier disagreed")
		},
	}, &result)

	require.ErrorContains(t, err, "chain A SDM: no PostExec tx")
	require.ErrorContains(t, err, "chain B SDM: verifier disagreed")
	require.True(t, result.InteropPassed)
	require.Empty(t, result.InteropError)
	require.Nil(t, result.SDMChainA)
	require.Equal(t, "no PostExec tx", result.SDMChainAError)
	require.Equal(t, "b", result.SDMChainB.RPCURL)
	require.Equal(t, "verifier disagreed", result.SDMChainBError)
}
