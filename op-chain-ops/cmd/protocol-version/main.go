package main

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const EnvPrefix = "OP_CHAIN_OPS_PROTOCOL_VERSION"

var (
	MajorFlag = &cli.Uint64Flag{
		Name:    "major",
		Value:   0,
		Usage:   "major version component",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "MAJOR"),
	}
	MinorFlag = &cli.Uint64Flag{
		Name:    "minor",
		Value:   0,
		Usage:   "minor version component",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "MINOR"),
	}
	PatchFlag = &cli.Uint64Flag{
		Name:    "patch",
		Value:   0,
		Usage:   "patch version component",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "PATCH"),
	}
	PrereleaseFlag = &cli.Uint64Flag{
		Name:    "prerelease",
		Value:   0,
		Usage:   "prerelease version component",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "PRERELEASE"),
	}
	BuildFlag = &cli.StringFlag{
		Name:    "build",
		Value:   "0000000000000000",
		Usage:   "build version component as 8-byte hex string without 0x prefix",
		EnvVars: opservice.PrefixEnvVar(EnvPrefix, "BUILD"),
	}
)

func main() {
	color := isatty.IsTerminal(os.Stderr.Fd())
	oplog.SetGlobalLogHandler(log.NewTerminalHandlerWithLevel(os.Stdout, slog.LevelDebug, color))

	app := &cli.App{
		Name:   "protocol-version",
		Usage:  "Util to interact with protocol-version data",
		Flags:  []cli.Flag{},
		Writer: os.Stdout,
	}
	app.Commands = []*cli.Command{
		{
			Name:  "encode",
			Usage: "Encode a protocol version (type 0) to its onchain bytes32 form.",
			Flags: []cli.Flag{
				MajorFlag, MinorFlag, PatchFlag, PrereleaseFlag, BuildFlag,
			},
			Action: encodeProtocolVersion,
		},
		{
			Name:      "decode",
			Usage:     "Decode a protocol version from its onchain bytes32 form (incl. 0x prefix).",
			ArgsUsage: "<bytes32>",
			Action:    decodeProtocolVersion,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Crit("critical error", "err", err)
	}
}

func encodeProtocolVersion(ctx *cli.Context) error {
	u32Flag := func(name string) (uint32, error) {
		v := ctx.Uint64(name)
		if v >= uint64(1<<32) {
			return 0, fmt.Errorf("value of flag %q must be valid uint32, %d (0x%x)", name, v, v)
		}
		return uint32(v), nil
	}
	major, err := u32Flag(MajorFlag.Name)
	if err != nil {
		return err
	}
	minor, err := u32Flag(MinorFlag.Name)
	if err != nil {
		return err
	}
	patch, err := u32Flag(PatchFlag.Name)
	if err != nil {
		return err
	}
	prerelease, err := u32Flag(PrereleaseFlag.Name)
	if err != nil {
		return err
	}
	buildVal := ctx.String(BuildFlag.Name)
	if len(buildVal) != 16 {
		return fmt.Errorf("build flag value must be 16 characters")
	}
	var build [8]byte
	if _, err := hex.Decode(build[:], []byte(buildVal)); err != nil {
		return fmt.Errorf("failed to decode build flag: %q", buildVal)
	}

	version := params.ProtocolVersionV0{
		Build:      build,
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		PreRelease: prerelease,
	}.Encode()
	_, err = fmt.Fprintln(ctx.App.Writer, common.Hash(version).String())
	return err
}

func decodeProtocolVersion(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return fmt.Errorf("expected 1 argument: protocol version")
	}
	hexVersion := ctx.Args().Get(0)
	var v params.ProtocolVersion
	if err := v.UnmarshalText([]byte(hexVersion)); err != nil {
		return fmt.Errorf("failed to decode protocol version: %w", err)
	}
	_, err := fmt.Fprintln(ctx.App.Writer, v.String())
	return err
}
