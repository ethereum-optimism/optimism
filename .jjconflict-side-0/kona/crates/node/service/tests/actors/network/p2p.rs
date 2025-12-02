use crate::actors::network::mocks::builder::TestNetworkBuilder;

#[tokio::test(flavor = "multi_thread")]
async fn test_p2p_network_conn() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new();
    let network_1 = builder.build(vec![]);
    let enr_1 = network_1.peer_enr().await?;

    let network_2 = builder.build(vec![enr_1]);

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
        let network = builder.build(bootnodes.clone());
        let enr = network.peer_enr().await?;
        networks.push(network);
        bootnodes.push(enr);
    }

    for network in networks.iter() {
        for other_network in networks.iter() {
            if network.peer_id().await? == other_network.peer_id().await? {
                continue;
            }

            network.is_connected_to_with_retries(other_network).await?;
        }
    }

    Ok(())
}
