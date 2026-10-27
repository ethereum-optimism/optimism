use std::time::Duration;

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

    let block = tokio::time::timeout(Duration::from_secs(10), validator.blocks_rx.recv())
        .await?
        .ok_or_else(|| anyhow::anyhow!("No block received"))?;

    assert_eq!(block, envelope);

    Ok(())
}

/// Blocks gossiped by the sequencer reach a validator connected over the in-memory transport.
#[tokio::test(flavor = "multi_thread")]
async fn test_sequencer_memory_network_conn() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().set_sequencer().in_memory();

    let sequencer = builder.build(vec![]).await;
    let mut validator = builder.build(vec![]).await;

    validator.connect_to(&sequencer).await?;

    validator.is_connected_to_with_retries(&sequencer).await?;
    sequencer.is_connected_to_with_retries(&validator).await?;

    let mut seed_generator = SEED_GENERATOR_BUILDER.next_generator();

    let envelope = seed_generator.random_valid_payload(PayloadVersion::V1)?;

    sequencer.gossip_payload_tx.send(envelope.clone()).await?;

    let block = tokio::time::timeout(Duration::from_secs(10), validator.blocks_rx.recv())
        .await?
        .ok_or_else(|| anyhow::anyhow!("No block received"))?;

    assert_eq!(block, envelope);

    Ok(())
}

/// Blocks gossiped by the sequencer reach every validator of a network set up as a line.
///
/// Each validator only bootstraps from the node before it. The block reaches the far end of the
/// line either over connections discovery adds, or by validators relaying it.
#[tokio::test(flavor = "multi_thread")]
async fn test_sequencer_network_propagation() -> anyhow::Result<()> {
    const VALIDATORS: usize = 10;

    let mut builder = TestNetworkBuilder::new().set_sequencer();

    let sequencer = builder.build(vec![]).await;
    let mut previous_enr = sequencer.peer_enr().await?;

    let mut validators = Vec::new();

    for _ in 0..VALIDATORS {
        let validator = builder.build(vec![previous_enr]).await;

        previous_enr = validator.peer_enr().await?;
        validators.push(validator);
    }

    // Check that each validator is connected to the node before it, so the line is complete.
    let mut previous = &sequencer;
    for validator in &validators {
        validator.is_connected_to_with_retries(previous).await?;
        previous = validator;
    }

    // Send a block to the sequencer.
    let mut seed_generator = SEED_GENERATOR_BUILDER.next_generator();

    let envelope = seed_generator.random_valid_payload(PayloadVersion::V1)?;

    sequencer.gossip_payload_tx.send(envelope.clone()).await?;

    // Check that the block propagates to all validators.
    for validator in &mut validators {
        let block = tokio::time::timeout(Duration::from_secs(10), validator.blocks_rx.recv())
            .await?
            .ok_or_else(|| anyhow::anyhow!("No block received"))?;

        assert_eq!(block, envelope);
    }

    Ok(())
}

/// Admin payloads go to the engine without being signed or gossiped.
#[tokio::test(flavor = "multi_thread")]
async fn admin_payload_is_forwarded_to_engine() -> anyhow::Result<()> {
    use kona_rpc::{AdminApiServer, AdminRpc};
    use std::time::Duration;

    let mut network = TestNetworkBuilder::new().build(vec![]).await;
    let rpc = AdminRpc::new(None, tokio::sync::mpsc::channel(1).0, network.admin_rpc_tx.clone());
    let envelope =
        SEED_GENERATOR_BUILDER.next_generator().random_valid_payload(PayloadVersion::V1)?;
    rpc.admin_post_unsafe_payload(envelope.clone()).await?;
    let forwarded = tokio::time::timeout(Duration::from_secs(10), network.blocks_rx.recv())
        .await?
        .ok_or_else(|| anyhow::anyhow!("No block received"))?;
    assert_eq!(forwarded, envelope);
    Ok(())
}
