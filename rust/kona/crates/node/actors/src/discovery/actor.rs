//! Discovery construction and execution.

use super::{Capacity, Handle, Network, handle::Message};
use discv5::{Discv5, Enr, Event, QueryError};
use enr::NodeId;
use futures::{StreamExt, future::BoxFuture, stream::FuturesUnordered};
use kona_disc::{Discv5Driver, Metrics};
use kona_peers::{BootStore, EnrValidation, enr_to_multiaddr};
use std::{future::Future, time::Duration};
use thiserror::Error;
use tokio::{
    select,
    sync::mpsc::{self, error::TrySendError},
    time::{self, Interval},
};

/// How many times to ask discv5 for its event stream before giving up.
const EVENT_STREAM_ATTEMPTS: usize = 10;
/// How long to wait between requests for the event stream.
const EVENT_STREAM_RETRY_INTERVAL: Duration = Duration::from_secs(2);
/// The routing table must hold more than this many nodes before one is removed at random.
const MIN_TABLE_SIZE_FOR_REMOVAL: usize = 20;

/// Constructs the handle and task.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    messages: mpsc::Receiver<Message>,
}

impl Builder {
    /// Creates the builder.
    pub fn new(capacity: Capacity) -> Self {
        let (messages_tx, messages) = mpsc::channel(capacity.get());
        Self { handle: Handle::new(messages_tx), messages }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// The future starts the discv5 service and bootstraps its table before handling anything.
    /// Dropping the future stops the actor and the service.
    pub fn build(
        self,
        driver: Discv5Driver,
        network: impl Network,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let messages = self.messages;
        async move { Actor::start(driver, messages, network).await?.run().await }
    }
}

/// A fatal error from the discovery actor.
#[derive(Debug, Error)]
pub enum ActorError {
    /// The discv5 service did not start.
    #[error("Failed to start discv5: {0:?}")]
    Start(discv5::Error),
    /// discv5 did not provide its event stream.
    #[error("Failed to open the discv5 event stream: {0:?}")]
    EventStream(discv5::Error),
    /// The discv5 event stream ended.
    #[error("The discv5 event stream ended")]
    EventStreamEnded,
    /// The network stopped taking peers to dial.
    #[error("The network stopped")]
    NetworkClosed,
}

/// Runs a discv5 service, passing every valid peer it finds to the network to dial.
struct Actor<N> {
    /// The started discv5 service.
    disc: Discv5,
    /// The L2 chain ID that peers must advertise.
    chain_id: u64,
    /// Peers found so far, synced to disk on an interval.
    store: BootStore,
    /// Events from the discv5 service.
    events: mpsc::Receiver<Event>,
    /// Queries and bans from handles.
    messages: mpsc::Receiver<Message>,
    /// Dials the peers this actor finds.
    network: N,
    /// Random node lookups in flight.
    lookups: FuturesUnordered<BoxFuture<'static, Result<Vec<Enr>, QueryError>>>,
    /// Paces the random node lookups.
    lookup_interval: Interval,
    /// Paces syncing the bootstore to disk.
    store_interval: Interval,
    /// Paces removing a random node from the routing table, when enabled.
    remove_interval: Option<Interval>,
}

impl<N: Network> Actor<N> {
    /// Starts the discv5 service, bootstraps its table, and asks the network to dial the valid
    /// peers in the bootstore.
    async fn start(
        driver: Discv5Driver,
        messages: mpsc::Receiver<Message>,
        network: N,
    ) -> Result<Self, ActorError> {
        let Discv5Driver {
            disc,
            bootstore,
            bootnodes,
            chain_id,
            interval,
            store_interval,
            remove_interval,
        } = driver.init().await.map_err(|err| {
            error!(target: "discovery", ?err, "Failed to start discovery service");
            ActorError::Start(err)
        })?;
        trace!(target: "discovery", "Discv5 Initialized");

        let store = Discv5Driver::bootstrap_peers(bootstore, bootnodes, chain_id, &disc).await;
        info!(target: "discovery", "Discv5 Started with {} ENRs", disc.table_entries_enr().len());

        let actor = Self {
            events: Self::event_stream(&disc).await?,
            disc,
            chain_id,
            store,
            messages,
            network,
            lookups: FuturesUnordered::new(),
            lookup_interval: time::interval(interval),
            store_interval: time::interval(store_interval),
            remove_interval: remove_interval.map(time::interval),
        };
        for enr in actor.store.valid_peers_with_chain_id(chain_id) {
            dial(&actor.network, enr.clone())?;
        }
        Ok(actor)
    }

    /// Asks discv5 for its event stream, retrying a fixed number of times.
    async fn event_stream(disc: &Discv5) -> Result<mpsc::Receiver<Event>, ActorError> {
        let mut attempt = 1;
        loop {
            match disc.event_stream().await {
                Ok(events) => return Ok(events),
                Err(err) if attempt < EVENT_STREAM_ATTEMPTS => {
                    warn!(target: "discovery", ?err, attempt, "Failed to start event stream, retrying");
                    attempt += 1;
                    time::sleep(EVENT_STREAM_RETRY_INTERVAL).await;
                }
                Err(err) => {
                    error!(target: "discovery", ?err, "Failed to start event stream");
                    return Err(ActorError::EventStream(err));
                }
            }
        }
    }

    async fn run(mut self) -> Result<(), ActorError> {
        loop {
            select! {
                // A closed queue only means no handle remains; discovery still finds peers.
                Some(message) = self.messages.recv(), if !self.messages.is_closed() => {
                    self.handle_message(message);
                }
                event = self.events.recv() => {
                    let event = event.ok_or_else(|| {
                        error!(target: "discovery", "The discv5 event stream ended");
                        ActorError::EventStreamEnded
                    })?;
                    self.handle_event(event)?;
                }
                Some(result) = self.lookups.next(), if !self.lookups.is_empty() => {
                    self.handle_lookup(result)?;
                }
                _ = self.lookup_interval.tick() => {
                    let id = NodeId::random();
                    trace!(target: "discovery", "Finding random node: {}", id);
                    metrics::gauge!(Metrics::FIND_NODE_REQUEST, "find_node" => "find_node").increment(1);
                    self.lookups.push(Box::pin(self.disc.find_node(id)));
                }
                _ = self.store_interval.tick() => self.sync_store(),
                _ = tick(self.remove_interval.as_mut()) => self.remove_random_node(),
            }
        }
    }

    fn handle_message(&self, message: Message) {
        // A handle that stopped waiting for its reply is not an error.
        match message {
            Message::LocalEnr(reply) => {
                let _ = reply.send(self.disc.local_enr());
            }
            Message::PeerCount(reply) => {
                let _ = reply.send(self.disc.connected_peers());
            }
            Message::TableEnrs(reply) => {
                let _ = reply.send(self.disc.table_entries_enr());
            }
            Message::TableInfos(reply) => {
                let _ = reply.send(self.disc.table_entries());
            }
            Message::Ban { addrs, duration } => {
                for enr in self.disc.table_entries_enr() {
                    if enr_to_multiaddr(&enr).is_some_and(|addr| addrs.contains(&addr)) {
                        self.disc.ban_node(&enr.node_id(), Some(duration));
                    }
                }
            }
        }
    }

    fn handle_event(&mut self, event: Event) -> Result<(), ActorError> {
        let (enr, kind) = match event {
            Event::Discovered(enr) => (enr, "discovered"),
            Event::SessionEstablished(enr, addr) => {
                debug!(target: "discovery", ?addr, "Session established");
                (enr, "session_established")
            }
            Event::UnverifiableEnr { enr, .. } => (enr, "unverifiable_enr"),
            _ => return Ok(()),
        };
        if !self.is_valid(&enr) {
            return Ok(());
        }
        debug!(target: "discovery", kind, "Valid ENR found, forwarding to the network: {:?}", enr);
        metrics::gauge!(Metrics::DISCOVERY_EVENT, "type" => kind).increment(1);
        self.store.add_enr(enr.clone());
        dial(&self.network, enr)
    }

    fn handle_lookup(&self, result: Result<Vec<Enr>, QueryError>) -> Result<(), ActorError> {
        match result {
            Ok(nodes) => {
                for enr in nodes.into_iter().filter(|enr| self.is_valid(enr)) {
                    dial(&self.network, enr)?;
                }
            }
            Err(err) => info!(target: "discovery", "Failed to find node: {:?}", err),
        }
        Ok(())
    }

    fn sync_store(&mut self) {
        let start = std::time::Instant::now();
        self.store.merge(self.disc.table_entries_enr());
        if let Err(e) = self.store.sync() {
            warn!(target: "discovery", "Failed to sync bootstore: {:?}", e);
        }
        let elapsed = start.elapsed();
        debug!(target: "discovery", "Bootstore ENRs stored in {:?}", elapsed);
        metrics::histogram!(Metrics::ENR_STORE_TIME, "store_time" => "store_time")
            .record(elapsed.as_secs_f64());
        metrics::gauge!(Metrics::DISCOVERY_PEER_COUNT).set(self.disc.connected_peers() as f64);
    }

    fn remove_random_node(&self) {
        let enrs = self.disc.table_entries_enr();
        if enrs.len() > MIN_TABLE_SIZE_FOR_REMOVAL {
            let enr = &enrs[rand::random_range(0..enrs.len())];
            debug!(target: "removal", "Removing random ENR: {:?}", enr);
            self.disc.remove_node(&enr.node_id());
        }
    }

    fn is_valid(&self, enr: &Enr) -> bool {
        EnrValidation::validate(enr, self.chain_id).is_valid()
    }
}

/// Asks the network to dial `enr`, dropping it if the network has no room.
fn dial(network: &impl Network, enr: Enr) -> Result<(), ActorError> {
    match network.dial(enr) {
        Ok(()) => Ok(()),
        Err(TrySendError::Full(enr)) => {
            debug!(target: "discovery", "Network queue full, dropping ENR: {:?}", enr);
            Ok(())
        }
        Err(TrySendError::Closed(_)) => {
            error!(target: "discovery", "The network stopped taking peers");
            Err(ActorError::NetworkClosed)
        }
    }
}

/// Waits for the next tick of `interval`, or forever without one.
async fn tick(interval: Option<&mut Interval>) {
    match interval {
        Some(interval) => {
            interval.tick().await;
        }
        None => std::future::pending().await,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::discovery::HandleError;
    use enr::CombinedKey;
    use kona_disc::{Discv5Builder, LocalNode};
    use kona_peers::{BootNode, BootNodes};
    use std::{
        collections::HashSet,
        net::{IpAddr, Ipv4Addr},
    };
    use tokio::task::JoinHandle;

    /// A chain without built-in bootnodes, so discovery stays off the internet.
    const CHAIN_ID: u64 = 901;

    /// Returns a driver whose service listens on a free loopback port and whose ENR advertises
    /// `tcp_port`, with `bootnodes` to bootstrap from.
    fn driver(tcp_port: u16, bootnodes: Vec<Enr>) -> Discv5Driver {
        let CombinedKey::Secp256k1(key) = CombinedKey::generate_secp256k1() else { unreachable!() };
        let ip = IpAddr::V4(Ipv4Addr::LOCALHOST);
        Discv5Builder::new(
            LocalNode::new(key, ip, tcp_port, 0),
            CHAIN_ID,
            discv5::ConfigBuilder::new(discv5::ListenConfig::from_ip(ip, 0)).build(),
        )
        .with_bootnodes(BootNodes(bootnodes.into_iter().map(BootNode::Enr).collect()))
        .build()
        .unwrap()
    }

    /// The ENR of a node that never runs.
    fn peer() -> Enr {
        driver(30303, vec![]).disc.local_enr()
    }

    /// Runs a discovery actor that bootstraps from `bootnodes` and passes peers to `network`.
    fn spawn(
        bootnodes: Vec<Enr>,
        network: mpsc::Sender<Enr>,
    ) -> (Handle, Enr, JoinHandle<Result<(), ActorError>>) {
        let driver = driver(0, bootnodes);
        let local_enr = driver.disc.local_enr();
        let builder = Builder::new(Capacity::try_from(8).unwrap());
        let handle = builder.handle();
        (handle, local_enr, tokio::spawn(builder.build(driver, network)))
    }

    #[tokio::test]
    async fn dials_bootnodes_and_answers_queries() {
        let bootnode = peer();
        let (network, mut dialed) = mpsc::channel(8);
        let (handle, local_enr, _task) = spawn(vec![bootnode.clone()], network);

        assert_eq!(dialed.recv().await.unwrap(), bootnode);
        assert_eq!(handle.local_enr().await.unwrap(), local_enr);
        assert_eq!(handle.peer_count().await.unwrap(), 0);
        assert_eq!(handle.table_enrs().await.unwrap(), vec![bootnode.clone()]);
        let infos = handle.table_infos().await.unwrap();
        assert_eq!(infos.len(), 1);
        assert_eq!(infos[0].0, bootnode.node_id());
    }

    #[tokio::test]
    async fn bans_the_nodes_at_the_banned_addresses() {
        let (banned, kept) = (peer(), peer());
        let (network, _dialed) = mpsc::channel(8);
        let (handle, _, _task) = spawn(vec![banned.clone(), kept.clone()], network);

        let addrs = HashSet::from([enr_to_multiaddr(&banned).unwrap()]);
        handle.ban(addrs, Duration::from_secs(60)).await.unwrap();
        assert_eq!(handle.table_enrs().await.unwrap(), vec![kept]);
    }

    #[tokio::test]
    async fn drops_peers_the_network_has_no_room_for() {
        let queued = peer();
        let (network, mut dialed) = mpsc::channel(1);
        network.try_send(queued.clone()).unwrap();
        let (handle, _, _task) = spawn(vec![peer()], network);

        // The actor started and dropped its bootnode instead of waiting for room.
        assert_eq!(handle.peer_count().await.unwrap(), 0);
        assert_eq!(dialed.recv().await.unwrap(), queued);
        assert!(dialed.try_recv().is_err());
    }

    #[tokio::test]
    async fn a_stopped_network_is_fatal() {
        let (network, dialed) = mpsc::channel(8);
        drop(dialed);
        let (handle, _, task) = spawn(vec![peer()], network);

        assert!(matches!(task.await.unwrap(), Err(ActorError::NetworkClosed)));
        assert_eq!(handle.peer_count().await, Err(HandleError));
    }

    #[tokio::test]
    async fn keeps_running_without_handles() {
        let (network, _dialed) = mpsc::channel(8);
        let (handle, _, mut task) = spawn(vec![], network);
        handle.peer_count().await.unwrap();
        drop(handle);

        assert!(tokio::time::timeout(Duration::from_millis(100), &mut task).await.is_err());
        task.abort();
    }
}
