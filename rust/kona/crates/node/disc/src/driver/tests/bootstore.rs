//! Tests for the bootstore: the records the driver saves, and hands over again.

use super::*;

/// A node with a public-only table filter should dial only the saved records it can reach, as
/// with the records other nodes relay to it, so it should leave out a loopback record in its
/// bootstore. Such a record may have been saved by a version that didn't check the records it
/// dials, or while the node ran on loopback, and nothing says the node it names is on this
/// host.
///
/// At startup, the driver hands the bootstore's records to gossip all at once, without waiting
/// in between, and this node has no bootnodes or peers to find others through. On this
/// single-threaded runtime, the whole batch is in the channel once the test gets the first
/// record. So the batch shows whether the loopback record was left out, without waiting to see
/// that it never comes.
#[tokio::test]
async fn test_public_node_leaves_out_saved_loopback_record() {
    let loopback = localhost_enr(TEST_CHAIN_ID);
    let public = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), TEST_CHAIN_ID);
    let (driver, _dir) = public_driver_with_bootstore(&[loopback, public.clone()]);
    let (_handler, mut records) = driver.start();

    let first = tokio::time::timeout(Duration::from_secs(5), records.recv())
        .await
        .expect("the driver hands over the saved public record")
        .expect("the driver runs");
    let mut batch = vec![first];
    while let Ok(record) = records.try_recv() {
        batch.push(record);
    }
    assert_eq!(batch, [public], "a public node dialed a saved loopback record");
}

/// The driver doesn't wait for room in the channel to gossip, so the saved records that don't fit
/// are dropped until the next store interval. It hands them over in a random order, so that
/// over a few intervals each gets through, rather than only the oldest ones every time.
///
/// With room for one of three records, the chance that one of them is left out of 50 rounds is
/// `3 * (2/3)^50`, about 5 in a billion.
#[test]
fn test_saved_records_take_turns_in_a_full_channel() {
    let mut store = BootStore::in_memory();
    let saved: Vec<Enr> = (1..=3)
        .map(|host| enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, host)), TEST_CHAIN_ID))
        .collect();
    for record in &saved {
        store.add_enr(record.clone());
    }
    let (sender, mut receiver) = mpsc::channel(1);

    let mut handed_over = Vec::new();
    for _ in 0..50 {
        forward_saved(&store, TEST_CHAIN_ID, &sender);
        let record = receiver.try_recv().expect("the driver hands over a record");
        assert!(receiver.try_recv().is_err(), "the channel holds only one");
        if !handed_over.contains(&record) {
            handed_over.push(record);
        }
    }
    assert_eq!(handed_over.len(), saved.len(), "a saved record never got through");
}

/// Gossip redials a peer only when discovery hands it over again, and discovery reports a
/// session only once, when it starts. So the driver should hand the saved records over again
/// every store interval, for gossip to redial the peers it lost.
#[tokio::test]
async fn test_saved_records_are_handed_over_again() {
    let public = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), TEST_CHAIN_ID);
    let (mut driver, _dir) = public_driver_with_bootstore(std::slice::from_ref(&public));
    driver.store_interval = Duration::from_millis(100);
    let (_handler, mut records) = driver.start();

    for _ in 0..2 {
        let record = tokio::time::timeout(Duration::from_secs(5), records.recv())
            .await
            .expect("the driver hands the saved record over")
            .expect("the driver runs");
        assert_eq!(record, public);
    }
}

/// Such a node should also remove the saved records it leaves out from its bootstore, so
/// that they don't stay there for good. The driver saves the bootstore once it has started, and
/// then every minute.
#[tokio::test]
async fn test_public_node_removes_saved_loopback_record() {
    let loopback = localhost_enr(TEST_CHAIN_ID);
    let public = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), TEST_CHAIN_ID);
    let (driver, dir) = public_driver_with_bootstore(&[loopback.clone(), public.clone()]);
    let path = dir.path().join("bootstore.json");
    let (_handler, _records) = driver.start();

    // A read can come back empty, so wait for one that holds the public record too.
    wait_for_saved(&path, |_, saved| saved.contains(&public) && !saved.contains(&loopback))
        .await
        .expect("a public node kept a saved loopback record");
}

/// The bootstore should save only the records a session confirmed, and the bootnodes, not the
/// rest of discv5's table. The table also holds records from other nodes' answers to our
/// lookups, which nothing has shown are at the address in them, and the bootstore would dial
/// them after a restart.
///
/// The node's table filter accepts only loopback records, and every record is on loopback or a
/// documentation address, so it contacts nothing outside this host.
#[tokio::test]
async fn test_bootstore_saves_only_confirmed_records() {
    // Kept in the bootstore, and the documentation address is removed from it at startup, so
    // a read with the one and without the other was written after startup.
    let kept = enr_at(IpAddr::V4(Ipv4Addr::new(127, 0, 0, 3)), TEST_CHAIN_ID);
    let removed = enr_at(IpAddr::V4(Ipv4Addr::new(192, 0, 2, 1)), TEST_CHAIN_ID);
    let (driver, dir) = driver_with_bootstore(
        |enr| enr.ip4().is_some_and(|ip| ip.is_loopback()),
        &[kept.clone(), removed.clone()],
    );
    // As if another node had returned it to one of our lookups.
    let unconfirmed = enr_at(IpAddr::V4(Ipv4Addr::new(127, 0, 0, 2)), TEST_CHAIN_ID);
    driver.disc.add_enr(unconfirmed.clone()).expect("the table takes the record");
    let path = dir.path().join("bootstore.json");
    let (_handler, _records) = driver.start();

    let saved =
        wait_for_saved(&path, |_, saved| saved.contains(&kept) && !saved.contains(&removed))
            .await
            .expect("the driver saves its bootstore at startup");
    assert!(!saved.contains(&unconfirmed), "the bootstore saved an unconfirmed record");
}

/// The default bootstore, `~/.kona/<chain_id>/bootstore.json`, is in a directory that nothing
/// creates, and a path given with `--p2p.bootstore` may be too. The driver should create the
/// directory and save the bootstore there, not keep it only in memory, which loses its records
/// at every restart.
///
/// The node saves its bootnode, at startup.
#[tokio::test]
async fn test_bootstore_is_saved_in_a_missing_directory() {
    let dir = tempdir().unwrap();
    let path = dir.path().join("datadir").join("bootstore.json");
    let bootnode = localhost_enr(TEST_CHAIN_ID);
    let driver = loopback_builder(loopback_config().build())
        .with_bootstore_file(Some(BootStoreFile::Custom(path.clone())))
        .with_bootnodes(BootNodes(vec![BootNode::Enr(bootnode.clone())]))
        .build()
        .expect("the driver builds");
    let (_handler, _records) = driver.start();

    wait_for_saved(&path, |_, saved| saved.contains(&bootnode))
        .await
        .expect("the driver didn't save the bootstore in a directory it had to create");
}

/// A bootstore that can't be opened, here because its path is a directory as in our docker
/// recipes, should stop the node from starting, as op-node's stores do. Otherwise the node
/// runs with a bootstore that it never saves, and nothing says so.
#[tokio::test]
async fn test_bootstore_that_cannot_be_opened_fails_the_build() {
    let dir = tempdir().unwrap();
    let result = loopback_builder(loopback_config().build())
        .with_bootstore_file(Some(BootStoreFile::Custom(dir.path().to_path_buf())))
        .build();

    let err = result.expect_err("the driver built with a bootstore it can't open");
    assert!(
        matches!(err, crate::Discv5BuilderError::Bootstore(BootStoreError::Open { .. })),
        "{err}"
    );
}
