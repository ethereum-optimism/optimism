//! Engine API version selection based on Optimism hardfork activations.
//!
//! Automatically selects the appropriate Engine API method versions based on
//! the rollup configuration and block timestamps. Different Optimism hardforks
//! require different Engine API versions to support new features.
//!
//! # Version Mapping
//!
//! - **Bedrock, Canyon, Delta** → V2 methods
//! - **Ecotone (Cancun)** → V3 methods
//! - **Isthmus** → V4 methods
//!
//! Adapted from the [OP Node version providers](https://github.com/ethereum-optimism/optimism/blob/develop/op-node/rollup/types.go#L546).

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
    /// Uses the [`RollupConfig`] to check which hardfork is active at the given timestamp.
    pub fn from_cfg(cfg: &RollupConfig, timestamp: u64) -> Self {
        if cfg.is_ecotone_active(timestamp) {
            // Cancun+
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
    /// Uses the [`RollupConfig`] to check which hardfork is active at the given timestamp.
    pub fn from_cfg(cfg: &RollupConfig, timestamp: u64) -> Self {
        if cfg.is_isthmus_active(timestamp) {
            Self::V4
        } else if cfg.is_ecotone_active(timestamp) {
            // Cancun
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
}

impl EngineGetPayloadVersion {
    /// Returns the appropriate [`EngineGetPayloadVersion`] for the chain at the given timestamp.
    ///
    /// Uses the [`RollupConfig`] to check which hardfork is active at the given timestamp.
    pub fn from_cfg(cfg: &RollupConfig, timestamp: u64) -> Self {
        if cfg.is_isthmus_active(timestamp) {
            Self::V4
        } else if cfg.is_ecotone_active(timestamp) {
            // Cancun
            Self::V3
        } else {
            Self::V2
        }
    }
}
