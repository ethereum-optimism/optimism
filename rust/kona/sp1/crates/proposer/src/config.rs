//! Environment-driven proposer configuration.
//!
//! Trimmed from op-succinct's `fault-proof/src/config.rs` (@ 13716c2c):
//! proving/defense knobs (mock mode, fast finality, range splitting, proof
//! provider settings) arrive with the defend path (#21463); challenger and
//! forge-deploy configs are not ported.

use std::{env, str::FromStr};

use alloy_primitives::{Address, B256};
use alloy_transport_http::reqwest::Url;
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

    /// The dispute game type to propose (ZK dispute game = 10).
    pub game_type: u32,

    /// The super-aggregation program verification key. Game implementations
    /// whose `absolutePrestate()` differs are foreign: never used as
    /// parents, never resolved or claimed, and creation pauses when the
    /// registered implementation mismatches (hardfork safety).
    pub program_vkey: B256,

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
            game_type: parsed_env_or("GAME_TYPE", 10u32)?,
            program_vkey: env::var("PROGRAM_VKEY")
                .context("PROGRAM_VKEY not set")?
                .parse()
                .map_err(|err| anyhow!("invalid PROGRAM_VKEY: {err}"))?,
            proposal_interval_seconds: parsed_env_or("PROPOSAL_INTERVAL_SECONDS", 3600u64)?,
            proposal_safety: parsed_env_or("PROPOSAL_SAFETY", ProposalSafety::Finalized)?,
            fetch_interval: parsed_env_or("FETCH_INTERVAL", 30u64)?,
            metrics_port: parsed_env_or("METRICS_PORT", 0u16)?,
            sync_l1_confirmations: parsed_env_or("SYNC_L1_CONFIRMATIONS", 0u64)?,
            tx_confirmation_timeout: parsed_env_or("TX_CONFIRMATION_TIMEOUT", 60u64)?,
        })
    }
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
}
