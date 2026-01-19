use super::EventHandler;
use crate::{
    ChainProcessorError, ProcessorState, chain_processor::Metrics, syncnode::ManagedNodeCommand,
};
use alloy_primitives::ChainId;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_interop::DerivedRefPair;
use kona_protocol::BlockInfo;
use tokio::sync::mpsc;
use tracing::{trace, warn};

/// Handler for cross unsafe blocks.
/// This handler processes cross unsafe blocks by updating the managed node.
#[derive(Debug, Constructor)]
pub struct CrossUnsafeHandler {
    chain_id: ChainId,
    managed_node_sender: mpsc::Sender<ManagedNodeCommand>,
}

#[async_trait]
impl EventHandler<BlockInfo> for CrossUnsafeHandler {
    async fn handle(
        &self,
        block: BlockInfo,
        _state: &mut ProcessorState,
    ) -> Result<BlockInfo, ChainProcessorError> {
        trace!(
            target: "supervisor::chain_processor",
            chain_id = self.chain_id,
            block_number = block.number,
            "Processing cross unsafe block"
        );

        let result = self.inner_handle(block).await;
        Metrics::record_block_processing(self.chain_id, Metrics::BLOCK_TYPE_CROSS_UNSAFE, &result);

        result
    }
}

impl CrossUnsafeHandler {
    async fn inner_handle(&self, block: BlockInfo) -> Result<BlockInfo, ChainProcessorError> {
        self.managed_node_sender
            .send(ManagedNodeCommand::UpdateCrossUnsafe { block_id: block.id() })
            .await
            .map_err(|err| {
                warn!(
                    target: "supervisor::chain_processor::managed_node",
                    chain_id = self.chain_id,
                    %block,
                    %err,
                    "Failed to send cross unsafe block update"
                );
                ChainProcessorError::ChannelSendFailed(err.to_string())
            })?;
        Ok(block)
    }
}

/// Handler for cross safe blocks.
/// This handler processes cross safe blocks by updating the managed node.
#[derive(Debug, Constructor)]
pub struct CrossSafeHandler {
    chain_id: ChainId,
    managed_node_sender: mpsc::Sender<ManagedNodeCommand>,
}

#[async_trait]
impl EventHandler<DerivedRefPair> for CrossSafeHandler {
    async fn handle(
        &self,
        derived_ref_pair: DerivedRefPair,
        _state: &mut ProcessorState,
    ) -> Result<BlockInfo, ChainProcessorError> {
        trace!(
            target: "supervisor::chain_processor",
            chain_id = self.chain_id,
            block_number = derived_ref_pair.derived.number,
            "Processing cross safe block"
        );

        let result = self.inner_handle(derived_ref_pair).await;
        Metrics::record_block_processing(self.chain_id, Metrics::BLOCK_TYPE_CROSS_SAFE, &result);
        result
    }
}

impl CrossSafeHandler {
    async fn inner_handle(
        &self,
        derived_ref_pair: DerivedRefPair,
    ) -> Result<BlockInfo, ChainProcessorError> {
        self.managed_node_sender
            .send(ManagedNodeCommand::UpdateCrossSafe {
                source_block_id: derived_ref_pair.source.id(),
                derived_block_id: derived_ref_pair.derived.id(),
            })
            .await
            .map_err(|err| {
                warn!(
                    target: "supervisor::chain_processor::managed_node",
                    chain_id = self.chain_id,
                    %derived_ref_pair,
                    %err,
                    "Failed to send cross safe block update"
                );
                ChainProcessorError::ChannelSendFailed(err.to_string())
            })?;
        Ok(derived_ref_pair.derived)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::syncnode::{
        BlockProvider, ManagedNodeController, ManagedNodeDataProvider, ManagedNodeError,
    };
    use alloy_primitives::B256;
    use alloy_rpc_types_eth::BlockNumHash;
    use async_trait::async_trait;
    use kona_interop::DerivedRefPair;
    use kona_protocol::BlockInfo;
    use kona_supervisor_types::{BlockSeal, OutputV0, Receipts};
    use mockall::mock;

    mock!(
        #[derive(Debug)]
        pub Node {}

        #[async_trait]
        impl BlockProvider for Node {
            async fn fetch_receipts(&self, _block_hash: B256) -> Result<Receipts, ManagedNodeError>;
            async fn block_by_number(&self, _number: u64) -> Result<BlockInfo, ManagedNodeError>;
        }

        #[async_trait]
        impl ManagedNodeDataProvider for Node {
            async fn output_v0_at_timestamp(
                &self,
                _timestamp: u64,
            ) -> Result<OutputV0, ManagedNodeError>;

            async fn pending_output_v0_at_timestamp(
                &self,
                _timestamp: u64,
            ) -> Result<OutputV0, ManagedNodeError>;

            async fn l2_block_ref_by_timestamp(
                &self,
                _timestamp: u64,
            ) -> Result<BlockInfo, ManagedNodeError>;
        }

        #[async_trait]
        impl ManagedNodeController for Node {
            async fn update_finalized(
                &self,
                _finalized_block_id: BlockNumHash,
            ) -> Result<(), ManagedNodeError>;

            async fn update_cross_unsafe(
                &self,
                cross_unsafe_block_id: BlockNumHash,
            ) -> Result<(), ManagedNodeError>;

            async fn update_cross_safe(
                &self,
                source_block_id: BlockNumHash,
                derived_block_id: BlockNumHash,
            ) -> Result<(), ManagedNodeError>;

            async fn reset(&self) -> Result<(), ManagedNodeError>;

            async fn invalidate_block(&self, seal: BlockSeal) -> Result<(), ManagedNodeError>;
        }
    );

    #[tokio::test]
    async fn test_handle_cross_unsafe_update_triggers() {
        use crate::syncnode::ManagedNodeCommand;

        let (tx, mut rx) = mpsc::channel(8);
        let chain_id = 1;
        let handler = CrossUnsafeHandler::new(chain_id, tx);

        let block =
            BlockInfo { number: 42, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 123456 };
        let mut state = ProcessorState::new();

        // Call the handler
        let result = handler.handle(block, &mut state).await;
        assert!(result.is_ok());

        // The handler should send the correct command
        if let Some(ManagedNodeCommand::UpdateCrossUnsafe { block_id }) = rx.recv().await {
            assert_eq!(block_id, block.id());
        } else {
            panic!("Expected UpdateCrossUnsafe command");
        }
    }

    #[tokio::test]
    async fn test_handle_cross_unsafe_update_error() {
        let (tx, rx) = mpsc::channel(8);
        let chain_id = 1;
        let handler = CrossUnsafeHandler::new(chain_id, tx);

        // Drop the receiver to simulate a send error
        drop(rx);

        let block =
            BlockInfo { number: 42, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 123456 };
        let mut state = ProcessorState::new();

        // Call the handler, which should now error
        let result = handler.handle(block, &mut state).await;
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn test_handle_cross_safe_update_triggers() {
        use crate::syncnode::ManagedNodeCommand;

        let (tx, mut rx) = mpsc::channel(8);
        let chain_id = 1;
        let handler = CrossSafeHandler::new(chain_id, tx);

        let derived =
            BlockInfo { number: 42, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 123456 };
        let source =
            BlockInfo { number: 1, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 123456 };
        let derived_ref_pair = DerivedRefPair { source, derived };
        let mut state = ProcessorState::new();

        // Call the handler
        let result = handler.handle(derived_ref_pair, &mut state).await;
        assert!(result.is_ok());

        // The handler should send the correct command
        if let Some(ManagedNodeCommand::UpdateCrossSafe { source_block_id, derived_block_id }) =
            rx.recv().await
        {
            assert_eq!(source_block_id, source.id());
            assert_eq!(derived_block_id, derived.id());
        } else {
            panic!("Expected UpdateCrossSafe command");
        }
    }

    #[tokio::test]
    async fn test_handle_cross_safe_update_error() {
        let (tx, rx) = mpsc::channel(8);
        let chain_id = 1;
        let handler = CrossSafeHandler::new(chain_id, tx);

        // Drop the receiver to simulate a send error
        drop(rx);

        let derived =
            BlockInfo { number: 42, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 123456 };
        let source =
            BlockInfo { number: 1, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 123456 };
        let derived_ref_pair = DerivedRefPair { source, derived };
        let mut state = ProcessorState::new();

        // Call the handler, which should now error
        let result = handler.handle(derived_ref_pair, &mut state).await;
        assert!(result.is_err());
    }
}
