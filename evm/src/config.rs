pub use alloy_op_evm::{
    spec as revm_spec, spec_by_timestamp_after_bedrock as revm_spec_by_timestamp_after_bedrock,
};

use alloy_consensus::BlockHeader;
use revm::primitives::{Address, Bytes, B256};

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
impl<H: BlockHeader> reth_rpc_eth_api::helpers::pending_block::BuildPendingEnv<H>
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
