//! Tests for gossip validation following the unsafe block signer as it rotates.

use super::*;
use crate::{
    SIGNER_GRACE_PERIOD,
    signer::tests::{address, payload},
};
use tokio::sync::watch;

fn driver(signer: watch::Receiver<Address>) -> GossipDriverStarter<ConnectionGater> {
    GossipDriverBuilder::new(
        RollupConfig {
            l2_chain_id: alloy_chains::Chain::optimism_mainnet(),
            block_time: 2,
            ..Default::default()
        },
        address(1),
        "/ip4/127.0.0.1/tcp/0".parse().unwrap(),
        Keypair::generate_secp256k1(),
    )
    .with_unsafe_block_signer_receiver(signer)
    .build()
    .unwrap()
}

#[tokio::test(start_paused = true)]
async fn idle_gossip_observes_rotation_before_next_payload() {
    let (sender, receiver) = watch::channel(address(1));
    let mut driver = driver(receiver).start().await.expect("listen");
    sender.send(address(2)).unwrap();
    let event = driver.next().await.expect("signer change");
    assert!(matches!(event, SwarmEvent::Behaviour(Event::SignerChanged)));
    driver.handle_event(event);

    // No gossip arrived during grace. Expiry is measured from the observed update,
    // not from this first old-key payload.
    tokio::time::advance(SIGNER_GRACE_PERIOD + Duration::from_nanos(1)).await;
    assert!(driver.handler.validate_signature(&payload(1)).is_err());
    assert!(driver.handler.validate_signature(&payload(2)).is_ok());
}

#[tokio::test]
async fn closed_signer_channel_retains_its_final_value_without_spinning() {
    let (sender, receiver) = watch::channel(address(1));
    let mut driver = driver(receiver).start().await.expect("listen");
    sender.send(address(2)).unwrap();
    drop(sender);
    // A closed channel has nothing more to report, rather than a change on every call.
    let mut next = Box::pin(driver.next());
    assert!(futures::poll!(&mut next).is_pending());
    drop(next);
    // Validation still reads the final value.
    assert!(driver.handler.validate_signature(&payload(2)).is_ok());
    assert!(driver.handler.validate_signature(&payload(1)).is_err());
}

#[tokio::test(start_paused = true)]
async fn signer_update_preserves_duplicate_connection_deadline() {
    let (sender, receiver) = watch::channel(address(1));
    let mut driver = driver(receiver).start().await.expect("listen");
    driver.next_duplicate_check = Some(Instant::now() + Duration::from_secs(1));
    sender.send(address(2)).unwrap();
    let event = driver.next().await.expect("signer change");
    assert!(matches!(event, SwarmEvent::Behaviour(Event::SignerChanged)));
    driver.handle_event(event);

    tokio::time::advance(Duration::from_secs(1)).await;
    let event = driver.next().await;
    assert!(matches!(event, Some(SwarmEvent::Behaviour(Event::DuplicateConnectionsDue))));
    assert!(driver.handler.validate_signature(&payload(2)).is_ok());
}
