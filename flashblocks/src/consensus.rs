use crate::{FlashBlockCompleteSequence, FlashBlockCompleteSequenceRx};
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadStatusEnum;
use op_alloy_rpc_types_engine::OpExecutionData;
use reth_engine_primitives::ConsensusEngineHandle;
use reth_optimism_payload_builder::OpPayloadTypes;
use reth_payload_primitives::{EngineApiMessageVersion, ExecutionPayload, PayloadTypes};
use ringbuffer::{AllocRingBuffer, RingBuffer};
use tracing::*;

/// Cache entry for block information: (block hash, block number, timestamp).
type BlockCacheEntry = (B256, u64, u64);

/// Consensus client that sends FCUs and new payloads using blocks from a [`FlashBlockService`]
///
/// [`FlashBlockService`]: crate::FlashBlockService
#[derive(Debug)]
pub struct FlashBlockConsensusClient<P = OpPayloadTypes>
where
    P: PayloadTypes,
{
    /// Handle to execution client.
    engine_handle: ConsensusEngineHandle<P>,
    sequence_receiver: FlashBlockCompleteSequenceRx,
    /// Caches previous block info for lookup: (block hash, block number, timestamp).
    block_hash_buffer: AllocRingBuffer<BlockCacheEntry>,
}

impl<P> FlashBlockConsensusClient<P>
where
    P: PayloadTypes,
    P::ExecutionData: for<'a> TryFrom<&'a FlashBlockCompleteSequence, Error: std::fmt::Display>,
{
    /// Create a new `FlashBlockConsensusClient` with the given Op engine and sequence receiver.
    pub fn new(
        engine_handle: ConsensusEngineHandle<P>,
        sequence_receiver: FlashBlockCompleteSequenceRx,
    ) -> eyre::Result<Self> {
        // Buffer size of 768 blocks (64 * 12) supports 1s block time chains like Unichain.
        // Oversized for 2s block time chains like Base, but acceptable given minimal memory usage.
        let block_hash_buffer = AllocRingBuffer::new(768);
        Ok(Self { engine_handle, sequence_receiver, block_hash_buffer })
    }

    /// Return the safe and finalized block hash for FCU calls.
    ///
    /// Safe blocks are considered 32 L1 blocks (approximately 384s at 12s/block) behind the head,
    /// and finalized blocks are 64 L1 blocks (approximately 768s) behind the head. This
    /// approximation, while not precisely matching the OP stack's derivation, provides
    /// sufficient proximity and enables op-reth to sync the chain independently of an op-node.
    /// The offset is dynamically adjusted based on the actual block time detected from the
    /// buffer.
    fn get_safe_and_finalized_block_hash(&self) -> (B256, B256) {
        let cached_blocks_count = self.block_hash_buffer.len();

        // Not enough blocks to determine safe/finalized yet
        if cached_blocks_count < 2 {
            return (B256::ZERO, B256::ZERO);
        }

        // Calculate average block time using block numbers to handle missing blocks correctly.
        // By dividing timestamp difference by block number difference, we get accurate block
        // time even when blocks are missing from the buffer.
        let (_, latest_block_number, latest_timestamp) =
            self.block_hash_buffer.get(cached_blocks_count - 1).unwrap();
        let (_, previous_block_number, previous_timestamp) =
            self.block_hash_buffer.get(cached_blocks_count - 2).unwrap();
        let timestamp_delta = latest_timestamp.saturating_sub(*previous_timestamp);
        let block_number_delta = latest_block_number.saturating_sub(*previous_block_number).max(1);
        let block_time_secs = timestamp_delta / block_number_delta;

        // L1 reference: 32 blocks * 12s = 384s for safe, 64 blocks * 12s = 768s for finalized
        const SAFE_TIME_SECS: u64 = 384;
        const FINALIZED_TIME_SECS: u64 = 768;

        // Calculate how many L2 blocks correspond to these L1 time periods
        let safe_block_offset =
            (SAFE_TIME_SECS / block_time_secs).min(cached_blocks_count as u64) as usize;
        let finalized_block_offset =
            (FINALIZED_TIME_SECS / block_time_secs).min(cached_blocks_count as u64) as usize;

        // Get safe hash: offset from end of buffer
        let safe_hash = self
            .block_hash_buffer
            .get(cached_blocks_count.saturating_sub(safe_block_offset))
            .map(|&(hash, _, _)| hash)
            .unwrap();

        // Get finalized hash: offset from end of buffer
        let finalized_hash = self
            .block_hash_buffer
            .get(cached_blocks_count.saturating_sub(finalized_block_offset))
            .map(|&(hash, _, _)| hash)
            .unwrap();

        (safe_hash, finalized_hash)
    }

    /// Receive the next flashblock sequence and cache its block information.
    ///
    /// Returns `None` if receiving fails (error is already logged).
    async fn receive_and_cache_sequence(&mut self) -> Option<FlashBlockCompleteSequence> {
        match self.sequence_receiver.recv().await {
            Ok(sequence) => {
                self.block_hash_buffer.push((
                    sequence.payload_base().parent_hash,
                    sequence.block_number(),
                    sequence.payload_base().timestamp,
                ));
                Some(sequence)
            }
            Err(err) => {
                error!(
                    target: "flashblocks",
                    %err,
                    "error while fetching flashblock completed sequence",
                );
                None
            }
        }
    }

    /// Convert a flashblock sequence to an execution payload.
    ///
    /// Returns `None` if conversion fails (error is already logged).
    fn convert_sequence_to_payload(
        &self,
        sequence: &FlashBlockCompleteSequence,
    ) -> Option<P::ExecutionData> {
        match P::ExecutionData::try_from(sequence) {
            Ok(payload) => Some(payload),
            Err(err) => {
                error!(
                    target: "flashblocks",
                    %err,
                    "error while converting to payload from completed sequence",
                );
                None
            }
        }
    }

    /// Submit a new payload to the engine.
    ///
    /// Returns `Ok(block_hash)` if the payload was accepted, `Err(())` otherwise (errors are
    /// logged).
    async fn submit_new_payload(
        &self,
        payload: P::ExecutionData,
        sequence: &FlashBlockCompleteSequence,
    ) -> Result<B256, ()> {
        let block_number = payload.block_number();
        let block_hash = payload.block_hash();

        match self.engine_handle.new_payload(payload).await {
            Ok(result) => {
                debug!(
                    target: "flashblocks",
                    flashblock_count = sequence.count(),
                    block_number,
                    %block_hash,
                    ?result,
                    "Submitted engine_newPayload",
                );

                if let PayloadStatusEnum::Invalid { validation_error } = result.status {
                    debug!(
                        target: "flashblocks",
                        block_number,
                        %block_hash,
                        %validation_error,
                        "Payload validation error",
                    );
                    return Err(());
                }

                Ok(block_hash)
            }
            Err(err) => {
                error!(
                    target: "flashblocks",
                    %err,
                    block_number,
                    "Failed to submit new payload",
                );
                Err(())
            }
        }
    }

    /// Submit a forkchoice update to the engine.
    async fn submit_forkchoice_update(
        &self,
        head_block_hash: B256,
        sequence: &FlashBlockCompleteSequence,
    ) {
        let block_number = sequence.block_number();
        let (safe_hash, finalized_hash) = self.get_safe_and_finalized_block_hash();
        let fcu_state = alloy_rpc_types_engine::ForkchoiceState {
            head_block_hash,
            safe_block_hash: safe_hash,
            finalized_block_hash: finalized_hash,
        };

        match self
            .engine_handle
            .fork_choice_updated(fcu_state, None, EngineApiMessageVersion::V5)
            .await
        {
            Ok(result) => {
                debug!(
                    target: "flashblocks",
                    flashblock_count = sequence.count(),
                    block_number,
                    %head_block_hash,
                    %safe_hash,
                    %finalized_hash,
                    ?result,
                    "Submitted engine_forkChoiceUpdated",
                )
            }
            Err(err) => {
                error!(
                    target: "flashblocks",
                    %err,
                    block_number,
                    %head_block_hash,
                    %safe_hash,
                    %finalized_hash,
                    "Failed to submit fork choice update",
                );
            }
        }
    }

    /// Spawn the client to start sending FCUs and new payloads by periodically fetching recent
    /// blocks.
    pub async fn run(mut self) {
        loop {
            let Some(sequence) = self.receive_and_cache_sequence().await else {
                continue;
            };

            let Some(payload) = self.convert_sequence_to_payload(&sequence) else {
                continue;
            };

            let Ok(block_hash) = self.submit_new_payload(payload, &sequence).await else {
                continue;
            };

            self.submit_forkchoice_update(block_hash, &sequence).await;
        }
    }
}

impl From<&FlashBlockCompleteSequence> for OpExecutionData {
    fn from(sequence: &FlashBlockCompleteSequence) -> Self {
        let mut data = Self::from_flashblocks_unchecked(sequence);
        // Replace payload's state_root with the calculated one. For flashblocks, there was an
        // option to disable state root calculation for blocks, and in that case, the payload's
        // state_root will be zero, and we'll need to locally calculate state_root before
        // proceeding to call engine_newPayload.
        if let Some(execution_outcome) = sequence.execution_outcome() {
            let payload = data.payload.as_v1_mut();
            payload.state_root = execution_outcome.state_root;
            payload.block_hash = execution_outcome.block_hash;
        }
        data
    }
}
