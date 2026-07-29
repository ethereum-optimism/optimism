//! GossipSub-based consensus layer networking for Optimism.
//!
//! This module implements the networking layer for Optimism consensus using libp2p's GossipSub
//! protocol. It handles the propagation and validation of OP Stack network payload messages
//! across the network mesh.
//!
//! ## Key Components
//!
//! - [`GossipDriver`]: The main driver that manages the libp2p swarm and event handling
//! - [`Behaviour`]: Custom libp2p behavior combining GossipSub, Ping, and Identify protocols
//! - [`BlockHandler`]: Validates and processes incoming block payloads
//! - [`ConnectionGater`]: Implements sophisticated connection management and rate limiting
//! - [`Event`]: High-level events emitted by the gossip system
//!
//! ## Network Architecture
//!
//! The gossip network uses a mesh topology where nodes maintain connections to a subset
//! of peers and propagate messages through the mesh. This provides efficient message
//! delivery with built-in redundancy and fault tolerance.
//!
//! ## Message Validation
//!
//! All incoming messages are validated through a multi-stage process:
//! 1. Basic structure validation
//! 2. Signature verification
//! 3. Content validation through [`BlockHandler`]
//! 4. Duplicate detection and caching
//!
//! ## Connection Management
//!
//! The module implements intelligent connection management through:
//! - Rate limiting for incoming connections
//! - IP-based filtering and subnet blocking
//! - Peer protection mechanisms
//! - Automatic connection pruning
//!
//! [`OpNetworkPayloadEnvelope`]: op_alloy_rpc_types_engine::OpNetworkPayloadEnvelope

mod behaviour;
pub use behaviour::{Behaviour, BehaviourError};

mod config;
pub use config::{
    DEFAULT_MESH_D, DEFAULT_MESH_DHI, DEFAULT_MESH_DLAZY, DEFAULT_MESH_DLO,
    GLOBAL_VALIDATE_THROTTLE, GOSSIP_HEARTBEAT, MAX_GOSSIP_SIZE, MAX_OUTBOUND_QUEUE,
    MAX_VALIDATE_QUEUE, MIN_GOSSIP_SIZE, PEER_SCORE_INSPECT_FREQUENCY, SEEN_MESSAGES_TTL,
    default_config, default_config_builder,
};

mod gate;
pub use gate::ConnectionGate; // trait

mod gater;
pub use gater::{
    ConnectionGater, // implementation
    DialInfo,
    GaterConfig,
};

mod builder;
pub use builder::GossipDriverBuilder;

mod error;
pub use error::{DialError, GossipDriverBuilderError, HandlerEncodeError, PublishError};

mod event;
pub use event::Event;

mod handler;
pub use handler::{BlockHandler, Handler};

mod driver;
pub use driver::GossipDriver;

mod block_validity;
pub use block_validity::BlockInvalidError;

#[cfg(test)]
pub(crate) use block_validity::tests::*;
