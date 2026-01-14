//! The types used in the p2p RPC API.

use core::net::IpAddr;
use derive_more::Display;

use alloy_primitives::{ChainId, map::HashMap};

/// The peer info.
///
/// <https://github.com/ethereum-optimism/optimism/blob/develop/op-node/p2p/rpc_api.go#L15>
#[derive(Clone, Default, Debug, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PeerInfo {
    /// The peer id.
    #[serde(rename = "peerID")]
    pub peer_id: String,
    /// The node id.
    #[serde(rename = "nodeID")]
    pub node_id: String,
    /// The user agent.
    pub user_agent: String,
    /// The protocol version.
    pub protocol_version: String,
    /// The enr for the peer.
    /// If the peer is not in the discovery table, this will not be set.
    #[serde(rename = "ENR")]
    #[serde(skip_serializing_if = "Option::is_none")]
    pub enr: Option<String>,
    /// The peer addresses.
    pub addresses: Vec<String>,
    /// Peer supported protocols
    pub protocols: Option<Vec<String>>,
    /// 0: "`NotConnected`",
    /// 1: "Connected",
    /// 2: "`CanConnect`" (gracefully disconnected)
    /// 3: "`CannotConnect`" (tried but failed)
    pub connectedness: Connectedness,
    /// 0: "Unknown",
    /// 1: "Inbound" (if the peer contacted us)
    /// 2: "Outbound" (if we connected to them)
    pub direction: Direction,
    /// Whether the peer is protected.
    pub protected: bool,
    /// The chain id.
    #[serde(rename = "chainID")]
    pub chain_id: ChainId,
    /// The peer latency in nanoseconds
    pub latency: u64,
    /// Whether the peer gossips
    pub gossip_blocks: bool,
    /// The peer scores.
    #[serde(rename = "scores")]
    pub peer_scores: PeerScores,
}

/// GossipSub topic-specific scoring metrics.
///
/// Tracks peer performance within specific gossip topics, used by the
/// GossipSub protocol to maintain mesh quality and route messages efficiently.
/// These scores influence peer selection for the gossip mesh topology.
///
/// Reference: <https://github.com/ethereum-optimism/optimism/blob/8dd17a7b114a7c25505cd2e15ce4e3d0f7e3f7c1/op-node/p2p/store/iface.go#L13>
#[derive(Clone, Default, Debug, Copy, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct TopicScores {
    /// Duration the peer has participated in the topic mesh.
    ///
    /// Longer participation indicates stability and commitment to the topic,
    /// contributing positively to the peer's mesh score.
    pub time_in_mesh: f64,

    /// Count of first-time message deliveries from this peer.
    ///
    /// Measures how often this peer is the first to deliver new messages,
    /// indicating their connectivity and responsiveness to the network.
    pub first_message_deliveries: f64,

    /// Count of messages delivered while in the mesh topology.
    ///
    /// Tracks consistent message forwarding behavior while the peer is
    /// an active participant in the mesh structure.
    pub mesh_message_deliveries: f64,

    /// Count of invalid or malicious messages from this peer.
    ///
    /// Penalizes peers that send invalid, duplicate, or malformed messages,
    /// helping maintain network health and preventing spam.
    pub invalid_message_deliveries: f64,
}

/// Comprehensive GossipSub scoring metrics for peer quality assessment.
///
/// Aggregates various scoring factors used by the GossipSub protocol to
/// evaluate peer quality and determine mesh topology. Higher scores indicate
/// more reliable and well-behaved peers.
///
/// Reference: <https://github.com/ethereum-optimism/optimism/blob/8dd17a7b114a7c25505cd2e15ce4e3d0f7e3f7c1/op-node/p2p/store/iface.go#L20C6-L20C18>
#[derive(Debug, Default, Clone, Copy, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct GossipScores {
    /// Aggregate score across all scoring dimensions.
    ///
    /// The final computed score that determines this peer's overall
    /// reputation in the gossip network.
    pub total: f64,

    /// Block-specific topic scores for consensus messages.
    ///
    /// Tracks peer behavior specifically for block gossip, which is
    /// the primary message type in OP Stack networks.
    pub blocks: TopicScores,

    /// Penalty for IP address colocation with other peers.
    ///
    /// Reduces scores for peers sharing IP addresses to prevent
    /// eclipse attacks and improve network decentralization.
    #[serde(rename = "IPColocationFactor")]
    pub ip_colocation_factor: f64,

    /// Penalty for problematic behavior patterns.
    ///
    /// Applied to peers exhibiting suspicious or harmful behavior
    /// that doesn't fit other specific scoring categories.
    pub behavioral_penalty: f64,
}

/// Request-response protocol scoring metrics.
///
/// Tracks peer performance in direct request-response interactions outside
/// of the gossip mesh, such as block synchronization requests.
///
/// Reference: <https://github.com/ethereum-optimism/optimism/blob/8dd17a7b114a7c25505cd2e15ce4e3d0f7e3f7c1/op-node/p2p/store/iface.go#L31C1-L35C2>
#[derive(Debug, Default, Clone, Copy, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ReqRespScores {
    /// Number of valid responses provided by this peer.
    ///
    /// Counts successful request-response exchanges where the peer
    /// provided correct and timely responses to queries.
    pub valid_responses: f64,

    /// Number of error responses or failed requests.
    ///
    /// Tracks cases where the peer returned errors, timeouts, or
    /// otherwise failed to properly respond to requests.
    pub error_responses: f64,
    /// Number of rejected payloads.
    pub rejected_payloads: f64,
}

/// Peer Scores
///
/// <https://github.com/ethereum-optimism/optimism/blob/8dd17a7b114a7c25505cd2e15ce4e3d0f7e3f7c1/op-node/p2p/store/iface.go#L81>
#[derive(Clone, Default, Debug, Copy, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PeerScores {
    /// The gossip scores
    pub gossip: GossipScores,
    /// The request-response scores.
    pub req_resp: ReqRespScores,
}

/// Peer count data.
#[derive(Clone, Default, Debug, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PeerCount {
    /// The total number of connected peers to the discovery service.
    pub connected_discovery: Option<usize>,
    /// The total number of connected peers to the gossip service.
    pub connected_gossip: usize,
}

/// A raw peer dump.
///
/// <https://github.com/ethereum-optimism/optimism/blob/40750a58e7a4a6f06370d18dfe6c6eab309012d9/op-node/p2p/rpc_api.go#L36>
#[derive(Clone, Default, Debug, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PeerDump {
    /// The total number of connected peers
    pub total_connected: u32,
    /// A map from peer id to peer info
    pub peers: HashMap<String, PeerInfo>,
    /// A list of banned peers.
    pub banned_peers: Vec<String>,
    /// A list of banned ip addresses.
    #[serde(rename = "bannedIPS")]
    pub banned_ips: Vec<IpAddr>,
    /// The banned subnets
    pub banned_subnets: Vec<ipnet::IpNet>,
}

/// Peer stats.
///
/// <https://github.com/ethereum-optimism/optimism/blob/develop/op-node/p2p/rpc_server.go#L203>
#[derive(Clone, Default, Debug, Copy, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct PeerStats {
    /// The number of connections
    pub connected: u32,
    /// The table.
    pub table: u32,
    /// The blocks topic.
    #[serde(rename = "blocksTopic")]
    pub blocks_topic: u32,
    /// The blocks v2 topic.
    #[serde(rename = "blocksTopicV2")]
    pub blocks_topic_v2: u32,
    /// The blocks v3 topic.
    #[serde(rename = "blocksTopicV3")]
    pub blocks_topic_v3: u32,
    /// The blocks v4 topic.
    #[serde(rename = "blocksTopicV4")]
    pub blocks_topic_v4: u32,
    /// The banned count.
    pub banned: u32,
    /// The known count.
    pub known: u32,
}

/// Represents the connectivity state of a peer in a network, indicating the reachability and
/// interaction status of a node with its peers.
#[derive(
    Clone,
    Debug,
    Display,
    PartialEq,
    Copy,
    Default,
    // We need to use `serde_repr` to serialize the enum as an integer to match the `op-node` API.
    serde_repr::Serialize_repr,
    serde_repr::Deserialize_repr,
)]
#[repr(u8)]
pub enum Connectedness {
    /// No current connection to the peer, and no recent history of a successful connection.
    #[default]
    #[display("Not Connected")]
    NotConnected = 0,

    /// An active, open connection to the peer exists.
    #[display("Connected")]
    Connected = 1,

    /// Connection to the peer is possible but not currently established; usually implies a past
    /// successful connection.
    #[display("Can Connect")]
    CanConnect = 2,

    /// Recent attempts to connect to the peer failed, indicating potential issues in reachability
    /// or peer status.
    #[display("Cannot Connect")]
    CannotConnect = 3,

    /// Connection to the peer is limited; may not have full capabilities.
    #[display("Limited")]
    Limited = 4,
}

impl From<u8> for Connectedness {
    fn from(value: u8) -> Self {
        match value {
            0 => Self::NotConnected,
            1 => Self::Connected,
            2 => Self::CanConnect,
            3 => Self::CannotConnect,
            4 => Self::Limited,
            _ => Self::NotConnected,
        }
    }
}
/// Direction represents the direction of a connection.
#[derive(Debug, Clone, Display, Copy, PartialEq, Eq, Default)]
pub enum Direction {
    /// Unknown is the default direction when the direction is not specified.
    #[default]
    #[display("Unknown")]
    Unknown = 0,
    /// Inbound is for when the remote peer initiated the connection.
    #[display("Inbound")]
    Inbound = 1,
    /// Outbound is for when the local peer initiated the connection.
    #[display("Outbound")]
    Outbound = 2,
}

impl serde::Serialize for Direction {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
    {
        serializer.serialize_u8(*self as u8)
    }
}

impl<'de> serde::Deserialize<'de> for Direction {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let value = u8::deserialize(deserializer)?;
        match value {
            0 => Ok(Self::Unknown),
            1 => Ok(Self::Inbound),
            2 => Ok(Self::Outbound),
            _ => Err(serde::de::Error::invalid_value(
                serde::de::Unexpected::Unsigned(value as u64),
                &"a value between 0 and 2",
            )),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_connectedness_from_u8() {
        assert_eq!(Connectedness::from(0), Connectedness::NotConnected);
        assert_eq!(Connectedness::from(1), Connectedness::Connected);
        assert_eq!(Connectedness::from(2), Connectedness::CanConnect);
        assert_eq!(Connectedness::from(3), Connectedness::CannotConnect);
        assert_eq!(Connectedness::from(4), Connectedness::Limited);
        assert_eq!(Connectedness::from(5), Connectedness::NotConnected);
    }

    #[test]
    fn test_direction_display() {
        assert_eq!(Direction::Unknown.to_string(), "Unknown");
        assert_eq!(Direction::Inbound.to_string(), "Inbound");
        assert_eq!(Direction::Outbound.to_string(), "Outbound");
    }

    #[test]
    fn test_direction_serialization() {
        assert_eq!(
            serde_json::to_string(&Direction::Unknown).unwrap(),
            "0",
            "Serialization failed for Direction::Unknown"
        );
        assert_eq!(
            serde_json::to_string(&Direction::Inbound).unwrap(),
            "1",
            "Serialization failed for Direction::Inbound"
        );
        assert_eq!(
            serde_json::to_string(&Direction::Outbound).unwrap(),
            "2",
            "Serialization failed for Direction::Outbound"
        );
    }

    #[test]
    fn test_direction_deserialization() {
        let unknown: Direction = serde_json::from_str("0").unwrap();
        let inbound: Direction = serde_json::from_str("1").unwrap();
        let outbound: Direction = serde_json::from_str("2").unwrap();

        assert_eq!(unknown, Direction::Unknown, "Deserialization mismatch for Direction::Unknown");
        assert_eq!(inbound, Direction::Inbound, "Deserialization mismatch for Direction::Inbound");
        assert_eq!(
            outbound,
            Direction::Outbound,
            "Deserialization mismatch for Direction::Outbound"
        );
    }

    #[test]
    fn test_peer_info_connectedness_serialization() {
        let peer_info = PeerInfo {
            peer_id: String::from("peer123"),
            node_id: String::from("node123"),
            user_agent: String::from("MyUserAgent"),
            protocol_version: String::from("v1"),
            enr: Some(String::from("enr123")),
            addresses: [String::from("127.0.0.1")].to_vec(),
            protocols: Some([String::from("eth"), String::from("p2p")].to_vec()),
            connectedness: Connectedness::Connected,
            direction: Direction::Outbound,
            protected: true,
            chain_id: 1,
            latency: 100,
            gossip_blocks: true,
            peer_scores: PeerScores {
                gossip: GossipScores {
                    total: 1.0,
                    blocks: TopicScores {
                        time_in_mesh: 10.0,
                        first_message_deliveries: 5.0,
                        mesh_message_deliveries: 2.0,
                        invalid_message_deliveries: 0.0,
                    },
                    ip_colocation_factor: 0.5,
                    behavioral_penalty: 0.1,
                },
                req_resp: ReqRespScores {
                    valid_responses: 10.0,
                    error_responses: 1.0,
                    rejected_payloads: 0.0,
                },
            },
        };

        let serialized = serde_json::to_string(&peer_info).expect("Serialization failed");

        let deserialized: PeerInfo =
            serde_json::from_str(&serialized).expect("Deserialization failed");

        assert_eq!(peer_info.peer_id, deserialized.peer_id);
        assert_eq!(peer_info.node_id, deserialized.node_id);
        assert_eq!(peer_info.user_agent, deserialized.user_agent);
        assert_eq!(peer_info.protocol_version, deserialized.protocol_version);
        assert_eq!(peer_info.enr, deserialized.enr);
        assert_eq!(peer_info.addresses, deserialized.addresses);
        assert_eq!(peer_info.protocols, deserialized.protocols);
        assert_eq!(peer_info.connectedness, deserialized.connectedness);
        assert_eq!(peer_info.direction, deserialized.direction);
        assert_eq!(peer_info.protected, deserialized.protected);
        assert_eq!(peer_info.chain_id, deserialized.chain_id);
        assert_eq!(peer_info.latency, deserialized.latency);
        assert_eq!(peer_info.gossip_blocks, deserialized.gossip_blocks);
        assert_eq!(peer_info.peer_scores.gossip.total, deserialized.peer_scores.gossip.total);
        assert_eq!(
            peer_info.peer_scores.req_resp.valid_responses,
            deserialized.peer_scores.req_resp.valid_responses
        );
    }
}
