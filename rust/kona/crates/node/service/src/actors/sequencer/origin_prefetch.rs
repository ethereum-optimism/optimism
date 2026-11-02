//! Single-writer preparation of L1 origins and the attributes inputs that go with them.

use super::{DelayedL1OriginSelectorProvider, L1OriginSelectorError, L1OriginSelectorProvider};
use crate::NodeActor;
use alloy_consensus::{Header, Receipt, TxEnvelope};
use alloy_eips::BlockNumHash;
use alloy_primitives::B256;
use async_trait::async_trait;
use kona_derive::{ChainProvider, PipelineError, PipelineErrorKind};
use kona_protocol::BlockInfo;
use std::{fmt::Debug, sync::Arc, time::Duration};
use tokio::{sync::watch, time::Interval};

/// An origin is published only after both header and receipts are ready. The selector and
/// attributes builder read the same immutable entries, so adoption cannot miss a receipts cache.
#[derive(Debug)]
struct PreparedOrigin {
    info: BlockInfo,
    header: Header,
    receipts: Vec<Receipt>,
}

#[derive(Debug, Clone, Default)]
struct OriginWindow {
    blocks: Vec<Arc<PreparedOrigin>>,
    orphaned: Option<BlockNumHash>,
    missing: Option<BlockNumHash>,
    validation: WindowValidation,
}

/// Partial publications must not inherit the validation stamp of the previous fork.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
enum WindowValidation {
    #[default]
    Pending,
    Checked(Option<BlockInfo>),
}

impl OriginWindow {
    fn get(&self, hash: B256) -> Option<&Arc<PreparedOrigin>> {
        self.blocks.iter().find(|block| block.info.hash == hash)
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct OriginRequest {
    parent: BlockNumHash,
    selected: BlockNumHash,
    recovery: bool,
}

/// Read-only origin provider. All network work belongs to [`OriginPrefetcher`].
#[derive(Debug)]
pub(crate) struct PreparedOriginProvider {
    window: watch::Receiver<OriginWindow>,
    request: watch::Sender<Option<OriginRequest>>,
    // This copy is used only for the confirmation policy; it never performs RPCs.
    delayed: DelayedL1OriginSelectorProvider,
}

impl PreparedOriginProvider {
    fn request(&self, request: OriginRequest) -> Result<(), L1OriginSelectorError> {
        if self.window.has_changed().is_err() || self.request.is_closed() {
            return Err(L1OriginSelectorError::Closed);
        }
        self.request.send_if_modified(|current| {
            if *current == Some(request) {
                return false;
            }
            *current = Some(request);
            true
        });
        let window = self.window.borrow();
        if window.missing == Some(request.parent) {
            return Err(L1OriginSelectorError::OriginNotFound(request.parent.hash));
        }
        if window
            .orphaned
            .is_some_and(|origin| origin == request.parent || origin == request.selected)
        {
            return Err(L1OriginSelectorError::NextOriginOrphaned);
        }
        Ok(())
    }
}

#[async_trait]
impl L1OriginSelectorProvider for PreparedOriginProvider {
    fn prepare(&self, origin: BlockNumHash, recovery: bool) -> Result<(), L1OriginSelectorError> {
        self.request(OriginRequest { parent: origin, selected: origin, recovery })
    }

    fn selected(
        &self,
        parent: BlockNumHash,
        selected: BlockNumHash,
        recovery: bool,
    ) -> Result<(), L1OriginSelectorError> {
        self.request(OriginRequest { parent, selected, recovery })
    }

    async fn refresh(&self) -> Result<(), L1OriginSelectorError> {
        let mut window = self.window.clone();
        window.borrow_and_update();
        self.request.send_modify(|_| {});
        window.changed().await.map_err(|_| L1OriginSelectorError::Closed)
    }

    async fn get_block_by_hash(
        &self,
        hash: B256,
    ) -> Result<Option<BlockInfo>, L1OriginSelectorError> {
        let window = self.window.borrow();
        if window.missing.is_some_and(|origin| origin.hash == hash) {
            return Ok(None);
        }
        window.get(hash).map(|block| Some(block.info)).ok_or(L1OriginSelectorError::Pending)
    }

    async fn get_block_by_number(
        &self,
        number: u64,
    ) -> Result<Option<BlockInfo>, L1OriginSelectorError> {
        if !self.delayed.is_confirmed(number) {
            return Ok(None);
        }
        let window = self.window.borrow();
        let request = *self.request.borrow();
        if window.orphaned.is_some_and(|origin| {
            origin.number.checked_add(1) == Some(number) &&
                request.is_some_and(|request| {
                    origin == request.parent || origin == request.selected
                })
        }) {
            return Err(L1OriginSelectorError::NextOriginOrphaned);
        }
        Ok(window.blocks.iter().find(|block| block.info.number == number).map(|block| block.info))
    }
}

/// Attributes-only provider with no network fallback. Missing preparation retries on the next
/// sequencer tick rather than putting an L1 RPC back on the block-building path.
#[derive(Debug, Clone)]
pub(crate) struct PreparedAttributesProvider {
    window: watch::Receiver<OriginWindow>,
}

impl PreparedAttributesProvider {
    fn get(&self, hash: B256) -> Result<Arc<PreparedOrigin>, PipelineErrorKind> {
        self.window.borrow().get(hash).cloned().ok_or_else(|| {
            PipelineError::Provider(format!("L1 attributes inputs not prepared for {hash}")).temp()
        })
    }
}

#[async_trait]
impl ChainProvider for PreparedAttributesProvider {
    type Error = PipelineErrorKind;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, Self::Error> {
        Ok(self.get(hash)?.header.clone())
    }

    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<Receipt>, Self::Error> {
        Ok(self.get(hash)?.receipts.clone())
    }

    async fn block_info_by_number(&mut self, number: u64) -> Result<BlockInfo, Self::Error> {
        self.window
            .borrow()
            .blocks
            .iter()
            .find(|block| block.info.number == number)
            .map(|block| block.info)
            .ok_or_else(|| {
                PipelineError::Provider(format!("L1 origin {number} not prepared")).temp()
            })
    }

    async fn block_info_and_transactions_by_hash(
        &mut self,
        _hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), Self::Error> {
        Err(PipelineError::Provider("Sequencer attributes do not use L1 transactions".into())
            .crit())
    }
}

/// Prepares the current origin and two successors, or three successors when adoption has already
/// been requested. The parent is retained until the following build, including failed builds.
/// Runs under the node supervisor, so failure, panic and shutdown cancel its peers normally.
#[derive(Debug)]
pub(crate) struct OriginPrefetcher<P, C> {
    origins: P,
    chain: C,
    request: watch::Receiver<Option<OriginRequest>>,
    window: watch::Sender<OriginWindow>,
    l1_head: watch::Receiver<Option<BlockInfo>>,
    retry: Interval,
    fetch_timeout: Duration,
}

/// Components that must share one preparation window.
#[derive(Debug)]
pub(crate) struct OriginPreparation<C> {
    pub(crate) prefetcher: OriginPrefetcher<DelayedL1OriginSelectorProvider, C>,
    pub(crate) origins: PreparedOriginProvider,
    pub(crate) attributes: PreparedAttributesProvider,
}

impl<C> OriginPreparation<C> {
    pub(crate) fn new(
        rpc: alloy_provider::RootProvider,
        l1_head: watch::Receiver<Option<BlockInfo>>,
        confirmation_depth: u64,
        chain: C,
    ) -> Self {
        let (request_tx, request) = watch::channel(None);
        let (window, window_rx) = watch::channel(OriginWindow::default());
        let provider = PreparedOriginProvider {
            request: request_tx,
            window: window_rx.clone(),
            delayed: DelayedL1OriginSelectorProvider::new(
                rpc.clone(),
                l1_head.clone(),
                confirmation_depth,
            ),
        };
        let attributes = PreparedAttributesProvider { window: window_rx };
        let mut retry = tokio::time::interval(Duration::from_secs(1));
        retry.set_missed_tick_behavior(tokio::time::MissedTickBehavior::Delay);
        let worker = OriginPrefetcher {
            origins: DelayedL1OriginSelectorProvider::new(rpc, l1_head.clone(), confirmation_depth),
            chain,
            request,
            window,
            l1_head,
            retry,
            fetch_timeout: Duration::from_secs(10),
        };
        Self { prefetcher: worker, origins: provider, attributes }
    }
}

impl<P, C> OriginPrefetcher<P, C>
where
    P: L1OriginSelectorProvider,
    C: ChainProvider + Send,
{
    async fn prepare(
        &mut self,
        info: BlockInfo,
    ) -> Result<Arc<PreparedOrigin>, L1OriginSelectorError> {
        if let Some(block) = self.window.borrow().get(info.hash) {
            return Ok(block.clone());
        }
        let header = tokio::time::timeout(self.fetch_timeout, self.chain.header_by_hash(info.hash))
            .await
            .map_err(|_| L1OriginSelectorError::Timeout)?
            .map_err(|err| L1OriginSelectorError::Attributes(err.to_string()))?;
        let receipts =
            tokio::time::timeout(self.fetch_timeout, self.chain.receipts_by_hash(info.hash))
                .await
                .map_err(|_| L1OriginSelectorError::Timeout)?
                .map_err(|err| L1OriginSelectorError::Attributes(err.to_string()))?;
        Ok(Arc::new(PreparedOrigin { info, header, receipts }))
    }

    async fn fill(
        &mut self,
        request: OriginRequest,
        recheck: bool,
    ) -> Result<(), L1OriginSelectorError> {
        let old = self.window.borrow().clone();
        let head = *self.l1_head.borrow();
        let recheck = recheck || old.validation != WindowValidation::Checked(head);
        let current = match old.get(request.parent.hash) {
            Some(block) => block.clone(),
            None => {
                let info = tokio::time::timeout(
                    self.fetch_timeout,
                    self.origins.get_block_by_hash(request.parent.hash),
                )
                .await
                .map_err(|_| L1OriginSelectorError::Timeout)??;
                let Some(info) = info else {
                    self.window.send_replace(OriginWindow {
                        missing: Some(request.parent),
                        ..Default::default()
                    });
                    return Ok(());
                };
                self.prepare(info).await?
            }
        };
        let mut window = OriginWindow { blocks: vec![current], ..Default::default() };
        if old.get(request.parent.hash).is_none() {
            self.window.send_replace(window.clone());
        }
        let last_number = request.selected.number.saturating_add(2);
        while window.blocks.last().expect("current origin exists").info.number < last_number {
            let parent = window.blocks.last().expect("current origin exists").info;
            let number = parent.number + 1;
            // Head updates recheck canonical linkage, even if the receipts are already warm.
            let next = if !recheck &&
                let Some(block) = old.blocks.iter().find(|block| {
                    block.info.number == number && block.info.parent_hash == parent.hash
                }) {
                Some(block.info)
            } else {
                tokio::time::timeout(self.fetch_timeout, self.origins.get_block_by_number(number))
                    .await
                    .map_err(|_| L1OriginSelectorError::Timeout)??
            };
            let Some(next) = next else { break };
            if next.parent_hash != parent.hash {
                // Retain the valid prefix, but never select the orphaned parent for adoption.
                window.orphaned = Some(parent.id());
                self.window.send_replace(window);
                return Ok(());
            }
            let prepared = match old.get(next.hash) {
                Some(block) => block.clone(),
                None => self.prepare(next).await?,
            };
            let newly_prepared = old.get(prepared.info.hash).is_none();
            window.blocks.push(prepared);
            if newly_prepared {
                self.window.send_replace(window.clone());
            }
        }
        window.validation = WindowValidation::Checked(head);
        self.window.send_replace(window);
        Ok(())
    }
}

#[async_trait]
impl<P, C> NodeActor for OriginPrefetcher<P, C>
where
    P: L1OriginSelectorProvider + Send + 'static,
    C: ChainProvider + Debug + Send + 'static,
{
    type Error = L1OriginSelectorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        let recheck = tokio::select! {
            result = self.request.changed() => {
                result.map_err(|_| L1OriginSelectorError::Closed)?;
                false
            }
            result = self.l1_head.changed() => {
                result.map_err(|_| L1OriginSelectorError::Closed)?;
                true
            }
            _ = self.retry.tick() => false,
        };
        let Some(request) = *self.request.borrow_and_update() else { return Ok(()) };
        // A new request or L1 view cancels in-flight work. Never publish a result from a
        // previous generation after reset, recovery-mode change, or a head update.
        let mut requests = self.request.clone();
        let mut heads = self.l1_head.clone();
        tokio::select! {
            // A max-drift refresh repeats the same request to wake an idle worker. It must
            // not restart an already-running fetch, or short L2 ticks can starve slow receipts.
            result = requests.wait_for(|current| *current != Some(request)) => {
                result.map(|_| ()).map_err(|_| L1OriginSelectorError::Closed)
            },
            result = heads.changed() => result.map_err(|_| L1OriginSelectorError::Closed),
            result = self.fill(request, recheck) => {
                if let Err(err) = result {
                    warn!(target: "l1_origin_selector", ?err, "L1 origin prefetch failed; retrying in background");
                }
                Ok(())
            }
        }
    }
}

#[cfg(test)]
#[path = "origin_prefetch_test.rs"]
mod tests;
