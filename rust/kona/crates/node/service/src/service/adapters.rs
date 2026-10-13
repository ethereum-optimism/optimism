//! Connects actor handles to dependency traits.

use alloy_rpc_types_engine::PayloadId;
use async_trait::async_trait;
use discv5::Enr;
use kona_node_actors::{
    EngineClientError, EngineClientResult, SequencerEngineClient, derivation, discovery, engine,
    engine::{Consolidation, EngineState, FinalizeBlockId},
    finality, network, sequencer, signer,
};
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use libp2p::Multiaddr;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{collections::HashSet, time::Duration};
use tokio::sync::{
    mpsc::error::{SendError, TrySendError},
    watch,
};

/// Adapts the engine handle to the traits of the actors that use the engine.
#[derive(Debug, Clone)]
pub(super) struct Engine {
    handle: engine::Handle,
    /// The engine's resets after this adapter was created.
    resets: watch::Receiver<EngineState>,
}

impl Engine {
    /// Creates the adapter, which sees the engine's resets from now on.
    pub(super) fn new(handle: engine::Handle) -> Self {
        Self { resets: handle.resets(), handle }
    }
}

#[async_trait]
impl SequencerEngineClient for Engine {
    async fn reset_engine_forkchoice(&self) -> EngineClientResult<()> {
        info!(target: "sequencer", "Sending reset request to engine.");
        let state = self.handle.reset().await?;
        info!(target: "sequencer", safe_head = ?state.safe_head(), "Engine reset successfully.");
        Ok(())
    }

    async fn start_build_block(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<PayloadId> {
        trace!(target: "sequencer", "Sending start build request to engine.");
        self.handle.start_build(attributes).await.inspect(|payload_id| {
            trace!(target: "sequencer", ?payload_id, "Start build request successfully.")
        })
    }

    async fn seal_and_canonicalize_block(
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

    async fn get_unsafe_head(&self) -> EngineClientResult<L2BlockInfo> {
        Ok(self.handle.state().borrow().unsafe_head())
    }
}

#[async_trait]
impl derivation::Engine for Engine {
    async fn reset(&self) -> EngineClientResult<EngineState> {
        info!(target: "derivation", "Sending reset request to engine.");
        self.handle.reset_if_el_synced().await.inspect(
            |state| info!(target: "derivation", safe_head = ?state.safe_head(), "Engine reset successfully."),
        )
    }

    async fn consolidate(
        &self,
        attributes: OpAttributesWithParent,
    ) -> EngineClientResult<Consolidation> {
        trace!(target: "derivation", ?attributes, "Sending derived attributes to engine.");
        self.handle.consolidate(attributes).await
    }

    async fn next_reset(&mut self) -> EngineClientResult<EngineState> {
        self.resets.changed().await.map_err(|_| {
            EngineClientError::RequestError("engine reset channel closed".to_string())
        })?;
        Ok(*self.resets.borrow_and_update())
    }

    async fn finalize(&self, block_id: FinalizeBlockId) -> EngineClientResult<()> {
        trace!(target: "derivation", ?block_id, "Sending finalized L2 block id to engine.");
        self.handle.finalize(block_id).await
    }

    async fn follow_safe_head(&self, safe_head: L2BlockInfo) -> EngineClientResult<()> {
        trace!(target: "derivation", ?safe_head, "Sending delegated safe head to engine.");
        self.handle.follow_safe_head(safe_head).await
    }
}

#[async_trait]
impl finality::Engine for Engine {
    async fn finalize(&self, block_id: FinalizeBlockId) -> EngineClientResult<()> {
        trace!(target: "finality", ?block_id, "Sending finalized L2 block id to engine.");
        self.handle.finalize(block_id).await
    }
}

#[async_trait]
impl network::Engine for Engine {
    async fn insert_unsafe_block(
        &self,
        block: OpExecutionPayloadEnvelope,
    ) -> EngineClientResult<()> {
        trace!(target: "network", ?block, "Sending unsafe block to engine.");
        self.handle.insert_unsafe_block(block).await
    }
}

#[derive(Debug, Clone)]
pub(super) struct Discovery(pub(super) discovery::Handle);

#[async_trait]
impl network::Discovery for Discovery {
    type Error = discovery::HandleError;

    async fn ban(&self, addrs: HashSet<Multiaddr>, duration: Duration) -> Result<(), Self::Error> {
        self.0.ban(addrs, duration).await
    }
}

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

#[derive(Debug, Clone)]
pub(super) struct Signer(pub(super) signer::Handle);

#[async_trait]
impl sequencer::Signer for Signer {
    type Error = tokio::sync::mpsc::error::SendError<OpExecutionPayloadEnvelope>;

    async fn send(&self, payload: OpExecutionPayloadEnvelope) -> Result<(), Self::Error> {
        self.0.send(payload).await
    }

    fn has_capacity(&self) -> bool {
        self.0.has_capacity()
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
        let builder = engine::Builder::new(engine::Capacity::try_from(1).unwrap());
        let mut engine = Engine::new(builder.handle());
        drop(builder);
        assert!(matches!(
            derivation::Engine::finalize(&engine, FinalizeBlockId::ByNumber(1)).await,
            Err(EngineClientError::RequestError(_))
        ));
        assert!(matches!(
            derivation::Engine::next_reset(&mut engine).await,
            Err(EngineClientError::RequestError(_))
        ));
    }

    #[tokio::test]
    async fn signer_preserves_backpressure_and_reports_closed_channels() {
        let builder = signer::Builder::new(signer::Capacity::try_from(1).unwrap());
        let signer = Signer(builder.handle());
        let payload = OpExecutionPayloadEnvelope::V1(ExecutionPayloadV1 {
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
        });
        assert!(signer.has_capacity());
        signer.send(payload.clone()).await.unwrap();
        assert!(!signer.has_capacity());

        let send = signer.send(payload.clone());
        tokio::pin!(send);
        tokio::select! {
            biased;
            result = &mut send => panic!("full queue accepted a second payload: {result:?}"),
            _ = ready(()) => {}
        }
        drop(builder);
        assert_eq!(send.await.unwrap_err().0, payload);
    }
}
