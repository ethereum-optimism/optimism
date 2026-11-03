//! Discovery Module.

use backon::{ExponentialBuilder, RetryableWithContext};
use derive_more::Debug;
use discv5::{Config, Discv5, Enr};
use enr::NodeId;
use kona_peers::{
    BootNode, BootNodes, BootStore, BootStoreFile, EnrValidation, peer_id_to_node_id,
};
use libp2p::{Multiaddr, multiaddr::Protocol};
use std::pin::Pin;
use tokio::{
    sync::mpsc::channel,
    task::JoinHandle,
    time::{Duration, sleep},
};
use tracing::Instrument;

use crate::{
    AddrScope, AdvertiseTcpPortError, Discv5Builder, Discv5Handler, HandlerRequest, LocalNode,
    builder::bootnode_ips,
};

/// Spawns `future` on Tokio in the current span, so that its logs carry the span discovery was
/// started in, such as the one each test node runs in.
fn spawn_in_current_span<F>(future: F) -> JoinHandle<F::Output>
where
    F: Future + Send + 'static,
    F::Output: Send + 'static,
{
    tokio::spawn(future.in_current_span())
}

/// Spawns discv5's own tasks like [`spawn_in_current_span`]. discv5's default executor spawns them
/// outside any span.
#[derive(Clone, Debug)]
pub(crate) struct CurrentSpanExecutor;

impl discv5::Executor for CurrentSpanExecutor {
    fn spawn(&self, future: Pin<Box<dyn Future<Output = ()> + Send>>) {
        spawn_in_current_span(future);
    }
}

/// The number of times to retry starting the [`Discv5`] service, with exponential backoff.
const START_MAX_RETRIES: usize = 3;

/// The number of attempts to start the [`Discv5`] event stream before giving up.
const EVENT_STREAM_MAX_ATTEMPTS: u32 = 10;

/// The delay between attempts to start the [`Discv5`] event stream.
const EVENT_STREAM_RETRY_DELAY: Duration = Duration::from_secs(2);

/// A record our own gossip should dial, see [`Discv5Driver::record_to_dial`].
#[derive(Debug, PartialEq, Eq)]
struct RecordToDial {
    /// The name of the discovery event that brought it, for metrics and logs.
    event: &'static str,
    /// The record.
    enr: Enr,
}

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
    /// Our own address scope, which discovered records and bootnodes must be reachable from. See
    /// [`AddrScope::own`]. Not public, as discv5's table filter was chosen for it and can't
    /// change.
    pub(crate) own_scope: AddrScope,
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
        own_scope: AddrScope,
        bootstore: Option<BootStoreFile>,
        bootnodes: BootNodes,
    ) -> Result<Self, std::io::Error> {
        Ok(Self {
            disc,
            chain_id,
            own_scope,
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
            .retry(ExponentialBuilder::default().with_max_times(START_MAX_RETRIES))
            .context(self)
            .notify(|err: &discv5::Error, dur: Duration| {
                warn!(target: "discovery", ?err, "Failed to start discovery service [Duration: {:?}]", dur);
            })
            .await;
        res.map(|_| s)
    }

    /// Returns the record a discovery event brings, if our own gossip should dial it. The driver
    /// then hands it to gossip and saves it in the bootstore; it doesn't send it to other nodes.
    /// Only a record valid for `chain_id` is dialed, from these events:
    ///
    /// - `Discovered`: a record in another node's answer to one of our lookups. discv5 sends it for
    ///   every record in the answer, before it applies its table filter or checks that the record
    ///   has an address to contact, so the record may not be live or reachable. It is dialed
    ///   because that is how gossip learns about nodes we have no session with, but only if a node
    ///   of scope `own_scope` can reach it, as the table filter checks: discv5 doesn't say which
    ///   node relayed it.
    /// - `SessionEstablished`: a session with the node started, in either direction, so the node is
    ///   live. discv5 only checked the record's UDP address of the family the packets came in on,
    ///   if it has one, against the address they came from. Gossip dials the record's TCP address,
    ///   so the record is only dialed if all its addresses are plausible from the address the
    ///   packets came from, as an unverifiable record is. An honest node's record holds that
    ///   address, so it is dialed even if our own scope couldn't reach it otherwise, such as a node
    ///   on our own host's loopback.
    ///
    ///   discv5 also adds such a node to its table without the table filter, and relays its record
    ///   to the nodes that look it up, who may not be able to reach it. discv5 has no hook to check
    ///   a record against the node asking for it, as geth does, so it is left to that node to
    ///   reject it.
    /// - `UnverifiableEnr`: a node started a session with us, but its record's UDP address is not
    ///   the one its packets came from. discv5 keeps the session but leaves the node out of its
    ///   table. It is dialed because the record's address may still be reachable: a NAT may have
    ///   rewritten the address the packets came from, while the record holds the one the NAT maps
    ///   to the node. The record may also be wrong, or hold an address the node can't be reached
    ///   at, so it is only dialed if its addresses are plausible from the address the packets came
    ///   from.
    fn record_to_dial(
        event: discv5::Event,
        chain_id: u64,
        own_scope: AddrScope,
    ) -> Option<RecordToDial> {
        let (kind, enr, check) = match event {
            discv5::Event::Discovered(enr) => {
                let check = AddrScope::check_reachable_enr(own_scope, &enr);
                ("discovered", enr, check)
            }
            discv5::Event::SessionEstablished(enr, socket) => {
                let check = AddrScope::check_relayed_enr(AddrScope::of(socket.ip()), &enr);
                ("session_established", enr, check)
            }
            discv5::Event::UnverifiableEnr { enr, socket, .. } => {
                let check = AddrScope::check_relayed_enr(AddrScope::of(socket.ip()), &enr);
                ("unverifiable_enr", enr, check)
            }
            _ => return None,
        };
        if EnrValidation::validate(&enr, chain_id).is_invalid() {
            return None;
        }
        if let Err(reason) = check {
            debug!(target: "discovery", event = kind, %reason, ?enr, "Ignoring a record with an address we can't reach");
            metrics::gauge!(crate::Metrics::DISCOVERY_EVENT, "type" => "unreachable_address")
                .increment(1);
            return None;
        }
        Some(RecordToDial { event: kind, enr })
    }

    /// Returns whether our own gossip should dial `enr`, a record one of our lookups returned. A
    /// lookup returns records from other nodes' answers, which discv5's table filter checked, but
    /// also nodes from discv5's own table, which got there through a session, without the filter.
    /// So check them all against our own scope, and the chain.
    fn is_lookup_record_to_dial(enr: &Enr, chain_id: u64, own_scope: AddrScope) -> bool {
        EnrValidation::validate(enr, chain_id).is_valid() &&
            AddrScope::check_reachable_enr(own_scope, enr).is_ok()
    }

    /// Bootstraps the [`Discv5`] table with bootnodes.
    async fn bootstrap_peers(
        bootstore: Option<BootStoreFile>,
        bootnodes: BootNodes,
        chain_id: u64,
        own_scope: AddrScope,
        disc: &Discv5,
    ) -> BootStore {
        // Note: if the bootstore file cannot be created, we use a default bootstore.
        let mut store = bootstore
            .map_or_else(BootStore::default, |bootstore| bootstore.try_into().unwrap_or_default());

        let initial_store_length = store.len();

        for bn in bootnodes.0.into_iter().chain(BootNodes::from_chain_id(chain_id).0) {
            let res = match &bn {
                BootNode::Enr(enr) => Ok(enr.clone()),
                BootNode::Enode(enode) => {
                    // Asking it for its record starts a session, which puts it in discv5's table
                    // without the table filter, so check its address first.
                    let unreachable = bootnode_ips(&bn)
                        .into_iter()
                        .find_map(|ip| AddrScope::check_reachable_addr(own_scope, ip).err());
                    if let Some(reason) = unreachable {
                        warn!(target: "discovery::bootstrap", %reason, ?own_scope, %enode, "Ignoring a boot node that is not reachable from our own address");
                        continue;
                    }
                    disc.request_enr(enode.clone()).await
                }
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

            // discv5's table filter would reject it too, but without saying why. A bootnode given
            // by enode URL is in the table already, from the session that fetched its record.
            if let Err(reason) = AddrScope::check_reachable_enr(own_scope, &enr) {
                warn!(target: "discovery::bootstrap", %reason, ?own_scope, ?enr, "Ignoring a boot node that is not reachable from our own address");
                disc.remove_node(&enr.node_id());
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
    /// Returns a [`Discv5Handler`] to communicate with the spawned task, and a receiver of the
    /// valid ENRs it discovers.
    ///
    /// The task is detached and stops when:
    /// - startup fails: [`Discv5`] does not start after `START_MAX_RETRIES` retries with
    ///   exponential backoff, or its event stream does not start after `EVENT_STREAM_MAX_ATTEMPTS`
    ///   attempts, `EVENT_STREAM_RETRY_DELAY` apart;
    /// - the [`Discv5`] event stream ends;
    /// - every [`Discv5Handler`] clone and the ENR receiver are dropped.
    ///
    /// Dropping only the handler, or only the ENR receiver, keeps it running for the other.
    ///
    /// Requests sent before the handler is dropped are answered, those sent during startup once it
    /// completes. If the task stops for another reason, pending requests are dropped, so their
    /// response receivers return an error. When the task stops, the ENR receiver returns `None`
    /// once drained.
    pub fn start(self) -> (Discv5Handler, tokio::sync::mpsc::Receiver<Enr>) {
        let enr = self.disc.local_enr();
        if enr.tcp4() == Some(0) || enr.tcp6() == Some(0) {
            warn!(
                target: "discovery",
                "The ENR advertises TCP port 0, which no peer can dial: set its TCP port with \
                 advertise_gossip_tcp_port before starting discovery"
            );
        }
        let (handler, enr_recv, _task) = self.spawn();
        (handler, enr_recv)
    }

    /// Sets the IPv4 `tcp` and IPv6 `tcp6` ports of the local ENR to the TCP port of
    /// `gossip_addr`, but only those that are unset, which [`LocalNode::tcp_port`] marks with 0.
    ///
    /// A port that is missing from the ENR is not added, and a port that is already set is kept,
    /// so calling this again with another address has no effect. The IP of `gossip_addr` is never
    /// used.
    ///
    /// Call it once gossip has bound its listen address, and before [`Self::start`], so that no
    /// peer sees a TCP port of 0.
    ///
    /// Fails if a port is unset and `gossip_addr` has no TCP port to set it to.
    pub fn advertise_gossip_tcp_port(
        &self,
        gossip_addr: &Multiaddr,
    ) -> Result<(), AdvertiseTcpPortError> {
        let enr = self.disc.local_enr();
        for (key, port) in [("tcp", enr.tcp4()), ("tcp6", enr.tcp6())] {
            if port != Some(0) {
                continue;
            }
            let gossip_port = gossip_addr
                .iter()
                .find_map(|p| match p {
                    Protocol::Tcp(port) => Some(port),
                    _ => None,
                })
                .ok_or_else(|| AdvertiseTcpPortError::NoGossipTcpPort(gossip_addr.clone()))?;
            self.disc
                .enr_insert(key, &gossip_port)
                .map_err(AdvertiseTcpPortError::EnrUpdateFailed)?;
        }
        Ok(())
    }

    /// Spawns a new [`Discv5`] discovery service in a new tokio task.
    ///
    /// Same as [`Discv5Driver::start`], but also returns the [`JoinHandle`] of the task.
    fn spawn(mut self) -> (Discv5Handler, tokio::sync::mpsc::Receiver<Enr>, JoinHandle<()>) {
        let chain_id = self.chain_id;
        let own_scope = self.own_scope;
        let (req_sender, mut req_recv) = channel::<HandlerRequest>(1024);
        let (enr_sender, enr_recv) = channel::<Enr>(1024);

        let task = spawn_in_current_span(async move {
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
            let mut store = Self::bootstrap_peers(
                self.bootstore,
                self.bootnodes,
                chain_id,
                own_scope,
                &self.disc,
            )
            .await;

            let enrs = self.disc.table_entries_enr();
            info!(target: "discovery", "Discv5 Started with {} ENRs", enrs.len());

            // Step 3: Forward ENRs in the bootstore to the enr receiver, for gossip to dial. Only
            // those our own scope can reach: the bootstore may hold records saved by an earlier
            // version, which didn't check them, or while we ran in another scope.
            if self.forward {
                for enr in store.valid_peers_with_chain_id(self.chain_id) {
                    if let Err(reason) = AddrScope::check_reachable_enr(own_scope, enr) {
                        debug!(target: "discovery", %reason, ?enr, "Ignoring a saved record with an address we can't reach");
                        continue;
                    }
                    // Fails only once the receiver is dropped, after which every send fails.
                    if enr_sender.send(enr.clone()).await.is_err() {
                        debug!(target: "discovery", "ENR receiver dropped, not forwarding the rest of the bootstore");
                        break;
                    }
                }
            }

            // Attempt to start the event stream, giving up after the last attempt fails.
            let mut attempt = 1;
            let mut event_stream = loop {
                match self.disc.event_stream().await {
                    Ok(event_stream) => {
                        break event_stream;
                    }
                    Err(e) if attempt < EVENT_STREAM_MAX_ATTEMPTS => {
                        warn!(target: "discovery", "Failed to start event stream: {:?}", e);
                        sleep(EVENT_STREAM_RETRY_DELAY).await;
                        attempt += 1;
                        info!(target: "discovery", "Retrying event stream startup... (Attempt {}/{})", attempt, EVENT_STREAM_MAX_ATTEMPTS);
                    }
                    Err(e) => {
                        error!(target: "discovery", "Failed to start event stream after {} attempts, aborting: {:?}", attempt, e);
                        return;
                    }
                }
            };

            // Step 4: Run the core driver loop.
            let mut handlers_dropped = false;
            loop {
                tokio::select! {
                    msg = req_recv.recv(), if !handlers_dropped => {
                        // `recv` returns `None` once every handler is dropped and the requests
                        // sent before that are handled. Discovery keeps running for consumers
                        // of the ENR receiver.
                        let Some(msg) = msg else {
                            handlers_dropped = true;
                            continue;
                        };
                        match msg {
                            HandlerRequest::Metrics(tx) => {
                                let metrics = Discv5::metrics();
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
                            HandlerRequest::BanPeers{peer_ids, ban_duration} => {
                                // By node ID, which the peer ID's key gives: the addresses a peer
                                // listens on needn't match the ones its ENR advertises.
                                for peer_id in peer_ids {
                                    let Some(node_id) = peer_id_to_node_id(&peer_id) else {
                                        warn!(target: "discovery", %peer_id, "Cannot ban a peer whose ID holds no secp256k1 key");
                                        continue;
                                    };
                                    self.disc.ban_node(&node_id, Some(ban_duration));
                                }
                            },
                        }
                    }
                    // Nobody is left to send requests or to receive ENRs.
                    _ = enr_sender.closed(), if handlers_dropped => {
                        debug!(target: "discovery", "Handler and ENR receiver dropped, stopping discovery");
                        return;
                    }
                    event = event_stream.recv() => {
                        let Some(event) = event else {
                            error!(target: "discovery", "Discv5 event stream ended, stopping discovery");
                            return;
                        };
                        if let Some(RecordToDial { event: kind, enr }) =
                            Self::record_to_dial(event, chain_id, own_scope)
                        {
                            debug!(target: "discovery", event = kind, ?enr, "Valid ENR, passing it to gossip to dial");
                            metrics::gauge!(crate::Metrics::DISCOVERY_EVENT, "type" => kind).increment(1);
                            store.add_enr(enr.clone());
                            let sender = enr_sender.clone();
                            spawn_in_current_span(async move {
                                if let Err(e) = sender.send(enr).await {
                                    debug!(target: "discovery", "Failed to send enr: {:?}", e);
                                }
                            });
                        }
                    }
                    _ = interval.tick() => {
                        let id = NodeId::random();
                        trace!(target: "discovery", "Finding random node: {}", id);
                        metrics::gauge!(crate::Metrics::FIND_NODE_REQUEST, "find_node" => "find_node").increment(1);
                        let fut = self.disc.find_node(id);
                        let enr_sender = enr_sender.clone();
                        spawn_in_current_span(async move {
                            match fut.await {
                                Ok(nodes) => {
                                    let enrs = nodes.into_iter().filter(|node| Self::is_lookup_record_to_dial(node, chain_id, own_scope));
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
                        metrics::histogram!(crate::Metrics::ENR_STORE_TIME, "store_time" => "store_time").record(elapsed.as_secs_f64());
                        metrics::gauge!(crate::Metrics::DISCOVERY_PEER_COUNT).set(self.disc.connected_peers() as f64);
                    }
                    // Only enabled when a removal interval is configured.
                    _ = removal_interval.tick(), if remove => {
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
        });

        (Discv5Handler::new(chain_id, req_sender), enr_recv, task)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{
        LocalNode,
        test_utils::{enr_at, enr_with, localhost_enr},
    };
    use discv5::{ConfigBuilder, ListenConfig, NodeContact};
    use enr::{CombinedKey, CombinedPublicKey};
    use kona_genesis::{OP_MAINNET_CHAIN_ID, OP_SEPOLIA_CHAIN_ID};
    use rstest::rstest;
    use tempfile::tempdir;

    use std::net::{IpAddr, Ipv4Addr, Ipv6Addr, SocketAddr};

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

    /// The socket a session in these tests is with.
    const SOCKET: SocketAddr = SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), 9223);

    /// Gossip dials the record a discovery, a session or an unverifiable record brings, if
    /// it is for the driver's chain, and nothing for other events. The driver is on loopback, so it
    /// can reach the records' loopback address.
    #[rstest]
    #[case::discovered(discv5::Event::Discovered, Some("discovered"))]
    #[case::session_established(
        |enr| discv5::Event::SessionEstablished(enr, SOCKET),
        Some("session_established")
    )]
    #[case::unverifiable_enr(
        |enr: Enr| discv5::Event::UnverifiableEnr { node_id: enr.node_id(), enr, socket: SOCKET },
        Some("unverifiable_enr")
    )]
    #[case::node_inserted(
        |enr: Enr| discv5::Event::NodeInserted { node_id: enr.node_id(), replaced: None },
        None
    )]
    fn test_record_to_dial(
        #[case] event: impl Fn(Enr) -> discv5::Event,
        #[case] expected: Option<&'static str>,
    ) {
        let enr = localhost_enr(OP_SEPOLIA_CHAIN_ID);
        assert_eq!(
            Discv5Driver::record_to_dial(
                event(enr.clone()),
                OP_SEPOLIA_CHAIN_ID,
                AddrScope::Loopback
            ),
            expected.map(|event| RecordToDial { event, enr })
        );
        assert_eq!(
            Discv5Driver::record_to_dial(
                event(localhost_enr(OP_MAINNET_CHAIN_ID)),
                OP_SEPOLIA_CHAIN_ID,
                AddrScope::Loopback
            ),
            None,
            "a record for another chain is never dialed"
        );
    }

    /// A session with a node whose packets came from `sender`.
    fn session_from(sender: &str) -> impl Fn(Enr) -> discv5::Event {
        let socket = SocketAddr::new(sender.parse().expect("a valid IP"), 9223);
        move |enr| discv5::Event::SessionEstablished(enr, socket)
    }

    /// A session with a node whose record doesn't match `sender`, the address its packets came
    /// from.
    fn unverifiable_from(sender: &str) -> impl Fn(Enr) -> discv5::Event {
        let socket = SocketAddr::new(sender.parse().expect("a valid IP"), 9223);
        move |enr: Enr| discv5::Event::UnverifiableEnr { node_id: enr.node_id(), enr, socket }
    }

    /// A relayed record is dialed only if a node of our own scope can reach it, as discv5
    /// doesn't say which node relayed it: a public node doesn't dial another node's loopback or
    /// LAN address. A session's or an unverifiable record only if it is plausible from the address
    /// its packets came from, whatever our own scope: an honest node's record holds that address.
    #[rstest]
    #[case::discovered_lan_while_lan(discv5::Event::Discovered, "10.0.0.1", AddrScope::Lan, true)]
    #[case::discovered_loopback_while_lan(
        discv5::Event::Discovered,
        "127.0.0.1",
        AddrScope::Lan,
        false
    )]
    #[case::discovered_loopback_while_public(
        discv5::Event::Discovered,
        "127.0.0.1",
        AddrScope::Public,
        false
    )]
    #[case::discovered_lan_while_public(
        discv5::Event::Discovered,
        "10.0.0.1",
        AddrScope::Public,
        false
    )]
    #[case::discovered_public_while_public(
        discv5::Event::Discovered,
        "1.1.1.1",
        AddrScope::Public,
        true
    )]
    #[case::discovered_lan_while_loopback(
        discv5::Event::Discovered,
        "10.0.0.1",
        AddrScope::Loopback,
        true
    )]
    #[case::discovered_special(discv5::Event::Discovered, "192.0.2.1", AddrScope::Loopback, false)]
    #[case::unverifiable_lan_from_loopback_sender(
        unverifiable_from("127.0.0.1"),
        "10.0.0.1",
        AddrScope::Public,
        true
    )]
    #[case::unverifiable_lan_from_public_sender(
        unverifiable_from("1.1.1.1"),
        "10.0.0.1",
        AddrScope::Loopback,
        false
    )]
    #[case::unverifiable_special(
        unverifiable_from("127.0.0.1"),
        "192.0.2.1",
        AddrScope::Loopback,
        false
    )]
    #[case::session_loopback_while_public(
        session_from("127.0.0.1"),
        "127.0.0.1",
        AddrScope::Public,
        true
    )]
    #[case::session_lan_from_public_sender(
        session_from("1.1.1.1"),
        "10.0.0.1",
        AddrScope::Loopback,
        false
    )]
    fn test_record_to_dial_by_scope(
        #[case] event: impl Fn(Enr) -> discv5::Event,
        #[case] ip: IpAddr,
        #[case] own_scope: AddrScope,
        #[case] dialed: bool,
    ) {
        let event = event(enr_at(ip, OP_SEPOLIA_CHAIN_ID));
        let dialed_record = Discv5Driver::record_to_dial(event, OP_SEPOLIA_CHAIN_ID, own_scope);
        assert_eq!(dialed_record.is_some(), dialed);
    }

    /// A session's record without a UDP address, holding loopback with a TCP port. discv5 checks
    /// only a record's UDP address against the address its packets came from, and passes a record
    /// without one.
    fn loopback_record_without_udp() -> Enr {
        enr_with(OP_SEPOLIA_CHAIN_ID, |record| {
            record.ip4(Ipv4Addr::LOCALHOST).tcp4(6379);
        })
    }

    /// A session's record whose IPv4 address matches the session, holding loopback with a TCP
    /// port in its IPv6 address. discv5 checks only the record's address of the family its packets
    /// came in on.
    fn loopback_record_in_other_family() -> Enr {
        enr_with(OP_SEPOLIA_CHAIN_ID, |record| {
            record.ip4(Ipv4Addr::new(1, 1, 1, 1)).udp4(9223).ip6(Ipv6Addr::LOCALHOST).tcp6(6379);
        })
    }

    /// A node should dial a session's record only at addresses plausible from the address the
    /// session's packets came from, as it does an unverifiable record. Gossip dials a record's TCP
    /// address, which discv5 doesn't check, so a public peer could otherwise start a session with
    /// a record that has gossip dial a port on our own loopback.
    #[rstest]
    #[case::without_udp(loopback_record_without_udp())]
    #[case::in_other_family(loopback_record_in_other_family())]
    fn test_session_record_is_checked_against_its_sender(#[case] enr: Enr) {
        let event = session_from("1.1.1.1")(enr);
        assert_eq!(
            Discv5Driver::record_to_dial(event, OP_SEPOLIA_CHAIN_ID, AddrScope::Public),
            None,
            "a public node dialed a loopback session record"
        );
    }

    /// A lookup's results include nodes from discv5's own table, which may have got there through a
    /// session, without the table filter, so they are checked against our own scope too.
    #[rstest]
    #[case::public_while_public("1.1.1.1", OP_SEPOLIA_CHAIN_ID, AddrScope::Public, true)]
    #[case::loopback_while_public("127.0.0.1", OP_SEPOLIA_CHAIN_ID, AddrScope::Public, false)]
    #[case::lan_while_lan("10.0.0.1", OP_SEPOLIA_CHAIN_ID, AddrScope::Lan, true)]
    #[case::other_chain("1.1.1.1", OP_MAINNET_CHAIN_ID, AddrScope::Public, false)]
    fn test_is_lookup_record_to_dial(
        #[case] ip: IpAddr,
        #[case] record_chain_id: u64,
        #[case] own_scope: AddrScope,
        #[case] dialed: bool,
    ) {
        let enr = enr_at(ip, record_chain_id);
        assert_eq!(
            Discv5Driver::is_lookup_record_to_dial(&enr, OP_SEPOLIA_CHAIN_ID, own_scope),
            dialed
        );
    }

    /// A public node should dial only the saved records it can reach, as with the records other
    /// nodes relay to it, so it should leave out a loopback record in its bootstore. Such a record
    /// may have been saved by a version that didn't check the records it dials, or while the node
    /// ran on loopback, and nothing says the node it names is on this host.
    ///
    /// At startup, the driver hands the bootstore's records to gossip one by one, in the order they
    /// were saved, before any record discovery finds: here the loopback record, then the public
    /// one. If it hands over the loopback record, gossip gets that one first. If it leaves it out,
    /// gossip gets the public one first. So the first record shows whether the loopback one was
    /// left out, without waiting to see that it never comes.
    ///
    /// The node has no bootnodes, and listens on loopback whatever it advertises, so it contacts
    /// nobody.
    #[tokio::test]
    async fn test_public_node_leaves_out_saved_loopback_record() {
        // Not in the registry, so it has no bootnodes of its own.
        const CHAIN_ID: u64 = 0xdead_beef;

        let dir = tempdir().unwrap();
        let path = dir.path().join("bootstore.json");
        let loopback = localhost_enr(CHAIN_ID);
        let public = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), CHAIN_ID);
        let mut store: BootStore =
            BootStoreFile::Custom(path.clone()).try_into().expect("the bootstore opens");
        store.add_enr(loopback);
        store.add_enr(public.clone());
        store.sync().expect("the bootstore saves");
        drop(store);

        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };
        let mut driver = Discv5Driver::builder(
            LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::new(1, 2, 3, 4)), 0, 0),
            CHAIN_ID,
            ConfigBuilder::new(SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), 0).into()).build(),
        )
        .build()
        .expect("the driver builds");
        driver.bootstore = Some(BootStoreFile::Custom(path));
        let (_handler, mut records) = driver.start();

        let first = tokio::time::timeout(Duration::from_secs(5), records.recv())
            .await
            .expect("the driver hands over the saved public record")
            .expect("the driver runs");
        assert_eq!(first, public, "a public node dialed a saved loopback record");
    }

    /// A public node should neither contact nor keep a loopback bootnode given by enode URL, as it
    /// skips one given by ENR. An enode URL holds only an address, so the node asks that address
    /// for the bootnode's record, which starts a session and so puts the bootnode in discv5's table
    /// without the table filter.
    #[tokio::test]
    async fn test_public_node_skips_loopback_enode_bootnode() {
        let bootnode = offline_driver().init().await.expect("the bootnode starts");
        let bootnode_enr = bootnode.disc.local_enr();
        let peer_id = kona_peers::enr_to_multiaddr(&bootnode_enr)
            .and_then(|addr| addr.iter().last().map(|protocol| protocol.acquire()))
            .expect("the bootnode's record has a peer ID");
        let enode = Multiaddr::from(Ipv4Addr::LOCALHOST)
            .with(Protocol::Udp(bootnode_enr.udp4().expect("the bootnode has a UDP port")))
            .with(Protocol::Tcp(0))
            .with(peer_id);

        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };
        let node = Discv5Driver::builder(
            LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::new(1, 2, 3, 4)), 0, 0),
            bootnode.chain_id,
            ConfigBuilder::new(SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), 0).into()).build(),
        )
        .build()
        .expect("the node builds")
        .init()
        .await
        .expect("the node starts");
        assert_eq!(node.own_scope, AddrScope::Public);

        Discv5Driver::bootstrap_peers(
            None,
            BootNodes(vec![BootNode::Enode(enode)]),
            node.chain_id,
            node.own_scope,
            &node.disc,
        )
        .await;

        assert!(
            !node
                .disc
                .table_entries_enr()
                .iter()
                .any(|enr| enr.node_id() == bootnode_enr.node_id()),
            "a public node kept a loopback enode bootnode"
        );
    }

    /// Builds a driver listening on localhost that does not contact any remote peer.
    fn offline_driver() -> Discv5Driver {
        // A chain ID that is not in the superchain registry has no default bootnodes.
        const UNKNOWN_CHAIN_ID: u64 = 0xdead_beef;

        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };
        let socket = SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), 0);
        Discv5Driver::builder(
            LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::LOCALHOST), 0, 0),
            UNKNOWN_CHAIN_ID,
            ConfigBuilder::new(socket.into()).build(),
        )
        .build()
        .expect("Failed to build discovery service")
    }

    /// Only an unset TCP port of 0 is set to the gossip TCP port, and only once: another port is
    /// kept, and so is the first gossip port. It fails if the gossip address has no TCP port to set
    /// it to, as with the memory transport. An IPv6 node's port is `tcp6`.
    #[rstest]
    #[case::unset(Ipv4Addr::LOCALHOST.into(), 0, "/ip4/127.0.0.1/tcp/9222", Some(9222))]
    #[case::unset_ipv6(Ipv6Addr::LOCALHOST.into(), 0, "/ip6/::1/tcp/9222", Some(9222))]
    #[case::set(Ipv4Addr::LOCALHOST.into(), 30303, "/ip4/127.0.0.1/tcp/9222", Some(30303))]
    #[case::set_without_gossip_port(Ipv4Addr::LOCALHOST.into(), 30303, "/memory/0", Some(30303))]
    #[case::unset_without_gossip_port(Ipv4Addr::LOCALHOST.into(), 0, "/memory/0", None)]
    #[tokio::test]
    async fn test_advertise_gossip_tcp_port(
        #[case] ip: IpAddr,
        #[case] tcp_port: u16,
        #[case] gossip_addr: Multiaddr,
        #[case] advertised: Option<u16>,
    ) {
        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };
        // A fixed listen port, so that building binds no socket, which IPv6 would need the host to
        // support.
        let driver = Discv5Driver::builder(
            LocalNode::new(secret_key, ip, tcp_port, 0),
            0xdead_beef,
            ConfigBuilder::new(ListenConfig::from_ip(ip, 30303)).build(),
        )
        .build()
        .unwrap();
        let enr_tcp_port = || {
            let enr = driver.disc.local_enr();
            if ip.is_ipv4() { enr.tcp4() } else { enr.tcp6() }
        };

        let result = driver.advertise_gossip_tcp_port(&gossip_addr);

        match advertised {
            Some(port) => {
                result.unwrap();
                assert_eq!(enr_tcp_port(), Some(port));
                // A second address changes nothing.
                driver.advertise_gossip_tcp_port(&"/ip4/127.0.0.1/tcp/1".parse().unwrap()).unwrap();
                assert_eq!(enr_tcp_port(), Some(port));
            }
            None => {
                assert!(matches!(result, Err(AdvertiseTcpPortError::NoGossipTcpPort(_))));
            }
        }
    }

    /// Waits until the driver has finished startup and runs its main loop.
    async fn wait_until_started(handler: &Discv5Handler) {
        // Requests are only answered by the main loop.
        tokio::time::timeout(Duration::from_secs(5), handler.peer_count())
            .await
            .expect("driver should finish startup")
            .expect("driver should not stop during startup");
    }

    /// `BanPeers` bans a peer by its libp2p peer ID, which discovery turns into the node ID of the
    /// same key: banning a node also removes it from the table.
    #[tokio::test]
    async fn test_ban_peers_bans_by_peer_id() {
        let key = CombinedKey::generate_secp256k1();
        let mut builder = Enr::builder();
        builder.ip4(Ipv4Addr::LOCALHOST).udp4(30303).tcp4(9222);
        let enr = builder.build(&key).unwrap();
        // The libp2p peer ID of the same key, at the end of its gossip address.
        let peer_id = match kona_peers::enr_to_multiaddr(&enr).unwrap().pop() {
            Some(libp2p::multiaddr::Protocol::P2p(peer_id)) => peer_id,
            other => panic!("address doesn't end in a peer ID: {other:?}"),
        };

        let driver = offline_driver();
        driver.disc.add_enr(enr.clone()).unwrap();
        let (handler, _enr_recv, _task) = driver.spawn();
        wait_until_started(&handler).await;
        async fn table(handler: &Discv5Handler) -> Vec<Enr> {
            tokio::time::timeout(Duration::from_secs(5), handler.table_enrs())
                .await
                .unwrap()
                .unwrap()
        }
        assert!(table(&handler).await.iter().any(|e| e.node_id() == enr.node_id()));

        handler
            .sender
            .send(HandlerRequest::BanPeers {
                peer_ids: [peer_id].into(),
                ban_duration: Duration::from_secs(60),
            })
            .await
            .unwrap();

        assert!(!table(&handler).await.iter().any(|e| e.node_id() == enr.node_id()));
    }

    /// [`Discv5Driver::start`]: the task stops when every [`Discv5Handler`] clone and the ENR
    /// receiver are dropped, without waiting for the next discovery tick.
    #[tokio::test]
    async fn test_discv5_driver_stops_when_handler_and_receiver_dropped() {
        let (handler, enr_recv, task) = offline_driver().spawn();

        wait_until_started(&handler).await;
        // Let the intervals fire their immediate first ticks, so the loop is idle.
        sleep(Duration::from_millis(100)).await;

        // Drop the handler first, like the network actor does, while the loop is waiting.
        drop(handler);
        sleep(Duration::from_millis(10)).await;
        drop(enr_recv);

        // Well within the discovery interval, so the driver must not wait for the next tick.
        tokio::time::timeout(Duration::from_secs(1), task)
            .await
            .expect("driver task should stop once nobody is listening")
            .expect("driver task should not panic");
    }

    /// [`Discv5Driver::start`]: dropping only the handler keeps the task running for the ENR
    /// receiver.
    #[tokio::test]
    async fn test_discv5_driver_keeps_running_for_enr_receiver() {
        let (handler, _enr_recv, task) = offline_driver().spawn();

        wait_until_started(&handler).await;
        drop(handler);

        sleep(Duration::from_millis(500)).await;
        assert!(!task.is_finished(), "driver task should keep running for the ENR receiver");
    }

    /// Once every handler was dropped, `recv` returned `None` on every iteration, so the loop kept
    /// a core busy for as long as the ENR receiver kept the task running.
    #[tokio::test]
    async fn regression_discv5_driver_idles_for_enr_receiver() {
        let (handler, _enr_recv, task) = offline_driver().spawn();

        wait_until_started(&handler).await;
        drop(handler);
        // Let the intervals fire their immediate first ticks, so the loop is idle.
        sleep(Duration::from_millis(100)).await;

        // The test runtime is single-threaded, so its only worker also runs the driver.
        let metrics = tokio::runtime::Handle::current().metrics();
        let before = metrics.worker_total_busy_duration(0);
        sleep(Duration::from_millis(500)).await;
        let busy = metrics.worker_total_busy_duration(0) - before;

        assert!(!task.is_finished(), "driver task should keep running for the ENR receiver");
        assert!(busy < Duration::from_millis(10), "driver should idle, but was busy for {busy:?}");
    }

    /// [`Discv5Driver::start`]: dropping only the ENR receiver keeps the task running for the
    /// handler, which still gets answers.
    #[tokio::test]
    async fn test_discv5_driver_keeps_running_for_handler() {
        let (handler, enr_recv, task) = offline_driver().spawn();

        wait_until_started(&handler).await;
        drop(enr_recv);

        sleep(Duration::from_millis(500)).await;
        assert!(!task.is_finished(), "driver task should keep running for the handler");
        tokio::time::timeout(Duration::from_secs(5), handler.peer_count())
            .await
            .expect("driver should answer requests")
            .expect("driver should not stop while the handler is alive");
    }

    /// [`Discv5Driver::start`]: requests sent before the handler is dropped are answered, and
    /// those sent during startup once it completes. Here the request is sent during startup and
    /// both ends are dropped before it completes.
    #[tokio::test]
    async fn test_discv5_driver_answers_requests_sent_before_handler_dropped() {
        // `select!` picks a random branch among those ready, so a race between handling the
        // request and stopping would only fail some of the time.
        for _ in 0..20 {
            let (handler, enr_recv, _task) = offline_driver().spawn();
            // Send through the channel, since a handler's request is only sent once awaited.
            let (tx, reply) = tokio::sync::oneshot::channel();
            handler.sender.send(HandlerRequest::PeerCount(tx)).await.unwrap();
            drop(handler);
            drop(enr_recv);

            tokio::time::timeout(Duration::from_secs(5), reply)
                .await
                .expect("driver should answer requests")
                .expect("driver should not drop requests sent before the handler was dropped");
        }
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
            discovery.own_scope,
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
            discovery.own_scope,
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
