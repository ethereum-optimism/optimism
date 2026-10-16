//! Implements the rollup client rpc endpoints. These endpoints serve data about the rollup state.
//!
//! Implemented in the op-node in <https://github.com/ethereum-optimism/optimism/blob/174e55f0a1e73b49b80a561fd3fedd4fea5770c6/op-service/sources/rollupclient.go#L16>

use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use alloy_provider::Provider;
use alloy_transport::TransportError;
use async_trait::async_trait;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_genesis::RollupConfig;
use kona_node_actors::{engine::EngineState, l1};
use kona_protocol::{BlockInfo, FromBlockError, L2BlockInfo, OutputRoot, Predeploys, SyncStatus};
use op_alloy_network::Optimism;
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::watch;

use crate::RollupNodeApiServer;

/// An [output response][or] for Optimism Rollup.
///
/// [or]: https://github.com/ethereum-optimism/optimism/blob/f20b92d3eb379355c876502c4f28e72a91ab902f/op-service/eth/output.go#L10-L17
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct OutputResponse {
    /// The output version.
    pub version: B256,
    /// The output root hash.
    pub output_root: B256,
    /// A reference to the L2 block.
    pub block_ref: L2BlockInfo,
    /// The withdrawal storage root.
    pub withdrawal_storage_root: B256,
    /// The state root.
    pub state_root: B256,
    /// The status of the node sync.
    pub sync_status: SyncStatus,
}

impl OutputResponse {
    /// Builds an [`OutputResponse`] from its parts.
    pub fn from_v0(v0: OutputRoot, sync_status: SyncStatus, block_ref: L2BlockInfo) -> Self {
        Self {
            version: v0.version(),
            output_root: v0.hash(),
            block_ref,
            withdrawal_storage_root: v0.bridge_storage_root,
            state_root: v0.state_root,
            sync_status,
        }
    }
}

/// An error encountered while reading or computing an L2 output.
#[derive(Error, Debug)]
pub enum OutputError {
    /// An RPC error occurred.
    #[error("An RPC error occurred: {0}")]
    RpcError(#[from] TransportError),

    /// An error occurred while decoding the payload.
    #[error("An error occurred while decoding the payload: {0}")]
    BlockInfoDecodeError(#[from] FromBlockError),

    /// No L2 block was found for the requested number or tag.
    #[error("No L2 block found for block number or tag: {0}")]
    NoL2BlockFound(BlockNumberOrTag),

    /// The block has no withdrawals root while Isthmus is active.
    #[error("No block withdrawals root while Isthmus is active")]
    NoWithdrawalsRoot,
}

/// The read-only L2 output query needed by [`RollupRpc`].
pub trait OutputProvider: Send + Sync {
    /// An error encountered while reading or computing an output.
    type Error;

    /// Reads the L2 block and computes its output root using the configured genesis and fork rules.
    fn output_at_block(
        &self,
        block: BlockNumberOrTag,
        config: &RollupConfig,
    ) -> impl Future<Output = Result<(L2BlockInfo, OutputRoot), Self::Error>> + Send;
}

impl<P: Provider<Optimism>> OutputProvider for P {
    type Error = OutputError;

    async fn output_at_block(
        &self,
        block: BlockNumberOrTag,
        config: &RollupConfig,
    ) -> Result<(L2BlockInfo, OutputRoot), Self::Error> {
        let output_block = self.get_block_by_number(block).full().await?;
        let output_block = output_block.ok_or(OutputError::NoL2BlockFound(block))?;
        // Decode the block info from the fetched block rather than making another request.
        let consensus_block = output_block.clone().into_consensus();
        let output_block_info = L2BlockInfo::from_block_and_genesis(
            &consensus_block.map_transactions(|tx| tx.inner.inner.into_inner()),
            &config.genesis,
        )?;

        let message_passer_storage_root = if config.is_isthmus_active(output_block.header.timestamp)
        {
            output_block.header.withdrawals_root.ok_or(OutputError::NoWithdrawalsRoot)?
        } else {
            self.get_proof(Predeploys::L2_TO_L1_MESSAGE_PASSER, Default::default())
                .block_id(block.into())
                .await?
                .storage_hash
        };

        let output_root = OutputRoot::from_parts(
            output_block.header.state_root,
            message_passer_storage_root,
            output_block.header.hash,
        );
        Ok((output_block_info, output_root))
    }
}

/// The latest L1 blocks the node observed, as [`RollupRpc`] reads them.
pub trait L1Status: Send + Sync {
    /// Returns the latest L1 blocks, or `None` once they are no longer tracked.
    fn l1_status(&self) -> Option<l1::Status>;
}

impl L1Status for l1::Handle {
    fn l1_status(&self) -> Option<l1::Status> {
        self.status().ok()
    }
}

/// Derivation's progress, as [`RollupRpc`] reads it.
pub trait Derivation: Send + Sync {
    /// Returns the L1 block derivation has read up to, which may not be fully derived yet.
    fn current_l1(&self) -> BlockInfo;
}

/// `RollupRpc`
///
/// This is a server implementation of [`crate::RollupNodeApiServer`].
#[derive(Debug)]
pub struct RollupRpc<L2, L1, D> {
    /// The application version.
    pub version: String,
    /// The rollup configuration.
    pub config: Arc<RollupConfig>,
    /// The engine state published by the engine actor.
    pub engine_state: watch::Receiver<EngineState>,
    /// The read-only L2 output query provider.
    l2: L2,
    /// The latest L1 blocks.
    l1: L1,
    /// Derivation, whose progress the sync status reports.
    derivation: D,
}

impl<L2, L1: L1Status, D: Derivation> RollupRpc<L2, L1, D> {
    /// Constructs a new [`RollupRpc`] from the application version, configuration, state, and
    /// clients.
    pub const fn new(
        version: String,
        config: Arc<RollupConfig>,
        engine_state: watch::Receiver<EngineState>,
        l2: L2,
        l1: L1,
        derivation: D,
    ) -> Self {
        Self { version, config, engine_state, l2, l1, derivation }
    }

    async fn sync_status(&self) -> RpcResult<SyncStatus> {
        let l1::Status { head: head_l1, safe: safe_l1, finalized: finalized_l1 } =
            self.l1.l1_status().ok_or_else(|| ErrorObject::from(ErrorCode::InternalError))?;
        let l2_sync_status = *self.engine_state.borrow();

        // Zero-out the fields that can't be derived yet to follow op-node's behaviour.
        Ok(SyncStatus {
            current_l1: self.derivation.current_l1(),
            current_l1_finalized: finalized_l1,
            head_l1,
            safe_l1,
            finalized_l1,
            unsafe_l2: l2_sync_status.unsafe_head(),
            local_safe_l2: l2_sync_status.local_safe_head(),
            safe_l2: l2_sync_status.safe_head(),
            finalized_l2: l2_sync_status.finalized_head(),
        })
    }
}

#[async_trait]
impl<L2: OutputProvider + 'static, L1: L1Status + 'static, D: Derivation + 'static>
    RollupNodeApiServer for RollupRpc<L2, L1, D>
{
    async fn op_output_at_block(&self, block_num: BlockNumberOrTag) -> RpcResult<OutputResponse> {
        let (l2_block_info, output_root) =
            self.l2
                .output_at_block(block_num, &self.config)
                .await
                .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;
        let sync_status = self.sync_status().await?;

        Ok(OutputResponse::from_v0(output_root, sync_status, l2_block_info))
    }

    async fn op_sync_status(&self) -> RpcResult<SyncStatus> {
        self.sync_status().await
    }

    async fn op_rollup_config(&self) -> RpcResult<RollupConfig> {
        Ok((*self.config).clone())
    }

    async fn op_version(&self) -> RpcResult<String> {
        Ok(self.version.clone())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;
    use alloy_rpc_types_eth::{Block, EIP1186AccountProofResponse};
    use kona_node_actors::engine::HeadsUpdate;
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use kona_sources::test_utils::RpcMock;
    use op_alloy_rpc_types::Transaction;
    use serde_json::json;

    impl L1Status for watch::Receiver<l1::Status> {
        fn l1_status(&self) -> Option<l1::Status> {
            self.has_changed().ok()?;
            Some(*self.borrow())
        }
    }

    impl Derivation for watch::Receiver<BlockInfo> {
        fn current_l1(&self) -> BlockInfo {
            *self.borrow()
        }
    }

    struct TestOutputProvider {
        block: BlockNumberOrTag,
        output: Result<(L2BlockInfo, OutputRoot), &'static str>,
    }

    impl OutputProvider for TestOutputProvider {
        type Error = &'static str;

        async fn output_at_block(
            &self,
            block: BlockNumberOrTag,
            _config: &RollupConfig,
        ) -> Result<(L2BlockInfo, OutputRoot), Self::Error> {
            assert_eq!(block, self.block);
            self.output
        }
    }

    #[tokio::test]
    async fn computes_output_roots_using_an_optimism_provider() {
        for isthmus in [false, true] {
            let mut block = Block::<Transaction>::default();
            block.header.inner.state_root = B256::repeat_byte(1);
            block.header.inner.withdrawals_root = Some(B256::repeat_byte(2));
            block.header.hash = block.header.inner.hash_slow();
            let mut config = RollupConfig::default();
            config.genesis.l2.hash = block.header.hash;
            config.hardforks.isthmus_time = isthmus.then_some(0);
            let mock = RpcMock::default();
            let provider = mock.provider::<Optimism>();
            mock.expect_params("eth_getBlockByNumber", json!(["0x0", true]), &block);
            let storage_hash = if isthmus { B256::repeat_byte(2) } else { B256::repeat_byte(3) };
            if !isthmus {
                mock.expect_params(
                    "eth_getProof",
                    json!([Predeploys::L2_TO_L1_MESSAGE_PASSER, [], "0x0"]),
                    EIP1186AccountProofResponse { storage_hash, ..Default::default() },
                );
            }
            let (info, root) =
                provider.output_at_block(BlockNumberOrTag::Number(0), &config).await.unwrap();
            assert_eq!(info.block_info.hash, block.header.hash);
            assert_eq!(
                root.hash(),
                OutputRoot::from_parts(block.header.state_root, storage_hash, block.header.hash)
                    .hash()
            );
            mock.assert_finished();
        }
    }

    #[tokio::test]
    async fn queries_outputs_through_a_single_method_provider() {
        let block = BlockNumberOrTag::Number(17);
        let block_info = L2BlockInfo {
            block_info: BlockInfo { number: 17, hash: B256::repeat_byte(1), ..Default::default() },
            ..Default::default()
        };
        let root = OutputRoot::from_parts(
            B256::repeat_byte(2),
            B256::repeat_byte(3),
            block_info.block_info.hash,
        );
        let config = Arc::new(RollupConfig::default());
        let (_, state_rx) = watch::channel(EngineState::default());
        let head_l1 = BlockInfo { number: 42, ..Default::default() };
        let (_l1_tx, l1_rx) = watch::channel(l1::Status { head: head_l1, ..Default::default() });
        let (_derivation_tx, derivation_status) = watch::channel(BlockInfo::default());
        let rpc = RollupRpc::new(
            "test".to_owned(),
            config.clone(),
            state_rx.clone(),
            TestOutputProvider { block, output: Ok((block_info, root)) },
            l1_rx.clone(),
            derivation_status.clone(),
        );
        let response = rpc.op_output_at_block(block).await;
        let response = response.unwrap();
        assert_eq!(response.block_ref, block_info);
        assert_eq!(response.output_root, root.hash());
        assert_eq!(response.state_root, root.state_root);
        assert_eq!(response.withdrawal_storage_root, root.bridge_storage_root);
        assert_eq!(response.sync_status.head_l1, head_l1);

        let rpc = RollupRpc::new(
            "test".to_owned(),
            config,
            state_rx,
            TestOutputProvider { block, output: Err("output unavailable") },
            l1_rx,
            derivation_status,
        );
        assert_eq!(
            rpc.op_output_at_block(block).await.unwrap_err().code(),
            ErrorCode::InternalError.code()
        );
    }

    #[tokio::test]
    async fn reads_config_and_published_actor_state() {
        let config = Arc::new(RollupConfig { block_time: 11, ..Default::default() });
        let l2 = RpcMock::default();
        let client = l2.provider::<Optimism>();
        let (state_tx, state_rx) = watch::channel(EngineState::default());
        let (l1_tx, l1_rx) = watch::channel(l1::Status::default());
        let (derivation_tx, derivation_status) = watch::channel(BlockInfo::default());
        let rpc = RollupRpc::new(
            "1.2.3-test".to_owned(),
            config.clone(),
            state_rx,
            client,
            l1_rx,
            derivation_status,
        );
        assert_eq!(rpc.op_rollup_config().await.unwrap(), *config);
        assert_eq!(rpc.op_sync_status().await.unwrap().current_l1, BlockInfo::default());

        let head = L2BlockInfo {
            block_info: BlockInfo { number: 17, ..Default::default() },
            ..Default::default()
        };
        state_tx.send_replace(EngineState::default().apply_update(HeadsUpdate {
            unsafe_head: Some(head),
            local_safe_head: Some(head),
            safe_head: Some(head),
            finalized_head: Some(head),
        }));
        let head_l1 = BlockInfo { number: 42, ..Default::default() };
        let safe_l1 = BlockInfo { number: 40, ..Default::default() };
        let finalized_l1 = BlockInfo { number: 38, ..Default::default() };
        l1_tx.send_replace(l1::Status { head: head_l1, safe: safe_l1, finalized: finalized_l1 });
        let current_l1 = BlockInfo { number: 35, ..Default::default() };
        derivation_tx.send_replace(current_l1);
        let status = rpc.op_sync_status().await.unwrap();
        assert_eq!(status.unsafe_l2, head);
        assert_eq!(status.current_l1, current_l1);
        assert_eq!(status.head_l1, head_l1);
        assert_eq!(status.safe_l1, safe_l1);
        assert_eq!(status.finalized_l1, finalized_l1);
        assert_eq!(status.current_l1_finalized, status.finalized_l1);

        let reorg = BlockInfo { number: 41, hash: B256::repeat_byte(1), ..Default::default() };
        l1_tx.send_modify(|status| status.head = reorg);
        let status = rpc.op_sync_status().await.unwrap();
        assert_eq!(status.head_l1, reorg);
        assert_eq!(status.safe_l1, safe_l1);
        assert_eq!(status.finalized_l1, finalized_l1);

        drop(l1_tx);
        assert_eq!(rpc.op_sync_status().await.unwrap_err().code(), ErrorCode::InternalError.code());
        l2.assert_finished();
    }

    #[tokio::test]
    async fn rejects_a_closed_l1_publisher() {
        let (_, engine_state) = watch::channel(EngineState::default());
        let (l1_tx, l1_rx) = watch::channel(l1::Status::default());
        let (_derivation_tx, derivation_status) = watch::channel(BlockInfo::default());
        let rpc = RollupRpc::new(
            "test".to_owned(),
            Arc::new(RollupConfig::default()),
            engine_state,
            TestOutputProvider { block: BlockNumberOrTag::Number(0), output: Err("unused") },
            l1_rx,
            derivation_status,
        );
        assert!(rpc.op_sync_status().await.is_ok());
        drop(l1_tx);
        assert_eq!(rpc.op_sync_status().await.unwrap_err().code(), ErrorCode::InternalError.code());
    }

    #[tokio::test]
    async fn returns_configured_version() {
        let config = Arc::new(RollupConfig::default());
        let l2 = RpcMock::default();
        let client = l2.provider::<Optimism>();
        let (_, state_rx) = watch::channel(EngineState::default());
        let (_l1_tx, l1_rx) = watch::channel(l1::Status::default());
        let (_derivation_tx, derivation_status) = watch::channel(BlockInfo::default());

        for version in ["1.2.3-test", "0.0.0-dev"] {
            let rpc = RollupRpc::new(
                version.to_owned(),
                config.clone(),
                state_rx.clone(),
                client.clone(),
                l1_rx.clone(),
                derivation_status.clone(),
            );
            assert_eq!(rpc.op_version().await.unwrap(), version);
        }

        l2.assert_finished();
    }
}
