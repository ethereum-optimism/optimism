pub use alloy_op_evm::{
    spec as revm_spec, spec_by_timestamp_after_bedrock as revm_spec_by_timestamp_after_bedrock,
};
use op_alloy_rpc_types_engine::OpFlashblockPayloadBase;
use revm::primitives::{Address, B256, Bytes};

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
impl<H: alloy_consensus::BlockHeader> reth_rpc_eth_api::helpers::pending_block::BuildPendingEnv<H>
    for OpNextBlockEnvAttributes
{
    fn build_pending_env(parent: &crate::SealedHeader<H>) -> Self {
        Self {
            timestamp: parent.timestamp().saturating_add(12),
            suggested_fee_recipient: parent.beneficiary(),
            prev_randao: B256::random(),
            gas_limit: parent.gas_limit(),
            parent_beacon_block_root: parent.parent_beacon_block_root(),
            extra_data: parent.extra_data().clone(),
        }
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
