use alloy_primitives::{B256, ChainId};
use kona_genesis::ChainGenesis;
use kona_interop::DerivedRefPair;
use kona_protocol::BlockInfo;
use std::collections::HashMap;

use crate::SupervisorError;

/// Genesis provides the genesis information relevant for Interop.
#[derive(Debug, Default, Clone)]
pub struct Genesis {
    /// The L1 [`BlockInfo`] that the rollup starts after.
    pub l1: BlockInfo,
    /// The L2 [`BlockInfo`] that the rollup starts from.
    pub l2: BlockInfo,
}

impl Genesis {
    /// Creates a new Genesis with the given L1 and L2 block seals.
    pub const fn new(l1: BlockInfo, l2: BlockInfo) -> Self {
        Self { l1, l2 }
    }

    /// Creates a new Genesis from a RollupConfig.
    pub const fn new_from_rollup_genesis(genesis: ChainGenesis, l1_block: BlockInfo) -> Self {
        Self {
            l1: l1_block,
            l2: BlockInfo::new(genesis.l2.hash, genesis.l2.number, B256::ZERO, genesis.l2_time),
        }
    }

    /// Returns the genesis as a [`DerivedRefPair`].
    pub const fn get_derived_pair(&self) -> DerivedRefPair {
        DerivedRefPair { derived: self.l2, source: self.l1 }
    }
}

/// RollupConfig contains the configuration for the Optimism rollup.
#[derive(Debug, Default, Clone)]
pub struct RollupConfig {
    /// Genesis anchor information for the rollup.
    pub genesis: Genesis,

    /// The block time of the L2, in seconds.
    pub block_time: u64,

    /// Activation time for the interop network upgrade.
    pub interop_time: Option<u64>,
}

impl RollupConfig {
    /// Creates a new RollupConfig with the given genesis and block time.
    pub const fn new(genesis: Genesis, block_time: u64, interop_time: Option<u64>) -> Self {
        Self { genesis, block_time, interop_time }
    }

    /// Creates a new [`RollupConfig`] with the given genesis and block time.
    pub fn new_from_rollup_config(
        config: kona_genesis::RollupConfig,
        l1_block: BlockInfo,
    ) -> Result<Self, SupervisorError> {
        if config.genesis.l1.number != l1_block.number {
            return Err(SupervisorError::L1BlockMismatch {
                expected: config.genesis.l1.number,
                got: l1_block.number,
            });
        }

        Ok(Self {
            genesis: Genesis::new_from_rollup_genesis(config.genesis, l1_block),
            block_time: config.block_time,
            interop_time: config.hardforks.interop_time,
        })
    }

    /// Returns `true` if the timestamp is at or after the interop activation time.
    ///
    /// Interop activates at [`interop_time`](Self::interop_time). This function checks whether the
    /// provided timestamp is before or after interop timestamp.
    ///
    /// Returns `false` if `interop_time` is not configured.
    pub fn is_interop(&self, timestamp: u64) -> bool {
        self.interop_time.is_some_and(|t| timestamp >= t)
    }

    /// Returns `true` if the timestamp is strictly after the interop activation block.
    ///
    /// Interop activates at [`interop_time`](Self::interop_time). This function checks whether the
    /// provided timestamp is *after* that activation, skipping the activation block
    /// itself.
    ///
    /// Returns `false` if `interop_time` is not configured.
    pub fn is_post_interop(&self, timestamp: u64) -> bool {
        self.is_interop(timestamp.saturating_sub(self.block_time))
    }

    /// Returns `true` if given block is the interop activation block.
    ///
    /// An interop activation block is defined as the block that is right after the
    /// interop activation time.
    ///
    /// Returns `false` if `interop_time` is not configured.
    pub fn is_interop_activation_block(&self, block: BlockInfo) -> bool {
        self.is_interop(block.timestamp) &&
            !self.is_interop(block.timestamp.saturating_sub(self.block_time))
    }
}

/// RollupConfigSet contains the configuration for multiple Optimism rollups.
#[derive(Debug, Clone, Default)]
pub struct RollupConfigSet {
    /// The rollup configurations for the Optimism rollups.
    pub rollups: HashMap<u64, RollupConfig>,
}

impl RollupConfigSet {
    /// Creates a new RollupConfigSet with the given rollup configurations.
    pub const fn new(rollups: HashMap<u64, RollupConfig>) -> Self {
        Self { rollups }
    }

    /// Returns the rollup configuration for the given chain id.
    pub fn get(&self, chain_id: u64) -> Option<&RollupConfig> {
        self.rollups.get(&chain_id)
    }

    /// adds a new rollup configuration to the set using the provided chain ID and RollupConfig.
    pub fn add_from_rollup_config(
        &mut self,
        chain_id: u64,
        config: kona_genesis::RollupConfig,
        l1_block: BlockInfo,
    ) -> Result<(), SupervisorError> {
        let rollup_config = RollupConfig::new_from_rollup_config(config, l1_block)?;
        self.rollups.insert(chain_id, rollup_config);
        Ok(())
    }

    /// Returns `true` if interop is enabled for the chain at given timestamp.
    pub fn is_post_interop(&self, chain_id: ChainId, timestamp: u64) -> bool {
        self.get(chain_id).map(|cfg| cfg.is_post_interop(timestamp)).unwrap_or(false) // if config not found, return false
    }

    /// Returns `true` if given block is the interop activation block for the specified chain.
    pub fn is_interop_activation_block(&self, chain_id: ChainId, block: BlockInfo) -> bool {
        self.get(chain_id).map(|cfg| cfg.is_interop_activation_block(block)).unwrap_or(false)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::ChainId;
    use kona_protocol::BlockInfo;

    fn dummy_blockinfo(number: u64) -> BlockInfo {
        BlockInfo::new(B256::ZERO, number, B256::ZERO, 0)
    }

    #[test]
    fn test_is_interop_enabled() {
        let mut set = RollupConfigSet::default();
        let chain_id = ChainId::from(1u64);

        // Interop time is 100, block_time is 10
        let rollup_config =
            RollupConfig::new(Genesis::new(dummy_blockinfo(0), dummy_blockinfo(0)), 10, Some(100));
        set.rollups.insert(chain_id, rollup_config);

        // Before interop time
        assert!(!set.is_post_interop(chain_id, 100));
        assert!(!set.is_post_interop(chain_id, 109));
        // After interop time (should be true)
        assert!(set.is_post_interop(chain_id, 110));
        assert!(set.is_post_interop(chain_id, 111));
        assert!(set.is_post_interop(chain_id, 200));

        // Unknown chain_id returns false
        assert!(!set.is_post_interop(ChainId::from(999u64), 200));
    }

    #[test]
    fn test_rollup_config_is_interop_interop_time_zero() {
        // Interop time is 100, block_time is 10
        let rollup_config =
            RollupConfig::new(Genesis::new(dummy_blockinfo(0), dummy_blockinfo(0)), 2, Some(0));

        assert!(rollup_config.is_interop(0));
        assert!(rollup_config.is_interop(1000));
    }
}
