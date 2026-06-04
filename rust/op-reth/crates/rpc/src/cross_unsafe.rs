//! Runtime cross-unsafe head RPC support.
//!
//! This is an intentionally simplified runtime gate. It validates the existence and basic
//! consistency of initiating messages referenced by a block's executing messages, but it does not
//! implement the full supervisor dependency graph (no cycle/hazard analysis, dependency-set
//! activation, or cross-chain frontier semantics).
//!
//! The cache holds locally validated blocks above the safe head. Each refresh re-anchors to the
//! current safe head, drops any cached block that is no longer canonical locally, drops any cached
//! block whose source dependencies are no longer canonical on their source chain, and then walks
//! forward toward local latest validating new blocks.
//!
//! Concurrency: a single mutex protects the cache and is held for the duration of a refresh,
//! including the source-chain RPC calls it makes. Each source RPC is bounded by
//! [`SOURCE_RPC_TIMEOUT`], so a slow source cannot stall the endpoint indefinitely, and the
//! background poller keeps the cache warm so user calls usually have little work to do.

use alloy_consensus::{BlockHeader, TxReceipt};
use alloy_eips::BlockHashOrNumber;
use alloy_primitives::{B256, U64, keccak256};
use alloy_rpc_client::{ClientBuilder, RpcClient};
use alloy_rpc_types_eth::{Block as RpcBlock, Filter, Log as RpcLog};
use jsonrpsee::proc_macros::rpc;
use jsonrpsee_core::{RpcResult, async_trait};
use kona_interop::{MESSAGE_EXPIRY_WINDOW, RawMessagePayload, parse_log_to_executing_message};
use reth_provider::BlockReaderIdExt;
use reth_rpc_server_types::result::internal_rpc_err;
use serde::{Deserialize, Serialize};
use std::{collections::BTreeMap, sync::Arc, time::Duration};
use tokio::{
    sync::{Mutex, OnceCell},
    time::{sleep, timeout},
};
use tracing::{debug, warn};

const CROSS_UNSAFE_HEAD_POLL_INTERVAL: Duration = Duration::from_secs(5);
/// Per-request timeout for every source-chain RPC call.
const SOURCE_RPC_TIMEOUT: Duration = Duration::from_secs(10);

/// A block head returned by `eth_crossUnsafeHead`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct CrossUnsafeHead {
    /// Block number.
    pub number: U64,
    /// Block hash.
    pub hash: B256,
}

#[cfg_attr(not(test), rpc(server, namespace = "eth"))]
#[cfg_attr(test, rpc(server, client, namespace = "eth"))]
pub trait CrossUnsafeHeadApi {
    /// Returns the latest runtime-validated cross-unsafe head.
    #[method(name = "crossUnsafeHead")]
    async fn cross_unsafe_head(&self) -> RpcResult<CrossUnsafeHead>;
}

/// RPC extension that computes a simplified cross-unsafe head on demand.
#[derive(Debug, Clone)]
pub struct CrossUnsafeHeadExt<Provider> {
    inner: Arc<CrossUnsafeHeadExtInner<Provider>>,
}

#[derive(Debug)]
struct CrossUnsafeHeadExtInner<Provider> {
    provider: Provider,
    source_clients: SourceLogClients,
    state: Mutex<CrossUnsafeState>,
}

impl<Provider> CrossUnsafeHeadExt<Provider> {
    /// Creates a new runtime cross-unsafe head extension.
    pub fn new(
        provider: Provider,
        source_rpcs: impl IntoIterator<Item = impl Into<String>>,
    ) -> Result<Self, String> {
        Ok(Self {
            inner: Arc::new(CrossUnsafeHeadExtInner {
                provider,
                source_clients: SourceLogClients::new(source_rpcs)?,
                state: Mutex::new(CrossUnsafeState::default()),
            }),
        })
    }
}

#[async_trait]
impl<Provider> CrossUnsafeHeadApiServer for CrossUnsafeHeadExt<Provider>
where
    Provider: BlockReaderIdExt + Clone + Send + Sync + 'static,
{
    async fn cross_unsafe_head(&self) -> RpcResult<CrossUnsafeHead> {
        self.compute_cross_unsafe_head().await
    }
}

impl<Provider> CrossUnsafeHeadExt<Provider>
where
    Provider: BlockReaderIdExt + Clone + Send + Sync + 'static,
{
    /// Continuously warms the runtime cross-unsafe head cache.
    pub async fn run_auto_poller(self) {
        loop {
            if let Err(err) = self.compute_cross_unsafe_head().await {
                debug!(target: "rpc::cross_unsafe", %err, "failed to prefetch cross-unsafe head");
            }
            sleep(CROSS_UNSAFE_HEAD_POLL_INTERVAL).await;
        }
    }

    async fn compute_cross_unsafe_head(&self) -> RpcResult<CrossUnsafeHead> {
        let safe = self.safe_anchor()?;
        let latest_number = self
            .inner
            .provider
            .latest_header()
            .map_err(|err| internal_rpc_err(format!("failed to read latest header: {err}")))?
            .ok_or_else(|| internal_rpc_err("latest header not found"))?
            .number();

        let mut state = self.inner.state.lock().await;
        state.reseed_safe(safe);
        self.rewind_to_canonical(&mut state)?;
        self.rewind_to_canonical_sources(&mut state).await;

        let mut expected_parent = state.head().hash;
        for number in state.head().number.saturating_add(1)..=latest_number {
            let header = self
                .inner
                .provider
                .sealed_header(number)
                .map_err(|err| internal_rpc_err(format!("failed to read header {number}: {err}")))?
                .ok_or_else(|| internal_rpc_err(format!("header {number} not found")))?;

            let hash = header.hash();
            let parent_hash = header.parent_hash();
            if parent_hash != expected_parent {
                debug!(
                    target: "rpc::cross_unsafe",
                    number,
                    %hash,
                    %parent_hash,
                    expected = %expected_parent,
                    "stopping cross-unsafe walk at non-contiguous block"
                );
                state.truncate_from(number);
                break;
            }

            if state.is_validated(number, hash, parent_hash) {
                expected_parent = hash;
                continue;
            }

            let validation = self.validate_block(number, header.timestamp()).await?;
            if !validation.valid {
                debug!(
                    target: "rpc::cross_unsafe",
                    number,
                    %hash,
                    "stopping cross-unsafe walk at unvalidated block"
                );
                state.truncate_from(number);
                break;
            }

            state.insert(CachedBlock { number, hash, parent_hash, sources: validation.sources });
            expected_parent = hash;
        }

        Ok(state.head().into())
    }

    /// Drops cached blocks from the top down until the head matches the local canonical chain.
    fn rewind_to_canonical(&self, state: &mut CrossUnsafeState) -> RpcResult<()> {
        while let Some((number, cached_hash)) =
            state.validated.iter().next_back().map(|(number, block)| (*number, block.hash))
        {
            let canonical = self
                .inner
                .provider
                .sealed_header(number)
                .map_err(|err| {
                    internal_rpc_err(format!("failed to read cached head header {number}: {err}"))
                })?
                .map(|header| header.hash());

            if canonical == Some(cached_hash) {
                break;
            }
            state.validated.remove(&number);
        }

        Ok(())
    }

    /// Drops cached blocks whose source dependencies are no longer canonical on their source chain.
    ///
    /// Each distinct source block referenced by the cache is rechecked once. A source block's hash
    /// commits to its logs, so an unchanged hash means the prior validation still holds. If a
    /// source block changed, every cached block at or above the earliest local block that
    /// referenced it is dropped.
    async fn rewind_to_canonical_sources(&self, state: &mut CrossUnsafeState) {
        let mut distinct: BTreeMap<(u64, u64), (B256, u64)> = BTreeMap::new();
        for (&local, block) in &state.validated {
            for source in &block.sources {
                distinct
                    .entry((source.chain_id, source.block_number))
                    .and_modify(|(_, min_local)| *min_local = (*min_local).min(local))
                    .or_insert((source.block_hash, local));
            }
        }

        let mut rewind_from: Option<u64> = None;
        for ((chain_id, block_number), (block_hash, min_local)) in distinct {
            // Only rewind on a *confirmed* mismatch (the source block is still reachable but its
            // hash changed → reorg). A transient source-RPC failure returns `None` and leaves the
            // cached blocks in place, so a momentary outage cannot regress the head backwards; the
            // next refresh re-checks once the source is reachable again.
            if self
                .inner
                .source_clients
                .source_block_canonical(chain_id, block_number, block_hash)
                .await ==
                Some(false)
            {
                rewind_from = Some(rewind_from.map_or(min_local, |current| current.min(min_local)));
            }
        }

        if let Some(number) = rewind_from {
            debug!(
                target: "rpc::cross_unsafe",
                number,
                "rewinding cross-unsafe head because a source block is no longer canonical"
            );
            state.truncate_from(number);
        }
    }

    fn safe_anchor(&self) -> RpcResult<CachedBlock> {
        if let Some(safe) = self
            .inner
            .provider
            .safe_block_num_hash()
            .map_err(|err| internal_rpc_err(format!("failed to read safe block: {err}")))?
        {
            return Ok(CachedBlock {
                number: safe.number,
                hash: safe.hash,
                parent_hash: B256::ZERO,
                sources: Vec::new(),
            });
        }

        let genesis = self
            .inner
            .provider
            .sealed_header(0)
            .map_err(|err| internal_rpc_err(format!("failed to read genesis header: {err}")))?
            .ok_or_else(|| internal_rpc_err("safe block unavailable and genesis not found"))?;

        Ok(CachedBlock {
            number: 0,
            hash: genesis.hash(),
            parent_hash: genesis.parent_hash(),
            sources: Vec::new(),
        })
    }

    async fn validate_block(
        &self,
        number: u64,
        executing_timestamp: u64,
    ) -> RpcResult<BlockValidation> {
        let receipts = self
            .inner
            .provider
            .receipts_by_block(BlockHashOrNumber::Number(number))
            .map_err(|err| {
                internal_rpc_err(format!("failed to read receipts for block {number}: {err}"))
            })?
            .ok_or_else(|| internal_rpc_err(format!("receipts for block {number} not found")))?;

        let mut sources: Vec<SourceRef> = Vec::new();
        for receipt in receipts {
            for log in receipt.logs() {
                let Some(message) = parse_log_to_executing_message(log) else { continue };

                let initiating_timestamp = message.identifier.timestamp.saturating_to::<u64>();
                // NOTE: this uses the default `MESSAGE_EXPIRY_WINDOW`. The canonical validation
                // path (`kona_interop::MessageGraph`) takes the window as a parameter and honors a
                // dependency set's `override_message_expiry_window`. This simplified gate does not
                // load the dependency set, so on a chain configured with a non-default expiry the
                // boundary here can diverge from consensus. Acceptable for the default config;
                // revisit if per-chain expiry overrides need to be tracked.
                if !initiating_timestamp_in_window(
                    initiating_timestamp,
                    executing_timestamp,
                    MESSAGE_EXPIRY_WINDOW,
                ) {
                    debug!(
                        target: "rpc::cross_unsafe",
                        number,
                        initiating_timestamp,
                        executing_timestamp,
                        message_expiry_window = MESSAGE_EXPIRY_WINDOW,
                        "executing message initiating timestamp is outside the valid window (future or expired)"
                    );
                    return Ok(BlockValidation::invalid());
                }

                let source = self
                    .inner
                    .source_clients
                    .validate_initiating_message(
                        message.identifier.chainId.saturating_to::<u64>(),
                        message.identifier.blockNumber.saturating_to::<u64>(),
                        message.identifier.logIndex.saturating_to::<u64>(),
                        initiating_timestamp,
                        message.identifier.origin,
                        message.payloadHash,
                    )
                    .await;

                let Some(source) = source else { return Ok(BlockValidation::invalid()) };
                if !sources.contains(&source) {
                    sources.push(source);
                }
            }
        }

        Ok(BlockValidation { valid: true, sources })
    }
}

#[derive(Debug)]
struct BlockValidation {
    valid: bool,
    sources: Vec<SourceRef>,
}

impl BlockValidation {
    const fn invalid() -> Self {
        Self { valid: false, sources: Vec::new() }
    }
}

/// Whether an executing message's initiating timestamp is acceptable for a block with the given
/// executing timestamp: not in the future, and not older than `expiry_window` seconds.
const fn initiating_timestamp_in_window(
    initiating_timestamp: u64,
    executing_timestamp: u64,
    expiry_window: u64,
) -> bool {
    initiating_timestamp <= executing_timestamp &&
        initiating_timestamp >= executing_timestamp.saturating_sub(expiry_window)
}

#[derive(Debug)]
struct SourceLogClients {
    clients: Vec<SourceLogClient>,
}

impl SourceLogClients {
    fn new(source_rpcs: impl IntoIterator<Item = impl Into<String>>) -> Result<Self, String> {
        let mut clients: Vec<SourceLogClient> = Vec::new();
        for source_rpc in source_rpcs {
            let endpoint = source_rpc.into();
            if clients.iter().any(|client| client.endpoint == endpoint) {
                return Err(format!("duplicate source RPC URL {endpoint:?}"));
            }
            clients.push(SourceLogClient::new(endpoint)?);
        }

        Ok(Self { clients })
    }

    /// Finds the configured source client serving `chain_id`, resolving each client's chain ID
    /// lazily via `eth_chainId`. A client whose RPC is not reachable yet resolves to `None` and is
    /// simply skipped, so a peer EL that has not started cannot match (and cannot deadlock
    /// startup); it will match on a later refresh once it is up.
    async fn client_for(&self, chain_id: u64) -> Option<&SourceLogClient> {
        for client in &self.clients {
            if client.resolve_chain_id().await == Some(chain_id) {
                return Some(client);
            }
        }
        None
    }

    async fn validate_initiating_message(
        &self,
        chain_id: u64,
        block_number: u64,
        log_index: u64,
        timestamp: u64,
        origin: alloy_primitives::Address,
        payload_hash: B256,
    ) -> Option<SourceRef> {
        let Some(client) = self.client_for(chain_id).await else {
            warn!(
                target: "rpc::cross_unsafe",
                chain_id,
                block_number,
                log_index,
                "no source RPC available for initiating message chain"
            );
            return None;
        };

        let block_hash = client
            .validate_initiating_message(block_number, log_index, timestamp, origin, payload_hash)
            .await?;
        Some(SourceRef { chain_id, block_number, block_hash })
    }

    /// Returns whether the cached source block is still canonical:
    /// - `Some(true)`  — confirmed canonical (hash matches),
    /// - `Some(false)` — confirmed reorged (the block is reachable but its hash changed),
    /// - `None`        — could not determine (source RPC unreachable / block not yet available), in
    ///   which case callers must NOT treat it as a reorg.
    async fn source_block_canonical(
        &self,
        chain_id: u64,
        block_number: u64,
        block_hash: B256,
    ) -> Option<bool> {
        let Some(client) = self.client_for(chain_id).await else {
            warn!(
                target: "rpc::cross_unsafe",
                chain_id,
                block_number,
                "no source RPC available for cached source block"
            );
            return None;
        };

        Some(client.block_hash(block_number).await? == block_hash)
    }
}

#[derive(Debug)]
struct SourceLogClient {
    endpoint: String,
    client: RpcClient,
    /// The source chain's ID, discovered once via `eth_chainId` and cached.
    chain_id: OnceCell<u64>,
}

impl SourceLogClient {
    fn new(endpoint: String) -> Result<Self, String> {
        let url = endpoint
            .parse()
            .map_err(|err| format!("invalid cross unsafe head source RPC URL: {err}"))?;
        let http = reqwest::Client::builder()
            .timeout(SOURCE_RPC_TIMEOUT)
            .build()
            .map_err(|err| format!("failed to build cross unsafe head source RPC client: {err}"))?;
        let client = ClientBuilder::default().http_with_client(http, url);
        Ok(Self { endpoint, client, chain_id: OnceCell::new() })
    }

    /// Resolves and caches this source's chain ID via `eth_chainId`.
    ///
    /// Returns `None` when the RPC is not reachable yet, so a peer EL that has not started cannot
    /// deadlock op-reth startup (no synchronous request is made when the clients are constructed).
    /// The result is cached on first success; failures are not cached, so the next refresh retries.
    async fn resolve_chain_id(&self) -> Option<u64> {
        self.chain_id
            .get_or_try_init(|| async {
                self.fetch_chain_id().await.map(|id| id.saturating_to::<u64>()).ok_or(())
            })
            .await
            .ok()
            .copied()
    }

    /// Validates an initiating message against this source chain, returning the source block hash
    /// on success (so the caller can record the dependency).
    async fn validate_initiating_message(
        &self,
        block_number: u64,
        log_index: u64,
        timestamp: u64,
        origin: alloy_primitives::Address,
        payload_hash: B256,
    ) -> Option<B256> {
        let block = self.fetch_block(block_number).await?;
        if block.header.timestamp() != timestamp {
            debug!(
                target: "rpc::cross_unsafe",
                endpoint = %self.endpoint,
                block_number,
                expected = timestamp,
                actual = block.header.timestamp(),
                "source block timestamp mismatch"
            );
            return None;
        }

        // Filter by the claimed origin address so a source block with very many logs cannot push
        // the target log past the source RPC's `eth_getLogs` response cap (which would otherwise
        // make a valid block look invalid). The block-global `log_index` is preserved by the RPC,
        // so matching it against the filtered subset is still correct.
        let logs = self.fetch_logs(block.header.hash, origin).await?;
        let Some(log) = logs.into_iter().find(|log| log.log_index == Some(log_index)) else {
            debug!(
                target: "rpc::cross_unsafe",
                endpoint = %self.endpoint,
                block_number,
                log_index,
                "source log not found"
            );
            return None;
        };

        if log.address() != origin {
            debug!(
                target: "rpc::cross_unsafe",
                endpoint = %self.endpoint,
                block_number,
                log_index,
                expected = %origin,
                actual = %log.address(),
                "source log origin mismatch"
            );
            return None;
        }

        let remote_payload_hash = keccak256(RawMessagePayload::from(&log.inner).as_ref());
        if remote_payload_hash != payload_hash {
            debug!(
                target: "rpc::cross_unsafe",
                endpoint = %self.endpoint,
                block_number,
                log_index,
                expected = %payload_hash,
                actual = %remote_payload_hash,
                "source log payload hash mismatch"
            );
            return None;
        }

        Some(block.header.hash)
    }

    /// Returns the canonical hash of `block_number` on the source chain, if available.
    async fn block_hash(&self, block_number: u64) -> Option<B256> {
        self.fetch_block(block_number).await.map(|block| block.header.hash)
    }

    async fn fetch_chain_id(&self) -> Option<U64> {
        match timeout(SOURCE_RPC_TIMEOUT, self.client.request_noparams::<U64>("eth_chainId")).await
        {
            Ok(Ok(chain_id)) => Some(chain_id),
            Ok(Err(err)) => {
                warn!(
                    target: "rpc::cross_unsafe",
                    %err,
                    endpoint = %self.endpoint,
                    "failed to fetch source chain ID"
                );
                None
            }
            Err(_) => {
                warn!(
                    target: "rpc::cross_unsafe",
                    endpoint = %self.endpoint,
                    "timed out fetching source chain ID"
                );
                None
            }
        }
    }

    async fn fetch_logs(
        &self,
        block_hash: B256,
        origin: alloy_primitives::Address,
    ) -> Option<Vec<RpcLog>> {
        let filter = Filter::new().at_block_hash(block_hash).address(origin);
        match timeout(
            SOURCE_RPC_TIMEOUT,
            self.client.request::<_, Vec<RpcLog>>("eth_getLogs", (filter,)),
        )
        .await
        {
            Ok(Ok(logs)) => Some(logs),
            Ok(Err(err)) => {
                warn!(
                    target: "rpc::cross_unsafe",
                    %err,
                    endpoint = %self.endpoint,
                    %block_hash,
                    "failed to fetch source logs"
                );
                None
            }
            Err(_) => {
                warn!(
                    target: "rpc::cross_unsafe",
                    endpoint = %self.endpoint,
                    %block_hash,
                    "timed out fetching source logs"
                );
                None
            }
        }
    }

    async fn fetch_block(&self, block_number: u64) -> Option<RpcBlock> {
        match timeout(
            SOURCE_RPC_TIMEOUT,
            self.client.request::<_, Option<RpcBlock>>(
                "eth_getBlockByNumber",
                (U64::from(block_number), false),
            ),
        )
        .await
        {
            Ok(Ok(Some(block))) => Some(block),
            Ok(Ok(None)) => {
                debug!(
                    target: "rpc::cross_unsafe",
                    endpoint = %self.endpoint,
                    block_number,
                    "source block not found"
                );
                None
            }
            Ok(Err(err)) => {
                warn!(
                    target: "rpc::cross_unsafe",
                    %err,
                    endpoint = %self.endpoint,
                    block_number,
                    "failed to fetch source block"
                );
                None
            }
            Err(_) => {
                warn!(
                    target: "rpc::cross_unsafe",
                    endpoint = %self.endpoint,
                    block_number,
                    "timed out fetching source block"
                );
                None
            }
        }
    }
}

/// A source-chain block that a cached cross-unsafe block depends on.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct SourceRef {
    chain_id: u64,
    block_number: u64,
    block_hash: B256,
}

/// In-memory cache of locally validated cross-unsafe blocks.
///
/// `safe` is the floor anchor; `validated` holds the contiguous chain of validated blocks strictly
/// above it. The head is always the highest validated block, or `safe` when none are cached.
#[derive(Debug, Default)]
struct CrossUnsafeState {
    safe: CachedBlock,
    validated: BTreeMap<u64, CachedBlock>,
}

impl CrossUnsafeState {
    fn head(&self) -> &CachedBlock {
        self.validated.values().next_back().unwrap_or(&self.safe)
    }

    /// Re-anchors to the current safe head, dropping any cached block at or below it.
    fn reseed_safe(&mut self, safe: CachedBlock) {
        self.validated = self.validated.split_off(&safe.number.saturating_add(1));
        self.safe = safe;
    }

    fn is_validated(&self, number: u64, hash: B256, parent_hash: B256) -> bool {
        self.validated
            .get(&number)
            .is_some_and(|block| block.hash == hash && block.parent_hash == parent_hash)
    }

    fn insert(&mut self, block: CachedBlock) {
        self.validated.insert(block.number, block);
    }

    /// Drops every cached block at or above `number`.
    fn truncate_from(&mut self, number: u64) {
        self.validated.split_off(&number);
    }
}

#[derive(Debug, Clone, Default)]
struct CachedBlock {
    number: u64,
    hash: B256,
    parent_hash: B256,
    /// Distinct source blocks the executing messages in this block depend on.
    sources: Vec<SourceRef>,
}

impl From<&CachedBlock> for CrossUnsafeHead {
    fn from(value: &CachedBlock) -> Self {
        Self { number: U64::from(value.number), hash: value.hash }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn block(number: u64, tag: u8, parent: u8) -> CachedBlock {
        CachedBlock {
            number,
            hash: B256::repeat_byte(tag),
            parent_hash: B256::repeat_byte(parent),
            sources: Vec::new(),
        }
    }

    #[test]
    fn initiating_timestamp_window_boundaries() {
        let exec = 1_000_000u64;
        let window = MESSAGE_EXPIRY_WINDOW;

        // Same timestamp is valid; one second in the future is not.
        assert!(initiating_timestamp_in_window(exec, exec, window));
        assert!(!initiating_timestamp_in_window(exec + 1, exec, window));

        // The oldest allowed timestamp is exactly `exec - window`; one older is expired.
        assert!(initiating_timestamp_in_window(exec - window, exec, window));
        assert!(!initiating_timestamp_in_window(exec - window - 1, exec, window));
        assert!(initiating_timestamp_in_window(exec - 1, exec, window));

        // The lower bound saturates: when the executing timestamp is below the window, any
        // non-future initiating timestamp (down to 0) is in window.
        assert!(initiating_timestamp_in_window(0, 100, window));
        assert!(!initiating_timestamp_in_window(101, 100, window));
    }

    #[test]
    fn builds_source_clients_from_urls() {
        let clients = SourceLogClients::new([
            "http://chain-a:8545".to_string(),
            "http://chain-b:8545".to_string(),
        ])
        .unwrap();

        assert_eq!(clients.clients.len(), 2);
    }

    #[test]
    fn rejects_invalid_or_duplicate_source_rpc_urls() {
        // Malformed URL.
        assert!(SourceLogClients::new(["not a url".to_string()]).is_err());
        // Duplicate URLs are rejected (chain IDs are not yet known at construction time).
        assert!(
            SourceLogClients::new([
                "http://chain-a:8545".to_string(),
                "http://chain-a:8545".to_string(),
            ])
            .is_err()
        );
    }

    #[test]
    fn head_defaults_to_safe_anchor() {
        let mut state = CrossUnsafeState::default();
        state.reseed_safe(block(5, 5, 0));
        assert_eq!(state.head().number, 5);
        assert_eq!(state.head().hash, B256::repeat_byte(5));
    }

    #[test]
    fn insert_advances_head_and_truncate_rewinds_to_safe() {
        let mut state = CrossUnsafeState::default();
        state.reseed_safe(block(5, 5, 0));
        state.insert(block(6, 6, 5));
        state.insert(block(7, 7, 6));
        assert_eq!(state.head().number, 7);

        state.truncate_from(7);
        assert_eq!(state.head().number, 6);

        state.truncate_from(6);
        assert_eq!(state.head().number, 5);
        assert_eq!(state.head().hash, B256::repeat_byte(5));
    }

    #[test]
    fn reseed_safe_drops_entries_at_or_below_safe() {
        let mut state = CrossUnsafeState::default();
        state.reseed_safe(block(5, 5, 0));
        state.insert(block(6, 6, 5));
        state.insert(block(7, 7, 6));

        state.reseed_safe(block(6, 60, 0));
        assert!(!state.validated.contains_key(&6));
        assert_eq!(state.safe.number, 6);
        assert_eq!(state.head().number, 7);

        // Re-anchoring above the cached head leaves only the safe anchor.
        state.reseed_safe(block(9, 90, 0));
        assert!(state.validated.is_empty());
        assert_eq!(state.head().number, 9);
    }

    #[test]
    fn is_validated_requires_hash_and_parent_match() {
        let mut state = CrossUnsafeState::default();
        state.reseed_safe(block(5, 5, 0));
        state.insert(block(6, 6, 5));

        assert!(state.is_validated(6, B256::repeat_byte(6), B256::repeat_byte(5)));
        assert!(!state.is_validated(6, B256::repeat_byte(99), B256::repeat_byte(5)));
        assert!(!state.is_validated(6, B256::repeat_byte(6), B256::repeat_byte(99)));
        assert!(!state.is_validated(8, B256::repeat_byte(8), B256::repeat_byte(7)));
    }
}
