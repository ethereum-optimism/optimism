// Command eth-liquidity-pause inspects and drives the ETHLiquidity pause: the L2 mirror of an
// L1 SuperchainConfig pause that stops SuperchainETHBridge sends, relays and refunds.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	op_service "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/log/logcli"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

const prefix = "ETH_LIQUIDITY_PAUSE"

// defaultFrom is the sender of the status sendETH eth_call. It needs no funds: the call
// overrides its balance.
var defaultFrom = common.HexToAddress("0x000000000000000000000000000000000000bEEF")

var (
	ConfigFile = &cli.StringFlag{
		Name:    "config",
		Usage:   "Path to a TOML config file supplying flag values. CLI flags and env vars override it.",
		EnvVars: op_service.PrefixEnvVar(prefix, "CONFIG"),
	}
	L1RPC = &cli.StringFlag{
		Name:    "l1",
		Usage:   "L1 execution RPC endpoint",
		EnvVars: op_service.PrefixEnvVar(prefix, "L1"),
	}
	L2RPCs = &cli.StringSliceFlag{
		Name:    "l2",
		Usage:   "L2 execution RPC endpoint (repeat once per chain)",
		EnvVars: op_service.PrefixEnvVar(prefix, "L2"),
	}
	L1Messengers = &cli.StringSliceFlag{
		Name:    "l1-messenger",
		Usage:   "L1CrossDomainMessenger proxy of each chain, in the same order as --l2",
		EnvVars: op_service.PrefixEnvVar(prefix, "L1_MESSENGER"),
	}
	From = &cli.StringFlag{
		Name:    "from",
		Usage:   "Sender of the sendETH eth_call (its balance is overridden, so it needs no funds)",
		EnvVars: op_service.PrefixEnvVar(prefix, "FROM"),
		Value:   defaultFrom.Hex(),
	}
	Destination = &cli.Uint64Flag{
		Name:    "destination",
		Usage:   "Destination chain ID for the sendETH eth_call (default: the chain's own ID + 1)",
		EnvVars: op_service.PrefixEnvVar(prefix, "DESTINATION"),
	}
	Expect = &cli.StringFlag{
		Name:    "expect",
		Usage:   "Exit non-zero unless every chain is 'paused' or 'unpaused'",
		EnvVars: op_service.PrefixEnvVar(prefix, "EXPECT"),
	}
	JSON = &cli.BoolFlag{
		Name:    "json",
		Usage:   "Print the result as JSON",
		EnvVars: op_service.PrefixEnvVar(prefix, "JSON"),
	}
	GuardianKey = &cli.StringFlag{
		Name:    "guardian-key",
		Usage:   "Private key (hex) of the devnet SuperchainConfig guardian",
		EnvVars: op_service.PrefixEnvVar(prefix, "GUARDIAN_KEY"),
	}
	Timeout = &cli.DurationFlag{
		Name:    "timeout",
		Usage:   "Maximum wait for each L2 to apply its sync's message, after the L1 receipt (the txmgr flags limit the L1 wait)",
		EnvVars: op_service.PrefixEnvVar(prefix, "TIMEOUT"),
		Value:   5 * time.Minute,
	}
	Force = &cli.BoolFlag{
		Name:    "force",
		Usage:   "Send the sync even for chains whose L2 already matches L1",
		EnvVars: op_service.PrefixEnvVar(prefix, "FORCE"),
	}
)

func chainFlags() []cli.Flag {
	return []cli.Flag{
		ConfigFile,
		altsrc.NewStringFlag(L1RPC),
		altsrc.NewStringSliceFlag(L2RPCs),
		altsrc.NewStringSliceFlag(L1Messengers),
	}
}

func statusFlags() []cli.Flag {
	return append(append(chainFlags(),
		altsrc.NewStringFlag(From),
		altsrc.NewUint64Flag(Destination),
		altsrc.NewStringFlag(Expect),
		altsrc.NewBoolFlag(JSON),
	), logcli.CLIFlags(prefix)...)
}

// txmgrFlags are the txmgr flags, which also select the L1 sender (--private-key, --mnemonic or
// --signer.*). --l1 sets the txmgr L1 endpoint.
func txmgrFlags() []cli.Flag {
	return tomlFlags(txmgr.CLIFlagsWithDefaults(prefix, txmgr.DefaultChallengerFlagValues))
}

// tomlFlags lets the config file set flags that other packages define. A flag type without an
// altsrc wrapper stays CLI and env only.
func tomlFlags(flags []cli.Flag) []cli.Flag {
	out := make([]cli.Flag, len(flags))
	for i, f := range flags {
		switch f := f.(type) {
		case *cli.StringFlag:
			out[i] = altsrc.NewStringFlag(f)
		case *cli.StringSliceFlag:
			out[i] = altsrc.NewStringSliceFlag(f)
		case *cli.BoolFlag:
			out[i] = altsrc.NewBoolFlag(f)
		case *cli.IntFlag:
			out[i] = altsrc.NewIntFlag(f)
		case *cli.Uint64Flag:
			out[i] = altsrc.NewUint64Flag(f)
		case *cli.Float64Flag:
			out[i] = altsrc.NewFloat64Flag(f)
		case *cli.DurationFlag:
			out[i] = altsrc.NewDurationFlag(f)
		case *cli.GenericFlag:
			out[i] = altsrc.NewGenericFlag(f)
		default:
			out[i] = f
		}
	}
	return out
}

func syncFlags() []cli.Flag {
	return append(append(append(chainFlags(),
		altsrc.NewDurationFlag(Timeout),
		altsrc.NewBoolFlag(Force),
	), txmgrFlags()...), logcli.CLIFlags(prefix)...)
}

func devnetFlags() []cli.Flag {
	return append(append(append(chainFlags(),
		altsrc.NewStringFlag(GuardianKey),
		altsrc.NewDurationFlag(Timeout),
	), txmgrFlags()...), logcli.CLIFlags(prefix)...)
}

// setup reads the logging config and wires interrupt cancellation onto the context.
func setup(c *cli.Context, logOut io.Writer) (log.Logger, context.Context) {
	logger := logcli.NewLogger(logOut, logcli.ReadCLIConfig(c))
	c.Context = ctxinterrupt.WithCancelOnInterrupt(c.Context)
	return logger, c.Context
}

func newApp() *cli.App {
	app := cli.NewApp()
	app.Name = "eth-liquidity-pause"
	app.Usage = "Inspect and sync the L2 ETHLiquidity pause that mirrors the L1 SuperchainConfig pause."
	app.Description = "status reads both sides on any chain; sync mirrors L1 to L2 (permissionless); " +
		"devnet-pause/devnet-unpause drive the guardian pause on devnets."
	app.Action = func(c *cli.Context) error {
		return errors.New("see sub-commands")
	}
	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr

	tomlSource := altsrc.NewTomlSourceFromFlagFunc(ConfigFile.Name)
	command := func(name, usage string, flags []cli.Flag, action cli.ActionFunc) *cli.Command {
		flags = cliapp.ProtectFlags(flags)
		return &cli.Command{
			Name:   name,
			Usage:  usage,
			Flags:  flags,
			Before: altsrc.InitInputSourceWithContext(flags, tomlSource),
			Action: action,
		}
	}
	app.Commands = []*cli.Command{
		command("status",
			"Show each chain's L2 ETHLiquidity.paused(), whether SuperchainETHBridge.sendETH reverts, and (with --l1) the L1 pause.",
			statusFlags(), statusAction),
		command("sync",
			"Call the permissionless L1CrossDomainMessenger."+syncSignature+" for each chain and wait for L2 to match L1.",
			syncFlags(), syncAction),
		command("devnet-pause",
			"Devnets only: pause each chain's ETHLockbox identifier as the SuperchainConfig guardian, then sync.",
			devnetFlags(), func(c *cli.Context) error { return devnetAction(c, true) }),
		command("devnet-unpause",
			"Devnets only: unpause each chain's ETHLockbox identifier as the SuperchainConfig guardian, then sync.",
			devnetFlags(), func(c *cli.Context) error { return devnetAction(c, false) }),
	}
	return app
}

func main() {
	if err := newApp().Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Application failed: %v\n", err)
		os.Exit(1)
	}
}
