use alloy_primitives::Address;
use async_trait::async_trait;
use kona_gossip::P2pRpcRequest;
use kona_rpc::NetworkAdminQuery;
use kona_sources::BlockSignerError;
use libp2p::TransportError;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpNetworkPayloadEnvelope};
use thiserror::Error;
use tokio::{self, select, sync::mpsc};
use tokio_util::sync::{CancellationToken, WaitForCancellationFuture};

use crate::{
    CancellableContext, NetworkEngineClient, NodeActor,
    actors::network::{
        builder::NetworkBuilder, driver::NetworkDriverError, error::NetworkBuilderError,
    },
};

/// The network actor handles two core networking components of the rollup node:
/// - *discovery*: Peer discovery over UDP using discv5.
/// - *gossip*: Block gossip over TCP using libp2p.
///
/// The network actor itself is a light wrapper around the [`NetworkBuilder`].
///
/// ## Example
///
/// ```rust,ignore
/// use kona_gossip::NetworkDriver;
/// use std::net::{IpAddr, Ipv4Addr, SocketAddr};
///
/// let chain_id = 10;
/// let signer = Address::random();
/// let socket = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 9099);
///
/// // Construct the `Network` using the builder.
/// // let mut driver = Network::builder()
/// //    .with_unsafe_block_signer(signer)
/// //    .with_chain_id(chain_id)
/// //    .with_gossip_addr(socket)
/// //    .build()
/// //    .unwrap();
///
/// // Construct the `NetworkActor` with the [`Network`].
/// // let actor = NetworkActor::new(driver);
/// ```
#[derive(Debug)]
pub struct NetworkActor<NetworkEngineClient_: NetworkEngineClient> {
    /// Network driver
    pub(super) builder: NetworkBuilder,
    /// The cancellation token, shared between all tasks.
    pub(super) cancellation_token: CancellationToken,
    /// A channel to receive the unsafe block signer address.
    pub(super) signer: mpsc::Receiver<Address>,
    /// Handler for p2p RPC Requests.
    pub(super) p2p_rpc: mpsc::Receiver<P2pRpcRequest>,
    /// A channel to receive admin rpc requests.
    pub(super) admin_rpc: mpsc::Receiver<NetworkAdminQuery>,
    /// A channel to receive unsafe blocks and send them through the gossip layer.
    pub(super) publish_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    /// A channel to use to interact with the engine actor.
    pub(super) engine_client: NetworkEngineClient_,
}

/// The inbound data for the network actor.
#[derive(Debug)]
pub struct NetworkInboundData {
    /// A channel to send the unsafe block signer address to the network actor.
    pub signer: mpsc::Sender<Address>,
    /// Handler for p2p RPC Requests sent to the network actor.
    pub p2p_rpc: mpsc::Sender<P2pRpcRequest>,
    /// Handler for admin RPC Requests.
    pub admin_rpc: mpsc::Sender<NetworkAdminQuery>,
    /// A channel to send unsafe blocks to the network actor.
    /// This channel should only be used by the sequencer actor/admin RPC api to forward their
    /// newly produced unsafe blocks to the network actor.
    pub gossip_payload_tx: mpsc::Sender<OpExecutionPayloadEnvelope>,
}

impl<NetworkEngineClient_: NetworkEngineClient> NetworkActor<NetworkEngineClient_> {
    /// Constructs a new [`NetworkActor`] given the [`NetworkBuilder`]
    pub fn new(
        engine_client: NetworkEngineClient_,
        cancellation_token: CancellationToken,
        driver: NetworkBuilder,
    ) -> (NetworkInboundData, Self) {
        let (signer_tx, signer_rx) = mpsc::channel(16);
        let (rpc_tx, rpc_rx) = mpsc::channel(1024);
        let (admin_rpc_tx, admin_rpc_rx) = mpsc::channel(1024);
        let (publish_tx, publish_rx) = tokio::sync::mpsc::channel(256);
        let actor = Self {
            builder: driver,
            cancellation_token,
            signer: signer_rx,
            p2p_rpc: rpc_rx,
            admin_rpc: admin_rpc_rx,
            publish_rx,
            engine_client,
        };
        let outbound_data = NetworkInboundData {
            signer: signer_tx,
            p2p_rpc: rpc_tx,
            admin_rpc: admin_rpc_tx,
            gossip_payload_tx: publish_tx,
        };
        (outbound_data, actor)
    }
}

impl<E: NetworkEngineClient> CancellableContext for NetworkActor<E> {
    fn cancelled(&self) -> WaitForCancellationFuture<'_> {
        self.cancellation_token.cancelled()
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
    type StartData = ();

    async fn start(mut self, _: Self::StartData) -> Result<(), Self::Error> {
        let mut handler = self.builder.build()?.start().await?;

        // New unsafe block channel.
        let (unsafe_block_tx, mut unsafe_block_rx) = tokio::sync::mpsc::unbounded_channel();

        loop {
            select! {
                _ = self.cancellation_token.cancelled() => {
                    info!(
                        target: "network",
                        "Received shutdown signal. Exiting network task."
                    );
                    return Ok(());
                }
                block = unsafe_block_rx.recv() => {
                    let Some(block) = block else {
                        error!(target: "node::p2p", "The unsafe block receiver channel has closed");
                        return Err(NetworkActorError::ChannelClosed);
                    };

                    if self.engine_client.send_unsafe_block(block).await.is_err() {
                        warn!(target: "network", "Failed to forward unsafe block to engine");
                        return Err(NetworkActorError::ChannelClosed);
                    }
                }
                signer = self.signer.recv() => {
                    let Some(signer) = signer else {
                        warn!(
                            target: "network",
                            "Found no unsafe block signer on receive"
                        );
                        return Err(NetworkActorError::ChannelClosed);
                    };
                    if handler.unsafe_block_signer_sender.send(signer).is_err() {
                        warn!(
                            target: "network",
                            "Failed to send unsafe block signer to network handler",
                        );
                    }
                }
                Some(block) = self.publish_rx.recv(), if !self.publish_rx.is_closed() => {
                    let timestamp = block.execution_payload.timestamp();
                    let selector = |handler: &kona_gossip::BlockHandler| {
                        handler.topic(timestamp)
                    };
                    let Some(signer) = handler.signer.as_ref() else {
                        warn!(target: "net", "No local signer available to sign the payload");
                        continue;
                    };

                    let chain_id = handler.discovery.chain_id;

                    let sender_address = *handler.unsafe_block_signer_sender.borrow();

                    let payload_hash = block.payload_hash();
                    let signature = signer.sign_block(payload_hash, chain_id, sender_address).await?;

                    let payload = OpNetworkPayloadEnvelope {
                        payload: block.execution_payload,
                        parent_beacon_block_root: block.parent_beacon_block_root,
                        signature,
                        payload_hash,
                    };

                    match handler.gossip.publish(selector, Some(payload)) {
                        Ok(id) => info!("Published unsafe payload | {:?}", id),
                        Err(e) => warn!("Failed to publish unsafe payload: {:?}", e),
                    }
                }
                event = handler.gossip.next() => {
                    let Some(event) = event else {
                        error!(target: "node::p2p", "The gossip swarm stream has ended");
                        return Err(NetworkActorError::ChannelClosed);
                    };

                    if let Some(payload) = handler.gossip.handle_event(event) {
                        if unsafe_block_tx.send(payload.into()).is_err() {
                            warn!(target: "node::p2p", "Failed to send unsafe block to network handler");
                        }
                    }
                },
                enr = handler.enr_receiver.recv() => {
                    let Some(enr) = enr else {
                        error!(target: "node::p2p", "The enr receiver channel has closed");
                        return Err(NetworkActorError::ChannelClosed);
                    };
                    handler.gossip.dial(enr);
                },
                _ = handler.peer_score_inspector.tick(), if handler.gossip.peer_monitoring.as_ref().is_some() => {
                    handler.handle_peer_monitoring().await;
                },
                Some(NetworkAdminQuery::PostUnsafePayload { payload }) = self.admin_rpc.recv(), if !self.admin_rpc.is_closed() => {
                    debug!(target: "node::p2p", "Broadcasting unsafe payload from admin api");
                    if unsafe_block_tx.send(payload).is_err() {
                        warn!(target: "node::p2p", "Failed to send unsafe block to network handler");
                    }
                },
                Some(req) = self.p2p_rpc.recv(), if !self.p2p_rpc.is_closed() => {
                    req.handle(&mut handler.gossip, &handler.discovery);
                },
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
