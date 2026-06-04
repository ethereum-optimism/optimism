//! Source-chain RPC clients for the runtime cross-unsafe head.
//!
//! Each configured source RPC is a plain URL; the source chain's ID is discovered lazily via
//! `eth_chainId` (no request is made at construction, so a peer EL that has not started cannot
//! deadlock op-reth startup). Each request is bounded by [`SOURCE_RPC_TIMEOUT`].
//!
//! The low-level RPC calls are abstracted behind the [`SourceRpc`] trait so the validation logic
//! ([`validate_initiating_message`], [`source_block_hash`]) can be unit-tested against canned
//! responses without a live endpoint.

use alloy_consensus::BlockHeader;
use alloy_primitives::{Address, B256, U64, keccak256};
use alloy_rpc_client::{ClientBuilder, RpcClient};
use alloy_rpc_types_eth::{Block as RpcBlock, Filter, Log as RpcLog};
use jsonrpsee_core::async_trait;
use kona_interop::RawMessagePayload;
use std::time::Duration;
use tokio::{sync::OnceCell, time::timeout};
use tracing::{debug, warn};

use super::state::SourceRef;

/// Per-request timeout for every source-chain RPC call.
const SOURCE_RPC_TIMEOUT: Duration = Duration::from_secs(10);

/// The low-level source-chain RPC calls the validation logic needs. Abstracted so the validation
/// can be exercised against canned responses in tests; the production implementation is
/// [`SourceLogClient`].
#[async_trait]
trait SourceRpc {
    /// A human-readable identifier for the source (used only in logs).
    fn endpoint(&self) -> &str;
    /// Fetches the source chain's ID (`eth_chainId`).
    async fn fetch_chain_id(&self) -> Option<U64>;
    /// Fetches a source block by number (`eth_getBlockByNumber`).
    async fn fetch_block(&self, block_number: u64) -> Option<RpcBlock>;
    /// Fetches the logs emitted by `origin` in the block with `block_hash` (`eth_getLogs`).
    async fn fetch_logs(&self, block_hash: B256, origin: Address) -> Option<Vec<RpcLog>>;
}

/// Validates an initiating message against a source chain, returning the source block hash on
/// success (so the caller can record the dependency), or `None` on any mismatch / missing data
/// (fail closed).
async fn validate_initiating_message<S: SourceRpc + ?Sized>(
    rpc: &S,
    block_number: u64,
    log_index: u64,
    timestamp: u64,
    origin: Address,
    payload_hash: B256,
) -> Option<B256> {
    let block = rpc.fetch_block(block_number).await?;
    if block.header.timestamp() != timestamp {
        debug!(
            target: "rpc::cross_unsafe",
            endpoint = rpc.endpoint(),
            block_number,
            expected = timestamp,
            actual = block.header.timestamp(),
            "source block timestamp mismatch"
        );
        return None;
    }

    // Filter by the claimed origin address: the message is only valid if the log at `log_index` was
    // emitted by `origin`, so restricting to that address is a correct narrowing and keeps the
    // response small. The block-global `log_index` is preserved by the RPC, so matching it against
    // the filtered subset is still correct.
    let logs = rpc.fetch_logs(block.header.hash, origin).await?;
    let Some(log) = logs.into_iter().find(|log| log.log_index == Some(log_index)) else {
        debug!(target: "rpc::cross_unsafe", endpoint = rpc.endpoint(), block_number, log_index, "source log not found");
        return None;
    };

    if log.address() != origin {
        debug!(
            target: "rpc::cross_unsafe",
            endpoint = rpc.endpoint(),
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
            endpoint = rpc.endpoint(),
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
async fn source_block_hash<S: SourceRpc + ?Sized>(rpc: &S, block_number: u64) -> Option<B256> {
    rpc.fetch_block(block_number).await.map(|block| block.header.hash)
}

#[derive(Debug)]
pub(super) struct SourceLogClients {
    clients: Vec<SourceLogClient>,
}

impl SourceLogClients {
    pub(super) fn new(
        source_rpcs: impl IntoIterator<Item = impl Into<String>>,
    ) -> Result<Self, String> {
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

    pub(super) async fn validate_initiating_message(
        &self,
        chain_id: u64,
        block_number: u64,
        log_index: u64,
        timestamp: u64,
        origin: Address,
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

        let block_hash = validate_initiating_message(
            client,
            block_number,
            log_index,
            timestamp,
            origin,
            payload_hash,
        )
        .await?;
        Some(SourceRef { chain_id, block_number, block_hash })
    }

    /// Returns whether the cached source block is still canonical:
    /// - `Some(true)`  — confirmed canonical (hash matches),
    /// - `Some(false)` — confirmed reorged (the block is reachable but its hash changed),
    /// - `None`        — could not determine (source RPC unreachable / block not yet available), in
    ///   which case callers must NOT treat it as a reorg.
    pub(super) async fn source_block_canonical(
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

        Some(source_block_hash(client, block_number).await? == block_hash)
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
}

#[async_trait]
impl SourceRpc for SourceLogClient {
    fn endpoint(&self) -> &str {
        &self.endpoint
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

    async fn fetch_logs(&self, block_hash: B256, origin: Address) -> Option<Vec<RpcLog>> {
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
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{Bytes, LogData};

    /// A `SourceRpc` returning canned responses, for exercising the validation logic without a live
    /// endpoint.
    struct MockSourceRpc {
        block: Option<RpcBlock>,
        logs: Vec<RpcLog>,
    }

    #[async_trait]
    impl SourceRpc for MockSourceRpc {
        fn endpoint(&self) -> &str {
            "mock"
        }
        async fn fetch_chain_id(&self) -> Option<U64> {
            None
        }
        async fn fetch_block(&self, _block_number: u64) -> Option<RpcBlock> {
            self.block.clone()
        }
        async fn fetch_logs(&self, _block_hash: B256, _origin: Address) -> Option<Vec<RpcLog>> {
            Some(self.logs.clone())
        }
    }

    fn source_block(hash: B256, timestamp: u64) -> RpcBlock {
        let mut block: RpcBlock = RpcBlock::default();
        block.header.hash = hash;
        block.header.inner.timestamp = timestamp;
        block
    }

    fn source_log(origin: Address, log_index: u64) -> RpcLog {
        let mut log: RpcLog = RpcLog::default();
        log.inner.address = origin;
        log.inner.data = LogData::new_unchecked(vec![], Bytes::from_static(b"payload"));
        log.log_index = Some(log_index);
        log
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

    #[tokio::test]
    async fn validate_initiating_message_accepts_a_matching_message() {
        let origin = Address::repeat_byte(0xab);
        let block_hash = B256::repeat_byte(0x11);
        let log = source_log(origin, 3);
        let payload = keccak256(RawMessagePayload::from(&log.inner).as_ref());
        let mock = MockSourceRpc { block: Some(source_block(block_hash, 1_000)), logs: vec![log] };

        assert_eq!(
            validate_initiating_message(&mock, 7, 3, 1_000, origin, payload).await,
            Some(block_hash),
        );
    }

    #[tokio::test]
    async fn validate_initiating_message_rejection_matrix() {
        let origin = Address::repeat_byte(0xab);
        let block_hash = B256::repeat_byte(0x11);
        let log = source_log(origin, 3);
        let payload = keccak256(RawMessagePayload::from(&log.inner).as_ref());
        let mock = MockSourceRpc { block: Some(source_block(block_hash, 1_000)), logs: vec![log] };

        // source block timestamp does not match the claimed initiating timestamp
        assert_eq!(validate_initiating_message(&mock, 7, 3, 999, origin, payload).await, None);
        // no log at the claimed block-global index
        assert_eq!(validate_initiating_message(&mock, 7, 9, 1_000, origin, payload).await, None);
        // the log at the index was not emitted by the claimed origin
        assert_eq!(
            validate_initiating_message(&mock, 7, 3, 1_000, Address::repeat_byte(0xcd), payload)
                .await,
            None,
        );
        // the recomputed payload hash does not match the claimed one
        assert_eq!(
            validate_initiating_message(&mock, 7, 3, 1_000, origin, B256::repeat_byte(0x99)).await,
            None,
        );

        // source block unavailable → cannot validate
        let unavailable = MockSourceRpc { block: None, logs: vec![] };
        assert_eq!(
            validate_initiating_message(&unavailable, 7, 3, 1_000, origin, payload).await,
            None,
        );
    }

    #[tokio::test]
    async fn source_block_hash_reports_canonical_hash() {
        let block_hash = B256::repeat_byte(0x22);
        let present = MockSourceRpc { block: Some(source_block(block_hash, 5)), logs: vec![] };
        assert_eq!(source_block_hash(&present, 7).await, Some(block_hash));

        let unavailable = MockSourceRpc { block: None, logs: vec![] };
        assert_eq!(source_block_hash(&unavailable, 7).await, None);
    }
}
