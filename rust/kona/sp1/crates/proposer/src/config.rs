//! Environment-driven proposer configuration.
//!
//! Trimmed from op-succinct's `fault-proof/src/config.rs` (@ 13716c2c):
//! proving/defense knobs (mock mode, fast finality, range splitting, proof
//! provider settings) arrive with the defend path (#21463); challenger and
//! forge-deploy configs are not ported.

use std::{collections::BTreeSet, env, str::FromStr};

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
    pub supernode_rpc: String,

    /// The address of the `DisputeGameFactory` contract.
    pub factory_address: Address,

    /// URL of a TOML document mapping program names to 32-byte prestate
    /// hashes (the `vkeys.toml` shape), accepting `file://` and `http(s)://`
    /// URIs. The value set is the known prestates: game creation pauses when
    /// the registered implementation's `absolutePrestate()` is not in the
    /// set (see `ensure_prestate_known`), and the set is re-fetched on a
    /// miss so a hardfork that registers a new prestate only requires
    /// updating the document, not restarting the proposer.
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
    optional_env(name)
        .map(|value| value.parse::<T>().map_err(|err| anyhow!("invalid {name}: {err}")))
        .transpose()
        .map(|value| value.unwrap_or(default))
}

impl ProposerConfig {
    /// Parses the configuration from environment variables, applying defaults
    /// for optional settings and failing on missing or invalid required ones.
    pub fn from_env() -> Result<Self> {
        Ok(Self {
            l1_rpc: env::var("L1_RPC")
                .context("L1_RPC not set")?
                .parse()
                .map_err(|err| anyhow!("invalid L1_RPC: {err}"))?,
            supernode_rpc: env::var("SUPERNODE_RPC").context("SUPERNODE_RPC not set")?,
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
            tx_confirmation_timeout: parsed_env_or("TX_CONFIRMATION_TIMEOUT", 60u64)?,
        })
    }
}

/// Loads the known prestate set from `url` (`file://` or `http(s)://`).
///
/// The document is TOML mapping program names to 32-byte hex hashes; the
/// value set is returned. Fails on unreachable documents, parse errors,
/// invalid hashes, or an empty set - an empty set would silently pause
/// creation forever.
pub async fn load_prestates(url: &Url) -> Result<BTreeSet<B256>> {
    let text = match url.scheme() {
        "file" => {
            let path = url
                .to_file_path()
                .map_err(|()| anyhow!("invalid file path in PRESTATES_URL: {url}"))?;
            std::fs::read_to_string(&path)
                .with_context(|| format!("failed to read prestates file {path:?}"))?
        }
        "http" | "https" => reqwest::get(url.clone())
            .await
            .with_context(|| format!("failed to fetch prestates from {url}"))?
            .error_for_status()
            .with_context(|| format!("prestates fetch returned an error status for {url}"))?
            .text()
            .await
            .context("failed to read prestates response body")?,
        other => bail!("unsupported PRESTATES_URL scheme {other} (expected file, http, or https)"),
    };
    parse_prestates(&text)
}

/// Parses a TOML document of `name = "0x..."` entries into the prestate set.
fn parse_prestates(text: &str) -> Result<BTreeSet<B256>> {
    let table: std::collections::BTreeMap<String, String> =
        toml::from_str(text).context("prestates document is not a TOML table of hash strings")?;
    let mut prestates = BTreeSet::new();
    for (name, raw) in &table {
        let hash: B256 = raw
            .parse()
            .map_err(|err| anyhow!("prestate entry {name} is not a 32-byte hash: {err}"))?;
        prestates.insert(hash);
    }
    if prestates.is_empty() {
        bail!("prestates document contains no entries");
    }
    Ok(prestates)
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

    mod prestates {
        use super::*;

        const HASH_A: &str = "0x0101010101010101010101010101010101010101010101010101010101010101";
        const HASH_B: &str = "0x0202020202020202020202020202020202020202020202020202020202020202";

        #[test]
        fn parses_named_entries_into_value_set() {
            let set = parse_prestates(&format!(
                "super-aggregation = {HASH_A:?}\nnext-fork = {HASH_B:?}\n"
            ))
            .unwrap();
            assert_eq!(set.len(), 2);
            assert!(set.contains(&HASH_A.parse::<B256>().unwrap()));
            assert!(set.contains(&HASH_B.parse::<B256>().unwrap()));
        }

        #[test]
        fn rejects_invalid_hash() {
            assert!(parse_prestates("bad = \"0x1234\"\n").is_err());
        }

        #[test]
        fn rejects_non_string_entry() {
            assert!(parse_prestates("bad = 7\n").is_err());
        }

        #[test]
        fn rejects_empty_document() {
            // An empty set would silently pause creation forever.
            assert!(parse_prestates("").is_err());
        }

        #[test]
        fn rejects_non_toml() {
            assert!(parse_prestates("{\"json\": true}").is_err());
        }

        #[tokio::test]
        async fn loads_from_file_url() {
            let path = std::env::temp_dir()
                .join(format!("kona-sp1-prestates-test-{}.toml", std::process::id()));
            std::fs::write(&path, format!("super-aggregation = {HASH_A:?}\n")).unwrap();
            let url = Url::from_file_path(&path).unwrap();
            let set = load_prestates(&url).await.unwrap();
            std::fs::remove_file(&path).ok();
            assert_eq!(set.len(), 1);
            assert!(set.contains(&HASH_A.parse::<B256>().unwrap()));
        }

        #[tokio::test]
        async fn missing_file_is_an_error() {
            let url = Url::parse("file:///nonexistent/kona-sp1-prestates.toml").unwrap();
            assert!(load_prestates(&url).await.is_err());
        }

        #[tokio::test]
        async fn unsupported_scheme_is_an_error() {
            let url = Url::parse("ftp://example.com/prestates.toml").unwrap();
            assert!(load_prestates(&url).await.is_err());
        }
    }
}
