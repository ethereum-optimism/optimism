//! The core `Pipeline` struct and public API.

use alloc::{collections::VecDeque, sync::Arc, vec::Vec};
use core::fmt::Debug;

use kona_derive::{
    AttributesBuilder, ChainProvider, DataAvailabilityProvider, L2ChainProvider, PipelineError,
    PipelineErrorKind, PipelineResult,
};
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{
    BatchReader, BlockInfo, Channel, Frame, L2BlockInfo, OpAttributesWithParent, SingleBatch,
};

/// Result of a single pipeline step.
#[derive(Debug, PartialEq, Eq)]
pub enum StepResult {
    /// Payload attributes were prepared and buffered.
    PreparedAttributes,
    /// The L1 origin was advanced.
    AdvancedOrigin,
    /// Origin advance failed.
    OriginAdvanceErr(PipelineErrorKind),
    /// Step failed with an error.
    StepFailed(PipelineErrorKind),
}

/// A flat, inlined derivation pipeline for the OP Stack (Holocene+ only).
///
/// All derivation stages are collapsed into fields on this struct. Methods
/// implement each stage's logic directly, without trait-based stage composition.
#[derive(Debug)]
pub struct Pipeline<CP, DAP, L2P, AB> {
    // === Config ===
    pub(crate) cfg: Arc<RollupConfig>,

    // === L1 Traversal state ===
    /// The current L1 origin block.
    pub(crate) l1_origin: Option<BlockInfo>,
    /// Whether the current L1 origin has been consumed by the retrieval step.
    pub(crate) l1_traversal_done: bool,
    /// The current system config, updated from L1 receipts.
    pub(crate) system_config: SystemConfig,

    // === L1 Retrieval state ===
    /// The current L1 block being used for data retrieval.
    pub(crate) retrieval_block: Option<BlockInfo>,

    // === Frame Queue state ===
    pub(crate) frame_queue: VecDeque<Frame>,

    // === Channel Assembler state (Holocene-only, no ChannelBank) ===
    pub(crate) channel: Option<Channel>,

    // === Channel Reader state ===
    pub(crate) batch_reader: Option<BatchReader>,

    // === Batch Stream state (span batch decomposition) ===
    pub(crate) single_batch_buffer: VecDeque<SingleBatch>,

    // === Batch Validator state ===
    /// The L1 origin tracked by the batch validator (may differ from pipeline origin).
    pub(crate) batch_origin: Option<BlockInfo>,
    /// A window of L1 blocks used for batch validation.
    pub(crate) l1_blocks: Vec<BlockInfo>,

    // === Attributes Queue state ===
    pub(crate) pending_batch: Option<SingleBatch>,
    pub(crate) is_last_in_span: bool,

    // === Output buffer ===
    pub(crate) prepared: VecDeque<OpAttributesWithParent>,

    // === External providers ===
    pub(crate) chain_provider: CP,
    pub(crate) da_provider: DAP,
    pub(crate) l2_provider: L2P,
    pub(crate) attributes_builder: AB,
}

impl<CP, DAP, L2P, AB> Pipeline<CP, DAP, L2P, AB>
where
    CP: ChainProvider + Send,
    DAP: DataAvailabilityProvider + Send,
    L2P: L2ChainProvider + Send,
    AB: AttributesBuilder + Send,
{
    /// Creates a new pipeline with the given configuration and providers.
    pub fn new(
        cfg: Arc<RollupConfig>,
        chain_provider: CP,
        da_provider: DAP,
        l2_provider: L2P,
        attributes_builder: AB,
    ) -> Self {
        Self {
            cfg,
            l1_origin: None,
            l1_traversal_done: false,
            system_config: SystemConfig::default(),
            retrieval_block: None,
            frame_queue: VecDeque::new(),
            channel: None,
            batch_reader: None,
            single_batch_buffer: VecDeque::new(),
            batch_origin: None,
            l1_blocks: Vec::new(),
            pending_batch: None,
            is_last_in_span: false,
            prepared: VecDeque::new(),
            chain_provider,
            da_provider,
            l2_provider,
            attributes_builder,
        }
    }

    /// Steps through the derivation pipeline once.
    ///
    /// On success, buffers a new `OpAttributesWithParent`. On EOF from the
    /// attributes stage, advances the L1 origin instead.
    pub async fn step(&mut self, cursor: L2BlockInfo) -> StepResult {
        match self.next_attributes(cursor).await {
            Ok(attrs) => {
                self.prepared.push_back(attrs);
                StepResult::PreparedAttributes
            }
            Err(PipelineErrorKind::Temporary(PipelineError::Eof)) => {
                if let Err(e) = self.advance_origin().await {
                    return StepResult::OriginAdvanceErr(e);
                }
                StepResult::AdvancedOrigin
            }
            Err(e) => StepResult::StepFailed(e),
        }
    }

    /// Resets the pipeline to a known state.
    ///
    /// This clears all intermediate state and sets the L1 origin and system config.
    pub async fn reset(
        &mut self,
        l1_origin: BlockInfo,
        l2_safe_head: L2BlockInfo,
    ) -> PipelineResult<()> {
        // Fetch system config for the L2 safe head.
        let system_config = self
            .l2_provider
            .system_config_by_number(l2_safe_head.block_info.number, Arc::clone(&self.cfg))
            .await
            .map_err(Into::into)?;

        // Reset L1 traversal.
        self.l1_origin = Some(l1_origin);
        self.l1_traversal_done = false;
        self.system_config = system_config;

        // Reset L1 retrieval.
        self.retrieval_block = Some(l1_origin);

        // Reset frame queue.
        self.frame_queue.clear();

        // Reset channel assembler.
        self.channel = None;

        // Reset channel reader.
        self.batch_reader = None;

        // Reset batch stream.
        self.single_batch_buffer.clear();

        // Reset batch validator.
        self.batch_origin = Some(l1_origin);
        self.l1_blocks.clear();
        self.l1_blocks.push(l1_origin);

        // Reset attributes queue.
        self.pending_batch = None;
        self.is_last_in_span = false;

        Ok(())
    }

    /// Flushes the current in-progress channel, discarding partial data.
    pub fn flush_channel(&mut self) {
        self.batch_reader = None;
        self.channel = None;
        self.single_batch_buffer.clear();
        self.pending_batch = None;
    }

    /// Returns the current L1 origin block, if set.
    pub const fn origin(&self) -> Option<BlockInfo> {
        self.l1_origin
    }

    /// Peeks at the next prepared attributes without consuming them.
    pub fn peek(&self) -> Option<&OpAttributesWithParent> {
        self.prepared.front()
    }

    /// Consumes and returns the next prepared attributes.
    pub fn pop_attributes(&mut self) -> Option<OpAttributesWithParent> {
        self.prepared.pop_front()
    }
}
