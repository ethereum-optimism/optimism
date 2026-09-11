//! RPC Module to serve the P2P API.
//!
//! Kona's P2P RPC API is a JSON-RPC API compatible with the [op-node] API.
//!
//!
//! [op-node]: https://github.com/ethereum-optimism/optimism/blob/7a6788836984996747193b91901a824c39032bd8/op-node/p2p/rpc_api.go#L45

use async_trait::async_trait;
use backon::{ExponentialBuilder, Retryable};
use ipnet::IpNet;
use jsonrpsee::{
    core::RpcResult,
    types::{ErrorCode, ErrorObject},
};
use kona_gossip::{P2pRpcRequest, PeerCount, PeerDump, PeerInfo, PeerStats};
use std::{net::IpAddr, str::FromStr, time::Duration};

use crate::{OpP2PApiServer, net::P2pRpc};

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
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_peerStats");
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

        self.sender.send(P2pRpcRequest::ConnectPeer { address: ma }).await.map_err(|_| {
            ErrorObject::borrowed(
                ErrorCode::InternalError.code(),
                "Failed to send connect peer request",
                None,
            )
        })?;

        // We need to wait until both peers are connected to each other to return from this method.
        // We try with an exponential backoff and return an error if we fail to connect to the peer.
        let is_connected = async || {
            let (tx, rx) = tokio::sync::oneshot::channel();

            self.sender
                .send(P2pRpcRequest::Peers { out: tx, connected: true })
                .await
                .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

            let peers = rx.await.map_err(|_| {
                ErrorObject::borrowed(ErrorCode::InternalError.code(), "Failed to get peers", None)
            })?;

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
        kona_macros::inc!(gauge, kona_gossip::Metrics::RPC_CALLS, "method" => "opp2p_disconnectPeer");
        let peer_id = match peer_id.parse() {
            Ok(id) => id,
            Err(err) => {
                warn!(target: "rpc", ?err, ?peer_id, "Failed to parse peer ID");
                return Err(ErrorObject::from(ErrorCode::InvalidParams));
            }
        };

        self.sender
            .send(P2pRpcRequest::DisconnectPeer { peer_id })
            .await
            .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

        // We need to wait until both peers are fully disconnected to each other to return from this
        // method. We try with an exponential backoff and return an error if we fail to
        // disconnect from the peer.
        let is_not_connected = async || {
            let (tx, rx) = tokio::sync::oneshot::channel();

            self.sender
                .send(P2pRpcRequest::Peers { out: tx, connected: true })
                .await
                .map_err(|_| ErrorObject::from(ErrorCode::InternalError))?;

            let peers = rx.await.map_err(|_| {
                ErrorObject::borrowed(ErrorCode::InternalError.code(), "Failed to get peers", None)
            })?;

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

#[cfg(test)]
mod tests {
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

    /// Never called; `AdminRpc` needs a concrete type parameter even when it holds `None`.
    #[cfg(feature = "metrics")]
    #[derive(Debug)]
    struct NoSequencer;

    #[cfg(feature = "metrics")]
    #[async_trait::async_trait]
    impl crate::SequencerAdminAPIClient for NoSequencer {
        async fn is_sequencer_active(&self) -> Result<bool, crate::SequencerAdminAPIError> {
            unreachable!("the admin rpc holds no sequencer client")
        }
        async fn is_conductor_enabled(&self) -> Result<bool, crate::SequencerAdminAPIError> {
            unreachable!("the admin rpc holds no sequencer client")
        }
        async fn is_recovery_mode(&self) -> Result<bool, crate::SequencerAdminAPIError> {
            unreachable!("the admin rpc holds no sequencer client")
        }
        async fn start_sequencer(&self) -> Result<(), crate::SequencerAdminAPIError> {
            unreachable!("the admin rpc holds no sequencer client")
        }
        async fn stop_sequencer(
            &self,
        ) -> Result<alloy_primitives::B256, crate::SequencerAdminAPIError> {
            unreachable!("the admin rpc holds no sequencer client")
        }
        async fn set_recovery_mode(
            &self,
            _mode: bool,
        ) -> Result<(), crate::SequencerAdminAPIError> {
            unreachable!("the admin rpc holds no sequencer client")
        }
        async fn override_leader(&self) -> Result<(), crate::SequencerAdminAPIError> {
            unreachable!("the admin rpc holds no sequencer client")
        }
        async fn reset_derivation_pipeline(&self) -> Result<(), crate::SequencerAdminAPIError> {
            unreachable!("the admin rpc holds no sequencer client")
        }
    }

    /// The pre-created `kona_node_rpc_calls` series must be exactly the ones these handlers emit.
    ///
    /// `kona-gossip` owns the metric and cannot see these emit sites, so this is the only place
    /// the two halves can be compared.
    #[cfg(feature = "metrics")]
    #[test]
    fn rpc_call_metric_is_pre_created_for_every_emitted_method() {
        use crate::{AdminApiServer, AdminRpc, OpP2PApiServer, net::P2pRpc};
        use metrics_util::debugging::DebuggingRecorder;
        use std::collections::BTreeSet;

        fn methods_of(f: impl FnOnce()) -> BTreeSet<String> {
            let recorder = DebuggingRecorder::new();
            let snapshotter = recorder.snapshotter();
            metrics::with_local_recorder(&recorder, f);

            snapshotter
                .snapshot()
                .into_vec()
                .into_iter()
                .map(|(ckey, ..)| ckey)
                .filter(|ckey| ckey.key().name() == kona_gossip::Metrics::RPC_CALLS)
                .map(|ckey| {
                    ckey.key()
                        .labels()
                        .find(|l| l.key() == "method")
                        .map(|l| l.value().to_string())
                        .expect("every emit carries a `method` label")
                })
                .collect()
        }

        // The receivers are dropped, so each handler's first send fails and it returns early.
        // Every one emits before that send.
        // `_`, not `_rx`: a named binding keeps the receiver alive and the handler blocks for
        // ever awaiting a reply.
        let (p2p_tx, _) = tokio::sync::mpsc::channel(1);
        let (admin_tx, _) = tokio::sync::mpsc::channel(1);
        let p2p = P2pRpc::new(p2p_tx);
        let admin = AdminRpc::<NoSequencer>::new(None, admin_tx);

        let runtime = tokio::runtime::Builder::new_current_thread().enable_all().build().unwrap();
        let emitted = methods_of(|| {
            runtime.block_on(async {
                let localhost = std::net::IpAddr::from([127, 0, 0, 1]);
                let subnet: ipnet::IpNet = "127.0.0.0/8".parse().unwrap();

                let _ = p2p.opp2p_self().await;
                let _ = p2p.opp2p_peer_count().await;
                let _ = p2p.opp2p_peers(true).await;
                let _ = p2p.opp2p_peer_stats().await;
                let _ = p2p.opp2p_discovery_table().await;
                let _ = p2p.opp2p_block_peer(String::new()).await;
                let _ = p2p.opp2p_unblock_peer(String::new()).await;
                let _ = p2p.opp2p_list_blocked_peers().await;
                let _ = p2p.opp2p_block_addr(localhost).await;
                let _ = p2p.opp2p_unblock_addr(localhost).await;
                let _ = p2p.opp2p_list_blocked_addrs().await;
                let _ = p2p.opp2p_block_subnet(subnet).await;
                let _ = p2p.opp2p_unblock_subnet(subnet).await;
                let _ = p2p.opp2p_list_blocked_subnets().await;
                let _ = p2p.opp2p_protect_peer(String::new()).await;
                let _ = p2p.opp2p_unprotect_peer(String::new()).await;
                let _ = p2p.opp2p_connect_peer(String::new()).await;
                let _ = p2p.opp2p_disconnect_peer(String::new()).await;

                let payload = op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope::V1(
                    alloy_rpc_types_engine::ExecutionPayloadV1::from_block_slow(
                        &alloy_consensus::Block::<op_alloy_consensus::OpTxEnvelope>::default(),
                    ),
                );
                let _ = admin.admin_post_unsafe_payload(payload).await;
            });
        });

        let pre_created = methods_of(kona_gossip::Metrics::zero);

        assert_eq!(
            emitted, pre_created,
            "every emitted `method` must be pre-created, and nothing else"
        );
    }
}
