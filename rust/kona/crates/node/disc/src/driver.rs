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
mod tests;
