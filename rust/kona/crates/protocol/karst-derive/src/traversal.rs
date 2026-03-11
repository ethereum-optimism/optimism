//! L1 traversal and origin advancing.

use kona_derive::{
    AttributesBuilder, ChainProvider, DataAvailabilityProvider, L2ChainProvider, PipelineError,
    PipelineResult, ResetError,
};
use kona_protocol::BlockInfo;
use tracing::{error, info, warn};

use crate::Pipeline;

impl<CP, DAP, L2P, AB> Pipeline<CP, DAP, L2P, AB>
where
    CP: ChainProvider + Send,
    DAP: DataAvailabilityProvider + Send,
    L2P: L2ChainProvider + Send,
    AB: AttributesBuilder + Send,
{
    /// Advances the L1 origin to the next block.
    ///
    /// Fetches the next L1 block, checks for reorgs, and updates the system config.
    pub(crate) async fn advance_origin(&mut self) -> PipelineResult<()> {
        let block = match self.l1_origin {
            Some(block) => block,
            None => {
                warn!(target: "karst", "Missing current block, can't advance origin.");
                return Err(PipelineError::Eof.temp());
            }
        };

        let next_l1_origin = self
            .chain_provider
            .block_info_by_number(block.number + 1)
            .await
            .map_err(Into::into)?;

        // Check for reorgs.
        if block.hash != next_l1_origin.parent_hash {
            return Err(ResetError::ReorgDetected(block.hash, next_l1_origin.parent_hash).into());
        }

        // Fetch receipts and update system config.
        let receipts = self
            .chain_provider
            .receipts_by_hash(next_l1_origin.hash)
            .await
            .map_err(Into::into)?;

        let addr = self.cfg.l1_system_config_address;
        let ecotone_active = self.cfg.is_ecotone_active(next_l1_origin.timestamp);
        match self
            .system_config
            .update_with_receipts(&receipts[..], addr, ecotone_active)
        {
            Ok(true) => {
                info!(target: "karst", "System config updated at block {}.", next_l1_origin.number);
            }
            Ok(false) => { /* No update applied */ }
            Err(err) => {
                error!(target: "karst", ?err, "Failed to update system config at block {}", next_l1_origin.number);
                return Err(PipelineError::SystemConfigUpdate(err).crit());
            }
        }

        // Update origin.
        self.l1_origin = Some(next_l1_origin);
        self.l1_traversal_done = false;

        // Reset retrieval state for the new origin.
        self.retrieval_block = None;
        self.da_provider.clear();

        Ok(())
    }

    /// Returns the next L1 block for data retrieval, consuming it.
    ///
    /// Can only be called once per origin; subsequent calls return EOF until
    /// `advance_origin()` is called.
    pub(crate) fn next_l1_block(&mut self) -> PipelineResult<BlockInfo> {
        if self.l1_traversal_done {
            return Err(PipelineError::Eof.temp());
        }
        self.l1_traversal_done = true;
        self.l1_origin.ok_or(PipelineError::MissingOrigin.crit())
    }
}
