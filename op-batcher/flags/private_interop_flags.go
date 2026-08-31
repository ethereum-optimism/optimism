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
			"parent-check follower: the previous range's terminal rendering block hash, the origin " +
			"and sequence-number bookkeeping to continue from, and the operator EOA's nonce all " +
			"come from it, and none of them can be computed.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_RENDERING_RPC"),
	}
	PrivateInteropMaxBlocksPerRangeFlag = &cli.Uint64Flag{
		Name: "private-interop.max-blocks-per-range",
		Usage: "The cadence: how many private blocks one range covers. It is the cadence, not a " +
			"size limit, that ends a range — every measured size limit has three orders of " +
			"magnitude of headroom at the default.",
		Value:   DefaultPrivateInteropMaxBlocksPerRange,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_MAX_BLOCKS_PER_RANGE"),
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
	PrivateInteropExtraEmittersFlag = &cli.StringSliceFlag{
		Name: "private-interop.extra-emitters",
		Usage: "Comma-separated extra emitter addresses, whose logs render at ANY topic. This is a " +
			"genesis-time configuration and must match the rendering's genesis exactly: it changes " +
			"which private logs are public, hence every later log's rendered index.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_EXTRA_EMITTERS"),
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
	PrivateInteropOperatorKeyFlag = &cli.StringFlag{
		Name: "private-interop.operator-key",
		Usage: "Hex private key of the operator EOA that signs the rendering's replay and claim " +
			"transactions. It is a LOCAL key on purpose: the rendering's transactions are consensus " +
			"data and must be a pure function of the range, which go-ethereum's RFC 6979 signing is " +
			"and a remote signer that adds entropy is not. Prefer the env var.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_OPERATOR_KEY"),
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
	PrivateInteropGasFeeCapFlag = &cli.Uint64Flag{
		Name: "private-interop.gas-fee-cap",
		Usage: "EIP-1559 fee cap, in wei, for every rendering transaction. Frozen configuration, not " +
			"an observed price: the rendering has no mempool, no fee market and one sender, and a " +
			"builder that priced from an oracle would produce a different chain on every run.",
		Value:   render.DefaultGasPolicy().GasFeeCap.Uint64(),
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_GAS_FEE_CAP"),
	}
	PrivateInteropGasTipCapFlag = &cli.Uint64Flag{
		Name:    "private-interop.gas-tip-cap",
		Usage:   "EIP-1559 tip cap, in wei, for every rendering transaction. Frozen configuration; see --private-interop.gas-fee-cap.",
		Value:   render.DefaultGasPolicy().GasTipCap.Uint64(),
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_GAS_TIP_CAP"),
	}
)

// PrivateInteropFlags is the whole group. It is appended to the batcher's optional flags: the
// group is optional as a whole, and internally all-or-nothing (Check).
var PrivateInteropFlags = []cli.Flag{
	PrivateInteropEnabledFlag,
	PrivateInteropRenderingRollupConfigFlag,
	PrivateInteropRenderingRPCFlag,
	PrivateInteropMaxBlocksPerRangeFlag,
	PrivateInteropClaimRegistryFlag,
	PrivateInteropEventReplayerFlag,
	PrivateInteropReplayMessengerFlag,
	PrivateInteropExtraEmittersFlag,
	PrivateInteropRollupConfigHashFlag,
	PrivateInteropDepSetHashFlag,
	PrivateInteropOperatorKeyFlag,
	PrivateInteropGasLimitExportFlag,
	PrivateInteropGasLimitImportFlag,
	PrivateInteropGasLimitEventFlag,
	PrivateInteropGasLimitClaimFlag,
	PrivateInteropGasFeeCapFlag,
	PrivateInteropGasTipCapFlag,
}
