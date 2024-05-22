package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const (
	ListenAddrFlagName = "addr"
	PortFlagName       = "port"
	AvailRPCUrl        = "avail.rpc"
	Seed               = "avail.seed"
	AppID              = "avail.appid"
	Timeout            = "avail.timeout"
)

const EnvVarPrefix = "OP_PLASMA_AVAIL_DA_SERVER"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	ListenAddrFlag = &cli.StringFlag{
		Name:    ListenAddrFlagName,
		Usage:   "server listening address",
		Value:   "127.0.0.1",
		EnvVars: prefixEnvVars("ADDR"),
	}
	PortFlag = &cli.IntFlag{
		Name:    PortFlagName,
		Usage:   "server listening port",
		Value:   3100,
		EnvVars: prefixEnvVars("PORT"),
	}
	AvailRPCFlag = &cli.StringFlag{
		Name:    AvailRPCUrl,
		Usage:   "rpc url for avail node",
		EnvVars: prefixEnvVars("AVAIL_RPC"),
	}
	SeedFlag = &cli.StringFlag{
		Name:    Seed,
		Usage:   "avail seed phrase",
		EnvVars: prefixEnvVars("AVAIL_SEED"),
	}
	AppIDFlag = &cli.StringFlag{
		Name:    AppID,
		Usage:   "avail app id for the rollup",
		EnvVars: prefixEnvVars("AVAIL_APPID"),
	}
	TimeoutFlag = &cli.DurationFlag{
		Name:    Timeout,
		Usage:   "timeout parameter for request to avail",
		EnvVars: prefixEnvVars("AVAIL_TIMEOUT"),
		Value:   100 * time.Second,
	}
)

var requiredFlags = []cli.Flag{
	ListenAddrFlag,
	PortFlag,
	AvailRPCFlag,
	SeedFlag,
	AppIDFlag,
}

var optionalFlags = []cli.Flag{
	TimeoutFlag,
}

func init() {
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	Flags = append(requiredFlags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

type CLIConfig struct {
	RPC     string
	Seed    string
	AppId   int
	Timeout time.Duration
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {

	return CLIConfig{
		RPC:     ctx.String(AvailRPCUrl),
		Seed:    ctx.String(Seed),
		AppId:   ctx.Int(AppID),
		Timeout: ctx.Duration(Timeout),
	}
}

func (c CLIConfig) Check() error {
	if c.RPC == "" {
		return errors.New("no rpc url provided")
	}
	if c.AppId == 0 {
		return errors.New("no app id provided")
	}
	if c.Seed == "" {
		return errors.New("seedphrase not provided")
	}
	return nil
}

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	return nil
}
