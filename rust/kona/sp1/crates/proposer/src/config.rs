//! Environment-driven proposer configuration.
//!
//! Trimmed from op-succinct's `fault-proof/src/config.rs` (@ 13716c2c):
//! proof-provider selection, SP1 network knobs, and range splitting are
//! ported here with the defend path; fast finality is tracked in #22112;
//! cluster proving, restart recovery, and the challenger and forge-deploy
//! configs are deliberately not ported (see PR #21463 for the ledger).

use std::{
    env,
    num::{NonZeroU8, NonZeroU64, NonZeroUsize},
    path::PathBuf,
    str::FromStr,
};

use alloy_primitives::{Address, B256};
use alloy_transport_http::reqwest::{self, Url};
use anyhow::{Context, Result, anyhow, bail};
use kona_sp1_host_utils::network::parse_fulfillment_strategy;
use sp1_sdk::{SP1ProofMode, network::FulfillmentStrategy};

/// Safety level gating how far proposals may advance.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProposalSafety {
    /// Propose up to the cross-safe super-root timestamp.
    Safe,
    /// Propose up to the finalized super-root timestamp (default).
    Finalized,
}

impl FromStr for ProposalSafety {
    type Err = anyhow::Error;

    fn from_str(value: &str) -> Result<Self> {
        match value.to_ascii_lowercase().as_str() {
            "safe" => Ok(Self::Safe),
            "finalized" => Ok(Self::Finalized),
            other => bail!("invalid PROPOSAL_SAFETY: {other} (expected safe|finalized)"),
        }
    }
}

/// Which proof provider drives the defend path.
///
/// There is no default: deployments state explicitly whether they buy real
/// SP1 proofs or run the ELF-free mock pipeline.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ProofProviderKind {
    /// Real SP1 proving via the Succinct Prover Network.
    Network,
    /// Native-core execution with placeholder proof bytes. Only a deployment
    /// whose game verifier accepts arbitrary bytes (devstack's mock
    /// verifier) can resolve games proven this way.
    Mock,
}

impl FromStr for ProofProviderKind {
    type Err = anyhow::Error;

    fn from_str(value: &str) -> Result<Self> {
        match value.to_ascii_lowercase().as_str() {
            "network" => Ok(Self::Network),
            "mock" => Ok(Self::Mock),
            other => bail!("invalid PROOF_PROVIDER: {other} (expected network|mock)"),
        }
    }
}

/// Runtime configuration for the proposer, parsed from environment variables.
#[derive(Debug, Clone)]
pub struct ProposerConfig {
    /// The L1 RPC URL.
    pub l1_rpc: Url,

    /// The supernode (or single-chain op-node) RPC URL serving
    /// `superroot_atTimestamp`.
    pub supernode_rpc: Url,

    /// The address of the `DisputeGameFactory` contract.
    pub factory_address: Address,

    /// Base URL of a prestate artifact directory, accepting `file://` and
    /// `http(s)://` URIs. Each prestate is keyed by the game's
    /// `absolutePrestate()` hash (the super-aggregation program vkey, which
    /// embeds the super-range program vkey and therefore uniquely identifies
    /// both) and maps to two program ELFs, following the op-challenger
    /// `--prestates-url` convention:
    /// - `<url>/<vkey>.agg.bin.gz` (super-aggregation program)
    /// - `<url>/<vkey>.range.bin.gz` (super-range program)
    ///
    /// The create path checks artifact availability (creation pauses when
    /// either artifact is missing, since the proposer could not defend such
    /// a game); the defend path loads the ELFs from the same location and,
    /// in network mode, verifies the aggregation ELF hashes to the prestate
    /// during proving-key setup.
    pub prestates_url: Url,

    /// Minimum spacing, in seconds of super-root timestamps, between the
    /// canonical head and a new proposal.
    pub proposal_interval_seconds: u64,

    /// Safety level bounding the newest proposable timestamp.
    pub proposal_safety: ProposalSafety,

    /// The interval in seconds between sync/schedule loop iterations.
    pub fetch_interval: u64,

    /// The metrics port. Metrics are disabled when 0.
    pub metrics_port: u16,

    /// Number of L1 blocks behind `latest` to pin reads during sync cycles.
    pub sync_l1_confirmations: u64,

    /// Maximum time (in seconds) to wait for an L1 transaction to reach the
    /// required confirmations before the watcher gives up.
    pub tx_confirmation_timeout: u64,

    /// Optional cap, in wei, on the EIP-1559 max fee per gas the fee
    /// estimator may set on submitted transactions. Unset = uncapped.
    /// A cap below prevailing fees delays inclusion until fees decay; for a
    /// deadline-driven actor that can cost far more than it saves. Leave
    /// unset unless a specific need exists.
    pub max_fee_per_gas: Option<u128>,

    /// Optional cap, in wei, on the EIP-1559 max priority fee per gas the
    /// fee estimator may set. Unset = uncapped. A cap below prevailing
    /// fees delays inclusion until fees decay; for a deadline-driven actor
    /// that can cost far more than it saves. Leave unset unless a specific
    /// need exists.
    pub max_priority_fee_per_gas: Option<u128>,

    /// Which proof provider defends challenged games.
    pub proof_provider: ProofProviderKind,

    /// The L1 beacon API URL serving blob sidecars for derivation witnesses.
    pub l1_beacon_rpc: Url,

    /// L2 execution-layer RPC URLs, one per chain in the dependency set.
    /// Order is irrelevant: hosts are keyed by their reported chain id.
    pub l2_rpcs: Vec<Url>,

    /// Optional rollup config files, one per chain (comma-separated env).
    /// Absent = kona-host falls back to the superchain registry, matching
    /// the super-range executor CLI.
    pub rollup_config_paths: Option<Vec<PathBuf>>,

    /// Optional L1 chain config file. Absent = superchain-registry fallback.
    pub l1_config_path: Option<PathBuf>,

    /// Optional dependency-set config file. Absent = superchain-registry
    /// fallback. The env name matches the executor's `DEPENDENCY_SET_PATH`.
    pub dependency_set_path: Option<PathBuf>,

    /// How many chunks a defended timestamp span is partitioned into.
    pub range_split_count: RangeSplitCount,

    /// Maximum concurrent child (range/consolidation) proofs within one
    /// game's defense.
    pub max_concurrent_range_proofs: NonZeroUsize,

    /// Maximum number of games being defended concurrently. Zero is
    /// rejected at parse time: it would silently disable defense.
    pub max_concurrent_defense_tasks: NonZeroU64,

    /// SP1 proof-provider settings (timeouts, strategies, limits, prices).
    pub proof_provider_config: ProofProviderConfig,
}

fn optional_env(name: &str) -> Option<String> {
    match env::var(name) {
        Ok(value) if !value.is_empty() => Some(value),
        _ => None,
    }
}

fn parsed_env_or<T: FromStr>(name: &str, default: T) -> Result<T>
where
    T::Err: std::fmt::Display,
{
    parsed_optional_env(name).map(|value| value.unwrap_or(default))
}

fn parsed_optional_env<T: FromStr>(name: &str) -> Result<Option<T>>
where
    T::Err: std::fmt::Display,
{
    optional_env(name)
        .map(|value| value.parse::<T>().map_err(|err| anyhow!("invalid {name}: {err}")))
        .transpose()
}

impl ProposerConfig {
    /// Parses the configuration from environment variables, applying defaults
    /// for optional settings and failing on missing or invalid required ones.
    pub fn from_env() -> Result<Self> {
        let tx_confirmation_timeout = parsed_env_or("TX_CONFIRMATION_TIMEOUT", 60u64)?;
        anyhow::ensure!(
            tx_confirmation_timeout > 0,
            "TX_CONFIRMATION_TIMEOUT must be positive (0 would time out every transaction immediately)"
        );

        let proof_provider = env::var("PROOF_PROVIDER")
            .context("PROOF_PROVIDER not set (expected network|mock; there is no default)")?
            .parse::<ProofProviderKind>()?;
        let l2_rpcs = parse_url_list(&env::var("L2_RPCS").context("L2_RPCS not set")?)
            .context("invalid L2_RPCS")?;
        Ok(Self {
            l1_rpc: env::var("L1_RPC")
                .context("L1_RPC not set")?
                .parse()
                .map_err(|err| anyhow!("invalid L1_RPC: {err}"))?,
            supernode_rpc: env::var("SUPERNODE_RPC")
                .context("SUPERNODE_RPC not set")?
                .parse()
                .map_err(|err| anyhow!("invalid SUPERNODE_RPC: {err}"))?,
            factory_address: env::var("FACTORY_ADDRESS")
                .context("FACTORY_ADDRESS not set")?
                .parse()
                .map_err(|err| anyhow!("invalid FACTORY_ADDRESS: {err}"))?,
            prestates_url: env::var("PRESTATES_URL")
                .context("PRESTATES_URL not set")?
                .parse()
                .map_err(|err| anyhow!("invalid PRESTATES_URL: {err}"))?,
            proposal_interval_seconds: parsed_env_or("PROPOSAL_INTERVAL_SECONDS", 3600u64)?,
            proposal_safety: parsed_env_or("PROPOSAL_SAFETY", ProposalSafety::Finalized)?,
            fetch_interval: parsed_env_or("FETCH_INTERVAL", 30u64)?,
            metrics_port: parsed_env_or("METRICS_PORT", 0u16)?,
            sync_l1_confirmations: parsed_env_or("SYNC_L1_CONFIRMATIONS", 0u64)?,
            tx_confirmation_timeout,
            max_fee_per_gas: parsed_optional_env("MAX_FEE_PER_GAS")?,
            max_priority_fee_per_gas: parsed_optional_env("MAX_PRIORITY_FEE_PER_GAS")?,
            proof_provider,
            l1_beacon_rpc: env::var("L1_BEACON_RPC")
                .context("L1_BEACON_RPC not set")?
                .parse()
                .map_err(|err| anyhow!("invalid L1_BEACON_RPC: {err}"))?,
            l2_rpcs,
            rollup_config_paths: optional_env("ROLLUP_CONFIG_PATHS")
                .map(|list| list.split(',').map(|path| PathBuf::from(path.trim())).collect()),
            l1_config_path: optional_env("L1_CONFIG_PATH").map(PathBuf::from),
            dependency_set_path: optional_env("DEPENDENCY_SET_PATH").map(PathBuf::from),
            range_split_count: parsed_env_or("RANGE_SPLIT_COUNT", RangeSplitCount::one())?,
            max_concurrent_range_proofs: parsed_env_or(
                "MAX_CONCURRENT_RANGE_PROOFS",
                NonZeroUsize::MIN,
            )?,
            max_concurrent_defense_tasks: parsed_env_or(
                "MAX_CONCURRENT_DEFENSE_TASKS",
                NonZeroU64::new(8).expect("8 is non-zero"),
            )?,
            proof_provider_config: ProofProviderConfig::from_env()?,
        })
    }
}

/// Renders a URL for logging with any userinfo stripped.
pub fn redacted_url(url: &Url) -> String {
    let mut url = url.clone();
    let _ = url.set_username("");
    let _ = url.set_password(None);
    url.to_string()
}

/// Parses a comma-separated list of URLs, requiring at least one entry.
fn parse_url_list(value: &str) -> Result<Vec<Url>> {
    let urls = value
        .split(',')
        .map(str::trim)
        .filter(|entry| !entry.is_empty())
        .map(|entry| entry.parse::<Url>().map_err(|err| anyhow!("invalid URL {entry}: {err}")))
        .collect::<Result<Vec<_>>>()?;
    anyhow::ensure!(!urls.is_empty(), "expected at least one URL");
    Ok(urls)
}

/// Default SP1 request cycle/gas limit (one trillion), matching upstream.
const DEFAULT_PROOF_LIMIT: u64 = 1_000_000_000_000;

/// SP1 proof-provider settings (timeouts, strategies, limits, prices).
///
/// Parsed unconditionally with upstream op-succinct's defaults; none of the
/// entries is required or secret, so mock deployments need to set none of
/// them. `NETWORK_PRIVATE_KEY` is deliberately NOT part of this struct: it
/// is read only when the network provider is constructed.
#[derive(Debug, Clone)]
pub struct ProofProviderConfig {
    /// Overall per-proof timeout in seconds: the server-side deadline for
    /// proof requests and the client-side maximum wait.
    pub timeout: u64,
    /// Timeout in seconds for individual network API calls (calls exceeding
    /// it are retried).
    pub network_calls_timeout: u64,
    /// Cancel requests still unassigned past `created_at + auction_timeout`
    /// seconds (mainnet auctions only).
    pub auction_timeout: u64,
    /// Fulfillment strategy for super-range proofs.
    pub range_proof_strategy: FulfillmentStrategy,
    /// Fulfillment strategy for the aggregation proof.
    pub agg_proof_strategy: FulfillmentStrategy,
    /// On-chain proof mode for the aggregation proof.
    pub agg_proof_mode: SP1ProofMode,
    /// Cycle limit for super-range proof requests.
    pub range_cycle_limit: u64,
    /// Gas limit for super-range proof requests.
    pub range_gas_limit: u64,
    /// Cycle limit for the aggregation proof request.
    pub agg_cycle_limit: u64,
    /// Gas limit for the aggregation proof request.
    pub agg_gas_limit: u64,
    /// Maximum price per proving gas unit.
    pub max_price_per_pgu: u64,
    /// Minimum auction period in seconds.
    pub min_auction_period: u64,
}

impl ProofProviderConfig {
    /// Parses the provider settings from environment variables, applying
    /// upstream op-succinct's defaults throughout.
    pub fn from_env() -> Result<Self> {
        let timeout = parsed_env_or("SP1_TIMEOUT_SECONDS", 14_400u64)?;
        anyhow::ensure!(
            timeout > 0,
            "SP1_TIMEOUT_SECONDS must be positive: 0 would abandon every proof request at its \
             first poll, right after paying to submit it"
        );
        let network_calls_timeout = parsed_env_or("NETWORK_CALLS_TIMEOUT", 15u64)?;
        anyhow::ensure!(
            network_calls_timeout > 0,
            "NETWORK_CALLS_TIMEOUT must be positive: 0 would time out every SPN call before any \
             I/O completes"
        );
        let auction_timeout = parsed_env_or("AUCTION_TIMEOUT", 60u64)?;
        anyhow::ensure!(
            auction_timeout > 0,
            "AUCTION_TIMEOUT must be positive: 0 would cancel every mainnet request by its second \
             poll"
        );
        Ok(Self {
            timeout,
            network_calls_timeout,
            auction_timeout,
            range_proof_strategy: parse_fulfillment_strategy(
                env::var("RANGE_PROOF_STRATEGY").unwrap_or_else(|_| "reserved".to_string()),
            )?,
            agg_proof_strategy: parse_fulfillment_strategy(
                env::var("AGG_PROOF_STRATEGY").unwrap_or_else(|_| "reserved".to_string()),
            )?,
            agg_proof_mode: parse_agg_proof_mode(
                &env::var("AGG_PROOF_MODE").unwrap_or_else(|_| "plonk".to_string()),
            )?,
            range_cycle_limit: parsed_env_or("RANGE_CYCLE_LIMIT", DEFAULT_PROOF_LIMIT)?,
            range_gas_limit: parsed_env_or("RANGE_GAS_LIMIT", DEFAULT_PROOF_LIMIT)?,
            agg_cycle_limit: parsed_env_or("AGG_CYCLE_LIMIT", DEFAULT_PROOF_LIMIT)?,
            agg_gas_limit: parsed_env_or("AGG_GAS_LIMIT", DEFAULT_PROOF_LIMIT)?,
            max_price_per_pgu: parsed_env_or("MAX_PRICE_PER_PGU", 300_000_000u64)?,
            min_auction_period: parsed_env_or("MIN_AUCTION_PERIOD", 1u64)?,
        })
    }
}

/// Parses the aggregation proof mode. Unlike upstream (which treats any
/// non-groth16 value as plonk), unknown values are rejected: a typo here
/// must not silently buy the wrong on-chain proof kind.
fn parse_agg_proof_mode(value: &str) -> Result<SP1ProofMode> {
    match value.to_ascii_lowercase().as_str() {
        "plonk" => Ok(SP1ProofMode::Plonk),
        "groth16" => Ok(SP1ProofMode::Groth16),
        other => bail!("invalid AGG_PROOF_MODE: {other} (expected plonk|groth16)"),
    }
}

/// How many chunks a defended timestamp span is partitioned into
/// (1-16 inclusive). Ported from upstream op-succinct's `RangeSplitCount`;
/// the unit here is super-root timestamps, not L2 blocks.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub struct RangeSplitCount(NonZeroU8);

impl RangeSplitCount {
    /// Maximum number of chunks.
    pub const MAX: u8 = 16;

    /// Creates a new `RangeSplitCount`, rejecting 0 and values above
    /// [`Self::MAX`].
    pub fn new(count: u8) -> Result<Self> {
        if count == 0 || count > Self::MAX {
            bail!("range splits must be between 1 and {}, got {count}", Self::MAX);
        }
        let count =
            NonZeroU8::new(count).ok_or_else(|| anyhow!("range splits must be non zero"))?;
        Ok(Self(count))
    }

    /// Returns a `RangeSplitCount` of one.
    pub const fn one() -> Self {
        Self(NonZeroU8::MIN)
    }

    /// Converts to `usize`.
    pub const fn to_usize(self) -> usize {
        self.0.get() as usize
    }

    /// Splits a timestamp span into up to `count` contiguous, non-empty
    /// chunks for proving.
    ///
    /// Each tuple `(start, end)` is a proving chunk where `start` is the
    /// agreed timestamp (already-proven checkpoint) and `end` is the claimed
    /// timestamp (included in the proof).
    ///
    /// - Errors if `start >= end`.
    /// - Caps the number of produced chunks at the span length.
    /// - Uses ceil division for even chunks; may yield fewer than requested (e.g. 9 timestamps / 4
    ///   -> step 3 -> 3 chunks).
    /// - Returned chunks exactly cover `(start, end]` with no gaps or overlaps.
    pub fn split(&self, start: u64, end: u64) -> Result<Vec<(u64, u64)>> {
        let total = end.checked_sub(start).ok_or_else(|| {
            anyhow!("end timestamp {end} is not greater than start timestamp {start}")
        })?;
        if total == 0 {
            bail!("start timestamp equals end timestamp ({start}); nothing to prove");
        }

        let splits = self.to_usize();
        if splits == 1 {
            return Ok(vec![(start, end)]);
        }

        // Never split into more chunks than there are timestamps.
        let segments = splits.min(total as usize);
        let mut ranges = Vec::with_capacity(segments);
        let step = total.div_ceil(segments as u64);

        let mut cur = start;
        for _ in 0..segments {
            if cur >= end {
                break;
            }
            let next = cur.saturating_add(step).min(end);
            ranges.push((cur, next));
            cur = next;
        }

        Ok(ranges)
    }
}

impl FromStr for RangeSplitCount {
    type Err = anyhow::Error;

    fn from_str(value: &str) -> Result<Self> {
        let count = value
            .parse::<u8>()
            .map_err(|err| anyhow!("invalid range split count {value}: {err}"))?;
        Self::new(count)
    }
}
/// Artifact suffix for the super-aggregation program ELF.
pub const AGGREGATION_ARTIFACT_SUFFIX: &str = ".agg.bin.gz";
/// Artifact suffix for the super-range program ELF (whose vkey the
/// aggregation program embeds).
pub const RANGE_ARTIFACT_SUFFIX: &str = ".range.bin.gz";

/// Returns the artifact URL for `prestate` with the given suffix under the
/// `base` directory URL: `<base>/<prestate><suffix>`.
pub fn prestate_artifact_url(base: &Url, prestate: B256, suffix: &str) -> Result<Url> {
    let mut url = base.clone();
    url.path_segments_mut()
        .map_err(|()| anyhow!("PRESTATES_URL cannot be a base: {}", redacted_url(base)))?
        .pop_if_empty()
        .push(&format!("{prestate}{suffix}"));
    Ok(url)
}

/// The program ELFs a prestate resolves to: the super-aggregation program
/// (whose vkey is the on-chain `absolutePrestate()`) and the super-range
/// program (whose vkey the aggregation program embeds, making the prestate
/// hash a unique key for both).
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PrestatePrograms {
    /// Decompressed super-aggregation program ELF.
    pub aggregation_elf: Vec<u8>,
    /// Decompressed super-range program ELF.
    pub range_elf: Vec<u8>,
}

/// Loads and decompresses both program ELFs for `prestate` from `base`
/// (`file://` reads the filesystem; `http(s)://` issues GET requests).
///
/// Fails when either artifact is missing, unreadable, or not valid gzip.
/// The create path uses a successful load as the proof that games with this
/// prestate are defendable; the defend path proves with the returned ELFs.
/// In network mode, `PrestateCache::proof_keys` later verifies during SP1
/// proving-key setup that the aggregation ELF actually hashes to `prestate`;
/// mock mode never executes the artifacts and skips that check.
pub async fn load_prestate(base: &Url, prestate: B256) -> Result<PrestatePrograms> {
    let aggregation_elf =
        fetch_artifact(&prestate_artifact_url(base, prestate, AGGREGATION_ARTIFACT_SUFFIX)?)
            .await?;
    let range_elf =
        fetch_artifact(&prestate_artifact_url(base, prestate, RANGE_ARTIFACT_SUFFIX)?).await?;
    Ok(PrestatePrograms { aggregation_elf, range_elf })
}

/// Maximum time for fetching a single prestate artifact, matching
/// op-challenger's prestate download timeout. Bounds the creation gate:
/// a hung prestates server must not stall the proposer's schedule loop.
const ARTIFACT_FETCH_TIMEOUT: std::time::Duration = std::time::Duration::from_secs(60);

/// Fetches a single artifact and gunzips it.
async fn fetch_artifact(url: &Url) -> Result<Vec<u8>> {
    let compressed = match url.scheme() {
        "file" => {
            let path = url
                .to_file_path()
                .map_err(|()| anyhow!("invalid file path in PRESTATES_URL: {url}"))?;
            tokio::fs::read(&path)
                .await
                .with_context(|| format!("failed to read prestate artifact {path:?}"))?
        }
        "http" | "https" => reqwest::Client::builder()
            .timeout(ARTIFACT_FETCH_TIMEOUT)
            .build()
            .context("failed to build prestate artifact client")?
            .get(url.clone())
            .send()
            .await
            .with_context(|| format!("failed to fetch prestate artifact at {}", redacted_url(url)))?
            .error_for_status()
            .with_context(|| format!("prestate artifact fetch failed for {}", redacted_url(url)))?
            .bytes()
            .await
            .with_context(|| {
                format!("failed to read prestate artifact body from {}", redacted_url(url))
            })?
            .to_vec(),
        other => bail!("unsupported PRESTATES_URL scheme {other} (expected file, http, or https)"),
    };

    let mut elf = Vec::new();
    std::io::Read::read_to_end(&mut flate2::read::GzDecoder::new(compressed.as_slice()), &mut elf)
        .with_context(|| format!("prestate artifact at {} is not valid gzip", redacted_url(url)))?;
    Ok(elf)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn proposal_safety_parsing() {
        assert_eq!(ProposalSafety::from_str("safe").unwrap(), ProposalSafety::Safe);
        assert_eq!(ProposalSafety::from_str("Finalized").unwrap(), ProposalSafety::Finalized);
        assert!(ProposalSafety::from_str("latest").is_err());
    }

    #[test]
    fn redacted_url_strips_userinfo() {
        let url: Url = "https://user:secret@rpc.example.com/key".parse().unwrap();
        assert_eq!(redacted_url(&url), "https://rpc.example.com/key");
        let plain: Url = "http://127.0.0.1:8545/".parse().unwrap();
        assert_eq!(redacted_url(&plain), "http://127.0.0.1:8545/");
        // file:// URLs cannot carry userinfo: set_username/set_password return
        // Err, which redacted_url ignores. This pins that choice (panicking on
        // Err would break file:// PRESTATES_URL) and that the URL renders
        // unchanged.
        let file: Url = "file:///data/prestates".parse().unwrap();
        assert_eq!(redacted_url(&file), "file:///data/prestates");
    }

    mod prestates {
        use super::*;

        const HASH_A: &str = "0x0101010101010101010101010101010101010101010101010101010101010101";

        fn artifact_dir(name: &str) -> std::path::PathBuf {
            let dir = std::env::temp_dir()
                .join(format!("kona-sp1-prestates-test-{}-{name}", std::process::id()));
            std::fs::create_dir_all(&dir).unwrap();
            dir
        }

        fn write_artifact(dir: &std::path::Path, suffix: &str, contents: &[u8]) {
            let mut gz = flate2::write::GzEncoder::new(Vec::new(), flate2::Compression::default());
            std::io::Write::write_all(&mut gz, contents).unwrap();
            std::fs::write(dir.join(format!("{HASH_A}{suffix}")), gz.finish().unwrap()).unwrap();
        }

        fn write_artifacts(dir: &std::path::Path) {
            write_artifact(dir, AGGREGATION_ARTIFACT_SUFFIX, b"aggregation-elf");
            write_artifact(dir, RANGE_ARTIFACT_SUFFIX, b"range-elf");
        }

        #[test]
        fn artifact_url_follows_challenger_naming() {
            let hash = HASH_A.parse::<B256>().unwrap();
            let base = Url::parse("https://example.com/prestates").unwrap();
            let url = prestate_artifact_url(&base, hash, AGGREGATION_ARTIFACT_SUFFIX).unwrap();
            assert_eq!(url.as_str(), format!("https://example.com/prestates/{HASH_A}.agg.bin.gz"));
            // A trailing slash on the base must not produce a double slash.
            let base = Url::parse("https://example.com/prestates/").unwrap();
            let url = prestate_artifact_url(&base, hash, RANGE_ARTIFACT_SUFFIX).unwrap();
            assert_eq!(
                url.as_str(),
                format!("https://example.com/prestates/{HASH_A}.range.bin.gz")
            );
        }

        #[tokio::test]
        async fn loads_and_decompresses_both_programs() {
            let dir = artifact_dir("present");
            write_artifacts(&dir);
            let base = Url::from_directory_path(&dir).unwrap();
            let programs = load_prestate(&base, HASH_A.parse::<B256>().unwrap()).await.unwrap();
            assert_eq!(programs.aggregation_elf, b"aggregation-elf");
            assert_eq!(programs.range_elf, b"range-elf");
        }

        #[tokio::test]
        async fn missing_artifacts_fail_to_load() {
            let dir = artifact_dir("missing");
            let base = Url::from_directory_path(&dir).unwrap();
            assert!(load_prestate(&base, HASH_A.parse::<B256>().unwrap()).await.is_err());
        }

        #[tokio::test]
        async fn one_missing_artifact_fails_to_load() {
            // Proving needs both programs: the aggregation ELF alone is not
            // enough to defend a game.
            let dir = artifact_dir("agg-only");
            write_artifact(&dir, AGGREGATION_ARTIFACT_SUFFIX, b"aggregation-elf");
            let base = Url::from_directory_path(&dir).unwrap();
            assert!(load_prestate(&base, HASH_A.parse::<B256>().unwrap()).await.is_err());
        }

        #[tokio::test]
        async fn corrupt_artifact_fails_to_load() {
            // A truncated or non-gzip artifact must not count as a usable
            // program.
            let dir = artifact_dir("corrupt");
            write_artifact(&dir, AGGREGATION_ARTIFACT_SUFFIX, b"aggregation-elf");
            std::fs::write(dir.join(format!("{HASH_A}{RANGE_ARTIFACT_SUFFIX}")), b"not gzip")
                .unwrap();
            let base = Url::from_directory_path(&dir).unwrap();
            assert!(load_prestate(&base, HASH_A.parse::<B256>().unwrap()).await.is_err());
        }

        #[tokio::test]
        async fn unsupported_scheme_is_an_error() {
            let url = Url::parse("ftp://example.com/prestates").unwrap();
            assert!(load_prestate(&url, HASH_A.parse::<B256>().unwrap()).await.is_err());
        }
    }

    mod proving_config {
        use super::*;

        #[test]
        fn proof_provider_kind_parse_rejects_unknown() {
            assert_eq!(ProofProviderKind::from_str("network").unwrap(), ProofProviderKind::Network);
            assert_eq!(ProofProviderKind::from_str("Mock").unwrap(), ProofProviderKind::Mock);
            assert!(ProofProviderKind::from_str("cluster").is_err());
            assert!(ProofProviderKind::from_str("").is_err());
        }

        #[test]
        fn l2_rpcs_parses_comma_list() {
            let urls = parse_url_list("http://a:1, http://b:2/").unwrap();
            assert_eq!(urls.len(), 2);
            assert_eq!(urls[1].as_str(), "http://b:2/");
            assert!(parse_url_list("").is_err());
            assert!(parse_url_list(" , ").is_err());
            assert!(parse_url_list("not a url").is_err());
        }

        #[test]
        fn agg_proof_mode_rejects_unknown() {
            assert!(matches!(parse_agg_proof_mode("plonk").unwrap(), SP1ProofMode::Plonk));
            assert!(matches!(parse_agg_proof_mode("Groth16").unwrap(), SP1ProofMode::Groth16));
            // Upstream silently maps unknown values to plonk; we reject:
            // a typo must not buy the wrong on-chain proof kind.
            assert!(parse_agg_proof_mode("core").is_err());
        }

        #[test]
        fn range_split_bounds() {
            assert!(RangeSplitCount::new(0).is_err());
            assert!(RangeSplitCount::new(17).is_err());
            assert_eq!(RangeSplitCount::one().to_usize(), 1);
            assert_eq!(RangeSplitCount::new(16).unwrap().to_usize(), 16);
        }

        #[test]
        fn range_split_chunks() {
            // Single split covers the whole span.
            assert_eq!(RangeSplitCount::one().split(100, 700).unwrap(), vec![(100, 700)]);
            // Ceil division may yield fewer chunks than requested:
            // 9 timestamps / 4 -> step 3 -> 3 chunks.
            assert_eq!(
                RangeSplitCount::new(4).unwrap().split(0, 9).unwrap(),
                vec![(0, 3), (3, 6), (6, 9)]
            );
            // Chunk count caps at the span length.
            assert_eq!(
                RangeSplitCount::new(16).unwrap().split(0, 4).unwrap(),
                vec![(0, 1), (1, 2), (2, 3), (3, 4)]
            );
            // Chunks exactly cover (start, end] with no gaps or overlaps.
            let chunks = RangeSplitCount::new(7).unwrap().split(10, 55).unwrap();
            assert_eq!(chunks.first().unwrap().0, 10);
            assert_eq!(chunks.last().unwrap().1, 55);
            for pair in chunks.windows(2) {
                assert_eq!(pair[0].1, pair[1].0);
            }
            // Degenerate spans are rejected.
            assert!(RangeSplitCount::one().split(5, 5).is_err());
            assert!(RangeSplitCount::one().split(6, 5).is_err());
        }

        /// `from_env` requires the new defend-path variables. Safe under
        /// nextest's process-per-test model; env mutation is `unsafe` on
        /// edition 2024.
        #[test]
        fn from_env_requires_defend_path_vars() {
            unsafe {
                env::set_var("L1_RPC", "http://127.0.0.1:8545");
                env::set_var("SUPERNODE_RPC", "http://127.0.0.1:9545");
                env::set_var("FACTORY_ADDRESS", "0x000000000000000000000000000000000000dEaD");
                env::set_var("PRESTATES_URL", "file:///tmp/prestates");
            }

            // PROOF_PROVIDER has no default.
            let err = ProposerConfig::from_env().unwrap_err().to_string();
            assert!(err.contains("PROOF_PROVIDER"), "unexpected error: {err}");

            unsafe { env::set_var("PROOF_PROVIDER", "mock") };
            let err = ProposerConfig::from_env().unwrap_err().to_string();
            assert!(err.contains("L2_RPCS"), "unexpected error: {err}");

            unsafe { env::set_var("L2_RPCS", "http://127.0.0.1:8646,http://127.0.0.1:8647") };
            let err = ProposerConfig::from_env().unwrap_err().to_string();
            assert!(err.contains("L1_BEACON_RPC"), "unexpected error: {err}");

            // Mock mode requires no NETWORK_* variables: with the witness
            // endpoints present, parsing succeeds without SPN credentials.
            unsafe { env::set_var("L1_BEACON_RPC", "http://127.0.0.1:5052") };
            let config = ProposerConfig::from_env().unwrap();
            assert_eq!(config.proof_provider, ProofProviderKind::Mock);
            assert_eq!(config.l2_rpcs.len(), 2);
            assert!(config.rollup_config_paths.is_none());
            assert_eq!(config.range_split_count, RangeSplitCount::one());
            assert_eq!(config.max_concurrent_defense_tasks.get(), 8);
            assert_eq!(config.proof_provider_config.timeout, 14_400);
        }

        /// A zero defense cap would silently disable defense entirely;
        /// `NonZeroU64` parsing rejects it at startup. Safe under nextest's
        /// process-per-test model; env mutation is `unsafe` on edition 2024.
        #[test]
        fn zero_defense_cap_is_rejected() {
            unsafe {
                env::set_var("L1_RPC", "http://127.0.0.1:8545");
                env::set_var("SUPERNODE_RPC", "http://127.0.0.1:9545");
                env::set_var("FACTORY_ADDRESS", "0x000000000000000000000000000000000000dEaD");
                env::set_var("PRESTATES_URL", "file:///tmp/prestates");
                env::set_var("PROOF_PROVIDER", "mock");
                env::set_var("L2_RPCS", "http://127.0.0.1:8646");
                env::set_var("L1_BEACON_RPC", "http://127.0.0.1:5052");
                env::set_var("MAX_CONCURRENT_DEFENSE_TASKS", "0");
            }
            let err = ProposerConfig::from_env().unwrap_err().to_string();
            assert!(err.contains("MAX_CONCURRENT_DEFENSE_TASKS"), "unexpected error: {err}");
        }

        /// Zero SPN timeouts are configuration errors, not degraded modes:
        /// each would spin or abandon paid work at the first poll. Safe
        /// under nextest's process-per-test model.
        #[test]
        fn zero_spn_timeouts_are_rejected() {
            unsafe {
                env::set_var("L1_RPC", "http://127.0.0.1:8545");
                env::set_var("SUPERNODE_RPC", "http://127.0.0.1:9545");
                env::set_var("FACTORY_ADDRESS", "0x000000000000000000000000000000000000dEaD");
                env::set_var("PRESTATES_URL", "file:///tmp/prestates");
                env::set_var("PROOF_PROVIDER", "mock");
                env::set_var("L2_RPCS", "http://127.0.0.1:8646");
                env::set_var("L1_BEACON_RPC", "http://127.0.0.1:5052");
            }
            for var in ["SP1_TIMEOUT_SECONDS", "NETWORK_CALLS_TIMEOUT", "AUCTION_TIMEOUT"] {
                unsafe { env::set_var(var, "0") };
                let err = ProposerConfig::from_env().unwrap_err().to_string();
                assert!(err.contains(var), "expected {var} rejection, got: {err}");
                unsafe { env::remove_var(var) };
            }
        }
    }
}
