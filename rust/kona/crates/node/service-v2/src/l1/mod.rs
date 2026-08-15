//! Shared L1 access service.
//!
//! The service is the single owner of L1 execution and beacon queries used by node workflows. It
//! publishes canonical labels while each consumer retains an independent processing cursor.

mod stream;

use alloy_consensus::{Header, Receipt, TxEnvelope};
use alloy_eips::{BlockId, BlockNumberOrTag, eip4844::Blob};
use alloy_primitives::{Address, B256};
use alloy_provider::{Provider, RootProvider};
use async_trait::async_trait;
use futures::StreamExt;
use kona_derive::{BlobProvider, ChainProvider, PipelineError, PipelineErrorKind};
use kona_genesis::{RollupConfig, SystemConfigLog, SystemConfigUpdate, UnsafeBlockSignerUpdate};
use kona_protocol::BlockInfo;
use kona_providers_alloy::{AlloyChainProvider, OnlineBeaconClient, OnlineBlobProvider};
use kona_rpc::{L1State, L1WatcherQueries, L1WatcherQuerySender};
use std::{sync::Arc, time::Duration};
use thiserror::Error;
use tokio::sync::{mpsc, oneshot, watch};
use tokio_util::sync::CancellationToken;

const HEAD_POLL_INTERVAL: Duration = Duration::from_secs(4);
const FINALIZED_POLL_INTERVAL: Duration = Duration::from_secs(60);
const REQUEST_CAPACITY: usize = 64;

/// Cloneable access to L1 labels and query operations.
#[derive(Debug, Clone)]
pub struct L1Client {
    head: watch::Receiver<Option<BlockInfo>>,
    finalized: watch::Receiver<Option<BlockInfo>>,
    request_tx: mpsc::Sender<L1Request>,
    query_tx: L1WatcherQuerySender,
}

impl L1Client {
    /// Returns an independent receiver for canonical L1 head updates.
    pub fn subscribe_head(&self) -> watch::Receiver<Option<BlockInfo>> {
        self.head.clone()
    }

    /// Returns an independent receiver for finalized L1 updates.
    pub fn subscribe_finalized(&self) -> watch::Receiver<Option<BlockInfo>> {
        self.finalized.clone()
    }

    /// Returns the query sender consumed by the rollup RPC implementation.
    pub fn query_sender(&self) -> L1WatcherQuerySender {
        self.query_tx.clone()
    }

    /// Looks up an L1 block by hash without treating absence as an error.
    pub async fn block_by_hash(&self, hash: B256) -> Result<Option<BlockInfo>, PipelineErrorKind> {
        self.request(|response| L1Request::OptionalBlockByHash { hash, response }).await
    }

    /// Looks up an L1 block by number without treating absence as an error.
    pub async fn block_by_number(
        &self,
        number: u64,
    ) -> Result<Option<BlockInfo>, PipelineErrorKind> {
        self.request(|response| L1Request::OptionalBlockByNumber { number, response }).await
    }

    async fn request<T>(
        &self,
        request: impl FnOnce(oneshot::Sender<Result<T, PipelineErrorKind>>) -> L1Request,
    ) -> Result<T, PipelineErrorKind> {
        let (response, result) = oneshot::channel();
        self.request_tx.send(request(response)).await.map_err(|_| unavailable())?;
        result.await.map_err(|_| unavailable())?
    }
}

#[async_trait]
impl ChainProvider for L1Client {
    type Error = PipelineErrorKind;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, Self::Error> {
        self.request(|response| L1Request::Header { hash, response }).await
    }

    async fn block_info_by_number(&mut self, number: u64) -> Result<BlockInfo, Self::Error> {
        self.request(|response| L1Request::BlockInfo { number, response }).await
    }

    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<Receipt>, Self::Error> {
        self.request(|response| L1Request::Receipts { hash, response }).await
    }

    async fn block_info_and_transactions_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), Self::Error> {
        self.request(|response| L1Request::BlockAndTransactions { hash, response }).await
    }
}

#[async_trait]
impl BlobProvider for L1Client {
    type Error = PipelineErrorKind;

    async fn get_and_validate_blobs(
        &mut self,
        block_ref: &BlockInfo,
        blob_hashes: &[B256],
    ) -> Result<Vec<Box<Blob>>, Self::Error> {
        self.request(|response| L1Request::Blobs {
            block_ref: *block_ref,
            blob_hashes: blob_hashes.to_vec(),
            response,
        })
        .await
    }
}

#[derive(Debug)]
enum L1Request {
    OptionalBlockByHash {
        hash: B256,
        response: oneshot::Sender<Result<Option<BlockInfo>, PipelineErrorKind>>,
    },
    OptionalBlockByNumber {
        number: u64,
        response: oneshot::Sender<Result<Option<BlockInfo>, PipelineErrorKind>>,
    },
    Header {
        hash: B256,
        response: oneshot::Sender<Result<Header, PipelineErrorKind>>,
    },
    BlockInfo {
        number: u64,
        response: oneshot::Sender<Result<BlockInfo, PipelineErrorKind>>,
    },
    Receipts {
        hash: B256,
        response: oneshot::Sender<Result<Vec<Receipt>, PipelineErrorKind>>,
    },
    BlockAndTransactions {
        hash: B256,
        response: oneshot::Sender<Result<(BlockInfo, Vec<TxEnvelope>), PipelineErrorKind>>,
    },
    Blobs {
        block_ref: BlockInfo,
        blob_hashes: Vec<B256>,
        response: oneshot::Sender<Result<Vec<Box<Blob>>, PipelineErrorKind>>,
    },
}

/// Long-running owner of L1 polling, queries, and system-config signer updates.
#[derive(Debug)]
pub struct L1Service {
    config: Arc<RollupConfig>,
    provider: RootProvider,
    chain_provider: AlloyChainProvider,
    blob_provider: OnlineBlobProvider<OnlineBeaconClient>,
    signer_tx: mpsc::Sender<Address>,
    head_tx: watch::Sender<Option<BlockInfo>>,
    finalized_tx: watch::Sender<Option<BlockInfo>>,
    request_rx: mpsc::Receiver<L1Request>,
    query_rx: mpsc::Receiver<L1WatcherQueries>,
}

impl L1Service {
    /// Creates the service and its cloneable client.
    pub fn new(
        config: Arc<RollupConfig>,
        provider: RootProvider,
        chain_provider: AlloyChainProvider,
        blob_provider: OnlineBlobProvider<OnlineBeaconClient>,
        signer_tx: mpsc::Sender<Address>,
    ) -> (Self, L1Client) {
        let (head_tx, head) = watch::channel(None);
        let (finalized_tx, finalized) = watch::channel(None);
        let (request_tx, request_rx) = mpsc::channel(REQUEST_CAPACITY);
        let (query_tx, query_rx) = mpsc::channel(REQUEST_CAPACITY);
        (
            Self {
                config,
                provider,
                chain_provider,
                blob_provider,
                signer_tx,
                head_tx,
                finalized_tx,
                request_rx,
                query_rx,
            },
            L1Client { head, finalized, request_tx, query_tx },
        )
    }

    /// Polls L1 labels and serializes shared L1 queries until shutdown.
    pub async fn run(mut self, shutdown: CancellationToken) -> Result<(), L1ServiceError> {
        let mut heads = stream::block_stream(
            self.provider.clone(),
            BlockNumberOrTag::Latest,
            HEAD_POLL_INTERVAL,
        );
        let mut finalized = stream::block_stream(
            self.provider.clone(),
            BlockNumberOrTag::Finalized,
            FINALIZED_POLL_INTERVAL,
        );

        loop {
            tokio::select! {
                biased;
                _ = shutdown.cancelled() => return Ok(()),
                request = self.request_rx.recv() => {
                    let request = request.ok_or(L1ServiceError::RequestChannelClosed)?;
                    self.handle_request(request).await;
                }
                head = heads.next() => {
                    let head = head.ok_or(L1ServiceError::HeadStreamEnded)?;
                    self.process_head(head).await?;
                }
                finalized = finalized.next() => {
                    let finalized = finalized.ok_or(L1ServiceError::FinalizedStreamEnded)?;
                    self.finalized_tx.send_replace(Some(finalized));
                }
                query = self.query_rx.recv() => {
                    let query = query.ok_or(L1ServiceError::QueryChannelClosed)?;
                    self.handle_rpc_query(query).await;
                }
            }
        }
    }

    async fn handle_request(&mut self, request: L1Request) {
        match request {
            L1Request::OptionalBlockByHash { hash, response } => {
                let result = self
                    .provider
                    .get_block_by_hash(hash)
                    .await
                    .map(|block| block.map(|block| block.into_consensus().into()))
                    .map_err(provider_error);
                let _ = response.send(result);
            }
            L1Request::OptionalBlockByNumber { number, response } => {
                let result = self
                    .provider
                    .get_block_by_number(number.into())
                    .await
                    .map(|block| block.map(|block| block.into_consensus().into()))
                    .map_err(provider_error);
                let _ = response.send(result);
            }
            L1Request::Header { hash, response } => {
                let result = self.chain_provider.header_by_hash(hash).await.map_err(Into::into);
                let _ = response.send(result);
            }
            L1Request::BlockInfo { number, response } => {
                let result =
                    self.chain_provider.block_info_by_number(number).await.map_err(Into::into);
                let _ = response.send(result);
            }
            L1Request::Receipts { hash, response } => {
                let result = self.chain_provider.receipts_by_hash(hash).await.map_err(Into::into);
                let _ = response.send(result);
            }
            L1Request::BlockAndTransactions { hash, response } => {
                let result = self
                    .chain_provider
                    .block_info_and_transactions_by_hash(hash)
                    .await
                    .map_err(Into::into);
                let _ = response.send(result);
            }
            L1Request::Blobs { block_ref, blob_hashes, response } => {
                let result = self
                    .blob_provider
                    .get_and_validate_blobs(&block_ref, &blob_hashes)
                    .await
                    .map_err(Into::into);
                let _ = response.send(result);
            }
        }
    }

    async fn process_head(&self, head: BlockInfo) -> Result<(), L1ServiceError> {
        self.head_tx.send_replace(Some(head));

        let logs = self
            .provider
            .get_logs(
                &alloy_rpc_types_eth::Filter::new()
                    .address(self.config.l1_system_config_address)
                    .select(head.hash),
            )
            .await?;
        let ecotone_active = self.config.is_ecotone_active(head.timestamp);
        for log in logs {
            let update = SystemConfigLog::new(log.into(), ecotone_active).build();
            if let Ok(SystemConfigUpdate::UnsafeBlockSigner(UnsafeBlockSignerUpdate {
                unsafe_block_signer,
            })) = update
            {
                info!(target: "l1", %unsafe_block_signer, "Unsafe block signer updated");
                self.signer_tx
                    .send(unsafe_block_signer)
                    .await
                    .map_err(|_| L1ServiceError::SignerChannelClosed)?;
            }
        }
        Ok(())
    }

    async fn handle_rpc_query(&self, query: L1WatcherQueries) {
        match query {
            L1WatcherQueries::Config(response) => {
                let _ = response.send((*self.config).clone());
            }
            L1WatcherQueries::L1State(response) => {
                let head_l1 = self.block_info(BlockId::latest()).await;
                let finalized_l1 = self.block_info(BlockId::finalized()).await;
                let safe_l1 = self.block_info(BlockId::safe()).await;
                let _ = response.send(L1State {
                    current_l1: *self.head_tx.borrow(),
                    current_l1_finalized: finalized_l1,
                    head_l1,
                    safe_l1,
                    finalized_l1,
                });
            }
        }
    }

    async fn block_info(&self, id: BlockId) -> Option<BlockInfo> {
        match self.provider.get_block(id).await {
            Ok(block) => block.map(|block| block.into_consensus().into()),
            Err(error) => {
                warn!(target: "l1", ?id, ?error, "Failed to query L1 label");
                None
            }
        }
    }
}

fn unavailable() -> PipelineErrorKind {
    PipelineError::Provider("shared L1 service is unavailable".to_string()).temp()
}

fn provider_error(error: alloy_transport::TransportError) -> PipelineErrorKind {
    PipelineError::Provider(error.to_string()).temp()
}

/// Terminal L1 service failure.
#[derive(Debug, Error)]
pub enum L1ServiceError {
    /// L1 transport failed.
    #[error("L1 transport failed: {0}")]
    Transport(#[from] alloy_transport::TransportError),
    /// Canonical head polling ended unexpectedly.
    #[error("canonical L1 head stream ended")]
    HeadStreamEnded,
    /// Finalized head polling ended unexpectedly.
    #[error("finalized L1 head stream ended")]
    FinalizedStreamEnded,
    /// Every shared-query client was dropped.
    #[error("L1 request channel closed")]
    RequestChannelClosed,
    /// The RPC query channel closed while the service was running.
    #[error("L1 RPC query channel closed")]
    QueryChannelClosed,
    /// The network service stopped accepting signer updates.
    #[error("unsafe block signer channel closed")]
    SignerChannelClosed,
}
