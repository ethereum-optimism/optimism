//! Test utilities for the protocol crate.

use alloc::{boxed::Box, format, string::String, sync::Arc, vec::Vec};
use alloy_primitives::hex;
use async_trait::async_trait;
use op_alloy_consensus::OpBlock;
use spin::Mutex;
use tracing::{Event, Level, Subscriber};
use tracing_subscriber::{Layer, layer::Context};

use crate::{
    BatchValidationProvider, L1BlockInfoBedrock, L1BlockInfoEcotone, L1BlockInfoIsthmus,
    L2BlockInfo,
};

/// Raw encoded bedrock L1 block info transaction.
pub const RAW_BEDROCK_INFO_TX: [u8; L1BlockInfoBedrock::L1_INFO_TX_LEN] = hex!(
    "015d8eb9000000000000000000000000000000000000000000000000000000000117c4eb0000000000000000000000000000000000000000000000000000000065280377000000000000000000000000000000000000000000000000000000026d05d953392012032675be9f94aae5ab442de73c5f4fb1bf30fa7dd0d2442239899a40fc00000000000000000000000000000000000000000000000000000000000000040000000000000000000000006887246668a3b87f54deb3b94ba47a6f63f3298500000000000000000000000000000000000000000000000000000000000000bc00000000000000000000000000000000000000000000000000000000000a6fe0"
);

/// Raw encoded ecotone L1 block info transaction.
pub const RAW_ECOTONE_INFO_TX: [u8; L1BlockInfoEcotone::L1_INFO_TX_LEN] = hex!(
    "440a5e2000000558000c5fc5000000000000000500000000661c277300000000012bec20000000000000000000000000000000000000000000000000000000026e9f109900000000000000000000000000000000000000000000000000000000000000011c4c84c50740386c7dc081efddd644405f04cde73e30a2e381737acce9f5add30000000000000000000000006887246668a3b87f54deb3b94ba47a6f63f32985"
);

/// Raw encoded isthmus L1 block info transaction.
pub const RAW_ISTHMUS_INFO_TX: [u8; L1BlockInfoIsthmus::L1_INFO_TX_LEN] = hex!(
    "098999be00000558000c5fc5000000000000000500000000661c277300000000012bec20000000000000000000000000000000000000000000000000000000026e9f109900000000000000000000000000000000000000000000000000000000000000011c4c84c50740386c7dc081efddd644405f04cde73e30a2e381737acce9f5add30000000000000000000000006887246668a3b87f54deb3b94ba47a6f63f329850000abcd000000000000dcba"
);

/// An error for implementations of the [`BatchValidationProvider`] trait.
#[derive(Debug, thiserror::Error)]
pub enum TestBatchValidatorError {
    /// The block was not found.
    #[error("Block not found")]
    BlockNotFound,
    /// The L2 block was not found.
    #[error("L2 Block not found")]
    L2BlockNotFound,
}

/// An [`TestBatchValidator`] implementation for testing.
#[derive(Default, Debug, Clone)]
pub struct TestBatchValidator {
    /// Blocks
    pub blocks: Vec<L2BlockInfo>,
    /// Short circuit the block return to be the first block.
    pub short_circuit: bool,
    /// Blocks
    pub op_blocks: Vec<OpBlock>,
}

impl TestBatchValidator {
    /// Creates a new [`TestBatchValidator`] with the given origin and batches.
    pub const fn new(blocks: Vec<L2BlockInfo>, op_blocks: Vec<OpBlock>) -> Self {
        Self { blocks, short_circuit: false, op_blocks }
    }
}

#[async_trait]
impl BatchValidationProvider for TestBatchValidator {
    type Error = TestBatchValidatorError;

    async fn l2_block_info_by_number(&mut self, number: u64) -> Result<L2BlockInfo, Self::Error> {
        if self.short_circuit {
            return self
                .blocks
                .first()
                .copied()
                .ok_or_else(|| TestBatchValidatorError::BlockNotFound);
        }
        self.blocks
            .iter()
            .find(|b| b.block_info.number == number)
            .cloned()
            .ok_or_else(|| TestBatchValidatorError::BlockNotFound)
    }

    async fn block_by_number(&mut self, number: u64) -> Result<OpBlock, Self::Error> {
        self.op_blocks
            .iter()
            .find(|p| p.header.number == number)
            .cloned()
            .ok_or_else(|| TestBatchValidatorError::L2BlockNotFound)
    }
}

/// The storage for the collected traces.
#[derive(Debug, Default, Clone)]
pub struct TraceStorage(pub Arc<Mutex<Vec<(Level, String)>>>);

impl TraceStorage {
    /// Returns the items in the storage that match the specified level.
    pub fn get_by_level(&self, level: Level) -> Vec<String> {
        self.0
            .lock()
            .iter()
            .filter_map(|(l, message)| if *l == level { Some(message.clone()) } else { None })
            .collect()
    }

    /// Returns if the storage is empty.
    pub fn is_empty(&self) -> bool {
        self.0.lock().is_empty()
    }
}

/// A subscriber layer that collects traces and their log levels.
#[derive(Debug, Default)]
pub struct CollectingLayer {
    /// The storage for the collected traces.
    pub storage: TraceStorage,
}

impl CollectingLayer {
    /// Creates a new collecting layer with the specified storage.
    pub const fn new(storage: TraceStorage) -> Self {
        Self { storage }
    }
}

impl<S: Subscriber> Layer<S> for CollectingLayer {
    fn on_event(&self, event: &Event<'_>, _ctx: Context<'_, S>) {
        let metadata = event.metadata();
        let level = *metadata.level();
        let message = format!("{event:?}");

        let mut storage = self.storage.0.lock();
        storage.push((level, message));
    }
}
