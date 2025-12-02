use super::EventHandler;
use crate::{
    ChainProcessorError, ProcessorState, chain_processor::Metrics, syncnode::ManagedNodeCommand,
};
use alloy_primitives::ChainId;
use async_trait::async_trait;
use derive_more::Constructor;
use kona_protocol::BlockInfo;
use kona_supervisor_storage::HeadRefStorageWriter;
use std::sync::Arc;
use tokio::sync::mpsc;
use tracing::{trace, warn};

/// Handler for finalized block updates.
/// This handler processes finalized block updates by updating the managed node and state manager.
#[derive(Debug, Constructor)]
pub struct FinalizedHandler<W> {
    chain_id: ChainId,
    managed_node_sender: mpsc::Sender<ManagedNodeCommand>,
    db_provider: Arc<W>,
}

#[async_trait]
impl<W> EventHandler<BlockInfo> for FinalizedHandler<W>
where
    W: HeadRefStorageWriter + Send + Sync + 'static,
{
    async fn handle(
        &self,
        finalized_source_block: BlockInfo,
        _state: &mut ProcessorState,
    ) -> Result<BlockInfo, ChainProcessorError> {
        trace!(
            target: "supervisor::chain_processor",
            chain_id = self.chain_id,
            block_number = finalized_source_block.number,
            "Processing finalized L1 update"
        );

        let result = self.inner_handle(finalized_source_block).await;
        Metrics::record_block_processing(self.chain_id, Metrics::BLOCK_TYPE_FINALIZED, &result);

        result
    }
}

impl<W> FinalizedHandler<W>
where
    W: HeadRefStorageWriter + Send + Sync + 'static,
{
    async fn inner_handle(
        &self,
        finalized_source_block: BlockInfo,
    ) -> Result<BlockInfo, ChainProcessorError> {
        let finalized_derived_block = self
            .db_provider
            .update_finalized_using_source(finalized_source_block)
            .inspect_err(|err| {
                warn!(
                    target: "supervisor::chain_processor::db",
                    chain_id = self.chain_id,
                    %finalized_source_block,
                    %err,
                    "Failed to update finalized block using source"
                );
            })?;

        self.managed_node_sender
            .send(ManagedNodeCommand::UpdateFinalized { block_id: finalized_derived_block.id() })
            .await
            .map_err(|err| {
                warn!(
                    target: "supervisor::chain_processor::managed_node",
                    chain_id = self.chain_id,
                    %finalized_derived_block,
                    %err,
                    "Failed to send finalized block update"
                );
                ChainProcessorError::ChannelSendFailed(err.to_string())
            })?;
        Ok(finalized_derived_block)
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
    use kona_supervisor_storage::{HeadRefStorageWriter, StorageError};
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

    mock!(
        #[derive(Debug)]
        pub Db {}

        impl HeadRefStorageWriter for Db {
            fn update_finalized_using_source(
                &self,
                block_info: BlockInfo,
            ) -> Result<BlockInfo, StorageError>;

            fn update_current_cross_unsafe(
                &self,
                block: &BlockInfo,
            ) -> Result<(), StorageError>;

            fn update_current_cross_safe(
                &self,
                block: &BlockInfo,
            ) -> Result<DerivedRefPair, StorageError>;
        }
    );

    #[tokio::test]
    async fn test_handle_finalized_source_update_triggers() {
        use crate::syncnode::ManagedNodeCommand;

        let mut mocknode = MockNode::new();
        let mut mockdb = MockDb::new();
        let mut state = ProcessorState::new();

        let finalized_source_block =
            BlockInfo { number: 99, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 1234578 };

        // The finalized_derived_block returned by update_finalized_using_source
        let finalized_derived_block =
            BlockInfo { number: 5, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 1234578 };

        // Expect update_finalized_using_source to be called with finalized_source_block
        mockdb.expect_update_finalized_using_source().returning(move |block_info: BlockInfo| {
            assert_eq!(block_info, finalized_source_block);
            Ok(finalized_derived_block)
        });

        // Expect update_finalized to be called with the derived block's id
        let finalized_derived_block_id = finalized_derived_block.id();
        mocknode.expect_update_finalized().returning(move |block_id| {
            assert_eq!(block_id, finalized_derived_block_id);
            Ok(())
        });

        let writer = Arc::new(mockdb);

        // Set up the channel and spawn a task to handle the command
        let (tx, mut rx) = mpsc::channel(8);

        let handler = FinalizedHandler::new(
            1, // chain_id
            tx, writer,
        );
        let result = handler.handle(finalized_source_block, &mut state).await;
        assert!(result.is_ok());

        // The handler should send the correct command
        if let Some(ManagedNodeCommand::UpdateFinalized { block_id }) = rx.recv().await {
            assert_eq!(block_id, finalized_derived_block.id());
        } else {
            panic!("Expected UpdateFinalized command");
        }
    }

    #[tokio::test]
    async fn test_handle_finalized_source_update_db_error() {
        let mut mocknode = MockNode::new();
        let mut mockdb = MockDb::new();
        let mut state = ProcessorState::new();

        let finalized_source_block =
            BlockInfo { number: 99, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 1234578 };

        // DB returns error
        mockdb
            .expect_update_finalized_using_source()
            .returning(|_block_info: BlockInfo| Err(StorageError::DatabaseNotInitialised));

        // Managed node's update_finalized should NOT be called
        mocknode.expect_update_finalized().never();

        let writer = Arc::new(mockdb);
        let (tx, mut rx) = mpsc::channel(8);

        let handler = FinalizedHandler::new(
            1, // chain_id
            tx, writer,
        );
        let result = handler.handle(finalized_source_block, &mut state).await;
        assert!(result.is_err());

        // Ensure no command was sent
        assert!(rx.try_recv().is_err());
    }

    #[tokio::test]
    async fn test_handle_finalized_source_update_managed_node_error() {
        let mut mockdb = MockDb::new();
        let mut state = ProcessorState::new();

        let finalized_source_block =
            BlockInfo { number: 99, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 1234578 };

        let finalized_derived_block =
            BlockInfo { number: 5, hash: B256::ZERO, parent_hash: B256::ZERO, timestamp: 1234578 };

        // DB returns the derived block as usual
        mockdb.expect_update_finalized_using_source().returning(move |block_info: BlockInfo| {
            assert_eq!(block_info, finalized_source_block);
            Ok(finalized_derived_block)
        });

        let writer = Arc::new(mockdb);

        // Set up the channel and immediately drop the receiver to simulate a send error
        let (tx, rx) = mpsc::channel(8);
        drop(rx);

        let handler = FinalizedHandler::new(
            1, // chain_id
            tx, writer,
        );
        let result = handler.handle(finalized_source_block, &mut state).await;
        assert!(result.is_err());
    }
}
