//! Tests for banning peers.

use super::*;

/// `BanPeers` bans a peer by its libp2p peer ID, which discovery turns into the node ID of the
/// same key: banning a node also removes it from the table.
#[tokio::test]
async fn test_ban_peers_bans_by_peer_id() {
    let key = CombinedKey::generate_secp256k1();
    let mut builder = Enr::builder();
    builder.ip4(Ipv4Addr::LOCALHOST).udp4(30303).tcp4(9222);
    let enr = builder.build(&key).unwrap();
    // The libp2p peer ID of the same key, at the end of its gossip address.
    let peer_id = match kona_peers::enr_to_multiaddr(&enr).unwrap().pop() {
        Some(libp2p::multiaddr::Protocol::P2p(peer_id)) => peer_id,
        other => panic!("address doesn't end in a peer ID: {other:?}"),
    };

    let driver = offline_driver();
    driver.disc.add_enr(enr.clone()).unwrap();
    let (handler, _enr_recv, _task) = driver.spawn();
    wait_until_started(&handler).await;
    async fn table(handler: &Discv5Handler) -> Vec<Enr> {
        tokio::time::timeout(Duration::from_secs(5), handler.table_enrs()).await.unwrap().unwrap()
    }
    assert!(table(&handler).await.iter().any(|e| e.node_id() == enr.node_id()));

    handler
        .sender
        .send(HandlerRequest::BanPeers {
            peer_ids: [peer_id].into(),
            ban_duration: Duration::from_secs(60),
        })
        .await
        .unwrap();

    assert!(!table(&handler).await.iter().any(|e| e.node_id() == enr.node_id()));
}
