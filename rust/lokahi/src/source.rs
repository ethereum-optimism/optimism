//! The source node of one chain: where the verifier reads its local-safe blocks and their data.

use alloy_consensus::{Header, Sealed, TxReceipt};
use alloy_eips::{BlockId, BlockNumHash};
use alloy_primitives::{B256, Log, U64, map::HashMap};
use alloy_provider::{Provider, RootProvider};
use async_trait::async_trait;
use jsonrpsee::http_client::{HttpClient, HttpClientBuilder};
use kona_genesis::RollupConfig;
use kona_interop::{BlockWithReceipts, InitiatingLogs, InteropProvider};
use kona_rpc::{DecimalChainId, SuperRootApiClient};
use op_alloy_consensus::OpReceiptEnvelope;
use op_alloy_network::Optimism;
use std::{collections::BTreeMap, time::Duration};
use thiserror::Error;
use url::Url;

/// Timeout of each request to a source node.
const REQUEST_TIMEOUT: Duration = Duration::from_secs(5);

/// A failed read from a source node. Every one is retried: a source that is down or behind only
/// holds the verifier back.
#[derive(Debug, Error)]
pub(crate) enum SourceError {
    /// The consensus RPC failed.
    #[error("consensus RPC: {0}")]
    Consensus(#[from] jsonrpsee::core::ClientError),
    /// The execution RPC failed.
    #[error("execution RPC: {0}")]
    Execution(#[from] alloy_provider::transport::TransportError),
    /// The execution RPC has no such block.
    #[error("block {0} not found")]
    BlockNotFound(BlockId),
    /// A message names a chain the verifier has no source for.
    #[error("no source for chain {0}")]
    UnknownChain(u64),
    /// A message names a block above the frontier being verified, which is not local-safe.
    #[error("block {number} of chain {chain_id} is above the verified frontier {frontier:?}")]
    BeyondFrontier {
        /// The chain.
        chain_id: u64,
        /// The block named.
        number: u64,
        /// The chain's block at the timestamp being verified.
        frontier: BlockNumHash,
    },
}

/// A chain's local-safe block at a timestamp.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct LocalSafeAt {
    /// The block.
    pub block: BlockNumHash,
    /// The L1 block that made it local-safe, from the source's safe head database.
    pub l1_inclusion: BlockNumHash,
}

/// What a source reports for a timestamp.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct SourceProgress {
    /// The block at the timestamp, once it is local-safe.
    pub at_timestamp: Option<LocalSafeAt>,
    /// The timestamp of the source's local-safe head.
    pub local_safe_timestamp: u64,
}

/// The source node of one chain.
#[derive(Debug)]
pub(crate) struct ChainSource {
    pub rollup_config: RollupConfig,
    cl: HttpClient,
    el: RootProvider<Optimism>,
}

impl ChainSource {
    /// Connects to the source's consensus RPC, over HTTP, and its execution RPC, over HTTP or
    /// `WebSocket`.
    pub(crate) async fn connect(
        rollup_config: RollupConfig,
        cl_rpc: &Url,
        el_rpc: &Url,
    ) -> eyre::Result<Self> {
        let cl = HttpClientBuilder::default().request_timeout(REQUEST_TIMEOUT).build(cl_rpc)?;
        let el = RootProvider::connect(el_rpc.as_str()).await?;
        Ok(Self { rollup_config, cl, el })
    }

    pub(crate) const fn chain_id(&self) -> u64 {
        self.rollup_config.l2_chain_id.id()
    }

    /// Whether the chain produced a block at exactly `timestamp`, rather than carrying an older
    /// one over it.
    pub(crate) const fn has_block_at(&self, timestamp: u64) -> bool {
        let genesis = self.rollup_config.genesis.l2_time;
        timestamp >= genesis && (timestamp - genesis).is_multiple_of(self.rollup_config.block_time)
    }

    /// The source's local-safe block at `timestamp` and the L1 block it became safe at.
    ///
    /// `superroot_atTimestamp` answers from the local-safe head and the safe head database: the
    /// block is at or below local-safe, and its L1 block is the earliest at which the local-safe
    /// head reached it, which later records at the same L1 block cannot change.
    pub(crate) async fn progress_at(&self, timestamp: u64) -> Result<SourceProgress, SourceError> {
        let response = self.cl.superroot_at_timestamp(U64::from(timestamp)).await?;
        let at_timestamp = response
            .optimistic_at_timestamp
            .get(&DecimalChainId(self.chain_id()))
            .map(|output| LocalSafeAt {
                block: BlockNumHash {
                    number: self.rollup_config.block_number_from_timestamp(timestamp),
                    hash: output.output.block_hash,
                },
                l1_inclusion: output.required_l1,
            });
        Ok(SourceProgress {
            at_timestamp,
            local_safe_timestamp: response.current_local_safe_timestamp,
        })
    }

    /// The block with hash `hash` and its receipts.
    pub(crate) async fn block_with_receipts(
        &self,
        hash: B256,
    ) -> Result<BlockWithReceipts, SourceError> {
        let header = self.header_by_hash(hash).await?;
        let receipts = self
            .el
            .get_block_receipts(BlockId::hash(hash))
            .await?
            .ok_or(SourceError::BlockNotFound(BlockId::hash(hash)))?
            .into_iter()
            .map(|receipt| {
                OpReceiptEnvelope::from(
                    receipt.inner.inner.map_receipt(|receipt| receipt.map_logs(|log| log.inner)),
                )
            })
            .collect();
        Ok(BlockWithReceipts { header: Sealed::new_unchecked(header, hash), receipts })
    }

    async fn header_by_hash(&self, hash: B256) -> Result<Header, SourceError> {
        let block = self
            .el
            .get_block_by_hash(hash)
            .await?
            .ok_or(SourceError::BlockNotFound(BlockId::hash(hash)))?;
        Ok(block.header.inner)
    }

    async fn header_by_number(&self, number: u64) -> Result<Header, SourceError> {
        let block = self
            .el
            .get_block_by_number(number.into())
            .await?
            .ok_or(SourceError::BlockNotFound(BlockId::number(number)))?;
        Ok(block.header.inner)
    }

    async fn logs_by_hash(&self, hash: B256) -> Result<Vec<Log>, SourceError> {
        Ok(self
            .el
            .get_block_receipts(BlockId::hash(hash))
            .await?
            .ok_or(SourceError::BlockNotFound(BlockId::hash(hash)))?
            .into_iter()
            .flat_map(|receipt| {
                receipt.inner.inner.receipt.into_logs().into_iter().map(|log| log.inner)
            })
            .collect())
    }
}

/// Serves [`kona_interop::MessageGraph`] the initiating blocks it checks, from the chains' source
/// nodes. A block at a chain's frontier is read by its hash; one below it is the source's
/// canonical block at its number.
#[derive(Debug)]
pub(crate) struct SourceProvider<'a> {
    pub sources: &'a BTreeMap<u64, ChainSource>,
    /// Each chain's block at the timestamp being verified.
    pub frontier: &'a HashMap<u64, BlockNumHash>,
}

#[async_trait]
impl InteropProvider for SourceProvider<'_> {
    type Error = SourceError;

    async fn header_by_number(&self, chain_id: u64, number: u64) -> Result<Header, SourceError> {
        let source = self.sources.get(&chain_id).ok_or(SourceError::UnknownChain(chain_id))?;
        let frontier = *self.frontier.get(&chain_id).ok_or(SourceError::UnknownChain(chain_id))?;
        if number > frontier.number {
            return Err(SourceError::BeyondFrontier { chain_id, number, frontier });
        }
        if number == frontier.number {
            return source.header_by_hash(frontier.hash).await;
        }
        source.header_by_number(number).await
    }

    async fn initiating_logs(
        &self,
        chain_id: u64,
        header: &Header,
    ) -> Result<InitiatingLogs, SourceError> {
        let source = self.sources.get(&chain_id).ok_or(SourceError::UnknownChain(chain_id))?;
        Ok(InitiatingLogs::Receipts(source.logs_by_hash(header.hash_slow()).await?))
    }
}
