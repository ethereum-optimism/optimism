use crate::actors::{
    generator::{block_builder::PayloadVersion, seed::SEED_GENERATOR_BUILDER},
    network::mocks::builder::TestNetworkBuilder,
};

/// Test that we can properly gossip blocks to the sequencer.
#[tokio::test(flavor = "multi_thread")]
async fn test_sequencer_network_conn() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().set_sequencer();

    let sequencer = builder.build(vec![]).await;
    let enr_1 = sequencer.peer_enr().await?;

    let mut validator = builder.build(vec![enr_1]).await;

    sequencer.is_connected_to_with_retries(&validator).await?;

    validator.is_connected_to_with_retries(&sequencer).await?;

    let mut seed_generator = SEED_GENERATOR_BUILDER.next_generator();

    let envelope = seed_generator.random_valid_payload(PayloadVersion::V1)?;

    sequencer.gossip_payload_tx.send(envelope.clone()).await?;

    let block =
        validator.blocks_rx.recv().await.ok_or_else(|| anyhow::anyhow!("No block received"))?;

    assert_eq!(block, envelope);

    Ok(())
}

/// Test that the network can properly propagate blocks to all connected peers.
///
/// We are setting up a linear network topology, and we check that the block propagates to every
/// node of the network.
#[tokio::test(flavor = "multi_thread")]
async fn test_sequencer_network_propagation() -> anyhow::Result<()> {
    const VALIDATORS: usize = 10;

    let mut builder = TestNetworkBuilder::new().set_sequencer();

    let sequencer = builder.build(vec![]).await;
    let mut previous_enrs = vec![sequencer.peer_enr().await?];

    let mut validators = Vec::new();

    for _ in 0..VALIDATORS {
        let validator = builder.build(previous_enrs.clone()).await;

        previous_enrs.push(validator.peer_enr().await?);
        validators.push(validator);
    }

    // Check that all validators are connected to the sequencer.
    for validator in &validators {
        validator.is_connected_to_with_retries(&sequencer).await?;
    }

    // Send a block to the sequencer.
    let mut seed_generator = SEED_GENERATOR_BUILDER.next_generator();

    let envelope = seed_generator.random_valid_payload(PayloadVersion::V1)?;

    sequencer.gossip_payload_tx.send(envelope.clone()).await?;

    // Check that the block propagates to all validators.
    for validator in &mut validators {
        let block =
            validator.blocks_rx.recv().await.ok_or_else(|| anyhow::anyhow!("No block received"))?;

        assert_eq!(block, envelope);
    }

    Ok(())
}
