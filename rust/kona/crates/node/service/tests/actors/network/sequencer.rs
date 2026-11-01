use std::time::Duration;

use crate::actors::{
    generator::{block_builder::PayloadVersion, seed::SEED_GENERATOR_BUILDER},
    network::mocks::builder::{TestNetworkBuilder, signer_address},
};
use kona_gossip::PeerAddr;
use libp2p::identity::Keypair;

/// Test that we can properly gossip blocks to the sequencer.
#[tokio::test(flavor = "multi_thread")]
async fn test_sequencer_network_conn() -> anyhow::Result<()> {
    let mut builder = TestNetworkBuilder::new().set_sequencer();

    let sequencer = builder.build(vec![]).await;
    let enr_1 = sequencer.peer_enr().await?;

    let mut validator = builder.build(vec![enr_1]).await;

    sequencer.check_connected_to_with_retries(&validator).await?;

    validator.check_connected_to_with_retries(&sequencer).await?;

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

    validator.check_connected_to_with_retries(&sequencer).await?;
    sequencer.check_connected_to_with_retries(&validator).await?;

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
        validator.check_connected_to_with_retries(previous).await?;
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

/// A hub relays blocks between spokes that only know the hub: each spoke's only static peer is the
/// hub, the hub lists none, and nobody runs discovery. The spokes never connect to each other, so
/// a block from one can only reach the other through the hub.
#[tokio::test(flavor = "multi_thread")]
async fn test_hub_relays_blocks_between_spokes() -> anyhow::Result<()> {
    // Every node accepts and relays only the sequencer's blocks: spoke A will sign with this key.
    let sequencer_keypair = Keypair::generate_secp256k1();
    let mut builder = TestNetworkBuilder::new()
        .in_memory()
        .with_unsafe_block_signer(signer_address(&sequencer_keypair));

    // The hub lists no static peers, and gets a random key.
    let hub = builder.build(vec![]).await;
    let hub_peer = PeerAddr::try_from(hub.listen_addr().await?)?;

    // Static peers and the keypair only apply to the next node built, so they are set per spoke.
    let builder = builder.with_static_peers(vec![hub_peer.clone()]);
    let mut builder = builder.with_custom_keypair(sequencer_keypair);
    let spoke_a = builder.build(vec![]).await;

    // Spoke B gets a random key.
    let mut builder = builder.with_static_peers(vec![hub_peer]);
    let mut spoke_b = builder.build(vec![]).await;

    // Publish only once the block can travel the whole way, from A to the hub and on to B.
    spoke_a.gossips_blocks_with_retries(&hub).await?;
    hub.gossips_blocks_with_retries(&spoke_b).await?;

    let envelope =
        SEED_GENERATOR_BUILDER.next_generator().random_valid_payload(PayloadVersion::V1)?;
    spoke_a.gossip_payload_tx.send(envelope.clone()).await?;

    let block = tokio::time::timeout(Duration::from_secs(10), spoke_b.blocks_rx.recv())
        .await?
        .ok_or_else(|| anyhow::anyhow!("No block received"))?;
    assert_eq!(block, envelope);

    // The spokes are only connected to the hub.
    for (spoke, other) in [(&spoke_a, &spoke_b), (&spoke_b, &spoke_a)] {
        assert!(!spoke.is_connected_to(other).await?);
        assert_eq!(spoke.peers().await?.peers.len(), 1);
    }
    for node in [&hub, &spoke_a, &spoke_b] {
        assert!(node.has_discovery_disabled().await);
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
