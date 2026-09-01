package flags

import (
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
)

// Private Interop's flag group.
//
// The ratified operator topology (op-private-interop/docs/DESIGN.md, "Operator topology") has no
// new batching binary: component 2 is "op-batcher with --private-interop flags". So everything
// the terminal seam needs is a flag on the stock service, and the service refuses to start rather
// than accept a value it cannot act on — see PrivateInteropCLIConfig.Check.
//
// The group is inert unless --private-interop.enabled is set. Nothing here changes a stock
// batcher's behaviour by existing.

const (
	// DefaultPrivateInteropMaxBlocksPerRange is the ratified cadence: ~300 blocks at 2 s is one span
	// batch every ten minutes.
	DefaultPrivateInteropMaxBlocksPerRange = 300
	// DefaultPrivateInteropMaxRangeBytes leaves ample room beneath the six-blob capacity of one L1
	// transaction even when attacker-controlled payloads do not compress.
	DefaultPrivateInteropMaxRangeBytes = 512 * 1024
)

var (
	PrivateInteropEnabledFlag = &cli.BoolFlag{
		Name: "private-interop.enabled",
		Usage: "Enable Private Interop mode: the batcher loads PRIVATE blocks and posts the bytes " +
			"that describe their public RENDERING. Requires the rest of the --private-interop.* group.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_ENABLED"),
	}
	PrivateInteropRenderingRollupConfigFlag = &cli.StringFlag{
		Name: "private-interop.rendering-rollup-config",
		Usage: "Path to the RENDERING's rollup.json. This is not the private chain's config: the " +
			"span batch is encoded against the chain being described, whose genesis, chain ID and " +
			"drift are its own. The private chain's config still comes from --rollup-rpc.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_RENDERING_ROLLUP_CONFIG"),
	}
	PrivateInteropRenderingRPCFlag = &cli.StringFlag{
		Name: "private-interop.rendering-rpc",
		Usage: "HTTP provider URL for an execution client following the RENDERING. It is the " +
			"parent-check follower: the previous range's terminal rendering block hash and the standard " +
			"batcher account's nonce " +
			"come from it, and none of them can be computed.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_RENDERING_RPC"),
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
		Usage: "Maximum estimated uncompressed rendering transaction bytes in one range. The range " +
			"closes early when this budget is reached; the builder separately refuses output requiring " +
			"more than one six-blob L1 transaction.",
		Value:   DefaultPrivateInteropMaxRangeBytes,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_MAX_RANGE_BYTES"),
	}
	PrivateInteropClaimRegistryFlag = &cli.StringFlag{
		Name: "private-interop.claim-registry",
		Usage: "Address of the ClaimRegistry on the rendering. It is placed by the rendering's " +
			"genesis builder, so it is per-deployment configuration and has no default; a zero " +
			"address fails loudly rather than sending the range's claim into the void.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_CLAIM_REGISTRY"),
	}
	PrivateInteropEventReplayerFlag = &cli.StringFlag{
		Name: "private-interop.event-replayer",
		Usage: "Address of the EventReplayer on the rendering, which re-emits the logs of " +
			"genesis-configured extra emitters. Genesis-assigned, no default, zero fails loudly.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_EVENT_REPLAYER"),
	}
	PrivateInteropReplayMessengerFlag = &cli.StringFlag{
		Name: "private-interop.replay-messenger",
		Usage: "Address the export replay implementation is installed at. The design pins it to the " +
			"L2ToL2CrossDomainMessenger predeploy — a re-emitted SentMessage must carry the emitter " +
			"every stock consumer expects — so the default is the only value that renders valid " +
			"exports. It is a flag so a deployment that moved it fails at its own hands, not ours.",
		Value:   predeploys.L2toL2CrossDomainMessenger,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_REPLAY_MESSENGER"),
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
	PrivateInteropEnabledFlag,
	PrivateInteropRenderingRollupConfigFlag,
	PrivateInteropRenderingRPCFlag,
	PrivateInteropMaxBlocksPerRangeFlag,
	PrivateInteropMaxRangeBytesFlag,
	PrivateInteropClaimRegistryFlag,
	PrivateInteropEventReplayerFlag,
	PrivateInteropReplayMessengerFlag,
	PrivateInteropRollupConfigHashFlag,
	PrivateInteropDepSetHashFlag,
	PrivateInteropGasLimitExportFlag,
	PrivateInteropGasLimitImportFlag,
	PrivateInteropGasLimitEventFlag,
	PrivateInteropGasLimitClaimFlag,
}
