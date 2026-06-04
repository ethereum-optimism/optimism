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
//! including the source-chain RPC calls it makes. Each source RPC is bounded by a per-request
//! timeout, so a slow source cannot stall the endpoint indefinitely, and the background poller
//! keeps the cache warm so user calls usually have little work to do.
//!
//! The cache data structures live in the `state` submodule and the source-chain RPC clients in the
//! `source` submodule.

use alloy_consensus::{BlockHeader, TxReceipt};
use alloy_eips::BlockHashOrNumber;
use alloy_primitives::{B256, U64, keccak256};
use jsonrpsee::proc_macros::rpc;
use jsonrpsee_core::{RpcResult, async_trait};
use kona_interop::{MESSAGE_EXPIRY_WINDOW, RawMessagePayload, parse_log_to_executing_message};
use reth_provider::BlockReaderIdExt;
use reth_rpc_server_types::result::internal_rpc_err;
use serde::{Deserialize, Serialize};
use std::{collections::BTreeMap, sync::Arc, time::Duration};
use tokio::{sync::Mutex, time::sleep};
use tracing::debug;

mod source;
mod state;

use source::SourceLogClients;
use state::{CachedBlock, CrossUnsafeState, SourceRef};

const CROSS_UNSAFE_HEAD_POLL_INTERVAL: Duration = Duration::from_secs(5);

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
    /// The chain ID of the local chain this extension runs on. Executing messages that initiate on
    /// this chain (intra-chain / self-references) are validated against the local provider rather
    /// than a source RPC.
    local_chain_id: u64,
    source_clients: SourceLogClients,
    state: Mutex<CrossUnsafeState>,
}

impl<Provider> CrossUnsafeHeadExt<Provider> {
    /// Creates a new runtime cross-unsafe head extension.
    pub fn new(
        provider: Provider,
        local_chain_id: u64,
        source_rpcs: impl IntoIterator<Item = impl Into<String>>,
    ) -> Result<Self, String> {
        Ok(Self {
            inner: Arc::new(CrossUnsafeHeadExtInner {
                provider,
                local_chain_id,
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

                let source_chain_id = message.identifier.chainId.saturating_to::<u64>();
                let source_block = message.identifier.blockNumber.saturating_to::<u64>();
                let source_log_index = message.identifier.logIndex.saturating_to::<u64>();

                // A message initiating on our own chain (intra-chain / self-reference) is validated
                // against the local provider — the referenced block already exists locally, and
                // (with the strict timestamp check above) it is strictly earlier than this block.
                // Cross-chain references go to the configured source RPC.
                let source = if source_chain_id == self.inner.local_chain_id {
                    self.validate_local_initiating_message(
                        source_block,
                        source_log_index,
                        initiating_timestamp,
                        message.identifier.origin,
                        message.payloadHash,
                    )?
                } else {
                    self.inner
                        .source_clients
                        .validate_initiating_message(
                            source_chain_id,
                            source_block,
                            source_log_index,
                            initiating_timestamp,
                            message.identifier.origin,
                            message.payloadHash,
                        )
                        .await
                };

                let Some(source) = source else { return Ok(BlockValidation::invalid()) };
                if !sources.contains(&source) {
                    sources.push(source);
                }
            }
        }

        Ok(BlockValidation { valid: true, sources })
    }

    /// Validates an initiating message that lives on the local chain, against the local provider.
    ///
    /// Mirrors the remote `SourceLogClient::validate_initiating_message` checks (block timestamp,
    /// log existence at the claimed block-global index, origin, payload hash) but reads from the
    /// local provider instead of a source RPC. Returns the local block hash as the dependency on
    /// success, or `None` on any mismatch / missing data (fail closed).
    fn validate_local_initiating_message(
        &self,
        block_number: u64,
        log_index: u64,
        timestamp: u64,
        origin: alloy_primitives::Address,
        payload_hash: B256,
    ) -> RpcResult<Option<SourceRef>> {
        let Some(header) = self.inner.provider.sealed_header(block_number).map_err(|err| {
            internal_rpc_err(format!("failed to read local header {block_number}: {err}"))
        })?
        else {
            return Ok(None);
        };
        if header.timestamp() != timestamp {
            debug!(
                target: "rpc::cross_unsafe",
                block_number,
                expected = timestamp,
                actual = header.timestamp(),
                "local initiating block timestamp mismatch"
            );
            return Ok(None);
        }

        let Some(receipts) = self
            .inner
            .provider
            .receipts_by_block(BlockHashOrNumber::Number(block_number))
            .map_err(|err| {
                internal_rpc_err(format!(
                    "failed to read receipts for local block {block_number}: {err}"
                ))
            })?
        else {
            return Ok(None);
        };

        // The identifier's `logIndex` is the block-global position across all logs in the block.
        let Some(log) = receipts.iter().flat_map(|receipt| receipt.logs()).nth(log_index as usize)
        else {
            debug!(target: "rpc::cross_unsafe", block_number, log_index, "local initiating log not found");
            return Ok(None);
        };

        if log.address != origin {
            debug!(
                target: "rpc::cross_unsafe",
                block_number,
                log_index,
                expected = %origin,
                actual = %log.address,
                "local initiating log origin mismatch"
            );
            return Ok(None);
        }

        let local_payload_hash = keccak256(RawMessagePayload::from(log).as_ref());
        if local_payload_hash != payload_hash {
            debug!(
                target: "rpc::cross_unsafe",
                block_number,
                log_index,
                expected = %payload_hash,
                actual = %local_payload_hash,
                "local initiating log payload hash mismatch"
            );
            return Ok(None);
        }

        Ok(Some(SourceRef {
            chain_id: self.inner.local_chain_id,
            block_number,
            block_hash: header.hash(),
        }))
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
/// executing timestamp: strictly earlier than it, and not older than `expiry_window` seconds.
///
/// Same-timestamp initiating messages are rejected. This is a conservative simplification that
/// closes the same-timestamp cycle gap without implementing cycle detection (cycles can only form
/// among same-timestamp messages), and the sequencer does not currently produce same-timestamp
/// cross-chain messages, so it costs no liveness in practice. It is stricter than the protocol; the
/// only effect is that such a block's cross-unsafe head stalls until it goes cross-safe, which is
/// fail-closed and self-healing.
const fn initiating_timestamp_in_window(
    initiating_timestamp: u64,
    executing_timestamp: u64,
    expiry_window: u64,
) -> bool {
    initiating_timestamp < executing_timestamp &&
        initiating_timestamp >= executing_timestamp.saturating_sub(expiry_window)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn initiating_timestamp_window_boundaries() {
        let exec = 1_000_000u64;
        let window = MESSAGE_EXPIRY_WINDOW;

        // Strictly earlier is valid; same timestamp and future are both rejected.
        assert!(initiating_timestamp_in_window(exec - 1, exec, window));
        assert!(!initiating_timestamp_in_window(exec, exec, window));
        assert!(!initiating_timestamp_in_window(exec + 1, exec, window));

        // The oldest allowed timestamp is exactly `exec - window`; one older is expired.
        assert!(initiating_timestamp_in_window(exec - window, exec, window));
        assert!(!initiating_timestamp_in_window(exec - window - 1, exec, window));

        // The lower bound saturates: when the executing timestamp is below the window, any
        // strictly-earlier initiating timestamp (down to 0) is in window.
        assert!(initiating_timestamp_in_window(0, 100, window));
        assert!(!initiating_timestamp_in_window(101, 100, window));
    }
}
