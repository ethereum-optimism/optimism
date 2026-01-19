//! [`ManagedNode`] implementation for handling events from the managed node.

use super::{
    BlockProvider, ManagedNodeClient, ManagedNodeController, ManagedNodeDataProvider,
    ManagedNodeError, SubscriptionHandler, resetter::Resetter,
};
use crate::event::ChainEvent;
use alloy_eips::BlockNumberOrTag;
use alloy_network::Ethereum;
use alloy_primitives::{B256, ChainId};
use alloy_provider::{Provider, RootProvider};
use alloy_rpc_types_eth::BlockNumHash;
use async_trait::async_trait;
use kona_interop::{BlockReplacement, DerivedRefPair};
use kona_protocol::BlockInfo;
use kona_supervisor_storage::{DerivationStorageReader, HeadRefStorageReader, LogStorageReader};
use kona_supervisor_types::{BlockSeal, OutputV0, Receipts};
use std::sync::Arc;
use tokio::sync::{Mutex, mpsc};
use tracing::{debug, error, trace, warn};

/// [`ManagedNode`] processes events dispatched from the managed node.
///
/// It implements `SubscriptionHandler`, forwards resulting `ChainEvent`s to the chain
/// processor, and delegates control operations to the underlying client/resetter.
/// The WebSocket subscription lifecycle (subscription creation, reconnection/restart)
/// is managed by the supervisor actor and the client, not by this type.
#[derive(Debug)]
pub struct ManagedNode<DB, C> {
    /// The attached web socket client
    client: Arc<C>,
    /// Shared L1 provider for fetching receipts
    l1_provider: RootProvider<Ethereum>,
    /// Resetter for handling node resets
    resetter: Arc<Resetter<DB, C>>,
    /// Channel for sending events to the chain processor
    chain_event_sender: mpsc::Sender<ChainEvent>,

    /// Cached chain ID
    chain_id: Mutex<Option<ChainId>>,
}

impl<DB, C> ManagedNode<DB, C>
where
    DB: LogStorageReader + DerivationStorageReader + HeadRefStorageReader + Send + Sync + 'static,
    C: ManagedNodeClient + Send + Sync + 'static,
{
    /// Creates a new [`ManagedNode`] with the specified client.
    pub fn new(
        client: Arc<C>,
        db_provider: Arc<DB>,
        l1_provider: RootProvider<Ethereum>,
        chain_event_sender: mpsc::Sender<ChainEvent>,
    ) -> Self {
        let resetter = Arc::new(Resetter::new(client.clone(), l1_provider.clone(), db_provider));

        Self { client, resetter, l1_provider, chain_event_sender, chain_id: Mutex::new(None) }
    }

    /// Returns the [`ChainId`] of the [`ManagedNode`].
    /// If the chain ID is already cached, it returns that.
    /// If not, it fetches the chain ID from the managed node.
    pub async fn chain_id(&self) -> Result<ChainId, ManagedNodeError> {
        // we are caching the chain ID here to avoid multiple calls to the client
        // there is a possibility that chain ID might be being cached in the client already
        // but we are caching it here to make sure it caches in the `ManagedNode` context
        let mut cache = self.chain_id.lock().await;
        if let Some(chain_id) = *cache {
            Ok(chain_id)
        } else {
            let chain_id = self.client.chain_id().await?;
            *cache = Some(chain_id);
            Ok(chain_id)
        }
    }
}

#[async_trait]
impl<DB, C> SubscriptionHandler for ManagedNode<DB, C>
where
    DB: LogStorageReader + DerivationStorageReader + HeadRefStorageReader + Send + Sync + 'static,
    C: ManagedNodeClient + Send + Sync + 'static,
{
    async fn handle_exhaust_l1(
        &self,
        derived_ref_pair: &DerivedRefPair,
    ) -> Result<(), ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(
            target: "supervisor::managed_node",
            %chain_id,
            %derived_ref_pair,
            "Handling L1 exhaust event"
        );

        let next_block_number = derived_ref_pair.source.number + 1;
        let next_block = self
            .l1_provider
            .get_block_by_number(BlockNumberOrTag::Number(next_block_number))
            .await
            .map_err(|err| {
                error!(target: "supervisor::managed_node", %chain_id, %err, "Failed to fetch next L1 block");
                ManagedNodeError::GetBlockByNumberFailed(next_block_number)
            })?;

        let block = match next_block {
            Some(block) => block,
            None => {
                // If the block is None, it means the block is either empty or unavailable.
                // ignore this case
                return Ok(());
            }
        };

        let new_source = BlockInfo {
            hash: block.header.hash,
            number: block.header.number,
            parent_hash: block.header.parent_hash,
            timestamp: block.header.timestamp,
        };

        if new_source.parent_hash != derived_ref_pair.source.hash {
            // this could happen due to a reorg.
            // this case should be handled by the reorg manager
            debug!(
                target: "supervisor::managed_node",
                %chain_id,
                %new_source,
                current_source = %derived_ref_pair.source,
                "Parent hash mismatch. Possible reorg detected"
            );
        }

        self.client.provide_l1(new_source).await.inspect_err(|err| {
            error!(
                target: "supervisor::managed_node",
                %chain_id,
                %new_source,
                %err,
                "Failed to provide L1 block"
            );
        })?;
        Ok(())
    }

    async fn handle_reset(&self, reset_id: &str) -> Result<(), ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(target: "supervisor::managed_node", %chain_id, reset_id, "Handling reset event");

        self.resetter.reset().await?;
        Ok(())
    }

    async fn handle_unsafe_block(&self, unsafe_block: &BlockInfo) -> Result<(), ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(target: "supervisor::managed_node", %chain_id, %unsafe_block, "Unsafe block event received");

        self.chain_event_sender.send(ChainEvent::UnsafeBlock { block: *unsafe_block }).await.map_err(|err| {
            warn!(target: "supervisor::managed_node", %chain_id, %err, "Failed to send unsafe block event");
            ManagedNodeError::ChannelSendFailed(err.to_string())
        })?;
        Ok(())
    }

    async fn handle_derivation_update(
        &self,
        derived_ref_pair: &DerivedRefPair,
    ) -> Result<(), ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(target: "supervisor::managed_node", %chain_id, "Derivation update event received");

        self.chain_event_sender.send(ChainEvent::DerivedBlock { derived_ref_pair: *derived_ref_pair }).await.map_err(|err| {
            warn!(target: "supervisor::managed_node", %chain_id, %err, "Failed to send derivation update event");
            ManagedNodeError::ChannelSendFailed(err.to_string())
        })?;
        Ok(())
    }

    async fn handle_replace_block(
        &self,
        replacement: &BlockReplacement,
    ) -> Result<(), ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(target: "supervisor::managed_node", %chain_id, %replacement, "Block replacement received");

        self.chain_event_sender.send(ChainEvent::BlockReplaced { replacement: *replacement }).await.map_err(|err| {
            warn!(target: "supervisor::managed_node", %chain_id, %err, "Failed to send block replacement event");
            ManagedNodeError::ChannelSendFailed(err.to_string())
        })?;
        Ok(())
    }

    async fn handle_derivation_origin_update(
        &self,
        origin: &BlockInfo,
    ) -> Result<(), ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(target: "supervisor::managed_node", %chain_id, %origin, "Derivation origin update received");

        self.chain_event_sender.send(ChainEvent::DerivationOriginUpdate { origin: *origin }).await.map_err(|err| {
            warn!(target: "supervisor::managed_node", %chain_id, %err, "Failed to send derivation origin update event");
            ManagedNodeError::ChannelSendFailed(err.to_string())
        })?;
        Ok(())
    }
}

/// Implements [`BlockProvider`] for [`ManagedNode`] by delegating to the underlying WebSocket
/// client.
#[async_trait]
impl<DB, C> BlockProvider for ManagedNode<DB, C>
where
    DB: LogStorageReader + DerivationStorageReader + HeadRefStorageReader + Send + Sync + 'static,
    C: ManagedNodeClient + Send + Sync + 'static,
{
    async fn block_by_number(&self, block_number: u64) -> Result<BlockInfo, ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(target: "supervisor::managed_node", %chain_id, block_number, "Fetching block by number");

        let block = self.client.block_ref_by_number(block_number).await?;
        Ok(block)
    }
    async fn fetch_receipts(&self, block_hash: B256) -> Result<Receipts, ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(target: "supervisor::managed_node", %chain_id, %block_hash, "Fetching receipts for block");

        let receipt = self.client.fetch_receipts(block_hash).await?;
        Ok(receipt)
    }
}

#[async_trait]
impl<DB, C> ManagedNodeDataProvider for ManagedNode<DB, C>
where
    DB: LogStorageReader + DerivationStorageReader + HeadRefStorageReader + Send + Sync + 'static,
    C: ManagedNodeClient + Send + Sync + 'static,
{
    async fn output_v0_at_timestamp(&self, timestamp: u64) -> Result<OutputV0, ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(target: "supervisor::managed_node", %chain_id, timestamp, "Fetching output v0 at timestamp");

        let outputv0 = self.client.output_v0_at_timestamp(timestamp).await?;
        Ok(outputv0)
    }

    async fn pending_output_v0_at_timestamp(
        &self,
        timestamp: u64,
    ) -> Result<OutputV0, ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(target: "supervisor::managed_node", %chain_id, timestamp, "Fetching pending output v0 at timestamp");

        let outputv0 = self.client.pending_output_v0_at_timestamp(timestamp).await?;
        Ok(outputv0)
    }

    async fn l2_block_ref_by_timestamp(
        &self,
        timestamp: u64,
    ) -> Result<BlockInfo, ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(target: "supervisor::managed_node", %chain_id, timestamp, "Fetching L2 block ref by timestamp");

        let block = self.client.l2_block_ref_by_timestamp(timestamp).await?;
        Ok(block)
    }
}

#[async_trait]
impl<DB, C> ManagedNodeController for ManagedNode<DB, C>
where
    DB: LogStorageReader + DerivationStorageReader + HeadRefStorageReader + Send + Sync + 'static,
    C: ManagedNodeClient + Send + Sync + 'static,
{
    async fn update_finalized(
        &self,
        finalized_block_id: BlockNumHash,
    ) -> Result<(), ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(
            target: "supervisor::managed_node",
            %chain_id,
            finalized_block_number = finalized_block_id.number,
            "Updating finalized block"
        );

        self.client.update_finalized(finalized_block_id).await?;
        Ok(())
    }

    async fn update_cross_unsafe(
        &self,
        cross_unsafe_block_id: BlockNumHash,
    ) -> Result<(), ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(
            target: "supervisor::managed_node",
            %chain_id,
            cross_unsafe_block_number = cross_unsafe_block_id.number,
            "Updating cross unsafe block",
        );

        self.client.update_cross_unsafe(cross_unsafe_block_id).await?;
        Ok(())
    }

    async fn update_cross_safe(
        &self,
        source_block_id: BlockNumHash,
        derived_block_id: BlockNumHash,
    ) -> Result<(), ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(
            target: "supervisor::managed_node",
            %chain_id,
            source_block_number = source_block_id.number,
            derived_block_number = derived_block_id.number,
            "Updating cross safe block"
        );
        self.client.update_cross_safe(source_block_id, derived_block_id).await?;
        Ok(())
    }

    async fn reset(&self) -> Result<(), ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(target: "supervisor::managed_node", %chain_id, "Resetting managed node state");

        self.resetter.reset().await?;
        Ok(())
    }

    async fn invalidate_block(&self, block_seal: BlockSeal) -> Result<(), ManagedNodeError> {
        let chain_id = self.chain_id().await?;
        trace!(
            target: "supervisor::managed_node",
            %chain_id,
            block_number = block_seal.number,
            "Invalidating block"
        );

        self.client.invalidate_block(block_seal).await?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::syncnode::ClientError;
    use alloy_primitives::{B256, ChainId, hex::FromHex};
    use alloy_provider::RootProvider;
    use alloy_rpc_client::RpcClient;
    use alloy_transport::mock::*;
    use jsonrpsee::core::client::Subscription;
    use kona_interop::{BlockReplacement, DerivedRefPair, SafetyLevel};
    use kona_protocol::BlockInfo;
    use kona_supervisor_storage::{
        DerivationStorageReader, HeadRefStorageReader, LogStorageReader, StorageError,
    };
    use kona_supervisor_types::{BlockSeal, Log, OutputV0, Receipts, SubscriptionEvent, SuperHead};
    use mockall::{mock, predicate::*};
    use std::sync::Arc;
    use tokio::sync::mpsc;

    mock! {
        #[derive(Debug)]
        pub Client {}

        #[async_trait]
        impl ManagedNodeClient for Client {
            async fn chain_id(&self) -> Result<ChainId, ClientError>;
            async fn subscribe_events(&self) -> Result<Subscription<SubscriptionEvent>, ClientError>;
            async fn fetch_receipts(&self, block_hash: B256) -> Result<Receipts, ClientError>;
            async fn output_v0_at_timestamp(&self, timestamp: u64) -> Result<OutputV0, ClientError>;
            async fn pending_output_v0_at_timestamp(&self, timestamp: u64) -> Result<OutputV0, ClientError>;
            async fn l2_block_ref_by_timestamp(&self, timestamp: u64) -> Result<BlockInfo, ClientError>;
            async fn block_ref_by_number(&self, block_number: u64) -> Result<BlockInfo, ClientError>;
            async fn reset_pre_interop(&self) -> Result<(), ClientError>;
            async fn reset(&self, unsafe_id: BlockNumHash, cross_unsafe_id: BlockNumHash, local_safe_id: BlockNumHash, cross_safe_id: BlockNumHash, finalised_id: BlockNumHash) -> Result<(), ClientError>;
            async fn invalidate_block(&self, seal: BlockSeal) -> Result<(), ClientError>;
            async fn provide_l1(&self, block_info: BlockInfo) -> Result<(), ClientError>;
            async fn update_finalized(&self, finalized_block_id: BlockNumHash) -> Result<(), ClientError>;
            async fn update_cross_unsafe(&self, cross_unsafe_block_id: BlockNumHash) -> Result<(), ClientError>;
            async fn update_cross_safe(&self, source_block_id: BlockNumHash, derived_block_id: BlockNumHash) -> Result<(), ClientError>;
            async fn reset_ws_client(&self);
        }
    }

    mock! {
        #[derive(Debug)]
        pub Db {}

        impl LogStorageReader for Db {
            fn get_block(&self, block_number: u64) -> Result<BlockInfo, StorageError>;
            fn get_latest_block(&self) -> Result<BlockInfo, StorageError>;
            fn get_log(&self, block_number: u64, log_index: u32) -> Result<Log, StorageError>;
            fn get_logs(&self, block_number: u64) -> Result<Vec<Log>, StorageError>;
        }

        impl DerivationStorageReader for Db {
            fn derived_to_source(&self, derived_block_id: BlockNumHash) -> Result<BlockInfo, StorageError>;
            fn latest_derived_block_at_source(&self, _source_block_id: BlockNumHash) -> Result<BlockInfo, StorageError>;
            fn latest_derivation_state(&self) -> Result<DerivedRefPair, StorageError>;
            fn get_source_block(&self, source_block_number: u64) -> Result<BlockInfo, StorageError>;
            fn get_activation_block(&self) -> Result<BlockInfo, StorageError>;
        }

        impl HeadRefStorageReader for Db {
            fn get_safety_head_ref(&self, level: SafetyLevel) -> Result<BlockInfo, StorageError>;
            fn get_super_head(&self) -> Result<SuperHead, StorageError>;
        }
    }

    #[tokio::test]
    async fn test_chain_id_caching() {
        let mut client = MockClient::new();

        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));

        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        // First call fetches from client
        let id1 = node.chain_id().await.unwrap();
        assert_eq!(id1, ChainId::from(42u64));
        // Second call uses cache
        let id2 = node.chain_id().await.unwrap();
        assert_eq!(id2, ChainId::from(42u64));
    }

    #[tokio::test]
    async fn test_handle_unsafe_block_sends_event() {
        let unsafe_block =
            BlockInfo { hash: B256::ZERO, number: 1, parent_hash: B256::ZERO, timestamp: 123 };

        let mut client = MockClient::new();

        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));

        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, mut rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let result = node.handle_unsafe_block(&unsafe_block).await;
        assert!(result.is_ok());

        let event = rx.recv().await.unwrap();
        match event {
            ChainEvent::UnsafeBlock { block } => assert_eq!(block.number, 1),
            _ => panic!("Wrong event"),
        }
    }

    #[tokio::test]
    async fn test_handle_derivation_update_sends_event() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, mut rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let derived_ref_pair = DerivedRefPair {
            source: BlockInfo::new(B256::from([0u8; 32]), 0, B256::ZERO, 0),
            derived: BlockInfo::new(B256::from([1u8; 32]), 1, B256::ZERO, 0),
        };

        let result = node.handle_derivation_update(&derived_ref_pair).await;
        assert!(result.is_ok());

        let event = rx.recv().await.unwrap();
        match event {
            ChainEvent::DerivedBlock { derived_ref_pair: pair } => {
                assert_eq!(pair, derived_ref_pair);
            }
            _ => panic!("Wrong event"),
        }
    }

    #[tokio::test]
    async fn test_handle_replace_block_sends_event() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, mut rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let replacement = BlockReplacement {
            replacement: BlockInfo::new(B256::from([1u8; 32]), 1, B256::ZERO, 0),
            invalidated: B256::from([2u8; 32]),
        };

        let result = node.handle_replace_block(&replacement).await;
        assert!(result.is_ok());

        let event = rx.recv().await.unwrap();
        match event {
            ChainEvent::BlockReplaced { replacement: rep } => assert_eq!(rep, replacement),
            _ => panic!("Wrong event"),
        }
    }

    #[tokio::test]
    async fn test_handle_derivation_origin_update_sends_event() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, mut rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let origin =
            BlockInfo { hash: B256::ZERO, number: 10, parent_hash: B256::ZERO, timestamp: 12345 };

        let result = node.handle_derivation_origin_update(&origin).await;
        assert!(result.is_ok());

        let event = rx.recv().await.unwrap();
        match event {
            ChainEvent::DerivationOriginUpdate { origin: block } => assert_eq!(block.number, 10),
            _ => panic!("Wrong event"),
        }
    }

    #[tokio::test]
    async fn test_handle_exhaust_l1_calls_provide_l1_on_success() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        client.expect_provide_l1().times(1).returning(|_| Ok(()));

        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());

        let derived_ref_pair = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from_hex(
                    "0x1f68ac259155e2f38211ddad0f0a15394d55417b185a93923e2abe71bb7a4d6d",
                )
                .unwrap(),
                number: 5,
                parent_hash: B256::from([14u8; 32]),
                timestamp: 300,
            },
            derived: BlockInfo {
                hash: B256::from([11u8; 32]),
                number: 40,
                parent_hash: B256::from([12u8; 32]),
                timestamp: 301,
            },
        };

        let next_block = r#"{
            "number": "6",
            "hash": "0xd5f1812548be429cbdc6376b29611fc49e06f1359758c4ceaaa3b393e2239f9c",
            "mixHash": "0x24900fb3da77674a861c428429dce0762707ecb6052325bbd9b3c64e74b5af9d",
            "parentHash": "0x1f68ac259155e2f38211ddad0f0a15394d55417b185a93923e2abe71bb7a4d6d",
            "nonce": "0x378da40ff335b070",
            "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
            "logsBloom": "0x00000000000000100000004080000000000500000000000000020000100000000800001000000004000001000000000000000800040010000020100000000400000010000000000000000040000000000000040000000000000000000000000000000400002400000000000000000000000000000004000004000000000000840000000800000080010004000000001000000800000000000000000000000000000000000800000000000040000000020000000000000000000800000400000000000000000000000600000400000000002000000000000000000000004000000000000000100000000000000000000000000000000000040000900010000000",
            "transactionsRoot":"0x4d0c8e91e16bdff538c03211c5c73632ed054d00a7e210c0eb25146c20048126",
            "stateRoot": "0x91309efa7e42c1f137f31fe9edbe88ae087e6620d0d59031324da3e2f4f93233",
            "receiptsRoot": "0x68461ab700003503a305083630a8fb8d14927238f0bc8b6b3d246c0c64f21f4a",
            "miner":"0xb42b6c4a95406c78ff892d270ad20b22642e102d",
            "difficulty": "0x66e619a",
            "totalDifficulty": "0x1e875d746ae",
            "extraData": "0xd583010502846765746885676f312e37856c696e7578",
            "size": "0x334",
            "gasLimit": "0x47e7c4",
            "gasUsed": "0x37993",
            "timestamp": "0x5835c54d",
            "uncles": [],
            "transactions": [
                "0xa0807e117a8dd124ab949f460f08c36c72b710188f01609595223b325e58e0fc",
                "0xeae6d797af50cb62a596ec3939114d63967c374fa57de9bc0f4e2b576ed6639d"
            ],
            "baseFeePerGas": "0x7",
            "withdrawalsRoot": "0x7a4ecf19774d15cf9c15adf0dd8e8a250c128b26c9e2ab2a08d6c9c8ffbd104f",
            "withdrawals": [],
            "blobGasUsed": "0x0",
            "excessBlobGas": "0x0",
            "parentBeaconBlockRoot": "0x95c4dbd5b19f6fe3cbc3183be85ff4e85ebe75c5b4fc911f1c91e5b7a554a685"
        }"#;

        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));

        asserter.push(MockResponse::Success(serde_json::from_str(next_block).unwrap()));

        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let result = node.handle_exhaust_l1(&derived_ref_pair).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_handle_exhaust_l1_calls_provide_l1_on_parent_hash_mismatch() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        client.expect_provide_l1().times(1).returning(|_| Ok(())); // Should be called

        let client = Arc::new(client);
        let db = MockDb::new();

        let derived_ref_pair = DerivedRefPair {
            source: BlockInfo {
                hash: B256::from([1u8; 32]), // This will NOT match parent_hash below
                number: 5,
                parent_hash: B256::from([14u8; 32]),
                timestamp: 300,
            },
            derived: BlockInfo {
                hash: B256::from([11u8; 32]),
                number: 40,
                parent_hash: B256::from([12u8; 32]),
                timestamp: 301,
            },
        };

        // Block with mismatched parent_hash
        let next_block = r#"{
            "number": "10",
            "hash": "0xd5f1812548be429cbdc6376b29611fc49e06f1359758c4ceaaa3b393e2239f9c",
            "mixHash": "0x24900fb3da77674a861c428429dce0762707ecb6052325bbd9b3c64e74b5af9d",
            "parentHash": "0x1f68ac259155e2f38211ddad0f0a15394d55417b185a93923e2abe71bb7a4d6d",
            "nonce": "0x378da40ff335b070",
            "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
            "logsBloom": "0x00000000000000100000004080000000000500000000000000020000100000000800001000000004000001000000000000000800040010000020100000000400000010000000000000000040000000000000040000000000000000000000000000000400002400000000000000000000000000000004000004000000000000840000000800000080010004000000001000000800000000000000000000000000000000000800000000000040000000020000000000000000000800000400000000000000000000000600000400000000002000000000000000000000004000000000000000100000000000000000000000000000000000040000900010000000",
            "transactionsRoot":"0x4d0c8e91e16bdff538c03211c5c73632ed054d00a7e210c0eb25146c20048126",
            "stateRoot": "0x91309efa7e42c1f137f31fe9edbe88ae087e6620d0d59031324da3e2f4f93233",
            "receiptsRoot": "0x68461ab700003503a305083630a8fb8d14927238f0bc8b6b3d246c0c64f21f4a",
            "miner":"0xb42b6c4a95406c78ff892d270ad20b22642e102d",
            "difficulty": "0x66e619a",
            "totalDifficulty": "0x1e875d746ae",
            "extraData": "0xd583010502846765746885676f312e37856c696e7578",
            "size": "0x334",
            "gasLimit": "0x47e7c4",
            "gasUsed": "0x37993",
            "timestamp": "0x5835c54d",
            "uncles": [],
            "transactions": [
                "0xa0807e117a8dd124ab949f460f08c36c72b710188f01609595223b325e58e0fc",
                "0xeae6d797af50cb62a596ec3939114d63967c374fa57de9bc0f4e2b576ed6639d"
            ],
            "baseFeePerGas": "0x7",
            "withdrawalsRoot": "0x7a4ecf19774d15cf9c15adf0dd8e8a250c128b26c9e2ab2a08d6c9c8ffbd104f",
            "withdrawals": [],
            "blobGasUsed": "0x0",
            "excessBlobGas": "0x0",
            "parentBeaconBlockRoot": "0x95c4dbd5b19f6fe3cbc3183be85ff4e85ebe75c5b4fc911f1c91e5b7a554a685"
        }"#;

        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));

        asserter.push(MockResponse::Success(serde_json::from_str(next_block).unwrap()));

        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), Arc::new(db), l1_provider, tx);

        let result = node.handle_exhaust_l1(&derived_ref_pair).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_handle_reset_calls_resetter() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(2).returning(|| Ok(ChainId::from(42u64)));
        client.expect_reset_pre_interop().times(1).returning(|| Ok(()));

        let mut db = MockDb::new();
        db.expect_latest_derivation_state()
            .times(1)
            .returning(|| Err(StorageError::DatabaseNotInitialised));

        let client = Arc::new(client);
        let db = Arc::new(db);
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        // Just check that it completes without error
        let result = node.handle_reset("reset_id").await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_block_by_number_delegates_to_client() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        client.expect_block_ref_by_number().with(eq(10)).times(1).returning(|_| {
            Ok(BlockInfo {
                hash: B256::from([1u8; 32]),
                number: 10,
                parent_hash: B256::from([2u8; 32]),
                timestamp: 12345,
            })
        });

        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let block = node.block_by_number(10).await.unwrap();
        assert_eq!(block.number, 10);
        assert_eq!(block.hash, B256::from([1u8; 32]));
    }

    #[tokio::test]
    async fn test_fetch_receipts_delegates_to_client() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        client
            .expect_fetch_receipts()
            .withf(|hash| *hash == B256::from([1u8; 32]))
            .times(1)
            .returning(|_| Ok(Receipts::default()));

        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let receipts = node.fetch_receipts(B256::from([1u8; 32])).await.unwrap();
        assert!(receipts.is_empty());
    }

    #[tokio::test]
    async fn test_output_v0_at_timestamp_delegates_to_client() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        client
            .expect_output_v0_at_timestamp()
            .with(eq(12345))
            .times(1)
            .returning(|_| Ok(OutputV0::default()));

        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let output = node.output_v0_at_timestamp(12345).await.unwrap();
        assert_eq!(output, OutputV0::default());
    }

    #[tokio::test]
    async fn test_pending_output_v0_at_timestamp_delegates_to_client() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        client
            .expect_pending_output_v0_at_timestamp()
            .with(eq(54321))
            .times(1)
            .returning(|_| Ok(OutputV0::default()));

        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let output = node.pending_output_v0_at_timestamp(54321).await.unwrap();
        assert_eq!(output, OutputV0::default());
    }

    #[tokio::test]
    async fn test_l2_block_ref_by_timestamp_delegates_to_client() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        client.expect_l2_block_ref_by_timestamp().with(eq(11111)).times(1).returning(|_| {
            Ok(BlockInfo {
                hash: B256::from([9u8; 32]),
                number: 99,
                parent_hash: B256::from([8u8; 32]),
                timestamp: 11111,
            })
        });

        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let block = node.l2_block_ref_by_timestamp(11111).await.unwrap();
        assert_eq!(block.number, 99);
        assert_eq!(block.timestamp, 11111);
    }

    #[tokio::test]
    async fn test_update_finalized_delegates_to_client() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        client
            .expect_update_finalized()
            .withf(|block_id| block_id.number == 100)
            .times(1)
            .returning(|_| Ok(()));

        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let block_id = BlockNumHash { number: 100, hash: B256::from([1u8; 32]) };
        let result = node.update_finalized(block_id).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_update_cross_unsafe_delegates_to_client() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        client
            .expect_update_cross_unsafe()
            .withf(|block_id| block_id.number == 200)
            .times(1)
            .returning(|_| Ok(()));

        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let block_id = BlockNumHash { number: 200, hash: B256::from([2u8; 32]) };
        let result = node.update_cross_unsafe(block_id).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_update_cross_safe_delegates_to_client() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        client
            .expect_update_cross_safe()
            .withf(|source, derived| source.number == 300 && derived.number == 301)
            .times(1)
            .returning(|_, _| Ok(()));

        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let source_block_id = BlockNumHash { number: 300, hash: B256::from([3u8; 32]) };
        let derived_block_id = BlockNumHash { number: 301, hash: B256::from([4u8; 32]) };
        let result = node.update_cross_safe(source_block_id, derived_block_id).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_invalidate_block_delegates_to_client() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(1).returning(|| Ok(ChainId::from(42u64)));
        client
            .expect_invalidate_block()
            .withf(|seal| seal.number == 400)
            .times(1)
            .returning(|_| Ok(()));

        let client = Arc::new(client);
        let db = Arc::new(MockDb::new());
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let block_seal = BlockSeal { number: 400, hash: B256::from([5u8; 32]), timestamp: 0 };
        let result = node.invalidate_block(block_seal).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_reset_calls_resetter() {
        let mut client = MockClient::new();
        client.expect_chain_id().times(2).returning(|| Ok(ChainId::from(42u64)));
        client.expect_reset_pre_interop().times(1).returning(|| Ok(()));

        let mut db = MockDb::new();
        db.expect_latest_derivation_state()
            .times(1)
            .returning(|| Err(StorageError::DatabaseNotInitialised));

        let client = Arc::new(client);
        let db = Arc::new(db);
        let asserter = Asserter::new();
        let transport = MockTransport::new(asserter.clone());
        let l1_provider = RootProvider::<Ethereum>::new(RpcClient::new(transport, false));
        let (tx, _rx) = mpsc::channel(10);
        let node = ManagedNode::new(client.clone(), db, l1_provider, tx);

        let result = node.reset().await;
        assert!(result.is_ok());
    }
}
