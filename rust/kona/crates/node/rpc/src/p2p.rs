//! RPC Module to serve the P2P API.
//!
//! Kona's P2P RPC API is a JSON-RPC API compatible with the [op-node] API.
//!
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/7a6788836984996747193b91901a824c39032bd8/op-node/p2p/rpc_api.go#L45

use alloy_primitives::map::HashMap;
use async_trait::async_trait;
use backon::{ExponentialBuilder, Retryable};
use enr::{NodeId, k256::ecdsa};
use ipnet::IpNet;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_disc::Discv5Handler;
use kona_gossip::{
    Connectedness, Direction, GossipCommand, GossipCommandSender, GossipQueryHandle, GossipScores,
    GossipState, PeerCount, PeerDump, PeerInfo, PeerScores, PeerStats,
};
use kona_peers::{OpStackEnr, peer_id_to_node_id};
use libp2p::{PeerId, multiaddr::Protocol};
use std::{net::IpAddr, str::FromStr, sync::Arc, time::Duration};

use crate::OpP2PApiServer;

/// Server implementation of [`crate::OpP2PApiServer`].
#[derive(Debug, Clone)]
pub struct P2pRpc {
    gossip: GossipQueryHandle,
    /// `None` if discovery is disabled.
    discovery: Option<Discv5Handler>,
    chain_id: u64,
    commands: GossipCommandSender,
    /// How old the gossip state a call answers from may be.
    snapshot_max_age: Duration,
}

/// Logs a failure the client can't fix, and returns it as an internal error that names the
/// operation that failed and why.
fn internal_error(operation: &str, err: impl std::fmt::Display) -> ErrorObject<'static> {
    let message = format!("{operation}: {err}");
    warn!(target: "p2p::rpc", "{message}");
    ErrorObject::owned(ErrorCode::InternalError.code(), message, None::<()>)
}

/// How old the gossip state an RPC call answers from may be, unless set with
/// [`P2pRpc::with_snapshot_max_age`].
///
/// What a command changes is there at once: the network actor publishes the state after each
/// one. Up to this old, changes from the network itself, such as connections and scores, may be
/// missing.
const SNAPSHOT_MAX_AGE: Duration = Duration::from_secs(1);

/// The error op-node's `opp2p_discoveryTable` returns when discovery is disabled.
fn discovery_disabled() -> ErrorObject<'static> {
    // The code op-node's server assigns to a plain error.
    ErrorObject::owned(-32000, "discovery disabled", None::<()>)
}

#[async_trait]
impl OpP2PApiServer for P2pRpc {
    async fn opp2p_self(&self) -> RpcResult<PeerInfo> {
        let gossip = self.gossip_snapshot().await?;
        let peer_id = gossip.local_peer_id;
        let chain_id = self.chain_id;
        // Like op-node, from the libp2p key, which is discovery's key too.
        let node_id = peer_id_to_node_id(&peer_id).ok_or_else(|| {
            internal_error("peer_id_to_node_id", "local peer ID holds no secp256k1 key")
        })?;
        let enr = match &self.discovery {
            Some(discovery) => {
                Some(discovery.local_enr().await.map_err(|e| internal_error("local_enr", e))?)
            }
            None => None,
        };
        let mut addresses = gossip
            .listen_addresses
            .iter()
            .map(|address| {
                let mut address = address.clone();
                address.push(Protocol::P2p(peer_id));
                address.to_string()
            })
            .collect::<Vec<_>>();
        addresses.extend(gossip.external_addresses.iter().map(ToString::to_string));
        Ok(PeerInfo {
            peer_id: peer_id.to_string(),
            // Display abbreviates the node ID; the RPC response needs the full value.
            node_id: format!("{node_id:?}"),
            user_agent: "kona".to_string(),
            protocol_version: String::new(),
            enr: enr.map(|enr| enr.to_string()),
            addresses,
            protocols: Some(vec![
                "/ipfs/id/push/1.0.0".to_string(),
                "/meshsub/1.1.0".to_string(),
                "/ipfs/ping/1.0.0".to_string(),
                "/meshsub/1.2.0".to_string(),
                "/meshsub/1.3.0".to_string(),
                "/ipfs/id/1.0.0".to_string(),
                "/meshsub/1.0.0".to_string(),
                "/floodsub/1.0.0".to_string(),
            ]),
            // Preserve the self-response values from the gossip RPC handler. Connection
            // direction, protection, latency, and peer scores do not describe a remote peer here.
            connectedness: Connectedness::Connected,
            direction: Direction::Inbound,
            protected: false,
            chain_id,
            latency: 0,
            gossip_blocks: true,
            peer_scores: PeerScores::default(),
        })
    }

    async fn opp2p_peer_count(&self) -> RpcResult<PeerCount> {
        let gossip = self.gossip_snapshot().await?;
        let connected_discovery = match &self.discovery {
            Some(discovery) => discovery
                .peer_count()
                .await
                .inspect_err(|e| {
                    warn!(target: "p2p::rpc", "Failed to receive peer count: {e}");
                })
                .ok(),
            None => None,
        };
        Ok(PeerCount { connected_discovery, connected_gossip: gossip.connected_peers.len() })
    }

    async fn opp2p_peers(&self, connected: bool) -> RpcResult<PeerDump> {
        let gossip = self.gossip_snapshot().await?;
        let total_connected = gossip
            .connected_peers
            .len()
            .try_into()
            .map_err(|e| internal_error("u32::try_from", e))?;
        let peer_ids: Vec<PeerId> = if connected {
            gossip.connected_peers.iter().copied().collect()
        } else {
            gossip.peerstore.keys().copied().collect()
        };
        let table_infos = match &self.discovery {
            Some(discovery) => {
                discovery.table_infos().await.map_err(|e| internal_error("table_infos", e))?
            }
            None => Vec::new(),
        };
        let pings = &gossip.pings;
        let node_to_peer_id = peer_ids.into_iter().filter_map(|id| {
            let Ok(pubkey) = libp2p::identity::PublicKey::try_decode_protobuf(&id.to_bytes()[2..]) else {
                error!(target: "p2p::rpc", peer_id = ?id, "Failed to decode public key from peer ID");
                return None;
            };
            let key = match pubkey.try_into_secp256k1().map_err(|e| e.to_string()).and_then(|key| {
                ecdsa::VerifyingKey::from_sec1_bytes(key.to_bytes().as_slice()).map_err(|e| e.to_string())
            }) {
                Ok(key) => key,
                Err(err) => {
                    error!(target: "p2p::rpc", peer_id = ?id, ?err, "Failed to convert public key to secp256k1 public key");
                    return None;
                }
            };
            Some((NodeId::from(key), id))
        }).collect::<HashMap<_, _>>();
        let table_infos = table_infos
            .into_iter()
            .filter(|(id, _, _)| node_to_peer_id.contains_key(id))
            .map(|(id, enr, status)| (id, (enr, status)))
            .collect::<HashMap<_, _>>();
        let peers = node_to_peer_id
            .iter()
            .map(|(node_id, peer_id)| {
                let (enr, status) = table_infos.get(node_id).cloned().unzip();
                let chain_id = enr
                    .clone()
                    .and_then(|enr| OpStackEnr::try_from(&enr).ok())
                    .map(|enr| enr.chain_id)
                    .unwrap_or(0);
                let info = gossip.peerstore.get(peer_id);
                let protocols = info
                    .filter(|info| !info.protocols.is_empty())
                    .map(|info| info.protocols.iter().map(ToString::to_string).collect());
                let addresses = info
                    .map(|info| {
                        info.listen_addrs
                            .iter()
                            .map(|address| {
                                let mut address = address.clone();
                                address.push(Protocol::P2p(*peer_id));
                                address.to_string()
                            })
                            .collect()
                    })
                    .unwrap_or_default();
                let connectedness = if gossip.connected_peers.contains(peer_id) {
                    Connectedness::Connected
                } else if gossip.blocked_peers.contains(peer_id) {
                    Connectedness::CannotConnect
                } else {
                    Connectedness::NotConnected
                };
                (
                    peer_id.to_string(),
                    PeerInfo {
                        peer_id: peer_id.to_string(),
                        node_id: format!("{node_id:?}"),
                        user_agent: info.map(|info| info.agent_version.clone()).unwrap_or_default(),
                        protocol_version: info
                            .map(|info| info.protocol_version.clone())
                            .unwrap_or_default(),
                        enr: enr.map(|enr| enr.to_string()),
                        addresses,
                        protocols,
                        connectedness,
                        direction: status
                            .map(|status| {
                                if status.is_incoming() {
                                    Direction::Inbound
                                } else {
                                    Direction::Outbound
                                }
                            })
                            .unwrap_or_default(),
                        chain_id,
                        // In nanoseconds, as op-node's `time.Duration` encodes it. op-node reports
                        // an average of recent pings, kona the latest one.
                        latency: pings
                            .get(peer_id)
                            .map(|duration| duration.as_nanos().try_into().unwrap_or(u64::MAX))
                            .unwrap_or(0),
                        gossip_blocks: gossip.gossip_peers.contains(peer_id),
                        protected: gossip.protected_peers.contains(peer_id),
                        peer_scores: PeerScores {
                            // libp2p exposes the total score but not its individual components.
                            gossip: GossipScores {
                                total: gossip.peer_scores.get(peer_id).copied().unwrap_or_default(),
                                ..Default::default()
                            },
                            req_resp: Default::default(),
                        },
                    },
                )
            })
            .collect();
        Ok(PeerDump {
            total_connected,
            peers,
            banned_peers: gossip.blocked_peers.iter().map(ToString::to_string).collect(),
            banned_ips: gossip.blocked_addresses.clone(),
            banned_subnets: gossip.blocked_subnets.clone(),
        })
    }

    async fn opp2p_peer_stats(&self) -> RpcResult<PeerStats> {
        let gossip = self.gossip_snapshot().await?;
        let table = match &self.discovery {
            Some(discovery) => {
                discovery.peer_count().await.map_err(|e| internal_error("peer_count", e))?
            }
            None => 0,
        };
        let to_u32 =
            |count: usize| count.try_into().map_err(|e| internal_error("u32::try_from", e));
        let block_topics = gossip
            .block_topics
            .iter()
            .map(|topic| to_u32(gossip.topic_peer_counts.get(topic).copied().unwrap_or_default()))
            .collect::<RpcResult<Vec<_>>>()?;
        Ok(PeerStats {
            connected: to_u32(gossip.connected_peers.len())?,
            table: to_u32(table)?,
            blocks_topic: block_topics[0],
            blocks_topic_v2: block_topics[1],
            blocks_topic_v3: block_topics[2],
            blocks_topic_v4: block_topics[3],
            banned: to_u32(gossip.blocked_peers.len())?,
            known: to_u32(gossip.peerstore.len())?,
        })
    }

    async fn opp2p_discovery_table(&self) -> RpcResult<Vec<String>> {
        self.gossip_snapshot().await?;
        self.discovery
            .as_ref()
            .ok_or_else(discovery_disabled)?
            .table_enrs()
            .await
            .map(|enrs| enrs.into_iter().map(|enr| enr.to_string()).collect())
            .map_err(|e| internal_error("table_enrs", e))
    }

    async fn opp2p_block_peer(&self, peer_id: String) -> RpcResult<()> {
        let id = libp2p::PeerId::from_str(&peer_id)
            .map_err(|_| ErrorObject::from(ErrorCode::InvalidParams))?;
        self.apply_command(GossipCommand::BlockPeer { id }).await
    }

    async fn opp2p_unblock_peer(&self, peer_id: String) -> RpcResult<()> {
        let id = libp2p::PeerId::from_str(&peer_id)
            .map_err(|_| ErrorObject::from(ErrorCode::InvalidParams))?;
        self.apply_command(GossipCommand::UnblockPeer { id }).await
    }

    async fn opp2p_list_blocked_peers(&self) -> RpcResult<Vec<String>> {
        let gossip = self.gossip_snapshot().await?;
        Ok(gossip.blocked_peers.iter().map(ToString::to_string).collect())
    }

    async fn opp2p_block_addr(&self, address: IpAddr) -> RpcResult<()> {
        self.apply_command(GossipCommand::BlockAddr { address }).await
    }

    async fn opp2p_unblock_addr(&self, address: IpAddr) -> RpcResult<()> {
        self.apply_command(GossipCommand::UnblockAddr { address }).await
    }

    async fn opp2p_list_blocked_addrs(&self) -> RpcResult<Vec<IpAddr>> {
        let gossip = self.gossip_snapshot().await?;
        Ok(gossip.blocked_addresses.clone())
    }

    async fn opp2p_block_subnet(&self, subnet: IpNet) -> RpcResult<()> {
        self.apply_command(GossipCommand::BlockSubnet { address: subnet }).await
    }

    async fn opp2p_unblock_subnet(&self, subnet: IpNet) -> RpcResult<()> {
        self.apply_command(GossipCommand::UnblockSubnet { address: subnet }).await
    }

    async fn opp2p_list_blocked_subnets(&self) -> RpcResult<Vec<IpNet>> {
        let gossip = self.gossip_snapshot().await?;
        Ok(gossip.blocked_subnets.clone())
    }

    async fn opp2p_protect_peer(&self, id: String) -> RpcResult<()> {
        let peer_id = libp2p::PeerId::from_str(&id)
            .map_err(|_| ErrorObject::from(ErrorCode::InvalidParams))?;
        self.apply_command(GossipCommand::ProtectPeer { peer_id }).await
    }

    async fn opp2p_unprotect_peer(&self, id: String) -> RpcResult<()> {
        let peer_id = libp2p::PeerId::from_str(&id)
            .map_err(|_| ErrorObject::from(ErrorCode::InvalidParams))?;
        self.apply_command(GossipCommand::UnprotectPeer { peer_id }).await
    }

    async fn opp2p_connect_peer(&self, _peer: String) -> RpcResult<()> {
        use std::str::FromStr;
        let ma = libp2p::Multiaddr::from_str(&_peer).map_err(|_| {
            ErrorObject::borrowed(ErrorCode::InvalidParams.code(), "Invalid multiaddr", None)
        })?;

        let peer_id = ma
            .iter()
            .find_map(|component| match component {
                libp2p::multiaddr::Protocol::P2p(peer_id) => Some(peer_id),
                _ => None,
            })
            .ok_or_else(|| {
                ErrorObject::borrowed(
                    ErrorCode::InvalidParams.code(),
                    "Impossible to extract peer ID from multiaddr",
                    None,
                )
            })?;

        self.apply_command(GossipCommand::ConnectPeer { address: ma }).await?;

        // We need to wait until both peers are connected to each other to return from this method.
        // We try with an exponential backoff and return an error if we fail to connect to the peer.
        let is_connected = async || {
            let peers = self.opp2p_peers(true).await?;
            Ok::<bool, ErrorObject<'_>>(peers.peers.contains_key(&peer_id.to_string()))
        };

        if !is_connected
            .retry(ExponentialBuilder::default().with_total_delay(Some(Duration::from_secs(10))))
            .await?
        {
            return Err(ErrorObject::borrowed(
                ErrorCode::InvalidParams.code(),
                "Peer not connected",
                None,
            ));
        }

        Ok(())
    }

    async fn opp2p_disconnect_peer(&self, peer_id: String) -> RpcResult<()> {
        let peer_id: PeerId =
            peer_id.parse().map_err(|_| ErrorObject::from(ErrorCode::InvalidParams))?;

        self.apply_command(GossipCommand::DisconnectPeer { peer_id }).await?;

        // We need to wait until both peers are fully disconnected to each other to return from this
        // method. We try with an exponential backoff and return an error if we fail to
        // disconnect from the peer.
        let is_not_connected = async || {
            let peers = self.opp2p_peers(true).await?;
            Ok::<bool, ErrorObject<'_>>(!peers.peers.contains_key(&peer_id.to_string()))
        };

        if !is_not_connected
            .retry(ExponentialBuilder::default().with_total_delay(Some(Duration::from_secs(10))))
            .await?
        {
            return Err(ErrorObject::borrowed(
                ErrorCode::InvalidParams.code(),
                "Peers are still connected",
                None,
            ));
        }

        Ok(())
    }
}

impl P2pRpc {
    /// Constructs an RPC server from read handles and a network command sender. `discovery` is
    /// `None` if discovery is disabled.
    pub const fn new(
        gossip: GossipQueryHandle,
        discovery: Option<Discv5Handler>,
        chain_id: u64,
        commands: GossipCommandSender,
    ) -> Self {
        Self { gossip, discovery, chain_id, commands, snapshot_max_age: SNAPSHOT_MAX_AGE }
    }

    /// Sets how old the gossip state a call answers from may be. With zero, each call waits for
    /// state the network actor builds after the call starts, as tests that watch the network
    /// need.
    pub const fn with_snapshot_max_age(mut self, max_age: Duration) -> Self {
        self.snapshot_max_age = max_age;
        self
    }

    async fn apply_command(&self, command: GossipCommand) -> RpcResult<()> {
        let (applied_tx, applied_rx) = tokio::sync::oneshot::channel();
        self.commands
            .send((command, applied_tx))
            .await
            .map_err(|e| internal_error("apply_command", e))?;
        applied_rx.await.map_err(|e| internal_error("apply_command", e))
    }

    async fn gossip_snapshot(&self) -> RpcResult<Arc<GossipState>> {
        self.gossip
            .snapshot(self.snapshot_max_age)
            .await
            .map_err(|e| internal_error("gossip_snapshot", e))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::Address;
    use kona_disc::{Discv5Handler, HandlerRequest};
    use kona_genesis::RollupConfig;
    use kona_gossip::{ConnectionGater, GossipCommandReceiver, GossipDriver, GossipQueryHandle};
    use libp2p::{StreamProtocol, identify::Info, identity::Keypair};
    use tokio::sync::{mpsc, watch};

    struct Fixture {
        rpc: P2pRpc,
        state_tx: watch::Sender<Arc<GossipState>>,
        /// Kept open, so a stale snapshot's refresh request waits instead of failing.
        #[allow(dead_code)]
        refresh_rx: mpsc::Receiver<()>,
        commands_rx: GossipCommandReceiver,
        discovery_rx: mpsc::Receiver<HandlerRequest>,
        driver: GossipDriver<ConnectionGater>,
    }

    fn fixture(discovery: bool) -> Fixture {
        let driver = GossipDriver::<ConnectionGater>::builder(
            RollupConfig::default(),
            Address::ZERO,
            "/ip4/127.0.0.1/tcp/0".parse().unwrap(),
            Keypair::generate_secp256k1(),
        )
        .build()
        .unwrap();
        let (state_tx, state_rx) = watch::channel(Arc::new(driver.snapshot()));
        let (refresh_tx, refresh_rx) = mpsc::channel(1);
        let (commands_tx, commands_rx) = mpsc::channel(8);
        let (discovery_tx, discovery_rx) = mpsc::channel(8);
        let rpc = P2pRpc::new(
            GossipQueryHandle::new(state_rx, refresh_tx),
            discovery.then(|| Discv5Handler::new(10, discovery_tx)),
            10,
            commands_tx,
        );
        Fixture { rpc, state_tx, refresh_rx, commands_rx, discovery_rx, driver }
    }

    #[tokio::test]
    async fn queries_assemble_peer_data_without_sending_network_requests() {
        let mut fixture = fixture(true);
        let connected_key = Keypair::generate_secp256k1();
        let disconnected_key = Keypair::generate_secp256k1();
        let connected = connected_key.public().to_peer_id();
        let disconnected = disconnected_key.public().to_peer_id();
        let mut state = fixture.driver.snapshot();
        for key in [&connected_key, &disconnected_key] {
            state.peerstore.insert(
                key.public().to_peer_id(),
                Info {
                    public_key: key.public(),
                    protocol_version: "protocol".to_owned(),
                    agent_version: "agent".to_owned(),
                    listen_addrs: vec!["/ip4/127.0.0.1/tcp/1234".parse().unwrap()],
                    protocols: vec![StreamProtocol::new("/meshsub/1.1.0")],
                    observed_addr: "/ip4/127.0.0.1/tcp/1234".parse().unwrap(),
                    signed_peer_record: None,
                },
            );
        }
        state.connected_peers.insert(connected);
        state.blocked_peers.push(disconnected);
        state.protected_peers.push(connected);
        state.gossip_peers.insert(connected);
        state.peer_scores.insert(connected, 3.5);
        state.pings.insert(connected, Duration::from_millis(25));
        state.blocked_addresses.push("192.0.2.1".parse().unwrap());
        state.blocked_subnets.push("192.0.2.0/24".parse().unwrap());
        fixture.state_tx.send_replace(Arc::new(state));

        for connected_only in [false, true] {
            let (dump, ()) = tokio::join!(fixture.rpc.opp2p_peers(connected_only), async {
                let Some(HandlerRequest::TableInfos(out)) = fixture.discovery_rx.recv().await
                else {
                    panic!("expected a direct discovery table query");
                };
                out.send(vec![]).unwrap();
            });
            let dump = dump.unwrap();
            assert_eq!(dump.total_connected, 1);
            assert_eq!(dump.peers.len(), if connected_only { 1 } else { 2 });
            let info = &dump.peers[&connected.to_string()];
            assert_eq!(info.connectedness, Connectedness::Connected);
            assert!(info.protected && info.gossip_blocks);
            assert_eq!(info.peer_scores.gossip.total, 3.5);
            // In nanoseconds, as op-node's `time.Duration` encodes it.
            assert_eq!(info.latency, 25_000_000);
            assert_eq!(info.user_agent, "agent");
            assert_eq!(info.protocol_version, "protocol");
            assert_eq!(info.protocols.as_ref().unwrap(), &["/meshsub/1.1.0"]);
            assert_eq!(info.addresses, vec![format!("/ip4/127.0.0.1/tcp/1234/p2p/{connected}")]);
            assert_eq!(info.chain_id, 0);
            assert!(info.enr.is_none());
            if !connected_only {
                assert_eq!(
                    dump.peers[&disconnected.to_string()].connectedness,
                    Connectedness::CannotConnect
                );
            }
            assert_eq!(dump.banned_peers, vec![disconnected.to_string()]);
            assert_eq!(dump.banned_ips, fixture.rpc.opp2p_list_blocked_addrs().await.unwrap());
            assert_eq!(
                dump.banned_subnets,
                fixture.rpc.opp2p_list_blocked_subnets().await.unwrap()
            );
        }
        assert!(matches!(fixture.commands_rx.try_recv(), Err(mpsc::error::TryRecvError::Empty)));
    }

    #[tokio::test]
    async fn queries_observe_updates_and_fail_after_the_publisher_exits() {
        let fixture = fixture(true);
        let clone = fixture.rpc.clone();
        assert!(clone.opp2p_list_blocked_peers().await.unwrap().is_empty());
        let mut state = fixture.driver.snapshot();
        let blocked = Keypair::generate_secp256k1().public().to_peer_id();
        state.blocked_peers.push(blocked);
        fixture.state_tx.send_replace(Arc::new(state));
        assert_eq!(clone.opp2p_list_blocked_peers().await.unwrap(), vec![blocked.to_string()]);
        drop(fixture.state_tx);
        let stopped =
            (ErrorCode::InternalError.code(), "gossip_snapshot: the network actor has stopped");
        let err = clone.opp2p_list_blocked_peers().await.unwrap_err();
        assert_eq!((err.code(), err.message()), stopped);
        let err = clone.opp2p_peer_count().await.unwrap_err();
        assert_eq!((err.code(), err.message()), stopped);
    }

    #[tokio::test]
    async fn discovery_failure_preserves_optional_peer_count_and_errors_for_stats() {
        let mut fixture = fixture(true);
        let mut state = fixture.driver.snapshot();
        state.connected_peers.insert(Keypair::generate_secp256k1().public().to_peer_id());
        fixture.state_tx.send_replace(Arc::new(state));
        fixture.discovery_rx.close();
        let count = fixture.rpc.opp2p_peer_count().await.unwrap();
        assert_eq!(count.connected_discovery, None);
        assert_eq!(count.connected_gossip, 1);
        let err = fixture.rpc.opp2p_peer_stats().await.unwrap_err();
        assert_eq!(
            (err.code(), err.message()),
            (ErrorCode::InternalError.code(), "peer_count: discovery service stopped")
        );
    }

    /// With discovery disabled, the queries answer as op-node's do: no ENR, an empty table in the
    /// stats, and an error for the table itself.
    #[tokio::test]
    async fn queries_answer_without_discovery() {
        let fixture = fixture(false);
        let mut state = fixture.driver.snapshot();
        let peer = Keypair::generate_secp256k1().public().to_peer_id();
        state.connected_peers.insert(peer);
        fixture.state_tx.send_replace(Arc::new(state));
        let rpc = &fixture.rpc;

        let info = rpc.opp2p_self().await.unwrap();
        assert_eq!(info.enr, None);
        assert_eq!(info.chain_id, 10);
        let node_id = peer_id_to_node_id(fixture.driver.local_peer_id()).unwrap();
        assert_eq!(info.node_id, format!("{node_id:?}"));

        let count = rpc.opp2p_peer_count().await.unwrap();
        assert_eq!(count.connected_discovery, None);
        assert_eq!(count.connected_gossip, 1);

        let dump = rpc.opp2p_peers(true).await.unwrap();
        assert!(dump.peers[&peer.to_string()].enr.is_none());

        let stats = rpc.opp2p_peer_stats().await.unwrap();
        assert_eq!(stats.table, 0);
        assert_eq!(stats.connected, 1);

        let err = rpc.opp2p_discovery_table().await.unwrap_err();
        assert_eq!(err.code(), -32000);
        assert_eq!(err.message(), "discovery disabled");
    }

    #[test]
    fn test_parse_multiaddr_string() {
        use std::str::FromStr;
        let ma = "/ip4/127.0.0.1/udt";
        let multiaddr = libp2p::Multiaddr::from_str(ma).unwrap();
        let components = multiaddr.iter().collect::<Vec<_>>();
        assert_eq!(
            components[0],
            libp2p::multiaddr::Protocol::Ip4(std::net::Ipv4Addr::new(127, 0, 0, 1))
        );
        assert_eq!(components[1], libp2p::multiaddr::Protocol::Udt);
    }
}
