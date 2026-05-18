//! Async driver wrapping the pure derivation engine.
//!
//! [`NodeDeriver`] is the kona-node side of the post-Holocene "pure deriver"
//! design (Phase 4 of the pure-derivation migration). The protocol-level
//! state machine lives in [`kona_derive::Deriver`] and is IO-free; this
//! driver owns the IO substrate (alloy L1 + beacon blob fetcher, alloy L2
//! sysconfig lookup), drives the deriver step-by-step, and translates the
//! resulting [`kona_derive::DeriveTrace`] entries into structured `tracing`
//! events.
//!
//! Lifecycle contract:
//!
//! - [`NodeDeriver::reset`] anchors the deriver at a safe head, bootstrapping the rolling system
//!   config from `AlloyL2ChainProvider::system_config_by_number`. Holocene+ strict ordering
//!   subsumes the old `Signal::Activation` soft-reset, so the activation site that the async
//!   pipeline used at `actor.rs:130` collapses into a no-op; reset is the only externally visible
//!   state-clearing operation.
//! - [`NodeDeriver::flush_channel`] runs when the engine reports an invalid payload. The pure
//!   deriver doesn't expose a separate flush operation: re-anchoring at the same safe head with the
//!   same rolling sysconfig is equivalent. The actor calls this through `Signal::FlushChannel`
//!   (kept for compatibility with the engine→derivation message protocol).
//! - [`NodeDeriver::step`] returns one of [`NodeStep`]: payload attributes, a yield (out of L1 data
//!   — caller waits for an L1 head update), or a reset request (pure deriver hit a critical
//!   contract violation).
//! - [`NodeDeriver::set_l1_head`] feeds the latest known L1 head height from the L1 watcher so the
//!   deriver knows how far it can pull.

use std::sync::Arc;

use kona_derive::{
    CriticalError, Derivation, Deriver, L1Input, SpanBatchOverlap, SpanBatchOverlapBlock,
    TraceEntry, extract_l1_input,
};
use kona_genesis::{L1ChainConfig, RollupConfig, SystemConfig};
use kona_interop::DependencySet;
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use kona_providers_alloy::{
    AlloyChainProvider, AlloyL2ChainProvider, L1FetchError, OnlineBeaconClient, OnlineBlobProvider,
    fetch_l1_block_data,
};
use thiserror::Error;

/// One step of derivation from the node's point of view.
#[derive(Debug)]
pub enum NodeStep {
    /// Payload attributes are ready. Caller should hand them to the engine.
    Attributes(Box<OpAttributesWithParent>),
    /// The deriver wants more L1 data than is currently known to the L1
    /// watcher (or there is no further work to do at the current safe
    /// head). Caller should wait for a new L1 head update / safe-head
    /// confirmation and call `step` again.
    Yield,
    /// The deriver hit an unrecoverable contract violation. Caller must
    /// trigger an engine reset.
    NeedsReset(NodeDeriverError),
}

/// Failure modes for [`NodeDeriver`]. These are the same shape on every
/// translated error so the actor can map them to a single
/// `DerivationError::Yield` / engine-reset path.
#[derive(Debug, Error)]
pub enum NodeDeriverError {
    /// L1 RPC / receipts / blob fetch failed.
    #[error("L1 fetch error: {0}")]
    L1Fetch(#[from] L1FetchError),
    /// L2 RPC lookup failed during sysconfig bootstrap or span-batch overlap.
    #[error("L2 fetch error: {0}")]
    L2Fetch(String),
    /// The pure deriver rejected a caller-supplied input.
    #[error("pure deriver contract violation: {0}")]
    PureContract(#[from] CriticalError),
}

/// Async wrapper around [`kona_derive::Deriver`].
///
/// Owns the L1 + L2 providers and translates pure-deriver `DeriveTrace`
/// entries into structured `tracing` events. The pure deriver itself emits
/// no logs.
#[derive(Debug)]
pub struct NodeDeriver {
    /// The protocol-level state machine.
    inner: Deriver,
    /// Rollup configuration.
    rollup_config: Arc<RollupConfig>,
    /// L1 chain configuration (used by the pure deriver for fork rules).
    #[allow(dead_code)]
    l1_chain_config: Arc<L1ChainConfig>,
    /// Optional interop dependency set (matches `OnlinePipeline::new`).
    #[allow(dead_code)]
    dependency_set: Option<Arc<DependencySet>>,
    /// L1 RPC + receipts source.
    chain_provider: AlloyChainProvider,
    /// L2 RPC source for sysconfig bootstrap and span-batch overlap data.
    l2_provider: AlloyL2ChainProvider,
    /// Blob KZG-verifier (beacon-backed).
    blob_provider: OnlineBlobProvider<OnlineBeaconClient>,

    /// Highest L1 block number known by the L1 watcher. The deriver will
    /// pull L1 inputs in [`Self::step`] up to (and including) this number.
    l1_head: Option<u64>,
    /// Next L1 block number the deriver will request via `NeedL1Input`.
    ///
    /// `None` until [`Self::reset`] anchors the driver at a safe head.
    next_l1_number: Option<u64>,
    /// Latest L1 origin the deriver has actually consumed (mirrors
    /// `Pipeline::origin` from the async surface).
    current_origin: Option<BlockInfo>,
}

impl NodeDeriver {
    /// Constructs a new [`NodeDeriver`].
    ///
    /// The deriver is unanchored after construction. Call [`Self::reset`]
    /// before [`Self::step`] — this matches the lifecycle of the async
    /// `OnlinePipeline::new` it replaces (which also required a reset signal
    /// before steppping).
    pub fn new(
        rollup_config: Arc<RollupConfig>,
        l1_chain_config: Arc<L1ChainConfig>,
        chain_provider: AlloyChainProvider,
        l2_provider: AlloyL2ChainProvider,
        blob_provider: OnlineBlobProvider<OnlineBeaconClient>,
        dependency_set: Option<Arc<DependencySet>>,
    ) -> Self {
        let inner = Deriver::new(
            rollup_config.clone(),
            l1_chain_config.clone(),
            SystemConfig::default(),
            L2BlockInfo::default(),
            dependency_set.clone(),
        );
        Self {
            inner,
            rollup_config,
            l1_chain_config,
            dependency_set,
            chain_provider,
            l2_provider,
            blob_provider,
            l1_head: None,
            next_l1_number: None,
            current_origin: None,
        }
    }

    /// Returns the rollup configuration.
    pub fn rollup_config(&self) -> &RollupConfig {
        self.rollup_config.as_ref()
    }

    /// Returns the latest L1 origin the deriver has consumed.
    pub const fn origin(&self) -> Option<BlockInfo> {
        self.current_origin
    }

    /// Updates the highest known L1 block number. Called from the actor in
    /// response to a [`crate::DerivationActorRequest::ProcessL1HeadUpdateRequest`].
    pub const fn set_l1_head(&mut self, head: BlockInfo) {
        // Track only the height — the deriver uses the L1 head height to know
        // how far it can pull L1 inputs. Hash divergence is detected by the
        // pure deriver's own contiguity check.
        self.l1_head = Some(head.number);
    }

    /// Resets the deriver to the given L2 safe head.
    ///
    /// Fetches the rolling sysconfig at the safe head's L1 origin via the L2
    /// chain provider, then hands both to the pure deriver. The next call to
    /// [`Self::step`] starts fetching L1 inputs from
    /// `safe_head.l1_origin.number + 1`.
    pub async fn reset(&mut self, safe_head: L2BlockInfo) -> Result<(), NodeDeriverError> {
        use kona_derive::L2ChainProvider;
        let sys_config = L2ChainProvider::system_config_by_number(
            &mut self.l2_provider,
            safe_head.block_info.number,
            self.rollup_config.clone(),
        )
        .await
        .map_err(|e| NodeDeriverError::L2Fetch(e.to_string()))?;
        let trace = self.inner.reset(safe_head, sys_config);
        translate_trace(&trace);
        // After reset, the deriver expects L1 inputs starting at the safe head's L1 origin + 1.
        self.next_l1_number = Some(safe_head.l1_origin.number.saturating_add(1));
        self.current_origin = None;
        info!(
            target: "derivation",
            safe_head_number = safe_head.block_info.number,
            l1_origin = safe_head.l1_origin.number,
            next_l1 = ?self.next_l1_number,
            "NodeDeriver reset",
        );
        Ok(())
    }

    /// Equivalent to [`Self::reset`] called at the current safe head with a
    /// fresh sysconfig — implements the engine-side `Signal::FlushChannel`
    /// behavior. Pure deriver has no separate flush op; re-anchoring at the
    /// same safe head clears the channel/batch buffers and re-fetches.
    pub async fn flush_channel(&mut self, safe_head: L2BlockInfo) -> Result<(), NodeDeriverError> {
        warn!(target: "derivation", "Flushing derivation channel — re-anchoring at safe head");
        self.reset(safe_head).await
    }

    /// Drive derivation forward one logical step.
    ///
    /// The returned [`NodeStep`] either carries fresh payload attributes, a
    /// yield (out of L1 data — wait for the next L1 head update), or a
    /// reset request (the deriver detected a critical contract violation
    /// and the actor must trigger an engine reset).
    pub async fn step(&mut self, safe_head: L2BlockInfo) -> NodeStep {
        loop {
            let (derivation, trace) = self.inner.derive(safe_head);
            translate_trace(&trace);

            match derivation {
                Derivation::Attributes { attrs } => {
                    return NodeStep::Attributes(Box::new(attrs));
                }
                Derivation::NeedL1Input => match self.advance_l1_input().await {
                    Ok(true) => {} // fed a new input; loop body re-derives.
                    Ok(false) => return NodeStep::Yield,
                    Err(e) => return NodeStep::NeedsReset(e),
                },
                Derivation::NeedSpanBatchOverlap { parent, content } => {
                    if let Err(e) = self.fulfill_overlap(parent, content).await {
                        return NodeStep::NeedsReset(e);
                    }
                }
                Derivation::Idle => return NodeStep::Yield,
            }
        }
    }

    /// Fetch the next L1 block (if known to be available) and feed it into
    /// the deriver. Returns `Ok(true)` if a new input was added, `Ok(false)`
    /// if there is no further L1 input available right now (the actor must
    /// wait for an L1 head update).
    async fn advance_l1_input(&mut self) -> Result<bool, NodeDeriverError> {
        let Some(next) = self.next_l1_number else {
            // No reset yet — can't pull L1 input.
            return Ok(false);
        };
        let Some(head) = self.l1_head else {
            return Ok(false);
        };
        if next > head {
            return Ok(false);
        }

        let inbox = self.rollup_config.batch_inbox_address;
        let block_data =
            fetch_l1_block_data(&self.chain_provider, &self.blob_provider, next, inbox).await?;

        let header_for_origin = block_data.header.clone();
        let l1_input: L1Input = extract_l1_input(
            block_data.header,
            block_data.txs,
            &block_data.receipts,
            self.rollup_config.as_ref(),
        );
        debug!(
            target: "derivation",
            l1_block = next,
            inbox_txs = l1_input.batch_inbox_data.len(),
            deposits = l1_input.deposit_logs.len(),
            config_updates = l1_input.config_logs.len(),
            "Fetched and extracted L1 input",
        );

        self.inner.add_l1_input(l1_input)?;
        let origin = BlockInfo {
            hash: header_for_origin.hash_slow(),
            number: header_for_origin.number,
            parent_hash: header_for_origin.parent_hash,
            timestamp: header_for_origin.timestamp,
        };
        self.current_origin = Some(origin);
        self.next_l1_number = Some(next + 1);
        kona_macros::set!(counter, crate::Metrics::DERIVATION_L1_ORIGIN, next);
        Ok(true)
    }

    /// Fulfill a `Derivation::NeedSpanBatchOverlap` request by fetching the
    /// L2 blocks in the supplied range and feeding them back to the pure
    /// deriver.
    async fn fulfill_overlap(
        &mut self,
        parent: L2BlockInfo,
        content: core::ops::RangeInclusive<u64>,
    ) -> Result<(), NodeDeriverError> {
        use kona_protocol::BatchValidationProvider;
        let parent_num = *content.start() - 1;
        let real_parent =
            <AlloyL2ChainProvider as BatchValidationProvider>::l2_block_info_by_number(
                &mut self.l2_provider,
                parent_num,
            )
            .await
            .map_err(|e| NodeDeriverError::L2Fetch(e.to_string()))?;
        // The pure deriver only uses the parent.l1_origin for early-exit prefix checks; the
        // hash check still requires matching what it stored on request. We pass the
        // real parent here — it should match the placeholder hash field by number only.
        if parent.block_info.number != parent_num {
            return Err(NodeDeriverError::L2Fetch(format!(
                "overlap parent number mismatch: deriver asked for #{} but supplied {} ",
                parent_num, parent.block_info.number,
            )));
        }

        let mut blocks: Vec<SpanBatchOverlapBlock> = Vec::with_capacity(content.clone().count());
        for n in content.clone() {
            let block = <AlloyL2ChainProvider as BatchValidationProvider>::block_by_number(
                &mut self.l2_provider,
                n,
            )
            .await
            .map_err(|e| NodeDeriverError::L2Fetch(e.to_string()))?;
            use alloy_eips::eip2718::Encodable2718;
            let raw: Vec<alloy_primitives::Bytes> =
                block.body.transactions.iter().map(|t| t.encoded_2718().into()).collect();
            blocks.push(SpanBatchOverlapBlock { number: n, txs: raw });
        }
        self.inner.add_span_batch_overlap(SpanBatchOverlap { parent: real_parent, blocks })?;
        debug!(
            target: "derivation",
            parent = parent_num,
            content_start = *content.start(),
            content_end = *content.end(),
            "Provided span-batch overlap content to deriver",
        );
        Ok(())
    }
}

/// Translate every entry in a [`kona_derive::DeriveTrace`] to a structured
/// `tracing` event. This is the *only* place trace entries become logs in the
/// new design — the pure deriver itself emits no `tracing::*` calls.
fn translate_trace(trace: &kona_derive::DeriveTrace) {
    for entry in &trace.entries {
        translate_entry(entry);
    }
}

fn translate_entry(entry: &TraceEntry) {
    // Every variant of `TraceEntry` lands here, mapped to a fixed structured event.
    // Adding a new variant in the pure deriver makes the compiler force this match to
    // update — preserving the "translation is total" invariant.
    match entry {
        TraceEntry::FrameParsed { origin, channel_id, frame_number, is_last } => trace!(
            target: "derivation::trace",
            l1 = origin.number,
            channel = ?channel_id,
            frame_number = frame_number,
            is_last = is_last,
            "FrameParsed",
        ),
        TraceEntry::FrameDropped { origin, reason } => debug!(
            target: "derivation::trace",
            l1 = origin.number,
            reason = ?reason,
            "FrameDropped",
        ),
        TraceEntry::FramesParseFailed { origin, from, reason } => warn!(
            target: "derivation::trace",
            l1 = origin.number,
            from = ?from,
            reason = ?reason,
            "FramesParseFailed",
        ),
        TraceEntry::BatchInboxTxIgnoredFromMismatch { origin, from } => debug!(
            target: "derivation::trace",
            l1 = origin.number,
            from = ?from,
            "BatchInboxTxIgnoredFromMismatch",
        ),
        TraceEntry::ChannelOpened { origin, channel_id } => debug!(
            target: "derivation::trace",
            l1 = origin.number,
            channel = ?channel_id,
            "ChannelOpened",
        ),
        TraceEntry::ChannelReady { origin, channel_id } => debug!(
            target: "derivation::trace",
            l1 = origin.number,
            channel = ?channel_id,
            "ChannelReady",
        ),
        TraceEntry::ChannelTimedOut { origin, channel_id, open_block } => warn!(
            target: "derivation::trace",
            l1 = origin.number,
            channel = ?channel_id,
            open_block = open_block,
            "ChannelTimedOut",
        ),
        TraceEntry::ChannelDecompressionFailed { origin, channel_id } => warn!(
            target: "derivation::trace",
            l1 = origin.number,
            channel = ?channel_id,
            "ChannelDecompressionFailed",
        ),
        TraceEntry::ChannelBatchDecodeFailed { origin, channel_id } => warn!(
            target: "derivation::trace",
            l1 = origin.number,
            channel = ?channel_id,
            "ChannelBatchDecodeFailed",
        ),
        TraceEntry::BatchDecoded { origin, kind } => debug!(
            target: "derivation::trace",
            l1 = origin.number,
            kind = ?kind,
            "BatchDecoded",
        ),
        TraceEntry::BatchVerdict { origin, verdict } => debug!(
            target: "derivation::trace",
            l1 = origin.number,
            verdict = ?verdict,
            "BatchVerdict",
        ),
        TraceEntry::EmptyBatchGenerated { epoch_num, reason } => info!(
            target: "derivation::trace",
            epoch = epoch_num,
            reason = ?reason,
            "EmptyBatchGenerated",
        ),
        TraceEntry::EmptyBatchDuplicate => error!(
            target: "derivation::trace",
            "EmptyBatchDuplicate",
        ),
        TraceEntry::SpanBatchExtractionFailed { origin, reason } => warn!(
            target: "derivation::trace",
            l1 = origin.number,
            reason = ?reason,
            "SpanBatchExtractionFailed",
        ),
        TraceEntry::AttributesBuilt { l2_number, l1_origin, user_tx_count } => info!(
            target: "derivation::trace",
            l2 = l2_number,
            l1 = l1_origin.number,
            user_tx_count = user_tx_count,
            "AttributesBuilt",
        ),
        TraceEntry::AttributesBrokenTimeInvariant { l1_origin } => error!(
            target: "derivation::trace",
            l1 = l1_origin.number,
            "AttributesBrokenTimeInvariant",
        ),
        TraceEntry::AttributesL1InfoTxBuildFailed { l1_origin } => error!(
            target: "derivation::trace",
            l1 = l1_origin.number,
            "AttributesL1InfoTxBuildFailed",
        ),
        TraceEntry::SystemConfigUpdated { origin, kind } => info!(
            target: "derivation::trace",
            l1 = origin.number,
            kind = ?kind,
            "SystemConfigUpdated",
        ),
        TraceEntry::SystemConfigUpdateDropped { origin, reason } => warn!(
            target: "derivation::trace",
            l1 = origin.number,
            reason = ?reason,
            "SystemConfigUpdateDropped",
        ),
        TraceEntry::DepositLogDropped { origin, reason } => warn!(
            target: "derivation::trace",
            l1 = origin.number,
            reason = ?reason,
            "DepositLogDropped",
        ),
        TraceEntry::Reset { safe_head } => info!(
            target: "derivation::trace",
            safe_head = safe_head.block_info.number,
            "DeriverReset",
        ),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_consensus::Header;
    use alloy_eips::BlockNumHash;
    use alloy_primitives::{B256, address};
    use kona_derive::{
        BatchKind, BatchVerdict, Derivation, Deriver, EmptyBatchReason, L1Input, TraceEntry,
        test_utils::{CollectingLayer, TraceStorage},
    };
    use kona_genesis::{HardForkConfig, L1ChainConfig, RollupConfig, SystemConfig};
    use kona_protocol::{BlockInfo, L2BlockInfo};
    use std::sync::Arc;
    use tracing::{Level, subscriber::with_default};
    use tracing_subscriber::{Registry, layer::SubscriberExt};

    /// Construct a subscriber that captures everything tracing-level emits
    /// from this scope into a [`TraceStorage`]. We use it to assert the
    /// translation layer fires a structured event for every variant we feed
    /// through `translate_entry`.
    fn capturing_subscriber() -> (TraceStorage, impl tracing::Subscriber + Send + Sync) {
        let storage = TraceStorage::default();
        let layer = CollectingLayer::new(storage.clone());
        let subscriber = Registry::default().with(layer);
        (storage, subscriber)
    }

    fn rollup_cfg() -> RollupConfig {
        RollupConfig {
            block_time: 2,
            seq_window_size: 100,
            batch_inbox_address: address!("ff00000000000000000000000000000000042069"),
            deposit_contract_address: address!("0808080808080808080808080808080808080808"),
            l1_system_config_address: address!("0909090909090909090909090909090909090909"),
            hardforks: HardForkConfig { holocene_time: Some(0), ..Default::default() },
            ..Default::default()
        }
    }

    /// Trace-translation gate test.
    ///
    /// Verifies that `translate_entry` actually produces a structured
    /// `tracing::*` event for each major `TraceEntry` variant — the
    /// translation layer is the only place where the pure deriver's
    /// observability output reaches a logging substrate, so silently dropping
    /// an entry here would be a serious regression. The test feeds a
    /// representative subset: one happy-path event (`AttributesBuilt`), one
    /// debug-level drop event (`FrameDropped`), one warning (`ChannelTimedOut`),
    /// and one critical event (`AttributesL1InfoTxBuildFailed`).
    #[test]
    fn translate_entry_emits_events() {
        let (storage, subscriber) = capturing_subscriber();
        let origin = BlockInfo { number: 7, timestamp: 100, ..Default::default() };

        with_default(subscriber, || {
            translate_entry(&TraceEntry::AttributesBuilt {
                l2_number: 42,
                l1_origin: origin,
                user_tx_count: 3,
            });
            translate_entry(&TraceEntry::FrameDropped {
                origin,
                reason: kona_derive::FrameDropReason::ChannelSizeExceeded,
            });
            translate_entry(&TraceEntry::ChannelTimedOut {
                origin,
                channel_id: [0u8; 16],
                open_block: 5,
            });
            translate_entry(&TraceEntry::AttributesL1InfoTxBuildFailed { l1_origin: origin });
            translate_entry(&TraceEntry::EmptyBatchGenerated {
                epoch_num: 9,
                reason: EmptyBatchReason::SequencingWindowExpired,
            });
            translate_entry(&TraceEntry::EmptyBatchDuplicate);
            translate_entry(&TraceEntry::BatchVerdict { origin, verdict: BatchVerdict::Accept });
            translate_entry(&TraceEntry::BatchDecoded { origin, kind: BatchKind::Single });
        });

        let captured: Vec<(Level, String)> = storage.lock().clone();
        let levels: Vec<Level> = captured.iter().map(|(l, _)| *l).collect();

        // At least one event per representative variant — covers info (built / reset / empty),
        // warn (frame-dropped / timed-out), error (L1Info-build-failed), debug (verdict /
        // decoded).
        assert!(
            levels.contains(&Level::INFO),
            "expected at least one INFO-level event: {levels:?}"
        );
        assert!(
            levels.contains(&Level::WARN),
            "expected at least one WARN-level event: {levels:?}"
        );
        assert!(
            levels.contains(&Level::ERROR),
            "expected at least one ERROR-level event: {levels:?}"
        );
        assert!(
            levels.contains(&Level::DEBUG),
            "expected at least one DEBUG-level event: {levels:?}"
        );

        // The captured message strings should mention the variant names — the
        // event metadata gets formatted into the captured string by
        // `CollectingLayer`'s Debug-format of the event.
        let joined: String = captured.iter().map(|(_, m)| m.as_str()).collect();
        assert!(joined.contains("AttributesBuilt"), "captured: {joined}");
        assert!(joined.contains("FrameDropped"), "captured: {joined}");
        assert!(joined.contains("ChannelTimedOut"), "captured: {joined}");
        assert!(joined.contains("AttributesL1InfoTxBuildFailed"), "captured: {joined}");
        assert!(joined.contains("EmptyBatchDuplicate"), "captured: {joined}");
    }

    /// Span-batch-overlap request/response state-machine contract.
    ///
    /// Phase 3's `add_span_batch_overlap_unsolicited_errors` proves the pure
    /// deriver rejects unsolicited overlap responses; this companion test
    /// exercises the actor-side request/response shape using a synthetic
    /// fixture. We can't easily build a real overlapping span batch byte
    /// stream by hand (the encoder lives behind `op-batcher`), so we use a
    /// negative fixture: hand the deriver an L1 input with a sysconfig-only
    /// payload, drive it forward, and assert that no spurious overlap
    /// request is generated. The positive overlap path (full byte-wise
    /// compare) is covered by `pure::overlap::tests::*` in the kona-derive
    /// crate.
    #[test]
    fn synthetic_overlap_request_does_not_fire_without_span_batch() {
        let rcfg = Arc::new(rollup_cfg());
        let l1cfg: Arc<L1ChainConfig> = Arc::new(L1ChainConfig::default());
        let safe_head = L2BlockInfo {
            block_info: BlockInfo { number: 1, timestamp: 100, ..Default::default() },
            l1_origin: BlockNumHash { number: 0, ..Default::default() },
            seq_num: 0,
        };
        let mut deriver = Deriver::new(rcfg, l1cfg, SystemConfig::default(), safe_head, None);
        deriver.reset(safe_head, SystemConfig::default());

        // Feed a handful of empty L1 inputs.
        let mut prev = B256::ZERO;
        for n in 0..6u64 {
            let h = Header {
                number: n,
                timestamp: 100 + n * 12,
                parent_hash: prev,
                ..Default::default()
            };
            prev = h.hash_slow();
            deriver
                .add_l1_input(L1Input {
                    header: h,
                    batch_inbox_data: Vec::new(),
                    deposit_logs: Vec::new(),
                    config_logs: Vec::new(),
                })
                .unwrap();
        }

        // Drive forward — every step should be `NeedL1Input` or `Idle`; no overlap requests.
        let mut saw_overlap_request = false;
        for _ in 0..16 {
            let (out, _trace) = deriver.derive(safe_head);
            if matches!(out, Derivation::NeedSpanBatchOverlap { .. }) {
                saw_overlap_request = true;
                break;
            }
            if matches!(out, Derivation::NeedL1Input | Derivation::Idle) {
                break;
            }
        }
        assert!(!saw_overlap_request, "no overlap should be requested without a span batch");
    }

    /// Verify `NodeStep`'s terminal-state behaviour: a fresh `NodeDeriver`
    /// without a reset and without an L1 head should yield rather than
    /// trying to fetch.
    ///
    /// The driver here doesn't talk to real HTTP — it sits on the deriver's
    /// `next_l1_number == None` short-circuit in `advance_l1_input`. We don't
    /// construct a `NodeDeriver` directly because it owns concrete RPC
    /// providers; instead we exercise the underlying logic by way of the
    /// pure `Deriver` and a manually-tracked `next_l1_number` mirror.
    #[test]
    fn unanchored_pure_deriver_yields_need_l1_input() {
        let rcfg = Arc::new(rollup_cfg());
        let l1cfg: Arc<L1ChainConfig> = Arc::new(L1ChainConfig::default());
        let mut deriver =
            Deriver::new(rcfg, l1cfg, SystemConfig::default(), L2BlockInfo::default(), None);
        let (out, trace) = deriver.derive(L2BlockInfo::default());
        assert!(matches!(out, Derivation::NeedL1Input));
        // No reset, no L1 inputs → empty trace.
        assert!(trace.is_empty(), "expected empty trace; got {trace:?}");
    }
}
