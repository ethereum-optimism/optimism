//! Sync core of the post-Holocene batch pipeline.
//!
//! Lifted from `stages/batch/batch_stream.rs` (channel-bytes → batches and
//! span batch → singular batches) and the IO-free pieces of
//! `stages/batch/batch_validator.rs` (origin tracking, empty-batch
//! derivation, single-batch verdict mapping).
//!
//! IO-bound checks — the post-Holocene span batch prefix check and the
//! sysconfig walkback — stay in the async stages for now. Phase 3 issues
//! those lookups via `Derivation::NeedSpanBatchOverlap`.

use alloc::{collections::VecDeque, vec::Vec};
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, L2BlockInfo, SingleBatch, SpanBatch, SpanBatchError};

/// Drains a [`SpanBatch`] into [`SingleBatch`]es, appending them to `buffer`.
///
/// Caller already validated the span batch prefix and committed to accepting
/// it. This is the post-Holocene "extract" step, IO-free since the L1 origin
/// list comes from the caller.
pub(crate) fn hydrate_span_batch(
    span: SpanBatch,
    parent: L2BlockInfo,
    l1_origins: &[BlockInfo],
    buffer: &mut VecDeque<SingleBatch>,
) -> Result<(), SpanBatchError> {
    buffer.extend(span.get_singular_batches(l1_origins, parent)?);
    Ok(())
}

/// Outcome of attempting to derive an empty batch when the sequencing window
/// has expired. Returned by [`try_derive_empty_batch`] in the place of the
/// stage's `PipelineResult<SingleBatch>` so the caller controls the error
/// translation.
#[derive(Debug)]
pub(crate) enum EmptyBatchOutcome {
    /// An empty batch was synthesized for the current epoch.
    Generated(SingleBatch),
    /// The sequencing window is still open. Caller returns EOF.
    Eof,
    /// We auto-generated every batch for the current epoch; the caller
    /// advanced into the next epoch and should retry.
    AdvancedEpoch,
}

/// Mirrors `BatchValidator::try_derive_empty_batch` without the `PipelineError`
/// coupling. Returns [`EmptyBatchOutcome`] so the caller can map it to either
/// `PipelineError::Eof` (today) or a structured `Derivation` trace (phase 3).
///
/// `l1_blocks` is the validator's current sliding window of L1 origins. It is
/// mutated to advance through epochs.
pub(crate) fn try_derive_empty_batch(
    cfg: &RollupConfig,
    parent: &L2BlockInfo,
    stage_origin: BlockInfo,
    l1_blocks: &mut Vec<BlockInfo>,
) -> EmptyBatchOutcome {
    let epoch = l1_blocks[0];
    let expiry_epoch = epoch.number + cfg.seq_window_size;
    let force_empty_batches = expiry_epoch <= stage_origin.number;
    let first_of_epoch = epoch.number == parent.l1_origin.number + 1;
    let next_timestamp = parent.block_info.timestamp + cfg.block_time;

    if !force_empty_batches {
        return EmptyBatchOutcome::Eof;
    }

    if l1_blocks.len() < 2 {
        return EmptyBatchOutcome::Eof;
    }

    let next_epoch = l1_blocks[1];

    if next_timestamp < next_epoch.timestamp || first_of_epoch {
        return EmptyBatchOutcome::Generated(SingleBatch {
            parent_hash: parent.block_info.hash,
            epoch_num: epoch.number,
            epoch_hash: epoch.hash,
            timestamp: next_timestamp,
            transactions: Vec::new(),
        });
    }

    // Auto-generated every batch for the current epoch; advance.
    l1_blocks.remove(0);
    EmptyBatchOutcome::AdvancedEpoch
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::{collections::VecDeque, sync::Arc, vec};
    use alloy_eips::BlockNumHash;
    use kona_genesis::RollupConfig;
    use kona_protocol::{BlockInfo, L2BlockInfo, SpanBatch, SpanBatchElement};

    #[test]
    fn hydrate_span_batch_single_block() {
        let span = SpanBatch {
            batches: vec![
                SpanBatchElement { epoch_num: 1, timestamp: 2, ..Default::default() },
                SpanBatchElement { epoch_num: 1, timestamp: 4, ..Default::default() },
            ],
            ..Default::default()
        };
        let l1_origins = [BlockInfo { number: 1, timestamp: 12, ..Default::default() }];
        let mut buffer = VecDeque::new();
        hydrate_span_batch(span, L2BlockInfo::default(), &l1_origins, &mut buffer)
            .expect("hydrate succeeds");
        assert_eq!(buffer.len(), 2);
        assert_eq!(buffer[0].epoch_num, 1);
        assert_eq!(buffer[0].timestamp, 2);
        assert_eq!(buffer[1].timestamp, 4);
    }

    #[test]
    fn empty_batch_eof_inside_window() {
        let cfg =
            Arc::new(RollupConfig { seq_window_size: 10, block_time: 2, ..Default::default() });
        let mut l1_blocks = vec![BlockInfo { number: 1, ..Default::default() }];
        let parent = L2BlockInfo {
            l1_origin: BlockNumHash { number: 1, ..Default::default() },
            ..Default::default()
        };
        let outcome = try_derive_empty_batch(
            &cfg,
            &parent,
            BlockInfo { number: 2, ..Default::default() },
            &mut l1_blocks,
        );
        assert!(matches!(outcome, EmptyBatchOutcome::Eof));
    }

    #[test]
    fn empty_batch_generated_first_of_epoch() {
        let cfg =
            Arc::new(RollupConfig { seq_window_size: 5, block_time: 2, ..Default::default() });
        // origin advanced past expiry → force empty batches.
        let mut l1_blocks = vec![
            BlockInfo { number: 1, timestamp: 0, ..Default::default() },
            BlockInfo { number: 2, timestamp: 10, ..Default::default() },
        ];
        // Parent's l1_origin one block behind the first epoch in l1_blocks →
        // `first_of_epoch` true, must generate.
        let parent = L2BlockInfo {
            l1_origin: BlockNumHash { number: 0, ..Default::default() },
            ..Default::default()
        };
        let outcome = try_derive_empty_batch(
            &cfg,
            &parent,
            BlockInfo { number: 6, ..Default::default() },
            &mut l1_blocks,
        );
        match outcome {
            EmptyBatchOutcome::Generated(batch) => {
                assert_eq!(batch.epoch_num, 1);
                assert!(batch.transactions.is_empty());
            }
            other => panic!("expected Generated, got {other:?}"),
        }
    }
}
