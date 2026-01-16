//! Span Batch Element

use crate::SingleBatch;
use alloc::vec::Vec;
use alloy_primitives::Bytes;

/// MAX_SPAN_BATCH_ELEMENTS is the maximum number of blocks, transactions in total,
/// or transaction per block allowed in a span batch.
pub const MAX_SPAN_BATCH_ELEMENTS: u64 = 10_000_000;

/// A single batch element is similar to the [`SingleBatch`] type
/// but does not contain the parent hash and epoch hash since spans
/// do not contain this data for every block in the span.
#[derive(Debug, Default, Clone, PartialEq, Eq)]
pub struct SpanBatchElement {
    /// The epoch number of the L1 block
    pub epoch_num: u64,
    /// The timestamp of the L2 block
    pub timestamp: u64,
    /// The transactions in the L2 block
    pub transactions: Vec<Bytes>,
}

impl From<SingleBatch> for SpanBatchElement {
    fn from(batch: SingleBatch) -> Self {
        Self {
            epoch_num: batch.epoch_num,
            timestamp: batch.timestamp,
            transactions: batch.transactions,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use proptest::{collection::vec, prelude::any, proptest};

    proptest! {
        #[test]
        fn test_span_batch_element_from_single_batch(epoch_num in 0u64..u64::MAX, timestamp in 0u64..u64::MAX, transactions in vec(any::<Bytes>(), 0..100)) {
            let single_batch = SingleBatch {
                epoch_num,
                timestamp,
                transactions: transactions.clone(),
                ..Default::default()
            };

            let span_batch_element: SpanBatchElement = single_batch.into();

            assert_eq!(span_batch_element.epoch_num, epoch_num);
            assert_eq!(span_batch_element.timestamp, timestamp);
            assert_eq!(span_batch_element.transactions, transactions);
        }
    }
}
