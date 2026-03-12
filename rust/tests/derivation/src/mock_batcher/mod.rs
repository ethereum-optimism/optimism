//! Mock batcher that encodes L2 blocks into L1 batch transactions.
//!
//! Handles the encoding pipeline: L2 block -> SingleBatch -> Batch encode -> zlib compress
//! -> Frame split -> prepend DERIVATION_VERSION_0 -> TxEnvelope calldata.

mod channel_encoder;
mod submitter;

pub use channel_encoder::*;
pub use submitter::*;

use crate::mock_sequencer::MockL2Block;
use alloy_consensus::TxEnvelope;
use alloy_primitives::Address;
use kona_protocol::{ChannelId, SingleBatch};

/// A mock batcher that encodes L2 blocks and submits them as L1 transactions.
#[derive(Debug)]
pub struct MockBatcher {
    /// Buffered L2 blocks waiting to be encoded into a channel.
    buffered_blocks: Vec<MockL2Block>,
    /// The batch inbox address on L1.
    batch_inbox: Address,
    /// The batcher address (tx sender).
    batcher_addr: Address,
    /// Maximum frame size for splitting.
    max_frame_size: usize,
    /// Channel counter for generating unique channel IDs.
    channel_counter: u64,
}

impl MockBatcher {
    /// Creates a new [`MockBatcher`].
    pub fn new(batch_inbox: Address, batcher_addr: Address) -> Self {
        Self {
            buffered_blocks: Vec::new(),
            batch_inbox,
            batcher_addr,
            max_frame_size: 120_000,
            channel_counter: 0,
        }
    }

    /// Buffers an L2 block for later batch submission.
    pub fn buffer_block(&mut self, block: &MockL2Block) {
        self.buffered_blocks.push(block.clone());
    }

    /// Encodes all buffered blocks into a channel, splits into frames,
    /// and returns L1 calldata transactions.
    pub fn submit_all(&mut self) -> Vec<TxEnvelope> {
        if self.buffered_blocks.is_empty() {
            return Vec::new();
        }

        let blocks = std::mem::take(&mut self.buffered_blocks);

        // 1. Encode blocks into SingleBatches
        let batches: Vec<SingleBatch> = blocks
            .iter()
            .map(|block| SingleBatch {
                parent_hash: block.parent_hash,
                epoch_num: block.l1_origin.number,
                epoch_hash: block.l1_origin.hash,
                timestamp: block.timestamp,
                transactions: block.transactions.clone(),
            })
            .collect();

        // 2. Encode and compress into channel data
        let channel_id = self.next_channel_id();
        let channel_data = encode_batches(&batches);

        // 3. Split into frames
        let frames = split_into_frames(channel_id, &channel_data, self.max_frame_size);

        // 4. Create L1 transactions
        frames
            .into_iter()
            .map(|frame| create_calldata_tx(frame, self.batch_inbox, self.batcher_addr))
            .collect()
    }

    /// Generates a unique channel ID.
    fn next_channel_id(&mut self) -> ChannelId {
        let id = self.channel_counter;
        self.channel_counter += 1;
        let mut channel_id = [0u8; 16];
        channel_id[..8].copy_from_slice(&id.to_be_bytes());
        channel_id
    }
}
