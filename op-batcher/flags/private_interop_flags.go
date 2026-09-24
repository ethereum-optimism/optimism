package flags

import (
	"time"

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
// The group is enabled by --private-interop.genesis: a batcher given a private-chain genesis is a
// private-interop batcher, and one without the flag is a stock batcher. There is no marker in the
// rollup config; private behaviour is explicit runtime configuration.

const (
	// DefaultPrivateInteropMaxBlocksPerRange is the ratified cadence: ~300 blocks at 2 s is one span
	// batch every ten minutes.
	DefaultPrivateInteropMaxBlocksPerRange = 300
	// DefaultPrivateInteropMaxRangeBytes leaves ample room beneath the six-blob capacity of one L1
	// transaction even when attacker-controlled payloads do not compress.
	DefaultPrivateInteropMaxRangeBytes = 512 * 1024
	// DefaultPrivateInteropSP1Prover is the producer's prover for sp1-private-projection-v1.
	DefaultPrivateInteropSP1Prover = "network"
	// DefaultPrivateInteropProofTimeout and DefaultPrivateInteropProofTimeoutPerBlock give the
	// producer timeout base + perBlock × (lastBlock − anchorBlock), so a span that must also
	// prove a long recovery interval gets proportionally longer.
	DefaultPrivateInteropProofTimeout         = 2 * time.Minute
	DefaultPrivateInteropProofTimeoutPerBlock = 100 * time.Millisecond
)

var (
	PrivateInteropGenesisFlag = &cli.StringFlag{
		Name: "private-interop.genesis",
		Usage: "Path or http(s) URL of the private-chain genesis. The public-projection genesis and rollup " +
			"config are derived from this artifact and the private rollup config loaded from --rollup-rpc.",
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
	PrivateInteropPublicProjectionRollupRPCFlag = &cli.StringFlag{
		Name:    "private-interop.public-projection-rollup-rpc",
		Usage:   "Rollup RPC of the public projection, used to skip already derived positions after an outage.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_PUBLIC_PROJECTION_ROLLUP_RPC"),
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
	PrivateInteropExtraEmittersFlag = &cli.StringSliceFlag{
		Name: "private-interop.extra-emitters",
		Usage: "Additional application log emitters whose private-chain logs are replayed onto the " +
			"public projection, as hex addresses. The two standard interop predeploys are implicit. " +
			"Consensus-relevant: every interop filter for the chain must be given the same set.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_EXTRA_EMITTERS"),
	}
	PrivateInteropRollupConfigHashFlag = &cli.StringFlag{
		Name: "private-interop.rollup-config-hash",
		Usage: "Optional cross-check of the claim's rollupConfigHash. The batcher always derives it as " +
			"the canonical ConfigHash of the DEPLOYED projection rollup config; if set, startup fails unless " +
			"this value equals it.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_ROLLUP_CONFIG_HASH"),
	}
	PrivateInteropDepSetHashFlag = &cli.StringFlag{
		Name: "private-interop.dep-set-hash",
		Usage: "Optional cross-check of the claim's depSetHash. The batcher always uses the deployed " +
			"private_projection.dependency_set_hash, and checks it against the dependency set the rollup " +
			"node serves; if set, startup fails unless this value equals it.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_DEP_SET_HASH"),
	}
	PrivateInteropPrivateRollupConfigFlag = &cli.PathFlag{
		Name: "private-interop.private-rollup-config",
		Usage: "Path to the private chain's deployed rollup.json, byte for byte the artifact the " +
			"projection's private_config_hash covers. Required for sp1-private-projection-v1.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_PRIVATE_ROLLUP_CONFIG"),
	}
	PrivateInteropL1ChainConfigFlag = &cli.PathFlag{
		Name: "private-interop.l1-chain-config",
		Usage: "Path to the L1 chain config JSON (geth ChainConfig), byte for byte the artifact the " +
			"projection's private_config_hash covers. Required for sp1-private-projection-v1.",
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_L1_CHAIN_CONFIG"),
	}
	PrivateInteropSP1ProverFlag = &cli.StringFlag{
		Name: "private-interop.sp1-prover",
		Usage: "Prover the proof command uses for sp1-private-projection-v1: network, cpu, mock or " +
			"native-mock. mock and native-mock are refused unless the deployed projection config sets mock_proofs.",
		Value:   DefaultPrivateInteropSP1Prover,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_SP1_PROVER"),
	}
	PrivateInteropProofTimeoutFlag = &cli.DurationFlag{
		Name: "private-interop.proof-timeout",
		Usage: "Base timeout of one proof command run. Publication blocks until a proof exists; a " +
			"timed-out run is retried.",
		Value:   DefaultPrivateInteropProofTimeout,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_PROOF_TIMEOUT"),
	}
	PrivateInteropProofTimeoutPerBlockFlag = &cli.DurationFlag{
		Name: "private-interop.proof-timeout-per-block",
		Usage: "Additional proof timeout per block from the range's anchor to its last block, so " +
			"spans that must prove a recovery interval get proportionally longer.",
		Value:   DefaultPrivateInteropProofTimeoutPerBlock,
		EnvVars: prefixEnvVars("PRIVATE_INTEROP_PROOF_TIMEOUT_PER_BLOCK"),
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

var PrivateInteropProofCommandFlag = &cli.StringFlag{
	Name: "private-interop.proof-command",
	Usage: "Path to the proof producer (kona-sp1-private-projection-executor), run as " +
		"`<command> --publication-request`. Required unless the deployed projection verifier is insecure-stub-v1.",
	EnvVars: prefixEnvVars("PRIVATE_INTEROP_PROOF_COMMAND"),
}

// PrivateInteropFlags is the whole group. It is appended to the batcher's optional flags: the
// group is optional as a whole, and internally all-or-nothing (Check).
var PrivateInteropFlags = []cli.Flag{
	PrivateInteropProofCommandFlag,
	PrivateInteropGenesisFlag,
	PrivateInteropPublicProjectionRPCFlag,
	PrivateInteropPublicProjectionRollupRPCFlag,
	PrivateInteropMaxBlocksPerRangeFlag,
	PrivateInteropMaxRangeBytesFlag,
	PrivateInteropExtraEmittersFlag,
	PrivateInteropRollupConfigHashFlag,
	PrivateInteropDepSetHashFlag,
	PrivateInteropGasLimitExportFlag,
	PrivateInteropGasLimitImportFlag,
	PrivateInteropGasLimitEventFlag,
	PrivateInteropGasLimitClaimFlag,
	PrivateInteropPrivateRollupConfigFlag,
	PrivateInteropL1ChainConfigFlag,
	PrivateInteropSP1ProverFlag,
	PrivateInteropProofTimeoutFlag,
	PrivateInteropProofTimeoutPerBlockFlag,
}
