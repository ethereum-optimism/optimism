use crate::actors::{
    generator::{block_builder::PayloadVersion, seed::SEED_GENERATOR_BUILDER},
    network::mocks::builder::TestNetworkBuilder,
};

/// Test that we can properly gossip blocks to the sequencer.
#[tokio::test(flavor = "multi_thread")]
async fn test_sequencer_network_conn() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().set_sequencer();

    let sequencer_network = builder.build(vec![]);
    let enr_1 = sequencer_network.peer_enr().await?;

    let mut validator_network = builder.build(vec![enr_1]);

    sequencer_network.is_connected_to_with_retries(&validator_network).await?;

    validator_network.is_connected_to_with_retries(&sequencer_network).await?;

    let mut seed_generator = SEED_GENERATOR_BUILDER.next_generator();

    let envelope = seed_generator.random_valid_payload(PayloadVersion::V1)?;

    sequencer_network.inbound_data.gossip_payload_tx.send(envelope.clone()).await?;

    let block =
        validator_network.blocks_rx.recv().await.ok_or(anyhow::anyhow!("No block received"))?;

    assert_eq!(block.parent_beacon_block_root, envelope.parent_beacon_block_root);
    assert_eq!(block.execution_payload, envelope.execution_payload);

    Ok(())
}

/// Test that the network can properly propagate blocks to all connected peers.
///
/// We are setting up a linear network topology, and we check that the block propagates to every
/// block of the network.
#[tokio::test(flavor = "multi_thread")]
async fn test_sequencer_network_propagation() -> anyhow::Result<()> {
    const NETWORKS: usize = 10;

    let mut builder = TestNetworkBuilder::new().set_sequencer();

    let sequencer_network = builder.build(vec![]);
    let mut previous_enrs = vec![sequencer_network.peer_enr().await?];

    let mut validator_networks = Vec::new();

    for _ in 0..NETWORKS {
        let network = builder.build(previous_enrs.clone());

        previous_enrs.push(network.peer_enr().await?);
        validator_networks.push(network);
    }

    // Check that all networks are connected to the sequencer.
    for network in validator_networks.iter() {
        network.is_connected_to_with_retries(&sequencer_network).await?;
    }

    // Send a block to the sequencer.
    let mut seed_generator = SEED_GENERATOR_BUILDER.next_generator();

    let envelope = seed_generator.random_valid_payload(PayloadVersion::V1)?;

    sequencer_network.inbound_data.gossip_payload_tx.send(envelope.clone()).await?;

    // Check that the block propagates to all networks.
    for network in validator_networks.iter_mut() {
        let block = network.blocks_rx.recv().await.ok_or(anyhow::anyhow!("No block received"))?;

        assert_eq!(block.parent_beacon_block_root, envelope.parent_beacon_block_root);
        assert_eq!(block.execution_payload, envelope.execution_payload);
    }

    Ok(())
}
