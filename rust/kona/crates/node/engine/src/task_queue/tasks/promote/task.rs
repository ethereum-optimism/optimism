//! A task that promotes the cross-safe head.

use crate::{
    EngineClient, EngineState, EngineTaskExt, PromoteCrossSafeTaskError, SynchronizeTask,
    state::CrossSafePromotion,
};
use async_trait::async_trait;
use derive_more::Constructor;
use kona_genesis::RollupConfig;
use std::sync::Arc;

/// The [`PromoteCrossSafeTask`] moves the engine's cross-safe head to the promoted block and
/// dispatches the forkchoice update that carries it to the execution layer as `safeBlockHash`.
///
/// This is the only enqueueable task that moves the cross-safe head. Constructing one requires a
/// [`CrossSafePromotion`], which only the holder of the engine's unique
/// [`CrossSafePromoter`] can mint.
///
/// [`CrossSafePromoter`]: crate::CrossSafePromoter
#[derive(Debug, Clone, Constructor)]
pub struct PromoteCrossSafeTask<EngineClient_: EngineClient> {
    /// The engine client.
    pub client: Arc<EngineClient_>,
    /// The rollup config.
    pub cfg: Arc<RollupConfig>,
    /// The promotion to apply.
    pub promotion: CrossSafePromotion,
}

#[async_trait]
impl<EngineClient_: EngineClient> EngineTaskExt for PromoteCrossSafeTask<EngineClient_> {
    type Output = ();

    type Error = PromoteCrossSafeTaskError;

    async fn execute(&self, state: &mut EngineState) -> Result<(), PromoteCrossSafeTaskError> {
        SynchronizeTask::promotion(self.client.clone(), self.cfg.clone(), self.promotion)
            .execute(state)
            .await?;

        info!(
            target: "engine",
            hash = %state.sync_state.cross_safe_head().block_info.hash,
            number = state.sync_state.cross_safe_head().block_info.number,
            "Promoted cross-safe head"
        );

        Ok(())
    }
}
