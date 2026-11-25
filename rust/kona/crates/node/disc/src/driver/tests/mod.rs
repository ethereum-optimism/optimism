//! Tests for the [`Discv5Driver`], one file per topic, with the helpers they share.

use super::*;
use crate::test_utils::{TEST_CHAIN_ID, enr_at, enr_at_localhost, enr_signed_with, enr_with};
use discv5::{ConfigBuilder, ListenConfig};
use enr::CombinedKey;
use kona_genesis::{OP_MAINNET_CHAIN_ID, OP_SEPOLIA_CHAIN_ID};
use kona_peers::{BootStoreError, BootStoreFile};
use tempfile::tempdir;

use std::net::{IpAddr, Ipv4Addr, Ipv6Addr, SocketAddr};

mod bans;
mod bootnodes;
mod bootstore;
mod dial;
mod lifecycle;
mod local_enr;

/// A table filter that accepts every record, for the tests of the other rules.
fn accept_all(_: &Enr) -> bool {
    true
}

/// Builds a driver listening on localhost that does not contact any remote peer.
fn offline_driver() -> Discv5Driver {
    loopback_driver(loopback_config().build())
}

/// A discv5 config that listens on a free port on loopback, for [`loopback_driver`].
fn loopback_config() -> ConfigBuilder {
    ConfigBuilder::new(SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), 0).into())
}

/// Builds a driver on [`TEST_CHAIN_ID`] that advertises loopback, with `config`, which should
/// listen there too, see [`loopback_config`]. It has no bootnodes.
fn loopback_driver(config: Config) -> Discv5Driver {
    loopback_builder(config).build().expect("the driver builds")
}

/// A builder of the driver [`loopback_driver`] builds, for the settings it leaves out.
fn loopback_builder(config: Config) -> Discv5Builder {
    let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
        unreachable!()
    };
    Discv5Driver::builder(
        LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::LOCALHOST), 0, 0),
        TEST_CHAIN_ID,
        config,
    )
}

/// A driver with [`crate::public_only`] as its table filter, with a bootstore of `records`, see
/// [`driver_with_bootstore`].
fn public_driver_with_bootstore(records: &[Enr]) -> (Discv5Driver, tempfile::TempDir) {
    driver_with_bootstore(crate::public_only, records)
}

/// A driver with `table_filter`, and a bootstore of `records` in a new temporary directory,
/// which is removed when the returned guard drops. The node has no bootnodes, and listens on
/// loopback.
fn driver_with_bootstore(
    table_filter: fn(&Enr) -> bool,
    records: &[Enr],
) -> (Discv5Driver, tempfile::TempDir) {
    let dir = tempdir().unwrap();
    let mut store =
        BootStore::open(dir.path().join("bootstore.json")).expect("the bootstore opens");
    for record in records {
        store.add_enr(record.clone());
    }

    let mut driver = loopback_driver(loopback_config().table_filter(table_filter).build());
    driver.bootstore = store;
    (driver, dir)
}

/// Notes in `store` that gossip reached the node of `record`, which it holds, at its TCP address
/// at `at`.
fn reach(store: &mut BootStore, record: &Enr, at: SystemTime) {
    let addr = kona_peers::enr_tcp_socket(record).expect("the record has a TCP address");
    store.set_tcp_verified(record.node_id(), addr, at);
}

/// Tells the running driver of `handler`, as gossip does, that gossip reached the node of `record`
/// at its TCP address.
fn reach_through(handler: &Discv5Handler, record: &Enr) {
    let addr = kona_peers::enr_tcp_socket(record).expect("the record has a TCP address");
    handler.tcp_verified(record.node_id(), addr);
}

/// Reads the bootstore at `path` until `done` holds for the records it holds, and returns those,
/// or `None` after 5s. A missing file holds no records.
async fn wait_for_bootstore(
    path: &std::path::Path,
    done: impl Fn(&[Enr]) -> bool,
) -> Option<Vec<Enr>> {
    let poll = async {
        loop {
            let store = BootStore::open(path.to_path_buf()).expect("the bootstore opens");
            let saved = store.iter().cloned().collect::<Vec<_>>();
            if done(&saved) {
                return saved;
            }
            tokio::time::sleep(Duration::from_millis(50)).await;
        }
    };
    tokio::time::timeout(Duration::from_secs(5), poll).await.ok()
}

/// The enode URL of the node with `enr`, at its UDP port on loopback.
fn loopback_enode(enr: &Enr) -> Multiaddr {
    let peer_id = kona_peers::enr_to_multiaddr(enr)
        .and_then(|addr| addr.iter().last().map(|protocol| protocol.acquire()))
        .expect("the record has a peer ID");
    Multiaddr::from(Ipv4Addr::LOCALHOST)
        .with(Protocol::Udp(enr.udp4().expect("the record has a UDP port")))
        .with(Protocol::Tcp(0))
        .with(peer_id)
}

/// Reads the discovery table of the driver of `handler` until `done` holds for the records it
/// holds, and returns those, or `None` after 5s. The driver seeds the table at its first store
/// interval, which may come after the first requests.
async fn wait_for_table(
    handler: &Discv5Handler,
    done: impl Fn(&[Enr]) -> bool,
) -> Option<Vec<Enr>> {
    let poll = async {
        loop {
            let table = handler.table_enrs().await.expect("the driver runs");
            if done(&table) {
                return table;
            }
            tokio::time::sleep(Duration::from_millis(50)).await;
        }
    };
    tokio::time::timeout(Duration::from_secs(5), poll).await.ok()
}

/// Waits until the driver has finished startup and runs its main loop.
async fn wait_until_started(handler: &Discv5Handler) {
    // Requests are only answered by the main loop.
    tokio::time::timeout(Duration::from_secs(5), handler.peer_count())
        .await
        .expect("driver should finish startup")
        .expect("driver should not stop during startup");
}

/// The time the tests of the driver's functions pass them as now, rather than read the clock.
fn fixed_now() -> SystemTime {
    SystemTime::UNIX_EPOCH + Duration::from_secs(1_000_000_000)
}

/// A store with `reached` records that gossip reached at [`fixed_now`], and `candidates` without
/// that evidence.
fn store_with_tiers(reached: &[Enr], candidates: &[Enr]) -> BootStore {
    let mut store = BootStore::in_memory();
    for record in reached.iter().chain(candidates) {
        store.add_enr(record.clone());
    }
    for record in reached {
        reach(&mut store, record, fixed_now());
    }
    store
}

/// A driver with a bootstore of one record, `old`, whose evidence is older than it counts, that
/// saves its bootstore every 50 ms and expires the evidence at the first store interval it may.
fn driver_with_old_evidence() -> (Discv5Driver, tempfile::TempDir, Enr) {
    let old = enr_at_localhost(TEST_CHAIN_ID);
    let (mut driver, dir) = driver_with_bootstore(accept_all, std::slice::from_ref(&old));
    let reached = SystemTime::now() - EVIDENCE_TTL - Duration::from_secs(60 * 60);
    reach(&mut driver.bootstore, &old, reached);
    driver.store_interval = Duration::from_millis(50);
    driver.expiry_interval = Duration::ZERO;
    (driver, dir, old)
}
