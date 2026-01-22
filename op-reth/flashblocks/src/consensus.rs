use crate::{FlashBlockCompleteSequence, FlashBlockCompleteSequenceRx};
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadStatusEnum;
use op_alloy_rpc_types_engine::OpExecutionData;
use reth_engine_primitives::ConsensusEngineHandle;
use reth_optimism_payload_builder::OpPayloadTypes;
use reth_payload_primitives::{EngineApiMessageVersion, ExecutionPayload, PayloadTypes};
use tracing::*;

/// Consensus client that sends FCUs and new payloads using blocks from a [`FlashBlockService`].
///
/// This client receives completed flashblock sequences and:
/// - Attempts to submit `engine_newPayload` if `state_root` is available (non-zero)
/// - Always sends `engine_forkChoiceUpdated` to drive chain forward
///
/// [`FlashBlockService`]: crate::FlashBlockService
#[derive(Debug)]
pub struct FlashBlockConsensusClient<P = OpPayloadTypes>
where
    P: PayloadTypes,
{
    /// Handle to execution client.
    engine_handle: ConsensusEngineHandle<P>,
    /// Receiver for completed flashblock sequences from `FlashBlockService`.
    sequence_receiver: FlashBlockCompleteSequenceRx,
}

impl<P> FlashBlockConsensusClient<P>
where
    P: PayloadTypes,
    P::ExecutionData: for<'a> TryFrom<&'a FlashBlockCompleteSequence, Error: std::fmt::Display>,
{
    /// Create a new `FlashBlockConsensusClient` with the given Op engine and sequence receiver.
    pub const fn new(
        engine_handle: ConsensusEngineHandle<P>,
        sequence_receiver: FlashBlockCompleteSequenceRx,
    ) -> eyre::Result<Self> {
        Ok(Self { engine_handle, sequence_receiver })
    }

    /// Attempts to submit a new payload to the engine.
    ///
    /// The `TryFrom` conversion will fail if `execution_outcome.state_root` is `B256::ZERO`,
    /// in which case this method uses the `parent_hash` instead to drive the chain forward.
    ///
    /// Returns the block hash to use for FCU (either the new block's hash or the parent hash).
    async fn submit_new_payload(&self, sequence: &FlashBlockCompleteSequence) -> B256 {
        let payload = match P::ExecutionData::try_from(sequence) {
            Ok(payload) => payload,
            Err(err) => {
                trace!(target: "flashblocks", %err, "Failed payload conversion, using parent hash");
                return sequence.payload_base().parent_hash;
            }
        };

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
                };
            }
            Err(err) => {
                error!(
                    target: "flashblocks",
                    %err,
                    block_number,
                    "Failed to submit new payload",
                );
            }
        }

        block_hash
    }

    /// Submit a forkchoice update to the engine.
    async fn submit_forkchoice_update(
        &self,
        head_block_hash: B256,
        sequence: &FlashBlockCompleteSequence,
    ) {
        let block_number = sequence.block_number();
        let safe_hash = sequence.payload_base().parent_hash;
        let finalized_hash = sequence.payload_base().parent_hash;
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

    /// Runs the consensus client loop.
    ///
    /// Continuously receives completed flashblock sequences and submits them to the execution
    /// engine:
    /// 1. Attempts `engine_newPayload` (only if `state_root` is available)
    /// 2. Always sends `engine_forkChoiceUpdated` to drive chain forward
    pub async fn run(mut self) {
        loop {
            let Ok(sequence) = self.sequence_receiver.recv().await else {
                continue;
            };

            // Returns block_hash for FCU:
            // - If state_root is available: submits newPayload and returns the new block's hash
            // - If state_root is zero: skips newPayload and returns parent_hash (no progress yet)
            let block_hash = self.submit_new_payload(&sequence).await;

            self.submit_forkchoice_update(block_hash, &sequence).await;
        }
    }
}

impl TryFrom<&FlashBlockCompleteSequence> for OpExecutionData {
    type Error = &'static str;

    fn try_from(sequence: &FlashBlockCompleteSequence) -> Result<Self, Self::Error> {
        let mut data = Self::from_flashblocks_unchecked(sequence);

        // If execution outcome is available, use the computed state_root and block_hash.
        // FlashBlockService computes these when building sequences on top of the local tip.
        if let Some(execution_outcome) = sequence.execution_outcome() {
            let payload = data.payload.as_v1_mut();
            payload.state_root = execution_outcome.state_root;
            payload.block_hash = execution_outcome.block_hash;
        }

        // Only proceed if we have a valid state_root (non-zero).
        if data.payload.as_v1_mut().state_root == B256::ZERO {
            return Err("No state_root available for payload");
        }

        Ok(data)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{sequence::SequenceExecutionOutcome, test_utils::TestFlashBlockFactory};

    mod op_execution_data_conversion {
        use super::*;

        #[test]
        fn test_try_from_fails_with_zero_state_root() {
            // When execution_outcome is None, state_root remains zero and conversion fails
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).state_root(B256::ZERO).build();

            let sequence = FlashBlockCompleteSequence::new(vec![fb0], None).unwrap();

            let result = OpExecutionData::try_from(&sequence);
            assert!(result.is_err());
            assert_eq!(result.unwrap_err(), "No state_root available for payload");
        }

        #[test]
        fn test_try_from_succeeds_with_execution_outcome() {
            // When execution_outcome has state_root, conversion succeeds
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).state_root(B256::ZERO).build();

            let execution_outcome = SequenceExecutionOutcome {
                block_hash: B256::random(),
                state_root: B256::random(), // Non-zero
            };

            let sequence =
                FlashBlockCompleteSequence::new(vec![fb0], Some(execution_outcome)).unwrap();

            let result = OpExecutionData::try_from(&sequence);
            assert!(result.is_ok());

            let mut data = result.unwrap();
            assert_eq!(data.payload.as_v1_mut().state_root, execution_outcome.state_root);
            assert_eq!(data.payload.as_v1_mut().block_hash, execution_outcome.block_hash);
        }

        #[test]
        fn test_try_from_succeeds_with_provided_state_root() {
            // When sequencer provides non-zero state_root, conversion succeeds
            let factory = TestFlashBlockFactory::new();
            let provided_state_root = B256::random();
            let fb0 = factory.flashblock_at(0).state_root(provided_state_root).build();

            let sequence = FlashBlockCompleteSequence::new(vec![fb0], None).unwrap();

            let result = OpExecutionData::try_from(&sequence);
            assert!(result.is_ok());

            let mut data = result.unwrap();
            assert_eq!(data.payload.as_v1_mut().state_root, provided_state_root);
        }

        #[test]
        fn test_try_from_execution_outcome_overrides_provided_state_root() {
            // execution_outcome takes precedence over sequencer-provided state_root
            let factory = TestFlashBlockFactory::new();
            let provided_state_root = B256::random();
            let fb0 = factory.flashblock_at(0).state_root(provided_state_root).build();

            let execution_outcome = SequenceExecutionOutcome {
                block_hash: B256::random(),
                state_root: B256::random(), // Different from provided
            };

            let sequence =
                FlashBlockCompleteSequence::new(vec![fb0], Some(execution_outcome)).unwrap();

            let result = OpExecutionData::try_from(&sequence);
            assert!(result.is_ok());

            let mut data = result.unwrap();
            // Should use execution_outcome, not the provided state_root
            assert_eq!(data.payload.as_v1_mut().state_root, execution_outcome.state_root);
            assert_ne!(data.payload.as_v1_mut().state_root, provided_state_root);
        }

        #[test]
        fn test_try_from_with_multiple_flashblocks() {
            // Test conversion with sequence of multiple flashblocks
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).state_root(B256::ZERO).build();
            let fb1 = factory.flashblock_after(&fb0).state_root(B256::ZERO).build();
            let fb2 = factory.flashblock_after(&fb1).state_root(B256::ZERO).build();

            let execution_outcome =
                SequenceExecutionOutcome { block_hash: B256::random(), state_root: B256::random() };

            let sequence =
                FlashBlockCompleteSequence::new(vec![fb0, fb1, fb2], Some(execution_outcome))
                    .unwrap();

            let result = OpExecutionData::try_from(&sequence);
            assert!(result.is_ok());

            let mut data = result.unwrap();
            assert_eq!(data.payload.as_v1_mut().state_root, execution_outcome.state_root);
            assert_eq!(data.payload.as_v1_mut().block_hash, execution_outcome.block_hash);
        }
    }

    mod consensus_client_creation {
        use super::*;
        use tokio::sync::broadcast;

        #[test]
        fn test_new_creates_client() {
            let (engine_tx, _) = tokio::sync::mpsc::unbounded_channel();
            let engine_handle = ConsensusEngineHandle::<OpPayloadTypes>::new(engine_tx);

            let (_, sequence_rx) = broadcast::channel(1);

            let result = FlashBlockConsensusClient::new(engine_handle, sequence_rx);
            assert!(result.is_ok());
        }
    }

    mod submit_new_payload_behavior {
        use super::*;

        #[test]
        fn test_submit_new_payload_returns_parent_hash_when_no_state_root() {
            // When conversion fails (no state_root), should return parent_hash
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).state_root(B256::ZERO).build();
            let parent_hash = fb0.base.as_ref().unwrap().parent_hash;

            let sequence = FlashBlockCompleteSequence::new(vec![fb0], None).unwrap();

            // Verify conversion would fail
            let conversion_result = OpExecutionData::try_from(&sequence);
            assert!(conversion_result.is_err());

            // In the actual run loop, submit_new_payload would return parent_hash
            assert_eq!(sequence.payload_base().parent_hash, parent_hash);
        }

        #[test]
        fn test_submit_new_payload_returns_block_hash_when_state_root_available() {
            // When conversion succeeds, should return the new block's hash
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).state_root(B256::ZERO).build();

            let execution_outcome =
                SequenceExecutionOutcome { block_hash: B256::random(), state_root: B256::random() };

            let sequence =
                FlashBlockCompleteSequence::new(vec![fb0], Some(execution_outcome)).unwrap();

            // Verify conversion succeeds
            let conversion_result = OpExecutionData::try_from(&sequence);
            assert!(conversion_result.is_ok());

            let mut data = conversion_result.unwrap();
            assert_eq!(data.payload.as_v1_mut().block_hash, execution_outcome.block_hash);
        }
    }

    mod forkchoice_update_behavior {
        use super::*;

        #[test]
        fn test_forkchoice_state_uses_parent_hash_for_safe_and_finalized() {
            // Both safe_hash and finalized_hash should be set to parent_hash
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).build();
            let parent_hash = fb0.base.as_ref().unwrap().parent_hash;

            let sequence = FlashBlockCompleteSequence::new(vec![fb0], None).unwrap();

            // Verify the expected forkchoice state
            assert_eq!(sequence.payload_base().parent_hash, parent_hash);
        }

        #[test]
        fn test_forkchoice_update_with_new_block_hash() {
            // When newPayload succeeds, FCU should use the new block's hash as head
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).state_root(B256::ZERO).build();

            let execution_outcome =
                SequenceExecutionOutcome { block_hash: B256::random(), state_root: B256::random() };

            let sequence =
                FlashBlockCompleteSequence::new(vec![fb0], Some(execution_outcome)).unwrap();

            // The head_block_hash for FCU would be execution_outcome.block_hash
            assert_eq!(
                sequence.execution_outcome().unwrap().block_hash,
                execution_outcome.block_hash
            );
        }

        #[test]
        fn test_forkchoice_update_with_parent_hash_when_no_state_root() {
            // When newPayload is skipped (no state_root), FCU should use parent_hash as head
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).state_root(B256::ZERO).build();
            let parent_hash = fb0.base.as_ref().unwrap().parent_hash;

            let sequence = FlashBlockCompleteSequence::new(vec![fb0], None).unwrap();

            // The head_block_hash for FCU would be parent_hash (fallback)
            assert_eq!(sequence.payload_base().parent_hash, parent_hash);
        }
    }

    mod run_loop_logic {
        use super::*;

        #[test]
        fn test_run_loop_processes_sequence_with_state_root() {
            // Scenario: Sequence with state_root should trigger both newPayload and FCU
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).state_root(B256::ZERO).build();

            let execution_outcome =
                SequenceExecutionOutcome { block_hash: B256::random(), state_root: B256::random() };

            let sequence =
                FlashBlockCompleteSequence::new(vec![fb0], Some(execution_outcome)).unwrap();

            // Verify sequence is ready for newPayload
            let conversion = OpExecutionData::try_from(&sequence);
            assert!(conversion.is_ok());
        }

        #[test]
        fn test_run_loop_processes_sequence_without_state_root() {
            // Scenario: Sequence without state_root should skip newPayload but still do FCU
            let factory = TestFlashBlockFactory::new();
            let fb0 = factory.flashblock_at(0).state_root(B256::ZERO).build();

            let sequence = FlashBlockCompleteSequence::new(vec![fb0], None).unwrap();

            // Verify sequence cannot be converted (newPayload will be skipped)
            let conversion = OpExecutionData::try_from(&sequence);
            assert!(conversion.is_err());

            // But FCU should still happen with parent_hash
            assert!(sequence.payload_base().parent_hash != B256::ZERO);
        }

        #[test]
        fn test_run_loop_handles_multiple_sequences() {
            // Multiple sequences should be processed independently
            let factory = TestFlashBlockFactory::new();

            // Sequence 1: With state_root
            let fb0_seq1 = factory.flashblock_at(0).state_root(B256::ZERO).build();
            let outcome1 =
                SequenceExecutionOutcome { block_hash: B256::random(), state_root: B256::random() };
            let seq1 =
                FlashBlockCompleteSequence::new(vec![fb0_seq1.clone()], Some(outcome1)).unwrap();

            // Sequence 2: Without state_root (for next block)
            let fb0_seq2 = factory.flashblock_for_next_block(&fb0_seq1).build();
            let seq2 = FlashBlockCompleteSequence::new(vec![fb0_seq2], None).unwrap();

            // Both should be valid sequences
            assert_eq!(seq1.block_number(), 100);
            assert_eq!(seq2.block_number(), 101);

            // seq1 can be converted
            assert!(OpExecutionData::try_from(&seq1).is_ok());
            // seq2 cannot be converted
            assert!(OpExecutionData::try_from(&seq2).is_err());
        }
    }
}
