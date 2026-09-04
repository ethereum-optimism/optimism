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
    fn build_pending_env(
        parent: &crate::SealedHeader<H>,
        block_overrides: Option<&alloy_rpc_types_eth::BlockOverrides>,
    ) -> Self {
        let mut attributes = Self {
            // OP chains have no slot time and the EL chain spec carries no block time, so the
            // next block's timestamp is unknown here. `parent + 1` is its lower bound for every
            // OP block time, which keeps `pending` from executing under a hardfork that the
            // block the sequencer actually produces next will not have activated.
            timestamp: parent.timestamp().saturating_add(1),
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
    use reth_optimism_chainspec::OP_MAINNET;
    use reth_optimism_forks::{OpHardfork, OpHardforks};
    use reth_primitives_traits::SealedHeader;
    use reth_rpc_eth_api::helpers::pending_block::BuildPendingEnv;

    fn pending_timestamp(parent_timestamp: u64) -> u64 {
        let parent =
            SealedHeader::seal_slow(Header { timestamp: parent_timestamp, ..Default::default() });
        OpNextBlockEnvAttributes::build_pending_env(&parent, None).timestamp
    }

    #[test]
    fn pending_timestamp_advances_past_the_parent() {
        assert!(pending_timestamp(0) > 0);
        assert!(pending_timestamp(u64::MAX - 1) > u64::MAX - 1);
    }

    #[test]
    fn pending_timestamp_does_not_cross_a_fork_boundary_early() {
        // Two blocks before each fork on a 2-second chain: the pending environment must still
        // resolve to the fork that is active for the parent, not the one the fork activates.
        for fork in OpHardfork::VARIANTS {
            let Some(activation) = OP_MAINNET.op_fork_activation(*fork).as_timestamp() else {
                continue;
            };
            let parent_timestamp = activation - 4;

            assert_eq!(
                revm_spec_by_timestamp_after_bedrock(
                    &*OP_MAINNET,
                    pending_timestamp(parent_timestamp)
                ),
                revm_spec_by_timestamp_after_bedrock(&*OP_MAINNET, parent_timestamp),
                "{fork:?} leaks into the pending environment"
            );
        }
    }
}
