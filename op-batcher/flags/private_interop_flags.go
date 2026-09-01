package flags

import (
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-private-interop/render"
)

// Private Interop's flag group.
//
// The ratified operator topology (op-private-interop/docs/DESIGN.md, "Operator topology") has no
// new batching binary: component 2 is "op-batcher with --private-interop flags". So everything
// the terminal seam needs is a flag on the stock service, and the service refuses to start rather
// than accept a value it cannot act on — see PrivateInteropCLIConfig.Check.
//
// The group is used only when the rollup config declares private interop. Nothing here changes a
// stock batcher's behaviour by itself.

const (
	// DefaultPrivateInteropMaxBlocksPerRange is the ratified cadence: ~300 blocks at 2 s is one span
	// batch every ten minutes.
	DefaultPrivateInteropMaxBlocksPerRange = 300
	// DefaultPrivateInteropMaxRangeBytes leaves ample room beneath the six-blob capacity of one L1
	// transaction even when attacker-controlled payloads do not compress.
	DefaultPrivateInteropMaxRangeBytes = 512 * 1024
)

var (
	PrivateInteropGenesisFlag = &cli.PathFlag{
		Name: "private-interop.genesis",
		Usage: "Path to the private-chain genesis. The public-projection genesis and rollup config " +
			"are derived from this local artifact and the private rollup config loaded from --rollup-rpc.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_GENESIS"),
	}
	PrivateInteropPublicProjectionRPCFlag = &cli.StringFlag{
		Name: "private-interop.public-projection-rpc",
		Usage: "HTTP provider URL for an execution client following the public projection. It is the " +
			"parent-check follower: the previous range's terminal public-projection block hash and the standard " +
			"batcher account's nonce " +
			"come from it, and none of them can be computed.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_PUBLIC_PROJECTION_RPC"),
	}
	PrivateInteropMaxBlocksPerRangeFlag = &cli.Uint64Flag{
		Name: "private-interop.max-blocks-per-range",
		Usage: "Maximum cadence: how many private blocks one range covers. A range may close sooner " +
			"when its uncompressed byte budget is reached.",
		Value:   DefaultPrivateInteropMaxBlocksPerRange,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_MAX_BLOCKS_PER_RANGE"),
	}
	PrivateInteropMaxRangeBytesFlag = &cli.Uint64Flag{
		Name: "private-interop.max-range-bytes",
		Usage: "Maximum estimated uncompressed public-projection transaction bytes in one range. The range " +
			"closes early when this budget is reached; the builder separately refuses output requiring " +
			"more than one six-blob L1 transaction.",
		Value:   DefaultPrivateInteropMaxRangeBytes,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_MAX_RANGE_BYTES"),
	}
	PrivateInteropRollupConfigHashFlag = &cli.StringFlag{
		Name: "private-interop.rollup-config-hash",
		Usage: "32-byte rollupConfigHash the range claim commits to: which chain the claim speaks " +
			"for. A configuration value, identical for the chain's life.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_ROLLUP_CONFIG_HASH"),
	}
	PrivateInteropDepSetHashFlag = &cli.StringFlag{
		Name: "private-interop.dep-set-hash",
		Usage: "32-byte depSetHash the range claim commits to: which dependency set the claim " +
			"speaks for. A configuration value, changing only with the dependency set.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_DEP_SET_HASH"),
	}
	PrivateInteropGasLimitExportFlag = &cli.Uint64Flag{
		Name:    "private-interop.gas-limit-export",
		Usage:   "Gas limit for an export replay transaction. UNMEASURED: the default is a generous guess, pending measurement against deployed replay contracts.",
		Value:   render.DefaultGasPolicy().GasLimitExport,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_GAS_LIMIT_EXPORT"),
	}
	PrivateInteropGasLimitImportFlag = &cli.Uint64Flag{
		Name:    "private-interop.gas-limit-import",
		Usage:   "Gas limit for an import (CrossL2Inbox.validateMessage) replay transaction. UNMEASURED: a generous guess.",
		Value:   render.DefaultGasPolicy().GasLimitImport,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_GAS_LIMIT_IMPORT"),
	}
	PrivateInteropGasLimitEventFlag = &cli.Uint64Flag{
		Name:    "private-interop.gas-limit-event",
		Usage:   "Gas limit for a generic log re-emission through EventReplayer. UNMEASURED: a generous guess.",
		Value:   render.DefaultGasPolicy().GasLimitEvent,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_GAS_LIMIT_EVENT"),
	}
	PrivateInteropGasLimitClaimFlag = &cli.Uint64Flag{
		Name:    "private-interop.gas-limit-claim",
		Usage:   "Gas limit for the range's leading claim transaction. UNMEASURED: a generous guess.",
		Value:   render.DefaultGasPolicy().GasLimitClaim,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_GAS_LIMIT_CLAIM"),
	}
)

// PrivateInteropFlags is the whole group. It is appended to the batcher's optional flags: the
// group is optional as a whole, and internally all-or-nothing (Check).
var PrivateInteropFlags = []cli.Flag{
	PrivateInteropGenesisFlag,
	PrivateInteropPublicProjectionRPCFlag,
	PrivateInteropMaxBlocksPerRangeFlag,
	PrivateInteropMaxRangeBytesFlag,
	PrivateInteropRollupConfigHashFlag,
	PrivateInteropDepSetHashFlag,
	PrivateInteropGasLimitExportFlag,
	PrivateInteropGasLimitImportFlag,
	PrivateInteropGasLimitEventFlag,
	PrivateInteropGasLimitClaimFlag,
}
