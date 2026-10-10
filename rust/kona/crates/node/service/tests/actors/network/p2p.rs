use crate::actors::network::mocks::builder::TestNetworkBuilder;
use kona_rpc::OpP2PApiServer;

#[tokio::test(flavor = "multi_thread")]
async fn test_p2p_network_conn() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new();
    let network_1 = builder.build(vec![]).await;
    let enr_1 = network_1.peer_enr().await?;

    let network_2 = builder.build(vec![enr_1]).await;

    network_2.is_connected_to_with_retries(&network_1).await?;

    network_1.is_connected_to_with_retries(&network_2).await?;

    Ok(())
}

#[tokio::test(flavor = "multi_thread")]
async fn test_large_network_conn() -> anyhow::Result<()> {
    const NETWORKS: usize = 10;

    let mut builder = TestNetworkBuilder::new();

    let (mut networks, mut bootnodes) = (vec![], vec![]);

    for _ in 0..NETWORKS {
        let network = builder.build(bootnodes.clone()).await;
        let enr = network.peer_enr().await?;
        networks.push(network);
        bootnodes.push(enr);
    }

    for network in &networks {
        for other_network in &networks {
            if network.peer_id().await? == other_network.peer_id().await? {
                continue;
            }

            network.is_connected_to_with_retries(other_network).await?;
        }
    }

    Ok(())
}

#[tokio::test(flavor = "multi_thread")]
async fn rpc_commands_publish_state_before_returning() -> anyhow::Result<()> {
    let network = TestNetworkBuilder::new().build(vec![]).await;
    let rpc = &network.p2p_rpc;
    let peer = libp2p::identity::Keypair::generate_secp256k1().public().to_peer_id().to_string();
    let address = "192.0.2.1".parse()?;
    let subnet = "192.0.2.0/24".parse()?;

    tokio::time::timeout(std::time::Duration::from_secs(10), async {
        rpc.opp2p_block_peer(peer.clone()).await?;
        assert_eq!(rpc.opp2p_list_blocked_peers().await?, vec![peer.clone()]);
        rpc.opp2p_block_addr(address).await?;
        assert_eq!(rpc.opp2p_list_blocked_addrs().await?, vec![address]);
        rpc.opp2p_block_subnet(subnet).await?;
        assert_eq!(rpc.opp2p_list_blocked_subnets().await?, vec![subnet]);
        rpc.opp2p_unblock_peer(peer).await?;
        rpc.opp2p_unblock_addr(address).await?;
        rpc.opp2p_unblock_subnet(subnet).await?;
        assert!(rpc.opp2p_list_blocked_peers().await?.is_empty());
        assert!(rpc.opp2p_list_blocked_addrs().await?.is_empty());
        assert!(rpc.opp2p_list_blocked_subnets().await?.is_empty());
        Ok::<(), anyhow::Error>(())
    })
    .await??;
    Ok(())
}
