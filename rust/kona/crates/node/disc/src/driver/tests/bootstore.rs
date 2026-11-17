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
    let loopback = enr_at_localhost(TEST_CHAIN_ID);
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

/// Gossip dials a saved record at its TCP address, so the driver should hand over only the saved
/// records that have one, as it does with the records of sessions. Otherwise gossip fails to dial
/// one without, and counts the failure, at every store interval. The bootstore may hold such
/// records: an earlier version saved records from discovery's table, a discovery-only bootnode
/// has only a UDP address, and a hand-edited file may hold anything.
#[test]
fn test_saved_records_without_a_tcp_address_are_not_handed_over() {
    let dialable = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), TEST_CHAIN_ID);
    // Ports, without an IP, as a node's own record before its peers' votes give it one.
    let without_ip = enr_at(IpAddr::V4(Ipv4Addr::UNSPECIFIED), TEST_CHAIN_ID);
    let udp_only = crate::test_utils::enr_with(TEST_CHAIN_ID, |record| {
        record.ip4(Ipv4Addr::new(1, 1, 1, 2)).udp4(9222);
    });
    let mut store = BootStore::in_memory();
    for record in [&dialable, &without_ip, &udp_only] {
        store.add_enr(record.clone());
    }
    let (sender, mut receiver) = mpsc::channel(8);

    forward_saved(&store, TEST_CHAIN_ID, &sender);

    let mut handed_over = Vec::new();
    while let Ok(record) = receiver.try_recv() {
        handed_over.push(record);
    }
    assert_eq!(handed_over, [dialable], "the driver handed over a record gossip can't dial");
}

/// The saved records handed over again every store interval should be only those gossip reached
/// before, not the candidates, which only sessions confirmed: anyone can start sessions under
/// fresh keys, and gossip would redial the candidates of a flood every minute. A candidate is
/// dialed when its session starts, and after a restart the saved candidates seed the table, so
/// they are dialed once a session with them starts again.
#[test]
fn test_candidates_are_not_handed_over() {
    let reached = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), TEST_CHAIN_ID);
    let candidate = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 2)), TEST_CHAIN_ID);
    let store = store_with_tiers(std::slice::from_ref(&reached), &[candidate]);
    let (sender, mut receiver) = mpsc::channel(8);

    forward_saved(&store, TEST_CHAIN_ID, &sender);

    let mut handed_over = Vec::new();
    while let Ok(record) = receiver.try_recv() {
        handed_over.push(record);
    }
    assert_eq!(handed_over, [reached], "the driver handed over a candidate");
}

/// Gossip redials a lost peer only for a day, as op-node keeps a discovered address for 24
/// hours: a connection refreshes its record's evidence, so the evidence of a lost peer dates from
/// about when gossip lost it, and redialing it every minute after that long only fails. The day
/// counts from now, not from the record gossip reached last: after a restart, the saved records
/// seed the table instead.
#[test]
fn test_records_lost_a_day_ago_are_not_handed_over() {
    let lost = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), TEST_CHAIN_ID);
    let mut store = store_with_tiers(&[], std::slice::from_ref(&lost));
    reach(&mut store, &lost, SystemTime::now() - Duration::from_secs(25 * 60 * 60));
    let (sender, mut receiver) = mpsc::channel(8);

    forward_saved(&store, TEST_CHAIN_ID, &sender);

    assert!(receiver.try_recv().is_err(), "the driver handed over a peer lost a day ago");
}

/// Session records go through the same channel to gossip, and the driver hands a session's
/// record over only once, when the session starts. So the saved records should leave half of
/// the channel free, rather than fill it at every store interval, which drops the session records
/// that arrive until gossip catches up.
#[test]
fn test_saved_records_leave_room_for_session_records() {
    let mut store = BootStore::in_memory();
    for host in 1..=4 {
        store.add_enr(enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, host)), TEST_CHAIN_ID));
    }
    let (sender, _receiver) = mpsc::channel(4);

    forward_saved(&store, TEST_CHAIN_ID, &sender);

    let free = sender.capacity();
    assert_eq!(free, 2, "the saved records left {free} of 4 places in the channel free");
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
    let loopback = enr_at_localhost(TEST_CHAIN_ID);
    let public = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), TEST_CHAIN_ID);
    let (driver, dir) = public_driver_with_bootstore(&[loopback.clone(), public.clone()]);
    let path = dir.path().join("bootstore.json");
    let (_handler, _records) = driver.start();

    wait_for_bootstore(&path, |saved| saved.contains(&public) && !saved.contains(&loopback))
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
        wait_for_bootstore(&path, |saved| saved.contains(&kept) && !saved.contains(&removed))
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
    let bootnode = enr_at_localhost(TEST_CHAIN_ID);
    let driver = loopback_builder(loopback_config().build())
        .with_bootstore_file(Some(BootStoreFile::Custom(path.clone())))
        .with_bootnodes(BootNodes(vec![BootNode::Enr(bootnode.clone())]))
        .build()
        .expect("the driver builds");
    assert!(path.is_file(), "building the driver didn't create the bootstore");
    let (_handler, _records) = driver.start();

    wait_for_bootstore(&path, |saved| saved.contains(&bootnode))
        .await
        .expect("the driver didn't save the bootstore in a directory it had to create");
}

/// After a restart, the saved records should seed discovery's table, as geth seeds its table
/// from its node database. Otherwise only the bootnodes do, many of which are dead (#24094), and
/// a node whose bootnodes are all down finds no peers through discovery, although its bootstore
/// names peers that answered before.
///
/// The node has no bootnodes, and its saved records are on loopback, where nothing answers.
#[tokio::test]
async fn test_saved_records_seed_the_table() {
    let saved = [enr_at_localhost(TEST_CHAIN_ID), enr_at_localhost(TEST_CHAIN_ID)];
    let (driver, _dir) = driver_with_bootstore(accept_all, &saved);
    let (handler, _records) = driver.start();
    wait_until_started(&handler).await;

    let table = handler.table_enrs().await.expect("the driver runs");
    for record in &saved {
        assert!(
            table.iter().any(|enr| enr.node_id() == record.node_id()),
            "a saved record didn't seed the table"
        );
    }
}

/// A node with many saved records should seed only [`SEED_COUNT`] of them, as geth does: lookups
/// find the rest, and a table full of records that no longer answer delays the ones that do.
///
/// Without the limit, most of these would fit: discv5's buckets hold 16 records each, and half
/// of the records fall into the farthest one, a quarter into the next, and so on.
#[tokio::test]
async fn test_seeds_are_limited() {
    let saved: Vec<_> = (0..2 * SEED_COUNT).map(|_| enr_at_localhost(TEST_CHAIN_ID)).collect();
    let (driver, _dir) = driver_with_bootstore(accept_all, &saved);
    let (handler, _records) = driver.start();
    wait_until_started(&handler).await;

    let table = handler.table_enrs().await.expect("the driver runs");
    assert!(table.len() <= SEED_COUNT, "{} saved records seeded the table", table.len());
}

/// A store with `reached` records that gossip reached, and `candidates` without that evidence.
fn store_with_tiers(reached: &[Enr], candidates: &[Enr]) -> BootStore {
    let mut store = BootStore::in_memory();
    for record in reached.iter().chain(candidates) {
        store.add_enr(record.clone());
    }
    for record in reached {
        reach(&mut store, record, SystemTime::now());
    }
    store
}

/// Notes in `store` that gossip reached the node of `record`, which it holds, at its TCP address
/// at `at`.
fn reach(store: &mut BootStore, record: &Enr, at: SystemTime) {
    let addr = kona_peers::enr_tcp_socket(record).expect("the record has a TCP address");
    store.set_tcp_verified(record.node_id(), addr, at);
}

/// The records gossip reached should seed the table before the candidates: anyone can start
/// sessions under fresh keys, so a flood of candidates shouldn't take the seeds of the peers
/// gossip reached.
#[test]
fn test_reached_records_seed_before_candidates() {
    let reached: Vec<_> = (0..3).map(|_| enr_at_localhost(TEST_CHAIN_ID)).collect();
    let candidates: Vec<_> = (0..3).map(|_| enr_at_localhost(TEST_CHAIN_ID)).collect();
    let store = store_with_tiers(&reached, &candidates);

    let order = seed_order(&store, TEST_CHAIN_ID);
    assert_eq!(order.len(), reached.len() + candidates.len());
    let (first, rest) = order.split_at(reached.len());
    assert!(first.iter().all(|enr| reached.contains(enr)), "a candidate seeded before a peer");
    assert!(rest.iter().all(|enr| candidates.contains(enr)));
}

/// Within a tier, the seeds should be drawn at random, not the most recently confirmed first:
/// those are the peers the node was connected to when it stopped, and if they were all an
/// attacker's, seeding them first would eclipse the node again.
///
/// The chance that one of three records never comes first in 50 rounds is `3 * (2/3)^50`, about
/// 5 in a billion.
#[test]
fn test_seeds_are_drawn_at_random() {
    let reached: Vec<_> = (0..3).map(|_| enr_at_localhost(TEST_CHAIN_ID)).collect();
    let store = store_with_tiers(&reached, &[]);

    let mut first = Vec::new();
    for _ in 0..50 {
        let seed = seed_order(&store, TEST_CHAIN_ID)[0].clone();
        if !first.contains(&seed) {
            first.push(seed);
        }
    }
    assert_eq!(first.len(), reached.len(), "a saved record never seeded first");
}

/// A record gossip reached long before the others should seed with the candidates, not before
/// them, much as geth seeds only nodes that answered within five days. Otherwise a peer reached
/// once, months ago, comes before every candidate, however recent, until the store evicts it.
///
/// The age counts from the newest record gossip reached, about when the node stopped, not from
/// now: a node that was down for a month should still seed the peers it reached last first.
///
/// The chance that the candidate never comes before the old record in 50 rounds is `(1/2)^50`.
#[rstest::rstest]
#[case::running(Duration::ZERO)]
#[case::after_downtime(Duration::from_secs(30 * 24 * 60 * 60))]
fn test_records_reached_long_ago_seed_with_candidates(#[case] downtime: Duration) {
    let stopped = SystemTime::now() - downtime;
    let recent = enr_at_localhost(TEST_CHAIN_ID);
    let old = enr_at_localhost(TEST_CHAIN_ID);
    let candidate = enr_at_localhost(TEST_CHAIN_ID);
    let mut store = store_with_tiers(&[], &[recent.clone(), old.clone(), candidate.clone()]);
    reach(&mut store, &recent, stopped);
    reach(&mut store, &old, stopped - SEED_MAX_AGE - Duration::from_secs(60 * 60));

    let mut candidate_before_old = false;
    for _ in 0..50 {
        let order = seed_order(&store, TEST_CHAIN_ID);
        assert_eq!(order[0], &recent, "a record reached long ago seeded before a recent one");
        candidate_before_old |= order[1] == &candidate;
    }
    assert!(candidate_before_old, "a record reached long ago always seeded before a candidate");
}

/// A record dated after now, written while the clock ran ahead, shouldn't push the records
/// reached since out of the first tier: the age of a record counts from the most recently reached
/// one only up to now.
#[test]
fn test_a_record_dated_after_now_doesnt_age_the_others() {
    let now = SystemTime::now();
    let ahead = enr_at_localhost(TEST_CHAIN_ID);
    let recent = enr_at_localhost(TEST_CHAIN_ID);
    let candidate = enr_at_localhost(TEST_CHAIN_ID);
    let mut store = store_with_tiers(&[], &[ahead.clone(), recent.clone(), candidate.clone()]);
    reach(&mut store, &ahead, now + 2 * SEED_MAX_AGE);
    reach(&mut store, &recent, now - Duration::from_secs(60 * 60));

    for _ in 0..50 {
        let order = seed_order(&store, TEST_CHAIN_ID);
        assert_eq!(order[2], &candidate, "a record dated after now aged one reached since");
    }
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

/// The table entry of `enr`, in `state`.
fn table_entry(enr: &Enr, state: discv5::ConnectionState) -> (NodeId, Enr, NodeStatus) {
    let status = NodeStatus { direction: discv5::ConnectionDirection::Outgoing, state };
    (enr.node_id(), enr.clone(), status)
}

/// At each store interval, a node should note that its sessions with the nodes discovery holds
/// as connected still work, as discv5 pings them: a session confirms its record only once it has
/// lasted the refresh delay, and its evidence would otherwise be no more recent than that. The
/// table may hold a newer record of the node, which isn't saved, as that would need the checks a
/// session's record gets: the saved record gets the evidence only if the newer one keeps its UDP
/// address, which discv5 pings.
#[rstest::rstest]
#[case::connected(discv5::ConnectionState::Connected, Newer::Same, 120, true)]
#[case::connected_too_briefly(discv5::ConnectionState::Connected, Newer::Same, 30, false)]
#[case::disconnected(discv5::ConnectionState::Disconnected, Newer::Same, 120, false)]
#[case::newer_with_another_udp_port(discv5::ConnectionState::Connected, Newer::UdpPort, 120, false)]
#[case::newer_with_another_tcp_port(discv5::ConnectionState::Connected, Newer::TcpPort, 120, true)]
fn test_store_interval_refreshes_connected_sessions(
    #[case] state: discv5::ConnectionState,
    #[case] newer: Newer,
    #[case] session_secs: u64,
    #[case] refreshed: bool,
) {
    let key = CombinedKey::generate_secp256k1();
    let saved = enr_signed_with(&key, TEST_CHAIN_ID, |record| {
        record.ip4(Ipv4Addr::new(1, 1, 1, 1)).udp4(9222).tcp4(9223);
    });
    let mut in_table = saved.clone();
    match newer {
        Newer::Same => {}
        Newer::UdpPort => {
            in_table.set_udp4(9000, &key).unwrap();
        }
        Newer::TcpPort => {
            in_table.set_tcp4(9000, &key).unwrap();
        }
    }
    let mut store = BootStore::in_memory().with_verified_refresh(Duration::from_secs(60));
    store.add_enr(saved.clone());
    let now = SystemTime::now();
    let socket = SocketAddr::V4(saved.udp4_socket().unwrap());
    store.session_started(saved.node_id(), socket, now - Duration::from_secs(session_secs));

    refresh_connected(&mut store, &[table_entry(&in_table, state)], now);

    let entry = store.get(saved.node_id()).expect("the record stays saved");
    assert_eq!(entry.enr(), &saved);
    assert_eq!(entry.udp_verified_at(), refreshed.then_some(now));
}

/// The record the table holds of a node, compared with the saved one.
#[derive(std::fmt::Debug)]
enum Newer {
    /// The same record.
    Same,
    /// A newer record with another UDP port.
    UdpPort,
    /// A newer record with another TCP port.
    TcpPort,
}

/// A connected node in the table whose record the node didn't save gets no record: only the
/// records sessions bring are saved, after their checks.
#[test]
fn test_store_interval_saves_no_unsaved_connected_node() {
    let mut store = BootStore::in_memory();
    let record = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), TEST_CHAIN_ID);

    refresh_connected(
        &mut store,
        &[table_entry(&record, discv5::ConnectionState::Connected)],
        SystemTime::now(),
    );

    assert!(store.is_empty());
}

/// A session that has just started shouldn't confirm its record's UDP address: anyone can start
/// one, under a fresh key, and a full store keeps the candidates confirmed most recently, so a
/// flood of fresh sessions would push out the candidates whose sessions last. Only a session that
/// discovery still holds once the refresh delay has passed should, much as geth saves a node only
/// once it has answered several pings. So right after a session, its saved record has no
/// `udp_verified_at`.
///
/// N accepts every record, and saves its bootstore every 100 ms. L has N as its bootnode, given
/// by an enode URL, so L's bootstrap starts a session with N, and N saves L's record.
#[tokio::test]
async fn test_a_new_session_doesnt_confirm_its_record() {
    let (mut node, dir) = driver_with_bootstore(accept_all, &[]);
    node.store_interval = Duration::from_millis(100);
    let path = dir.path().join("bootstore.json");
    let enode = loopback_enode(&node.disc.local_enr());
    let (_node, _records) = node.start();

    let mut loopback = offline_driver();
    loopback.bootnodes = BootNodes(vec![BootNode::Enode(enode)]);
    let loopback_id = loopback.disc.local_enr().node_id();
    let (_loopback, _) = loopback.start();

    wait_for_bootstore(&path, |saved| saved.iter().any(|enr| enr.node_id() == loopback_id))
        .await
        .expect("the node saves the record of its session");
    let saved = BootStore::open(path).expect("the bootstore opens");
    let entry = saved.get(loopback_id).expect("the record is saved");
    assert_eq!(
        entry.udp_verified_at(),
        None,
        "a session that had just started confirmed its record"
    );
}

/// A session's record is confirmed at a store interval once the session has lasted the refresh
/// delay: the driver checks discovery's table at each one. With no delay, the first store
/// interval after the session confirms it.
///
/// As in [`test_a_new_session_doesnt_confirm_its_record`], L's bootstrap starts a session with N,
/// and N saves L's record.
#[tokio::test]
async fn test_store_interval_confirms_a_lasting_session() {
    let (mut node, dir) = driver_with_bootstore(accept_all, &[]);
    node.store_interval = Duration::from_millis(100);
    node.bootstore = std::mem::replace(&mut node.bootstore, BootStore::in_memory())
        .with_verified_refresh(Duration::ZERO);
    let path = dir.path().join("bootstore.json");
    let enode = loopback_enode(&node.disc.local_enr());
    let (_node, _records) = node.start();

    let mut loopback = offline_driver();
    loopback.bootnodes = BootNodes(vec![BootNode::Enode(enode)]);
    let loopback_id = loopback.disc.local_enr().node_id();
    let (_loopback, _) = loopback.start();

    let confirmed = async {
        loop {
            let saved = BootStore::open(path.clone()).expect("the bootstore opens");
            if saved.get(loopback_id).is_some_and(|entry| entry.udp_verified_at().is_some()) {
                return;
            }
            tokio::time::sleep(Duration::from_millis(50)).await;
        }
    };
    tokio::time::timeout(Duration::from_secs(5), confirmed)
        .await
        .expect("a store interval didn't confirm the record of a session that lasted");
}
