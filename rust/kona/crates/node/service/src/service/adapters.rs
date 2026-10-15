//! Connects actor handles to dependency traits.

use alloy_rpc_types_engine::PayloadId;
use async_trait::async_trait;
use discv5::Enr;
use kona_node_actors::{
    EngineClientError, EngineClientResult, SequencerEngineClient, derivation, discovery, engine,
    engine::{Derived, Safe},
    finality, l1, l1_signer_updater, network, sequencer, signer,
};
use kona_protocol::{BlockInfo, L2BlockInfo, OpAttributesWithParent};
use libp2p::Multiaddr;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{collections::HashSet, time::Duration};
use tokio::sync::mpsc::error::{SendError, TrySendError};

/// Adapts the engine handle to the traits of the actors that use the engine.
#[derive(Debug, Clone)]
pub(super) struct Engine {
    handle: engine::Handle,
}

impl Engine {
    /// Creates the adapter.
    pub(super) const fn new(handle: engine::Handle) -> Self {
        Self { handle }
    }
}

#[async_trait]
impl SequencerEngineClient for Engine {
    async fn start_build_block(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<PayloadId> {
        trace!(target: "sequencer", "Sending start build request to engine.");
        self.handle.start_build(attributes).await.inspect(|payload_id| {
            trace!(target: "sequencer", ?payload_id, "Start build request successfully.")
        })
    }

    async fn seal_block(
        &self,
        payload_id: PayloadId,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<OpExecutionPayloadEnvelope> {
        trace!(target: "sequencer", ?attributes, "Sending seal request to engine.");
        self.handle
            .seal(payload_id, attributes)
            .await
            .inspect(|payload| trace!(target: "sequencer", ?payload, "Seal succeeded."))
            .inspect_err(|err| {
                if matches!(err, EngineClientError::SealError(_)) {
                    info!(target: "sequencer", ?err, "Seal failed.");
                }
            })
    }

    async fn canonicalize_block(
        &self,
        payload: OpExecutionPayloadEnvelope,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<()> {
        trace!(target: "sequencer", "Sending canonicalize request to engine.");
        self.handle.canonicalize(payload, attributes).await.inspect_err(|err| {
            info!(target: "sequencer", ?err, "Canonicalize failed.");
        })
    }

    async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo> {
        Ok(self.handle.state().borrow().unsafe_head())
    }
}

impl network::Engine for Engine {
    fn try_insert_unsafe_block(
        &self,
        block: OpExecutionPayloadEnvelope,
    ) -> Result<(), TrySendError<()>> {
        trace!(target: "network", ?block, "Sending unsafe block to engine.");
        self.handle.try_insert_unsafe_block(block)
    }
}

/// What decides the node's safe head: derivation from L1, or a trusted source. The engine drives
/// it, and the rollup RPC reads its progress.
#[derive(Debug, Clone)]
pub(super) enum Safety {
    /// Derivation from L1.
    Untrusted(derivation::untrusted::Handle),
    /// Derivation delegated to a trusted source, which needs no resets or flushes.
    Trusted(Box<derivation::trusted::Handle>),
}

#[async_trait]
impl engine::Safety for Safety {
    type Error = derivation::Stopped;

    async fn next(&mut self, safe_head: L2BlockInfo) -> Result<Derived, derivation::Stopped> {
        match self {
            Self::Untrusted(handle) => engine::Safety::next(handle, safe_head).await,
            Self::Trusted(handle) => handle.next_safe(safe_head).await.map(Derived::Safe),
        }
    }

    async fn reset(&mut self, safe: L2BlockInfo) -> Result<(), derivation::Stopped> {
        match self {
            Self::Untrusted(handle) => engine::Safety::reset(handle, safe).await,
            Self::Trusted(_) => Ok(()),
        }
    }

    async fn flush(&mut self) -> Result<(), derivation::Stopped> {
        match self {
            Self::Untrusted(handle) => engine::Safety::flush(handle).await,
            Self::Trusted(_) => Ok(()),
        }
    }
}

impl kona_rpc::Derivation for Safety {
    fn current_l1(&self) -> BlockInfo {
        match self {
            Self::Untrusted(handle) => handle.current_l1(),
            Self::Trusted(handle) => handle.current_l1(),
        }
    }
}

/// What decides the node's finalized head: the finality actor, from the L1 blocks the safe heads
/// were derived from, or a trusted source.
#[derive(Debug, Clone)]
pub(super) enum Finality {
    /// The finality actor.
    Untrusted(finality::Handle),
    /// A trusted source.
    Trusted(Box<derivation::trusted::Handle>),
}

#[async_trait]
impl engine::Finality for Finality {
    type Error = finality::Stopped;

    async fn next(&mut self, safe: Safe) -> Result<L2BlockInfo, finality::Stopped> {
        match self {
            Self::Untrusted(handle) => engine::Finality::next(handle, safe).await,
            Self::Trusted(handle) => {
                engine::Finality::next(handle.as_mut(), safe).await.map_err(|_| finality::Stopped)
            }
        }
    }
}

/// Adapts the L1 handle to the traits of the actors that follow L1.
///
/// The adapter remembers the last head and finalized block it returned, and waits for the next
/// ones, so each actor needs its own.
#[derive(Debug)]
pub(super) struct L1 {
    handle: l1::Handle,
    head: BlockInfo,
    finalized: BlockInfo,
}

impl L1 {
    /// Creates the adapter, which returns the current head and finalized block first.
    pub(super) fn new(handle: l1::Handle) -> Self {
        Self { handle, head: BlockInfo::default(), finalized: BlockInfo::default() }
    }
}

impl l1_signer_updater::L1 for L1 {
    async fn next_head(&mut self) -> Option<BlockInfo> {
        self.head = self.handle.next_head(self.head).await.ok()?;
        Some(self.head)
    }
}

#[async_trait]
impl finality::L1 for L1 {
    async fn next_finalized(&mut self) -> Option<BlockInfo> {
        self.finalized = self.handle.next_finalized(self.finalized).await.ok()?;
        Some(self.finalized)
    }
}

/// The discovery actor the network bans nodes through, or `None` for a node that runs no
/// discovery, which has no discovery nodes to ban.
#[derive(Debug, Clone)]
pub(super) struct Discovery(pub(super) Option<discovery::Handle>);

impl network::Discovery for Discovery {
    fn try_ban(
        &self,
        addrs: HashSet<Multiaddr>,
        duration: Duration,
    ) -> Result<(), TrySendError<()>> {
        self.0.as_ref().map_or(Ok(()), |discovery| discovery.try_ban(addrs, duration))
    }
}

/// The network actor, which discovery dials peers through, and the signer gossips blocks through.
#[derive(Debug, Clone)]
pub(super) struct Network(pub(super) network::Handle);

impl discovery::Network for Network {
    fn dial(&self, enr: Enr) -> Result<(), TrySendError<Enr>> {
        self.0.dial(enr)
    }
}

#[async_trait]
impl signer::Network for Network {
    type Error = SendError<signer::Payload>;

    async fn publish(&self, payload: signer::Payload) -> Result<(), Self::Error> {
        self.0.publish(payload).await
    }
}

/// The signer actor the sequencer hands its blocks to for signing and gossip, or `None` for a node
/// without P2P, which drops them unsigned, having no peers to gossip them to.
#[derive(Debug, Clone)]
pub(super) struct Signer(pub(super) Option<signer::Handle>);

#[async_trait]
impl sequencer::Signer for Signer {
    type Error = signer::Stopped;

    async fn send(&self, payload: OpExecutionPayloadEnvelope) -> Result<(), Self::Error> {
        let Some(signer) = &self.0 else { return Ok(()) };
        signer.send(payload).await
    }

    async fn ready(&self) -> Result<(), Self::Error> {
        let Some(signer) = &self.0 else { return Ok(()) };
        signer.ready().await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rpc_types_engine::ExecutionPayloadV1;
    use sequencer::Signer as _;
    use std::future::ready;

    #[tokio::test]
    async fn engine_reports_a_closed_queue() {
        let builder = engine::Builder::new(engine::EngineState::default());
        let engine = Engine::new(builder.handle());
        drop(builder);
        assert!(matches!(
            engine
                .start_build_block(OpAttributesWithParent::new(
                    Default::default(),
                    L2BlockInfo::default(),
                    None,
                    false
                ))
                .await,
            Err(EngineClientError::RequestError(_))
        ));
    }

    /// Trusted safety needs no resets or flushes, and reports a stopped actor, as does trusted
    /// finality.
    #[tokio::test]
    async fn trusted_safety_needs_no_resets_or_flushes() {
        let builder = derivation::trusted::Builder::new();
        let mut safety = Safety::Trusted(Box::new(builder.handle()));
        let mut finality = Finality::Trusted(Box::new(builder.handle()));
        let block = L2BlockInfo::default();
        assert_eq!(kona_rpc::Derivation::current_l1(&safety), BlockInfo::default());
        engine::Safety::reset(&mut safety, block).await.unwrap();
        engine::Safety::flush(&mut safety).await.unwrap();
        drop(builder);
        assert!(matches!(engine::Safety::next(&mut safety, block).await, Err(derivation::Stopped)));
        let safe = Safe { head: block, source: BlockInfo::default() };
        assert!(matches!(
            engine::Finality::next(&mut finality, safe).await,
            Err(finality::Stopped)
        ));
    }

    /// Untrusted safety and finality report stopped actors on every request.
    #[tokio::test]
    async fn untrusted_safety_and_finality_report_stopped_actors() {
        let builder = derivation::untrusted::Builder::new();
        let mut safety = Safety::Untrusted(builder.handle());
        let block = L2BlockInfo::default();
        assert_eq!(kona_rpc::Derivation::current_l1(&safety), BlockInfo::default());
        drop(builder);
        assert!(matches!(
            engine::Safety::reset(&mut safety, block).await,
            Err(derivation::Stopped)
        ));
        assert!(matches!(engine::Safety::flush(&mut safety).await, Err(derivation::Stopped)));
        assert!(matches!(engine::Safety::next(&mut safety, block).await, Err(derivation::Stopped)));

        let builder = finality::Builder::new();
        let mut finality = Finality::Untrusted(builder.handle());
        drop(builder);
        let safe = Safe { head: block, source: BlockInfo::default() };
        assert!(matches!(
            engine::Finality::next(&mut finality, safe).await,
            Err(finality::Stopped)
        ));
    }

    fn payload() -> OpExecutionPayloadEnvelope {
        OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1 {
            parent_hash: Default::default(),
            fee_recipient: Default::default(),
            state_root: Default::default(),
            receipts_root: Default::default(),
            logs_bloom: Default::default(),
            prev_randao: Default::default(),
            block_number: 1,
            gas_limit: 30_000_000,
            gas_used: 0,
            timestamp: 1,
            extra_data: Default::default(),
            base_fee_per_gas: Default::default(),
            block_hash: Default::default(),
            transactions: vec![],
        })
    }

    #[tokio::test]
    async fn signer_preserves_backpressure_and_reports_closed_channels() {
        let builder = signer::Builder::new();
        let signer = Signer(Some(builder.handle()));
        let payload = payload();
        signer.ready().await.unwrap();
        signer.send(payload.clone()).await.unwrap();
        let waiting = signer.ready();
        tokio::pin!(waiting);
        tokio::select! {
            biased;
            result = &mut waiting => panic!("a full queue reported room: {result:?}"),
            _ = ready(()) => {}
        }

        let send = signer.send(payload.clone());
        tokio::pin!(send);
        tokio::select! {
            biased;
            result = &mut send => panic!("full queue accepted a second payload: {result:?}"),
            _ = ready(()) => {}
        }
        drop(builder);
        assert_eq!(send.await, Err(signer::Stopped));
        assert_eq!(waiting.await, Err(signer::Stopped));
    }

    /// Without P2P, the sequencer's blocks are dropped unsigned, so it never waits on a signer.
    #[tokio::test]
    async fn without_a_signer_blocks_are_dropped() {
        let signer = Signer(None);
        for _ in 0..3 {
            signer.ready().await.unwrap();
            signer.send(payload()).await.unwrap();
        }
    }
}
