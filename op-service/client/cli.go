package client

import (
	"fmt"
	"net/http"
	"strings"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli"
)

type FlagNames struct {
	RPC       string
	LegacyRPC string
	Cookies   string
	Headers   string
}

var L1FlagNames = FlagNames{
	RPC:       "l1",
	LegacyRPC: "l1-eth-rpc",
	Cookies:   "l1.cookies",
	Headers:   "l1.headers",
}

var L2FlagNames = FlagNames{
	RPC:       "l2",
	LegacyRPC: "l2-eth-rpc",
	Cookies:   "l2.cookies",
	Headers:   "l2.headers",
}

func L1CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   L1FlagNames.RPC,
			Usage:  "HTTP provider URL for L1",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L1_ETH_RPC"),
		},
		// for compatibility with old batcher / proposer flag
		cli.StringFlag{
			Name:   L1FlagNames.LegacyRPC,
			Usage:  fmt.Sprintf("(deprecated flag, use --%s instead)", L1FlagNames.RPC),
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L1_ETH_RPC_LEGACY"),
		},
		cli.BoolFlag{
			Name:   L1FlagNames.Cookies,
			Usage:  "Enable cookie support on the L1 HTTP provider.",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L1_ETH_RPC_COOKIES"),
		},
		cli.StringSliceFlag{
			Name:   L1FlagNames.Headers,
			Usage:  "Custom headers to pass to the L1 HTTP provider.",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L1_ETH_RPC_HEADERS"),
		},
	}
}

func L2CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   L2FlagNames.RPC,
			Usage:  "HTTP provider URL for L2",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L2_ENGINE_RPC"),
		},
		// for compatibility with old batcher / proposer flag
		cli.StringFlag{
			Name:   L2FlagNames.LegacyRPC,
			Usage:  fmt.Sprintf("(deprecated flag, use --%s instead)", L2FlagNames.RPC),
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L2_ETH_RPC"),
		},
		cli.BoolFlag{
			Name:   L2FlagNames.Cookies,
			Usage:  "Enable cookie support on the L2 HTTP provider.",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L2_ETH_RPC_COOKIES"),
		},
		cli.StringSliceFlag{
			Name:   L2FlagNames.Headers,
			Usage:  "Custom headers to pass to the L2 HTTP provider.",
			EnvVar: opservice.PrefixEnvVar(envPrefix, "L2_ETH_RPC_HEADERS"),
		},
	}
}

type CLIConfig struct {
	Addr    string
	Cookies bool
	Headers http.Header
}

func (cfg CLIConfig) Check() error {
	if cfg.Addr == "" {
		return fmt.Errorf("missing flag: %s or %s", L1FlagNames.RPC, L2FlagNames.RPC)
	}
	return nil
}

func ReadCLIConfig(ctx *cli.Context, names FlagNames) CLIConfig {
	nodeAddr := ctx.GlobalString(names.RPC)
	if nodeAddr == "" {
		nodeAddr = ctx.GlobalString(names.LegacyRPC)
	}
	return CLIConfig{
		Addr:    nodeAddr,
		Cookies: ctx.GlobalBool(names.Cookies),
		Headers: ParseHttpHeader(ctx.GlobalStringSlice(names.Headers)),
	}
}

func ReadL1CLIConfig(ctx *cli.Context) CLIConfig {
	return ReadCLIConfig(ctx, L1FlagNames)
}

func ReadL2CLIConfig(ctx *cli.Context) CLIConfig {
	return ReadCLIConfig(ctx, L2FlagNames)
}

// ParseHttpHeader takes a slice of strings of the form "K=V" and returns a http.Header
func ParseHttpHeader(slice []string) http.Header {
	if len(slice) == 0 {
		return nil
	}
	header := make(http.Header)
	for _, s := range slice {
		split := strings.SplitN(s, "=", 2)
		val := ""
		if len(split) >= 2 {
			val = split[1]
		}
		header.Add(split[0], val)
	}
	return header
}
