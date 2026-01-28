//! Discovery Module.

use backon::{ExponentialBuilder, RetryableWithContext};
use derive_more::Debug;
use discv5::{Config, Discv5, Enr, enr::NodeId};
use kona_peers::{BootNode, BootNodes, BootStore, BootStoreFile, EnrValidation, enr_to_multiaddr};
use tokio::{
    sync::mpsc::channel,
    time::{Duration, sleep},
};

use crate::{Discv5Builder, Discv5Handler, HandlerRequest, LocalNode};

/// The [`Discv5Driver`] drives the discovery service.
///
/// Calling [`Discv5Driver::start`] spawns a new [`Discv5`]
/// discovery service in a new tokio task and returns a
/// [`Discv5Handler`].
///
/// Channels are used to communicate between the [`Discv5Handler`]
/// and the spawned task containing the [`Discv5`] service.
///
/// Since some requested operations are asynchronous, this pattern of message
/// passing is used as opposed to wrapping the [`Discv5`] in an `Arc<Mutex<>>`.
/// If an `Arc<Mutex<>>` were used, a lock held across the operation's future
/// would be needed since some asynchronous operations require a mutable
/// reference to the [`Discv5`] service.
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
    /// Whether to forward ENRs to the enr receiver on startup.
    pub forward: bool,
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
            forward: true,
            remove_interval: None,
            store_interval: Duration::from_secs(60),
            bootstore,
        })
    }

    /// Starts the inner [`Discv5`] service.
    async fn init(self) -> Result<Self, discv5::Error> {
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

    /// Bootstraps the [`Discv5`] table with bootnodes.
    async fn bootstrap_peers(
        bootstore: Option<BootStoreFile>,
        bootnodes: BootNodes,
        chain_id: u64,
        disc: &Discv5,
    ) -> BootStore {
        // Note: if the bootstore file cannot be created, we use a default bootstore.
        let mut store = bootstore
            .map_or_else(BootStore::default, |bootstore| bootstore.try_into().unwrap_or_default());

        let initial_store_length = store.len();

        for bn in bootnodes.0.into_iter().chain(BootNodes::from_chain_id(chain_id).0.into_iter()) {
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

    /// Spawns a new [`Discv5`] discovery service in a new tokio task.
    ///
    /// Returns a [`Discv5Handler`] to communicate with the spawned task.
    pub fn start(mut self) -> (Discv5Handler, tokio::sync::mpsc::Receiver<Enr>) {
        let chain_id = self.chain_id;
        let (req_sender, mut req_recv) = channel::<HandlerRequest>(1024);
        let (enr_sender, enr_recv) = channel::<Enr>(1024);

        tokio::spawn(async move {
            let remove = self.remove_interval.is_some();
            let remove_dur = self.remove_interval.unwrap_or(std::time::Duration::from_secs(600));
            let mut removal_interval = tokio::time::interval(remove_dur);
            let mut interval = tokio::time::interval(self.interval);
            let mut store_interval = tokio::time::interval(self.store_interval);

            // Step 1: Start the discovery service.
            let Ok(s) = self.init().await else {
                error!(target: "discovery", "Failed to start discovery service");
                return;
            };
            self = s;
            trace!(target: "discovery", "Discv5 Initialized");

            // Step 2: Bootstrap the discovery table with bootnodes.
            let mut store =
                Self::bootstrap_peers(self.bootstore, self.bootnodes, chain_id, &self.disc).await;

            let enrs = self.disc.table_entries_enr();
            info!(target: "discovery", "Discv5 Started with {} ENRs", enrs.len());

            // Step 3: Forward ENRs in the bootstore to the enr receiver.
            if self.forward {
                for enr in store.valid_peers_with_chain_id(self.chain_id) {
                    if let Err(e) = enr_sender.send(enr.clone()).await {
                        debug!(target: "discovery", "Failed to forward enr: {:?}", e);
                    }
                }
            }

            // Continuously attempt to start the event stream with a retry limit and shutdown
            // signal.
            let mut retries = 0;
            let max_retries = 10; // Maximum number of retries before giving up.
            let mut event_stream = loop {
                if retries >= max_retries {
                    error!(target: "discovery", "Exceeded maximum retries for event stream startup. Aborting...");
                    return; // Exit the task if the retry limit is reached.
                }
                match self.disc.event_stream().await {
                    Ok(event_stream) => {
                        break event_stream;
                    }
                    Err(e) => {
                        warn!(target: "discovery", "Failed to start event stream: {:?}", e);
                        retries += 1;
                        sleep(Duration::from_secs(2)).await;
                        info!(target: "discovery", "Retrying event stream startup... (Attempt {}/{})", retries, max_retries);
                    }
                }
            };

            // Step 4: Run the core driver loop.
            loop {
                tokio::select! {
                    msg = req_recv.recv() => {
                        match msg {
                            Some(msg) => match msg {
                                HandlerRequest::Metrics(tx) => {
                                    let metrics = self.disc.metrics();
                                    if let Err(e) = tx.send(metrics) {
                                        warn!(target: "discovery", "Failed to send metrics: {:?}", e);
                                    }
                                }
                                HandlerRequest::PeerCount(tx) => {
                                    let peers = self.disc.connected_peers();
                                    if let Err(e) = tx.send(peers) {
                                        warn!(target: "discovery", "Failed to send peer count: {:?}", e);
                                    }
                                }
                                HandlerRequest::LocalEnr(tx) => {
                                    let enr = self.disc.local_enr().clone();
                                    if let Err(e) = tx.send(enr.clone()) {
                                        warn!(target: "discovery", "Failed to send local enr: {:?}", e);
                                    }
                                }
                                HandlerRequest::AddEnr(enr) => {
                                    let _ = self.disc.add_enr(enr);
                                }
                                HandlerRequest::RequestEnr{out, addr} => {
                                    let enr = self.disc.request_enr(addr).await;
                                    if let Err(e) = out.send(enr) {
                                        warn!(target: "discovery", "Failed to send request enr: {:?}", e);
                                    }
                                }
                                HandlerRequest::TableEnrs(tx) => {
                                    let enrs = self.disc.table_entries_enr();
                                    if let Err(e) = tx.send(enrs) {
                                        warn!(target: "discovery", "Failed to send table enrs: {:?}", e);
                                    }
                                },
                                HandlerRequest::TableInfos(tx) => {
                                    let infos = self.disc.table_entries();
                                    if let Err(e) = tx.send(infos) {
                                        warn!(target: "discovery", "Failed to send table infos: {:?}", e);
                                    }
                                },
                                HandlerRequest::BanAddrs{addrs_to_ban, ban_duration} => {
                                    let enrs = self.disc.table_entries_enr();

                                    for enr in enrs {
                                        let Some(multi_addr) = enr_to_multiaddr(&enr) else {
                                            continue;
                                        };

                                        if addrs_to_ban.contains(&multi_addr) {
                                            self.disc.ban_node(&enr.node_id(), Some(ban_duration));
                                        }
                                    }
                                },
                            }
                            None => {
                                trace!(target: "discovery", "Receiver `None` peer enr");
                            }
                        }
                    }
                    event = event_stream.recv() => {
                        let Some(event) = event else {
                            trace!(target: "discovery", "Received `None` event");
                            continue;
                        };
                        match event {
                            discv5::Event::Discovered(enr) => {
                                if EnrValidation::validate(&enr, chain_id).is_valid() {
                                    debug!(target: "discovery", "Valid ENR discovered, forwarding to swarm: {:?}", enr);
                                    kona_macros::inc!(gauge, crate::Metrics::DISCOVERY_EVENT, "type" => "discovered");
                                    store.add_enr(enr.clone());
                                    let sender = enr_sender.clone();
                                    tokio::spawn(async move {
                                        if let Err(e) = sender.send(enr).await {
                                            debug!(target: "discovery", "Failed to send enr: {:?}", e);
                                        }
                                    });
                                }
                            }
                            discv5::Event::SessionEstablished(enr, addr) => {
                                if EnrValidation::validate(&enr, chain_id).is_valid() {
                                    debug!(target: "discovery", "Session established with valid ENR, forwarding to swarm. Address: {:?}, ENR: {:?}", addr, enr);
                                    kona_macros::inc!(gauge, crate::Metrics::DISCOVERY_EVENT, "type" => "session_established");
                                    store.add_enr(enr.clone());
                                    let sender = enr_sender.clone();
                                    tokio::spawn(async move {
                                        if let Err(e) = sender.send(enr).await {
                                            debug!(target: "discovery", "Failed to send enr: {:?}", e);
                                        }
                                    });
                                }
                            }
                            discv5::Event::UnverifiableEnr { enr, .. } => {
                                if EnrValidation::validate(&enr, chain_id).is_valid() {
                                    debug!(target: "discovery", "Valid ENR discovered, forwarding to swarm: {:?}", enr);
                                    kona_macros::inc!(gauge, crate::Metrics::DISCOVERY_EVENT, "type" => "unverifiable_enr");
                                    store.add_enr(enr.clone());
                                    let sender = enr_sender.clone();
                                    tokio::spawn(async move {
                                        if let Err(e) = sender.send(enr).await {
                                            debug!(target: "discovery", "Failed to send enr: {:?}", e);
                                        }
                                    });
                                }

                            }
                            _ => {}
                        }
                    }
                    _ = interval.tick() => {
                        let id = NodeId::random();
                        trace!(target: "discovery", "Finding random node: {}", id);
                        kona_macros::inc!(gauge, crate::Metrics::FIND_NODE_REQUEST, "find_node" => "find_node");
                        let fut = self.disc.find_node(id);
                        let enr_sender = enr_sender.clone();
                        tokio::spawn(async move {
                            match fut.await {
                                Ok(nodes) => {
                                    let enrs = nodes.into_iter().filter(|node| EnrValidation::validate(node, chain_id).is_valid());
                                    for enr in enrs {
                                        _ = enr_sender.send(enr).await;
                                    }
                                }
                                Err(err) => {
                                    info!(target: "discovery", "Failed to find node: {:?}", err);
                                }
                            }
                        });
                    }
                    _ = store_interval.tick() => {
                        let start = std::time::Instant::now();
                        let enrs = self.disc.table_entries_enr();
                        store.merge(enrs);

                        if let Err(e) = store.sync() {
                            warn!(target: "discovery", "Failed to sync bootstore: {:?}", e);
                        }

                        let elapsed = start.elapsed();
                        debug!(target: "discovery", "Bootstore ENRs stored in {:?}", elapsed);
                        kona_macros::record!(histogram, crate::Metrics::ENR_STORE_TIME, "store_time", "store_time", elapsed.as_secs_f64());
                        kona_macros::set!(gauge, crate::Metrics::DISCOVERY_PEER_COUNT, self.disc.connected_peers() as f64);
                    }
                    _ = removal_interval.tick() => {
                        if remove {
                            let enrs = self.disc.table_entries_enr();
                            if enrs.len() > 20 {
                                let mut rng = rand::rng();
                                let index = rand::Rng::random_range(&mut rng, 0..enrs.len());
                                let enr = enrs[index].clone();
                                debug!(target: "removal", "Removing random ENR: {:?}", enr);
                                self.disc.remove_node(&enr.node_id());
                            }
                        }
                    }
                }
            }
        });

        (Discv5Handler::new(chain_id, req_sender), enr_recv)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::LocalNode;
    use discv5::{
        ConfigBuilder,
        enr::{CombinedKey, CombinedPublicKey},
        handler::NodeContact,
    };
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
        let (handle, _) = discovery.start();
        assert_eq!(handle.chain_id, OP_SEPOLIA_CHAIN_ID);
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
                if let BootNode::Enr(enr) = node {
                    if EnrValidation::validate(enr, OP_MAINNET_CHAIN_ID).is_invalid() {
                        return None;
                    }
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
