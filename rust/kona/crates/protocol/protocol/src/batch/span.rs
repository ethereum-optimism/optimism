//! Span Batch implementation for efficient multi-block L2 transaction batching.
//!
//! Span batches are an advanced batching format that can contain transactions for multiple
//! L2 blocks in a single compressed structure. This provides significant efficiency gains
//! over single batches by amortizing overhead across multiple blocks and enabling
//! sophisticated compression techniques.

use alloc::vec::Vec;
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::FixedBytes;
use kona_genesis::RollupConfig;
use op_alloy_consensus::OpTxType;
use tracing::{info, warn};

use crate::{
    BatchDropReason, BatchType, BatchValidationProvider, BatchValidity, BlockInfo, L2BlockInfo,
    RawSpanBatch, SingleBatch, SpanBatchBits, SpanBatchElement, SpanBatchError, SpanBatchPayload,
    SpanBatchPrefix, SpanBatchTransactions,
};

/// Container for the inputs required to build a span of L2 blocks in derived form.
///
/// A [`SpanBatch`] represents a compressed format for multiple L2 blocks that enables
/// significant space savings compared to individual single batches. The format uses
/// differential encoding, bit packing, and shared data structures to minimize the
/// L1 footprint while maintaining all necessary information for L2 block reconstruction.
///
/// # Compression Techniques
///
/// ## Temporal Compression
/// - **Relative timestamps**: Store timestamps relative to genesis to reduce size
/// - **Differential encoding**: Encode changes between consecutive blocks
/// - **Epoch sharing**: Multiple blocks can share the same L1 origin
///
/// ## Spatial Compression
/// - **Shared prefixes**: Common data shared across all blocks in span
/// - **Transaction batching**: Transactions grouped and compressed together
/// - **Bit packing**: Use minimal bits for frequently-used fields
///
/// # Format Structure
///
/// ```text
/// SpanBatch {
///   prefix: {
///     rel_timestamp,     // Relative to genesis
///     l1_origin_num,     // Final L1 block number
///     parent_check,      // First 20 bytes of parent hash
///     l1_origin_check,   // First 20 bytes of L1 origin hash
///   },
///   payload: {
///     block_count,       // Number of blocks in span
///     origin_bits,       // Bit array indicating L1 origin changes
///     block_tx_counts,   // Transaction count per block
///     txs,              // Compressed transaction data
///   }
/// }
/// ```
///
/// # Validation and Integrity
///
/// The span batch format includes several integrity checks:
/// - **Parent check**: Validates continuity with previous span
/// - **L1 origin check**: Ensures proper L1 origin binding
/// - **Transaction count validation**: Verifies transaction distribution
/// - **Bit field consistency**: Ensures origin bits match block count
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SpanBatch {
    /// The wire version this span was decoded from, and the one it re-encodes to.
    ///
    /// Only [`BatchType::SpanV2`] can express sibling blocks; [`BatchType::Span`] is the Delta
    /// format, where every element is one block time after its predecessor.
    pub version: BatchType,
    /// First 20 bytes of the parent hash of the first block in the span.
    ///
    /// This field provides a collision-resistant check to ensure the span batch
    /// builds properly on the expected parent block. Using only 20 bytes saves
    /// space while maintaining strong integrity guarantees.
    pub parent_check: FixedBytes<20>,
    /// First 20 bytes of the L1 origin hash of the last block in the span.
    ///
    /// This field enables validation that the span batch references the correct
    /// L1 origin block, ensuring proper derivation ordering and preventing
    /// replay attacks across different L1 contexts.
    pub l1_origin_check: FixedBytes<20>,
    /// Genesis block timestamp for relative timestamp calculations.
    ///
    /// All timestamps in the span batch are stored relative to this genesis
    /// timestamp to minimize storage requirements. This enables efficient
    /// timestamp compression while maintaining full precision.
    pub genesis_timestamp: u64,
    /// Chain ID for transaction validation and network identification.
    ///
    /// Required for proper transaction signature validation and to prevent
    /// cross-chain replay attacks. All transactions in the span must be
    /// valid for this chain ID.
    pub chain_id: u64,
    /// Ordered list of block elements contained in this span.
    ///
    /// Each element represents the derived data for one L2 block, including
    /// timestamp, epoch information, and transaction references. The order
    /// must match the intended L2 block sequence.
    pub batches: Vec<SpanBatchElement>,
    /// Cached bit array indicating L1 origin changes between consecutive blocks.
    ///
    /// This compressed representation allows efficient encoding of which blocks
    /// in the span advance to a new L1 origin. Bit `i` is set if block `i+1`
    /// has a different L1 origin than block `i`.
    pub origin_bits: SpanBatchBits,
    /// Cached bit array marking the elements that share their predecessor's timestamp.
    ///
    /// Present exactly for [`BatchType::SpanV2`] spans. Bit `i` is set if element `i` has the
    /// timestamp of element `i - 1`; bit 0 is set if the first element is a sibling of the
    /// span's parent block.
    pub same_ts_bits: Option<SpanBatchBits>,
    /// Cached transaction count for each block in the span.
    ///
    /// Pre-computed transaction counts enable efficient random access to
    /// transactions for specific blocks without scanning the entire transaction
    /// list. Index `i` contains the transaction count for block `i`.
    pub block_tx_counts: Vec<u64>,
    /// Cached compressed transaction data for all blocks in the span.
    ///
    /// Contains all transactions from all blocks in a compressed format that
    /// enables efficient encoding and decoding. Transactions are grouped and
    /// compressed using span-specific techniques.
    pub txs: SpanBatchTransactions,
}

/// A [`SingleBatch`] extracted from a [`SpanBatch`], together with the span's claim about how it
/// relates to the block it builds on.
///
/// The claim cannot ride on the [`SingleBatch`] itself, whose RLP encoding is the `0x00` wire
/// format, which has no way to express a sibling.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SpanSingleBatch {
    /// The extracted batch.
    pub batch: SingleBatch,
    /// Whether the span marked this element as sharing the timestamp of the block it builds on.
    pub is_sibling: bool,
}

/// The verdict for a span batch that adds no block the safe chain does not already hold.
fn no_new_blocks_validity(cfg: &RollupConfig, inclusion_block: &BlockInfo) -> BatchValidity {
    if cfg.is_holocene_active(inclusion_block.timestamp) {
        BatchValidity::Past
    } else {
        BatchValidity::Drop(BatchDropReason::SpanBatchNoNewBlocksPreHolocene)
    }
}

/// The outcome of the [`SpanBatch`] prefix and Holocene checks.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SpanBatchOutcome {
    /// The checks passed. Carries the L2 block the span's first element builds on, which is what
    /// turns element indices into block numbers.
    Accepted(L2BlockInfo),
    /// The checks did not pass, with the verdict to report.
    Rejected(BatchValidity),
}

impl SpanBatchOutcome {
    /// The verdict this outcome reports.
    pub const fn validity(&self) -> BatchValidity {
        match self {
            Self::Accepted(_) => BatchValidity::Accept,
            Self::Rejected(validity) => *validity,
        }
    }
}

// Manually implemented because [`BatchType`] has no meaningful `Default`: a span defaults to the
// Delta format, not to the first batch type.
impl Default for SpanBatch {
    fn default() -> Self {
        Self {
            version: BatchType::Span,
            parent_check: FixedBytes::default(),
            l1_origin_check: FixedBytes::default(),
            genesis_timestamp: 0,
            chain_id: 0,
            batches: Vec::new(),
            origin_bits: SpanBatchBits::default(),
            same_ts_bits: None,
            block_tx_counts: Vec::new(),
            txs: SpanBatchTransactions::default(),
        }
    }
}

impl SpanBatch {
    /// Returns the starting timestamp for the first batch in the span.
    ///
    /// This is the absolute timestamp (not relative to genesis) of the first
    /// block in the span batch. Used for validation and block sequencing.
    ///
    /// # Panics
    /// Panics if the span batch contains no elements (`batches` is empty).
    /// This should never happen in valid span batches as they must contain
    /// at least one block.
    ///
    /// # Usage
    /// Typically used during span batch validation to ensure proper temporal
    /// ordering with respect to the parent block and L1 derivation window.
    pub fn starting_timestamp(&self) -> u64 {
        self.batches[0].timestamp
    }

    /// Returns the final timestamp for the last batch in the span.
    ///
    /// This is the absolute timestamp (not relative to genesis) of the last
    /// block in the span batch. Used for validation and determining the
    /// span's temporal range.
    ///
    /// # Panics
    /// Panics if the span batch contains no elements (`batches` is empty).
    /// This should never happen in valid span batches as they must contain
    /// at least one block.
    ///
    /// # Usage
    /// Used during validation to ensure the span doesn't exceed maximum
    /// temporal ranges and fits within L1 derivation windows.
    pub fn final_timestamp(&self) -> u64 {
        self.batches[self.batches.len() - 1].timestamp
    }

    /// Returns the L1 epoch number for the first batch in the span.
    ///
    /// The epoch number corresponds to the L1 block number that serves as
    /// the L1 origin for the first L2 block in this span. This establishes
    /// the L1 derivation context for the span.
    ///
    /// # Panics
    /// Panics if the span batch contains no elements (`batches` is empty).
    /// This should never happen in valid span batches as they must contain
    /// at least one block.
    ///
    /// # Usage
    /// Used during validation to ensure proper L1 origin sequencing and
    /// that the span begins with the expected L1 context.
    pub fn starting_epoch_num(&self) -> u64 {
        self.batches[0].epoch_num
    }

    /// Validates that the L1 origin hash matches the span's L1 origin check.
    ///
    /// Compares the first 20 bytes of the provided hash against the stored
    /// `l1_origin_check` field. This provides a collision-resistant validation
    /// that the span batch was derived from the expected L1 context.
    ///
    /// # Arguments
    /// * `hash` - The full 32-byte L1 origin hash to validate
    ///
    /// # Returns
    /// * `true` - If the first 20 bytes match the span's L1 origin check
    /// * `false` - If there's a mismatch, indicating invalid L1 context
    ///
    /// # Algorithm
    /// ```text
    /// l1_origin_check[0..20] == hash[0..20]
    /// ```
    ///
    /// Using only 20 bytes provides strong collision resistance (2^160 space)
    /// while saving 12 bytes per span compared to storing full hashes.
    pub fn check_origin_hash(&self, hash: FixedBytes<32>) -> bool {
        self.l1_origin_check == hash[..20]
    }

    /// Validates that the parent hash matches the span's parent check.
    ///
    /// Compares the first 20 bytes of the provided hash against the stored
    /// `parent_check` field. This ensures the span batch builds on the
    /// expected parent block, maintaining chain continuity.
    ///
    /// # Arguments
    /// * `hash` - The full 32-byte parent hash to validate
    ///
    /// # Returns
    /// * `true` - If the first 20 bytes match the span's parent check
    /// * `false` - If there's a mismatch, indicating discontinuity
    ///
    /// # Algorithm
    /// ```text
    /// parent_check[0..20] == hash[0..20]
    /// ```
    ///
    /// This validation is critical for maintaining the integrity of the L2
    /// chain and preventing insertion of span batches in wrong locations.
    pub fn check_parent_hash(&self, hash: FixedBytes<32>) -> bool {
        self.parent_check == hash[..20]
    }

    /// Accesses the nth element from the end of the batch list.
    ///
    /// This is a convenience method for accessing recent elements in the span,
    /// typically used during validation or processing algorithms that need to
    /// examine the latest elements in the sequence.
    ///
    /// # Arguments
    /// * `n` - Offset from the end (0 = last element, 1 = second-to-last, etc.)
    ///
    /// # Returns
    /// Reference to the nth element from the end of the batch list
    ///
    /// # Panics
    /// Panics if `n >= batches.len()`, i.e., if trying to access beyond
    /// the available elements.
    ///
    /// # Algorithm
    /// ```text
    /// index = batches.len() - 1 - n
    /// return &batches[index]
    /// ```
    fn peek(&self, n: usize) -> &SpanBatchElement {
        &self.batches[self.batches.len() - 1 - n]
    }

    /// Converts this span batch to its raw serializable format.
    ///
    /// Transforms the derived span batch into a [`RawSpanBatch`] that can be
    /// serialized and transmitted over the network. This involves organizing
    /// the cached data into the proper prefix and payload structure.
    ///
    /// # Returns
    /// * `Ok(RawSpanBatch)` - Successfully converted raw span batch
    /// * `Err(SpanBatchError)` - Conversion failed, typically due to empty batch
    ///
    /// # Errors
    /// Returns [`SpanBatchError::EmptySpanBatch`] if the span contains no blocks,
    /// which is invalid as span batches must contain at least one block.
    ///
    /// # Algorithm
    /// The conversion process:
    /// 1. **Validation**: Ensure the span is not empty
    /// 2. **Prefix Construction**: Build prefix with temporal and origin data
    /// 3. **Payload Assembly**: Package cached data into payload structure
    /// 4. **Relative Timestamp Calculation**: Convert absolute to relative timestamp
    ///
    /// The relative timestamp is calculated as:
    /// ```text
    /// rel_timestamp = first_block_timestamp - genesis_timestamp
    /// ```
    ///
    /// This enables efficient timestamp encoding in the serialized format.
    pub fn to_raw_span_batch(&self) -> Result<RawSpanBatch, SpanBatchError> {
        if !self.version.is_span() {
            return Err(SpanBatchError::NotASpanVersion);
        }
        if self.batches.is_empty() {
            return Err(SpanBatchError::EmptySpanBatch);
        }

        // These should never error since we check for an empty batch above.
        let span_start = self.batches.first().ok_or(SpanBatchError::EmptySpanBatch)?;
        let span_end = self.batches.last().ok_or(SpanBatchError::EmptySpanBatch)?;

        Ok(RawSpanBatch {
            version: self.version,
            prefix: SpanBatchPrefix {
                rel_timestamp: span_start.timestamp - self.genesis_timestamp,
                l1_origin_num: span_end.epoch_num,
                parent_check: self.parent_check,
                l1_origin_check: self.l1_origin_check,
            },
            payload: SpanBatchPayload {
                block_count: self.batches.len() as u64,
                origin_bits: self.origin_bits.clone(),
                same_ts_bits: self.same_ts_bits.clone(),
                block_tx_counts: self.block_tx_counts.clone(),
                txs: self.txs.clone(),
            },
        })
    }

    /// Returns true if element `index` shares the timestamp of the block it builds on: element
    /// `index - 1`, or the span's parent block for element 0.
    ///
    /// The wire version is the authority on whether a span can express siblings at all, so a
    /// bitlist carried by a Delta span is ignored rather than honoured.
    pub fn is_sibling(&self, index: usize) -> bool {
        self.version.has_same_ts_bits() &&
            self.same_ts_bits.as_ref().and_then(|bits| bits.get_bit(index)) == Some(1)
    }

    /// The block number of the span's parent, derived from the elements the safe chain already
    /// holds.
    ///
    /// Only correct where every element is exactly one block time after its predecessor, which
    /// makes an element's timestamp decide whether the safe chain holds it: siblings share a
    /// timestamp, so counting them this way would place the parent too high.
    pub fn parent_number_from_timestamps(&self, l2_safe_head: L2BlockInfo) -> u64 {
        let applied = self
            .batches
            .iter()
            .filter(|b| b.timestamp <= l2_safe_head.block_info.timestamp)
            .count() as u64;
        l2_safe_head.block_info.number.saturating_sub(applied)
    }

    /// Converts all [`SpanBatchElement`]s after the L2 safe head to [`SingleBatch`]es. The
    /// resulting [`SingleBatch`]es do not contain a parent hash, as it is populated by the
    /// Batch Queue stage.
    ///
    /// `span_parent_number` is the block number of the span's parent, i.e. the block its first
    /// element builds on. The elements are the consecutive blocks after it, so their numbers —
    /// not their timestamps, which siblings share — decide which ones the safe chain already
    /// holds.
    pub fn get_singular_batches(
        &self,
        l1_origins: &[BlockInfo],
        l2_safe_head: L2BlockInfo,
        span_parent_number: u64,
    ) -> Result<Vec<SpanSingleBatch>, SpanBatchError> {
        let mut single_batches = Vec::with_capacity(self.batches.len());
        let mut origin_index = 0;
        for (index, batch) in self.batches.iter().enumerate() {
            if span_parent_number + 1 + index as u64 <= l2_safe_head.block_info.number {
                continue;
            }
            // Overlapping span batches can pass the prefix checks but then the
            // first batch after the safe head has an outdated L1 origin.
            if batch.epoch_num < l2_safe_head.l1_origin.number {
                return Err(SpanBatchError::L1OriginBeforeSafeHead);
            }
            let origin_epoch_hash = l1_origins[origin_index..l1_origins.len()]
                .iter()
                .enumerate()
                .find(|(_, origin)| origin.number == batch.epoch_num)
                .map(|(i, origin)| {
                    origin_index = i;
                    origin.hash
                })
                .ok_or(SpanBatchError::MissingL1Origin)?;
            let single_batch = SingleBatch {
                epoch_num: batch.epoch_num,
                epoch_hash: origin_epoch_hash,
                timestamp: batch.timestamp,
                transactions: batch.transactions.clone(),
                ..Default::default()
            };
            single_batches
                .push(SpanSingleBatch { batch: single_batch, is_sibling: self.is_sibling(index) });
        }
        Ok(single_batches)
    }

    /// Append a [`SingleBatch`] to the [`SpanBatch`]. Updates the L1 origin check if need be.
    ///
    /// `parent_timestamp` is the timestamp of the span's parent block, i.e. the block the first
    /// element builds on. It is the only way to tell whether that first element is a sibling of
    /// the parent, which a span cannot observe from its own elements.
    pub fn append_singular_batch(
        &mut self,
        singular_batch: SingleBatch,
        seq_num: u64,
        parent_timestamp: u64,
    ) -> Result<(), SpanBatchError> {
        // If the new element is not ordered with respect to the last element, panic.
        assert!(
            self.batches.is_empty() || self.peek(0).timestamp <= singular_batch.timestamp,
            "Batch is not ordered"
        );

        // Decided before any mutation: a v1 span cannot express a sibling, so the rejected append
        // must leave the span untouched rather than half-written.
        let predecessor_timestamp =
            self.batches.last().map_or(parent_timestamp, |batch| batch.timestamp);
        let same_ts_bit = predecessor_timestamp == singular_batch.timestamp;
        if same_ts_bit && !self.version.has_same_ts_bits() {
            return Err(SpanBatchError::SameTimestampBitsMismatch);
        }

        let SingleBatch { epoch_hash, parent_hash, .. } = singular_batch;

        // Always append the new batch and set the L1 origin check.
        self.batches.push(singular_batch.into());
        // Always update the L1 origin check.
        self.l1_origin_check = epoch_hash[..20].try_into().expect("Sub-slice cannot fail");

        let epoch_bit = if self.batches.len() == 1 {
            // If there is only one batch, initialize the parent check and set the epoch bit based
            // on the sequence number.
            self.parent_check = parent_hash[..20].try_into().expect("Sub-slice cannot fail");
            seq_num == 0
        } else {
            // If there is more than one batch, set the epoch bit based on the last two batches.
            self.peek(1).epoch_num < self.peek(0).epoch_num
        };

        // Set the respective bit in the origin bits.
        self.origin_bits.set_bit(self.batches.len() - 1, epoch_bit);

        if self.version.has_same_ts_bits() {
            self.same_ts_bits
                .get_or_insert_with(SpanBatchBits::default)
                .set_bit(self.batches.len() - 1, same_ts_bit);
        }

        let new_txs = self.peek(0).transactions.clone();

        // Update the block tx counts cache with the latest batch's transaction count.
        self.block_tx_counts.push(new_txs.len() as u64);

        // Add the new transactions to the transaction cache.
        self.txs.add_txs(new_txs, self.chain_id)
    }

    /// Checks if the span batch is valid.
    pub async fn check_batch<BV: BatchValidationProvider>(
        &self,
        cfg: &RollupConfig,
        l1_blocks: &[BlockInfo],
        l2_safe_head: L2BlockInfo,
        inclusion_block: &BlockInfo,
        fetcher: &mut BV,
    ) -> BatchValidity {
        // The pre-Holocene queue validates a span as a whole and applies it without streaming its
        // elements, so it cannot tell siblings apart from the blocks they share a timestamp with.
        if self.version.has_same_ts_bits() {
            warn!(target: "batch_span", "received a span batch v2 before Holocene");
            return BatchValidity::Drop(BatchDropReason::SpanBatchV2PreHolocene);
        }

        let parent_block = match self
            .check_batch_prefix(cfg, l1_blocks, l2_safe_head, inclusion_block, fetcher)
            .await
        {
            SpanBatchOutcome::Accepted(parent_block) => parent_block,
            SpanBatchOutcome::Rejected(validity) => return validity,
        };

        let starting_epoch_num = self.starting_epoch_num();

        let mut origin_index = 0;
        let mut origin_advanced = starting_epoch_num == parent_block.l1_origin.number + 1;
        for (i, batch) in self.batches.iter().enumerate() {
            let batch_timestamp = batch.timestamp;
            let batch_epoch = batch.epoch_num;

            if batch_timestamp <= l2_safe_head.block_info.timestamp {
                continue;
            }
            if batch_epoch < l2_safe_head.l1_origin.number {
                warn!(
                    target: "batch_span",
                    "batch L1 origin is before safe head L1 origin, batch_epoch: {}, safe_head_epoch: {:?}",
                    batch_epoch,
                    l2_safe_head.l1_origin
                );
                return BatchValidity::Drop(BatchDropReason::L1OriginBeforeSafeHead);
            }

            // Find the L1 origin for the batch.
            let Some((offset, l1_origin)) =
                l1_blocks[origin_index..].iter().enumerate().find(|(_, b)| batch_epoch == b.number)
            else {
                warn!(
                    target: "batch_span",
                    "unable to find L1 origin for batch, batch_epoch: {}, batch_timestamp: {}",
                    batch_epoch,
                    batch_timestamp
                );
                return BatchValidity::Drop(BatchDropReason::MissingL1Origin);
            };
            origin_index += offset;

            if i > 0 {
                origin_advanced = batch_epoch > self.batches[i - 1].epoch_num;
            }
            if batch_timestamp < l1_origin.timestamp {
                warn!(
                    target: "batch_span",
                    "batch timestamp is less than L1 origin timestamp, l2_timestamp: {}, l1_timestamp: {}, origin: {:?}",
                    batch_timestamp,
                    l1_origin.timestamp,
                    l1_origin.id()
                );
                return BatchValidity::Drop(BatchDropReason::TimestampBeforeL1Origin);
            }

            // Check if we ran out of sequencer time drift
            let max_drift = cfg.max_sequencer_drift(l1_origin.timestamp);
            if batch_timestamp > l1_origin.timestamp + max_drift {
                if batch.transactions.is_empty() {
                    // If the sequencer is co-operating by producing an empty batch,
                    // then allow the batch if it was the right thing to do to maintain the L2 time
                    // >= L1 time invariant. We only check batches that do not
                    // advance the epoch, to ensure epoch advancement regardless of time drift is
                    // allowed.
                    if !origin_advanced {
                        if origin_index + 1 >= l1_blocks.len() {
                            info!(
                                target: "batch_span",
                                "without the next L1 origin we cannot determine yet if this empty batch that exceeds the time drift is still valid"
                            );
                            return BatchValidity::Undecided;
                        }
                        if batch_timestamp >= l1_blocks[origin_index + 1].timestamp {
                            // check if the next L1 origin could have been adopted
                            warn!(
                                target: "batch_span",
                                "batch exceeded sequencer time drift without adopting next origin, and next L1 origin would have been valid"
                            );
                            return BatchValidity::Drop(
                                BatchDropReason::SequencerDriftNotAdoptedNextOrigin,
                            );
                        }
                        info!(
                            target: "batch_span",
                            "continuing with empty batch before late L1 block to preserve L2 time invariant"
                        );
                    }
                } else {
                    // If the sequencer is ignoring the time drift rule, then drop the batch and
                    // force an empty batch instead, as the sequencer is not
                    // allowed to include anything past this point without moving to the next epoch.
                    warn!(
                        target: "batch_span",
                        "batch exceeded sequencer time drift, sequencer must adopt new L1 origin to include transactions again, max_time: {}",
                        l1_origin.timestamp + max_drift
                    );
                    return BatchValidity::Drop(BatchDropReason::SequencerDriftExceeded);
                }
            }

            // Check that the transactions are not empty and do not contain any deposits.
            for (i, tx) in batch.transactions.iter().enumerate() {
                let Some(first_byte) = tx.as_ref().first().copied() else {
                    warn!(
                        target: "batch_span",
                        "transaction data must not be empty, but found empty tx, tx_index: {}",
                        i
                    );
                    return BatchValidity::Drop(BatchDropReason::EmptyTransaction);
                };
                // A leading byte that doesn't decode to a typed transaction (e.g. a legacy RLP
                // list header) isn't one of the restricted types, so it falls through to `Accept`.
                match OpTxType::try_from(first_byte) {
                    Ok(OpTxType::Deposit) => {
                        warn!(
                            target: "batch_span",
                            "sequencers may not embed any deposits into batch data, but found tx that has one, tx_index: {}",
                            i
                        );
                        return BatchValidity::Drop(BatchDropReason::DepositTransaction);
                    }
                    Ok(OpTxType::Eip7702) if !cfg.is_isthmus_active(batch.timestamp) => {
                        warn!(target: "batch_span", "EIP-7702 transactions are not supported pre-isthmus. tx_index: {}", i);
                        return BatchValidity::Drop(BatchDropReason::Eip7702PreIsthmus);
                    }
                    Ok(OpTxType::PostExec) if !cfg.is_sdm_active(batch.timestamp) => {
                        warn!(target: "batch_span", "PostExec transactions are not supported pre-Lagoon. tx_index: {}", i);
                        return BatchValidity::Drop(BatchDropReason::PostExecPreLagoon);
                    }
                    _ => {}
                }
            }
        }

        // Check overlapped blocks
        let overlap_validity =
            self.check_batch_overlap(cfg, parent_block, l2_safe_head, fetcher).await;
        if !overlap_validity.is_accept() {
            return overlap_validity;
        }

        BatchValidity::Accept
    }

    /// Validates the portion of a span batch that overlaps the safe chain: every overlapped
    /// element's transactions and L1 origin must match the canonical block at the same height.
    /// A batch whose overlap disagrees with the safe chain — possible since interop block
    /// replacement — describes a different history and is dropped as a whole, so that its
    /// elements past the safe head cannot be applied. Returns [`BatchValidity::Undecided`] if a
    /// canonical payload cannot be fetched right now.
    ///
    /// The prefix rules must have accepted the batch first: `parent_block` must be the
    /// prefix-determined parent (an ancestor of, or equal to, `l2_safe_head`), and the batch must
    /// extend past the safe head — these bound the loop and the element indexing.
    pub async fn check_batch_overlap<BV: BatchValidationProvider>(
        &self,
        cfg: &RollupConfig,
        parent_block: L2BlockInfo,
        l2_safe_head: L2BlockInfo,
        fetcher: &mut BV,
    ) -> BatchValidity {
        let parent_num = parent_block.block_info.number;
        let overlap = l2_safe_head.block_info.number.saturating_sub(parent_num);
        if overlap > self.batches.len() as u64 {
            // The prefix rules reject a span whose last element is at or below the safe head, so
            // this cannot happen after they accepted it. Reported rather than indexed past the
            // end, because a panic here would take the node and the fault proof program down.
            warn!(
                target: "batch_span",
                "span batch overlaps {overlap} safe blocks but only has {} elements",
                self.batches.len()
            );
            return BatchValidity::Drop(BatchDropReason::SpanBatchNotOverlappedExactly);
        }
        // Reused encoding buffer for the per-transaction comparison below, hoisted out of the
        // loops to avoid one allocation per compared transaction.
        let mut buf = Vec::new();
        for i in 0..overlap {
            let safe_block_num = parent_num + i + 1;
            let safe_block_payload = match fetcher.block_by_number(safe_block_num).await {
                Ok(p) => p,
                Err(e) => {
                    warn!(target: "batch_span", "failed to fetch block number {safe_block_num}: {e}");
                    return BatchValidity::Undecided;
                }
            };
            let safe_block = &safe_block_payload.body;
            let batch_txs = &self.batches[i as usize].transactions;
            // Execution payload has deposit txs but batch does not.
            let deposit_count: usize =
                safe_block.transactions.iter().map(|tx| if tx.is_deposit() { 1 } else { 0 }).sum();
            if safe_block.transactions.len() - deposit_count != batch_txs.len() {
                warn!(
                    target: "batch_span",
                    "overlapped block's tx count does not match, safe_block_txs: {}, batch_txs: {}",
                    safe_block.transactions.len(),
                    batch_txs.len()
                );
                return BatchValidity::Drop(BatchDropReason::OverlappedTxCountMismatch);
            }
            let batch_txs_len = batch_txs.len();
            #[allow(clippy::needless_range_loop)]
            for j in 0..batch_txs_len {
                buf.clear();
                safe_block.transactions[j + deposit_count].encode_2718(&mut buf);
                if buf != batch_txs[j].0 {
                    warn!(target: "batch_span", "overlapped block's transaction does not match");
                    return BatchValidity::Drop(BatchDropReason::OverlappedTxMismatch);
                }
            }
            let safe_block_ref = match L2BlockInfo::from_block_and_genesis(
                &safe_block_payload,
                &cfg.genesis,
            ) {
                Ok(r) => r,
                Err(e) => {
                    warn!(
                        target: "batch_span",
                        "failed to extract L2BlockInfo from execution payload, hash: {}, err: {e}",
                        safe_block_payload.header.hash_slow()
                    );
                    return BatchValidity::Drop(BatchDropReason::L2BlockInfoExtractionFailed);
                }
            };
            if safe_block_ref.l1_origin.number != self.batches[i as usize].epoch_num {
                warn!(
                    "overlapped block's L1 origin number does not match {}, {}",
                    safe_block_ref.l1_origin.number, self.batches[i as usize].epoch_num
                );
                return BatchValidity::Drop(BatchDropReason::OverlappedL1OriginMismatch);
            }
        }

        BatchValidity::Accept
    }

    /// Checks the span batch prefix rules shared by the legacy full checks and the Holocene
    /// checks.
    pub async fn check_batch_prefix<BF: BatchValidationProvider>(
        &self,
        cfg: &RollupConfig,
        l1_origins: &[BlockInfo],
        l2_safe_head: L2BlockInfo,
        inclusion_block: &BlockInfo,
        fetcher: &mut BF,
    ) -> SpanBatchOutcome {
        if l1_origins.is_empty() {
            warn!(
                target: "batch_span",
                chain_id = cfg.l2_chain_id.id(),
                l1_origin_count = l1_origins.len(),
                span_block_count = self.batches.len(),
                safe_head_number = l2_safe_head.block_info.number,
                safe_head_timestamp = l2_safe_head.block_info.timestamp,
                safe_head_l1_origin_number = l2_safe_head.l1_origin.number,
                inclusion_l1_block_number = inclusion_block.number,
                inclusion_l1_block_timestamp = inclusion_block.timestamp,
                holocene_active = cfg.is_holocene_active(inclusion_block.timestamp),
                "missing L1 block input, cannot proceed with batch checking"
            );
            return SpanBatchOutcome::Rejected(BatchValidity::Undecided);
        }
        if self.batches.is_empty() {
            warn!(target: "batch_span", "empty span batch, cannot proceed with batch checking");
            return SpanBatchOutcome::Rejected(BatchValidity::Undecided);
        }

        let epoch = l1_origins[0];
        let safe_head_timestamp = l2_safe_head.block_info.timestamp;

        let starting_epoch_num = self.starting_epoch_num();
        let mut batch_origin = epoch;
        if starting_epoch_num == batch_origin.number + 1 {
            if l1_origins.len() < 2 {
                info!(
                    target: "batch_span",
                    "eager batch wants to advance current epoch {:?}, but could not without more L1 blocks",
                    epoch.id()
                );
                return SpanBatchOutcome::Rejected(BatchValidity::Undecided);
            }
            batch_origin = l1_origins[1];
        }
        if !cfg.is_delta_active(batch_origin.timestamp) {
            warn!(
                target: "batch_span",
                "received SpanBatch (id {:?}) with L1 origin (timestamp {}) before Delta hard fork",
                batch_origin.id(),
                batch_origin.timestamp
            );
            return SpanBatchOutcome::Rejected(BatchValidity::Drop(
                BatchDropReason::SpanBatchPreDelta,
            ));
        }

        // The span batch v2 format is gated on the L1 inclusion block, exactly like the Delta
        // gate above.
        if self.version.has_same_ts_bits() && !cfg.is_multi_block_active(inclusion_block.timestamp)
        {
            warn!(
                target: "batch_span",
                "received a span batch v2 included at L1 timestamp {}, before multi-blocks activate",
                inclusion_block.timestamp
            );
            return SpanBatchOutcome::Rejected(BatchValidity::Drop(
                BatchDropReason::SpanBatchV2PreActivation,
            ));
        }

        // A span that starts with a sibling continues the safe head's own timestamp; every other
        // span starts one block time after the block it builds on.
        let first_is_sibling = self.is_sibling(0);
        let max_starting_timestamp = if first_is_sibling {
            safe_head_timestamp
        } else {
            safe_head_timestamp + cfg.block_time
        };
        if self.starting_timestamp() > max_starting_timestamp {
            warn!(
                target: "batch_span",
                "received out-of-order batch for future processing after next batch ({} > {})",
                self.starting_timestamp(),
                max_starting_timestamp
            );

            // After holocene is activated, gaps are disallowed.
            if cfg.is_holocene_active(inclusion_block.timestamp) {
                return SpanBatchOutcome::Rejected(BatchValidity::Drop(
                    BatchDropReason::FutureTimestampHolocene,
                ));
            }
            return SpanBatchOutcome::Rejected(BatchValidity::Future);
        }

        // Drop the batch if it has no new blocks after the safe head. This decides only the
        // clear-cut case, where even the span's last element predates the safe head; the
        // authoritative rule is the one by block number below, which needs the span's parent.
        let last_element_is_old = if self.version.has_same_ts_bits() {
            self.final_timestamp() < safe_head_timestamp
        } else {
            self.final_timestamp() < safe_head_timestamp + cfg.block_time
        };
        if last_element_is_old {
            let span_start_timestamp = self.starting_timestamp();
            let span_final_timestamp = self.final_timestamp();
            let span_lag_seconds = max_starting_timestamp.saturating_sub(span_final_timestamp);
            let holocene_active = cfg.is_holocene_active(inclusion_block.timestamp);
            let batch_validity = if holocene_active { "past" } else { "drop" };
            warn!(
                target: "batch_span",
                chain_id = cfg.l2_chain_id.id(),
                span_start_timestamp,
                span_final_timestamp,
                safe_head_timestamp,
                next_expected_timestamp = max_starting_timestamp,
                span_lag_seconds,
                inclusion_l1_block_number = inclusion_block.number,
                inclusion_l1_block_timestamp = inclusion_block.timestamp,
                holocene_active,
                batch_validity,
                "span batch has no new blocks after safe head"
            );
            return SpanBatchOutcome::Rejected(no_new_blocks_validity(cfg, inclusion_block));
        }

        let parent_block =
            match self.locate_parent(cfg, l2_safe_head, first_is_sibling, fetcher).await {
                Ok(block) => block,
                Err(validity) => return SpanBatchOutcome::Rejected(validity),
            };

        // Element `i` is block `parent.number + i + 1`, so a span whose last element the safe
        // chain already holds has nothing left to apply. This is what bounds the overlap against
        // the element count, and it is the only form of the rule that survives siblings, whose
        // shared timestamps break the timestamp-to-block-number bijection.
        if parent_block.block_info.number + self.batches.len() as u64 <=
            l2_safe_head.block_info.number
        {
            warn!(
                target: "batch_span",
                "span batch's last element is at or below the safe head, safe_head_number: {}, span_parent_number: {}, span_block_count: {}",
                l2_safe_head.block_info.number,
                parent_block.block_info.number,
                self.batches.len()
            );
            return SpanBatchOutcome::Rejected(no_new_blocks_validity(cfg, inclusion_block));
        }

        // Filter out batches that were included too late.
        if starting_epoch_num + cfg.seq_window_size < inclusion_block.number {
            warn!(target: "batch_span", "batch was included too late, sequence window expired");
            return SpanBatchOutcome::Rejected(BatchValidity::Drop(
                BatchDropReason::IncludedTooLate,
            ));
        }

        // Check the L1 origin of the batch
        if starting_epoch_num > parent_block.l1_origin.number + 1 {
            warn!(
                target: "batch_span",
                "batch is for future epoch too far ahead, while it has the next timestamp, so it must be invalid. starting epoch: {} | next epoch: {}",
                starting_epoch_num,
                parent_block.l1_origin.number + 1
            );
            return SpanBatchOutcome::Rejected(BatchValidity::Drop(
                BatchDropReason::EpochTooFarInFuture,
            ));
        }

        // Verify the l1 origin hash for each l1 block.
        // SAFETY: `Self::batches` is not empty, so the last element is guaranteed to exist.
        let end_epoch_num = self.batches.last().unwrap().epoch_num;
        let mut origin_checked = false;
        // l1Blocks is supplied from batch queue and its length is limited to SequencerWindowSize.
        for l1_block in l1_origins {
            if l1_block.number == end_epoch_num {
                if !self.check_origin_hash(l1_block.hash) {
                    warn!(
                        target: "batch_span",
                        l1_block_number = ?l1_block.number,
                        l1_block_hash = ?l1_block.hash,
                        l1_origin_number = ?starting_epoch_num,
                        l1_check_hash = ?self.l1_origin_check,
                        "batch is for different L1 chain, epoch hash does not match",
                    );
                    return SpanBatchOutcome::Rejected(BatchValidity::Drop(
                        BatchDropReason::EpochHashMismatch,
                    ));
                }
                origin_checked = true;
                break;
            }
        }
        if !origin_checked {
            info!(target: "batch_span", "need more l1 blocks to check entire origins of span batch");
            return SpanBatchOutcome::Rejected(BatchValidity::Undecided);
        }

        if starting_epoch_num < parent_block.l1_origin.number {
            warn!(target: "batch_span", "dropped batch, epoch is too old, minimum: {:?}", parent_block.block_info.id());
            return SpanBatchOutcome::Rejected(BatchValidity::Drop(BatchDropReason::EpochTooOld));
        }

        let sibling_validity = self.check_sibling_rules(cfg, parent_block, fetcher).await;
        if !sibling_validity.is_accept() {
            return SpanBatchOutcome::Rejected(sibling_validity);
        }

        SpanBatchOutcome::Accepted(parent_block)
    }

    /// Locates the L2 block the span's first element builds on: the block at `parent_timestamp`
    /// whose hash the span's `parent_check` names.
    ///
    /// The walk starts from the safe head. Each timestamp between the parent and the safe head
    /// carries at least one block, which bounds how far below the safe head the parent can be;
    /// the run of blocks sharing `parent_timestamp` is then at most `max_multi_blocks` long.
    /// Returns the validity to report when no such block exists.
    async fn locate_parent<BF: BatchValidationProvider>(
        &self,
        cfg: &RollupConfig,
        l2_safe_head: L2BlockInfo,
        first_is_sibling: bool,
        fetcher: &mut BF,
    ) -> Result<L2BlockInfo, BatchValidity> {
        async fn fetch<BF: BatchValidationProvider>(
            fetcher: &mut BF,
            number: u64,
        ) -> Result<L2BlockInfo, BatchValidity> {
            fetcher.l2_block_info_by_number(number).await.map_err(|e| {
                warn!(target: "batch_span", "failed to fetch L2 block number {number}: {e}");
                // Only the pre-Holocene BatchQueue retains an undecided batch for a retry; the
                // Holocene BatchStream has already consumed it and skips it.
                BatchValidity::Undecided
            })
        }

        let safe_head_timestamp = l2_safe_head.block_info.timestamp;
        let starting_timestamp = self.starting_timestamp();
        // How far the first element's timestamp is above the one it builds on.
        let step = if first_is_sibling { 0 } else { cfg.block_time };

        let mut block = l2_safe_head;
        let parent_timestamp = if starting_timestamp > safe_head_timestamp {
            // The span starts past the safe head, which the future check above bounds to one
            // block time, so it can only build on the safe head itself.
            if starting_timestamp != safe_head_timestamp + step {
                warn!(target: "batch_span", "batch has misaligned timestamp, block time is too short");
                return Err(BatchValidity::Drop(BatchDropReason::SpanBatchMisalignedTimestamp));
            }
            safe_head_timestamp
        } else {
            let lag = safe_head_timestamp - starting_timestamp + step;
            if lag != 0 && !lag.is_multiple_of(cfg.block_time) {
                warn!(target: "batch_span", "batch has misaligned timestamp, not overlapped exactly");
                return Err(BatchValidity::Drop(BatchDropReason::SpanBatchNotOverlappedExactly));
            }
            let timestamps = if lag == 0 { 0 } else { lag / cfg.block_time };
            // An upper bound on the parent's block number: every timestamp between it and the
            // safe head carries at least one block.
            let Some(mut number) = l2_safe_head.block_info.number.checked_sub(timestamps) else {
                warn!(target: "batch_span", "batch's parent would be below the L2 genesis block");
                return Err(BatchValidity::Drop(BatchDropReason::SpanBatchParentBelowGenesis));
            };
            let parent_timestamp = safe_head_timestamp - lag;
            if timestamps > 0 {
                block = fetch(fetcher, number).await?;
                // Each of those timestamps carries at most `max_multi_blocks` blocks, so that is
                // how far below the estimate the parent can be. Without siblings it is exact.
                let mut remaining =
                    timestamps.saturating_mul(cfg.max_multi_blocks().saturating_sub(1));
                while block.block_info.timestamp > parent_timestamp &&
                    remaining > 0 &&
                    number > cfg.genesis.l2.number
                {
                    remaining -= 1;
                    number -= 1;
                    block = fetch(fetcher, number).await?;
                }
            }
            parent_timestamp
        };

        // `block` is the last block at `parent_timestamp`, the only possible parent of an element
        // at a later timestamp. A sibling may instead build on any earlier member of that group.
        let group_members = if first_is_sibling { cfg.max_multi_blocks() } else { 1 };
        let mut number = block.block_info.number;
        for member in 0..group_members {
            if member > 0 {
                if number <= cfg.genesis.l2.number {
                    break;
                }
                number -= 1;
                block = fetch(fetcher, number).await?;
                if block.block_info.timestamp != parent_timestamp {
                    break;
                }
            }
            if self.check_parent_hash(block.block_info.hash) {
                // The parent is named by its timestamp as much as by its hash, so a 20-byte
                // prefix match on a block elsewhere on the chain does not make it the parent.
                if block.block_info.timestamp != parent_timestamp {
                    warn!(
                        target: "batch_span",
                        "parent hash check matches block {} at timestamp {}, not {parent_timestamp}",
                        block.block_info.number,
                        block.block_info.timestamp,
                    );
                    return Err(BatchValidity::Drop(
                        BatchDropReason::SpanBatchNotOverlappedExactly,
                    ));
                }
                return Ok(block);
            }
        }
        warn!(
            target: "batch_span",
            "parent block mismatch, no block at timestamp {parent_timestamp} matches parent hash check {}",
            self.parent_check,
        );
        Err(BatchValidity::Drop(BatchDropReason::ParentHashMismatch))
    }

    /// Checks the rules that govern blocks sharing a timestamp.
    ///
    /// A group is a run of consecutive blocks with the same timestamp. Its length is a property
    /// of the chain, so the run the parent belongs to is recomputed here rather than carried over
    /// from the span that sequenced it.
    async fn check_sibling_rules<BF: BatchValidationProvider>(
        &self,
        cfg: &RollupConfig,
        parent_block: L2BlockInfo,
        fetcher: &mut BF,
    ) -> BatchValidity {
        let max_multi_blocks = cfg.max_multi_blocks();
        let mut group_length = if self.is_sibling(0) {
            let mut length = 1;
            let mut number = parent_block.block_info.number;
            while length < max_multi_blocks && number > cfg.genesis.l2.number {
                number -= 1;
                let block = match fetcher.l2_block_info_by_number(number).await {
                    Ok(block) => block,
                    Err(e) => {
                        warn!(target: "batch_span", "failed to fetch L2 block number {number}: {e}");
                        return BatchValidity::Undecided;
                    }
                };
                if block.block_info.timestamp != parent_block.block_info.timestamp {
                    break;
                }
                length += 1;
            }
            length
        } else {
            0
        };

        for (index, element) in self.batches.iter().enumerate() {
            if !self.is_sibling(index) {
                group_length = 1;
                continue;
            }
            if !cfg.siblings_allowed(element.timestamp) {
                warn!(
                    target: "batch_span",
                    "span element {index} shares its parent's timestamp {} where siblings are not allowed",
                    element.timestamp
                );
                return BatchValidity::Drop(BatchDropReason::SiblingsNotAllowed);
            }
            let predecessor_epoch = index
                .checked_sub(1)
                .map_or(parent_block.l1_origin.number, |i| self.batches[i].epoch_num);
            if element.epoch_num != predecessor_epoch {
                warn!(
                    target: "batch_span",
                    "span element {index} is a sibling but adopts L1 origin {} instead of {predecessor_epoch}",
                    element.epoch_num
                );
                return BatchValidity::Drop(BatchDropReason::SiblingOriginMismatch);
            }
            group_length += 1;
            if group_length > max_multi_blocks {
                warn!(
                    target: "batch_span",
                    "{group_length} blocks share timestamp {}, more than max_multi_blocks {max_multi_blocks}",
                    element.timestamp
                );
                return BatchValidity::Drop(BatchDropReason::MultiBlockGroupTooLarge);
            }
        }

        BatchValidity::Accept
    }

    /// Checks the Holocene span batch rules: prefix validity followed by overlap validity.
    pub async fn check_batch_holocene<BV: BatchValidationProvider>(
        &self,
        cfg: &RollupConfig,
        l1_origins: &[BlockInfo],
        l2_safe_head: L2BlockInfo,
        inclusion_block: &BlockInfo,
        fetcher: &mut BV,
    ) -> SpanBatchOutcome {
        let parent_block = match self
            .check_batch_prefix(cfg, l1_origins, l2_safe_head, inclusion_block, fetcher)
            .await
        {
            SpanBatchOutcome::Accepted(parent_block) => parent_block,
            rejected => return rejected,
        };
        let overlap_validity =
            self.check_batch_overlap(cfg, parent_block, l2_safe_head, fetcher).await;
        if !overlap_validity.is_accept() {
            return SpanBatchOutcome::Rejected(overlap_validity);
        }
        SpanBatchOutcome::Accepted(parent_block)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::{CollectingLayer, TestBatchValidator, TraceStorage};
    use alloc::vec;
    use alloy_consensus::{Header, constants::EIP1559_TX_TYPE_ID};
    use alloy_eips::BlockNumHash;
    use alloy_primitives::{B256, Bytes, b256};
    use kona_genesis::{ChainGenesis, HardForkConfig};
    use op_alloy_consensus::{OpBlock, POST_EXEC_TX_TYPE_ID};
    use tracing::Level;
    use tracing_subscriber::layer::SubscriberExt;

    fn gen_l1_blocks(
        start_num: u64,
        count: u64,
        start_timestamp: u64,
        interval: u64,
    ) -> Vec<BlockInfo> {
        (0..count)
            .map(|i| BlockInfo {
                number: start_num + i,
                timestamp: start_timestamp + i * interval,
                hash: B256::left_padding_from(&i.to_be_bytes()),
                ..Default::default()
            })
            .collect()
    }

    #[test]
    fn test_timestamp() {
        let timestamp = 10;
        let first_element = SpanBatchElement { timestamp, ..Default::default() };
        let batch =
            SpanBatch { batches: vec![first_element, Default::default()], ..Default::default() };
        assert_eq!(batch.starting_timestamp(), timestamp);
    }

    #[test]
    fn test_starting_timestamp() {
        let timestamp = 10;
        let first_element = SpanBatchElement { timestamp, ..Default::default() };
        let batch =
            SpanBatch { batches: vec![first_element, Default::default()], ..Default::default() };
        assert_eq!(batch.starting_timestamp(), timestamp);
    }

    #[test]
    fn test_final_timestamp() {
        let timestamp = 10;
        let last_element = SpanBatchElement { timestamp, ..Default::default() };
        let batch =
            SpanBatch { batches: vec![Default::default(), last_element], ..Default::default() };
        assert_eq!(batch.final_timestamp(), timestamp);
    }

    #[test]
    fn test_starting_epoch_num() {
        let epoch_num = 10;
        let first_element = SpanBatchElement { epoch_num, ..Default::default() };
        let batch =
            SpanBatch { batches: vec![first_element, Default::default()], ..Default::default() };
        assert_eq!(batch.starting_epoch_num(), epoch_num);
    }

    #[test]
    fn test_peek() {
        let first_element = SpanBatchElement { epoch_num: 10, ..Default::default() };
        let second_element = SpanBatchElement { epoch_num: 11, ..Default::default() };
        let batch =
            SpanBatch { batches: vec![first_element, second_element], ..Default::default() };
        assert_eq!(batch.peek(0).epoch_num, 11);
        assert_eq!(batch.peek(1).epoch_num, 10);
    }

    #[test]
    fn test_append_empty_singular_batch() {
        let mut batch = SpanBatch::default();
        let singular_batch = SingleBatch {
            epoch_num: 10,
            epoch_hash: FixedBytes::from([17u8; 32]),
            parent_hash: FixedBytes::from([17u8; 32]),
            timestamp: 10,
            transactions: vec![],
        };
        assert!(batch.append_singular_batch(singular_batch, 0, 8).is_ok());
        assert_eq!(batch.batches.len(), 1);
        assert_eq!(batch.origin_bits.get_bit(0), Some(1));
        assert_eq!(batch.block_tx_counts, vec![0]);
        assert_eq!(batch.txs.tx_data.len(), 0);

        // Add another empty single batch.
        let singular_batch = SingleBatch {
            epoch_num: 11,
            epoch_hash: FixedBytes::from([17u8; 32]),
            parent_hash: FixedBytes::from([17u8; 32]),
            timestamp: 20,
            transactions: vec![],
        };
        assert!(batch.append_singular_batch(singular_batch, 1, 8).is_ok());
    }

    fn sibling_singular_batch(epoch_num: u64, timestamp: u64) -> SingleBatch {
        SingleBatch {
            epoch_num,
            epoch_hash: FixedBytes::from([17u8; 32]),
            parent_hash: FixedBytes::from([19u8; 32]),
            timestamp,
            transactions: vec![],
        }
    }

    /// Bit 0 is the only bit a span cannot derive from its own elements: it relates the first
    /// element to the block the span builds on.
    #[test]
    fn test_append_singular_batch_sets_same_ts_bit_zero() {
        let mut batch = SpanBatch { version: BatchType::SpanV2, ..Default::default() };
        batch.append_singular_batch(sibling_singular_batch(10, 100), 1, 100).unwrap();
        let same_ts_bits = batch.same_ts_bits.as_ref().unwrap();
        assert_eq!(same_ts_bits.get_bit(0), Some(1));

        let mut batch = SpanBatch { version: BatchType::SpanV2, ..Default::default() };
        batch.append_singular_batch(sibling_singular_batch(10, 100), 1, 98).unwrap();
        assert_eq!(batch.same_ts_bits.as_ref().unwrap().get_bit(0), Some(0));
    }

    /// A v1 span has no way to encode a sibling, so appending one is an encoder error rather than
    /// a silently dropped bit — and it must leave the span exactly as it was.
    #[test]
    fn test_append_sibling_to_v1_span_errors() {
        let mut batch = SpanBatch::default();
        let err = batch.append_singular_batch(sibling_singular_batch(10, 100), 1, 100).unwrap_err();
        assert_eq!(err, SpanBatchError::SameTimestampBitsMismatch);
        assert_eq!(batch, SpanBatch::default());
    }

    /// A span batch can only be encoded as one of the span wire versions, so the type byte
    /// [`crate::Batch::encode`] writes can never mislabel the payload that follows it.
    #[test]
    fn test_to_raw_span_batch_rejects_non_span_version() {
        let batch = SpanBatch {
            version: BatchType::Single,
            batches: vec![SpanBatchElement { epoch_num: 1, timestamp: 100, transactions: vec![] }],
            ..Default::default()
        };
        assert_eq!(batch.to_raw_span_batch().unwrap_err(), SpanBatchError::NotASpanVersion);
    }

    /// A group of two siblings followed by a second group of two, appended and then re-derived
    /// through the wire format.
    #[test]
    fn test_span_batch_v2_round_trip() {
        let block_time = 2;
        let genesis_timestamp = 1000;
        let mut batch =
            SpanBatch { version: BatchType::SpanV2, genesis_timestamp, ..Default::default() };
        for (seq_num, timestamp) in [1020, 1020, 1022, 1022].into_iter().enumerate() {
            batch
                .append_singular_batch(sibling_singular_batch(100, timestamp), seq_num as u64, 1018)
                .unwrap();
        }
        assert_eq!(batch.same_ts_bits, Some(SpanBatchBits::new(vec![0b1010])));

        let mut encoded = Vec::new();
        let raw = batch.to_raw_span_batch().unwrap();
        raw.encode(&mut encoded).unwrap();

        let mut decoded = RawSpanBatch::decode(&mut encoded.as_slice(), BatchType::SpanV2).unwrap();
        let derived = decoded.derive(block_time, genesis_timestamp, 10).unwrap();
        assert_eq!(derived.version, BatchType::SpanV2);
        assert_eq!(
            derived.batches.iter().map(|b| b.timestamp).collect::<Vec<_>>(),
            vec![1020, 1020, 1022, 1022]
        );
        // The bits reach validity checking through the derived span, not just the raw one.
        assert_eq!(derived.same_ts_bits, batch.same_ts_bits);
    }

    /// The same elements without siblings, as a v1 span: the derived span carries no bitlist at
    /// all, so nothing downstream can mistake it for a span that could hold siblings.
    #[test]
    fn test_span_batch_v1_carries_no_same_ts_bits() {
        let block_time = 2;
        let genesis_timestamp = 1000;
        let mut batch = SpanBatch { genesis_timestamp, ..Default::default() };
        for (seq_num, timestamp) in [1020, 1022].into_iter().enumerate() {
            batch
                .append_singular_batch(sibling_singular_batch(100, timestamp), seq_num as u64, 1018)
                .unwrap();
        }
        assert_eq!(batch.same_ts_bits, None);

        let mut encoded = Vec::new();
        batch.to_raw_span_batch().unwrap().encode(&mut encoded).unwrap();

        let mut decoded = RawSpanBatch::decode(&mut encoded.as_slice(), BatchType::Span).unwrap();
        let derived = decoded.derive(block_time, genesis_timestamp, 10).unwrap();
        assert_eq!(derived.version, BatchType::Span);
        assert_eq!(derived.same_ts_bits, None);
        assert_eq!(
            derived.batches.iter().map(|b| b.timestamp).collect::<Vec<_>>(),
            vec![1020, 1022]
        );
    }

    /// A chain with `block_time` 2 whose blocks may have siblings from timestamp 1000 on, at most
    /// `max_multi_blocks` of them.
    fn multi_block_cfg(max_multi_blocks: u64) -> RollupConfig {
        RollupConfig {
            block_time: 2,
            seq_window_size: 100,
            max_sequencer_drift: 1000,
            genesis: ChainGenesis {
                l2: BlockNumHash { number: 0, hash: B256::ZERO },
                l1: BlockNumHash { number: 0, hash: B256::ZERO },
                ..Default::default()
            },
            hardforks: HardForkConfig {
                delta_time: Some(0),
                holocene_time: Some(0),
                karst_time: Some(0),
                ..Default::default()
            },
            multi_block_time: Some(1000),
            max_multi_blocks: Some(max_multi_blocks),
            ..Default::default()
        }
    }

    fn l2_block(number: u64, timestamp: u64, origin: u64) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                number,
                timestamp,
                hash: B256::repeat_byte(number as u8),
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: origin, hash: B256::repeat_byte(origin as u8) },
            seq_num: 0,
        }
    }

    /// The safe chain the multi-block span tests build on: a group of two blocks at timestamp
    /// 1010 followed by one at 1012, all on L1 origin 100.
    fn multi_block_chain() -> Vec<L2BlockInfo> {
        vec![
            l2_block(9, 1008, 100),
            l2_block(10, 1010, 100),
            l2_block(11, 1010, 100),
            l2_block(12, 1012, 100),
        ]
    }

    fn multi_block_l1_origins() -> Vec<BlockInfo> {
        vec![
            BlockInfo {
                number: 100,
                timestamp: 900,
                hash: B256::repeat_byte(100),
                ..Default::default()
            },
            BlockInfo {
                number: 101,
                timestamp: 902,
                hash: B256::repeat_byte(101),
                ..Default::default()
            },
        ]
    }

    /// Builds a span batch of `(timestamp, epoch_num, is_sibling)` elements, naming `parent` as
    /// the block its first element builds on.
    fn multi_block_span(parent: L2BlockInfo, elements: &[(u64, u64, bool)]) -> SpanBatch {
        let mut same_ts_bits = SpanBatchBits::default();
        for (index, (_, _, is_sibling)) in elements.iter().enumerate() {
            same_ts_bits.set_bit(index, *is_sibling);
        }
        let last_epoch = elements.last().unwrap().1;
        SpanBatch {
            version: BatchType::SpanV2,
            parent_check: FixedBytes::<20>::from_slice(&parent.block_info.hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(
                &B256::repeat_byte(last_epoch as u8)[..20],
            ),
            same_ts_bits: Some(same_ts_bits),
            batches: elements
                .iter()
                .map(|(timestamp, epoch_num, _)| SpanBatchElement {
                    epoch_num: *epoch_num,
                    timestamp: *timestamp,
                    transactions: vec![],
                })
                .collect(),
            ..Default::default()
        }
    }

    /// Builds a Delta span batch over `timestamps`, all on L1 origin 100, naming `parent` as the
    /// block its first element builds on.
    fn delta_span(parent: L2BlockInfo, timestamps: &[u64]) -> SpanBatch {
        SpanBatch {
            version: BatchType::Span,
            parent_check: FixedBytes::<20>::from_slice(&parent.block_info.hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&B256::repeat_byte(100)[..20]),
            batches: timestamps
                .iter()
                .map(|timestamp| SpanBatchElement {
                    epoch_num: 100,
                    timestamp: *timestamp,
                    transactions: vec![],
                })
                .collect(),
            ..Default::default()
        }
    }

    /// The span continues the group the safe head is in, so bit 0 is set and the group's third
    /// block is still within `max_multi_blocks`.
    #[tokio::test]
    async fn test_check_batch_prefix_v2_continues_group_at_safe_head() {
        let cfg = multi_block_cfg(3);
        let chain = multi_block_chain();
        let safe_head = chain[2];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = multi_block_span(safe_head, &[(1010, 100, true), (1012, 100, false)]);
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Accepted(safe_head)
        );
    }

    /// The same span against a chain that allows only two blocks per timestamp: the group the
    /// parent is already in is what makes the element the third one, so the walk back over the
    /// safe chain is what catches it.
    #[tokio::test]
    async fn test_check_batch_prefix_v2_group_too_large_across_spans() {
        let cfg = multi_block_cfg(2);
        let chain = multi_block_chain();
        let safe_head = chain[2];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = multi_block_span(safe_head, &[(1010, 100, true), (1012, 100, false)]);
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Rejected(BatchValidity::Drop(
                BatchDropReason::MultiBlockGroupTooLarge
            ))
        );
    }

    /// A group entirely inside one span cannot exceed the maximum either.
    #[tokio::test]
    async fn test_check_batch_prefix_v2_group_too_large_within_span() {
        let cfg = multi_block_cfg(2);
        let chain = multi_block_chain();
        let safe_head = chain[3];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = multi_block_span(
            safe_head,
            &[(1014, 100, false), (1014, 100, true), (1014, 100, true)],
        );
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Rejected(BatchValidity::Drop(
                BatchDropReason::MultiBlockGroupTooLarge
            ))
        );
    }

    /// Only the first block of a timestamp may adopt a new L1 origin.
    #[tokio::test]
    async fn test_check_batch_prefix_v2_sibling_changes_origin() {
        let cfg = multi_block_cfg(3);
        let chain = multi_block_chain();
        let safe_head = chain[3];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = multi_block_span(safe_head, &[(1014, 100, false), (1014, 101, true)]);
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Rejected(BatchValidity::Drop(BatchDropReason::SiblingOriginMismatch))
        );
    }

    /// A span whose first element is a sibling inherits the parent's L1 origin, not the one the
    /// span would like to move to.
    #[tokio::test]
    async fn test_check_batch_prefix_v2_first_sibling_changes_origin() {
        let cfg = multi_block_cfg(3);
        let chain = multi_block_chain();
        let safe_head = chain[3];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = multi_block_span(safe_head, &[(1012, 101, true)]);
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Rejected(BatchValidity::Drop(BatchDropReason::SiblingOriginMismatch))
        );
    }

    /// A fork activation block is identified by its timestamp alone, so it must not have
    /// siblings.
    #[tokio::test]
    async fn test_check_batch_prefix_v2_sibling_at_fork_activation() {
        let mut cfg = multi_block_cfg(3);
        cfg.hardforks.lagoon_time = Some(1014);
        let chain = multi_block_chain();
        let safe_head = chain[3];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = multi_block_span(safe_head, &[(1014, 100, false), (1014, 100, true)]);
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Rejected(BatchValidity::Drop(BatchDropReason::SiblingsNotAllowed))
        );
    }

    /// The v2 format is gated on the L1 inclusion block, like the Delta span batch format.
    #[tokio::test]
    async fn test_check_batch_prefix_v2_before_activation() {
        let cfg = multi_block_cfg(3);
        let chain = multi_block_chain();
        let safe_head = chain[3];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = multi_block_span(safe_head, &[(1014, 100, false)]);
        let inclusion_block = BlockInfo { number: 110, timestamp: 999, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Rejected(BatchValidity::Drop(
                BatchDropReason::SpanBatchV2PreActivation
            ))
        );
    }

    /// A span that re-includes blocks the safe chain already holds: with bit 0 clear its parent
    /// is the last block below its first element's timestamp.
    #[tokio::test]
    async fn test_check_batch_prefix_v2_overlapping_bit_zero_clear() {
        let cfg = multi_block_cfg(3);
        let chain = multi_block_chain();
        let safe_head = chain[3];
        let parent = chain[0];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = multi_block_span(
            parent,
            &[(1010, 100, false), (1010, 100, true), (1012, 100, false), (1014, 100, false)],
        );
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Accepted(parent)
        );
    }

    /// The same overlap starting one block later: bit 0 is set, so the parent is the member of
    /// the group at that timestamp whose hash the span names, not the last one.
    #[tokio::test]
    async fn test_check_batch_prefix_v2_overlapping_bit_zero_set() {
        let cfg = multi_block_cfg(3);
        let chain = multi_block_chain();
        let safe_head = chain[3];
        let parent = chain[1];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch =
            multi_block_span(parent, &[(1010, 100, true), (1012, 100, false), (1014, 100, false)]);
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Accepted(parent)
        );
    }

    /// A span whose elements all sit at or below the safe head is old, even though its last
    /// element carries the safe head's own timestamp.
    #[tokio::test]
    async fn test_check_batch_prefix_v2_fully_overlapping_is_past() {
        let cfg = multi_block_cfg(3);
        let chain = multi_block_chain();
        let safe_head = chain[2];
        let parent = chain[0];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = multi_block_span(parent, &[(1010, 100, false), (1010, 100, true)]);
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Rejected(BatchValidity::Past)
        );
    }

    /// A Delta span, which cannot express siblings, still has to be placed by block number on a
    /// chain that has them: its elements are consecutive blocks after its parent, and counting
    /// timestamps would put its last element past a safe head that already holds it.
    #[tokio::test]
    async fn test_check_batch_prefix_v1_span_over_sibling_group_is_past() {
        let cfg = multi_block_cfg(3);
        let chain = vec![
            l2_block(9, 1008, 100),
            l2_block(10, 1010, 100),
            l2_block(11, 1010, 100),
            l2_block(12, 1010, 100),
            l2_block(13, 1012, 100),
        ];
        let safe_head = chain[4];
        let parent = chain[0];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = delta_span(parent, &[1010, 1012, 1014]);
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Rejected(BatchValidity::Past)
        );
    }

    /// The walk back towards the parent stops at the L2 genesis block, and a parent hash that
    /// matches a block at some other timestamp does not name that block as the parent.
    #[tokio::test]
    async fn test_check_batch_prefix_parent_timestamp_below_genesis_block() {
        let cfg = multi_block_cfg(3);
        let chain = vec![l2_block(0, 1000, 100), l2_block(1, 1000, 100), l2_block(2, 1002, 100)];
        let safe_head = chain[2];
        let genesis_block = chain[0];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = delta_span(genesis_block, &[1000, 1002, 1004]);
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Rejected(BatchValidity::Drop(
                BatchDropReason::SpanBatchNotOverlappedExactly
            ))
        );
    }

    /// A span reaching so far back that its parent would sit below the L2 genesis block.
    #[tokio::test]
    async fn test_check_batch_prefix_parent_below_genesis() {
        let cfg = multi_block_cfg(3);
        let chain = vec![l2_block(0, 1000, 100), l2_block(1, 1002, 100)];
        let safe_head = chain[1];
        let parent = chain[0];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = delta_span(parent, &[996, 998, 1000, 1002, 1004]);
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch_prefix(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            SpanBatchOutcome::Rejected(BatchValidity::Drop(
                BatchDropReason::SpanBatchParentBelowGenesis
            ))
        );
    }

    /// The pre-Holocene batch queue applies a span as a whole and cannot stream its elements, so
    /// it has no way to validate siblings.
    #[tokio::test]
    async fn test_check_batch_v2_pre_holocene() {
        let mut cfg = multi_block_cfg(3);
        cfg.hardforks.holocene_time = None;
        let chain = multi_block_chain();
        let safe_head = chain[3];
        let mut fetcher = TestBatchValidator { blocks: chain, ..Default::default() };
        let batch = multi_block_span(safe_head, &[(1014, 100, false)]);
        let inclusion_block = BlockInfo { number: 110, timestamp: 1100, ..Default::default() };
        assert_eq!(
            batch
                .check_batch(
                    &cfg,
                    &multi_block_l1_origins(),
                    safe_head,
                    &inclusion_block,
                    &mut fetcher
                )
                .await,
            BatchValidity::Drop(BatchDropReason::SpanBatchV2PreHolocene)
        );
    }

    /// Elements are skipped by block number, so a group that straddles the safe head keeps the
    /// siblings that are not on the chain yet — and each one carries its own sibling flag.
    #[test]
    fn test_get_singular_batches_skips_by_number() {
        let batch = multi_block_span(
            l2_block(9, 1008, 100),
            &[(1010, 100, false), (1010, 100, true), (1010, 100, true), (1012, 100, false)],
        );
        let safe_head = l2_block(11, 1010, 100);
        let singles = batch.get_singular_batches(&multi_block_l1_origins(), safe_head, 9).unwrap();
        assert_eq!(
            singles.iter().map(|s| (s.batch.timestamp, s.is_sibling)).collect::<Vec<_>>(),
            vec![(1010, true), (1012, false)]
        );
    }

    #[test]
    fn test_check_origin_hash() {
        let l1_origin_check = FixedBytes::from([17u8; 20]);
        let hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let batch = SpanBatch { l1_origin_check, ..Default::default() };
        assert!(batch.check_origin_hash(hash));
        // This hash has 19 matching bytes, the other 13 are zeros.
        let invalid = b256!("1111111111111111111111111111111111111100000000000000000000000000");
        assert!(!batch.check_origin_hash(invalid));
    }

    #[test]
    fn test_check_parent_hash() {
        let parent_check = FixedBytes::from([17u8; 20]);
        let hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let batch = SpanBatch { parent_check, ..Default::default() };
        assert!(batch.check_parent_hash(hash));
        // This hash has 19 matching bytes, the other 13 are zeros.
        let invalid = b256!("1111111111111111111111111111111111111100000000000000000000000000");
        assert!(!batch.check_parent_hash(invalid));
    }

    #[tokio::test]
    async fn test_check_batch_missing_l1_block_input() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            hardforks: HardForkConfig { holocene_time: Some(200), ..Default::default() },
            l2_chain_id: 10.into(),
            ..Default::default()
        };
        let l1_blocks = vec![];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 22, timestamp: 120, ..Default::default() },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 14, timestamp: 150, ..Default::default() };
        let mut fetcher: TestBatchValidator = TestBatchValidator::default();
        let batch = SpanBatch::default();
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Undecided
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("missing L1 block input, cannot proceed with batch checking"));
        assert!(logs[0].contains("chain_id: 10"));
        assert!(logs[0].contains("l1_origin_count: 0"));
        assert!(logs[0].contains("span_block_count: 0"));
        assert!(logs[0].contains("safe_head_number: 22"));
        assert!(logs[0].contains("safe_head_timestamp: 120"));
        assert!(logs[0].contains("safe_head_l1_origin_number: 9"));
        assert!(logs[0].contains("inclusion_l1_block_number: 14"));
        assert!(logs[0].contains("inclusion_l1_block_timestamp: 150"));
        assert!(logs[0].contains("holocene_active: false"));
    }

    #[tokio::test]
    async fn test_check_batches_is_empty() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig::default();
        let l1_blocks = vec![BlockInfo::default()];
        let l2_safe_head = L2BlockInfo::default();
        let inclusion_block = BlockInfo::default();
        let mut fetcher: TestBatchValidator = TestBatchValidator::default();
        let batch = SpanBatch::default();
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Undecided
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("empty span batch, cannot proceed with batch checking"));
    }

    #[tokio::test]
    async fn test_singular_batches_outdated_l1_origin() {
        let l1_block = BlockInfo { number: 10, timestamp: 20, ..Default::default() };
        let l1_blocks = vec![l1_block];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 10, ..Default::default() },
            l1_origin: BlockNumHash { number: 10, ..Default::default() },
            ..Default::default()
        };
        let first = SpanBatchElement { epoch_num: 9, timestamp: 20, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 10, timestamp: 30, ..Default::default() };
        let batch = SpanBatch { batches: vec![first, second], ..Default::default() };
        assert_eq!(
            batch.get_singular_batches(&l1_blocks, l2_safe_head, 0),
            Err(SpanBatchError::L1OriginBeforeSafeHead),
        );
    }

    #[tokio::test]
    async fn test_singular_batches_missing_l1_origin() {
        let l1_block = BlockInfo { number: 10, timestamp: 20, ..Default::default() };
        let l1_blocks = vec![l1_block];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 10, ..Default::default() },
            l1_origin: BlockNumHash { number: 10, ..Default::default() },
            ..Default::default()
        };
        let first = SpanBatchElement { epoch_num: 10, timestamp: 20, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 30, ..Default::default() };
        let batch = SpanBatch { batches: vec![first, second], ..Default::default() };
        assert_eq!(
            batch.get_singular_batches(&l1_blocks, l2_safe_head, 0),
            Err(SpanBatchError::MissingL1Origin),
        );
    }

    #[tokio::test]
    async fn test_eager_block_missing_origins() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig::default();
        let block = BlockInfo { number: 9, ..Default::default() };
        let l1_blocks = vec![block];
        let l2_safe_head = L2BlockInfo::default();
        let inclusion_block = BlockInfo::default();
        let mut fetcher: TestBatchValidator = TestBatchValidator::default();
        let first = SpanBatchElement { epoch_num: 10, ..Default::default() };
        let batch = SpanBatch { batches: vec![first], ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Undecided
        );
        let logs = trace_store.get_by_level(Level::INFO);
        assert_eq!(logs.len(), 1);
        let str = alloc::format!(
            "eager batch wants to advance current epoch {:?}, but could not without more L1 blocks",
            block.id()
        );
        assert!(logs[0].contains(&str));
    }

    #[tokio::test]
    async fn test_check_batch_delta_inactive() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            hardforks: HardForkConfig { delta_time: Some(10), ..Default::default() },
            ..Default::default()
        };
        let block = BlockInfo { number: 10, timestamp: 9, ..Default::default() };
        let l1_blocks = vec![block];
        let l2_safe_head = L2BlockInfo::default();
        let inclusion_block = BlockInfo::default();
        let mut fetcher: TestBatchValidator = TestBatchValidator::default();
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let batch = SpanBatch { batches: vec![first], ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::SpanBatchPreDelta)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        let str = alloc::format!(
            "received SpanBatch (id {:?}) with L1 origin (timestamp {}) before Delta hard fork",
            block.id(),
            block.timestamp
        );
        assert!(logs[0].contains(&str));
    }

    #[tokio::test]
    async fn test_check_batch_out_of_order() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let block = BlockInfo { number: 10, timestamp: 10, ..Default::default() };
        let l1_blocks = vec![block];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 10, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        let mut fetcher: TestBatchValidator = TestBatchValidator::default();
        let first = SpanBatchElement { epoch_num: 10, timestamp: 21, ..Default::default() };
        let batch = SpanBatch { batches: vec![first], ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Future
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains(
            "received out-of-order batch for future processing after next batch (21 > 20)"
        ));
    }

    #[tokio::test]
    async fn test_check_batch_no_new_blocks() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let block = BlockInfo { number: 10, timestamp: 10, ..Default::default() };
        let l1_blocks = vec![block];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 20, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 12, timestamp: 25, ..Default::default() };
        let mut fetcher: TestBatchValidator = TestBatchValidator::default();
        let first = SpanBatchElement { epoch_num: 10, timestamp: 0, ..Default::default() };
        let last = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let batch = SpanBatch { batches: vec![first, last], ..Default::default() };
        for (holocene_time, expected_validity) in [
            (Some(26), BatchValidity::Drop(BatchDropReason::SpanBatchNoNewBlocksPreHolocene)),
            (Some(0), BatchValidity::Past),
        ] {
            let cfg = RollupConfig {
                hardforks: HardForkConfig {
                    delta_time: Some(0),
                    holocene_time,
                    ..Default::default()
                },
                block_time: 10,
                l2_chain_id: 10.into(),
                ..Default::default()
            };
            assert_eq!(
                batch
                    .check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher,)
                    .await,
                expected_validity
            );
        }
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 2);
        for log in &logs {
            assert!(log.contains("span batch has no new blocks after safe head"));
            assert!(log.contains("chain_id: 10"));
            assert!(log.contains("span_start_timestamp: 0"));
            assert!(log.contains("span_final_timestamp: 10"));
            assert!(log.contains("safe_head_timestamp: 20"));
            assert!(log.contains("next_expected_timestamp: 30"));
            assert!(log.contains("span_lag_seconds: 20"));
            assert!(log.contains("inclusion_l1_block_number: 12"));
            assert!(log.contains("inclusion_l1_block_timestamp: 25"));
        }
        assert!(logs[0].contains("holocene_active: false"));
        assert!(logs[0].contains("batch_validity: \"drop\""));
        assert!(logs[1].contains("holocene_active: true"));
        assert!(logs[1].contains("batch_validity: \"past\""));
    }

    #[tokio::test]
    async fn test_check_batch_overlapping_blocks_tx_count_mismatch() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            max_sequencer_drift: 1000,
            ..Default::default()
        };
        let l1_blocks = gen_l1_blocks(9, 3, 0, 10);
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 10, timestamp: 20, ..Default::default() },
            l1_origin: BlockNumHash { number: 11, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        let mut fetcher: TestBatchValidator = TestBatchValidator {
            op_blocks: vec![OpBlock {
                header: Header { number: 9, ..Default::default() },
                body: alloy_consensus::BlockBody {
                    transactions: Vec::new(),
                    ommers: Vec::new(),
                    withdrawals: None,
                },
            }],
            blocks: vec![
                L2BlockInfo {
                    block_info: BlockInfo { number: 8, timestamp: 0, ..Default::default() },
                    l1_origin: BlockNumHash { number: 9, ..Default::default() },
                    ..Default::default()
                },
                L2BlockInfo {
                    block_info: BlockInfo { number: 9, timestamp: 10, ..Default::default() },
                    l1_origin: BlockNumHash { number: 10, ..Default::default() },
                    ..Default::default()
                },
                L2BlockInfo {
                    block_info: BlockInfo { number: 10, timestamp: 20, ..Default::default() },
                    l1_origin: BlockNumHash { number: 11, ..Default::default() },
                    ..Default::default()
                },
            ],
            ..Default::default()
        };
        let first = SpanBatchElement {
            epoch_num: 10,
            timestamp: 10,
            transactions: vec![Bytes(vec![EIP1559_TX_TYPE_ID].into())],
        };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let third = SpanBatchElement { epoch_num: 11, timestamp: 30, ..Default::default() };
        let batch = SpanBatch { batches: vec![first, second, third], ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::OverlappedTxCountMismatch)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains(
            "overlapped block's tx count does not match, safe_block_txs: 0, batch_txs: 1"
        ));
    }

    #[tokio::test]
    async fn test_check_batch_overlapping_blocks_tx_mismatch() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            max_sequencer_drift: 1000,
            ..Default::default()
        };
        let l1_blocks = gen_l1_blocks(9, 3, 0, 10);
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 10, timestamp: 20, ..Default::default() },
            l1_origin: BlockNumHash { number: 11, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        let mut fetcher: TestBatchValidator = TestBatchValidator {
            op_blocks: vec![OpBlock {
                header: Header { number: 9, ..Default::default() },
                body: alloy_consensus::BlockBody {
                    transactions: vec![op_alloy_consensus::OpTxEnvelope::Eip1559(
                        alloy_consensus::Signed::new_unchecked(
                            alloy_consensus::TxEip1559 {
                                chain_id: 0,
                                nonce: 0,
                                gas_limit: 2,
                                max_fee_per_gas: 1,
                                max_priority_fee_per_gas: 1,
                                to: alloy_primitives::TxKind::Create,
                                value: alloy_primitives::U256::from(3),
                                ..Default::default()
                            },
                            alloy_primitives::Signature::test_signature(),
                            alloy_primitives::B256::ZERO,
                        ),
                    )],
                    ommers: Vec::new(),
                    withdrawals: None,
                },
            }],
            blocks: vec![
                L2BlockInfo {
                    block_info: BlockInfo { number: 8, timestamp: 0, ..Default::default() },
                    l1_origin: BlockNumHash { number: 9, ..Default::default() },
                    ..Default::default()
                },
                L2BlockInfo {
                    block_info: BlockInfo { number: 9, timestamp: 10, ..Default::default() },
                    l1_origin: BlockNumHash { number: 10, ..Default::default() },
                    ..Default::default()
                },
                L2BlockInfo {
                    block_info: BlockInfo { number: 10, timestamp: 20, ..Default::default() },
                    l1_origin: BlockNumHash { number: 11, ..Default::default() },
                    ..Default::default()
                },
            ],
            ..Default::default()
        };
        let first = SpanBatchElement {
            epoch_num: 10,
            timestamp: 10,
            transactions: vec![Bytes(vec![EIP1559_TX_TYPE_ID].into())],
        };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let third = SpanBatchElement { epoch_num: 11, timestamp: 30, ..Default::default() };
        let batch = SpanBatch { batches: vec![first, second, third], ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::OverlappedTxMismatch)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("overlapped block's transaction does not match"));
    }

    #[tokio::test]
    async fn test_check_batch_block_timestamp_lt_l1_origin() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let l1_block = BlockInfo { number: 10, timestamp: 20, ..Default::default() };
        let l1_blocks = vec![l1_block];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 10, ..Default::default() },
            l1_origin: BlockNumHash { number: 10, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        let mut fetcher: TestBatchValidator = TestBatchValidator::default();
        let first = SpanBatchElement { epoch_num: 10, timestamp: 20, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 10, timestamp: 19, ..Default::default() };
        let third = SpanBatchElement { epoch_num: 10, timestamp: 30, ..Default::default() };
        let batch = SpanBatch { batches: vec![first, second, third], ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::TimestampBeforeL1Origin)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        let str = alloc::format!(
            "batch timestamp is less than L1 origin timestamp, l2_timestamp: 19, l1_timestamp: 20, origin: {:?}",
            l1_block.id(),
        );
        assert!(logs[0].contains(&str));
    }

    #[tokio::test]
    async fn test_check_batch_misaligned_timestamp() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let block = BlockInfo { number: 10, timestamp: 10, ..Default::default() };
        let l1_blocks = vec![block];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 10, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        let mut fetcher: TestBatchValidator = TestBatchValidator::default();
        let first = SpanBatchElement { epoch_num: 10, timestamp: 11, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 21, ..Default::default() };
        let batch = SpanBatch { batches: vec![first, second], ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::SpanBatchMisalignedTimestamp)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("batch has misaligned timestamp, block time is too short"));
    }

    #[tokio::test]
    async fn test_check_batch_misaligned_without_overlap() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let block = BlockInfo { number: 10, timestamp: 10, ..Default::default() };
        let l1_blocks = vec![block];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { timestamp: 10, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        let mut fetcher: TestBatchValidator = TestBatchValidator::default();
        let first = SpanBatchElement { epoch_num: 10, timestamp: 8, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let batch = SpanBatch { batches: vec![first, second], ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::SpanBatchNotOverlappedExactly)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("batch has misaligned timestamp, not overlapped exactly"));
    }

    #[tokio::test]
    async fn test_check_batch_failed_to_fetch_l2_block() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let block = BlockInfo { number: 10, timestamp: 10, ..Default::default() };
        let l1_blocks = vec![block];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 41, timestamp: 10, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        let mut fetcher: TestBatchValidator = TestBatchValidator::default();
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let batch = SpanBatch { batches: vec![first, second], ..Default::default() };
        // parent number = 41 - (10 - 10) / 10 - 1 = 40
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Undecided
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("failed to fetch L2 block number 40: Block not found"));
    }

    #[tokio::test]
    async fn test_check_batch_parent_hash_fail() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let block = BlockInfo { number: 10, timestamp: 10, ..Default::default() };
        let l1_blocks = vec![block];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 41, timestamp: 10, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo::default();
        let l2_block = L2BlockInfo {
            block_info: BlockInfo { number: 41, timestamp: 10, ..Default::default() },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        fetcher.short_circuit = true;
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let batch = SpanBatch {
            batches: vec![first, second],
            parent_check: FixedBytes::<20>::from_slice(
                &b256!("1111111111111111111111111111111111111111000000000000000000000000")[..20],
            ),
            ..Default::default()
        };
        // parent number = 41 - (10 - 10 + 10) / 10 = 40
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::ParentHashMismatch)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("no block at timestamp 0 matches parent hash check"));
    }

    #[tokio::test]
    async fn test_check_sequence_window_expired() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let block = BlockInfo { number: 10, timestamp: 10, ..Default::default() };
        let l1_blocks = vec![block];
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 41, timestamp: 10, parent_hash, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo {
                number: 40,
                hash: parent_hash,
                timestamp: 0,
                ..Default::default()
            },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let batch = SpanBatch {
            batches: vec![first, second],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            ..Default::default()
        };
        // parent number = 41 - (10 - 10) / 10 - 1 = 40
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::IncludedTooLate)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("batch was included too late, sequence window expired"));
    }

    #[tokio::test]
    async fn test_starting_epoch_too_far_ahead() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let block = BlockInfo { number: 10, timestamp: 10, ..Default::default() };
        let l1_blocks = vec![block];
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 41, timestamp: 10, parent_hash, ..Default::default() },
            l1_origin: BlockNumHash { number: 8, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo {
                number: 40,
                hash: parent_hash,
                timestamp: 0,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 8, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let batch = SpanBatch {
            batches: vec![first, second],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            ..Default::default()
        };
        // parent number = 41 - (10 - 10) / 10 - 1 = 40
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::EpochTooFarInFuture)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        let str = "batch is for future epoch too far ahead, while it has the next timestamp, so it must be invalid. starting epoch: 10 | next epoch: 9";
        assert!(logs[0].contains(str));
    }

    #[tokio::test]
    async fn test_check_batch_epoch_hash_mismatch() {
        use crate::alloc::string::ToString;
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let l1_block_hash =
            b256!("3333333333333333333333333333333333333333000000000000000000000000");
        let block =
            BlockInfo { number: 11, timestamp: 10, hash: l1_block_hash, ..Default::default() };
        let l1_blocks = vec![block];
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                number: 41,
                timestamp: 10,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo {
                number: 40,
                timestamp: 0,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let batch = SpanBatch {
            batches: vec![first, second],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::EpochHashMismatch)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        let str = "batch is for different L1 chain, epoch hash does not match".to_string();
        assert!(logs[0].contains(&str));
    }

    #[tokio::test]
    async fn test_need_more_l1_blocks() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let l1_block_hash =
            b256!("3333333333333333333333333333333333333333000000000000000000000000");
        let block =
            BlockInfo { number: 10, timestamp: 10, hash: l1_block_hash, ..Default::default() };
        let l1_blocks = vec![block];
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 41, timestamp: 10, parent_hash, ..Default::default() },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo {
                number: 40,
                timestamp: 0,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let batch = SpanBatch {
            batches: vec![first, second],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_block_hash[..20]),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Undecided
        );
        let logs = trace_store.get_by_level(Level::INFO);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("need more l1 blocks to check entire origins of span batch"));
    }

    #[tokio::test]
    async fn test_drop_batch_epoch_too_old() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let l1_block_hash =
            b256!("3333333333333333333333333333333333333333000000000000000000000000");
        let block =
            BlockInfo { number: 11, timestamp: 10, hash: l1_block_hash, ..Default::default() };
        let l1_blocks = vec![block];
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 41, timestamp: 10, parent_hash, ..Default::default() },
            l1_origin: BlockNumHash { number: 13, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo {
                number: 40,
                timestamp: 0,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 14, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let batch = SpanBatch {
            batches: vec![first, second],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_block_hash[..20]),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::EpochTooOld)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        let str = alloc::format!(
            "dropped batch, epoch is too old, minimum: {:?}",
            l2_block.block_info.id(),
        );
        assert!(logs[0].contains(&str));
    }

    #[tokio::test]
    async fn test_check_batch_exceeds_max_seq_drif() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            max_sequencer_drift: 0,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let l1_blocks = gen_l1_blocks(9, 3, 10, 0);
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                number: 41,
                timestamp: 10,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo { number: 40, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let first = SpanBatchElement { epoch_num: 10, timestamp: 20, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 10, timestamp: 30, ..Default::default() };
        let third = SpanBatchElement { epoch_num: 11, timestamp: 40, ..Default::default() };
        let batch = SpanBatch {
            batches: vec![first, second, third],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_blocks[1].hash[..20]),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::SequencerDriftNotAdoptedNextOrigin)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("batch exceeded sequencer time drift without adopting next origin, and next L1 origin would have been valid"));
    }

    #[tokio::test]
    async fn test_continuing_with_empty_batch() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default()
            .with(layer)
            .with(tracing_subscriber::fmt::layer());
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            max_sequencer_drift: 0,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        // Create two L1 blocks with number,timestamp: (10,10) and (11,40) so that the second batch
        // in the span batch is valid even though it doesn't advance the origin, because its
        // timestamp is 30 < 40. Then the third batch advances the origin to L1 block 11
        // with timestamp 40, which is also the third batch's timestamp.
        let l1_blocks = gen_l1_blocks(10, 2, 10, 30);
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                number: 41,
                timestamp: 10,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo { number: 40, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let first = SpanBatchElement { epoch_num: 10, timestamp: 20, transactions: vec![] };
        let second = SpanBatchElement { epoch_num: 10, timestamp: 30, transactions: vec![] };
        let third = SpanBatchElement { epoch_num: 11, timestamp: 40, transactions: vec![] };
        let batch = SpanBatch {
            batches: vec![first, second, third],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_blocks[1].hash[..20]),
            txs: SpanBatchTransactions::default(),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Accept
        );
        let infos = trace_store.get_by_level(Level::INFO);
        assert_eq!(infos.len(), 1);
        assert!(infos[0].contains(
            "continuing with empty batch before late L1 block to preserve L2 time invariant"
        ));
    }

    #[tokio::test]
    async fn test_check_batch_exceeds_sequencer_time_drift() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            max_sequencer_drift: 0,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let l1_blocks = gen_l1_blocks(9, 3, 10, 0);
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                number: 41,
                timestamp: 10,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo { number: 40, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let first = SpanBatchElement {
            epoch_num: 10,
            timestamp: 20,
            transactions: vec![Default::default()],
        };
        let second = SpanBatchElement {
            epoch_num: 10,
            timestamp: 20,
            transactions: vec![Default::default()],
        };
        let third = SpanBatchElement {
            epoch_num: 11,
            timestamp: 20,
            transactions: vec![Default::default()],
        };
        let batch = SpanBatch {
            batches: vec![first, second, third],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_blocks[0].hash[..20]),
            txs: SpanBatchTransactions::default(),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::SequencerDriftExceeded)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("batch exceeded sequencer time drift, sequencer must adopt new L1 origin to include transactions again, max_time: 10"));
    }

    #[tokio::test]
    async fn test_check_batch_empty_txs() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            max_sequencer_drift: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let l1_block_hash =
            b256!("3333333333333333333333333333333333333333000000000000000000000000");
        let l1_a =
            BlockInfo { number: 10, timestamp: 5, hash: l1_block_hash, ..Default::default() };
        let l1_b =
            BlockInfo { number: 11, timestamp: 10, hash: l1_block_hash, ..Default::default() };
        let l1_c =
            BlockInfo { number: 12, timestamp: 21, hash: l1_block_hash, ..Default::default() };
        let l1_blocks = vec![l1_a, l1_b, l1_c];
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                number: 41,
                timestamp: 10,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo { number: 40, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let first = SpanBatchElement {
            epoch_num: 10,
            timestamp: 20,
            transactions: vec![Default::default()],
        };
        let second = SpanBatchElement {
            epoch_num: 10,
            timestamp: 20,
            transactions: vec![Default::default()],
        };
        let third = SpanBatchElement { epoch_num: 11, timestamp: 20, transactions: vec![] };
        let batch = SpanBatch {
            batches: vec![first, second, third],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_block_hash[..20]),
            txs: SpanBatchTransactions::default(),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::EmptyTransaction)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("transaction data must not be empty, but found empty tx"));
    }

    #[tokio::test]
    async fn test_check_batch_with_deposit_tx() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            max_sequencer_drift: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let l1_blocks = gen_l1_blocks(9, 3, 0, 10);
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                number: 41,
                timestamp: 10,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo { number: 40, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let filler_bytes = Bytes::copy_from_slice(&[EIP1559_TX_TYPE_ID]);
        let first = SpanBatchElement {
            epoch_num: 10,
            timestamp: 20,
            transactions: vec![filler_bytes.clone()],
        };
        let second = SpanBatchElement {
            epoch_num: 10,
            timestamp: 20,
            transactions: vec![Bytes::copy_from_slice(&[u8::from(OpTxType::Deposit)])],
        };
        let third =
            SpanBatchElement { epoch_num: 11, timestamp: 20, transactions: vec![filler_bytes] };
        let batch = SpanBatch {
            batches: vec![first, second, third],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_blocks[0].hash[..20]),
            txs: SpanBatchTransactions::default(),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::DepositTransaction)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("sequencers may not embed any deposits into batch data, but found tx that has one, tx_index: 0"));
    }

    #[tokio::test]
    async fn test_check_batch_with_eip7702_tx() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            max_sequencer_drift: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let l1_blocks = gen_l1_blocks(9, 3, 0, 10);
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                number: 41,
                timestamp: 10,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo { number: 40, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let filler_bytes = Bytes::copy_from_slice(&[EIP1559_TX_TYPE_ID]);
        let first = SpanBatchElement {
            epoch_num: 10,
            timestamp: 20,
            transactions: vec![filler_bytes.clone()],
        };
        let second = SpanBatchElement {
            epoch_num: 10,
            timestamp: 20,
            transactions: vec![Bytes::copy_from_slice(&[u8::from(
                alloy_consensus::TxType::Eip7702,
            )])],
        };
        let third =
            SpanBatchElement { epoch_num: 11, timestamp: 20, transactions: vec![filler_bytes] };
        let batch = SpanBatch {
            batches: vec![first, second, third],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_blocks[0].hash[..20]),
            txs: SpanBatchTransactions::default(),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::Eip7702PreIsthmus)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(
            logs[0].contains("EIP-7702 transactions are not supported pre-isthmus. tx_index: 0")
        );
    }

    #[tokio::test]
    async fn test_check_batch_with_post_exec_tx_pre_sdm() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            max_sequencer_drift: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let l1_blocks = gen_l1_blocks(9, 3, 0, 10);
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                number: 41,
                timestamp: 10,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo { number: 40, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let filler_bytes = Bytes::copy_from_slice(&[EIP1559_TX_TYPE_ID]);
        let first = SpanBatchElement {
            epoch_num: 10,
            timestamp: 20,
            transactions: vec![filler_bytes.clone()],
        };
        let second = SpanBatchElement {
            epoch_num: 10,
            timestamp: 20,
            transactions: vec![Bytes::copy_from_slice(&[POST_EXEC_TX_TYPE_ID])],
        };
        let third =
            SpanBatchElement { epoch_num: 11, timestamp: 20, transactions: vec![filler_bytes] };
        let batch = SpanBatch {
            batches: vec![first, second, third],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_blocks[0].hash[..20]),
            txs: SpanBatchTransactions::default(),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::PostExecPreLagoon)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(
            logs[0].contains("PostExec transactions are not supported pre-Lagoon. tx_index: 0")
        );
    }

    #[tokio::test]
    async fn test_check_batch_failed_to_fetch_payload() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let l1_block_hash =
            b256!("3333333333333333333333333333333333333333000000000000000000000000");
        let block =
            BlockInfo { number: 11, timestamp: 10, hash: l1_block_hash, ..Default::default() };
        let l1_blocks = vec![block];
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 41, timestamp: 10, parent_hash, ..Default::default() },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo {
                number: 40,
                timestamp: 0,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let mut fetcher: TestBatchValidator =
            TestBatchValidator { blocks: vec![l2_block], ..Default::default() };
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let batch = SpanBatch {
            batches: vec![first, second],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_block_hash[..20]),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Undecided
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("failed to fetch block number 41: L2 Block not found"));
    }

    #[tokio::test]
    async fn test_check_batch_failed_to_extract_l2_block_info() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let cfg = RollupConfig {
            seq_window_size: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            ..Default::default()
        };
        let l1_block_hash =
            b256!("3333333333333333333333333333333333333333000000000000000000000000");
        let block =
            BlockInfo { number: 11, timestamp: 10, hash: l1_block_hash, ..Default::default() };
        let l1_blocks = vec![block];
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 41, timestamp: 10, parent_hash, ..Default::default() },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo {
                number: 40,
                timestamp: 0,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let block = OpBlock {
            header: Header { number: 41, ..Default::default() },
            body: alloy_consensus::BlockBody {
                transactions: Vec::new(),
                ommers: Vec::new(),
                withdrawals: None,
            },
        };
        let mut fetcher: TestBatchValidator = TestBatchValidator {
            blocks: vec![l2_block],
            op_blocks: vec![block],
            ..Default::default()
        };
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let batch = SpanBatch {
            batches: vec![first, second],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_block_hash[..20]),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::L2BlockInfoExtractionFailed)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        let str = alloc::format!(
            "failed to extract L2BlockInfo from execution payload, hash: {:?}",
            b256!("0e2ee9abe94ee4514b170d7039d8151a7469d434a8575dbab5bd4187a27732dd"),
        );
        assert!(logs[0].contains(&str));
    }

    #[tokio::test]
    async fn test_overlapped_blocks_origin_mismatch() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let payload_block_hash =
            b256!("0e2ee9abe94ee4514b170d7039d8151a7469d434a8575dbab5bd4187a27732dd");
        let cfg = RollupConfig {
            seq_window_size: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            genesis: ChainGenesis {
                l2: BlockNumHash { number: 41, hash: payload_block_hash },
                ..Default::default()
            },
            ..Default::default()
        };
        let l1_block_hash =
            b256!("3333333333333333333333333333333333333333000000000000000000000000");
        let block =
            BlockInfo { number: 11, timestamp: 10, hash: l1_block_hash, ..Default::default() };
        let l1_blocks = vec![block];
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 41, timestamp: 10, parent_hash, ..Default::default() },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo {
                number: 40,
                hash: parent_hash,
                timestamp: 0,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let block = OpBlock {
            header: Header { number: 41, ..Default::default() },
            body: alloy_consensus::BlockBody {
                transactions: Vec::new(),
                ommers: Vec::new(),
                withdrawals: None,
            },
        };
        let mut fetcher: TestBatchValidator = TestBatchValidator {
            blocks: vec![l2_block],
            op_blocks: vec![block],
            ..Default::default()
        };
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let batch = SpanBatch {
            batches: vec![first, second],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_block_hash[..20]),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::OverlappedL1OriginMismatch)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("overlapped block's L1 origin number does not match"));
    }

    #[tokio::test]
    async fn test_overlapped_blocks_origin_outdated() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let cfg = RollupConfig {
            seq_window_size: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            genesis: ChainGenesis {
                l2: BlockNumHash { number: 40, hash: parent_hash },
                ..Default::default()
            },
            ..Default::default()
        };
        let l1_block_hash =
            b256!("3333333333333333333333333333333333333333000000000000000000000000");
        let l1_block =
            BlockInfo { number: 10, timestamp: 5, hash: l1_block_hash, ..Default::default() };
        let l1_blocks = vec![l1_block];
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 41, timestamp: 10, parent_hash, ..Default::default() },
            l1_origin: l1_block.id(),
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_parent = L2BlockInfo {
            block_info: BlockInfo {
                number: 40,
                hash: parent_hash,
                timestamp: 0,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        // A valid overlapped canonical block (L1 info deposit only, origin 9), so the overlap
        // content checks pass and the per-element origin checks are reached.
        let l1_info = crate::L1BlockInfoBedrock::new(
            9,
            0,
            0,
            B256::ZERO,
            0,
            alloy_primitives::Address::ZERO,
            alloy_primitives::U256::ZERO,
            alloy_primitives::U256::ZERO,
        );
        let info_tx = op_alloy_consensus::OpTxEnvelope::Deposit(alloy_primitives::Sealed::new(
            op_alloy_consensus::TxDeposit {
                input: l1_info.encode_calldata(),
                ..Default::default()
            },
        ));
        let block = OpBlock {
            header: Header { number: 41, ..Default::default() },
            body: alloy_consensus::BlockBody {
                transactions: vec![info_tx],
                ommers: Vec::new(),
                withdrawals: None,
            },
        };
        let mut fetcher: TestBatchValidator = TestBatchValidator {
            blocks: vec![l2_parent],
            op_blocks: vec![block],
            ..Default::default()
        };
        let first = SpanBatchElement { epoch_num: 9, timestamp: 10, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 9, timestamp: 20, ..Default::default() };
        let third = SpanBatchElement { epoch_num: 10, timestamp: 30, ..Default::default() };
        let batch = SpanBatch {
            batches: vec![first, second, third],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_block_hash[..20]),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop(BatchDropReason::L1OriginBeforeSafeHead)
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("batch L1 origin is before safe head L1 origin"));
    }

    #[tokio::test]
    async fn test_check_batch_valid_with_genesis_epoch() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        let subscriber = tracing_subscriber::Registry::default().with(layer);
        let _guard = tracing::subscriber::set_default(subscriber);

        let payload_block_hash =
            b256!("0e2ee9abe94ee4514b170d7039d8151a7469d434a8575dbab5bd4187a27732dd");
        let cfg = RollupConfig {
            seq_window_size: 100,
            hardforks: HardForkConfig { delta_time: Some(0), ..Default::default() },
            block_time: 10,
            genesis: ChainGenesis {
                l2: BlockNumHash { number: 41, hash: payload_block_hash },
                l1: BlockNumHash { number: 10, ..Default::default() },
                ..Default::default()
            },
            ..Default::default()
        };
        let l1_block_hash =
            b256!("3333333333333333333333333333333333333333000000000000000000000000");
        let block =
            BlockInfo { number: 11, timestamp: 10, hash: l1_block_hash, ..Default::default() };
        let l1_blocks = vec![block];
        let parent_hash = b256!("1111111111111111111111111111111111111111000000000000000000000000");
        let l2_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                number: 41,
                timestamp: 10,
                hash: parent_hash,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let inclusion_block = BlockInfo { number: 50, ..Default::default() };
        let l2_block = L2BlockInfo {
            block_info: BlockInfo {
                number: 40,
                hash: parent_hash,
                timestamp: 0,
                ..Default::default()
            },
            l1_origin: BlockNumHash { number: 9, ..Default::default() },
            ..Default::default()
        };
        let block = OpBlock {
            header: Header { number: 41, ..Default::default() },
            body: alloy_consensus::BlockBody {
                transactions: Vec::new(),
                ommers: Vec::new(),
                withdrawals: None,
            },
        };
        let mut fetcher: TestBatchValidator = TestBatchValidator {
            blocks: vec![l2_block],
            op_blocks: vec![block],
            ..Default::default()
        };
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let second = SpanBatchElement { epoch_num: 11, timestamp: 20, ..Default::default() };
        let batch = SpanBatch {
            batches: vec![first, second],
            parent_check: FixedBytes::<20>::from_slice(&parent_hash[..20]),
            l1_origin_check: FixedBytes::<20>::from_slice(&l1_block_hash[..20]),
            ..Default::default()
        };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Accept
        );
        assert!(trace_store.is_empty());
    }
}
