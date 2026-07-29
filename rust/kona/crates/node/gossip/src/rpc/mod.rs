//! RPC API types and request handling for P2P administration.
//!
//! This module provides a JSON-RPC compatible interface for monitoring and controlling
//! the P2P networking stack. It offers comprehensive visibility into network status,
//! peer management, and operational metrics.
//!
//! ## API Categories
//!
//! ### Peer Information
//! - [`PeerInfo`]: Comprehensive peer details including connection status and capabilities
//! - [`PeerStats`]: Connection statistics and performance metrics
//! - [`PeerCount`]: Current peer count across different connection states
//! - [`PeerDump`]: Complete dump of all known peers
//!
//! ### Scoring and Quality
//! - [`PeerScores`]: Peer reputation scores used for mesh maintenance
//! - [`GossipScores`]: GossipSub-specific scoring metrics
//! - [`TopicScores`]: Per-topic scoring information
//! - [`ReqRespScores`]: Request-response protocol scoring
//!
//! ### Connection Management
//! - [`Connectedness`]: Peer connection state enumeration
//! - [`Direction`]: Connection direction (inbound/outbound)
//!
//! ## RPC Methods
//!
//! The [`P2pRpcRequest`] enum defines all available RPC methods, including:
//! - Node identity and status queries
//! - Peer listing and statistics
//! - Connection management (block/unblock peers)
//! - Network address filtering
//! - Discovery table inspection
//!
//! ## Usage
//!
//! The RPC interface is designed to be compatible with existing OP Stack tooling
//! and monitoring systems, providing the same API surface as the reference `op-node`
//! implementation where applicable.
//!
//! ## Security Considerations
//!
//! Administrative RPC methods should be exposed only to authorized clients, as they
//! can affect network connectivity and peer relationships. Consider implementing
//! appropriate authentication and access controls in production deployments.

mod request;
pub use request::P2pRpcRequest;

mod types;
pub use types::{
    Connectedness, Direction, GossipScores, PeerCount, PeerDump, PeerInfo, PeerScores, PeerStats,
    ReqRespScores, TopicScores,
};
