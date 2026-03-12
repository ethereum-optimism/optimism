//! The core `Pipeline` struct and public API.

use alloc::{collections::VecDeque, sync::Arc, vec::Vec};
use core::fmt::Debug;
use tracing::info;

use kona_derive::{
    AttributesBuilder, ChainProvider, DataAvailabilityProvider, L2ChainProvider, PipelineError,
    PipelineErrorKind, PipelineResult, ResetError,
};
use kona_genesis::{RollupConfig, SystemConfig};
use kona_protocol::{
    BatchReader, BlockInfo, Channel, Frame, L2BlockInfo, OpAttributesWithParent, SingleBatch,
};


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
            chain_provider,
            da_provider,
            l2_provider,
            attributes_builder,
        }
    }

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

        let batch = self.pending_batch.take().ok_or(PipelineError::Eof.temp())?;

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
        let mut attributes =
            self.attributes_builder.prepare_payload_attributes(parent, batch.epoch()).await?;

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
        let populated =
            OpAttributesWithParent::new(attributes, parent, Some(origin), self.is_last_in_span);

        // Clear pending state (batch already consumed by take() above).
        self.is_last_in_span = false;

        Ok(populated)
    }

    /// Attempts to progress the pipeline by one step.
    ///
    /// Calls `next_attributes` once. On success, returns a single-element vec.
    /// On EOF, advances the L1 origin and returns an empty vec.
    /// On any other error, returns the error for the caller to handle.
    pub async fn step(&mut self, cursor: L2BlockInfo) -> PipelineResult<Vec<OpAttributesWithParent>> {
        match self.next_attributes(cursor).await {
            Ok(attrs) => Ok(alloc::vec![attrs]),
            Err(PipelineErrorKind::Temporary(PipelineError::Eof)) => {
                self.advance_origin().await?;
                Ok(Vec::new())
            }
            Err(e) => Err(e),
        }
    }

    /// Steps the pipeline repeatedly, collecting all available attributes from the
    /// current origin until EOF is reached and the origin is advanced.
    ///
    /// Returns all collected attributes (may be empty if no attributes were available
    /// before EOF). Any non-EOF error is returned immediately.
    pub async fn step_all(&mut self, cursor: L2BlockInfo) -> PipelineResult<Vec<OpAttributesWithParent>> {
        let mut all_attrs = Vec::new();
        loop {
            match self.step(cursor).await {
                Ok(attrs) if attrs.is_empty() => {
                    return Ok(all_attrs);
                }
                Ok(attrs) => {
                    all_attrs.extend(attrs);
                }
                Err(e) => return Err(e),
            }
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
}
