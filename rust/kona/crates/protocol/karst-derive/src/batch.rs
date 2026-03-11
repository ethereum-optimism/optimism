//! Batch processing: span batch decomposition and batch validation.

use alloc::vec::Vec;

use kona_derive::{
    AttributesBuilder, ChainProvider, DataAvailabilityProvider, L2ChainProvider, PipelineError,
    PipelineErrorKind, PipelineResult, ResetError,
};
use kona_protocol::{
    Batch, BatchValidity, BatchWithInclusionBlock, L2BlockInfo, SingleBatch,
};
use tracing::{debug, error, info, warn};

use crate::Pipeline;

impl<CP, DAP, L2P, AB> Pipeline<CP, DAP, L2P, AB>
where
    CP: ChainProvider + Send,
    DAP: DataAvailabilityProvider + Send,
    L2P: L2ChainProvider + Send,
    AB: AttributesBuilder + Send,
{
    /// Pulls a single batch, decomposing span batches if necessary.
    ///
    /// This is the Holocene-only batch stream logic (always active).
    pub(crate) async fn pull_single_batch(
        &mut self,
        parent: L2BlockInfo,
    ) -> PipelineResult<SingleBatch> {
        // If we have buffered single batches, return the next one.
        if let Some(batch) = self.single_batch_buffer.pop_front() {
            return Ok(batch);
        }

        // Pull a batch from the channel reader.
        let origin = self.l1_origin.ok_or(PipelineError::MissingOrigin.crit())?;
        let batch = self.next_batch_from_reader().await?;
        let batch_with_inclusion = BatchWithInclusionBlock::new(origin, batch);

        match batch_with_inclusion.batch {
            Batch::Single(b) => Ok(b),
            Batch::Span(b) => {
                // Validate the span batch prefix.
                let (validity, _) = b
                    .check_batch_prefix(
                        self.cfg.as_ref(),
                        self.l1_blocks.as_ref(),
                        parent,
                        &batch_with_inclusion.inclusion_block,
                        &mut self.l2_provider,
                    )
                    .await;

                match validity {
                    BatchValidity::Accept => {
                        // Decompose span batch into single batches.
                        match b.get_singular_batches(self.l1_blocks.as_ref(), parent) {
                            Ok(singles) => {
                                self.single_batch_buffer.extend(singles);
                                self.single_batch_buffer
                                    .pop_front()
                                    .ok_or(PipelineError::NotEnoughData.temp())
                            }
                            Err(e) => {
                                warn!(target: "karst", "Span batch decomposition failed: {}", e);
                                self.flush_channel();
                                Err(PipelineError::NotEnoughData.temp())
                            }
                        }
                    }
                    BatchValidity::Drop(_) => {
                        self.flush_channel();
                        Err(PipelineError::NotEnoughData.temp())
                    }
                    BatchValidity::Past
                    | BatchValidity::Undecided
                    | BatchValidity::Future => Err(PipelineError::NotEnoughData.temp()),
                }
            }
        }
    }

    /// Returns `true` if the pipeline origin is behind the parent's L1 origin.
    fn origin_behind(&self, parent: &L2BlockInfo) -> bool {
        self.l1_origin
            .is_none_or(|origin| origin.number < parent.l1_origin.number)
    }

    /// Updates the batch validator's view of L1 origin blocks.
    pub(crate) fn update_origins(&mut self, parent: &L2BlockInfo) -> PipelineResult<()> {
        let origin_behind = self.origin_behind(parent);

        if self.batch_origin != self.l1_origin {
            self.batch_origin = self.l1_origin;
            if origin_behind {
                self.l1_blocks.clear();
            } else {
                let origin = self.batch_origin.ok_or(PipelineError::MissingOrigin.crit())?;
                self.l1_blocks.push(origin);
            }
            debug!(
                target: "karst",
                "Advancing batch validator origin to L1 block #{}.{}",
                self.batch_origin.map(|b| b.number).unwrap_or_default(),
                if origin_behind { " (origin behind)" } else { "" }
            );
        }

        // Advance epoch if parent has moved past the first L1 block.
        if !self.l1_blocks.is_empty() && parent.l1_origin.number > self.l1_blocks[0].number {
            for (i, block) in self.l1_blocks.iter().enumerate() {
                if parent.l1_origin.number == block.number {
                    self.l1_blocks.drain(0..i);
                    debug!(target: "karst", "Advancing internal L1 epoch");
                    break;
                }
            }
        }

        Ok(())
    }

    /// Attempts to derive an empty batch when the sequencing window has expired.
    pub(crate) fn try_derive_empty_batch(
        &mut self,
        parent: &L2BlockInfo,
    ) -> PipelineResult<SingleBatch> {
        let epoch = self.l1_blocks[0];

        let stage_origin = self.batch_origin.ok_or(PipelineError::MissingOrigin.crit())?;
        let expiry_epoch = epoch.number + self.cfg.seq_window_size;
        let force_empty_batches = expiry_epoch <= stage_origin.number;
        let first_of_epoch = epoch.number == parent.l1_origin.number + 1;
        let next_timestamp = parent.block_info.timestamp + self.cfg.block_time;

        if !force_empty_batches {
            return Err(PipelineError::Eof.temp());
        }

        if self.l1_blocks.len() < 2 {
            return Err(PipelineError::Eof.temp());
        }

        let next_epoch = self.l1_blocks[1];

        if next_timestamp < next_epoch.timestamp || first_of_epoch {
            info!(target: "karst", "Generating empty batch for epoch #{}", epoch.number);
            return Ok(SingleBatch {
                parent_hash: parent.block_info.hash,
                epoch_num: epoch.number,
                epoch_hash: epoch.hash,
                timestamp: next_timestamp,
                transactions: Vec::new(),
            });
        }

        debug!(
            target: "karst",
            "Advancing batch validator epoch: {}, timestamp: {}, epoch timestamp: {}",
            next_epoch.number, next_timestamp, next_epoch.timestamp
        );
        self.l1_blocks.remove(0);
        Err(PipelineError::Eof.temp())
    }

    /// Validates and returns the next batch for the given parent.
    ///
    /// This combines the batch stream (decomposition) and batch validator logic.
    pub(crate) async fn validate_and_get_batch(
        &mut self,
        parent: L2BlockInfo,
    ) -> PipelineResult<SingleBatch> {
        // Update L1 origin blocks.
        self.update_origins(&parent)?;

        let stage_origin = self.batch_origin.ok_or(PipelineError::MissingOrigin.crit())?;

        // If origin is behind parent, drain to catch up.
        if self.origin_behind(&parent) || parent.l1_origin.number == stage_origin.number {
            self.pull_single_batch(parent).await?;
            return Err(PipelineError::NotEnoughData.temp());
        }

        // Need at least 2 L1 blocks for validation.
        if self.l1_blocks.len() < 2 {
            return Err(PipelineError::MissingOrigin.crit());
        }

        let epoch = self.l1_blocks[0];
        if parent.l1_origin != epoch.id() && parent.l1_origin.number != epoch.number - 1 {
            return Err(PipelineErrorKind::Reset(ResetError::L1OriginMismatch(
                parent.l1_origin.number,
                epoch.number - 1,
            )));
        }

        // Pull the next batch.
        let mut next_batch = match self.pull_single_batch(parent).await {
            Ok(batch) => batch,
            Err(PipelineErrorKind::Temporary(PipelineError::Eof)) => {
                return self.try_derive_empty_batch(&parent);
            }
            Err(e) => return Err(e),
        };
        next_batch.parent_hash = parent.block_info.hash;

        // Validate the batch.
        match next_batch.check_batch(
            self.cfg.as_ref(),
            self.l1_blocks.as_ref(),
            parent,
            &stage_origin,
        ) {
            BatchValidity::Accept => {
                info!(target: "karst", "Found next batch (epoch #{})", next_batch.epoch_num);
                Ok(next_batch)
            }
            BatchValidity::Past => {
                warn!(target: "karst", "Dropping old batch");
                Err(PipelineError::NotEnoughData.temp())
            }
            BatchValidity::Drop(reason) => {
                warn!(target: "karst", "Invalid batch ({}), flushing channel.", reason);
                self.flush_channel();
                Err(PipelineError::NotEnoughData.temp())
            }
            BatchValidity::Undecided => Err(PipelineError::NotEnoughData.temp()),
            BatchValidity::Future => {
                error!(target: "karst", "Future batch detected.");
                Err(PipelineError::InvalidBatchValidity.crit())
            }
        }
    }
}
