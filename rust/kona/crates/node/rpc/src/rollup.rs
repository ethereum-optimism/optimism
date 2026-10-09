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
use kona_engine::EngineState;
use kona_genesis::RollupConfig;
use kona_protocol::{BlockInfo, FromBlockError, L2BlockInfo, OutputRoot, Predeploys, SyncStatus};
use op_alloy_network::Optimism;
use std::sync::Arc;
use thiserror::Error;
use tokio::sync::watch;

use crate::{DerivationStatus, RollupNodeApiServer};

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

/// `RollupRpc`
///
/// This is a server implementation of [`crate::RollupNodeApiServer`].
#[derive(Debug)]
pub struct RollupRpc<L2> {
    /// The application version.
    pub version: String,
    /// The rollup configuration.
    pub config: Arc<RollupConfig>,
    /// The engine state published by the engine task queue.
    pub engine_state: watch::Receiver<EngineState>,
    /// The read-only L2 output query provider.
    l2: L2,
    /// The latest published L1 head.
    pub head_l1: watch::Receiver<BlockInfo>,
    /// The latest published safe L1 head.
    pub safe_l1: watch::Receiver<BlockInfo>,
    /// The latest published finalized L1 head.
    pub finalized_l1: watch::Receiver<BlockInfo>,
    /// The progress published by the derivation actor.
    pub derivation_status: watch::Receiver<DerivationStatus>,
}

impl<L2> RollupRpc<L2> {
    /// Constructs a new [`RollupRpc`] from the application version, configuration, state, and
    /// clients.
    #[allow(clippy::too_many_arguments)]
    pub const fn new(
        version: String,
        config: Arc<RollupConfig>,
        engine_state: watch::Receiver<EngineState>,
        l2: L2,
        head_l1: watch::Receiver<BlockInfo>,
        safe_l1: watch::Receiver<BlockInfo>,
        finalized_l1: watch::Receiver<BlockInfo>,
        derivation_status: watch::Receiver<DerivationStatus>,
    ) -> Self {
        Self {
            version,
            config,
            engine_state,
            l2,
            head_l1,
            safe_l1,
            finalized_l1,
            derivation_status,
        }
    }

    fn sync_status(&self) -> RpcResult<SyncStatus> {
        // Do not serve a stale snapshot after a publisher exits.
        for receiver in [&self.head_l1, &self.safe_l1, &self.finalized_l1] {
            receiver.has_changed().map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;
        }
        self.derivation_status
            .has_changed()
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;
        let l2_sync_status = *self.engine_state.borrow();
        let head_l1 = *self.head_l1.borrow();
        let safe_l1 = *self.safe_l1.borrow();
        let finalized_l1 = *self.finalized_l1.borrow();
        let derivation_status = *self.derivation_status.borrow();

        // Zero-out the fields that can't be derived yet to follow op-node's behaviour.
        Ok(SyncStatus {
            current_l1: derivation_status.current_l1.unwrap_or_default(),
            current_l1_finalized: finalized_l1,
            head_l1,
            safe_l1,
            finalized_l1,
            unsafe_l2: l2_sync_status.sync_state.unsafe_head(),
            local_safe_l2: l2_sync_status.sync_state.local_safe_head(),
            safe_l2: l2_sync_status.sync_state.safe_head(),
            finalized_l2: l2_sync_status.sync_state.finalized_head(),
        })
    }
}

#[async_trait]
impl<L2: OutputProvider + 'static> RollupNodeApiServer for RollupRpc<L2> {
    async fn op_output_at_block(&self, block_num: BlockNumberOrTag) -> RpcResult<OutputResponse> {
        let (l2_block_info, output_root) =
            self.l2
                .output_at_block(block_num, &self.config)
                .await
                .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;
        let sync_status = self.sync_status()?;

        Ok(OutputResponse::from_v0(output_root, sync_status, l2_block_info))
    }

    async fn op_sync_status(&self) -> RpcResult<SyncStatus> {
        self.sync_status()
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
    use kona_engine::test_utils::{RpcMock, TestEngineStateBuilder, test_engine_client};
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use op_alloy_rpc_types::Transaction;
    use serde_json::json;

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
        let (_head_tx, head_rx) = watch::channel(head_l1);
        let (_safe_tx, safe_rx) = watch::channel(BlockInfo::default());
        let (_finalized_tx, finalized_rx) = watch::channel(BlockInfo::default());
        let (_derivation_tx, derivation_status) = watch::channel(DerivationStatus::default());
        let rpc = RollupRpc::new(
            "test".to_owned(),
            config.clone(),
            state_rx.clone(),
            TestOutputProvider { block, output: Ok((block_info, root)) },
            head_rx.clone(),
            safe_rx.clone(),
            finalized_rx.clone(),
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
            head_rx,
            safe_rx,
            finalized_rx,
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
        let (client, l1, l2) = test_engine_client(config.clone());
        let (state_tx, state_rx) = watch::channel(EngineState::default());
        let (head_tx, head_rx) = watch::channel(BlockInfo::default());
        let (safe_tx, safe_rx) = watch::channel(BlockInfo::default());
        let (finalized_tx, finalized_rx) = watch::channel(BlockInfo::default());
        let (derivation_tx, derivation_status) = watch::channel(DerivationStatus::default());
        let rpc = RollupRpc::new(
            "1.2.3-test".to_owned(),
            config.clone(),
            state_rx,
            client,
            head_rx,
            safe_rx,
            finalized_rx,
            derivation_status,
        );
        assert_eq!(rpc.op_rollup_config().await.unwrap(), *config);
        assert_eq!(rpc.op_sync_status().await.unwrap().current_l1, BlockInfo::default());

        let head = L2BlockInfo {
            block_info: BlockInfo { number: 17, ..Default::default() },
            ..Default::default()
        };
        state_tx.send_replace(TestEngineStateBuilder::new().with_unsafe_head(head).build());
        let head_l1 = BlockInfo { number: 42, ..Default::default() };
        let safe_l1 = BlockInfo { number: 40, ..Default::default() };
        let finalized_l1 = BlockInfo { number: 38, ..Default::default() };
        head_tx.send_replace(head_l1);
        safe_tx.send_replace(safe_l1);
        finalized_tx.send_replace(finalized_l1);
        let current_l1 = BlockInfo { number: 35, ..Default::default() };
        derivation_tx.send_replace(DerivationStatus { current_l1: Some(current_l1) });
        let status = rpc.op_sync_status().await.unwrap();
        assert_eq!(status.unsafe_l2, head);
        assert_eq!(status.current_l1, current_l1);
        assert_eq!(status.head_l1, head_l1);
        assert_eq!(status.safe_l1, safe_l1);
        assert_eq!(status.finalized_l1, finalized_l1);
        assert_eq!(status.current_l1_finalized, status.finalized_l1);

        let reorg = BlockInfo { number: 41, hash: B256::repeat_byte(1), ..Default::default() };
        head_tx.send_replace(reorg);
        let status = rpc.op_sync_status().await.unwrap();
        assert_eq!(status.head_l1, reorg);
        assert_eq!(status.safe_l1, safe_l1);
        assert_eq!(status.finalized_l1, finalized_l1);

        drop(derivation_tx);
        assert_eq!(rpc.op_sync_status().await.unwrap_err().code(), ErrorCode::InternalError.code());
        l1.assert_finished();
        l2.assert_finished();
    }

    #[tokio::test]
    async fn rejects_closed_block_publishers() {
        for closed in 0..3 {
            let (_, engine_state) = watch::channel(EngineState::default());
            let (head_tx, head_rx) = watch::channel(BlockInfo::default());
            let (safe_tx, safe_rx) = watch::channel(BlockInfo::default());
            let (finalized_tx, finalized_rx) = watch::channel(BlockInfo::default());
            let (_derivation_tx, derivation_status) = watch::channel(DerivationStatus::default());
            let rpc = RollupRpc::new(
                "test".to_owned(),
                Arc::new(RollupConfig::default()),
                engine_state,
                TestOutputProvider { block: BlockNumberOrTag::Number(0), output: Err("unused") },
                head_rx,
                safe_rx,
                finalized_rx,
                derivation_status,
            );
            let mut publishers = vec![head_tx, safe_tx, finalized_tx];
            assert!(rpc.op_sync_status().await.is_ok());
            drop(publishers.remove(closed));
            assert_eq!(
                rpc.op_sync_status().await.unwrap_err().code(),
                ErrorCode::InternalError.code()
            );
        }
    }

    #[tokio::test]
    async fn returns_configured_version() {
        let config = Arc::new(RollupConfig::default());
        let (client, l1, l2) = test_engine_client(config.clone());
        let (_, state_rx) = watch::channel(EngineState::default());
        let (_head_tx, head_rx) = watch::channel(BlockInfo::default());
        let (_safe_tx, safe_rx) = watch::channel(BlockInfo::default());
        let (_finalized_tx, finalized_rx) = watch::channel(BlockInfo::default());
        let (_derivation_tx, derivation_status) = watch::channel(DerivationStatus::default());

        for version in ["1.2.3-test", "0.0.0-dev"] {
            let rpc = RollupRpc::new(
                version.to_owned(),
                config.clone(),
                state_rx.clone(),
                client.clone(),
                head_rx.clone(),
                safe_rx.clone(),
                finalized_rx.clone(),
                derivation_status.clone(),
            );
            assert_eq!(rpc.op_version().await.unwrap(), version);
        }

        l1.assert_finished();
        l2.assert_finished();
    }
}
