//! Proposer service for super-root ZK dispute games.
//!
//! Handles super-root claims sourced from a supernode, `parentIndex || superRootProof`
//! extra data, prestate-based ownership, two-phase `DelayedWETH` bond claims, and defense
//! of challenged games in the owned set. Witness collection and native output computation
//! live in [`proving`]; SP1 proof providers live in [`prover`].

/// Prefix for all proposer-owned environment variables.
pub const ENV_VAR_PREFIX: &str = "KONA_SP1_PROPOSER";

/// Builds a proposer-owned environment-variable name.
pub fn env_var(suffix: &str) -> String {
    kona_sp1_host_utils::prefixed_env_var(ENV_VAR_PREFIX, suffix)
}

pub mod config;
pub mod contract;
pub mod metrics;
pub mod proposer;
pub mod prover;
pub mod proving;
pub mod signer;
pub mod superroot;

mod adapters;
mod ports;

use alloy_provider::RootProvider;

/// The L1 provider type used throughout the proposer (a plain alloy root provider).
pub type L1Provider = RootProvider;

/// The dispute game type this proposer plays. Game type 10 is reserved as the
/// ZK dispute game type for all OP Stack networks; not configurable to avoid
/// misconfiguration.
pub const ZK_GAME_TYPE: u32 = 10;

/// Prefix used for transaction revert errors.
pub const TX_REVERTED_PREFIX: &str = "transaction reverted:";

/// Extension trait for checking transaction error types.
pub trait TxErrorExt {
    /// Whether the error is a transaction revert.
    fn is_revert(&self) -> bool;
}

impl TxErrorExt for anyhow::Error {
    fn is_revert(&self) -> bool {
        self.to_string().starts_with(TX_REVERTED_PREFIX)
    }
}

#[cfg(test)]
mod tests {
    use super::{TX_REVERTED_PREFIX, TxErrorExt};

    /// `is_revert` matches on the OUTERMOST rendering. Context added above a
    /// bail site defeats the prefix check - pinned here so any refactor of
    /// `is_revert`'s matching (e.g. switching to `chain()` traversal) must
    /// revisit this contract. Typed error lands with #22019.
    #[test]
    fn revert_detection_pins_prefix_rendering() {
        let revert = anyhow::anyhow!("{TX_REVERTED_PREFIX} receipt");
        assert!(revert.is_revert());
        let wrapped = revert.context("submitting resolution");
        assert!(!wrapped.is_revert());
        assert!(!anyhow::anyhow!("other failure").is_revert());
    }
}
