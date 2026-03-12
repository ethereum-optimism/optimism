//! Mock L2 sequencer that produces deterministic L2 block data for the batcher.

use alloy_primitives::{B256, Bytes, Keccak256};
use kona_protocol::BlockInfo;

/// A mock L2 block produced by the sequencer, for the batcher to consume.
#[derive(Debug, Clone)]
pub struct MockL2Block {
    /// L2 block number.
    pub number: u64,
    /// L2 block timestamp.
    pub timestamp: u64,
    /// Parent block hash.
    pub parent_hash: B256,
    /// Block hash (deterministically derived from number).
    pub hash: B256,
    /// L1 origin block that this L2 block references.
    pub l1_origin: BlockInfo,
    /// Transactions in this block (RLP-encoded).
    pub transactions: Vec<Bytes>,
}

impl MockL2Block {
    /// Computes a deterministic hash from the block number and parent hash.
    pub fn compute_hash(number: u64, parent_hash: B256) -> B256 {
        let mut hasher = Keccak256::new();
        hasher.update(b"mock_l2_block");
        hasher.update(number.to_be_bytes());
        hasher.update(parent_hash);
        hasher.finalize()
    }
}

/// A deterministic mock sequencer that produces L2 blocks.
#[derive(Debug)]
pub struct MockSequencer {
    /// The current block number (next to produce).
    next_number: u64,
    /// The current L2 timestamp.
    current_timestamp: u64,
    /// L2 block time.
    block_time: u64,
    /// The parent hash of the next block to produce.
    parent_hash: B256,
    /// All produced blocks.
    pub blocks: Vec<MockL2Block>,
}

impl MockSequencer {
    /// Creates a new [`MockSequencer`].
    pub fn new(genesis_hash: B256, genesis_timestamp: u64, block_time: u64) -> Self {
        Self {
            next_number: 1,
            current_timestamp: genesis_timestamp,
            block_time,
            parent_hash: genesis_hash,
            blocks: Vec::new(),
        }
    }

    /// Produces the next L2 block with the given transactions and L1 origin.
    pub fn produce_block(
        &mut self,
        transactions: Vec<Bytes>,
        l1_origin: BlockInfo,
    ) -> MockL2Block {
        let timestamp = self.current_timestamp + self.block_time;
        let number = self.next_number;
        let parent_hash = self.parent_hash;
        let hash = MockL2Block::compute_hash(number, parent_hash);

        let block = MockL2Block {
            number,
            timestamp,
            parent_hash,
            hash,
            l1_origin,
            transactions,
        };

        self.parent_hash = hash;
        self.current_timestamp = timestamp;
        self.next_number += 1;
        self.blocks.push(block.clone());
        block
    }

    /// Returns the latest produced block, if any.
    pub fn latest_block(&self) -> Option<&MockL2Block> {
        self.blocks.last()
    }
}
