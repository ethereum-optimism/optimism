//! Additional configuration for the OP builder

use alloy_consensus::BlockHeader;
use alloy_rpc_types_engine::PayloadId;
use reth_chainspec::EthChainSpec;
use reth_optimism_forks::OpHardforks;
use reth_optimism_txpool::interop::InteropFailsafe;
use std::{
    collections::VecDeque,
    sync::{
        Arc, Mutex,
        atomic::{AtomicBool, AtomicU64, Ordering},
    },
};

/// Inputs available to a producer's base-fee policy.
#[derive(Clone, Copy)]
pub struct BaseFeePolicyInput<'a> {
    /// Parent block header.
    pub parent: &'a dyn BlockHeader,
    /// Timestamp of the block being built.
    pub next_timestamp: u64,
    /// Base fee selected by the legacy Jovian EIP-1559 algorithm.
    pub legacy_base_fee: u64,
}

impl core::fmt::Debug for BaseFeePolicyInput<'_> {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("BaseFeePolicyInput")
            .field("parent_number", &self.parent.number())
            .field("next_timestamp", &self.next_timestamp)
            .field("legacy_base_fee", &self.legacy_base_fee)
            .finish()
    }
}

/// Error returned when a producer cannot select a base fee.
#[derive(Debug, Clone, thiserror::Error)]
#[error("{message}")]
pub struct BaseFeePolicyError {
    message: String,
}

impl BaseFeePolicyError {
    /// Creates a policy error from a displayable message.
    pub fn msg(message: impl Into<String>) -> Self {
        Self { message: message.into() }
    }
}

/// Producer-only policy for selecting the base fee of a Lagoon block.
///
/// Verifiers never invoke this policy. They execute with the fee committed in the block's trailing
/// `PostExec` transaction instead.
pub trait BaseFeePolicy: core::fmt::Debug + Send + Sync + 'static {
    /// Selects one immutable base fee for the payload job.
    fn select_base_fee(&self, input: BaseFeePolicyInput<'_>) -> Result<u64, BaseFeePolicyError>;

    /// Returns a provisional fee for pending transaction classification and RPC suggestions.
    ///
    /// Policies with stateful selection may override this to avoid consuming or mutating a payload
    /// decision while serving a quote.
    fn quote_base_fee(&self, input: BaseFeePolicyInput<'_>) -> Result<u64, BaseFeePolicyError> {
        self.select_base_fee(input)
    }
}

/// Compatibility policy that preserves the legacy Jovian EIP-1559 result at Lagoon activation.
#[derive(Debug, Default)]
pub struct JovianBaseFeePolicy;

impl BaseFeePolicy for JovianBaseFeePolicy {
    fn select_base_fee(&self, input: BaseFeePolicyInput<'_>) -> Result<u64, BaseFeePolicyError> {
        Ok(input.legacy_base_fee)
    }
}

/// Quotes the next block's fee using legacy consensus rules before Lagoon and `policy` after it.
pub fn quote_base_fee<ChainSpec, H>(
    policy: &dyn BaseFeePolicy,
    chain_spec: &ChainSpec,
    parent: &H,
    next_timestamp: u64,
) -> Result<u64, BaseFeePolicyError>
where
    ChainSpec: EthChainSpec<Header = H> + OpHardforks,
    H: BlockHeader,
{
    let legacy_base_fee = chain_spec
        .next_block_base_fee(parent, next_timestamp)
        .unwrap_or_else(|| parent.base_fee_per_gas().unwrap_or_default());
    if !chain_spec.is_lagoon_active_at_timestamp(next_timestamp) {
        return Ok(legacy_base_fee);
    }

    policy.quote_base_fee(BaseFeePolicyInput { parent, next_timestamp, legacy_base_fee })
}

type BaseFeeSelections = VecDeque<(PayloadId, Result<u64, BaseFeePolicyError>)>;

/// Bounded cache that keeps a policy selection immutable across retries of one payload job.
#[derive(Debug, Clone, Default)]
pub struct BaseFeeSelectionCache {
    inner: Arc<Mutex<BaseFeeSelections>>,
}

impl BaseFeeSelectionCache {
    const CAPACITY: usize = 64;

    /// Returns the selection already resolved for `payload_id`, or resolves and stores it once.
    pub fn resolve(
        &self,
        payload_id: PayloadId,
        select: impl FnOnce() -> Result<u64, BaseFeePolicyError>,
    ) -> Result<u64, BaseFeePolicyError> {
        let mut entries = self.inner.lock().unwrap_or_else(std::sync::PoisonError::into_inner);
        if let Some((_, selected)) = entries.iter().find(|(id, _)| *id == payload_id) {
            return selected.clone();
        }

        let selected = select();
        if entries.len() == Self::CAPACITY {
            entries.pop_front();
        }
        entries.push_back((payload_id, selected.clone()));
        selected
    }
}

/// Settings for the OP builder.
#[derive(Debug, Clone)]
pub struct OpBuilderConfig {
    /// Data availability configuration for the OP builder.
    pub da_config: OpDAConfig,
    /// Gas limit configuration for the OP builder.
    pub gas_limit_config: OpGasLimitConfig,
    /// Producer-only policy used to select the base fee for normal Lagoon payloads.
    pub base_fee_policy: Arc<dyn BaseFeePolicy>,
    /// Selections shared by retries of the same payload job.
    pub base_fee_selection_cache: BaseFeeSelectionCache,
    /// Local SDM refund production operator opt-in. Shared with the admin RPC.
    pub operator_sdm_opt_in: OperatorSdmOptIn,
    /// Interop failsafe gate. Set by the interop filter client; read by the builder to exclude
    /// interop txs from blocks while it is enabled.
    pub interop_failsafe: InteropFailsafe,
    /// Maximum cumulative uncompressed (EIP-2718 encoded) block size in bytes.
    ///
    /// `None` disables the limit (the historical behavior). When set, the payload builder stops
    /// pulling mempool transactions once including the next one would push the block's total
    /// EIP-2718 encoded transaction size past this value. This bounds the size of the
    /// `engine_getPayload` response so it does not exceed the limits assumed by consensus-layer
    /// clients (e.g. the common 10 MiB JSON payload cap). op-geth enforces an equivalent but
    /// non-configurable cap via `params.MaxBlockSize`.
    pub max_uncompressed_block_size: Option<u64>,
}

impl Default for OpBuilderConfig {
    fn default() -> Self {
        Self {
            da_config: OpDAConfig::default(),
            gas_limit_config: OpGasLimitConfig::default(),
            base_fee_policy: Arc::new(JovianBaseFeePolicy),
            base_fee_selection_cache: BaseFeeSelectionCache::default(),
            operator_sdm_opt_in: OperatorSdmOptIn::default(),
            interop_failsafe: InteropFailsafe::default(),
            max_uncompressed_block_size: None,
        }
    }
}

impl OpBuilderConfig {
    /// Creates a new OP builder configuration with the given data availability configuration.
    pub fn new(da_config: OpDAConfig, gas_limit_config: OpGasLimitConfig) -> Self {
        Self {
            da_config,
            gas_limit_config,
            base_fee_policy: Arc::new(JovianBaseFeePolicy),
            base_fee_selection_cache: BaseFeeSelectionCache::default(),
            operator_sdm_opt_in: OperatorSdmOptIn::default(),
            interop_failsafe: InteropFailsafe::default(),
            max_uncompressed_block_size: None,
        }
    }

    /// Replaces the producer's base-fee policy.
    pub fn with_base_fee_policy(mut self, policy: Arc<dyn BaseFeePolicy>) -> Self {
        self.base_fee_policy = policy;
        self
    }

    /// Returns the Data Availability configuration for the OP builder, if it has configured
    /// constraints.
    pub fn constrained_da_config(&self) -> Option<&OpDAConfig> {
        if self.da_config.is_empty() { None } else { Some(&self.da_config) }
    }
}

/// Shareable operator opt-in flag for SDM `PostExec` production.
///
/// `false` on construction. The admin RPC writes; the payload builder reads. The protocol gate
/// (chain spec Interop activation) is checked separately; both must be true to actually produce.
#[derive(Debug, Clone, Default)]
pub struct OperatorSdmOptIn {
    inner: Arc<AtomicBool>,
}

impl OperatorSdmOptIn {
    /// Returns the current opt-in state.
    pub fn enabled(&self) -> bool {
        self.inner.load(Ordering::Acquire)
    }

    /// Sets the opt-in state.
    pub fn set(&self, enabled: bool) {
        self.inner.store(enabled, Ordering::Release);
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
    use alloy_consensus::Header;
    use reth_optimism_chainspec::OpChainSpecBuilder;

    #[derive(Debug)]
    struct TestBaseFeePolicy;

    impl BaseFeePolicy for TestBaseFeePolicy {
        fn select_base_fee(
            &self,
            _input: BaseFeePolicyInput<'_>,
        ) -> Result<u64, BaseFeePolicyError> {
            Ok(999)
        }

        fn quote_base_fee(
            &self,
            _input: BaseFeePolicyInput<'_>,
        ) -> Result<u64, BaseFeePolicyError> {
            Ok(777)
        }
    }

    #[test]
    fn quote_base_fee_uses_policy_only_at_lagoon() {
        let parent = Header {
            timestamp: 1,
            base_fee_per_gas: Some(100),
            gas_limit: 30_000_000,
            ..Default::default()
        };
        let pre_lagoon =
            OpChainSpecBuilder::default().chain(10.into()).genesis(Default::default()).build();
        let expected_legacy =
            pre_lagoon.next_block_base_fee(&parent, parent.timestamp).unwrap_or_default();
        assert_eq!(
            quote_base_fee(&TestBaseFeePolicy, &pre_lagoon, &parent, parent.timestamp).unwrap(),
            expected_legacy,
        );

        let lagoon = OpChainSpecBuilder::default()
            .chain(10.into())
            .genesis(Default::default())
            .lagoon_activated()
            .build();
        assert_eq!(
            quote_base_fee(&TestBaseFeePolicy, &lagoon, &parent, parent.timestamp).unwrap(),
            777,
        );
    }

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
