use alloy_consensus::BlockHeader;
use op_revm::OpSpecId;
use reth_optimism_forks::OpHardforks;
use revm_primitives::{Address, Bytes, B256};

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

/// Map the latest active hardfork at the given header to a revm [`OpSpecId`].
pub fn revm_spec(chain_spec: impl OpHardforks, header: impl BlockHeader) -> OpSpecId {
    revm_spec_by_timestamp_after_bedrock(chain_spec, header.timestamp())
}

/// Returns the revm [`OpSpecId`] at the given timestamp.
///
/// # Note
///
/// This is only intended to be used after the Bedrock, when hardforks are activated by
/// timestamp.
pub fn revm_spec_by_timestamp_after_bedrock(
    chain_spec: impl OpHardforks,
    timestamp: u64,
) -> OpSpecId {
    if chain_spec.is_interop_active_at_timestamp(timestamp) {
        OpSpecId::INTEROP
    } else if chain_spec.is_isthmus_active_at_timestamp(timestamp) {
        OpSpecId::ISTHMUS
    } else if chain_spec.is_holocene_active_at_timestamp(timestamp) {
        OpSpecId::HOLOCENE
    } else if chain_spec.is_granite_active_at_timestamp(timestamp) {
        OpSpecId::GRANITE
    } else if chain_spec.is_fjord_active_at_timestamp(timestamp) {
        OpSpecId::FJORD
    } else if chain_spec.is_ecotone_active_at_timestamp(timestamp) {
        OpSpecId::ECOTONE
    } else if chain_spec.is_canyon_active_at_timestamp(timestamp) {
        OpSpecId::CANYON
    } else if chain_spec.is_regolith_active_at_timestamp(timestamp) {
        OpSpecId::REGOLITH
    } else {
        OpSpecId::BEDROCK
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Header;
    use reth_chainspec::ChainSpecBuilder;
    use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};

    #[test]
    fn test_revm_spec_by_timestamp_after_merge() {
        #[inline(always)]
        fn op_cs(f: impl FnOnce(OpChainSpecBuilder) -> OpChainSpecBuilder) -> OpChainSpec {
            let cs = ChainSpecBuilder::mainnet().chain(reth_chainspec::Chain::from_id(10)).into();
            f(cs).build()
        }
        assert_eq!(
            revm_spec_by_timestamp_after_bedrock(op_cs(|cs| cs.interop_activated()), 0),
            OpSpecId::INTEROP
        );
        assert_eq!(
            revm_spec_by_timestamp_after_bedrock(op_cs(|cs| cs.isthmus_activated()), 0),
            OpSpecId::ISTHMUS
        );
        assert_eq!(
            revm_spec_by_timestamp_after_bedrock(op_cs(|cs| cs.holocene_activated()), 0),
            OpSpecId::HOLOCENE
        );
        assert_eq!(
            revm_spec_by_timestamp_after_bedrock(op_cs(|cs| cs.granite_activated()), 0),
            OpSpecId::GRANITE
        );
        assert_eq!(
            revm_spec_by_timestamp_after_bedrock(op_cs(|cs| cs.fjord_activated()), 0),
            OpSpecId::FJORD
        );
        assert_eq!(
            revm_spec_by_timestamp_after_bedrock(op_cs(|cs| cs.ecotone_activated()), 0),
            OpSpecId::ECOTONE
        );
        assert_eq!(
            revm_spec_by_timestamp_after_bedrock(op_cs(|cs| cs.canyon_activated()), 0),
            OpSpecId::CANYON
        );
        assert_eq!(
            revm_spec_by_timestamp_after_bedrock(op_cs(|cs| cs.bedrock_activated()), 0),
            OpSpecId::BEDROCK
        );
        assert_eq!(
            revm_spec_by_timestamp_after_bedrock(op_cs(|cs| cs.regolith_activated()), 0),
            OpSpecId::REGOLITH
        );
    }

    #[test]
    fn test_to_revm_spec() {
        #[inline(always)]
        fn op_cs(f: impl FnOnce(OpChainSpecBuilder) -> OpChainSpecBuilder) -> OpChainSpec {
            let cs = ChainSpecBuilder::mainnet().chain(reth_chainspec::Chain::from_id(10)).into();
            f(cs).build()
        }
        assert_eq!(
            revm_spec(op_cs(|cs| cs.isthmus_activated()), Header::default()),
            OpSpecId::ISTHMUS
        );
        assert_eq!(
            revm_spec(op_cs(|cs| cs.holocene_activated()), Header::default()),
            OpSpecId::HOLOCENE
        );
        assert_eq!(
            revm_spec(op_cs(|cs| cs.granite_activated()), Header::default()),
            OpSpecId::GRANITE
        );
        assert_eq!(revm_spec(op_cs(|cs| cs.fjord_activated()), Header::default()), OpSpecId::FJORD);
        assert_eq!(
            revm_spec(op_cs(|cs| cs.ecotone_activated()), Header::default()),
            OpSpecId::ECOTONE
        );
        assert_eq!(
            revm_spec(op_cs(|cs| cs.canyon_activated()), Header::default()),
            OpSpecId::CANYON
        );
        assert_eq!(
            revm_spec(op_cs(|cs| cs.bedrock_activated()), Header::default()),
            OpSpecId::BEDROCK
        );
        assert_eq!(
            revm_spec(op_cs(|cs| cs.regolith_activated()), Header::default()),
            OpSpecId::REGOLITH
        );
    }
}
