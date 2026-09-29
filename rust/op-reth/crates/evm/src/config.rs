pub use alloy_op_evm::{
    spec as revm_spec, spec_by_timestamp_after_bedrock as revm_spec_by_timestamp_after_bedrock,
};
use op_alloy_rpc_types_engine::OpFlashblockPayloadBase;
#[cfg(feature = "rpc")]
use reth_evm::ConfigureEvm;
#[cfg(feature = "rpc")]
use reth_primitives_traits::HeaderTy;
#[cfg(feature = "rpc")]
use reth_rpc_eth_api::helpers::pending_block::PendingEnvBuilder;
use revm::primitives::{Address, B256, Bytes};

/// Default OP Stack block time in seconds.
#[cfg(feature = "rpc")]
pub const DEFAULT_OP_BLOCK_TIME: u64 = 2;

/// Context relevant for execution of a next block w.r.t OP.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OpNextBlockEnvAttributes {
    /// The timestamp of the next block.
    pub timestamp: u64,
    /// The suggested fee recipient for the next block.
    pub suggested_fee_recipient: Address,
    /// The randomness value for the next block.
    pub prev_randao: B256,
    /// Block gas limit.
    pub gas_limit: u64,
    /// The parent beacon block root.
    pub parent_beacon_block_root: Option<B256>,
    /// Encoded EIP-1559 parameters to include into block's `extra_data` field.
    pub extra_data: Bytes,
}

#[cfg(feature = "rpc")]
impl OpNextBlockEnvAttributes {
    /// Builds pending-block attributes using the configured block time.
    fn build_pending_env_with_block_time<H: alloy_consensus::BlockHeader>(
        parent: &crate::SealedHeader<H>,
        block_overrides: Option<&alloy_rpc_types_eth::BlockOverrides>,
        block_time: u64,
    ) -> Self {
        let mut attributes = Self {
            timestamp: parent.timestamp().saturating_add(block_time),
            suggested_fee_recipient: parent.beneficiary(),
            prev_randao: B256::random(),
            gas_limit: parent.gas_limit(),
            parent_beacon_block_root: parent.parent_beacon_block_root(),
            extra_data: parent.extra_data().clone(),
        };

        // Only the beacon root override must be applied here: it is consumed during EVM
        // environment construction. All other `BlockOverrides` fields are applied directly
        // to the constructed environment by the caller, matching the upstream
        // `NextBlockEnvAttributes::build_pending_env` behavior.
        if attributes.parent_beacon_block_root.is_some() &&
            let Some(beacon_root) = block_overrides.and_then(|overrides| overrides.beacon_root)
        {
            attributes.parent_beacon_block_root = Some(beacon_root);
        }

        attributes
    }
}

/// Builds OP pending-block attributes using the configured block time.
#[cfg(feature = "rpc")]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct OpPendingEnvBuilder {
    block_time: u64,
}

#[cfg(feature = "rpc")]
impl OpPendingEnvBuilder {
    /// Creates a pending environment builder for the given block time in seconds.
    pub const fn new(block_time: u64) -> Self {
        Self { block_time }
    }
}

#[cfg(feature = "rpc")]
impl Default for OpPendingEnvBuilder {
    fn default() -> Self {
        Self::new(DEFAULT_OP_BLOCK_TIME)
    }
}

#[cfg(feature = "rpc")]
impl<Evm> PendingEnvBuilder<Evm> for OpPendingEnvBuilder
where
    Evm: ConfigureEvm<NextBlockEnvCtx = OpNextBlockEnvAttributes>,
    HeaderTy<Evm::Primitives>: alloy_consensus::BlockHeader,
{
    fn pending_env_attributes(
        &self,
        parent: &crate::SealedHeader<HeaderTy<Evm::Primitives>>,
        block_overrides: Option<&alloy_rpc_types_eth::BlockOverrides>,
    ) -> Result<Evm::NextBlockEnvCtx, reth_rpc_eth_types::EthApiError> {
        Ok(OpNextBlockEnvAttributes::build_pending_env_with_block_time(
            parent,
            block_overrides,
            self.block_time,
        ))
    }
}

#[cfg(feature = "rpc")]
impl<H: alloy_consensus::BlockHeader> reth_rpc_eth_api::helpers::pending_block::BuildPendingEnv<H>
    for OpNextBlockEnvAttributes
{
    /// UPSTREAM-MIRROR(copy): reth@rev:4553cf1
    /// `reth_rpc_eth_api::helpers::pending_block::NextBlockEnvAttributes::build_pending_env`
    ///
    /// Copies upstream pending-environment defaults for the OP attribute type, except that it uses
    /// the default OP block time and `parent_beacon_block_root` carries the parent's actual root
    /// where upstream zeroes it.
    fn build_pending_env(
        parent: &crate::SealedHeader<H>,
        block_overrides: Option<&alloy_rpc_types_eth::BlockOverrides>,
    ) -> Self {
        Self::build_pending_env_with_block_time(parent, block_overrides, DEFAULT_OP_BLOCK_TIME)
    }
}

impl From<OpFlashblockPayloadBase> for OpNextBlockEnvAttributes {
    fn from(base: OpFlashblockPayloadBase) -> Self {
        Self {
            timestamp: base.timestamp,
            suggested_fee_recipient: base.fee_recipient,
            prev_randao: base.prev_randao,
            gas_limit: base.gas_limit,
            parent_beacon_block_root: Some(base.parent_beacon_block_root),
            extra_data: base.extra_data,
        }
    }
}

#[cfg(all(test, feature = "rpc"))]
mod tests {
    use super::*;
    use alloy_consensus::Header;
    use reth_primitives_traits::SealedHeader;
    use reth_rpc_eth_api::helpers::pending_block::BuildPendingEnv;

    #[test]
    fn pending_env_uses_default_op_block_time() {
        let parent = SealedHeader::seal_slow(Header { timestamp: 100, ..Default::default() });

        let attributes = OpNextBlockEnvAttributes::build_pending_env(&parent, None);

        assert_eq!(attributes.timestamp, 102);
    }

    #[test]
    fn pending_env_uses_configured_block_time() {
        let parent = SealedHeader::seal_slow(Header { timestamp: 100, ..Default::default() });

        let attributes =
            OpNextBlockEnvAttributes::build_pending_env_with_block_time(&parent, None, 1);

        assert_eq!(attributes.timestamp, 101);
    }
}
