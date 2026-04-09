//!
use alloy_consensus::BlockHeader;
use alloy_op_hardforks::OpHardforks;
use op_revm::OpSpecId;

/// Map the latest active hardfork at the given header to a revm [`OpSpecId`].
pub fn spec(chain_spec: impl OpHardforks, header: impl BlockHeader) -> OpSpecId {
    spec_by_timestamp_after_bedrock(chain_spec, header.timestamp())
}

/// Returns the revm [`OpSpecId`] at the given timestamp.
///
/// # Note
///
/// This is only intended to be used after the Bedrock, when hardforks are activated by
/// timestamp.
pub fn spec_by_timestamp_after_bedrock(chain_spec: impl OpHardforks, timestamp: u64) -> OpSpecId {
    if chain_spec.is_interop_active_at_timestamp(timestamp) {
        OpSpecId::INTEROP
    } else if chain_spec.is_jovian_active_at_timestamp(timestamp) {
        OpSpecId::JOVIAN
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
