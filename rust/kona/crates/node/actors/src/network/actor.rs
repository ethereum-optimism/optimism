//! Network construction and execution.

use super::{
    Discovery, Engine, Handle,
    handle::{Applied, Message},
};
use crate::signer;
use discv5::Enr;
use kona_gossip::{
    ConnectionGate, ConnectionGater, GossipDriver, GossipState, PEER_SCORE_INSPECT_FREQUENCY,
};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
use std::{collections::HashSet, future::Future, sync::Arc};
use thiserror::Error;
use tokio::{
    select,
    sync::{
        mpsc::{self, error::TrySendError},
        watch,
    },
    time::{self, Interval},
};

/// How many peers discovery can queue for the network to dial. Discovery finds peers in bursts and
/// drops those that find the queue full: at startup it dials its stored peers, newest first, of
/// which the default bootnodes alone number 25, and each lookup returns up to 16. An entry is a
/// few hundred bytes.
const DIAL_QUEUE_CAPACITY: usize = 64;

/// Constructs the handle and task.
#[derive(Debug)]
pub struct Builder {
    handle: Handle,
    gossip: GossipDriver<ConnectionGater>,
    state: watch::Sender<Arc<GossipState>>,
    messages: mpsc::Receiver<Message>,
    peers: mpsc::Receiver<Enr>,
    payloads: mpsc::Receiver<signer::Payload>,
}

impl Builder {
    /// Creates the builder around a gossip driver whose swarm is listening, as
    /// [`NetworkConfig::start`](super::NetworkConfig::start) leaves it, and publishes its state.
    ///
    /// The message queue holds one change, which waits to be applied anyway, and the queue of
    /// signed payloads to publish holds one payload, so the signer waits while the network falls
    /// a payload behind.
    pub fn new(gossip: GossipDriver<ConnectionGater>) -> Self {
        let (state, state_rx) = watch::channel(Arc::new(gossip.snapshot()));
        let (messages_tx, messages) = mpsc::channel(1);
        let (peers_tx, peers) = mpsc::channel(DIAL_QUEUE_CAPACITY);
        let (payloads_tx, payloads) = mpsc::channel(1);
        Self {
            handle: Handle::new(state_rx, messages_tx, peers_tx, payloads_tx),
            gossip,
            state,
            messages,
            peers,
            payloads,
        }
    }

    /// Returns a handle that can be wired into other components before the actor starts.
    pub fn handle(&self) -> Handle {
        self.handle.clone()
    }

    /// Supplies dependencies and produces the actor's lifetime future without spawning it.
    ///
    /// Dropping the future stops the actor and the swarm.
    pub fn build<Engine_: Engine + 'static, Discovery_: Discovery + 'static>(
        self,
        engine: Engine_,
        discovery: Discovery_,
    ) -> impl Future<Output = Result<(), ActorError>> + Send + 'static {
        let Self { handle: _, gossip, state, messages, peers, payloads } = self;
        async move {
            Actor {
                gossip,
                state,
                messages,
                peers,
                payloads,
                engine,
                discovery,
                peer_score_inspector: time::interval(*PEER_SCORE_INSPECT_FREQUENCY),
            }
            .run()
            .await
        }
    }
}

/// A fatal error from the network actor.
#[derive(Debug, Error)]
pub enum ActorError {
    /// The engine stopped taking unsafe blocks.
    #[error("The engine stopped")]
    EngineClosed,
    /// The gossip swarm's event stream ended.
    #[error("The gossip swarm stream ended")]
    SwarmEnded,
}

/// Runs the libp2p gossip swarm: publishes signed payloads, passes received blocks to the engine,
/// dials the peers discovery finds, and bans peers below the score threshold.
#[derive(Debug)]
struct Actor<Engine_, Discovery_> {
    /// The gossip driver, whose swarm is listening.
    gossip: GossipDriver<ConnectionGater>,
    /// Publishes the gossip state to handle readers.
    state: watch::Sender<Arc<GossipState>>,
    /// Changes from handles.
    messages: mpsc::Receiver<Message>,
    /// Peers to dial, from discovery.
    peers: mpsc::Receiver<Enr>,
    /// Signed payloads to publish, from the signer.
    payloads: mpsc::Receiver<signer::Payload>,
    /// Inserts the unsafe blocks received over gossip.
    engine: Engine_,
    /// Bans the addresses of disconnected peers in discovery.
    discovery: Discovery_,
    /// Paces the peer score checks that ban peers below the threshold.
    peer_score_inspector: Interval,
}

impl<Engine_: Engine, Discovery_: Discovery> Actor<Engine_, Discovery_> {
    async fn run(mut self) -> Result<(), ActorError> {
        loop {
            let applied = select! {
                Some(signed) = self.payloads.recv(), if !self.payloads.is_closed() => {
                    self.publish(signed);
                    None
                }
                event = self.gossip.next() => {
                    let event = event.ok_or_else(|| {
                        error!(target: "node::p2p", "The gossip swarm stream has ended");
                        ActorError::SwarmEnded
                    })?;
                    if let Some(block) = self.gossip.handle_event(event) {
                        forward(&self.engine, block)?;
                    }
                    None
                }
                // A closed queue only means no handle remains.
                Some(enr) = self.peers.recv(), if !self.peers.is_closed() => {
                    self.gossip.dial(enr);
                    None
                }
                _ = self.peer_score_inspector.tick(), if self.gossip.peer_monitoring.is_some() => {
                    self.ban_low_scoring_peers();
                    None
                }
                Some(message) = self.messages.recv(), if !self.messages.is_closed() => {
                    Some(self.handle_message(message))
                }
            };
            self.state.send_replace(Arc::new(self.gossip.snapshot()));
            // A handle that stopped waiting for its change is not an error.
            if let Some(applied) = applied {
                let _ = applied.send(());
            }
        }
    }

    fn publish(&mut self, signed: signer::Payload) {
        // Published even if the signer rotated since signing: peers then reject the block, which
        // is harmless.
        let timestamp = signed.payload.timestamp();
        let selector = |handler: &kona_gossip::BlockHandler| handler.topic(timestamp);
        match self.gossip.publish(selector, signed.payload, signed.signature) {
            Ok(id) => info!("Published unsafe payload | {:?}", id),
            Err(e) => warn!("Failed to publish unsafe payload: {:?}", e),
        }
    }

    /// Makes a change to the gossip driver, and returns the sender to reply on once the state
    /// that follows it is published.
    fn handle_message(&mut self, message: Message) -> Applied {
        let gossip = &mut self.gossip;
        match message {
            Message::BlockPeer(id, applied) => {
                gossip.connection_gate.block_peer(&id);
                gossip.swarm.behaviour_mut().gossipsub.blacklist_peer(&id);
                applied
            }
            Message::UnblockPeer(id, applied) => {
                gossip.connection_gate.unblock_peer(&id);
                gossip.swarm.behaviour_mut().gossipsub.remove_blacklisted_peer(&id);
                applied
            }
            Message::BlockAddr(address, applied) => {
                gossip.connection_gate.block_addr(address);
                applied
            }
            Message::UnblockAddr(address, applied) => {
                gossip.connection_gate.unblock_addr(address);
                applied
            }
            Message::BlockSubnet(subnet, applied) => {
                gossip.connection_gate.block_subnet(subnet);
                applied
            }
            Message::UnblockSubnet(subnet, applied) => {
                gossip.connection_gate.unblock_subnet(subnet);
                applied
            }
            Message::ConnectPeer(address, applied) => {
                gossip.dial_multiaddr(address);
                applied
            }
            Message::ProtectPeer(id, applied) => {
                gossip.connection_gate.protect_peer(id);
                applied
            }
            Message::UnprotectPeer(id, applied) => {
                gossip.connection_gate.unprotect_peer(id);
                applied
            }
            Message::DisconnectPeer(id, applied) => {
                if let Err(e) = gossip.swarm.disconnect_peer_id(id) {
                    warn!(target: "p2p::rpc", "Failed to disconnect peer {}: {:?}", id, e);
                } else {
                    info!(target: "p2p::rpc", "Disconnected peer {}", id);
                    if let Some(start_time) = gossip.peer_connection_start.remove(&id) {
                        metrics::histogram!(
                            kona_gossip::Metrics::GOSSIP_PEER_CONNECTION_DURATION_SECONDS
                        )
                        .record(start_time.elapsed().as_secs_f64());
                    }
                }
                applied
            }
        }
    }

    /// Disconnects the peers whose score is below the ban threshold, and asks discovery to ban
    /// their addresses.
    fn ban_low_scoring_peers(&mut self) {
        let Some(ban_peers) = self.gossip.peer_monitoring.as_ref() else {
            return;
        };

        let peers_to_remove = self
            .gossip
            .swarm
            .connected_peers()
            .filter_map(|peer_id| {
                // A peer without a score counts as zero.
                let score =
                    self.gossip.swarm.behaviour().gossipsub.peer_score(peer_id).unwrap_or_default();
                metrics::histogram!(kona_gossip::Metrics::PEER_SCORES).record(score);
                (score < ban_peers.ban_threshold).then_some(*peer_id)
            })
            .collect::<Vec<_>>();

        let addrs_to_ban = peers_to_remove
            .into_iter()
            .filter_map(|peer_to_remove| {
                if self.gossip.swarm.disconnect_peer_id(peer_to_remove).is_err() {
                    warn!(peer = ?peer_to_remove, "Trying to disconnect a non-existing peer from the gossip driver.");
                }

                if let Some(start_time) = self.gossip.peer_connection_start.remove(&peer_to_remove)
                {
                    metrics::histogram!(
                        kona_gossip::Metrics::GOSSIP_PEER_CONNECTION_DURATION_SECONDS
                    )
                    .record(start_time.elapsed().as_secs_f64());
                }

                let info = self.gossip.peerstore.remove(&peer_to_remove)?;
                self.gossip.connection_gate.remove_dial(&peer_to_remove);
                let score = self
                    .gossip
                    .swarm
                    .behaviour()
                    .gossipsub
                    .peer_score(&peer_to_remove)
                    .unwrap_or_default();
                metrics::gauge!(kona_gossip::Metrics::BANNED_PEERS, "peer_id" => peer_to_remove.to_string(), "score" => score.to_string()).increment(1);
                Some(info.listen_addrs)
            })
            .flatten()
            .collect::<HashSet<_>>();
        if addrs_to_ban.is_empty() {
            return;
        }

        // The peers are already disconnected: a ban only keeps discovery from finding them again,
        // so it is skipped rather than waited for while discovery is busy.
        match self.discovery.try_ban(addrs_to_ban, ban_peers.ban_duration) {
            Ok(()) => {}
            Err(TrySendError::Full(())) => {
                warn!("Discovery is busy: not banning the addresses of low-scoring peers")
            }
            Err(TrySendError::Closed(())) => {
                warn!("Failed to ask discovery to ban the addresses of low-scoring peers")
            }
        }
    }
}

/// Passes an unsafe block received over gossip to the engine.
///
/// The block is dropped while the engine's queue is full, so that a backed-up engine does not
/// stop the network from serving its peers. The gap it leaves is filled later:
///
/// - The engine inserts a later block whose parent it lacks: the execution layer answers that it is
///   syncing, and the forkchoice update that follows has it sync the gap from its own peers. An
///   execution layer without peers cannot, and the engine resets when derivation next needs a
///   missing block.
/// - An execution layer that accepts the later block without syncing leaves the gap until
///   derivation reaches it.
/// - No later block follows a dropped tip, so it is not inserted until derivation reaches it.
fn forward(engine: &impl Engine, block: OpExecutionPayloadEnvelope) -> Result<(), ActorError> {
    let (number, hash) = (block.block_number(), block.block_hash());
    match engine.try_insert_unsafe_block(block) {
        Ok(()) => Ok(()),
        Err(TrySendError::Full(())) => {
            debug!(target: "network", number, %hash, "Dropping an unsafe block: the engine is backed up");
            metrics::counter!(crate::Metrics::NETWORK_UNSAFE_BLOCKS_DROPPED).increment(1);
            Ok(())
        }
        Err(TrySendError::Closed(())) => {
            warn!(target: "network", "Failed to forward unsafe block to engine");
            Err(ActorError::EngineClosed)
        }
    }
}

#[cfg(test)]
pub(super) mod tests {
    use super::*;
    use alloy_primitives::Address;
    use kona_genesis::RollupConfig;
    use libp2p::{Multiaddr, identity::Keypair};
    use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelope;
    use std::time::Duration;

    /// A gossip driver whose swarm is not listening.
    fn gossip() -> GossipDriver<ConnectionGater> {
        GossipDriver::<ConnectionGater>::builder(
            RollupConfig::default(),
            Address::ZERO,
            "/ip4/127.0.0.1/tcp/0".parse().unwrap(),
            Keypair::generate_secp256k1(),
        )
        .build()
        .unwrap()
    }

    /// The state of a gossip driver whose swarm is not listening.
    pub(in crate::network) fn state() -> GossipState {
        gossip().snapshot()
    }

    #[derive(Debug)]
    struct UnusedEngine;

    impl Engine for UnusedEngine {
        fn try_insert_unsafe_block(
            &self,
            _block: OpExecutionPayloadEnvelope,
        ) -> Result<(), TrySendError<()>> {
            panic!("no block is gossiped")
        }
    }

    #[derive(Debug)]
    struct UnusedDiscovery;

    impl Discovery for UnusedDiscovery {
        fn try_ban(
            &self,
            _addrs: HashSet<Multiaddr>,
            _duration: Duration,
        ) -> Result<(), TrySendError<()>> {
            panic!("peer monitoring is off")
        }
    }

    #[tokio::test]
    async fn publishes_state_from_construction_until_the_builder_or_actor_stops() {
        let builder = Builder::new(gossip());
        let state = builder.handle().state();
        assert!(state.borrow().connected_peers.is_empty());
        assert!(state.has_changed().is_ok());

        drop(builder);
        assert!(state.has_changed().is_err());
    }

    #[tokio::test]
    async fn a_change_returns_once_the_state_that_follows_it_is_published() {
        let builder = Builder::new(gossip());
        let handle = builder.handle();
        let task = tokio::spawn(builder.build(UnusedEngine, UnusedDiscovery));
        let state = handle.state();
        let id = libp2p::PeerId::random();

        handle.block_peer(id).await.unwrap();
        assert_eq!(state.borrow().blocked_peers, vec![id]);
        handle.unblock_peer(id).await.unwrap();
        assert!(state.borrow().blocked_peers.is_empty());

        let subnet = "192.0.2.0/24".parse().unwrap();
        handle.block_subnet(subnet).await.unwrap();
        assert_eq!(state.borrow().blocked_subnets, vec![subnet]);
        handle.protect_peer(id).await.unwrap();
        assert_eq!(state.borrow().protected_peers, vec![id]);

        task.abort();
        assert!(task.await.unwrap_err().is_cancelled());
        assert!(state.has_changed().is_err());
        assert_eq!(handle.unprotect_peer(id).await, Err(super::super::HandleError));
    }

    #[test]
    fn drops_blocks_while_the_engine_is_backed_up_and_stops_once_it_closes() {
        let (engine, mut queued) = mpsc::channel(1);
        forward(&engine, crate::test_payload(1)).unwrap();
        // The queue is full: the block is dropped, and the network carries on.
        forward(&engine, crate::test_payload(2)).unwrap();
        assert_eq!(queued.try_recv().unwrap(), crate::test_payload(1));
        assert!(queued.try_recv().is_err());

        drop(queued);
        assert!(matches!(forward(&engine, crate::test_payload(3)), Err(ActorError::EngineClosed)));
    }
}
