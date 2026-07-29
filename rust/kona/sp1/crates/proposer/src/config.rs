//! Environment-driven proposer configuration.
//!
//! Trimmed from op-succinct's `fault-proof/src/config.rs` (@ 13716c2c):
//! proving/defense knobs (mock mode, fast finality, range splitting, proof
//! provider settings) arrive with the defend path (#21463); challenger and
//! forge-deploy configs are not ported.

use std::{env, str::FromStr};

use alloy_primitives::{Address, B256};
use alloy_transport_http::reqwest::{self, Url};
use anyhow::{Context, Result, anyhow, bail};

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
    /// The create path only checks artifact existence (see
    /// `prestate_available`): creation pauses when either artifact is
    /// missing, since the proposer could not defend such a game once proving
    /// lands. The defend path will load the ELFs from the same location.
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
    pub max_fee_per_gas: Option<u128>,

    /// Optional cap, in wei, on the EIP-1559 max priority fee per gas the
    /// fee estimator may set. Unset = uncapped.
    pub max_priority_fee_per_gas: Option<u128>,
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
        .map_err(|()| anyhow!("PRESTATES_URL cannot be a base: {base}"))?
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
/// Verifying that the aggregation ELF actually hashes to `prestate` requires
/// an SP1 proving-key setup and lands with the proving logic.
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
            .with_context(|| format!("failed to fetch prestate artifact at {url}"))?
            .error_for_status()
            .with_context(|| format!("prestate artifact fetch failed for {url}"))?
            .bytes()
            .await
            .with_context(|| format!("failed to read prestate artifact body from {url}"))?
            .to_vec(),
        other => bail!("unsupported PRESTATES_URL scheme {other} (expected file, http, or https)"),
    };

    let mut elf = Vec::new();
    std::io::Read::read_to_end(&mut flate2::read::GzDecoder::new(compressed.as_slice()), &mut elf)
        .with_context(|| format!("prestate artifact at {url} is not valid gzip"))?;
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
}
