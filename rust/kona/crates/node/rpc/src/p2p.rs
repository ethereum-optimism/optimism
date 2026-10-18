//! RPC Module to serve the P2P API.
//!
//! Kona's P2P RPC API is a JSON-RPC API compatible with the [op-node] API.
//!
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/7a6788836984996747193b91901a824c39032bd8/op-node/p2p/rpc_api.go#L45

use async_trait::async_trait;
use ipnet::IpNet;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_gossip::{P2pRpcRequest, PeerCount, PeerDump, PeerInfo, PeerStats};
use std::{net::IpAddr, str::FromStr, time::Duration};

use crate::{OpP2PApiServer, net::P2pRpc};

impl P2pRpc {
    /// Admission is not confirmation: observe the requested effect within one total deadline.
    async fn change_peer_connection(
        &self,
        request: P2pRpcRequest,
        peer_id: libp2p::PeerId,
        connected: bool,
    ) -> RpcResult<()> {
        let peer_id = peer_id.to_string();
        tokio::time::timeout(Duration::from_secs(10), async {
            self.sender.send(request).await.map_err(|_| {
                if connected {
                    ErrorObject::borrowed(
                        ErrorCode::InternalError.code(),
                        "Failed to send connect peer request",
                        None,
                    )
                } else {
                    ErrorObject::from(ErrorCode::InternalError)
                }
            })?;
            loop {
                let (tx, rx) = tokio::sync::oneshot::channel();
                self.sender
                    .send(P2pRpcRequest::Peers { out: tx, connected: true })
                    .await
                    .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;
                let peers = rx.await.map_err(|_| {
                    ErrorObject::borrowed(
                        ErrorCode::InternalError.code(),
                        "Failed to get peers",
                        None,
                    )
                })?;
                if peers.peers.contains_key(&peer_id) == connected {
                    return Ok(());
                }
                tokio::time::sleep(Duration::from_millis(100)).await;
            }
        })
        .await
        .map_err(|_| {
            ErrorObject::borrowed(
                ErrorCode::InvalidParams.code(),
                if connected { "Peer not connected" } else { "Peers are still connected" },
                None,
            )
        })?
    }
}

#[async_trait]
impl OpP2PApiServer for P2pRpc {
    async fn opp2p_self(&self) -> RpcResult<PeerInfo> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_self");
        let (tx, rx) = tokio::sync::oneshot::channel();
        self.sender
            .send(P2pRpcRequest::PeerInfo(tx))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        rx.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_peer_count(&self) -> RpcResult<PeerCount> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_peerCount");
        let (tx, rx) = tokio::sync::oneshot::channel();
        self.sender
            .send(P2pRpcRequest::PeerCount(tx))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        let (connected_discovery, connected_gossip) =
            rx.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        Ok(PeerCount { connected_discovery, connected_gossip })
    }

    async fn opp2p_peers(&self, connected: bool) -> RpcResult<PeerDump> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_peers");
        let (tx, rx) = tokio::sync::oneshot::channel();
        self.sender
            .send(P2pRpcRequest::Peers { out: tx, connected })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        let dump = rx.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        Ok(dump)
    }

    async fn opp2p_peer_stats(&self) -> RpcResult<PeerStats> {
        let (tx, rx) = tokio::sync::oneshot::channel();
        self.sender
            .send(P2pRpcRequest::PeerStats(tx))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        let stats = rx.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        Ok(stats)
    }

    async fn opp2p_discovery_table(&self) -> RpcResult<Vec<String>> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_discoveryTable");
        let (tx, rx) = tokio::sync::oneshot::channel();
        self.sender
            .send(P2pRpcRequest::DiscoveryTable(tx))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        rx.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_block_peer(&self, peer_id: String) -> RpcResult<()> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_blockPeer");
        let id = libp2p::PeerId::from_str(&peer_id)
            .map_err(|_| ErrorObject::from(ErrorCode::InvalidParams))?;
        self.sender
            .send(P2pRpcRequest::BlockPeer { id })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_unblock_peer(&self, peer_id: String) -> RpcResult<()> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_unblockPeer");
        let id = libp2p::PeerId::from_str(&peer_id)
            .map_err(|_| ErrorObject::from(ErrorCode::InvalidParams))?;
        self.sender
            .send(P2pRpcRequest::UnblockPeer { id })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_list_blocked_peers(&self) -> RpcResult<Vec<String>> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_listBlockedPeers");
        let (tx, rx) = tokio::sync::oneshot::channel();
        self.sender
            .send(P2pRpcRequest::ListBlockedPeers(tx))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        rx.await
            .map(|peers| peers.iter().map(|p| p.to_string()).collect())
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_block_addr(&self, address: IpAddr) -> RpcResult<()> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_blockAddr");
        self.sender
            .send(P2pRpcRequest::BlockAddr { address })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_unblock_addr(&self, address: IpAddr) -> RpcResult<()> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_unblockAddr");
        self.sender
            .send(P2pRpcRequest::UnblockAddr { address })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_list_blocked_addrs(&self) -> RpcResult<Vec<IpAddr>> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_listBlockedAddrs");
        let (tx, rx) = tokio::sync::oneshot::channel();
        self.sender
            .send(P2pRpcRequest::ListBlockedAddrs(tx))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        rx.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_block_subnet(&self, subnet: IpNet) -> RpcResult<()> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_blockSubnet");
        self.sender
            .send(P2pRpcRequest::BlockSubnet { address: subnet })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_unblock_subnet(&self, subnet: IpNet) -> RpcResult<()> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_unblockSubnet");

        self.sender
            .send(P2pRpcRequest::UnblockSubnet { address: subnet })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_list_blocked_subnets(&self) -> RpcResult<Vec<IpNet>> {
        kona_macros::inc!(
            gauge,
            kona_gossip::Metrics::RPC_CALLS,
            "method" => "opp2p_listBlockedSubnets"
        );
        let (tx, rx) = tokio::sync::oneshot::channel();
        self.sender
            .send(P2pRpcRequest::ListBlockedSubnets(tx))
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        rx.await.map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_protect_peer(&self, id: String) -> RpcResult<()> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_protectPeer");
        let peer_id = libp2p::PeerId::from_str(&id)
            .map_err(|_| ErrorObject::from(ErrorCode::InvalidParams))?;
        self.sender
            .send(P2pRpcRequest::ProtectPeer { peer_id })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_unprotect_peer(&self, id: String) -> RpcResult<()> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_unprotectPeer");
        let peer_id = libp2p::PeerId::from_str(&id)
            .map_err(|_| ErrorObject::from(ErrorCode::InvalidParams))?;
        self.sender
            .send(P2pRpcRequest::UnprotectPeer { peer_id })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))
    }

    async fn opp2p_connect_peer(&self, _peer: String) -> RpcResult<()> {
        use std::str::FromStr;
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_connectPeer");
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

        self.change_peer_connection(P2pRpcRequest::ConnectPeer { address: ma }, peer_id, true).await
    }

    async fn opp2p_disconnect_peer(&self, peer_id: String) -> RpcResult<()> {
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_disconnectPeer");
        let peer_id = match peer_id.parse() {
            Ok(id) => id,
            Err(err) => {
                warn!(target: "rpc", ?err, ?peer_id, "Failed to parse peer ID");
                return Err(ErrorObject::from(ErrorCode::InvalidParams));
            }
        };

        self.change_peer_connection(P2pRpcRequest::DisconnectPeer { peer_id }, peer_id, false).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use jsonrpsee::{core::server::MethodsError, rpc_params};
    use kona_gossip::{ConnectionGate, ConnectionGater};
    use libp2p::PeerId;
    use tokio::{
        sync::mpsc,
        time::{self, Instant},
    };

    fn peers(peer: PeerId, connected: bool) -> PeerDump {
        let mut peers = PeerDump::default();
        if connected {
            peers.peers.insert(
                peer.to_string(),
                PeerInfo { peer_id: peer.to_string(), ..Default::default() },
            );
        }
        peers
    }

    fn assert_mutation(request: P2pRpcRequest, peer: PeerId, connecting: bool) {
        match request {
            P2pRpcRequest::ConnectPeer { address } if connecting => {
                assert_eq!(address.to_string(), format!("/ip4/127.0.0.1/tcp/9000/p2p/{peer}"));
            }
            P2pRpcRequest::DisconnectPeer { peer_id } if !connecting => assert_eq!(peer_id, peer),
            other => panic!("unexpected mutation {other:?}"),
        }
    }

    #[tokio::test(start_paused = true)]
    async fn connection_changes_wait_for_observed_effect_and_pace_unsatisfied_reads() {
        for connecting in [true, false] {
            let peer = PeerId::random();
            let (sender, mut receiver) = mpsc::channel(1);
            let rpc = P2pRpc::new(sender).into_rpc();
            let method = if connecting { "opp2p_connectPeer" } else { "opp2p_disconnectPeer" };
            let param = if connecting {
                format!("/ip4/127.0.0.1/tcp/9000/p2p/{peer}")
            } else {
                peer.to_string()
            };
            let backend = async {
                assert_mutation(receiver.recv().await.unwrap(), peer, connecting);
                let P2pRpcRequest::Peers { out, connected } = receiver.recv().await.unwrap() else {
                    panic!("expected peer observation")
                };
                assert!(connected);
                out.send(peers(peer, !connecting)).unwrap();
                let first = Instant::now();
                assert!(time::timeout(Duration::from_millis(99), receiver.recv()).await.is_err());
                let P2pRpcRequest::Peers { out, connected } = receiver.recv().await.unwrap() else {
                    panic!("expected retried observation")
                };
                assert!(connected);
                assert_eq!(first.elapsed(), Duration::from_millis(100));
                out.send(peers(peer, connecting)).unwrap();
            };
            let (result, ()) = tokio::join!(rpc.call::<_, ()>(method, rpc_params![param]), backend);
            result.unwrap();
            assert!(receiver.try_recv().is_err());
        }
    }

    #[tokio::test(start_paused = true)]
    async fn permanently_unsatisfied_peer_state_retries_until_the_total_deadline() {
        for connecting in [true, false] {
            let peer = PeerId::random();
            let (sender, mut receiver) = mpsc::channel(1);
            let rpc = P2pRpc::new(sender).into_rpc();
            let method = if connecting { "opp2p_connectPeer" } else { "opp2p_disconnectPeer" };
            let param = if connecting {
                format!("/ip4/127.0.0.1/tcp/9000/p2p/{peer}")
            } else {
                peer.to_string()
            };
            let mut observations = 0;
            let began = Instant::now();
            let error = {
                let backend = async {
                    assert_mutation(receiver.recv().await.unwrap(), peer, connecting);
                    loop {
                        let P2pRpcRequest::Peers { out, connected } =
                            receiver.recv().await.unwrap()
                        else {
                            panic!("expected peer observation")
                        };
                        assert!(connected);
                        observations += 1;
                        let _ = out.send(peers(peer, !connecting));
                    }
                };
                tokio::pin!(backend);
                tokio::select! {
                    result = rpc.call::<_, ()>(method, rpc_params![param]) => result.unwrap_err(),
                    _ = &mut backend => panic!("backend unexpectedly ended"),
                }
            };
            assert_eq!(began.elapsed(), Duration::from_secs(10));
            assert!(observations > 1 && observations <= 100);
            assert!(matches!(error, MethodsError::JsonRpc(error) if error.code() == -32602 &&
                error.message() == if connecting { "Peer not connected" } else { "Peers are still connected" }));
        }
    }

    #[tokio::test(start_paused = true)]
    async fn one_deadline_includes_mutation_admission_observation_admission_and_reply() {
        for connecting in [true, false] {
            for phase in 0..3 {
                let peer = PeerId::random();
                let (sender, mut receiver) = mpsc::channel(1);
                // Phase 0 blocks mutation admission. Phase 1 leaves the admitted mutation
                // filling the queue, so the first observation cannot be admitted.
                if phase != 1 {
                    let (reply, _) = tokio::sync::oneshot::channel();
                    sender.try_send(P2pRpcRequest::PeerCount(reply)).unwrap();
                }
                let rpc = P2pRpc::new(sender).into_rpc();
                let method = if connecting { "opp2p_connectPeer" } else { "opp2p_disconnectPeer" };
                let param = if connecting {
                    format!("/ip4/127.0.0.1/tcp/9000/p2p/{peer}")
                } else {
                    peer.to_string()
                };
                let began = Instant::now();
                let error = {
                    let backend = async {
                        if phase != 2 {
                            std::future::pending::<()>().await;
                        }
                        // Spend part of the SAME deadline waiting for mutation admission,
                        // then hold its observation reply. A per-phase timeout would take 16s.
                        time::sleep(Duration::from_secs(6)).await;
                        assert!(matches!(
                            receiver.recv().await.unwrap(),
                            P2pRpcRequest::PeerCount(_)
                        ));
                        assert_mutation(receiver.recv().await.unwrap(), peer, connecting);
                        let P2pRpcRequest::Peers { out, .. } = receiver.recv().await.unwrap()
                        else {
                            panic!("expected held peer reply")
                        };
                        let _held = out;
                        std::future::pending::<()>().await;
                    };
                    tokio::pin!(backend);
                    tokio::select! {
                        result = rpc.call::<_, ()>(method, rpc_params![param]) => result.unwrap_err(),
                        _ = &mut backend => panic!("backend unexpectedly ended"),
                    }
                };
                assert_eq!(began.elapsed(), Duration::from_secs(10));
                assert!(matches!(error, MethodsError::JsonRpc(error) if error.code() == -32602));
                if phase == 0 {
                    assert!(matches!(receiver.try_recv().unwrap(), P2pRpcRequest::PeerCount(_)));
                    assert!(receiver.try_recv().is_err(), "timed-out mutation was never admitted");
                } else if phase == 1 {
                    assert_mutation(receiver.try_recv().unwrap(), peer, connecting);
                    assert!(
                        receiver.try_recv().is_err(),
                        "timed-out observation was never admitted"
                    );
                }
            }
        }
    }

    #[tokio::test(start_paused = true)]
    async fn closed_admission_or_reply_fails_without_retrying() {
        for connecting in [true, false] {
            for phase in 0..3 {
                let peer = PeerId::random();
                let (sender, mut receiver) = mpsc::channel(1);
                let rpc = P2pRpc::new(sender).into_rpc();
                let method = if connecting { "opp2p_connectPeer" } else { "opp2p_disconnectPeer" };
                let param = if connecting {
                    format!("/ip4/127.0.0.1/tcp/9000/p2p/{peer}")
                } else {
                    peer.to_string()
                };
                if phase == 0 {
                    receiver.close();
                }
                let began = Instant::now();
                let error = {
                    let backend = async {
                        if phase == 0 {
                            std::future::pending::<()>().await;
                        }
                        assert_mutation(receiver.recv().await.unwrap(), peer, connecting);
                        if phase == 1 {
                            receiver.close();
                        } else {
                            let P2pRpcRequest::Peers { out, .. } = receiver.recv().await.unwrap()
                            else {
                                panic!("expected peer reply")
                            };
                            drop(out);
                        }
                        std::future::pending::<()>().await;
                    };
                    tokio::pin!(backend);
                    tokio::select! {
                        result = rpc.call::<_, ()>(method, rpc_params![param]) => result.unwrap_err(),
                        _ = &mut backend => panic!("backend unexpectedly ended"),
                    }
                };
                assert_eq!(began.elapsed(), Duration::ZERO);
                assert!(matches!(error, MethodsError::JsonRpc(error) if error.code() == -32603));
            }
        }
    }

    #[tokio::test]
    async fn invalid_connection_params_never_reach_admission() {
        let (sender, mut receiver) = mpsc::channel(1);
        let rpc = P2pRpc::new(sender).into_rpc();
        for method in ["opp2p_connectPeer", "opp2p_disconnectPeer"] {
            let error = rpc.call::<_, ()>(method, rpc_params!["invalid-peer"]).await.unwrap_err();
            assert!(matches!(error, MethodsError::JsonRpc(error) if error.code() == -32602));
        }
        let error = rpc
            .call::<_, ()>("opp2p_connectPeer", rpc_params!["/ip4/127.0.0.1/tcp/9000"])
            .await
            .unwrap_err();
        assert!(matches!(error, MethodsError::JsonRpc(error) if error.code() == -32602));
        assert!(receiver.try_recv().is_err());
    }

    #[tokio::test]
    async fn canonical_and_legacy_block_addr_names_validate_and_mutate_the_same_gate() {
        let (sender, mut receiver) = mpsc::channel(1);
        let rpc = P2pRpc::new(sender).into_rpc();
        let address: IpAddr = "127.0.0.42".parse().unwrap();
        let backend = async {
            let mut gate = ConnectionGater::default();
            while let Some(request) = receiver.recv().await {
                match request {
                    P2pRpcRequest::BlockAddr { address } => gate.block_addr(address),
                    P2pRpcRequest::UnblockAddr { address } => gate.unblock_addr(address),
                    P2pRpcRequest::ListBlockedAddrs(reply) => {
                        reply.send(gate.list_blocked_addrs()).unwrap();
                    }
                    other => panic!("unexpected request {other:?}"),
                }
            }
        };
        let client = async move {
            for method in ["opp2p_blockAddr", "opp2p_blocAddr"] {
                for params in [rpc_params![], rpc_params!["not-an-ip"]] {
                    let error = rpc.call::<_, ()>(method, params).await.unwrap_err();
                    assert!(
                        matches!(error, MethodsError::JsonRpc(error) if error.code() == -32602)
                    );
                }
                rpc.call::<_, ()>(method, rpc_params![address]).await.unwrap();
                assert_eq!(
                    rpc.call::<_, Vec<IpAddr>>("opp2p_listBlockedAddrs", rpc_params![])
                        .await
                        .unwrap(),
                    vec![address]
                );
                rpc.call::<_, ()>("opp2p_unblockAddr", rpc_params![address]).await.unwrap();
                assert!(
                    rpc.call::<_, Vec<IpAddr>>("opp2p_listBlockedAddrs", rpc_params![])
                        .await
                        .unwrap()
                        .is_empty()
                );
            }
        };
        tokio::join!(client, backend);
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
