//! An Engine API Client.

use alloy_eips::{BlockId, eip1898::BlockNumberOrTag};
use alloy_network::Network;
use alloy_primitives::B256;
use alloy_provider::{EthGetBlock, Provider, RootProvider, ext::EngineApi};
use alloy_rpc_types_engine::{ExecutionPayloadV1, PayloadStatus};
use alloy_rpc_types_eth::Block;
use alloy_transport::{RpcError, TransportErrorKind, TransportResult};
use kona_genesis::RollupConfig;
use kona_protocol::FromBlockError;
use kona_providers_alloy::{AlloyChainProvider, AlloyChainProviderError};
use op_alloy_network::Optimism;
use op_alloy_rpc_types::Transaction;
use std::sync::Arc;
use thiserror::Error;

mod query;
pub use query::EngineQueryClient;

/// An error that occurred in the [`EngineClient`] or [`EngineQueryClient`].
#[derive(Error, Debug)]
pub enum EngineClientError {
    /// An RPC error occurred
    #[error("An RPC error occurred: {0}")]
    RpcError(#[from] RpcError<TransportErrorKind>),

    /// An error occurred while decoding the payload
    #[error("An error occurred while decoding the payload: {0}")]
    BlockInfoDecodeError(#[from] FromBlockError),

    /// No L2 block was found for the requested number or tag.
    #[error("No L2 block found for block number or tag: {0}")]
    NoL2BlockFound(BlockNumberOrTag),

    /// No L2 block was found for the requested hash.
    #[error("No L2 block found for block hash: {0}")]
    NoL2BlockFoundByHash(alloy_primitives::B256),

    /// The block has no withdrawals root while Isthmus is active.
    #[error("No block withdrawals root while Isthmus is active")]
    NoWithdrawalsRoot,
}
/// Client for L1 reads and L2 Engine API calls. Providers erase their transport type, so tests
/// use the same client with a mock transport. Task helpers select the Engine API version.
#[derive(Clone, Debug)]
pub struct EngineClient {
    engine: RootProvider<Optimism>,
    /// The node's shared L1 client.
    l1: AlloyChainProvider,
    cfg: Arc<RollupConfig>,
}

impl EngineClient {
    /// Creates a client from existing providers.
    pub const fn new(
        l1: AlloyChainProvider,
        engine: RootProvider<Optimism>,
        cfg: Arc<RollupConfig>,
    ) -> Self {
        Self { engine, l1, cfg }
    }

    /// Returns the rollup configuration.
    pub fn cfg(&self) -> &RollupConfig {
        &self.cfg
    }

    /// Returns the hash of the canonical L1 block at `number`, or `None` if the L1 chain does not
    /// reach that height yet.
    pub async fn l1_block_hash_by_number(
        &self,
        number: u64,
    ) -> Result<Option<B256>, AlloyChainProviderError> {
        match self.l1.block_info_by_number(number).await {
            Ok(block) => Ok(Some(block.hash)),
            Err(AlloyChainProviderError::BlockNotFound(_)) => Ok(None),
            Err(error) => Err(error),
        }
    }

    /// Fetches an L2 block.
    pub fn get_l2_block(
        &self,
        block: BlockId,
    ) -> EthGetBlock<<Optimism as Network>::BlockResponse> {
        self.engine.get_block(block)
    }

    /// Submits a Paris payload, whose method is provided by Alloy's Ethereum Engine API extension.
    pub async fn new_payload_v1(
        &self,
        payload: ExecutionPayloadV1,
    ) -> TransportResult<PayloadStatus> {
        self.engine.new_payload_v1(payload).await
    }

    /// Fetches an L2 block with full transactions.
    pub async fn l2_block_by_label(
        &self,
        numtag: BlockNumberOrTag,
    ) -> Result<Option<Block<Transaction>>, EngineClientError> {
        Ok(self.engine.get_block_by_number(numtag).full().await?)
    }

    /// Returns a restricted handle sharing this client's L2 connection.
    pub fn query_client(&self) -> EngineQueryClient {
        EngineQueryClient { cfg: self.cfg.clone(), engine: self.engine.clone() }
    }
}

/// Engine API calls use the L2 provider through OP Alloy's blanket implementation.
impl Provider<Optimism> for EngineClient {
    fn root(&self) -> &RootProvider<Optimism> {
        &self.engine
    }
}

#[cfg(test)]
mod request_log_tests {
    use super::*;
    use alloy_primitives::Bytes;
    use alloy_rpc_client::ClientBuilder;
    use kona_sources::rpc;
    use std::{io, sync::Mutex};

    #[derive(Clone, Default)]
    struct LogWriter(Arc<Mutex<Vec<u8>>>);

    impl io::Write for LogWriter {
        fn write(&mut self, bytes: &[u8]) -> io::Result<usize> {
            self.0.lock().unwrap().extend_from_slice(bytes);
            Ok(bytes.len())
        }

        fn flush(&mut self) -> io::Result<()> {
            Ok(())
        }
    }

    #[test]
    fn build_requests_log_once_with_task_context_at_info() {
        use crate::{
            BuildTask, ExecutableEngineTask,
            test_utils::{RpcMock, TestAttributesBuilder, TestEngineStateBuilder, test_block_info},
        };
        use alloy_rpc_types_engine::{ForkchoiceUpdated, PayloadId, PayloadStatusEnum};

        let logs = LogWriter::default();
        let writer = logs.clone();
        let subscriber = tracing_subscriber::fmt()
            .without_time()
            .with_ansi(false)
            .with_max_level(tracing::Level::INFO)
            .with_writer(move || writer.clone())
            .finish();
        let _guard = tracing::subscriber::set_default(subscriber);
        let parent = test_block_info(0);
        tokio::runtime::Builder::new_current_thread().enable_all().build().unwrap().block_on(
            async {
                for outcome in ["valid", "transport_error", "malformed_response"] {
                    let l1 = RpcMock::default();
                    let l2 = RpcMock::default();
                    let payload_id = PayloadId::new([1; 8]);
                    match outcome {
                        "transport_error" => {
                            l2.expect_transport_error("engine_forkchoiceUpdatedV3")
                        }
                        "malformed_response" => {
                            l2.expect("engine_forkchoiceUpdatedV3", Option::<()>::None)
                        }
                        _ => l2.expect(
                            "engine_forkchoiceUpdatedV3",
                            ForkchoiceUpdated {
                                payload_status: PayloadStatus::from_status(
                                    PayloadStatusEnum::Valid,
                                ),
                                payload_id: Some(payload_id),
                            },
                        ),
                    }
                    let provider = RootProvider::new(
                        ClientBuilder::default()
                            .layer(rpc::Observe::new("engine", tracing::Level::INFO))
                            .transport(l2.clone(), true),
                    );
                    let mut config = RollupConfig::default();
                    config.hardforks.ecotone_time = Some(0);
                    let config = Arc::new(config);
                    let engine = EngineClient::new(
                        AlloyChainProvider::new(
                            l1.provider(),
                            std::num::NonZeroUsize::new(8).unwrap(),
                            true,
                        ),
                        provider,
                        config.clone(),
                    );
                    let attributes = TestAttributesBuilder::new()
                        .with_parent(parent)
                        .with_timestamp(2)
                        .with_transactions(vec![Bytes::from_static(&[0x7e])])
                        .build();
                    let (tx, mut rx) = tokio::sync::mpsc::channel(1);
                    let mut state = TestEngineStateBuilder::new().with_unsafe_head(parent).build();
                    BuildTask::new(Arc::new(engine), config, attributes, tx)
                        .execute(&mut state)
                        .await
                        .unwrap();
                    let result = rx.recv().await.unwrap();
                    if outcome == "valid" {
                        assert_eq!(result.unwrap(), payload_id);
                    } else {
                        assert!(result.is_err());
                    }
                    l1.assert_finished();
                    l2.assert_finished();
                }
            },
        );
        let output = String::from_utf8(logs.0.lock().unwrap().clone()).unwrap();
        let requests: Vec<_> =
            output.lines().filter(|line| line.contains("JSON-RPC request completed")).collect();
        assert_eq!(requests.len(), 3, "{output}");
        for log in &requests {
            for field in [
                "build",
                "parent=",
                "unsafe_hash=",
                "safe_hash=",
                "finalized_hash=",
                "txs=1",
                "is_deposits=true",
                "duration=",
                "engine_forkchoiceUpdatedV3",
            ] {
                assert!(log.contains(field), "missing {field}: {log}");
            }
            assert!(log.contains(&parent.block_info.hash.to_string()), "{log}");
        }
        assert!(requests[0].contains("outcome=\"ok\""), "{}", requests[0]);
        assert!(requests[1].contains("injected transport failure"), "{}", requests[1]);
        assert!(requests[1].contains("WARN"), "{}", requests[1]);
        // The transport observes the JSON-RPC envelope; typed decoding belongs to the caller.
        assert!(requests[2].contains("outcome=\"ok\""), "{}", requests[2]);
        assert!(!output.contains("Failed to decode Engine RPC response"), "{output}");
        for old_message in [
            "block build started",
            "Forkchoice update with attributes successful",
            "Starting new build job",
            "Forkchoice update failed",
        ] {
            assert!(!output.contains(old_message), "duplicate caller log: {output}");
        }
    }
}

#[cfg(test)]
mod provider_tests {
    use super::*;
    use crate::test_utils::test_engine_client;
    use alloy_primitives::B256;
    use alloy_rpc_types_eth::EIP1186AccountProofResponse;
    use kona_protocol::{OutputRoot, Predeploys};
    use serde_json::json;

    #[tokio::test]
    async fn query_handle_shares_l2_connection_and_computes_output_roots() {
        for isthmus in [false, true] {
            let mut block = Block::<Transaction>::default();
            block.header.inner.state_root = B256::repeat_byte(1);
            block.header.inner.withdrawals_root = Some(B256::repeat_byte(2));
            block.header.hash = block.header.inner.hash_slow();
            let mut config = RollupConfig::default();
            config.genesis.l2.hash = block.header.hash;
            config.hardforks.isthmus_time = isthmus.then_some(0);
            let config = Arc::new(config);
            let (client, l1, l2) = test_engine_client(config);
            let query = client.query_client();
            l1.expect_params("eth_getBlockByNumber", json!(["0x4", false]), Option::<Block>::None);
            l2.expect_params(
                "eth_getBlockByNumber",
                json!(["latest", false]),
                Option::<Block<Transaction>>::None,
            );
            l2.expect_params("eth_getBlockByNumber", json!(["0x0", true]), &block);
            let storage_hash = if isthmus { B256::repeat_byte(2) } else { B256::repeat_byte(3) };
            if !isthmus {
                l2.expect_params(
                    "eth_getProof",
                    json!([Predeploys::L2_TO_L1_MESSAGE_PASSER, [], block.header.hash]),
                    EIP1186AccountProofResponse { storage_hash, ..Default::default() },
                );
            }
            assert!(client.l1_block_hash_by_number(4).await.unwrap().is_none());
            assert!(client.get_l2_block(BlockId::latest()).await.unwrap().is_none());
            let (info, root) = query.output_at_block(BlockNumberOrTag::Number(0)).await.unwrap();
            assert_eq!(info.block_info.hash, block.header.hash);
            assert_eq!(
                root.hash(),
                OutputRoot::from_parts(block.header.state_root, storage_hash, block.header.hash)
                    .hash()
            );
            l1.assert_finished();
            l2.assert_finished();
        }
    }
}
