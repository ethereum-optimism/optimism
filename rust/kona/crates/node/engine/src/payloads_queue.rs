//! Buffering for unsafe execution payloads received from the network.

use alloy_primitives::B256;
use kona_protocol::L2BlockInfo;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::collections::{BTreeMap, HashSet, VecDeque};
use thiserror::Error;

/// The maximum memory allocated to buffered unsafe payloads.
const MAX_UNSAFE_PAYLOADS_MEMORY: u64 = 500 * 1024 * 1024;

/// Approximate fixed memory cost of a buffered payload.
const PAYLOAD_MEM_FIXED_COST: u64 = 1000;
/// Approximate memory overhead of each transaction's byte-vector entry.
const PAYLOAD_TX_MEM_OVERHEAD: u64 = 24;

/// An error encountered while buffering an unsafe payload.
#[derive(Debug, Error, PartialEq, Eq)]
pub enum UnsafePayloadQueueError {
    /// The payload is already buffered.
    #[error("unsafe payload {0} is already buffered")]
    Duplicate(B256),
    /// The payload is larger than the entire queue capacity.
    #[error("unsafe payload {hash} requires {size} bytes, exceeding queue capacity {max_size}")]
    PayloadTooLarge {
        /// The payload block hash.
        hash: B256,
        /// The approximate payload memory size.
        size: u64,
        /// The queue's configured maximum size.
        max_size: u64,
    },
    /// The newly added payload was the lowest block and was evicted to enforce the queue limit.
    #[error("unsafe payload {0} was evicted because the queue is full")]
    Evicted(B256),
}

/// An unsafe payload and its approximate memory size.
#[derive(Debug)]
struct QueuedPayload {
    envelope: OpExecutionPayloadEnvelope,
    size: u64,
}

/// Buffers unsafe payloads in ascending block-number order.
///
/// Payloads may arrive with gaps or out of order. Once initial EL sync has completed, only a
/// payload that directly extends the current unsafe head is returned for processing. Before then,
/// the lowest available payload is returned so its forkchoice update can drive EL sync.
#[derive(Debug)]
pub struct UnsafePayloadQueue {
    payloads: BTreeMap<u64, VecDeque<QueuedPayload>>,
    block_hashes: HashSet<B256>,
    len: usize,
    current_size: u64,
    max_size: u64,
}

impl Default for UnsafePayloadQueue {
    fn default() -> Self {
        Self::new(MAX_UNSAFE_PAYLOADS_MEMORY)
    }
}

impl UnsafePayloadQueue {
    /// Creates an empty queue with the given approximate memory limit.
    pub fn new(max_size: u64) -> Self {
        Self {
            payloads: BTreeMap::new(),
            block_hashes: HashSet::new(),
            len: 0,
            current_size: 0,
            max_size,
        }
    }

    /// Returns the number of buffered payloads.
    pub const fn len(&self) -> usize {
        self.len
    }

    /// Returns whether the queue contains no payloads.
    pub const fn is_empty(&self) -> bool {
        self.len == 0
    }

    /// Returns the approximate memory occupied by buffered payloads.
    pub const fn memory_size(&self) -> u64 {
        self.current_size
    }

    /// Adds an unsafe payload to the queue.
    ///
    /// If the queue exceeds its memory limit, payloads at the lowest block numbers are evicted.
    /// Higher payloads are retained because they remain useful for longer while derivation catches
    /// up to the unsafe chain.
    pub fn push(
        &mut self,
        envelope: OpExecutionPayloadEnvelope,
    ) -> Result<(), UnsafePayloadQueueError> {
        let block_hash = envelope.block_hash();
        if self.block_hashes.contains(&block_hash) {
            return Err(UnsafePayloadQueueError::Duplicate(block_hash));
        }

        let size = payload_memory_size(&envelope);
        if size > self.max_size {
            return Err(UnsafePayloadQueueError::PayloadTooLarge {
                hash: block_hash,
                size,
                max_size: self.max_size,
            });
        }

        self.payloads
            .entry(envelope.block_number())
            .or_default()
            .push_back(QueuedPayload { envelope, size });
        self.block_hashes.insert(block_hash);
        self.len += 1;
        self.current_size = self.current_size.saturating_add(size);

        let mut inserted_payload_evicted = false;
        while self.current_size > self.max_size {
            let Some(evicted) = self.pop() else {
                break;
            };
            info!(
                target: "engine",
                hash = %evicted.block_hash(),
                number = evicted.block_number(),
                max_size = self.max_size,
                "Dropping unsafe payload because the payload queue is full"
            );
            inserted_payload_evicted |= evicted.block_hash() == block_hash;
        }

        if inserted_payload_evicted {
            return Err(UnsafePayloadQueueError::Evicted(block_hash));
        }

        Ok(())
    }

    /// Removes and returns the next payload that can be processed.
    ///
    /// Before initial EL sync completes, the lowest available payload is returned even if it has a
    /// gap from the current unsafe head. Its forkchoice update is used to drive EL sync. After EL
    /// sync, stale and conflicting payloads are discarded, gaps are retained, and only the payload
    /// directly extending the unsafe head is returned.
    pub fn pop_applicable(
        &mut self,
        unsafe_head: L2BlockInfo,
        safe_head: L2BlockInfo,
        el_sync_finished: bool,
    ) -> Option<OpExecutionPayloadEnvelope> {
        if !el_sync_finished {
            return self.pop();
        }

        loop {
            let payload = self.peek()?;
            let block_number = payload.block_number();

            if block_number <= safe_head.block_info.number {
                let dropped = self.pop().expect("peeked payload must still be present");
                info!(
                    target: "engine",
                    hash = %dropped.block_hash(),
                    number = block_number,
                    safe = safe_head.block_info.number,
                    "Dropping unsafe payload at or behind the safe head"
                );
                continue;
            }

            if block_number <= unsafe_head.block_info.number {
                let dropped = self.pop().expect("peeked payload must still be present");
                info!(
                    target: "engine",
                    hash = %dropped.block_hash(),
                    number = block_number,
                    unsafe_head = unsafe_head.block_info.number,
                    "Dropping unsafe payload at or behind the unsafe head"
                );
                continue;
            }

            if block_number > unsafe_head.block_info.number.saturating_add(1) {
                return None;
            }

            if payload.parent_hash() != unsafe_head.block_info.hash {
                let dropped = self.pop().expect("peeked payload must still be present");
                info!(
                    target: "engine",
                    hash = %dropped.block_hash(),
                    parent = %dropped.parent_hash(),
                    expected_parent = %unsafe_head.block_info.hash,
                    number = block_number,
                    "Dropping unsafe payload that does not extend the unsafe head"
                );
                continue;
            }

            return self.pop();
        }
    }

    /// Returns the lowest-numbered payload without removing it.
    fn peek(&self) -> Option<&OpExecutionPayloadEnvelope> {
        self.payloads.first_key_value()?.1.front().map(|payload| &payload.envelope)
    }

    /// Removes and returns the lowest-numbered payload.
    fn pop(&mut self) -> Option<OpExecutionPayloadEnvelope> {
        let mut entry = self.payloads.first_entry()?;
        let queued = entry.get_mut().pop_front().expect("payload map entries must not be empty");
        if entry.get().is_empty() {
            entry.remove();
        }

        self.block_hashes.remove(&queued.envelope.block_hash());
        self.len -= 1;
        self.current_size -= queued.size;
        Some(queued.envelope)
    }
}

/// Returns an approximate in-memory size for an execution payload envelope.
fn payload_memory_size(envelope: &OpExecutionPayloadEnvelope) -> u64 {
    envelope.transactions().iter().fold(PAYLOAD_MEM_FIXED_COST, |size, transaction| {
        size.saturating_add(
            u64::try_from(transaction.len())
                .unwrap_or(u64::MAX)
                .saturating_add(PAYLOAD_TX_MEM_OVERHEAD),
        )
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{Bytes, U256};
    use alloy_rpc_types_engine::ExecutionPayloadV1;
    use kona_protocol::BlockInfo;

    fn hash(value: u8) -> B256 {
        B256::repeat_byte(value)
    }

    fn envelope(number: u64, parent_hash: B256, block_hash: B256) -> OpExecutionPayloadEnvelope {
        OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1 {
            parent_hash,
            fee_recipient: Default::default(),
            state_root: B256::ZERO,
            receipts_root: B256::ZERO,
            logs_bloom: Default::default(),
            prev_randao: B256::ZERO,
            block_number: number,
            gas_limit: 0,
            gas_used: 0,
            timestamp: number,
            extra_data: Bytes::new(),
            base_fee_per_gas: U256::ZERO,
            block_hash,
            transactions: vec![],
        })
    }

    fn block_info(number: u64, block_hash: B256) -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo { number, hash: block_hash, ..Default::default() },
            ..Default::default()
        }
    }

    #[test]
    fn buffers_post_sync_payloads_until_the_gap_is_filled() {
        let mut queue = UnsafePayloadQueue::default();
        let hashes = (0..=7).map(hash).collect::<Vec<_>>();

        for number in [5, 3, 4, 7] {
            queue
                .push(envelope(number, hashes[(number - 1) as usize], hashes[number as usize]))
                .unwrap();
        }

        assert!(
            queue
                .pop_applicable(block_info(1, hashes[1]), block_info(0, hashes[0]), true)
                .is_none()
        );

        queue.push(envelope(2, hashes[1], hashes[2])).unwrap();
        for number in 2..=5 {
            let payload = queue
                .pop_applicable(
                    block_info(number - 1, hashes[(number - 1) as usize]),
                    block_info(0, hashes[0]),
                    true,
                )
                .unwrap();
            assert_eq!(payload.block_number(), number);
        }

        assert!(
            queue
                .pop_applicable(block_info(5, hashes[5]), block_info(0, hashes[0]), true)
                .is_none()
        );

        queue.push(envelope(6, hashes[5], hashes[6])).unwrap();
        for number in 6..=7 {
            let payload = queue
                .pop_applicable(
                    block_info(number - 1, hashes[(number - 1) as usize]),
                    block_info(0, hashes[0]),
                    true,
                )
                .unwrap();
            assert_eq!(payload.block_number(), number);
        }
        assert!(queue.is_empty());
    }

    #[test]
    fn returns_gapped_payload_while_el_sync_is_active() {
        let mut queue = UnsafePayloadQueue::default();
        queue.push(envelope(5, hash(4), hash(5))).unwrap();

        let payload =
            queue.pop_applicable(block_info(1, hash(1)), block_info(0, hash(0)), false).unwrap();
        assert_eq!(payload.block_number(), 5);
    }

    #[test]
    fn drops_conflicting_payload_before_returning_matching_payload() {
        let mut queue = UnsafePayloadQueue::default();
        queue.push(envelope(2, hash(9), hash(8))).unwrap();
        queue.push(envelope(2, hash(1), hash(2))).unwrap();

        let payload =
            queue.pop_applicable(block_info(1, hash(1)), block_info(0, hash(0)), true).unwrap();
        assert_eq!(payload.block_hash(), hash(2));
        assert!(queue.is_empty());
    }

    #[test]
    fn rejects_duplicates_and_allows_readding_after_pop() {
        let mut queue = UnsafePayloadQueue::default();
        let payload = envelope(1, hash(0), hash(1));
        queue.push(payload.clone()).unwrap();

        assert_eq!(queue.push(payload.clone()), Err(UnsafePayloadQueueError::Duplicate(hash(1))));

        assert!(
            queue.pop_applicable(L2BlockInfo::default(), L2BlockInfo::default(), false).is_some()
        );
        queue.push(payload).unwrap();
    }

    #[test]
    fn evicts_lowest_payloads_when_full() {
        let mut queue = UnsafePayloadQueue::new(PAYLOAD_MEM_FIXED_COST * 2);
        queue.push(envelope(2, hash(1), hash(2))).unwrap();
        queue.push(envelope(3, hash(2), hash(3))).unwrap();

        assert_eq!(
            queue.push(envelope(1, hash(0), hash(1))),
            Err(UnsafePayloadQueueError::Evicted(hash(1)))
        );
        assert_eq!(queue.len(), 2);
        assert_eq!(queue.memory_size(), PAYLOAD_MEM_FIXED_COST * 2);

        let first =
            queue.pop_applicable(L2BlockInfo::default(), L2BlockInfo::default(), false).unwrap();
        assert_eq!(first.block_number(), 2);
    }
}
