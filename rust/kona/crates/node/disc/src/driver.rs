//! Discovery Module.

use backon::{ExponentialBuilder, RetryableWithContext};
use derive_more::Debug;
use discv5::{Config, Discv5, Enr, NodeContact};
use enr::NodeId;
use kona_peers::{
    BootNode, BootNodes, BootStore, BootStoreFile, EnrValidation, peer_id_to_node_id,
};
use libp2p::{Multiaddr, multiaddr::Protocol};
use std::{net::IpAddr, pin::Pin};
use tokio::{
    sync::mpsc::channel,
    task::JoinHandle,
    time::{Duration, sleep},
};
use tracing::Instrument;

use crate::{AdvertiseTcpPortError, Discv5Builder, Discv5Handler, HandlerRequest, LocalNode};

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

/// Returns whether the TCP address gossip dials from `enr`, see [`kona_peers::enr_tcp_socket`], has
/// `session_ip`, the IP that a session with the node came from. Then the session has confirmed the
/// address gossip dials, which is also the one it dials after a restart, once the bootstore has
/// saved the record.
fn holds_session_ip(enr: &Enr, session_ip: IpAddr) -> bool {
    kona_peers::enr_tcp_socket(enr)
        .is_some_and(|socket| socket.ip().to_canonical() == session_ip.to_canonical())
}

/// Returns whether `table_filter` accepts the address of `enode`, a bootnode's enode URL, so that
/// discovery may contact it: it judges a record at that address, signed with a throwaway key.
///
/// Discovery asks an enode bootnode for its record, which starts a session, and discv5 keeps a
/// node it has a session with whatever its address, so the address is checked before contact. It
/// reads the address with the parse `Discv5::request_enr` does, so it judges the one discovery
/// would contact, and fails with that parse's error for an enode URL discovery can't contact, such
/// as one with a DNS name or without a UDP port.
fn accepts_enode(table_filter: fn(&Enr) -> bool, enode: &Multiaddr) -> Result<bool, &'static str> {
    let contact = NodeContact::try_from_multiaddr(enode.clone())?;
    let mut record = Enr::builder();
    match contact.socket_addr().ip() {
        IpAddr::V4(ip) => record.ip4(ip),
        IpAddr::V6(ip) => record.ip6(ip),
    };
    Ok(record
        .build(&enr::CombinedKey::generate_secp256k1())
        .is_ok_and(|record| table_filter(&record)))
}

/// Hands gossip the saved records, the bootnodes and the records a session confirmed, in a task of
/// its own. Gossip dials those it isn't connected to. It redials a peer only when discovery hands
/// it over again, and discovery reports a session only when it starts, which discv5 keeps going
/// for as long as it pings the node.
fn forward_saved(store: &BootStore, chain_id: u64, sender: &tokio::sync::mpsc::Sender<Enr>) {
    let enrs: Vec<Enr> = store.valid_peers_with_chain_id(chain_id).into_iter().cloned().collect();
    let sender = sender.clone();
    spawn_in_current_span(async move {
        for enr in enrs {
            if sender.send(enr).await.is_err() {
                return;
            }
        }
    });
}

/// A record our own gossip should dial, see [`Discv5Driver::record_to_dial`].
#[derive(Debug, PartialEq, Eq)]
struct RecordToDial {
    /// The name of the discovery event that brought it, for metrics and logs.
    event: &'static str,
    /// The record.
    enr: Enr,
    /// Whether the bootstore should save it, which the table filter decides.
    save: bool,
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
    ///
    /// discv5 has one event stream, which the driver takes when it starts: a stream taken from
    /// here before then closes at that point.
    #[debug(skip)]
    pub disc: Discv5,
    /// The optional [`BootStoreFile`] to use for the bootstore.
    pub bootstore: Option<BootStoreFile>,
    /// Bootnodes used to bootstrap the discovery service.
    pub bootnodes: BootNodes,
    /// The chain ID of the network.
    pub chain_id: u64,
    /// discv5's table filter, which saved records must pass too before they are dialed. Not
    /// public, as it must stay the one discv5 was built with.
    pub(crate) table_filter: fn(&Enr) -> bool,
    /// The interval to discovery random nodes.
    pub interval: Duration,
    /// Whether to hand the saved records to the enr receiver, at startup and then every
    /// `store_interval`, so that gossip redials the peers it lost.
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

    /// Instantiates a new [`Discv5Driver`]. `table_filter` must be the one `disc` was built with,
    /// as the driver also drops the saved records it rejects. `disc` should be built with
    /// `report_discovered_peers` off, as [`Discv5Builder`] does: the driver doesn't use those
    /// reports, and they can push the events it does use out of discv5's event channel.
    pub const fn new(
        disc: Discv5,
        table_filter: fn(&Enr) -> bool,
        interval: Duration,
        chain_id: u64,
        bootstore: Option<BootStoreFile>,
        bootnodes: BootNodes,
    ) -> Result<Self, std::io::Error> {
        Ok(Self {
            disc,
            chain_id,
            table_filter,
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

    /// Returns the peer a discovery event brings, if our own gossip should dial it: only a peer
    /// discovery has had a session with, and only if the address gossip dials from its record has
    /// the IP the session's packets came from, as only that address has shown that the node runs
    /// discovery. The driver then hands it to gossip, and saves its record in the bootstore, which
    /// dials the same address after a restart. Only a record valid for `chain_id` is dialed.
    ///
    /// A record that `table_filter` rejects, such as a private one on a node that uses only public
    /// addresses, is dialed, but not saved. Our lookups and bootnodes go through the filter, so
    /// such a session is one the other node started, from the address we dial. Refusing the dial
    /// wouldn't keep the two apart, as that node can dial us, and gossip accepts it. But after a
    /// restart, the bootstore would dial the record without a session.
    ///
    /// The record comes from these events:
    ///
    /// - `SessionEstablished`: a session with the node started, in either direction. Our lookups
    ///   start one with each node they contact. discv5 only checked the record's UDP address of the
    ///   family the packets came in on, if it has one, not the TCP address gossip dials.
    ///
    ///   discv5 also adds such a node to its table without the table filter, and relays its record
    ///   to the nodes that look it up, who may not be able to reach it. discv5 has no hook to check
    ///   a record against the node asking for it, as geth does, so it is left to that node to
    ///   reject it.
    /// - `UnverifiableEnr`: a node started a session with us, but the record it presented doesn't
    ///   match the session: its UDP address isn't the one the packets came from, or it is another
    ///   node's record. discv5 keeps the session but leaves the node out of its table. The record
    ///   is dialed only if it is the session's node's, and its IP is the session's, as when a NAT
    ///   rewrites only the port the packets come from. Another node's record isn't: no session has
    ///   confirmed that node.
    ///
    /// A `Discovered` record, from another node's answer to one of our lookups, isn't dialed:
    /// discv5 sends it before any check, and nothing shows that the node is at the address in it.
    /// The lookup that found it contacts it, if the table filter lets it, and a session follows.
    /// [`Discv5Builder`] turns these reports off, so the driver warns if they arrive.
    fn record_to_dial(
        event: discv5::Event,
        chain_id: u64,
        table_filter: fn(&Enr) -> bool,
    ) -> Option<RecordToDial> {
        let (kind, enr, socket) = match event {
            discv5::Event::Discovered(_) => {
                static WARNED: std::sync::Once = std::sync::Once::new();
                WARNED.call_once(|| {
                    warn!(target: "discovery", "discv5 reports discovered records, which the driver expects to be off: they can fill discv5's event channel and push out the session events that bring the peers to dial; turn off report_discovered_peers, as Discv5Builder does");
                });
                return None;
            }
            discv5::Event::SessionEstablished(enr, socket) => ("session_established", enr, socket),
            discv5::Event::UnverifiableEnr { enr, socket, node_id } => {
                if enr.node_id() != node_id {
                    debug!(target: "discovery", %socket, ?enr, "Ignoring another node's record, presented in a session");
                    return None;
                }
                ("unverifiable_enr", enr, socket)
            }
            _ => return None,
        };
        if EnrValidation::validate(&enr, chain_id).is_invalid() {
            return None;
        }
        if !holds_session_ip(&enr, socket.ip()) {
            debug!(target: "discovery", event = kind, %socket, "Ignoring a record without a TCP address at the IP its session came from");
            return None;
        }
        let save = table_filter(&enr);
        if !save {
            debug!(target: "discovery", event = kind, %socket, "Not saving a session at an address discovery doesn't use; --p2p.discovery.private allows private ones");
        }
        Some(RecordToDial { event: kind, enr, save })
    }

    /// Bootstraps the [`Discv5`] table with bootnodes.
    async fn bootstrap_peers(
        bootstore: Option<BootStoreFile>,
        bootnodes: BootNodes,
        chain_id: u64,
        table_filter: fn(&Enr) -> bool,
        disc: &Discv5,
    ) -> BootStore {
        // Note: if the bootstore file cannot be created, we use a default bootstore.
        let mut store = bootstore
            .map_or_else(BootStore::default, |bootstore| bootstore.try_into().unwrap_or_default());

        let initial_store_length = store.len();

        for bn in bootnodes.0.into_iter().chain(BootNodes::from_chain_id(chain_id).0) {
            let res = match bn {
                BootNode::Enr(enr) => Ok(enr),
                BootNode::Enode(enode) => {
                    match accepts_enode(table_filter, &enode) {
                        Ok(true) => {}
                        Ok(false) => {
                            warn!(target: "discovery::bootstrap", %enode, "Ignoring a boot node at an address discovery doesn't use; --p2p.discovery.private allows private ones");
                            continue;
                        }
                        Err(err) => {
                            warn!(target: "discovery::bootstrap", %enode, err, "Ignoring a boot node discovery can't contact");
                            continue;
                        }
                    }
                    disc.request_enr(enode).await
                }
            };

            let Ok(enr) = res else {
                debug!(target: "discovery::bootstrap", ?res, "Failed to add boot node ENR to discovery table");
                continue;
            };

            // A bootnode given by enode URL may be in the table already, from the session that
            // fetched its record, which skips the table filter: remove the ones not kept.
            if !table_filter(&enr) {
                warn!(target: "discovery::bootstrap", ?enr, "Ignoring a boot node at an address discovery doesn't use; --p2p.discovery.private allows private ones");
                disc.remove_node(&enr.node_id());
                continue;
            }

            // Every bootnode is an entry point for lookups, which find the nodes of every chain, so
            // it goes into the table whatever its chain. One given by enode URL is there already,
            // from the session that fetched its record.
            if let Err(e) = disc.add_enr(enr.clone()) {
                debug!(target: "discovery::bootstrap", ?enr, "Failed to add boot node: {e}");
                continue;
            }

            // But only a peer of our chain is saved, and so dialed by gossip. OP Mainnet's built-in
            // enode bootnodes, for one, have no `opstack` key, and its built-in ENRs are of other
            // chains.
            let validation = EnrValidation::validate(&enr, chain_id);
            if validation.is_invalid() {
                trace!(target: "discovery::bootstrap", ?enr, %validation, "Not saving a boot node that isn't a peer of this chain");
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
    /// records for gossip to dial: those of the peers discovery has a session with, at the address
    /// the session came from, and the saved records, at startup and again every store interval, so
    /// that gossip redials the peers it lost. It doesn't yield the results of lookups.
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
        let table_filter = self.table_filter;
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

            // Step 2: Start the event stream before any session starts: discv5 drops the events
            // it has no stream for, and a session is reported only when it starts.
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

            // Step 3: Bootstrap the discovery table with bootnodes.
            let mut store = Self::bootstrap_peers(
                self.bootstore,
                self.bootnodes,
                chain_id,
                table_filter,
                &self.disc,
            )
            .await;

            let enrs = self.disc.table_entries_enr();
            info!(target: "discovery", "Discv5 Started with {} ENRs", enrs.len());

            // Step 4: Remove the saved records the table filter rejects: the bootstore may hold
            // records saved by an earlier version, which didn't check them, or while the filter
            // allowed more. The file changes at the next sync, once the driver has started.
            let removed = store.retain(table_filter);
            if removed > 0 {
                debug!(target: "discovery", removed, "Removed saved records at addresses discovery doesn't use");
            }

            // Step 5: Run the core driver loop. Its first store tick, which is immediate, saves the
            // bootstore and hands its records to gossip.
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
                        if let Some(RecordToDial { event: kind, enr, save }) =
                            Self::record_to_dial(event, chain_id, table_filter)
                        {
                            debug!(target: "discovery", event = kind, ?enr, "Valid ENR, passing it to gossip to dial");
                            metrics::gauge!(crate::Metrics::DISCOVERY_EVENT, "type" => kind).increment(1);
                            if save {
                                store.add_enr(enr.clone());
                            }
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
                        // Its results aren't dialed: the lookup contacts them, and gossip dials
                        // those discovery gets a session with.
                        spawn_in_current_span(async move {
                            if let Err(err) = fut.await {
                                info!(target: "discovery", "Failed to find node: {:?}", err);
                            }
                        });
                    }
                    _ = store_interval.tick() => {
                        let start = std::time::Instant::now();
                        // Only what the driver added: the bootnodes, and the records a session
                        // confirmed. discv5's table also holds records from other nodes' answers
                        // to our lookups, which nothing confirmed, and the bootstore would dial
                        // them at the address in them after a restart.
                        if let Err(e) = store.sync() {
                            warn!(target: "discovery", "Failed to sync bootstore: {:?}", e);
                        }
                        if self.forward {
                            forward_saved(&store, chain_id, &enr_sender);
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
    use discv5::{ConfigBuilder, ListenConfig};
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

    /// Gossip dials the peer a session or an unverifiable record brings, if it is for the driver's
    /// chain, and nothing for other events: a discovered record isn't dialed until a session with
    /// its node shows where it is.
    #[rstest]
    #[case::discovered(discv5::Event::Discovered, None)]
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
            Discv5Driver::record_to_dial(event(enr.clone()), OP_SEPOLIA_CHAIN_ID, accept_all),
            expected.map(|event| RecordToDial { event, enr, save: true })
        );
        assert_eq!(
            Discv5Driver::record_to_dial(
                event(localhost_enr(OP_MAINNET_CHAIN_ID)),
                OP_SEPOLIA_CHAIN_ID,
                accept_all
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

    /// A session from `sender` with another node than the one whose record it presented.
    fn unverifiable_with_other_node_from(sender: &str) -> impl Fn(Enr) -> discv5::Event {
        let socket = SocketAddr::new(sender.parse().expect("a valid IP"), 9223);
        move |enr: Enr| discv5::Event::UnverifiableEnr { node_id: NodeId::random(), enr, socket }
    }

    /// A session's record holding another public host's address, such as a web server's, with a
    /// TCP port and no UDP port, which discv5 doesn't check against the session.
    fn third_party_record() -> Enr {
        enr_with(OP_SEPOLIA_CHAIN_ID, |record| {
            record.ip4(Ipv4Addr::new(93, 184, 216, 34)).tcp4(443);
        })
    }

    /// A session's record holding loopback with a TCP port, and no UDP port.
    fn loopback_record_without_udp() -> Enr {
        enr_with(OP_SEPOLIA_CHAIN_ID, |record| {
            record.ip4(Ipv4Addr::LOCALHOST).tcp4(6379);
        })
    }

    /// A session's record whose IPv4 address has a UDP port but no TCP port, with loopback and a
    /// TCP port in its IPv6 address.
    fn record_with_tcp_only_in_other_family() -> Enr {
        enr_with(OP_SEPOLIA_CHAIN_ID, |record| {
            record.ip4(Ipv4Addr::new(1, 2, 3, 4)).udp4(9223).ip6(Ipv6Addr::LOCALHOST).tcp6(6379);
        })
    }

    /// A record whose address matches its session's: `1.2.3.4`, with UDP and TCP ports.
    fn matching_record() -> Enr {
        enr_with(OP_SEPOLIA_CHAIN_ID, |record| {
            record.ip4(Ipv4Addr::new(1, 2, 3, 4)).udp4(9223).tcp4(9222);
        })
    }

    /// A session should confirm the address gossip dials: a node should dial, and save, a
    /// session's record only if the address gossip would dial from it, its IPv4 TCP address or else
    /// its IPv6 one, has the IP the session's packets came from. Then it dials that address. A
    /// record naming another host is dropped, not dialed elsewhere, because the bootstore saves
    /// the record: after a restart, it would be dialed at the address in it.
    ///
    /// An unverifiable record whose IP matches, but not its UDP port, as behind a NAT that
    /// rewrites ports, is dialed too, but only if it is the session's node's: one that another
    /// node presented hasn't been confirmed by any session with the node it names.
    #[rstest]
    #[case::matching(session_from("1.2.3.4"), matching_record(), Some(9222))]
    #[case::matching_ip_unverifiable(unverifiable_from("1.2.3.4"), matching_record(), Some(9222))]
    #[case::matching_ip_other_node(
        unverifiable_with_other_node_from("1.2.3.4"),
        matching_record(),
        None
    )]
    #[case::third_party(session_from("1.2.3.4"), third_party_record(), None)]
    #[case::third_party_unverifiable(unverifiable_from("1.2.3.4"), third_party_record(), None)]
    #[case::loopback(session_from("1.2.3.4"), loopback_record_without_udp(), None)]
    #[case::tcp_only_in_other_family(
        session_from("1.2.3.4"),
        record_with_tcp_only_in_other_family(),
        None
    )]
    fn test_session_record_is_dialed_only_at_the_session_address(
        #[case] event: impl Fn(Enr) -> discv5::Event,
        #[case] record: Enr,
        #[case] tcp_port: Option<u16>,
    ) {
        let dialed = Discv5Driver::record_to_dial(event(record), OP_SEPOLIA_CHAIN_ID, accept_all)
            .map(|record| {
                let addr = kona_peers::enr_to_multiaddr(&record.enr)
                    .expect("a dialed record has a TCP address");
                addr.iter().take(2).map(|p| p.acquire()).collect::<Vec<_>>()
            });
        assert_eq!(
            dialed,
            tcp_port
                .map(|port| vec![Protocol::Ip4(Ipv4Addr::new(1, 2, 3, 4)), Protocol::Tcp(port)]),
        );
    }

    /// A node with a public-only table filter should dial only the saved records it can reach, as
    /// with the records other nodes relay to it, so it should leave out a loopback record in its
    /// bootstore. Such a record may have been saved by a version that didn't check the records it
    /// dials, or while the node ran on loopback, and nothing says the node it names is on this
    /// host.
    ///
    /// At startup, the driver hands the bootstore's records to gossip one by one, in the order they
    /// were saved, and this node has no bootnodes or peers to find others through: here the
    /// loopback record, then the public one. If it hands over the loopback record, gossip gets that
    /// one first. If it leaves it out, gossip gets the public one first. So the first record shows
    /// whether the loopback one was left out, without waiting to see that it never comes.
    #[tokio::test]
    async fn test_public_node_leaves_out_saved_loopback_record() {
        let loopback = localhost_enr(BOOTSTORE_CHAIN_ID);
        let public = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), BOOTSTORE_CHAIN_ID);
        let (driver, _dir) = public_driver_with_bootstore(&[loopback, public.clone()]);
        let (_handler, mut records) = driver.start();

        let first = tokio::time::timeout(Duration::from_secs(5), records.recv())
            .await
            .expect("the driver hands over the saved public record")
            .expect("the driver runs");
        assert_eq!(first, public, "a public node dialed a saved loopback record");
    }

    /// Gossip redials a peer only when discovery hands it over again, and discovery reports a
    /// session only once, when it starts. So the driver should hand the saved records over again
    /// every store interval, for gossip to redial the peers it lost.
    #[tokio::test]
    async fn test_saved_records_are_handed_over_again() {
        let public = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), BOOTSTORE_CHAIN_ID);
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
        let loopback = localhost_enr(BOOTSTORE_CHAIN_ID);
        let public = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), BOOTSTORE_CHAIN_ID);
        let (driver, dir) = public_driver_with_bootstore(&[loopback.clone(), public.clone()]);
        let path = dir.path().join("bootstore.json");
        let (_handler, _records) = driver.start();

        // Read while the driver may be writing, so a read can come back empty: wait for one that
        // holds the public record too.
        let saved_without_loopback = async {
            loop {
                let store: BootStore =
                    BootStoreFile::Custom(path.clone()).try_into().expect("the bootstore opens");
                let saved = store.valid_peers();
                if saved.contains(&&public) && !saved.contains(&&loopback) {
                    return;
                }
                tokio::time::sleep(Duration::from_millis(50)).await;
            }
        };
        tokio::time::timeout(Duration::from_secs(5), saved_without_loopback)
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
        let kept = enr_at(IpAddr::V4(Ipv4Addr::new(127, 0, 0, 3)), BOOTSTORE_CHAIN_ID);
        let removed = enr_at(IpAddr::V4(Ipv4Addr::new(192, 0, 2, 1)), BOOTSTORE_CHAIN_ID);
        let (driver, dir) = driver_with_bootstore(
            |enr| enr.ip4().is_some_and(|ip| ip.is_loopback()),
            &[kept.clone(), removed.clone()],
        );
        // As if another node had returned it to one of our lookups.
        let unconfirmed = enr_at(IpAddr::V4(Ipv4Addr::new(127, 0, 0, 2)), BOOTSTORE_CHAIN_ID);
        driver.disc.add_enr(unconfirmed.clone()).expect("the table takes the record");
        let path = dir.path().join("bootstore.json");
        let (_handler, _records) = driver.start();

        let saved_after_startup = async {
            loop {
                let store: BootStore =
                    BootStoreFile::Custom(path.clone()).try_into().expect("the bootstore opens");
                let saved = store.valid_peers().into_iter().cloned().collect::<Vec<_>>();
                if saved.contains(&kept) && !saved.contains(&removed) {
                    return saved;
                }
                tokio::time::sleep(Duration::from_millis(50)).await;
            }
        };
        let saved = tokio::time::timeout(Duration::from_secs(5), saved_after_startup)
            .await
            .expect("the driver saves its bootstore at startup");
        assert!(!saved.contains(&unconfirmed), "the bootstore saved an unconfirmed record");
    }

    /// The chain ID of the bootstore tests: not in the registry, so it has no bootnodes of its own.
    const BOOTSTORE_CHAIN_ID: u64 = 0xdead_beef;

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
        let path = dir.path().join("bootstore.json");
        let mut store: BootStore =
            BootStoreFile::Custom(path.clone()).try_into().expect("the bootstore opens");
        for record in records {
            store.add_enr(record.clone());
        }
        store.sync().expect("the bootstore saves");
        drop(store);

        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };
        let mut driver = Discv5Driver::builder(
            LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::LOCALHOST), 0, 0),
            BOOTSTORE_CHAIN_ID,
            ConfigBuilder::new(SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), 0).into())
                .table_filter(table_filter)
                .build(),
        )
        .build()
        .expect("the driver builds");
        driver.bootstore = Some(BootStoreFile::Custom(path));
        (driver, dir)
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

        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };
        let node = Discv5Driver::builder(
            LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::LOCALHOST), 0, 0),
            bootnode.chain_id,
            ConfigBuilder::new(SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), 0).into())
                .table_filter(crate::public_only)
                .build(),
        )
        .build()
        .expect("the node builds")
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
            !node
                .disc
                .table_entries_enr()
                .iter()
                .any(|enr| enr.node_id() == bootnode_enr.node_id()),
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

    /// The driver should take discv5's event stream before it bootstraps, which can take seconds
    /// when bootnodes don't answer: discv5 drops the events it has no stream for, so a node that
    /// starts a session with us meanwhile would never be dialed.
    ///
    /// N's only bootnode is a socket that never answers, so N's bootstrap waits for its request to
    /// time out, 3s here. P has N as its bootnode, and contacts N in its first lookup, during N's
    /// bootstrap.
    #[tokio::test]
    async fn test_driver_reports_sessions_started_during_bootstrap() {
        let silent = std::net::UdpSocket::bind((Ipv4Addr::LOCALHOST, 0)).expect("a free UDP port");
        let silent_peer_id = kona_peers::enr_to_multiaddr(&localhost_enr(OP_SEPOLIA_CHAIN_ID))
            .and_then(|addr| addr.iter().last().map(|protocol| protocol.acquire()))
            .expect("the record has a peer ID");
        let silent_enode = Multiaddr::from(Ipv4Addr::LOCALHOST)
            .with(Protocol::Udp(silent.local_addr().expect("the socket is bound").port()))
            .with(silent_peer_id);

        let CombinedKey::Secp256k1(secret_key) = CombinedKey::generate_secp256k1() else {
            unreachable!()
        };
        let listen = SocketAddr::new(IpAddr::V4(Ipv4Addr::LOCALHOST), 0);
        let mut node = Discv5Driver::builder(
            LocalNode::new(secret_key, IpAddr::V4(Ipv4Addr::LOCALHOST), 0, 0),
            BOOTSTORE_CHAIN_ID,
            ConfigBuilder::new(listen.into())
                .request_timeout(Duration::from_secs(3))
                .request_retries(0)
                .build(),
        )
        .build()
        .expect("the node builds");
        node.bootnodes = BootNodes(vec![BootNode::Enode(silent_enode)]);
        let node_enr = node.disc.local_enr();
        let (_node, mut records) = node.start();

        let mut peer = offline_driver();
        peer.bootnodes = BootNodes(vec![BootNode::Enr(node_enr)]);
        peer.interval = Duration::from_millis(100);
        let peer_id = peer.disc.local_enr().node_id();
        let (_peer, _) = peer.start();

        let handed_over = async {
            loop {
                if records.recv().await.expect("the driver runs").node_id() == peer_id {
                    return;
                }
            }
        };
        tokio::time::timeout(Duration::from_secs(10), handed_over)
            .await
            .expect("the node didn't hand over a peer that started a session during its bootstrap");
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

    /// discv5 reports every record in a NODES answer as `Discovered`, up to 16 an answer, on an
    /// event channel of 100, 30 with the reports off, that drops the events that don't fit. The
    /// driver doesn't dial those records, so it should turn the reports off: otherwise they can
    /// push out the `SessionEstablished` events that bring the peers it dials.
    ///
    /// N looks up O through B, whose table holds O. A lookup asks each node for the nodes at its
    /// distance to the target, so B's answer holds O only for a target O shares a bucket with:
    /// O's own ID is one.
    #[tokio::test]
    async fn test_driver_turns_off_discovered_reports() {
        let other = offline_driver().init().await.expect("the other node starts");
        let other_id = other.disc.local_enr().node_id();
        let bootnode = offline_driver().init().await.expect("the bootnode starts");
        bootnode.disc.add_enr(other.disc.local_enr()).expect("the bootnode takes the record");
        let node = offline_driver().init().await.expect("the node starts");
        let mut events = node.disc.event_stream().await.expect("the node has an event stream");
        node.disc.add_enr(bootnode.disc.local_enr()).expect("the node takes the bootnode");

        let found = node.disc.find_node(other_id).await.expect("the lookup runs");
        assert!(
            found.iter().any(|enr| enr.node_id() == other_id),
            "the lookup didn't find the bootnode's other node"
        );

        // discv5 reports a record before the lookup that found it ends.
        while let Ok(event) = events.try_recv() {
            assert!(
                !matches!(event, discv5::Event::Discovered(_)),
                "discv5 reported a discovered record: {event:?}"
            );
        }
    }

    /// A table filter that accepts every record, for the tests of the other rules.
    fn accept_all(_: &Enr) -> bool {
        true
    }

    /// A session's record is dialed whatever the table filter, but saved only if the filter
    /// accepts it, as the bootstore dials what it holds without a session.
    #[rstest]
    #[case::public_only(crate::public_only, false)]
    #[case::allowing_private(accept_all, true)]
    fn test_record_to_dial_saves_only_what_the_table_filter_accepts(
        #[case] table_filter: fn(&Enr) -> bool,
        #[case] save: bool,
    ) {
        let enr = localhost_enr(OP_SEPOLIA_CHAIN_ID);
        let event = session_from("127.0.0.1")(enr.clone());
        assert_eq!(
            Discv5Driver::record_to_dial(event, OP_SEPOLIA_CHAIN_ID, table_filter),
            Some(RecordToDial { event: "session_established", enr, save })
        );
    }

    /// A node with a public-only table filter should dial a node at a private address back after a
    /// session with it, but not save it. The private node reached us first: our lookups and
    /// bootnodes go through the filter, so a session from a private address is one the other node
    /// started. Its packets came from that address, so the dial reaches a node on our own network,
    /// not a host it pointed us at. Refusing the dial wouldn't keep the two apart either, as the
    /// private node can dial us, and gossip accepts it. Not saving the record keeps private
    /// addresses out of the bootstore, which dials what it holds after a restart.
    ///
    /// The driver dials a record by handing it over to gossip, which dials it. There's no gossip
    /// here, so the test checks the handover.
    ///
    /// N has a public-only filter, and a bootstore with a public record. L has no filter, and has
    /// N as its bootnode, given by an enode URL on loopback, so L's bootstrap starts a session with
    /// N from loopback, which L's record holds.
    #[tokio::test]
    async fn test_public_only_node_dials_a_private_session_back_without_saving_it() {
        let public = enr_at(IpAddr::V4(Ipv4Addr::new(1, 1, 1, 1)), BOOTSTORE_CHAIN_ID);
        let (mut node, dir) = public_driver_with_bootstore(std::slice::from_ref(&public));
        node.store_interval = Duration::from_millis(100);
        let path = dir.path().join("bootstore.json");
        let node_enr = node.disc.local_enr();
        let enode = loopback_enode(&node_enr);
        let (_node, mut records) = node.start();

        let mut loopback = offline_driver();
        loopback.bootnodes = BootNodes(vec![BootNode::Enode(enode)]);
        let loopback_id = loopback.disc.local_enr().node_id();
        let (_loopback, _) = loopback.start();

        // The driver also hands over the saved public record, at startup and every store interval.
        let handed_over = async {
            loop {
                let record = records.recv().await.expect("the driver runs");
                if record.node_id() == loopback_id {
                    return record;
                }
            }
        };
        let handed_over = tokio::time::timeout(Duration::from_secs(5), handed_over)
            .await
            .expect("a public-only node didn't hand a session at a loopback address to gossip");

        // Wait for a save after the handover. A read while the driver writes can come back empty,
        // so wait for one that holds the public record.
        let handed_over_at = std::time::SystemTime::now();
        let saved_after_handover = async {
            loop {
                let modified = std::fs::metadata(&path).and_then(|meta| meta.modified());
                if modified.is_ok_and(|modified| modified > handed_over_at) {
                    let store: BootStore = BootStoreFile::Custom(path.clone())
                        .try_into()
                        .expect("the bootstore opens");
                    let saved = store.valid_peers().into_iter().cloned().collect::<Vec<_>>();
                    if saved.contains(&public) {
                        return saved;
                    }
                }
                tokio::time::sleep(Duration::from_millis(50)).await;
            }
        };
        let saved = tokio::time::timeout(Duration::from_secs(5), saved_after_handover)
            .await
            .expect("the driver saves its bootstore every store interval");
        assert!(!saved.contains(&handed_over), "a public-only node saved a loopback record");
    }

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
        let enode =
            addr.parse::<Multiaddr>().expect("a valid multiaddr").with(Protocol::P2p(peer_id));
        assert_eq!(accepts_enode(table_filter, &enode).ok(), expected);
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
}
