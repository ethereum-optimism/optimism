use crate::{CrossSafetyError, event::ChainEvent, safety_checker::traits::SafetyPromoter};
use alloy_primitives::ChainId;
use kona_protocol::BlockInfo;
use kona_supervisor_storage::CrossChainSafetyProvider;
use op_alloy_consensus::interop::SafetyLevel;

/// CrossUnsafePromoter implements [`SafetyPromoter`] for [`SafetyLevel::CrossUnsafe`]
#[derive(Debug)]
pub struct CrossUnsafePromoter;

impl SafetyPromoter for CrossUnsafePromoter {
    fn target_level(&self) -> SafetyLevel {
        SafetyLevel::CrossUnsafe
    }

    fn lower_bound_level(&self) -> SafetyLevel {
        SafetyLevel::LocalUnsafe
    }

    fn update_and_emit_event(
        &self,
        provider: &dyn CrossChainSafetyProvider,
        chain_id: ChainId,
        block: &BlockInfo,
    ) -> Result<ChainEvent, CrossSafetyError> {
        provider.update_current_cross_unsafe(chain_id, block)?;
        Ok(ChainEvent::CrossUnsafeUpdate { block: *block })
    }
}

/// CrossSafePromoter implements [`SafetyPromoter`] for [`SafetyLevel::CrossSafe`]
#[derive(Debug)]
pub struct CrossSafePromoter;

impl SafetyPromoter for CrossSafePromoter {
    fn target_level(&self) -> SafetyLevel {
        SafetyLevel::CrossSafe
    }

    fn lower_bound_level(&self) -> SafetyLevel {
        SafetyLevel::LocalSafe
    }

    fn update_and_emit_event(
        &self,
        provider: &dyn CrossChainSafetyProvider,
        chain_id: ChainId,
        block: &BlockInfo,
    ) -> Result<ChainEvent, CrossSafetyError> {
        let derived_ref_pair = provider.update_current_cross_safe(chain_id, block)?;
        Ok(ChainEvent::CrossSafeUpdate { derived_ref_pair })
    }
}
