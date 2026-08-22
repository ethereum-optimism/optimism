//! The output of composing a single chain's actor group.

use crate::{
    ChainControllerRequest, ChainControllerRpcRequest, DerivationActorRequest,
    service::spawn::BoxedNodeActor,
};
use alloy_primitives::Address;
use kona_engine::CrossSafePromoter;
use kona_genesis::RollupConfig;
use kona_protocol::BlockInfo;
use kona_rpc::L1WatcherQueries;
use std::sync::Arc;
use tokio::sync::{mpsc, watch};

/// One chain's actors, together with the channel endpoints an L1 watcher needs to drive that
/// chain.
///
/// [`RollupNode::compose`] produces this; what happens next is the host's decision, and that
/// split is the point of the type. A single-chain host ([`RollupNode::start`]) composes one chain,
/// builds an [`L1WatcherActor`] over its [`L1WatcherPorts`], and runs the lot. A multi-chain host
/// composes one chain per configured rollup, concatenates every `actors` list into one set, and
/// gives the ports of *all* chains to a single L1 watcher — so the L1 is followed once per process
/// rather than once per chain, and no second copy of the per-chain wiring exists to drift from
/// this one.
///
/// [`RollupNode::compose`]: crate::RollupNode::compose
/// [`RollupNode::start`]: crate::RollupNode::start
/// [`L1WatcherActor`]: crate::L1WatcherActor
pub struct ComposedChain {
    /// The chain's actors, error-erased and ready to be handed to
    /// [`run_actors`](crate::run_actors).
    ///
    /// This is every actor the chain needs *except* its L1 watcher, which is excluded precisely
    /// because it is the one component a multi-chain host shares between chains.
    pub actors: Vec<BoxedNodeActor>,
    /// The endpoints of this chain that an L1 watcher owns.
    pub l1_watcher_ports: L1WatcherPorts,
    /// The chain's state-mutating request channel, for a host that drives the chain from outside
    /// its own actor set.
    ///
    /// Every actor that needs this already holds a clone, so a single-chain host has no use for
    /// it. An interop host does: its cross-chain verifier is one actor for the whole process and
    /// sends each chain's promotions here.
    pub controller_request_tx: mpsc::Sender<ChainControllerRequest>,
    /// The chain's read-only query channel, answered by the chain controller's RPC peer.
    ///
    /// Handed out for the same reason as [`Self::controller_request_tx`]: the verifier observes
    /// every chain, and reads the local-safe snapshot and output roots through the same queue the
    /// chain's own JSON-RPC server reads them through, rather than through a second view of the
    /// engine state that could disagree with it.
    pub controller_rpc_request_tx: mpsc::Sender<ChainControllerRpcRequest>,
    /// The chain's L1 query channel, answered by whichever L1 watcher was attached to this
    /// chain's [`L1WatcherPorts`].
    ///
    /// Handed out for the same reason as [`Self::controller_rpc_request_tx`]: a multi-chain host
    /// answers questions about the whole set — a supernode's sync status is the per-chain sync
    /// status aggregated — and the L1 half of one chain's sync status is only knowable through
    /// this channel. Reading it here means the host and the chain's own `optimism_syncStatus`
    /// report the same L1 progress rather than two views that can disagree.
    pub l1_query_tx: mpsc::Sender<L1WatcherQueries>,
    /// The capability to promote this chain's cross-safe head, when the chain was composed with
    /// an externally fed one.
    ///
    /// [`None`] for a standalone chain, whose cross-safe head trivially follows local-safe and
    /// which therefore has no promoter to hand out. There is at most one of these per chain, and
    /// it is not [`Clone`], so whoever takes it is by construction that chain's only cross-safe
    /// writer.
    pub cross_safe_promoter: Option<CrossSafePromoter>,
}

impl core::fmt::Debug for ComposedChain {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        // The actors are boxed trait objects and cannot be formatted; their count is the useful
        // part anyway.
        f.debug_struct("ComposedChain")
            .field("actors", &self.actors.len())
            .field("l1_watcher_ports", &self.l1_watcher_ports)
            .finish()
    }
}

/// The channel endpoints an [`L1WatcherActor`](crate::L1WatcherActor) needs in order to serve one
/// chain.
///
/// Composition creates the channels and keeps the ends the chain's own actors read from, so a host
/// can attach a watcher to a chain without reaching back into the rest of its wiring.
#[derive(Debug)]
pub struct L1WatcherPorts {
    /// The chain's rollup configuration, which the watcher needs for the chain's
    /// `SystemConfig` address and activation times.
    pub rollup_config: Arc<RollupConfig>,
    /// Where the watcher sends L1 head and finalized updates for this chain.
    pub derivation_actor_request_tx: mpsc::Sender<DerivationActorRequest>,
    /// Where the watcher sends unsafe-block-signer changes read from this chain's `SystemConfig`.
    pub unsafe_signer_tx: mpsc::Sender<Address>,
    /// The RPC queries about the L1 addressed to this chain, which the watcher answers.
    pub l1_query_rx: mpsc::Receiver<L1WatcherQueries>,
    /// Where the watcher publishes the observed L1 head; this chain's sequencer reads it through
    /// its origin selector.
    pub l1_head_updates_tx: watch::Sender<Option<BlockInfo>>,
}
