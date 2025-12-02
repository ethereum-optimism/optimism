//! Error types for the gossip networking module.

use crate::BehaviourError;
use derive_more::From;
use libp2p::{Multiaddr, PeerId};
use std::net::IpAddr;
use thiserror::Error;

/// Error encountered when publishing a payload to the gossip network.
///
/// Represents failures in the payload publishing pipeline, including
/// network-level publishing errors and payload encoding issues.
#[derive(Debug, Error)]
pub enum PublishError {
    /// Failed to publish the payload via GossipSub protocol.
    ///
    /// This can occur due to network connectivity issues, mesh topology
    /// problems, or protocol-level errors in the libp2p stack.
    #[error("Failed to publish payload: {0}")]
    PublishError(#[from] libp2p::gossipsub::PublishError),

    /// Failed to encode the payload before publishing.
    ///
    /// Indicates an issue with serializing the payload data structure
    /// into the binary format expected by the network protocol.
    #[error("Failed to encode payload: {0}")]
    EncodeError(#[from] HandlerEncodeError),
}

/// Error encountered when encoding payloads in the block handler.
///
/// Represents failures in the payload serialization process, typically
/// occurring when converting OP Stack data structures to network format.
#[derive(Debug, Error)]
pub enum HandlerEncodeError {
    /// Failed to encode the OP Stack payload envelope.
    ///
    /// This error indicates issues with serializing the OP Stack network payload
    /// structure, which contains the consensus data being gossiped.
    #[error("Failed to encode payload: {0}")]
    PayloadEncodeError(#[from] op_alloy_rpc_types_engine::PayloadEnvelopeEncodeError),

    /// Attempted to publish to an unknown or unsubscribed topic.
    ///
    /// This error occurs when trying to publish to a GossipSub topic that
    /// is not recognized or that the node is not subscribed to.
    #[error("Unknown topic: {0}")]
    UnknownTopic(libp2p::gossipsub::TopicHash),
}

/// An error type for the [`crate::GossipDriverBuilder`].
#[derive(Debug, Clone, PartialEq, Eq, From, Error)]
pub enum GossipDriverBuilderError {
    /// A TCP error.
    #[error("TCP error")]
    TcpError,
    /// An error when setting the behaviour on the swarm builder.
    #[error("error setting behaviour on swarm builder")]
    WithBehaviourError,
    /// An error when building the gossip behaviour.
    #[error("error building gossip behaviour")]
    BehaviourError(BehaviourError),
    /// An error when setting up the sync request/response protocol.
    #[error("error setting up sync request/response protocol")]
    SetupSyncReqRespError,
    /// The sync request/response protocol has already been accepted.
    #[error("sync request/response protocol already accepted")]
    SyncReqRespAlreadyAccepted,
}

/// An error type representing reasons why a peer cannot be dialed.
#[derive(Debug, Clone, Error)]
pub enum DialError {
    /// Failed to extract PeerId from Multiaddr.
    #[error("Failed to extract PeerId from Multiaddr: {addr}")]
    InvalidMultiaddr {
        /// The multiaddress that failed to be parsed or does not contain a valid PeerId component
        addr: Multiaddr,
    },
    /// Already dialing this peer.
    #[error("Already dialing peer: {peer_id}")]
    AlreadyDialing {
        /// The PeerId of the peer that is already being dialed
        peer_id: PeerId,
    },
    /// Dial threshold reached for this peer.
    #[error("Dial threshold reached for peer: {addr}")]
    ThresholdReached {
        /// The multiaddress of the peer that has reached the maximum dial attempts
        addr: Multiaddr,
    },
    /// Peer is blocked.
    #[error("Peer is blocked: {peer_id}")]
    PeerBlocked {
        /// The PeerId of the peer that is on the blocklist
        peer_id: PeerId,
    },
    /// Failed to extract IP address from Multiaddr.
    #[error("Failed to extract IP address from Multiaddr: {addr}")]
    InvalidIpAddress {
        /// The multiaddress that does not contain a valid IP address component
        addr: Multiaddr,
    },
    /// IP address is blocked.
    #[error("IP address is blocked: {ip}")]
    AddressBlocked {
        /// The IP address that is on the blocklist
        ip: IpAddr,
    },
    /// IP address is in a blocked subnet.
    #[error("IP address {ip} is in a blocked subnet")]
    SubnetBlocked {
        /// The IP address that belongs to a blocked subnet range
        ip: IpAddr,
    },
}
