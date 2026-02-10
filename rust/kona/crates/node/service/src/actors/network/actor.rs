use alloy_primitives::Address;
use async_trait::async_trait;
use kona_gossip::P2pRpcRequest;
use kona_rpc::NetworkAdminQuery;
use kona_sources::BlockSignerError;
use libp2p::TransportError;
use op_alloy_rpc_types_engine::{OpExecutionPayloadEnvelope, OpNetworkPayloadEnvelope};
use thiserror::Error;
use tokio::{self, select, sync::mpsc};

use crate::{
    NetworkEngineClient, NodeActor,
    actors::network::{
        builder::NetworkBuilder, driver::NetworkDriverError, error::NetworkBuilderError,
        handler::NetworkHandler,
    },
};

/// Builder for the [`NetworkActor`].
#[derive(Debug)]
pub struct NetworkActorBuilder<E: NetworkEngineClient> {
    /// The engine client used to interact with the engine actor.
    pub engine_client: E,
    /// The network builder used to construct the network handler.
    pub network_builder: NetworkBuilder,
}

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
    /// A channel to receive the unsafe block signer address.
    signer: mpsc::Receiver<Address>,
    /// Handler for p2p RPC Requests.
    p2p_rpc: mpsc::Receiver<P2pRpcRequest>,
    /// A channel to receive admin rpc requests.
    admin_rpc: mpsc::Receiver<NetworkAdminQuery>,
    /// A channel to receive unsafe blocks and send them through the gossip layer.
    publish_rx: mpsc::Receiver<OpExecutionPayloadEnvelope>,
    /// A channel to use to interact with the engine actor.
    engine_client: NetworkEngineClient_,
    /// The network handler.
    handler: NetworkHandler,
    /// Sender for unsafe blocks received from gossip.
    unsafe_block_tx: mpsc::UnboundedSender<OpExecutionPayloadEnvelope>,
    /// Receiver for unsafe blocks received from gossip.
    unsafe_block_rx: mpsc::UnboundedReceiver<OpExecutionPayloadEnvelope>,
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
    type Builder = NetworkActorBuilder<NetworkEngineClient_>;
    type InboundData = NetworkInboundData;

    async fn init(builder: Self::Builder) -> Result<(Self::InboundData, Self), Self::Error> {
        let (signer_tx, signer_rx) = mpsc::channel(16);
        let (rpc_tx, rpc_rx) = mpsc::channel(1024);
        let (admin_rpc_tx, admin_rpc_rx) = mpsc::channel(1024);
        let (publish_tx, publish_rx) = mpsc::channel(256);

        let handler = builder.network_builder.build()?.start().await?;

        let (unsafe_block_tx, unsafe_block_rx) = mpsc::unbounded_channel();

        let actor = Self {
            signer: signer_rx,
            p2p_rpc: rpc_rx,
            admin_rpc: admin_rpc_rx,
            publish_rx,
            engine_client: builder.engine_client,
            handler,
            unsafe_block_tx,
            unsafe_block_rx,
        };
        let inbound_data = NetworkInboundData {
            signer: signer_tx,
            p2p_rpc: rpc_tx,
            admin_rpc: admin_rpc_tx,
            gossip_payload_tx: publish_tx,
        };
        Ok((inbound_data, actor))
    }

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
            }
            signer = self.signer.recv() => {
                let Some(signer) = signer else {
                    warn!(
                        target: "network",
                        "Found no unsafe block signer on receive"
                    );
                    return Err(NetworkActorError::ChannelClosed);
                };
                if self.handler.unsafe_block_signer_sender.send(signer).is_err() {
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
            },
            enr = self.handler.enr_receiver.recv() => {
                let Some(enr) = enr else {
                    error!(target: "node::p2p", "The enr receiver channel has closed");
                    return Err(NetworkActorError::ChannelClosed);
                };
                self.handler.gossip.dial(enr);
            },
            _ = self.handler.peer_score_inspector.tick(), if self.handler.gossip.peer_monitoring.as_ref().is_some() => {
                self.handler.handle_peer_monitoring().await;
            },
            Some(NetworkAdminQuery::PostUnsafePayload { payload }) = self.admin_rpc.recv(), if !self.admin_rpc.is_closed() => {
                debug!(target: "node::p2p", "Broadcasting unsafe payload from admin api");
                if self.unsafe_block_tx.send(payload).is_err() {
                    warn!(target: "node::p2p", "Failed to send unsafe block to network handler");
                }
            },
            Some(req) = self.p2p_rpc.recv(), if !self.p2p_rpc.is_closed() => {
                req.handle(&mut self.handler.gossip, &self.handler.discovery);
            },
        }
        Ok(())
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
