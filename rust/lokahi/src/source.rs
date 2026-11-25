//! The source node of one chain: where the verifier reads its local-safe blocks and their data.

use crate::{state::Verified, store::StoreError};
use alloy_consensus::{Header, Sealed, TxReceipt};
use alloy_eips::{BlockId, BlockNumHash};
use alloy_primitives::{B256, Log, U64, map::HashMap};
use alloy_provider::{Provider, RootProvider};
use async_trait::async_trait;
use jsonrpsee::{
    core::client::ClientT,
    http_client::{HttpClient, HttpClientBuilder},
    rpc_params,
};
use kona_genesis::RollupConfig;
use kona_interop::{BlockWithReceipts, InitiatingLogs, InteropProvider};
use kona_protocol::{OutputRoot, SyncStatus};
use kona_rpc::{DecimalChainId, SuperRootApiClient};
use op_alloy_consensus::OpReceiptEnvelope;
use op_alloy_network::Optimism;
use std::{collections::BTreeMap, sync::RwLock, time::Duration};
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
    /// The execution RPC answered a block other than the one asked for by hash.
    #[error("asked for block {requested}, got {actual}")]
    WrongBlock {
        /// The hash asked for.
        requested: B256,
        /// The hash of the header answered.
        actual: B256,
    },
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
    /// A block below the frontier that the verifier has no verified result for.
    #[error("block {number} of chain {chain_id} is below the frontier but not verified")]
    Unverified {
        /// The chain.
        chain_id: u64,
        /// The block named.
        number: u64,
    },
    /// The verifier's store failed.
    #[error(transparent)]
    Store(#[from] StoreError),
    /// The consensus RPC answered a sync status that does not decode.
    #[error("sync status: {0}")]
    SyncStatus(#[from] serde_json::Error),
}

/// The source of each chain, by chain ID.
pub(crate) type Sources = BTreeMap<u64, ChainSource>;

/// The sources of `sources` by chain ID.
pub(crate) fn by_chain(sources: Vec<ChainSource>) -> Sources {
    sources.into_iter().map(|source| (source.chain_id(), source)).collect()
}

/// A chain's local-safe block at a timestamp.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct LocalSafeAt {
    /// The block.
    pub block: BlockNumHash,
    /// The L1 block that made it local-safe, from the source's safe head database.
    pub l1_inclusion: BlockNumHash,
    /// The block's output root preimage.
    pub output: OutputRoot,
}

/// What a source reports for a timestamp.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(crate) struct SourceProgress {
    /// The block at the timestamp, once it is local-safe.
    pub at_timestamp: Option<LocalSafeAt>,
    /// The L1 block the source's derivation is at: it derived every L1 block strictly below it.
    pub current_l1: BlockNumHash,
    /// The timestamp of the source's safe head.
    pub safe_timestamp: u64,
    /// The timestamp of the source's local-safe head.
    pub local_safe_timestamp: u64,
    /// The timestamp of the source's finalized head.
    pub finalized_timestamp: u64,
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

    /// The timestamp of block `number`.
    pub(crate) const fn block_timestamp(&self, number: u64) -> u64 {
        let genesis = &self.rollup_config.genesis;
        genesis.l2_time +
            number.saturating_sub(genesis.l2.number).saturating_mul(self.rollup_config.block_time)
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
                output: OutputRoot::from_parts(
                    output.output.state_root,
                    output.output.message_passer_storage_root,
                    output.output.block_hash,
                ),
            });
        Ok(SourceProgress {
            at_timestamp,
            current_l1: response.current_l1,
            safe_timestamp: response.current_safe_timestamp,
            local_safe_timestamp: response.current_local_safe_timestamp,
            finalized_timestamp: response.current_finalized_timestamp,
        })
    }

    /// The source's `optimism_syncStatus`, as it answered it and decoded.
    pub(crate) async fn sync_status(&self) -> Result<(serde_json::Value, SyncStatus), SourceError> {
        let raw: serde_json::Value = self.cl.request("optimism_syncStatus", rpc_params![]).await?;
        let status = serde_json::from_value(raw.clone())?;
        Ok((raw, status))
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

    /// The header of the block with hash `hash`, checked to hash to it.
    async fn header_by_hash(&self, hash: B256) -> Result<Header, SourceError> {
        let block = self
            .el
            .get_block_by_hash(hash)
            .await?
            .ok_or(SourceError::BlockNotFound(BlockId::hash(hash)))?;
        let actual = block.header.inner.hash_slow();
        if actual != hash {
            return Err(SourceError::WrongBlock { requested: hash, actual });
        }
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
/// nodes, by hash: a block at a chain's frontier by the frontier's hash, one below it by the hash
/// the verifier verified at its height. A block before activation, which the verifier verified no
/// timestamp of, is the source's canonical block at its number.
#[derive(Debug)]
pub(crate) struct SourceProvider<'a> {
    pub sources: &'a Sources,
    /// Each chain's block at the timestamp being verified.
    pub frontier: &'a HashMap<u64, BlockNumHash>,
    /// The verifier's state, with the blocks verified below the frontier.
    pub verified: &'a RwLock<Verified>,
}

impl SourceProvider<'_> {
    /// The hash of block `number` of `chain_id` in the verified result at its timestamp, or
    /// `None` for a block before activation.
    fn verified_hash(
        &self,
        source: &ChainSource,
        chain_id: u64,
        number: u64,
    ) -> Result<Option<B256>, SourceError> {
        let timestamp = source.block_timestamp(number);
        let verified = self.verified.read().expect("state lock");
        if timestamp < verified.activation_timestamp() {
            return Ok(None);
        }
        let block = verified
            .result_at(timestamp)?
            .and_then(|result| result.chains.get(&chain_id).map(|chain| chain.block))
            .filter(|block| block.number == number)
            .ok_or(SourceError::Unverified { chain_id, number })?;
        Ok(Some(block.hash))
    }
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
        match self.verified_hash(source, chain_id, number)? {
            Some(hash) => source.header_by_hash(hash).await,
            None => source.header_by_number(number).await,
        }
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::state::{
        VerifiedResult,
        tests::{chain_result, open_state},
    };
    use alloy_rpc_types_eth::{Block, BlockTransactions, Header as RpcHeader};
    use jsonrpsee::{
        RpcModule,
        server::{Server, ServerHandle},
    };
    use kona_genesis::ChainGenesis;

    const CHAIN: u64 = 901;
    const GENESIS_TIME: u64 = 1000;
    /// Later than genesis, so that blocks 0 and 1 are before activation.
    const ACTIVATION: u64 = 1004;

    fn header(number: u64, extra: &'static [u8]) -> Header {
        Header {
            number,
            timestamp: GENESIS_TIME + 2 * number,
            extra_data: extra.into(),
            ..Default::default()
        }
    }

    /// The block lokahi verified at height 3.
    fn verified() -> Header {
        header(3, b"verified")
    }

    /// The source's canonical block at height 3 and below: not the verified one.
    fn canonical(number: u64) -> Header {
        header(number, b"canonical")
    }

    /// An execution RPC answering `known` blocks by hash and [`canonical`] blocks by number.
    async fn serve_execution(known: Vec<Header>) -> (Url, ServerHandle) {
        let json = |header: Header| {
            serde_json::to_value(Block::<()>::new(
                RpcHeader::new(header),
                BlockTransactions::Hashes(vec![]),
            ))
            .unwrap()
        };
        let mut module = RpcModule::new(());
        module
            .register_method("eth_getBlockByHash", move |params, _, _| {
                let (hash, _full): (B256, bool) = params.parse().unwrap();
                known
                    .iter()
                    .find(|header| header.hash_slow() == hash)
                    .map_or(serde_json::Value::Null, |header| json(header.clone()))
            })
            .unwrap();
        module
            .register_method("eth_getBlockByNumber", move |params, _, _| {
                let (number, _full): (U64, bool) = params.parse().unwrap();
                json(canonical(number.to()))
            })
            .unwrap();
        let server = Server::builder().build("127.0.0.1:0").await.unwrap();
        let url = format!("http://{}", server.local_addr().unwrap()).parse().unwrap();
        (url, server.start(module))
    }

    async fn source(el: &Url) -> ChainSource {
        let mut rollup_config = RollupConfig {
            l2_chain_id: CHAIN.into(),
            block_time: 2,
            genesis: ChainGenesis { l2_time: GENESIS_TIME, ..Default::default() },
            ..Default::default()
        };
        rollup_config.hardforks.lagoon_time = Some(0);
        ChainSource::connect(rollup_config, &"http://127.0.0.1:1".parse().unwrap(), el)
            .await
            .unwrap()
    }

    /// The state with every timestamp from activation to the one of block 4 verified, block 3
    /// being [`verified`].
    fn state(dir: &std::path::Path) -> RwLock<Verified> {
        let mut state = open_state(dir, ACTIVATION, BTreeMap::from([(CHAIN, GENESIS_TIME)]));
        for timestamp in ACTIVATION..GENESIS_TIME + 8 {
            let number = (timestamp - GENESIS_TIME) / 2;
            let hash = if number == 3 { verified().hash_slow() } else { B256::with_last_byte(1) };
            let result = VerifiedResult {
                l1_inclusion: BlockNumHash::default(),
                chains: BTreeMap::from([(CHAIN, chain_result(BlockNumHash { number, hash }))]),
            };
            state.commit(timestamp, result).unwrap();
        }
        RwLock::new(state)
    }

    async fn header_below_frontier(known: Vec<Header>, number: u64) -> Result<Header, SourceError> {
        let dir = tempfile::tempdir().unwrap();
        let (el, _server) = serve_execution(known).await;
        let sources = BTreeMap::from([(CHAIN, source(&el).await)]);
        let frontier = HashMap::from_iter([(
            CHAIN,
            BlockNumHash { number: 4, hash: B256::with_last_byte(4) },
        )]);
        let verified = state(dir.path());
        let provider =
            SourceProvider { sources: &sources, frontier: &frontier, verified: &verified };
        provider.header_by_number(CHAIN, number).await
    }

    /// A block below the frontier is the one verified at its height, not the source's canonical
    /// block there.
    #[tokio::test]
    async fn a_block_below_the_frontier_is_read_by_its_verified_hash() {
        let header = header_below_frontier(vec![verified(), canonical(3)], 3).await.unwrap();

        assert_eq!(header.hash_slow(), verified().hash_slow());
    }

    /// A source without the verified block fails the read: it does not fall back to its
    /// canonical block.
    #[tokio::test]
    async fn a_source_without_the_verified_block_fails_the_read() {
        let err = header_below_frontier(vec![canonical(3)], 3).await.unwrap_err();

        assert!(matches!(err, SourceError::BlockNotFound(_)), "got {err:?}");
    }

    /// A block before activation has no verified hash: it is the source's canonical block.
    #[tokio::test]
    async fn a_block_before_activation_is_read_by_number() {
        let header = header_below_frontier(vec![verified()], 1).await.unwrap();

        assert_eq!(header.hash_slow(), canonical(1).hash_slow());
    }
}
