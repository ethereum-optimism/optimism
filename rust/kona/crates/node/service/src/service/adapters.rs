//! Connects actor handles to dependency traits.

use async_trait::async_trait;
use kona_node_actors::{DerivationActorRequest, l1_watcher, sequencer, signer};
use kona_protocol::BlockInfo;
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use tokio::sync::mpsc;

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

/// Queues L1 observations until derivation has its own handle.
#[derive(Debug, Clone)]
pub(super) struct Derivation(pub(super) mpsc::Sender<DerivationActorRequest>);

#[async_trait]
impl l1_watcher::Derivation for Derivation {
    type Error = mpsc::error::SendError<DerivationActorRequest>;

    async fn send_finalized_l1_block(&self, block: BlockInfo) -> Result<(), Self::Error> {
        self.0.send(DerivationActorRequest::ProcessFinalizedL1Block(Box::new(block))).await
    }

    async fn send_new_l1_head(&self, block: BlockInfo) -> Result<(), Self::Error> {
        self.0.send(DerivationActorRequest::ProcessL1HeadUpdateRequest(Box::new(block))).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rpc_types_engine::ExecutionPayloadV1;
    use l1_watcher::Derivation as _;
    use sequencer::Signer as _;
    use std::future::ready;

    #[tokio::test]
    async fn derivation_forwards_observations_and_returns_closed_channel_errors() {
        let (tx, mut rx) = mpsc::channel(2);
        let derivation = Derivation(tx);
        let block = BlockInfo { number: 42, ..Default::default() };
        derivation.send_new_l1_head(block).await.unwrap();
        derivation.send_finalized_l1_block(block).await.unwrap();
        assert!(
            matches!(rx.recv().await.unwrap(), DerivationActorRequest::ProcessL1HeadUpdateRequest(received) if *received == block)
        );
        assert!(
            matches!(rx.recv().await.unwrap(), DerivationActorRequest::ProcessFinalizedL1Block(received) if *received == block)
        );
        drop(rx);
        assert!(
            matches!(derivation.send_new_l1_head(block).await.unwrap_err().0, DerivationActorRequest::ProcessL1HeadUpdateRequest(received) if *received == block)
        );
        assert!(
            matches!(derivation.send_finalized_l1_block(block).await.unwrap_err().0, DerivationActorRequest::ProcessFinalizedL1Block(received) if *received == block)
        );
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
