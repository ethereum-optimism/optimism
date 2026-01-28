use crate::{CrossSafetyError, event::ChainEvent};
use alloy_primitives::ChainId;
use kona_protocol::BlockInfo;
use kona_supervisor_storage::CrossChainSafetyProvider;
use op_alloy_consensus::interop::SafetyLevel;

/// Defines the logic for promoting a block to a specific [`SafetyLevel`].
///
/// Each implementation handles:
/// - Which safety level it promotes to
/// - Its required lower bound
/// - Updating state and generating the corresponding [`ChainEvent`]
pub trait SafetyPromoter {
    /// Target safety level this promoter upgrades to.
    fn target_level(&self) -> SafetyLevel;

    /// Required lower bound level for promotion eligibility.
    fn lower_bound_level(&self) -> SafetyLevel;

    /// Performs the promotion by updating state and returning the event to broadcast.
    fn update_and_emit_event(
        &self,
        provider: &dyn CrossChainSafetyProvider,
        chain_id: ChainId,
        block: &BlockInfo,
    ) -> Result<ChainEvent, CrossSafetyError>;
}
