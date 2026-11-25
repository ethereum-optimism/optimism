//! Engine API version selection based on Optimism hardfork activations.
//!
//! Automatically selects the appropriate Engine API method versions based on
//! the rollup configuration and block timestamps. The required method version
//! tracks the L1 (Ethereum) hardfork implied by the active OP hardfork, so
//! versions are selected by querying the L1 fork activations that the rollup
//! config derives from the canonical OP fork → L1 fork mapping in
//! `alloy-op-hardforks`.
//!
//! # Version Mapping
//!
//! - **pre-Cancun (Bedrock, Canyon, Delta)** → V2 methods
//! - **Cancun (Ecotone)** → V3 methods
//! - **Prague (Isthmus)** → V4 methods
//! - **Osaka (Karst)** → V5 `getPayload` (`newPayload`/`forkchoiceUpdated` stay at their V4/V3)
//!
//! Adapted from the [OP Node version providers](https://github.com/ethereum-optimism/optimism/blob/develop/op-node/rollup/types.go#L546).

use alloy_hardforks::EthereumHardforks;
use kona_genesis::RollupConfig;

/// Engine API version for `engine_forkchoiceUpdated` method calls.
///
/// Selects between V2 and V3 based on hardfork activation. V3 is required
/// for Ecotone/Cancun and later hardforks to support new consensus features.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum EngineForkchoiceVersion {
    /// Version 2: Used for Bedrock, Canyon, and Delta hardforks.
    V2,
    /// Version 3: Required for Ecotone/Cancun and later hardforks.
    V3,
}

impl EngineForkchoiceVersion {
    /// Returns the appropriate [`EngineForkchoiceVersion`] for the chain at the given attributes.
    ///
    /// Uses the [`RollupConfig`] to check which L1 hardfork is implied at the given timestamp.
    pub fn from_cfg(cfg: &RollupConfig, timestamp: u64) -> Self {
        if cfg.is_cancun_active_at_timestamp(timestamp) {
            // Ecotone+
            Self::V3
        } else {
            // Bedrock, Canyon, Delta
            Self::V2
        }
    }
}

/// Engine API version for `engine_newPayload` method calls.
///
/// Progressive version selection based on hardfork activation:
/// - V2: Basic payload processing
/// - V3: Adds Cancun/Ecotone support
/// - V4: Adds Isthmus hardfork features
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum EngineNewPayloadVersion {
    /// Version 2: Basic payload processing for early hardforks.
    V2,
    /// Version 3: Adds Cancun/Ecotone consensus features.
    V3,
    /// Version 4: Adds Isthmus hardfork support.
    V4,
}

impl EngineNewPayloadVersion {
    /// Returns the appropriate [`EngineNewPayloadVersion`] for the chain at the given timestamp.
    ///
    /// Uses the [`RollupConfig`] to check which L1 hardfork is implied at the given timestamp.
    pub fn from_cfg(cfg: &RollupConfig, timestamp: u64) -> Self {
        if cfg.is_prague_active_at_timestamp(timestamp) {
            Self::V4
        } else if cfg.is_cancun_active_at_timestamp(timestamp) {
            Self::V3
        } else {
            Self::V2
        }
    }
}

/// Engine API version for `engine_getPayload` method calls.
///
/// Matches the payload version used for retrieval with the version
/// used during payload construction, ensuring API compatibility.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum EngineGetPayloadVersion {
    /// Version 2: Basic payload retrieval.
    V2,
    /// Version 3: Enhanced payload data for Cancun/Ecotone.
    V3,
    /// Version 4: Extended payload format for Isthmus.
    V4,
    /// Version 5: Osaka (`engine_getPayloadV5`); reuses the V4-shaped envelope.
    V5,
}

impl EngineGetPayloadVersion {
    /// Returns the appropriate [`EngineGetPayloadVersion`] for the chain at the given timestamp.
    ///
    /// Uses the [`RollupConfig`] to check which L1 hardfork is implied at the given timestamp.
    /// Osaka (Karst) bumps only `getPayload` to V5; `newPayload`/`forkchoiceUpdated` are
    /// unchanged.
    pub fn from_cfg(cfg: &RollupConfig, timestamp: u64) -> Self {
        if cfg.is_osaka_active_at_timestamp(timestamp) {
            Self::V5
        } else if cfg.is_prague_active_at_timestamp(timestamp) {
            Self::V4
        } else if cfg.is_cancun_active_at_timestamp(timestamp) {
            Self::V3
        } else {
            Self::V2
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use kona_genesis::HardForkConfig;

    fn cfg() -> RollupConfig {
        RollupConfig {
            hardforks: HardForkConfig {
                ecotone_time: Some(10),
                isthmus_time: Some(20),
                karst_time: Some(30),
                ..Default::default()
            },
            ..Default::default()
        }
    }

    #[test]
    fn forkchoice_version_selects_by_active_hardfork() {
        let cfg = cfg();
        assert_eq!(EngineForkchoiceVersion::from_cfg(&cfg, 5), EngineForkchoiceVersion::V2);
        assert_eq!(EngineForkchoiceVersion::from_cfg(&cfg, 10), EngineForkchoiceVersion::V3);
        assert_eq!(EngineForkchoiceVersion::from_cfg(&cfg, 35), EngineForkchoiceVersion::V3);
    }

    #[test]
    fn new_payload_version_selects_by_active_hardfork() {
        let cfg = cfg();
        assert_eq!(EngineNewPayloadVersion::from_cfg(&cfg, 5), EngineNewPayloadVersion::V2);
        assert_eq!(EngineNewPayloadVersion::from_cfg(&cfg, 15), EngineNewPayloadVersion::V3);
        assert_eq!(EngineNewPayloadVersion::from_cfg(&cfg, 25), EngineNewPayloadVersion::V4);
        // Karst (Osaka) does not bump newPayload.
        assert_eq!(EngineNewPayloadVersion::from_cfg(&cfg, 35), EngineNewPayloadVersion::V4);
    }

    #[test]
    fn get_payload_version_selects_by_active_hardfork() {
        let cfg = cfg();
        assert_eq!(EngineGetPayloadVersion::from_cfg(&cfg, 5), EngineGetPayloadVersion::V2);
        assert_eq!(EngineGetPayloadVersion::from_cfg(&cfg, 15), EngineGetPayloadVersion::V3);
        assert_eq!(EngineGetPayloadVersion::from_cfg(&cfg, 25), EngineGetPayloadVersion::V4);
        // Karst (Osaka) selects the new V5 getPayload, at and after its activation timestamp.
        assert_eq!(EngineGetPayloadVersion::from_cfg(&cfg, 30), EngineGetPayloadVersion::V5);
        assert_eq!(EngineGetPayloadVersion::from_cfg(&cfg, 35), EngineGetPayloadVersion::V5);
    }
}
