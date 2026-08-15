//! Node-owned canonical L1 observations and direct reader access.
//!
//! The watcher publishes coherent canonical-label snapshots. Engine and Derivation each own a
//! direct [`crate::l1::L1Reader`] and an independent processing cursor; ordinary provider requests
//! are never serialized through the watcher task.

use alloy_consensus::{Header, Receipt, TxEnvelope};
use alloy_eips::{BlockId, eip4844::Blob};
use alloy_primitives::{Address, B256, b256};
use alloy_provider::{Provider, RootProvider};
use async_trait::async_trait;
use kona_derive::{BlobProvider, ChainProvider, PipelineError, PipelineErrorKind};
use kona_genesis::RollupConfig;
use kona_protocol::BlockInfo;
use kona_providers_alloy::{AlloyChainProvider, OnlineBeaconClient, OnlineBlobProvider};
use kona_rpc::{L1State, L1WatcherQueries, L1WatcherQuerySender};
use std::{sync::Arc, time::Duration};
use thiserror::Error;
use tokio::sync::{mpsc, oneshot, watch};

const LABEL_POLL_INTERVAL: Duration = Duration::from_secs(4);
const QUERY_CAPACITY: usize = 64;

/// A canonical L1 head replacement observed by the watcher.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct L1Reorg {
    /// Previously observed canonical head.
    pub previous: BlockInfo,
    /// New canonical head target.
    pub current: BlockInfo,
}

/// One coherent observation of canonical L1 labels.
///
/// Snapshots are targets, not event logs. A watch receiver may skip intermediate revisions, so a
/// consumer must validate its own cursor hash and query any missing blocks directly.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct L1Snapshot {
    /// Latest canonical L1 block.
    pub head: Option<BlockInfo>,
    /// Safe L1 block reported by the execution provider.
    pub safe: Option<BlockInfo>,
    /// Finalized L1 block reported by the execution provider.
    pub finalized: Option<BlockInfo>,
    /// Most recently detected head replacement, for diagnostics only.
    pub reorg: Option<L1Reorg>,
    /// Monotonically increasing observation revision.
    pub revision: u64,
}

/// The result of validating a consumer-owned cursor against the canonical L1 chain.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum L1CursorStatus {
    /// The cursor remains canonical.
    Canonical(BlockInfo),
    /// The cursor was replaced; this is its nearest canonical ancestor known by hash.
    Reorg {
        /// Replaced consumer cursor.
        previous: BlockInfo,
        /// Nearest ancestor that remains canonical, if available by hash.
        common_ancestor: Option<BlockInfo>,
    },
}

/// Direct, cloneable execution and beacon access with a reader-local cache.
#[derive(Debug, Clone)]
pub struct L1Reader {
    provider: RootProvider,
    chain_provider: AlloyChainProvider,
    blob_provider: OnlineBlobProvider<OnlineBeaconClient>,
}

impl L1Reader {
    /// Constructs a direct reader. Cloning it gives a consumer an independent cache and cursor.
    pub const fn new(
        provider: RootProvider,
        chain_provider: AlloyChainProvider,
        blob_provider: OnlineBlobProvider<OnlineBeaconClient>,
    ) -> Self {
        Self { provider, chain_provider, blob_provider }
    }

    /// Looks up an L1 block by hash without treating absence as an error.
    pub async fn block_by_hash(&self, hash: B256) -> Result<Option<BlockInfo>, PipelineErrorKind> {
        self.provider
            .get_block_by_hash(hash)
            .await
            .map(|block| block.map(|block| block.into_consensus().into()))
            .map_err(provider_error)
    }

    /// Looks up a canonical L1 block by number without treating absence as an error.
    pub async fn block_by_number(
        &self,
        number: u64,
    ) -> Result<Option<BlockInfo>, PipelineErrorKind> {
        self.provider
            .get_block_by_number(number.into())
            .await
            .map(|block| block.map(|block| block.into_consensus().into()))
            .map_err(provider_error)
    }

    /// Reads the authoritative unsafe-block signer from `SystemConfig` at a canonical block hash.
    pub async fn unsafe_block_signer_at(
        &self,
        system_config: Address,
        block_hash: B256,
    ) -> Result<Address, PipelineErrorKind> {
        /// `bytes32(uint256(keccak256("systemconfig.unsafeblocksigner")) - 1)`.
        const UNSAFE_BLOCK_SIGNER_SLOT: B256 =
            b256!("65a7ed542fb37fe237fdfbdd70b31598523fe5b32879e307bae27a0bd9581c08");
        let value = self
            .provider
            .get_storage_at(system_config, UNSAFE_BLOCK_SIGNER_SLOT.into())
            .hash(block_hash)
            .await
            .map_err(provider_error)?;
        Ok(Address::from_slice(&value.to_be_bytes_vec()[12..]))
    }

    /// Validates a consumer cursor and, after a reorg, follows its old branch to the nearest
    /// ancestor that is still canonical.
    pub async fn validate_cursor(
        &self,
        cursor: BlockInfo,
    ) -> Result<L1CursorStatus, PipelineErrorKind> {
        if self
            .block_by_number(cursor.number)
            .await?
            .is_some_and(|canonical| canonical.hash == cursor.hash)
        {
            return Ok(L1CursorStatus::Canonical(cursor));
        }

        let mut candidate = Some(cursor);
        while let Some(block) = candidate {
            if self
                .block_by_number(block.number)
                .await?
                .is_some_and(|canonical| canonical.hash == block.hash)
            {
                return Ok(L1CursorStatus::Reorg {
                    previous: cursor,
                    common_ancestor: Some(block),
                });
            }
            if block.number == 0 {
                break;
            }
            candidate = self.block_by_hash(block.parent_hash).await?;
        }

        Ok(L1CursorStatus::Reorg { previous: cursor, common_ancestor: None })
    }
}

#[async_trait]
impl ChainProvider for L1Reader {
    type Error = PipelineErrorKind;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, Self::Error> {
        self.chain_provider.header_by_hash(hash).await.map_err(Into::into)
    }

    async fn block_info_by_number(&mut self, number: u64) -> Result<BlockInfo, Self::Error> {
        self.chain_provider.block_info_by_number(number).await.map_err(Into::into)
    }

    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<Receipt>, Self::Error> {
        self.chain_provider.receipts_by_hash(hash).await.map_err(Into::into)
    }

    async fn block_info_and_transactions_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), Self::Error> {
        self.chain_provider.block_info_and_transactions_by_hash(hash).await.map_err(Into::into)
    }
}

#[async_trait]
impl BlobProvider for L1Reader {
    type Error = PipelineErrorKind;

    async fn get_and_validate_blobs(
        &mut self,
        block_ref: &BlockInfo,
        blob_hashes: &[B256],
    ) -> Result<Vec<Box<Blob>>, Self::Error> {
        self.blob_provider.get_and_validate_blobs(block_ref, blob_hashes).await.map_err(Into::into)
    }
}

/// Shared L1 infrastructure handed to independently progressing node domains.
#[derive(Debug, Clone)]
pub struct L1Access {
    reader: L1Reader,
    snapshots: watch::Receiver<L1Snapshot>,
    query_tx: L1WatcherQuerySender,
}

impl L1Access {
    /// Returns a direct reader with reader-local provider caches.
    pub fn reader(&self) -> L1Reader {
        self.reader.clone()
    }

    /// Subscribes to coherent canonical-label targets.
    pub fn subscribe(&self) -> watch::Receiver<L1Snapshot> {
        self.snapshots.clone()
    }

    /// Returns the compatibility query sender consumed by rollup RPC.
    pub fn query_sender(&self) -> L1WatcherQuerySender {
        self.query_tx.clone()
    }
}

/// Polls canonical L1 labels and publishes coherent snapshots.
#[derive(Debug)]
pub struct L1Watcher {
    config: Arc<RollupConfig>,
    provider: RootProvider,
    snapshots_tx: watch::Sender<L1Snapshot>,
    query_rx: mpsc::Receiver<L1WatcherQueries>,
}

impl L1Watcher {
    /// Creates the watcher and shared access facade.
    pub fn new(
        config: Arc<RollupConfig>,
        provider: RootProvider,
        reader: L1Reader,
    ) -> (Self, L1Access) {
        let (snapshots_tx, snapshots) = watch::channel(L1Snapshot::default());
        let (query_tx, query_rx) = mpsc::channel(QUERY_CAPACITY);
        (
            Self { config, provider, snapshots_tx, query_rx },
            L1Access { reader, snapshots, query_tx },
        )
    }

    /// Runs until the Node-owned lifecycle sender requests shutdown.
    pub async fn run(mut self, mut shutdown: oneshot::Receiver<()>) -> Result<(), L1WatcherError> {
        let mut poll = tokio::time::interval(LABEL_POLL_INTERVAL);
        poll.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Skip);
        let mut query_open = true;

        loop {
            tokio::select! {
                biased;
                _ = &mut shutdown => return Ok(()),
                query = self.query_rx.recv(), if query_open => {
                    if let Some(query) = query {
                        self.handle_rpc_query(query).await;
                    } else {
                        query_open = false;
                    }
                }
                _ = poll.tick() => {
                    match self.observe().await {
                        Ok(snapshot) => {
                            self.snapshots_tx.send_replace(snapshot);
                        }
                        Err(L1WatcherError::Transport(error)) => {
                            warn!(target: "l1", ?error, "Failed to observe coherent L1 labels; retrying");
                        }
                        Err(error) => return Err(error),
                    };
                }
            }
        }
    }

    async fn observe(&self) -> Result<L1Snapshot, L1WatcherError> {
        let (head, safe, finalized) = tokio::join!(
            self.block_info(BlockId::latest()),
            self.block_info(BlockId::safe()),
            self.block_info(BlockId::finalized()),
        );
        let head = head?;
        let safe = safe?;
        let finalized = finalized?;
        let previous = *self.snapshots_tx.borrow();

        self.validate_finalized(previous.finalized, finalized).await?;
        let reorg = self.detect_reorg(previous.head, head).await?;
        let labels_changed = previous.head != head ||
            previous.safe != safe ||
            previous.finalized != finalized ||
            reorg.is_some();

        Ok(L1Snapshot {
            head,
            safe,
            finalized,
            reorg,
            revision: previous.revision.saturating_add(u64::from(labels_changed)),
        })
    }

    async fn detect_reorg(
        &self,
        previous: Option<BlockInfo>,
        current: Option<BlockInfo>,
    ) -> Result<Option<L1Reorg>, L1WatcherError> {
        let (Some(previous), Some(current)) = (previous, current) else {
            return Ok(None);
        };
        if previous.hash == current.hash {
            return Ok(None);
        }

        let replaced = if current.number == previous.number.saturating_add(1) {
            current.parent_hash != previous.hash
        } else if current.number <= previous.number {
            true
        } else {
            self.canonical_block(previous.number)
                .await?
                .is_none_or(|block| block.hash != previous.hash)
        };
        Ok(replaced.then_some(L1Reorg { previous, current }))
    }

    async fn validate_finalized(
        &self,
        previous: Option<BlockInfo>,
        current: Option<BlockInfo>,
    ) -> Result<(), L1WatcherError> {
        let (Some(previous), Some(current)) = (previous, current) else {
            return Ok(());
        };
        validate_finalized_transition(previous, current)?;
        if current.number > previous.number &&
            self.canonical_block(previous.number)
                .await?
                .is_none_or(|block| block.hash != previous.hash)
        {
            return Err(L1WatcherError::FinalizedReplacement { previous, current });
        }
        Ok(())
    }

    async fn canonical_block(&self, number: u64) -> Result<Option<BlockInfo>, L1WatcherError> {
        self.provider
            .get_block_by_number(number.into())
            .await
            .map(|block| block.map(|block| block.into_consensus().into()))
            .map_err(L1WatcherError::Transport)
    }

    async fn block_info(&self, id: BlockId) -> Result<Option<BlockInfo>, L1WatcherError> {
        self.provider
            .get_block(id)
            .await
            .map(|block| block.map(|block| block.into_consensus().into()))
            .map_err(L1WatcherError::Transport)
    }

    async fn handle_rpc_query(&self, query: L1WatcherQueries) {
        match query {
            L1WatcherQueries::Config(response) => {
                let _ = response.send((*self.config).clone());
            }
            L1WatcherQueries::L1State(response) => {
                let snapshot = *self.snapshots_tx.borrow();
                let _ = response.send(L1State {
                    current_l1: snapshot.head,
                    current_l1_finalized: snapshot.finalized,
                    head_l1: snapshot.head,
                    safe_l1: snapshot.safe,
                    finalized_l1: snapshot.finalized,
                });
            }
        }
    }
}

fn validate_finalized_transition(
    previous: BlockInfo,
    current: BlockInfo,
) -> Result<(), L1WatcherError> {
    if current.number < previous.number {
        return Err(L1WatcherError::FinalizedRegression { previous, current });
    }
    if current.number == previous.number && current.hash != previous.hash {
        return Err(L1WatcherError::FinalizedReplacement { previous, current });
    }
    Ok(())
}

fn provider_error(error: alloy_transport::TransportError) -> PipelineErrorKind {
    PipelineError::Provider(error.to_string()).temp()
}

/// Terminal canonical L1 watcher failure.
#[derive(Debug, Error)]
pub enum L1WatcherError {
    /// A transient provider failure. The run loop normally logs and retries this variant.
    #[error("L1 transport failed: {0}")]
    Transport(#[from] alloy_transport::TransportError),
    /// Finalized L1 moved backwards, which would require unfinalizing L2.
    #[error("finalized L1 regressed from {previous:?} to {current:?}")]
    FinalizedRegression {
        /// Previously observed finalized block.
        previous: BlockInfo,
        /// Regressed finalized target.
        current: BlockInfo,
    },
    /// An already-finalized L1 hash was replaced.
    #[error("finalized L1 was replaced: previous {previous:?}, current {current:?}")]
    FinalizedReplacement {
        /// Previously observed finalized block.
        previous: BlockInfo,
        /// New label that conflicts with prior finality.
        current: BlockInfo,
    },
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;

    fn block(number: u64, hash: u8) -> BlockInfo {
        BlockInfo { number, hash: B256::repeat_byte(hash), ..Default::default() }
    }

    #[test]
    fn finalized_l1_regression_is_fatal() {
        assert!(matches!(
            validate_finalized_transition(block(10, 1), block(9, 1)),
            Err(L1WatcherError::FinalizedRegression { .. })
        ));
    }

    #[test]
    fn finalized_l1_hash_replacement_is_fatal() {
        assert!(matches!(
            validate_finalized_transition(block(10, 1), block(10, 2)),
            Err(L1WatcherError::FinalizedReplacement { .. })
        ));
    }

    #[test]
    fn finalized_l1_may_advance_or_repeat_identically() {
        assert!(validate_finalized_transition(block(10, 1), block(10, 1)).is_ok());
        assert!(validate_finalized_transition(block(10, 1), block(11, 2)).is_ok());
    }
}
