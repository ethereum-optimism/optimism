//! Additional configuration for the OP builder

use std::sync::{Arc, atomic::AtomicU64};

/// Settings for the OP builder.
#[derive(Debug, Clone, Default)]
pub struct OpBuilderConfig {
    /// Data availability configuration for the OP builder.
    pub da_config: OpDAConfig,
    /// Gas limit configuration for the OP builder.
    pub gas_limit_config: OpGasLimitConfig,
}

impl OpBuilderConfig {
    /// Creates a new OP builder configuration with the given data availability configuration.
    pub const fn new(da_config: OpDAConfig, gas_limit_config: OpGasLimitConfig) -> Self {
        Self { da_config, gas_limit_config }
    }

    /// Returns the Data Availability configuration for the OP builder, if it has configured
    /// constraints.
    pub fn constrained_da_config(&self) -> Option<&OpDAConfig> {
        if self.da_config.is_empty() { None } else { Some(&self.da_config) }
    }
}

/// Contains the Data Availability configuration for the OP builder.
///
/// This type is shareable and can be used to update the DA configuration for the OP payload
/// builder.
#[derive(Debug, Clone, Default)]
pub struct OpDAConfig {
    inner: Arc<OpDAConfigInner>,
}

impl OpDAConfig {
    /// Creates a new Data Availability configuration with the given maximum sizes.
    pub fn new(max_da_tx_size: u64, max_da_block_size: u64) -> Self {
        let this = Self::default();
        this.set_max_da_size(max_da_tx_size, max_da_block_size);
        this
    }

    /// Returns whether the configuration is empty.
    pub fn is_empty(&self) -> bool {
        self.max_da_tx_size().is_none() && self.max_da_block_size().is_none()
    }

    /// Returns the max allowed data availability size per transactions, if any.
    pub fn max_da_tx_size(&self) -> Option<u64> {
        let val = self.inner.max_da_tx_size.load(std::sync::atomic::Ordering::Relaxed);
        if val == 0 { None } else { Some(val) }
    }

    /// Returns the max allowed data availability size per block, if any.
    pub fn max_da_block_size(&self) -> Option<u64> {
        let val = self.inner.max_da_block_size.load(std::sync::atomic::Ordering::Relaxed);
        if val == 0 { None } else { Some(val) }
    }

    /// Sets the maximum data availability size currently allowed for inclusion. 0 means no maximum.
    pub fn set_max_da_size(&self, max_da_tx_size: u64, max_da_block_size: u64) {
        self.set_max_tx_size(max_da_tx_size);
        self.set_max_block_size(max_da_block_size);
    }

    /// Sets the maximum data availability size per transaction currently allowed for inclusion. 0
    /// means no maximum.
    pub fn set_max_tx_size(&self, max_da_tx_size: u64) {
        self.inner.max_da_tx_size.store(max_da_tx_size, std::sync::atomic::Ordering::Relaxed);
    }

    /// Sets the maximum data availability size per block currently allowed for inclusion. 0 means
    /// no maximum.
    pub fn set_max_block_size(&self, max_da_block_size: u64) {
        self.inner.max_da_block_size.store(max_da_block_size, std::sync::atomic::Ordering::Relaxed);
    }
}

#[derive(Debug, Default)]
struct OpDAConfigInner {
    /// Don't include any transactions with data availability size larger than this in any built
    /// block
    ///
    /// 0 means no limit.
    max_da_tx_size: AtomicU64,
    /// Maximum total data availability size for a block
    ///
    /// 0 means no limit.
    max_da_block_size: AtomicU64,
}

/// Contains the Gas Limit configuration for the OP builder.
///
/// This type is shareable and can be used to update the Gas Limit configuration for the OP payload
/// builder.
#[derive(Debug, Clone, Default)]
pub struct OpGasLimitConfig {
    /// Gas limit for a transaction
    ///
    /// 0 means use the default gas limit.
    gas_limit: Arc<AtomicU64>,
}

impl OpGasLimitConfig {
    /// Creates a new Gas Limit configuration with the given maximum gas limit.
    pub fn new(max_gas_limit: u64) -> Self {
        let this = Self::default();
        this.set_gas_limit(max_gas_limit);
        this
    }
    /// Returns the gas limit for a transaction, if any.
    pub fn gas_limit(&self) -> Option<u64> {
        let val = self.gas_limit.load(std::sync::atomic::Ordering::Relaxed);
        if val == 0 { None } else { Some(val) }
    }
    /// Sets the gas limit for a transaction. 0 means use the default gas limit.
    pub fn set_gas_limit(&self, gas_limit: u64) {
        self.gas_limit.store(gas_limit, std::sync::atomic::Ordering::Relaxed);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_da() {
        let da = OpDAConfig::default();
        assert_eq!(da.max_da_tx_size(), None);
        assert_eq!(da.max_da_block_size(), None);
        da.set_max_da_size(100, 200);
        assert_eq!(da.max_da_tx_size(), Some(100));
        assert_eq!(da.max_da_block_size(), Some(200));
        da.set_max_da_size(0, 0);
        assert_eq!(da.max_da_tx_size(), None);
        assert_eq!(da.max_da_block_size(), None);
    }

    #[test]
    fn test_da_constrained() {
        let config = OpBuilderConfig::default();
        assert!(config.constrained_da_config().is_none());
    }

    #[test]
    fn test_gas_limit() {
        let gas_limit = OpGasLimitConfig::default();
        assert_eq!(gas_limit.gas_limit(), None);
        gas_limit.set_gas_limit(50000);
        assert_eq!(gas_limit.gas_limit(), Some(50000));
        gas_limit.set_gas_limit(0);
        assert_eq!(gas_limit.gas_limit(), None);
    }
}
