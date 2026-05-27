use alloy_primitives::Address;
use async_trait::async_trait;
use kona_gossip::P2pRpcRequest;
use kona_rpc::NetworkAdminQuery;
use kona_sources::BlockSignerError;
use libp2p::TransportError;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpNetworkPayloadEnvelope};
use thiserror::Error;
use tokio::{
    self, select,
    sync::mpsc::{self, UnboundedReceiver, UnboundedSender},
};

use crate::{
    NetworkEngineClient, NodeActor,
    actors::network::{
        driver::NetworkDriverError, error::NetworkBuilderError, handler::NetworkHandler,
    },
};

/// The network actor handles two core networking components of the rollup node:
/// - *discovery*: Peer discovery over UDP using discv5.
/// - *gossip*: Block gossip over TCP using libp2p.
#[derive(Debug)]
pub struct NetworkActor<NetworkEngineClient_: NetworkEngineClient> {
    /// The live libp2p [`NetworkHandler`].
    handler: NetworkHandler,
    /// A channel to receive the unsafe block signer address.
    unsafe_block_signer_rx: mpsc::Receiver<Address>,
    /// A channel to receive p2p RPC requests.
    p2p_rpc_rx: mpsc::Receiver<P2pRpcRequest>,
    /// A channel to receive admin RPC queries.
    admin_query_rx: mpsc::Receiver<NetworkAdminQuery>,
    /// A channel to receive unsafe blocks and send them through the gossip layer.
    publish_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    /// A client to use to interact with the engine actor.
    engine_client: NetworkEngineClient_,
    // Purely-internal channel: loops gossip-swarm events back into this actor's own select. It
    // never crosses an actor boundary, so it lives here rather than being injected.
    unsafe_block_tx: UnboundedSender<OpExecutionPayloadEnvelope>,
    unsafe_block_rx: UnboundedReceiver<OpExecutionPayloadEnvelope>,
}

impl<NetworkEngineClient_: NetworkEngineClient> NetworkActor<NetworkEngineClient_> {
    /// Constructs a new [`NetworkActor`].
    ///
    /// `handler` must already be live — i.e. the libp2p swarm it wraps must already have been
    /// built and started — before being passed in. Passing an unstarted handler will cause
    /// `step()` to hang or fail on its first poll of the gossip swarm. Keeping the constructor
    /// sync and treating the "is this live?" invariant as the caller's responsibility is the
    /// deliberate trade-off over an `init()`-style trait method.
    pub fn new(
        engine_client: NetworkEngineClient_,
        handler: NetworkHandler,
        unsafe_block_signer_rx: mpsc::Receiver<Address>,
        p2p_rpc_rx: mpsc::Receiver<P2pRpcRequest>,
        admin_query_rx: mpsc::Receiver<NetworkAdminQuery>,
        publish_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    ) -> Self {
        let (unsafe_block_tx, unsafe_block_rx) = mpsc::unbounded_channel();
        Self {
            handler,
            unsafe_block_signer_rx,
            p2p_rpc_rx,
            admin_query_rx,
            publish_rx,
            engine_client,
            unsafe_block_tx,
            unsafe_block_rx,
        }
    }
}

/// An error from the network actor.
#[derive(Debug, Error)]
pub enum NetworkActorError {
    /// Network builder error.
    #[error(transparent)]
    NetworkBuilder(#[from] NetworkBuilderError),
    /// Network driver error.
    #[error(transparent)]
    NetworkDriver(#[from] NetworkDriverError),
    /// Driver startup failed.
    #[error(transparent)]
    DriverStartup(#[from] TransportError<std::io::Error>),
    /// The network driver was missing its unsafe block receiver.
    #[error("Missing unsafe block receiver in network driver")]
    MissingUnsafeBlockReceiver,
    /// The network driver was missing its unsafe block signer sender.
    #[error("Missing unsafe block signer in network driver")]
    MissingUnsafeBlockSigner,
    /// Channel closed unexpectedly.
    #[error("Channel closed unexpectedly")]
    ChannelClosed,
    /// Failed to sign the payload.
    #[error("Failed to sign the payload: {0}")]
    FailedToSignPayload(#[from] BlockSignerError),
}

#[async_trait]
impl<NetworkEngineClient_: NetworkEngineClient + 'static> NodeActor
    for NetworkActor<NetworkEngineClient_>
{
    type Error = NetworkActorError;

    async fn step(&mut self) -> Result<(), Self::Error> {
        select! {
            block = self.unsafe_block_rx.recv() => {
                let Some(block) = block else {
                    error!(target: "node::p2p", "The unsafe block receiver channel has closed");
                    return Err(NetworkActorError::ChannelClosed);
                };

                if self.engine_client.send_unsafe_block(block).await.is_err() {
                    warn!(target: "network", "Failed to forward unsafe block to engine");
                    return Err(NetworkActorError::ChannelClosed);
                }
                Ok(())
            }
            unsafe_block_signer = self.unsafe_block_signer_rx.recv() => {
                let Some(unsafe_block_signer) = unsafe_block_signer else {
                    warn!(
                        target: "network",
                        "Found no unsafe block signer on receive"
                    );
                    return Err(NetworkActorError::ChannelClosed);
                };
                if self.handler.unsafe_block_signer_sender.send(unsafe_block_signer).is_err() {
                    warn!(
                        target: "network",
                        "Failed to send unsafe block signer to network handler",
                    );
                }
                Ok(())
            }
            Some(block) = self.publish_rx.recv(), if !self.publish_rx.is_closed() => {
                let timestamp = block.execution_payload.timestamp();
                let selector = |handler: &kona_gossip::BlockHandler| {
                    handler.topic(timestamp)
                };
                let Some(signer) = self.handler.signer.as_ref() else {
                    warn!(target: "net", "No local signer available to sign the payload");
                    return Ok(());
                };

                let chain_id = self.handler.discovery.chain_id;

                let sender_address = *self.handler.unsafe_block_signer_sender.borrow();

                let payload_hash = block.payload_hash();
                let signature = signer.sign_block(payload_hash, chain_id, sender_address).await?;

                let payload = OpNetworkPayloadEnvelope {
                    payload: block.execution_payload,
                    parent_beacon_block_root: block.parent_beacon_block_root,
                    signature,
                    payload_hash,
                };

                match self.handler.gossip.publish(selector, Some(payload)) {
                    Ok(id) => info!("Published unsafe payload | {:?}", id),
                    Err(e) => warn!("Failed to publish unsafe payload: {:?}", e),
                }
                Ok(())
            }
            event = self.handler.gossip.next() => {
                let Some(event) = event else {
                    error!(target: "node::p2p", "The gossip swarm stream has ended");
                    return Err(NetworkActorError::ChannelClosed);
                };

                if let Some(payload) = self.handler.gossip.handle_event(event)
                    && self.unsafe_block_tx.send(payload.into()).is_err()
                {
                    warn!(target: "node::p2p", "Failed to send unsafe block to network handler");
                }
                Ok(())
            }
            enr = self.handler.enr_receiver.recv() => {
                let Some(enr) = enr else {
                    error!(target: "node::p2p", "The enr receiver channel has closed");
                    return Err(NetworkActorError::ChannelClosed);
                };
                self.handler.gossip.dial(enr);
                Ok(())
            }
            _ = self.handler.peer_score_inspector.tick(), if self.handler.gossip.peer_monitoring.as_ref().is_some() => {
                self.handler.handle_peer_monitoring().await;
                Ok(())
            }
            Some(NetworkAdminQuery::PostUnsafePayload { payload }) = self.admin_query_rx.recv(), if !self.admin_query_rx.is_closed() => {
                debug!(target: "node::p2p", "Broadcasting unsafe payload from admin api");
                if self.unsafe_block_tx.send(payload).is_err() {
                    warn!(target: "node::p2p", "Failed to send unsafe block to network handler");
                }
                Ok(())
            }
            Some(req) = self.p2p_rpc_rx.recv(), if !self.p2p_rpc_rx.is_closed() => {
                req.handle(&mut self.handler.gossip, &self.handler.discovery);
                Ok(())
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::B256;
    use alloy_rpc_types_engine::{ExecutionPayloadV1, ExecutionPayloadV3};
    use alloy_signer::SignerSync;
    use alloy_signer_local::PrivateKeySigner;
    use arbitrary::Arbitrary;
    use op_alloy_rpc_types_engine::OpExecutionPayload;
    use rand::Rng;

    #[test]
    fn test_payload_signature_roundtrip_v1() {
        let mut bytes = [0u8; 4096];
        rand::rng().fill(bytes.as_mut_slice());

        let pubkey = PrivateKeySigner::random();
        let expected_address = pubkey.address();
        const CHAIN_ID: u64 = 1337;

        let block = OpExecutionPayloadEnvelope {
            execution_payload: OpExecutionPayload::V1(
                ExecutionPayloadV1::arbitrary(&mut arbitrary::Unstructured::new(&bytes)).unwrap(),
            ),
            parent_beacon_block_root: None,
        };

        let payload_hash = block.payload_hash();
        let signature = pubkey.sign_hash_sync(&payload_hash.signature_message(CHAIN_ID)).unwrap();
        let payload = OpNetworkPayloadEnvelope {
            payload: block.execution_payload,
            parent_beacon_block_root: block.parent_beacon_block_root,
            signature,
            payload_hash,
        };
        let encoded_payload = payload.encode_v1().unwrap();

        let decoded_payload = OpNetworkPayloadEnvelope::decode_v1(&encoded_payload).unwrap();

        let msg = decoded_payload.payload_hash.signature_message(CHAIN_ID);
        let msg_signer = decoded_payload.signature.recover_address_from_prehash(&msg).unwrap();

        assert_eq!(expected_address, msg_signer);
    }

    #[test]
    fn test_payload_signature_roundtrip_v3() {
        let mut bytes = [0u8; 4096];
        rand::rng().fill(bytes.as_mut_slice());

        let pubkey = PrivateKeySigner::random();
        let expected_address = pubkey.address();
        const CHAIN_ID: u64 = 1337;

        let block = OpExecutionPayloadEnvelope {
            execution_payload: OpExecutionPayload::V3(
                ExecutionPayloadV3::arbitrary(&mut arbitrary::Unstructured::new(&bytes)).unwrap(),
            ),
            parent_beacon_block_root: Some(B256::random()),
        };

        let payload_hash = block.payload_hash();
        let signature = pubkey.sign_hash_sync(&payload_hash.signature_message(CHAIN_ID)).unwrap();
        let payload = OpNetworkPayloadEnvelope {
            payload: block.execution_payload,
            parent_beacon_block_root: block.parent_beacon_block_root,
            signature,
            payload_hash,
        };
        let encoded_payload = payload.encode_v3().unwrap();

        let decoded_payload = OpNetworkPayloadEnvelope::decode_v3(&encoded_payload).unwrap();

        let msg = decoded_payload.payload_hash.signature_message(CHAIN_ID);
        let msg_signer = decoded_payload.signature.recover_address_from_prehash(&msg).unwrap();

        assert_eq!(expected_address, msg_signer);
    }
}
