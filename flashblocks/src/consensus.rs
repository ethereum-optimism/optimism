use crate::FlashBlockCompleteSequenceRx;
use alloy_primitives::B256;
use reth_node_api::{ConsensusEngineHandle, EngineApiMessageVersion};
use reth_optimism_payload_builder::OpPayloadTypes;
use ringbuffer::{AllocRingBuffer, RingBuffer};
use tracing::warn;

/// Consensus client that sends FCUs and new payloads using blocks from a [`FlashBlockService`]
///
/// [`FlashBlockService`]: crate::FlashBlockService
#[derive(Debug)]
pub struct FlashBlockConsensusClient {
    /// Handle to execution client.
    engine_handle: ConsensusEngineHandle<OpPayloadTypes>,
    sequence_receiver: FlashBlockCompleteSequenceRx,
}

impl FlashBlockConsensusClient {
    /// Create a new `FlashBlockConsensusClient` with the given Op engine and sequence receiver.
    pub const fn new(
        engine_handle: ConsensusEngineHandle<OpPayloadTypes>,
        sequence_receiver: FlashBlockCompleteSequenceRx,
    ) -> eyre::Result<Self> {
        Ok(Self { engine_handle, sequence_receiver })
    }

    /// Get previous block hash using previous block hash buffer. If it isn't available (buffer
    /// started more recently than `offset`), return default zero hash
    fn get_previous_block_hash(
        &self,
        previous_block_hashes: &AllocRingBuffer<B256>,
        offset: usize,
    ) -> B256 {
        *previous_block_hashes
            .len()
            .checked_sub(offset)
            .and_then(|index| previous_block_hashes.get(index))
            .unwrap_or_default()
    }

    /// Spawn the client to start sending FCUs and new payloads by periodically fetching recent
    /// blocks.
    pub async fn run(mut self) {
        let mut previous_block_hashes = AllocRingBuffer::new(64);

        loop {
            match self.sequence_receiver.recv().await {
                Ok(sequence) => {
                    let block_hash = sequence.payload_base().parent_hash;
                    previous_block_hashes.push(block_hash);

                    if sequence.state_root().is_none() {
                        warn!("Missing state root for the complete sequence")
                    }

                    // Load previous block hashes. We're using (head - 32) and (head - 64) as the
                    // safe and finalized block hashes.
                    let safe_block_hash = self.get_previous_block_hash(&previous_block_hashes, 32);
                    let finalized_block_hash =
                        self.get_previous_block_hash(&previous_block_hashes, 64);

                    let state = alloy_rpc_types_engine::ForkchoiceState {
                        head_block_hash: block_hash,
                        safe_block_hash,
                        finalized_block_hash,
                    };

                    // Send FCU
                    let _ = self
                        .engine_handle
                        .fork_choice_updated(state, None, EngineApiMessageVersion::V3)
                        .await;
                }
                Err(err) => {
                    warn!(
                        target: "consensus::flashblock-client",
                        %err,
                        "error while fetching flashblock completed sequence"
                    );
                    break;
                }
            }
        }
    }
}
