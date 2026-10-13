//! Discovery Module.

use backon::{ExponentialBuilder, RetryableWithContext};
use derive_more::Debug;
use discv5::{Config, Discv5};
use kona_peers::{BootNode, BootNodes, BootStore, BootStoreFile, EnrValidation};
use tokio::time::Duration;

use crate::{Discv5Builder, LocalNode};

/// A [`Discv5`] discovery service and its settings, ready to be started.
///
/// [`Self::init`] starts the service and [`Self::bootstrap_peers`] fills its table. A caller
/// then runs the service's event loop.
#[derive(Debug)]
pub struct Discv5Driver {
    /// The [`Discv5`] discovery service.
    #[debug(skip)]
    pub disc: Discv5,
    /// The optional [`BootStoreFile`] to use for the bootstore.
    pub bootstore: Option<BootStoreFile>,
    /// Bootnodes used to bootstrap the discovery service.
    pub bootnodes: BootNodes,
    /// The chain ID of the network.
    pub chain_id: u64,
    /// The interval to discovery random nodes.
    pub interval: Duration,
    /// The interval at which to store the ENRs in the bootstore.
    /// This is set to 60 seconds by default.
    pub store_interval: Duration,
    /// The frequency at which to remove random nodes from the discovery table.
    /// This is not enabled (`None`) by default.
    pub remove_interval: Option<Duration>,
}

impl Discv5Driver {
    /// Returns a new [`Discv5Builder`] instance.
    pub fn builder(
        local_node: LocalNode,
        chain_id: u64,
        discovery_config: Config,
    ) -> Discv5Builder {
        Discv5Builder::new(local_node, chain_id, discovery_config)
    }

    /// Instantiates a new [`Discv5Driver`].
    pub const fn new(
        disc: Discv5,
        interval: Duration,
        chain_id: u64,
        bootstore: Option<BootStoreFile>,
        bootnodes: BootNodes,
    ) -> Result<Self, std::io::Error> {
        Ok(Self {
            disc,
            chain_id,
            bootnodes,
            interval,
            remove_interval: None,
            store_interval: Duration::from_secs(60),
            bootstore,
        })
    }

    /// Starts the inner [`Discv5`] service, retrying with backoff.
    pub async fn init(self) -> Result<Self, discv5::Error> {
        let (s, res) = {
            |mut v: Self| async {
                let res = v.disc.start().await;
                (v, res)
            }
        }
            .retry(ExponentialBuilder::default())
            .context(self)
            .notify(|err: &discv5::Error, dur: Duration| {
                warn!(target: "discovery", ?err, "Failed to start discovery service [Duration: {:?}]", dur);
            })
            .await;
        res.map(|_| s)
    }

    /// Bootstraps the [`Discv5`] table with bootnodes, and returns the bootstore with them added.
    pub async fn bootstrap_peers(
        bootstore: Option<BootStoreFile>,
        bootnodes: BootNodes,
        chain_id: u64,
        disc: &Discv5,
    ) -> BootStore {
        // Note: if the bootstore file cannot be created, we use a default bootstore.
        let mut store = bootstore
            .map_or_else(BootStore::default, |bootstore| bootstore.try_into().unwrap_or_default());

        let initial_store_length = store.len();

        for bn in bootnodes.0.into_iter().chain(BootNodes::from_chain_id(chain_id).0) {
            let res = match bn {
                BootNode::Enr(enr) => Ok(enr.clone()),
                BootNode::Enode(enode) => disc.request_enr(enode.clone()).await,
            };

            let Ok(enr) = res else {
                debug!(target: "discovery::bootstrap", ?res, "Failed to add boot node ENR to discovery table");
                continue;
            };

            let validation = EnrValidation::validate(&enr, chain_id);
            if validation.is_invalid() {
                trace!(target: "discovery::bootstrap", "Ignoring Invalid Bootnode ENR: {:?}. {:?}", enr, validation);
                continue;
            }

            if let Err(e) = disc.add_enr(enr.clone()) {
                debug!(target: "discovery::bootstrap", "Failed to add enr: {:?}", e);
                continue;
            }

            store.add_enr(enr);
        }

        let new_store_len = store.len();

        debug!(target: "discovery::bootstrap",
            added=%(new_store_len - initial_store_length),
            total=%new_store_len,
            "Added new ENRs to discv5 bootstore"
        );

        store
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::LocalNode;
    use discv5::{ConfigBuilder, NodeContact};
    use enr::{CombinedKey, CombinedPublicKey};
    use kona_genesis::{OP_MAINNET_CHAIN_ID, OP_SEPOLIA_CHAIN_ID};
    use tempfile::tempdir;

    use std::net::{IpAddr, Ipv4Addr, SocketAddr};

    #[tokio::test]
    async fn test_online_discv5_driver() {
        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };

        let socket = SocketAddr::new(IpAddr::V4(Ipv4Addr::UNSPECIFIED), 0);
        let discovery = Discv5Driver::builder(
            LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::UNSPECIFIED), 0, 0),
            OP_SEPOLIA_CHAIN_ID,
            ConfigBuilder::new(socket.into()).build(),
        )
        .build()
        .expect("Failed to build discovery service");
        let discovery = discovery.init().await.expect("Failed to initialize discovery service");
        assert_eq!(discovery.chain_id, OP_SEPOLIA_CHAIN_ID);
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

        // Filter out ENRs that are not valid.
        let mainnet = BootNodes::mainnet();
        let mainnet: Vec<CombinedPublicKey> = mainnet
            .iter()
            .filter_map(|node| {
                if let BootNode::Enr(enr) = node &&
                    EnrValidation::validate(enr, OP_MAINNET_CHAIN_ID).is_invalid()
                {
                    return None;
                }
                let node_contact =
                    NodeContact::try_from_multiaddr(node.to_multiaddr().unwrap()).unwrap();

                Some(node_contact.public_key())
            })
            .collect();

        // There should be 16 valid ENRs for the mainnet.
        assert_eq!(mainnet.len(), 16);

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
}
