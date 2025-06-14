package flags

import (
	"time"

	"github.com/urfave/cli/v2"

	openum "github.com/ethereum-optimism/optimism/op-service/enum"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

var (
	L1NodeAddr = &cli.StringFlag{
		Name:    "l1",
		Usage:   "Address of L1 User JSON-RPC endpoint to use (eth namespace required)",
		Value:   "http://127.0.0.1:8545",
		EnvVars: prefixEnvVars("L1_ETH_RPC"),
		// Note: originally in rollup-category, but non-breaking change.
		Category: L1RPCCategory,
	}
	BeaconAddr = &cli.StringFlag{
		Name:  "l1.beacon",
		Usage: "Address of L1 Beacon-node HTTP endpoint to use.",
		// Warning: op-node does not have a default, because of pre-ecotone logic.
		Value:   "http://127.0.0.1:9000",
		EnvVars: prefixEnvVars("L1_BEACON"),
		// Note: originally in rollup-category, but non-breaking change.
		Category: L1RPCCategory,
	}
	BeaconHeader = &cli.StringFlag{
		Name:     "l1.beacon-header",
		Usage:    "Optional HTTP header to add to all requests to the L1 Beacon endpoint. Format: 'X-Key: Value'",
		Required: false,
		EnvVars:  prefixEnvVars("L1_BEACON_HEADER"),
		Category: L1RPCCategory,
	}
	BeaconFallbackAddrs = &cli.StringSliceFlag{
		Name:     "l1.beacon-fallbacks",
		Aliases:  []string{"l1.beacon-archiver"},
		Usage:    "Addresses of L1 Beacon-API compatible HTTP fallback endpoints. Used to fetch blob sidecars not available at the l1.beacon (e.g. expired blobs).",
		EnvVars:  append(prefixEnvVars("L1_BEACON_FALLBACKS"), prefixEnvVars("L1_BEACON_ARCHIVER")...),
		Category: L1RPCCategory,
	}
	BeaconCheckIgnore = &cli.BoolFlag{
		Name:     "l1.beacon.ignore",
		Usage:    "When false, halts op-node startup if the healthcheck to the Beacon-node endpoint fails.",
		Required: false,
		Value:    false,
		EnvVars:  prefixEnvVars("L1_BEACON_IGNORE"),
		Category: L1RPCCategory,
	}
	BeaconFetchAllSidecars = &cli.BoolFlag{
		Name:     "l1.beacon.fetch-all-sidecars",
		Usage:    "If true, all sidecars are fetched and filtered locally. Workaround for buggy Beacon nodes.",
		Required: false,
		Value:    false,
		EnvVars:  prefixEnvVars("L1_BEACON_FETCH_ALL_SIDECARS"),
		Category: L1RPCCategory,
	}
	L1TrustRPC = &cli.BoolFlag{
		Name:     "l1.trustrpc",
		Usage:    "Trust the L1 RPC, sync faster at risk of malicious/buggy RPC providing bad or inconsistent L1 data",
		EnvVars:  prefixEnvVars("L1_TRUST_RPC"),
		Category: L1RPCCategory,
	}
	L1RPCProviderKind = &cli.GenericFlag{
		Name: "l1.rpckind",
		Usage: "The kind of RPC provider, used to inform optimal transactions receipts fetching, and thus reduce costs. Valid options: " +
			openum.EnumString(sources.RPCProviderKinds),
		EnvVars: prefixEnvVars("L1_RPC_KIND"),
		Value: func() *sources.RPCProviderKind {
			out := sources.RPCKindStandard
			return &out
		}(),
		Category: L1RPCCategory,
	}
	L1RPCMaxConcurrency = &cli.IntFlag{
		Name:     "l1.max-concurrency",
		Usage:    "Maximum number of concurrent RPC requests to make to the L1 RPC provider.",
		EnvVars:  prefixEnvVars("L1_MAX_CONCURRENCY"),
		Value:    10,
		Category: L1RPCCategory,
	}
	L1RPCRateLimit = &cli.Float64Flag{
		Name:     "l1.rpc-rate-limit",
		Usage:    "Optional self-imposed global rate-limit on L1 RPC requests, specified in requests / second. Disabled if set to 0.",
		EnvVars:  prefixEnvVars("L1_RPC_RATE_LIMIT"),
		Value:    0,
		Category: L1RPCCategory,
	}
	L1RPCMaxBatchSize = &cli.IntFlag{
		Name:     "l1.rpc-max-batch-size",
		Usage:    "Maximum number of RPC requests to bundle, e.g. during L1 blocks receipt fetching. The L1 RPC rate limit counts this as N items, but allows it to burst at once.",
		EnvVars:  prefixEnvVars("L1_RPC_MAX_BATCH_SIZE"),
		Value:    20,
		Category: L1RPCCategory,
	}
	L1CacheSize = &cli.UintFlag{
		Name: "l1.cache-size",
		Usage: "Cache size for blocks, receipts and transactions. " +
			"If this flag is set to 0, 2/3 of the sequencing window size is used (usually 2400). " +
			"The default value of 900 (~3h of L1 blocks) is good for (high-throughput) networks that see frequent safe head increments. " +
			"On (low-throughput) networks with infrequent safe head increments, it is recommended to set this value to 0, " +
			"or a value that well covers the typical span between safe head increments. " +
			"Note that higher values will cause significantly increased memory usage.",
		EnvVars:  prefixEnvVars("L1_CACHE_SIZE"),
		Value:    900, // ~3h of L1 blocks
		Category: L1RPCCategory,
	}
	L1HTTPPollInterval = &cli.DurationFlag{
		Name:     "l1.http-poll-interval",
		Usage:    "Polling interval for latest-block subscription when using an HTTP RPC provider. Ignored for other types of RPC endpoints.",
		EnvVars:  prefixEnvVars("L1_HTTP_POLL_INTERVAL"),
		Value:    time.Second * 12,
		Category: L1RPCCategory,
	}
	L1VerifierConfs = &cli.Uint64Flag{
		Name:     "l1.verifier-confs",
		Aliases:  []string{"verifier.l1-confs"}, // legacy op-node CLI flag name
		Usage:    "Number of L1 blocks to keep distance from the L1 head before deriving L2 data from. Reorgs are supported, but may be slow to perform.",
		EnvVars:  prefixEnvVars("VERIFIER_L1_CONFS"),
		Value:    3, // Warning: this used to default to 0.
		Category: L1RPCCategory,
	}
	L1EpochPollIntervalFlag = &cli.DurationFlag{
		Name:     "l1.epoch-poll-interval",
		Usage:    "Poll interval for retrieving new L1 epoch updates such as safe and finalized block changes. Disabled if 0 or negative.",
		EnvVars:  prefixEnvVars("L1_EPOCH_POLL_INTERVAL"),
		Value:    time.Second * 12 * 32,
		Category: L1RPCCategory,
	}
	RuntimeConfigReloadIntervalFlag = &cli.DurationFlag{
		Name:     "l1.runtime-config-reload-interval",
		Usage:    "Poll interval for reloading the runtime config, useful when config events are not being picked up. Disabled if 0 or negative.",
		EnvVars:  prefixEnvVars("L1_RUNTIME_CONFIG_RELOAD_INTERVAL"),
		Value:    time.Minute * 10,
		Category: L1RPCCategory,
	}
)

var L1RPCFlags = []cli.Flag{
	L1NodeAddr,
	BeaconAddr,
	BeaconHeader,
	BeaconFallbackAddrs,
	BeaconCheckIgnore,
	BeaconFetchAllSidecars,
	L1TrustRPC,
	L1RPCProviderKind,
	L1RPCMaxConcurrency,
	L1RPCRateLimit,
	L1RPCMaxBatchSize,
	L1CacheSize,
	L1HTTPPollInterval,
	L1VerifierConfs,
	L1EpochPollIntervalFlag,
	RuntimeConfigReloadIntervalFlag,
}
