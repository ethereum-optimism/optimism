use crate::actors::{
    generator::{block_builder::PayloadVersion, seed::SEED_GENERATOR_BUILDER},
    network::{TestNetwork, mocks::builder::TestNetworkBuilder},
};
use kona_node_service::network;
use std::time::Duration;

/// Test that we can properly gossip blocks to the sequencer.
#[tokio::test(flavor = "multi_thread")]
async fn test_sequencer_network_conn() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().set_sequencer();

    let sequencer_network = builder.build(vec![]).await;
    let enr_1 = sequencer_network.peer_enr().await?;

    let mut validator_network = builder.build(vec![enr_1]).await;

    sequencer_network.is_connected_to_with_retries(&validator_network).await?;

    validator_network.is_connected_to_with_retries(&sequencer_network).await?;

    let mut seed_generator = SEED_GENERATOR_BUILDER.next_generator();

    let envelope = seed_generator.random_valid_payload(PayloadVersion::V1)?;

    sequencer_network.signer.send(envelope.clone()).await?;

    let block = validator_network
        .blocks_rx
        .recv()
        .await
        .ok_or_else(|| anyhow::anyhow!("No block received"))?;

    assert_eq!(block, envelope);

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

    let sequencer_network = builder.build(vec![]).await;
    let mut previous_enrs = vec![sequencer_network.peer_enr().await?];

    let mut validator_networks = Vec::new();

    for _ in 0..NETWORKS {
        let network = builder.build(previous_enrs.clone()).await;

        previous_enrs.push(network.peer_enr().await?);
        validator_networks.push(network);
    }

    // Check that all networks are connected to the sequencer.
    for network in &validator_networks {
        network.is_connected_to_with_retries(&sequencer_network).await?;
    }

    // Send a block to the sequencer.
    let mut seed_generator = SEED_GENERATOR_BUILDER.next_generator();

    let envelope = seed_generator.random_valid_payload(PayloadVersion::V1)?;

    sequencer_network.signer.send(envelope.clone()).await?;

    // Check that the block propagates to all networks.
    for network in &mut validator_networks {
        let block =
            network.blocks_rx.recv().await.ok_or_else(|| anyhow::anyhow!("No block received"))?;

        assert_eq!(block, envelope);
    }

    Ok(())
}

/// A validator whose engine has stopped stops its network actor when the next block arrives.
#[tokio::test(flavor = "multi_thread")]
async fn a_stopped_engine_stops_the_network_actor() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().set_sequencer();

    let sequencer_network = builder.build(vec![]).await;
    let validator_network = builder.build(vec![sequencer_network.peer_enr().await?]).await;
    sequencer_network.is_connected_to_with_retries(&validator_network).await?;
    validator_network.is_connected_to_with_retries(&sequencer_network).await?;

    let TestNetwork { blocks_rx, handle, .. } = validator_network;
    drop(blocks_rx);
    let envelope =
        SEED_GENERATOR_BUILDER.next_generator().random_valid_payload(PayloadVersion::V1)?;
    sequencer_network.signer.send(envelope).await?;

    let result = tokio::time::timeout(Duration::from_secs(60), handle).await??;
    assert!(matches!(result, Err(network::ActorError::EngineClosed)));
    Ok(())
}
