package flags

import (
	"fmt"

	"github.com/urfave/cli/v2"

	opnodeflags "github.com/ethereum-optimism/optimism/op-node/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

const EnvVarPrefix = "OP_SUPERNODE"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	ChainsFlag = &cli.Uint64SliceFlag{
		Name:    "chains",
		Usage:   "List of chain IDs to run (repeatable or comma-separated)",
		EnvVars: prefixEnvVars("CHAINS"),
		Value:   cli.NewUint64Slice(),
	}
	DataDirFlag = &cli.StringFlag{
		Name:     "data-dir",
		Usage:    "Data directory for op-supernode",
		EnvVars:  prefixEnvVars("DATA_DIR"),
		Value:    "./datadir",
		Required: false,
	}
	L1NodeAddr = &cli.StringFlag{
		Name:     "l1",
		Usage:    "Address of L1 User JSON-RPC endpoint to use (eth namespace required)",
		EnvVars:  prefixEnvVars("L1_ETH_RPC"),
		Required: true,
	}
	L1BeaconAddr = &cli.StringFlag{
		Name:     "l1.beacon",
		Usage:    "Address of L1 Beacon-node HTTP endpoint to use",
		EnvVars:  prefixEnvVars("L1_BEACON"),
		Required: false,
	}
)

var requiredFlags = []cli.Flag{
	ChainsFlag,
	L1NodeAddr,
}

var optionalFlags []cli.Flag

func init() {
	optionalFlags = append(optionalFlags, L1BeaconAddr)
	optionalFlags = append(optionalFlags, DataDirFlag)
	optionalFlags = append(optionalFlags, oprpc.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(EnvVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

var Flags []cli.Flag

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	return nil
}

// FullDynamicFlags returns the base supernode flags plus dynamically-generated
// vn.* flags cloned from op-node flags for all-chains and per-chain IDs.
func FullDynamicFlags(chains []uint64) []cli.Flag {
	// start with the base supernode flags
	final := make([]cli.Flag, 0, len(Flags)+len(opnodeflags.Flags)*(1+len(chains)))
	final = append(final, Flags...)

	// for each op-node flag, add vn.all.<name> and vn.<id>.<name> variants
	for _, f := range opnodeflags.Flags {
		baseName := f.Names()[0]
		// vn.all.* env var/alias prefixing
		allEnvs := prefixEnvVar(f, "VN_ALL_")
		allAliases := prefixAliases(f, VNFlagGlobalPrefix)
		final = append(final, renameFlagWithEnv(f, VNFlagGlobalPrefix+baseName, allEnvs, allAliases))
		// per-chain
		for _, id := range chains {
			perChainEnvs := prefixEnvVar(f, fmt.Sprintf("VN_%d_", id))
			perAliases := prefixAliases(f, fmt.Sprintf("%s%d.", VNFlagNamePrefix, id))
			final = append(final, renameFlagWithEnv(f, fmt.Sprintf("%s%d.%s", VNFlagNamePrefix, id, baseName), perChainEnvs, perAliases))
		}
	}
	return final
}
