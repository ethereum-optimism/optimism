//! Tests for bootstrapping from bootnodes, and which of them the driver contacts and keeps.

use super::*;
use enr::CombinedPublicKey;
use rstest::rstest;

/// An enode bootnode is contacted only if the table filter accepts its address. One discovery
/// can't contact is an error, whatever the filter says.
#[rstest]
#[case::loopback(crate::public_only, "/ip4/127.0.0.1/udp/9223", Some(false))]
#[case::lan_v6(crate::public_only, "/ip6/fd00::1/udp/9223", Some(false))]
#[case::public(crate::public_only, "/ip4/1.1.1.1/udp/9223", Some(true))]
#[case::allowing_private(accept_all, "/ip4/127.0.0.1/udp/9223", Some(true))]
#[case::dns(accept_all, "/dns4/example.com/udp/9223", None)]
#[case::without_udp_port(accept_all, "/ip4/127.0.0.1/tcp/9222", None)]
fn test_accepts_enode(
    #[case] table_filter: fn(&Enr) -> bool,
    #[case] addr: &str,
    #[case] expected: Option<bool>,
) {
    let peer_id = libp2p::identity::Keypair::generate_secp256k1().public().to_peer_id();
    let enode = addr.parse::<Multiaddr>().expect("a valid multiaddr").with(Protocol::P2p(peer_id));
    assert_eq!(accepts_enode(table_filter, &enode).ok(), expected);
}

/// A node with a public-only table filter should neither contact nor keep a loopback bootnode
/// given by enode URL, as discv5 doesn't keep one given by ENR. An enode URL holds only an
/// address, so the node would ask that address for the bootnode's record, which starts a
/// session and so puts each node in the other's table without the table filter.
#[tokio::test]
async fn test_public_only_node_drops_loopback_enode_bootnode() {
    let bootnode = offline_driver().init().await.expect("the bootnode starts");
    let bootnode_enr = bootnode.disc.local_enr();
    let enode = loopback_enode(&bootnode_enr);

    let node = loopback_driver(loopback_config().table_filter(crate::public_only).build())
        .init()
        .await
        .expect("the node starts");

    Discv5Driver::bootstrap_peers(
        None,
        BootNodes(vec![BootNode::Enode(enode)]),
        node.chain_id,
        node.table_filter,
        &node.disc,
    )
    .await;

    assert!(
        !node.disc.table_entries_enr().iter().any(|enr| enr.node_id() == bootnode_enr.node_id()),
        "a public-only node kept a loopback enode bootnode"
    );
    // The bootnode adds a node to its table in the handshake, before it answers, so it would
    // hold the node by now if the node had contacted it.
    assert!(
        !bootnode
            .disc
            .table_entries_enr()
            .iter()
            .any(|enr| enr.node_id() == node.disc.local_enr().node_id()),
        "a public-only node contacted a loopback enode bootnode"
    );
}

/// A bootnode whose record isn't valid for our chain should be in the table, as an entry point
/// for lookups, though it isn't saved or dialed. OP Mainnet's built-in enode bootnodes have no
/// `opstack` key, so they are invalid for every chain, and a node with an empty bootstore would
/// have nobody to ask without them. Its built-in ENRs are of other chains. An enode bootnode is
/// in the table from the session that fetched its record, an ENR one only if the bootstrap
/// adds it.
#[rstest]
#[case::enode(|enr: &Enr| BootNode::Enode(loopback_enode(enr)))]
#[case::enr(|enr: &Enr| BootNode::Enr(enr.clone()))]
#[tokio::test]
async fn test_bootnode_for_another_chain_is_in_the_table(
    #[case] bootnode_as: fn(&Enr) -> BootNode,
) {
    let bootnode = offline_driver().init().await.expect("the bootnode starts");
    let bootnode_enr = bootnode.disc.local_enr();

    let mut node = offline_driver();
    node.chain_id = bootnode.chain_id + 1;
    let node = node.init().await.expect("the node starts");

    let store = Discv5Driver::bootstrap_peers(
        None,
        BootNodes(vec![bootnode_as(&bootnode_enr)]),
        node.chain_id,
        node.table_filter,
        &node.disc,
    )
    .await;

    assert!(
        node.disc.table_entries_enr().iter().any(|enr| enr.node_id() == bootnode_enr.node_id()),
        "a bootnode for another chain isn't in the node's table"
    );
    assert!(store.is_empty(), "the node saved a bootnode for another chain");
}

/// An enode URL holds only an address, so the bootstrap should check the record it fetches
/// from that address too: a record the table filter rejects should be neither saved nor kept
/// in the table, where the session that fetched it put it.
///
/// The filter accepts a record only on `127.0.0.1` and without IPv6, as `public_only` rejects
/// a dual-stack record with a private IPv6 address. B's enode URL is on `127.0.0.1`, and its
/// record also holds `::1`.
#[tokio::test]
async fn test_enode_bootnode_with_a_record_the_filter_rejects_is_dropped() {
    fn ipv4_loopback_only(enr: &Enr) -> bool {
        enr.ip4() == Some(Ipv4Addr::LOCALHOST) && enr.ip6().is_none()
    }

    let key = CombinedKey::generate_secp256k1();
    let port = std::net::UdpSocket::bind((Ipv4Addr::LOCALHOST, 0))
        .and_then(|socket| socket.local_addr())
        .expect("a free UDP port")
        .port();
    let record = Enr::builder()
        .ip4(Ipv4Addr::LOCALHOST)
        .udp4(port)
        .tcp4(port)
        .ip6(Ipv6Addr::LOCALHOST)
        .build(&key)
        .expect("the record builds");
    let listen = ListenConfig::from_ip(IpAddr::V4(Ipv4Addr::LOCALHOST), port);
    let mut bootnode = Discv5::new(record.clone(), key, ConfigBuilder::new(listen).build())
        .expect("the bootnode builds");
    bootnode.start().await.expect("the bootnode starts");

    let (node, _dir) = driver_with_bootstore(ipv4_loopback_only, &[]);
    let node = node.init().await.expect("the node starts");
    let store = Discv5Driver::bootstrap_peers(
        None,
        BootNodes(vec![BootNode::Enode(loopback_enode(&record))]),
        node.chain_id,
        node.table_filter,
        &node.disc,
    )
    .await;

    assert!(
        !node.disc.table_entries_enr().iter().any(|enr| enr.node_id() == record.node_id()),
        "the node kept a bootnode whose record the filter rejects"
    );
    assert!(store.is_empty(), "the node saved a bootnode whose record the filter rejects");
}

#[tokio::test]
async fn test_online_discv5_driver_bootstrap_testnet() {
    // Use a test file to make sure bootstore
    // doesn't conflict with a local bootstore.
    let file = tempdir().unwrap();
    let file = file.path().join("bootstore.json");

    let socket = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 0);
    let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
        unreachable!()
    };
    let mut discovery = Discv5Driver::builder(
        LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::UNSPECIFIED), 0, 0),
        OP_SEPOLIA_CHAIN_ID,
        ConfigBuilder::new(socket.into()).build(),
    )
    .build()
    .expect("Failed to build discovery service");
    discovery.bootstore = Some(BootStoreFile::Custom(file));

    discovery = discovery.init().await.expect("Failed to initialize discovery service");

    // There are no ENRs for `OP_SEPOLIA_CHAIN_ID` in the bootstore.
    // If an ENR is added, this check will fail.
    Discv5Driver::bootstrap_peers(
        discovery.bootstore,
        discovery.bootnodes,
        OP_SEPOLIA_CHAIN_ID,
        discovery.table_filter,
        &discovery.disc,
    )
    .await;
    assert!(
        discovery.disc.table_entries_enr().len() >= 5,
        "Discovery table should have at least 5 ENRs"
    );

    // It should have the same number of entries as the testnet table.
    let testnet = BootNodes::testnet();

    // Filter out testnet ENRs that are not valid.
    let testnet: Vec<CombinedPublicKey> = testnet
        .iter()
        .filter_map(|node| {
            if let BootNode::Enr(enr) = node {
                // Check that the ENR is valid for the testnet.
                if EnrValidation::validate(enr, OP_SEPOLIA_CHAIN_ID).is_invalid() {
                    return None;
                }
            }
            let node_contact =
                NodeContact::try_from_multiaddr(node.to_multiaddr().unwrap()).unwrap();

            Some(node_contact.public_key())
        })
        .collect();

    // There should be 8 valid ENRs for the testnet.
    assert_eq!(testnet.len(), 8);

    // Those 8 ENRs should be in the discovery table.
    let disc_enrs = discovery.disc.table_entries_enr();
    for public_key in testnet {
        assert!(
            disc_enrs.iter().any(|enr| enr.public_key() == public_key),
            "Discovery table does not contain testnet ENR: {public_key:?}"
        );
    }
}

#[tokio::test]
async fn test_online_discv5_driver_bootstrap_mainnet() {
    kona_cli::init_test_tracing();

    // Use a test file to make sure bootstore
    // doesn't conflict with a local bootstore.
    let file = tempdir().unwrap();
    let file = file.path().join("bootstore.json");

    // Every bootnode goes into the table, those of other chains too.
    let mainnet: Vec<CombinedPublicKey> = BootNodes::mainnet()
        .iter()
        .map(|node| match node {
            BootNode::Enr(enr) => enr.public_key(),
            BootNode::Enode(enode) => {
                NodeContact::try_from_multiaddr(enode.clone()).unwrap().public_key()
            }
        })
        .collect();
    assert_eq!(mainnet.len(), 24);

    let socket = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 0);

    let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
        unreachable!()
    };

    let mut discovery = Discv5Driver::builder(
        LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::UNSPECIFIED), 0, 0),
        OP_MAINNET_CHAIN_ID,
        ConfigBuilder::new(socket.into()).build(),
    )
    .build()
    .expect("Failed to build discovery service");
    discovery.bootstore = Some(BootStoreFile::Custom(file));

    discovery = discovery.init().await.expect("Failed to initialize discovery service");

    // There are no ENRs for op mainnet in the bootstore.
    // If an ENR is added, this check will fail.
    Discv5Driver::bootstrap_peers(
        discovery.bootstore,
        discovery.bootnodes,
        OP_MAINNET_CHAIN_ID,
        discovery.table_filter,
        &discovery.disc,
    )
    .await;
    assert!(
        discovery.disc.table_entries_enr().len() >= 10,
        "Discovery table should have at least 10 ENRs"
    );

    // Those ENRs should be in the mainnet bootnodes.
    let disc_enrs = discovery.disc.table_entries_enr();
    for enr in disc_enrs {
        assert!(
            mainnet.iter().any(|pub_key| pub_key == &enr.public_key()),
            "Discovery table does not contain mainnet ENR: {enr:?}"
        );
    }
}
