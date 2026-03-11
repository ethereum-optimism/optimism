//! Attributes construction: transforming batches into payload attributes.

use kona_derive::{
    AttributesBuilder, ChainProvider, DataAvailabilityProvider, L2ChainProvider, PipelineError,
    PipelineResult, ResetError,
};
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use tracing::info;

use crate::Pipeline;

impl<CP, DAP, L2P, AB> Pipeline<CP, DAP, L2P, AB>
where
    CP: ChainProvider + Send,
    DAP: DataAvailabilityProvider + Send,
    L2P: L2ChainProvider + Send,
    AB: AttributesBuilder + Send,
{
    /// Returns the next `OpAttributesWithParent` for the given parent block.
    ///
    /// This is the top-level method that:
    /// 1. Loads a pending batch (validating it if needed)
    /// 2. Transforms the batch into payload attributes
    /// 3. Returns the populated attributes
    pub(crate) async fn next_attributes(
        &mut self,
        parent: L2BlockInfo,
    ) -> PipelineResult<OpAttributesWithParent> {
        // Load a batch if we don't have one pending.
        if self.pending_batch.is_none() {
            let batch = self.validate_and_get_batch(parent).await?;
            self.pending_batch = Some(batch);
            self.is_last_in_span = self.single_batch_buffer.is_empty();
        }

        let batch = self
            .pending_batch
            .take()
            .ok_or(PipelineError::Eof.temp())?;

        // Sanity check parent hash.
        if batch.parent_hash != parent.block_info.hash {
            return Err(ResetError::BadParentHash(batch.parent_hash, parent.block_info.hash).into());
        }

        // Sanity check timestamp.
        let expected_timestamp = parent.block_info.timestamp + self.cfg.block_time;
        if batch.timestamp != expected_timestamp {
            return Err(ResetError::BadTimestamp(batch.timestamp, expected_timestamp).into());
        }

        // Build payload attributes.
        let tx_count = batch.transactions.len();
        let mut attributes = self
            .attributes_builder
            .prepare_payload_attributes(parent, batch.epoch())
            .await?;

        attributes.no_tx_pool = Some(true);
        match attributes.transactions {
            Some(ref mut txs) => txs.extend(batch.transactions),
            None => {
                if !batch.transactions.is_empty() {
                    attributes.transactions = Some(batch.transactions);
                }
            }
        }

        info!(
            target: "karst",
            txs = tx_count,
            timestamp = batch.timestamp,
            "generated attributes in payload queue",
        );

        let origin = self.l1_origin.ok_or(PipelineError::MissingOrigin.crit())?;
        let populated = OpAttributesWithParent::new(
            attributes,
            parent,
            Some(origin),
            self.is_last_in_span,
        );

        // Clear pending state (batch already consumed by take() above).
        self.is_last_in_span = false;

        Ok(populated)
    }
}
