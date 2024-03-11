package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

const (
	ListenAddrFlagName  = "addr"
	PortFlagName        = "port"
	S3BucketFlagName    = "s3.bucket"
	LevelDBPathFlagName = "leveldb.path"
)

const EnvVarPrefix = "OP_PLASMA_DA_SERVER"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	ListenAddrFlag = &cli.StringFlag{
		Name:    ListenAddrFlagName,
		Usage:   "rpc listening address",
		Value:   "0.0.0.0",
		EnvVars: prefixEnvVars("ADDR"),
	}
	PortFlag = &cli.IntFlag{
		Name:    PortFlagName,
		Usage:   "rpc listening port",
		Value:   3100,
		EnvVars: prefixEnvVars("PORT"),
	}
	LevelDBPathFlag = &cli.StringFlag{
		Name:    LevelDBPathFlagName,
		Usage:   "path to LevelDB storage",
		Value:   "",
		EnvVars: prefixEnvVars("LEVELDB_PATH"),
	}
	S3BucketFlag = &cli.StringFlag{
		Name:    S3BucketFlagName,
		Usage:   "bucket name for S3 storage",
		Value:   "",
		EnvVars: prefixEnvVars("S3_BUCKET"),
	}
)

var requiredFlags = []cli.Flag{
	ListenAddrFlag,
	PortFlag,
}

var optionalFlags = []cli.Flag{
	LevelDBPathFlag,
	S3BucketFlag,
}

func init() {
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

type CLIConfig struct {
	LevelDBPath string
	S3Bucket    string
}

func ReadCLIConfig(ctx *cli.Context) CLIConfig {
	return CLIConfig{
		LevelDBPath: ctx.String(LevelDBPathFlagName),
		S3Bucket:    ctx.String(S3BucketFlagName),
	}
}

func (c CLIConfig) Check() error {
	if !c.S3Enabled() && !c.LevelDBEnabled() {
		return fmt.Errorf("at least one storage backend must be enabled")
	}
	if c.S3Enabled() && c.LevelDBEnabled() {
		return fmt.Errorf("only one storage backend can be enabled")
	}
	return nil
}

func (c CLIConfig) S3Enabled() bool {
	return c.S3Bucket != ""
}

func (c CLIConfig) LevelDBEnabled() bool {
	return c.LevelDBPath != ""
}

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	return nil
}
