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
    BatchValidationProvider, BatchValidity, BlockInfo, L2BlockInfo, RawSpanBatch, SingleBatch,
    SpanBatchBits, SpanBatchElement, SpanBatchError, SpanBatchPayload, SpanBatchPrefix,
    SpanBatchTransactions,
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
#[derive(Debug, Default, Clone, PartialEq, Eq)]
pub struct SpanBatch {
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
        if self.batches.is_empty() {
            return Err(SpanBatchError::EmptySpanBatch);
        }

        // These should never error since we check for an empty batch above.
        let span_start = self.batches.first().ok_or(SpanBatchError::EmptySpanBatch)?;
        let span_end = self.batches.last().ok_or(SpanBatchError::EmptySpanBatch)?;

        Ok(RawSpanBatch {
            prefix: SpanBatchPrefix {
                rel_timestamp: span_start.timestamp - self.genesis_timestamp,
                l1_origin_num: span_end.epoch_num,
                parent_check: self.parent_check,
                l1_origin_check: self.l1_origin_check,
            },
            payload: SpanBatchPayload {
                block_count: self.batches.len() as u64,
                origin_bits: self.origin_bits.clone(),
                block_tx_counts: self.block_tx_counts.clone(),
                txs: self.txs.clone(),
            },
        })
    }

    /// Converts all [`SpanBatchElement`]s after the L2 safe head to [`SingleBatch`]es. The
    /// resulting [`SingleBatch`]es do not contain a parent hash, as it is populated by the
    /// Batch Queue stage.
    pub fn get_singular_batches(
        &self,
        l1_origins: &[BlockInfo],
        l2_safe_head: L2BlockInfo,
    ) -> Result<Vec<SingleBatch>, SpanBatchError> {
        let mut single_batches = Vec::with_capacity(self.batches.len());
        let mut origin_index = 0;
        for batch in &self.batches {
            if batch.timestamp <= l2_safe_head.block_info.timestamp {
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
            single_batches.push(single_batch);
        }
        Ok(single_batches)
    }

    /// Append a [`SingleBatch`] to the [`SpanBatch`]. Updates the L1 origin check if need be.
    pub fn append_singular_batch(
        &mut self,
        singular_batch: SingleBatch,
        seq_num: u64,
    ) -> Result<(), SpanBatchError> {
        // If the new element is not ordered with respect to the last element, panic.
        if !self.batches.is_empty() && self.peek(0).timestamp > singular_batch.timestamp {
            panic!("Batch is not ordered");
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
        let (prefix_validity, parent_block) =
            self.check_batch_prefix(cfg, l1_blocks, l2_safe_head, inclusion_block, fetcher).await;
        if !matches!(prefix_validity, BatchValidity::Accept) {
            return prefix_validity;
        }

        let starting_epoch_num = self.starting_epoch_num();
        let parent_block = parent_block.expect("parent_block must be Some");

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
                return BatchValidity::Drop;
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
                return BatchValidity::Drop;
            };
            origin_index += offset;

            if i > 0 {
                origin_advanced = false;
                if batch_epoch > self.batches[i - 1].epoch_num {
                    origin_advanced = true;
                }
            }
            if batch_timestamp < l1_origin.timestamp {
                warn!(
                    target: "batch_span",
                    "batch timestamp is less than L1 origin timestamp, l2_timestamp: {}, l1_timestamp: {}, origin: {:?}",
                    batch_timestamp,
                    l1_origin.timestamp,
                    l1_origin.id()
                );
                return BatchValidity::Drop;
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
                            return BatchValidity::Drop;
                        } else {
                            info!(
                                target: "batch_span",
                                "continuing with empty batch before late L1 block to preserve L2 time invariant"
                            );
                        }
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
                    return BatchValidity::Drop;
                }
            }

            // Check that the transactions are not empty and do not contain any deposits.
            for (i, tx) in batch.transactions.iter().enumerate() {
                if tx.is_empty() {
                    warn!(
                        target: "batch_span",
                        "transaction data must not be empty, but found empty tx, tx_index: {}",
                        i
                    );
                    return BatchValidity::Drop;
                }
                if tx.as_ref().first() == Some(&(OpTxType::Deposit as u8)) {
                    warn!(
                        target: "batch_span",
                        "sequencers may not embed any deposits into batch data, but found tx that has one, tx_index: {}",
                        i
                    );
                    return BatchValidity::Drop;
                }

                // If isthmus is not active yet and the transaction is a 7702, drop the batch.
                if !cfg.is_isthmus_active(batch.timestamp) &&
                    tx.as_ref().first() == Some(&(OpTxType::Eip7702 as u8))
                {
                    warn!(target: "batch_span", "EIP-7702 transactions are not supported pre-isthmus. tx_index: {}", i);
                    return BatchValidity::Drop;
                }
            }
        }

        // Check overlapped blocks
        let parent_num = parent_block.block_info.number;
        let next_timestamp = l2_safe_head.block_info.timestamp + cfg.block_time;
        if self.starting_timestamp() < next_timestamp {
            for i in 0..(l2_safe_head.block_info.number - parent_num) {
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
                let deposit_count: usize = safe_block
                    .transactions
                    .iter()
                    .map(|tx| if tx.is_deposit() { 1 } else { 0 })
                    .sum();
                if safe_block.transactions.len() - deposit_count != batch_txs.len() {
                    warn!(
                        target: "batch_span",
                        "overlapped block's tx count does not match, safe_block_txs: {}, batch_txs: {}",
                        safe_block.transactions.len(),
                        batch_txs.len()
                    );
                    return BatchValidity::Drop;
                }
                let batch_txs_len = batch_txs.len();
                #[allow(clippy::needless_range_loop)]
                for j in 0..batch_txs_len {
                    let mut buf = Vec::new();
                    safe_block.transactions[j + deposit_count].encode_2718(&mut buf);
                    if buf != batch_txs[j].0 {
                        warn!(target: "batch_span", "overlapped block's transaction does not match");
                        return BatchValidity::Drop;
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
                        return BatchValidity::Drop;
                    }
                };
                if safe_block_ref.l1_origin.number != self.batches[i as usize].epoch_num {
                    warn!(
                        "overlapped block's L1 origin number does not match {}, {}",
                        safe_block_ref.l1_origin.number, self.batches[i as usize].epoch_num
                    );
                    return BatchValidity::Drop;
                }
            }
        }

        BatchValidity::Accept
    }

    /// Checks the validity of the batch's prefix.
    ///
    /// This function is used for post-Holocene hardfork to perform batch validation
    /// as each batch is being loaded in.
    pub async fn check_batch_prefix<BF: BatchValidationProvider>(
        &self,
        cfg: &RollupConfig,
        l1_origins: &[BlockInfo],
        l2_safe_head: L2BlockInfo,
        inclusion_block: &BlockInfo,
        fetcher: &mut BF,
    ) -> (BatchValidity, Option<L2BlockInfo>) {
        if l1_origins.is_empty() {
            warn!(target: "batch_span", "missing L1 block input, cannot proceed with batch checking");
            return (BatchValidity::Undecided, None);
        }
        if self.batches.is_empty() {
            warn!(target: "batch_span", "empty span batch, cannot proceed with batch checking");
            return (BatchValidity::Undecided, None);
        }

        let epoch = l1_origins[0];
        let next_timestamp = l2_safe_head.block_info.timestamp + cfg.block_time;

        let starting_epoch_num = self.starting_epoch_num();
        let mut batch_origin = epoch;
        if starting_epoch_num == batch_origin.number + 1 {
            if l1_origins.len() < 2 {
                info!(
                    target: "batch_span",
                    "eager batch wants to advance current epoch {:?}, but could not without more L1 blocks",
                    epoch.id()
                );
                return (BatchValidity::Undecided, None);
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
            return (BatchValidity::Drop, None);
        }

        if self.starting_timestamp() > next_timestamp {
            warn!(
                target: "batch_span",
                "received out-of-order batch for future processing after next batch ({} > {})",
                self.starting_timestamp(),
                next_timestamp
            );

            // After holocene is activated, gaps are disallowed.
            if cfg.is_holocene_active(inclusion_block.timestamp) {
                return (BatchValidity::Drop, None);
            }
            return (BatchValidity::Future, None);
        }

        // Drop the batch if it has no new blocks after the safe head.
        if self.final_timestamp() < next_timestamp {
            warn!(target: "batch_span", "span batch has no new blocks after safe head");
            return if cfg.is_holocene_active(inclusion_block.timestamp) {
                (BatchValidity::Past, None)
            } else {
                (BatchValidity::Drop, None)
            };
        }

        // Find the parent block of the span batch.
        // If the span batch does not overlap the current safe chain, parent block should be the L2
        // safe head.
        let mut parent_num = l2_safe_head.block_info.number;
        let mut parent_block = l2_safe_head;
        if self.starting_timestamp() < next_timestamp {
            if self.starting_timestamp() > l2_safe_head.block_info.timestamp {
                // Batch timestamp cannot be between safe head and next timestamp.
                warn!(target: "batch_span", "batch has misaligned timestamp, block time is too short");
                return (BatchValidity::Drop, None);
            }
            if !(l2_safe_head.block_info.timestamp - self.starting_timestamp())
                .is_multiple_of(cfg.block_time)
            {
                warn!(target: "batch_span", "batch has misaligned timestamp, not overlapped exactly");
                return (BatchValidity::Drop, None);
            }
            parent_num = l2_safe_head.block_info.number -
                (l2_safe_head.block_info.timestamp - self.starting_timestamp()) / cfg.block_time -
                1;
            parent_block = match fetcher.l2_block_info_by_number(parent_num).await {
                Ok(block) => block,
                Err(e) => {
                    warn!(target: "batch_span", "failed to fetch L2 block number {parent_num}: {e}");
                    // Unable to validate the batch for now. Retry later.
                    return (BatchValidity::Undecided, None);
                }
            };
        }
        if !self.check_parent_hash(parent_block.block_info.hash) {
            warn!(
                target: "batch_span",
                "parent block mismatch, expected: {parent_num}, received: {}. parent hash: {}, parent hash check: {}",
                parent_block.block_info.number, parent_block.block_info.hash, self.parent_check,
            );
            return (BatchValidity::Drop, None);
        }

        // Filter out batches that were included too late.
        if starting_epoch_num + cfg.seq_window_size < inclusion_block.number {
            warn!(target: "batch_span", "batch was included too late, sequence window expired");
            return (BatchValidity::Drop, None);
        }

        // Check the L1 origin of the batch
        if starting_epoch_num > parent_block.l1_origin.number + 1 {
            warn!(
                target: "batch_span",
                "batch is for future epoch too far ahead, while it has the next timestamp, so it must be invalid. starting epoch: {} | next epoch: {}",
                starting_epoch_num,
                parent_block.l1_origin.number + 1
            );
            return (BatchValidity::Drop, None);
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
                    return (BatchValidity::Drop, None);
                }
                origin_checked = true;
                break;
            }
        }
        if !origin_checked {
            info!(target: "batch_span", "need more l1 blocks to check entire origins of span batch");
            return (BatchValidity::Undecided, None);
        }

        if starting_epoch_num < parent_block.l1_origin.number {
            warn!(target: "batch_span", "dropped batch, epoch is too old, minimum: {:?}", parent_block.block_info.id());
            return (BatchValidity::Drop, None);
        }

        (BatchValidity::Accept, Some(parent_block))
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
    use op_alloy_consensus::OpBlock;
    use tracing::Level;
    use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

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
        assert!(batch.append_singular_batch(singular_batch, 0).is_ok());
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
        assert!(batch.append_singular_batch(singular_batch, 1).is_ok());
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
        tracing_subscriber::Registry::default().with(layer).init();

        let cfg = RollupConfig::default();
        let l1_blocks = vec![];
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
        assert!(logs[0].contains("missing L1 block input, cannot proceed with batch checking"));
    }

    #[tokio::test]
    async fn test_check_batches_is_empty() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
            batch.get_singular_batches(&l1_blocks, l2_safe_head),
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
            batch.get_singular_batches(&l1_blocks, l2_safe_head),
            Err(SpanBatchError::MissingL1Origin),
        );
    }

    #[tokio::test]
    async fn test_eager_block_missing_origins() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
        tracing_subscriber::Registry::default().with(layer).init();

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
            BatchValidity::Drop
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
        tracing_subscriber::Registry::default().with(layer).init();

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
        tracing_subscriber::Registry::default().with(layer).init();

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
        let first = SpanBatchElement { epoch_num: 10, timestamp: 10, ..Default::default() };
        let batch = SpanBatch { batches: vec![first], ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("span batch has no new blocks after safe head"));
    }

    #[tokio::test]
    async fn test_check_batch_overlapping_blocks_tx_count_mismatch() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
        let second = SpanBatchElement { epoch_num: 11, timestamp: 60, ..Default::default() };
        let batch = SpanBatch { batches: vec![first, second], ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop
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
        tracing_subscriber::Registry::default().with(layer).init();

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
        let second = SpanBatchElement { epoch_num: 11, timestamp: 60, ..Default::default() };
        let batch = SpanBatch { batches: vec![first, second], ..Default::default() };
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("overlapped block's transaction does not match"));
    }

    #[tokio::test]
    async fn test_check_batch_block_timestamp_lt_l1_origin() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
            BatchValidity::Drop
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
        tracing_subscriber::Registry::default().with(layer).init();

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
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("batch has misaligned timestamp, block time is too short"));
    }

    #[tokio::test]
    async fn test_check_batch_misaligned_without_overlap() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("batch has misaligned timestamp, not overlapped exactly"));
    }

    #[tokio::test]
    async fn test_check_batch_failed_to_fetch_l2_block() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
        tracing_subscriber::Registry::default().with(layer).init();

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
        // parent number = 41 - (10 - 10) / 10 - 1 = 40
        assert_eq!(
            batch.check_batch(&cfg, &l1_blocks, l2_safe_head, &inclusion_block, &mut fetcher).await,
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("parent block mismatch, expected: 40, received: 41"));
    }

    #[tokio::test]
    async fn test_check_sequence_window_expired() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
                timestamp: 10,
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
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("batch was included too late, sequence window expired"));
    }

    #[tokio::test]
    async fn test_starting_epoch_too_far_ahead() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
                timestamp: 10,
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
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        let str = "batch is for future epoch too far ahead, while it has the next timestamp, so it must be invalid. starting epoch: 10 | next epoch: 9";
        assert!(logs[0].contains(str));
    }

    #[tokio::test]
    async fn test_check_batch_epoch_hash_mismatch() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
                timestamp: 10,
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
            BatchValidity::Drop
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
        tracing_subscriber::Registry::default().with(layer).init();

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
                timestamp: 10,
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
        tracing_subscriber::Registry::default().with(layer).init();

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
                timestamp: 10,
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
            BatchValidity::Drop
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
        tracing_subscriber::Registry::default().with(layer).init();

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
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("batch exceeded sequencer time drift without adopting next origin, and next L1 origin would have been valid"));
    }

    #[tokio::test]
    async fn test_continuing_with_empty_batch() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default()
            .with(layer)
            .with(tracing_subscriber::fmt::layer())
            .init();

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
        tracing_subscriber::Registry::default().with(layer).init();

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
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("batch exceeded sequencer time drift, sequencer must adopt new L1 origin to include transactions again, max_time: 10"));
    }

    #[tokio::test]
    async fn test_check_batch_empty_txs() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("transaction data must not be empty, but found empty tx"));
    }

    #[tokio::test]
    async fn test_check_batch_with_deposit_tx() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
            transactions: vec![Bytes::copy_from_slice(&[OpTxType::Deposit as u8])],
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
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("sequencers may not embed any deposits into batch data, but found tx that has one, tx_index: 0"));
    }

    #[tokio::test]
    async fn test_check_batch_with_eip7702_tx() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
            transactions: vec![Bytes::copy_from_slice(&[alloy_consensus::TxType::Eip7702 as u8])],
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
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(
            logs[0].contains("EIP-7702 transactions are not supported pre-isthmus. tx_index: 0")
        );
    }

    #[tokio::test]
    async fn test_check_batch_failed_to_fetch_payload() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
                timestamp: 10,
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
        tracing_subscriber::Registry::default().with(layer).init();

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
                timestamp: 10,
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
            BatchValidity::Drop
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
        tracing_subscriber::Registry::default().with(layer).init();

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
                timestamp: 10,
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
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("overlapped block's L1 origin number does not match"));
    }

    #[tokio::test]
    async fn test_overlapped_blocks_origin_outdated() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
        let block = OpBlock {
            header: Header { number: 41, ..Default::default() },
            body: alloy_consensus::BlockBody {
                transactions: Vec::new(),
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
            BatchValidity::Drop
        );
        let logs = trace_store.get_by_level(Level::WARN);
        assert_eq!(logs.len(), 1);
        assert!(logs[0].contains("batch L1 origin is before safe head L1 origin"));
    }

    #[tokio::test]
    async fn test_check_batch_valid_with_genesis_epoch() {
        let trace_store: TraceStorage = Default::default();
        let layer = CollectingLayer::new(trace_store.clone());
        tracing_subscriber::Registry::default().with(layer).init();

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
                timestamp: 10,
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
