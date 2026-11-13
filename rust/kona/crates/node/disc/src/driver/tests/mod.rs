//! Tests for the [`Discv5Driver`], one file per topic, with the helpers they share.

use super::*;
use crate::test_utils::{enr_at, localhost_enr};
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

/// The chain ID of the offline tests: not in the registry, so a node on it has no bootnodes of
/// its own.
const TEST_CHAIN_ID: u64 = 0xdead_beef;

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
    let (mut store, _) =
        BootStore::open(dir.path().join("bootstore.json")).expect("the bootstore opens");
    for record in records {
        store.add_enr(record.clone());
    }

    let mut driver = loopback_driver(loopback_config().table_filter(table_filter).build());
    driver.bootstore = store;
    (driver, dir)
}

/// Reads the bootstore at `path` until `done` holds for the records it holds, and returns those,
/// or `None` after 5s. A missing file holds no records.
async fn wait_for_bootstore(
    path: &std::path::Path,
    done: impl Fn(&[Enr]) -> bool,
) -> Option<Vec<Enr>> {
    let poll = async {
        loop {
            let (store, _) = BootStore::open(path.to_path_buf()).expect("the bootstore opens");
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

/// Waits until the driver has finished startup and runs its main loop.
async fn wait_until_started(handler: &Discv5Handler) {
    // Requests are only answered by the main loop.
    tokio::time::timeout(Duration::from_secs(5), handler.peer_count())
        .await
        .expect("driver should finish startup")
        .expect("driver should not stop during startup");
}
