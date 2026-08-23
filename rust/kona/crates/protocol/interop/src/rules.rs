//! The interop message-validity rules.
//!
//! This module is the single home of the rules that decide whether an interop message is valid:
//! the activation invariant, timestamp ordering, the message expiry window and the same-timestamp
//! cycle check. Both the fault-proof consolidation path ([`MessageGraph`](crate::MessageGraph))
//! and the node-side verifier apply these rules via [`MessageRules`] so that the two never
//! drift — a divergence here is a prestate divergence.
//!
//! Rules reference: <https://specs.optimism.io/interop/messaging.html#invalid-messages>

use crate::{errors::MessageGraphError, message::EnrichedExecutingMessage};
use alloc::{collections::BTreeMap, string::ToString, vec, vec::Vec};
use core::fmt::Debug;
use kona_genesis::RollupConfig;
use kona_registry::{HashMap, ROLLUP_CONFIGS};
use tracing::warn;

/// The interop message-validity rules, bound to the configuration they need.
///
/// Constructed once per validation pass — by [`MessageGraph`](crate::MessageGraph) on the
/// fault-proof consolidation path, and by the node-side verifier — so that call sites pass only
/// the per-message timestamps. Grouping the rules onto a single type keeps them discoverable and
/// gives both callers one place to reach for, so the two can never drift.
///
/// Only [`Self::rollup_config_for`] and [`Self::check_message_expiry`] read from `self`; the
/// remaining rules operate purely on their arguments and are associated functions.
#[derive(Debug, Clone, Copy)]
pub struct MessageRules<'a> {
    /// Backup rollup configs for each chain, consulted when the superchain registry has no entry
    /// for the chain.
    rollup_configs: &'a HashMap<u64, RollupConfig>,
    /// The message expiry window (in seconds) for validating initiating message timestamps.
    message_expiry_window: u64,
}

impl<'a> MessageRules<'a> {
    /// Creates a new [`MessageRules`] from the backup rollup configs and the message expiry
    /// window.
    pub const fn new(
        rollup_configs: &'a HashMap<u64, RollupConfig>,
        message_expiry_window: u64,
    ) -> Self {
        Self { rollup_configs, message_expiry_window }
    }

    /// Resolves the [`RollupConfig`] for `chain_id`, preferring the superchain registry and
    /// falling back to the locally supplied configs.
    pub fn rollup_config_for<E: Debug>(
        &self,
        chain_id: u64,
    ) -> Result<&'a RollupConfig, MessageGraphError<E>> {
        ROLLUP_CONFIGS
            .get(&chain_id)
            .or_else(|| self.rollup_configs.get(&chain_id))
            .ok_or(MessageGraphError::MissingRollupConfig(chain_id))
    }

    /// Returns `true` if interop has been active on `config`'s chain for at least one full block
    /// at `timestamp`, i.e. interop is active and `timestamp` is not the activation block.
    fn interop_active_for_full_block(config: &RollupConfig, timestamp: u64) -> bool {
        config.is_interop_active(timestamp) && !config.is_first_interop_block(timestamp)
    }

    /// Activation invariant, executing side: interop must have been active on the executing chain
    /// for at least one full block at `executing_timestamp`.
    pub fn check_executing_activation<E: Debug>(
        executing_config: &RollupConfig,
        executing_timestamp: u64,
    ) -> Result<(), MessageGraphError<E>> {
        Self::interop_active_for_full_block(executing_config, executing_timestamp)
            .then_some(())
            .ok_or(MessageGraphError::ExecutedTooEarly {
                activation_time: executing_config.hardforks.lagoon_time.unwrap_or_default(),
                executing_message_time: executing_timestamp,
            })
    }

    /// Activation invariant, initiating side: interop must have been active on the initiating
    /// chain for at least one full block at `initiating_timestamp`.
    pub fn check_initiating_activation<E: Debug>(
        initiating_config: &RollupConfig,
        initiating_timestamp: u64,
    ) -> Result<(), MessageGraphError<E>> {
        Self::interop_active_for_full_block(initiating_config, initiating_timestamp)
            .then_some(())
            .ok_or(MessageGraphError::InitiatedTooEarly {
                activation_time: initiating_config.hardforks.lagoon_time.unwrap_or_default(),
                initiating_message_time: initiating_timestamp,
            })
    }

    /// Timestamp ordering invariant: the initiating message must not be in the future relative to
    /// the executing message.
    pub fn check_message_ordering<E: Debug>(
        initiating_timestamp: u64,
        executing_timestamp: u64,
    ) -> Result<(), MessageGraphError<E>> {
        (initiating_timestamp <= executing_timestamp).then_some(()).ok_or(
            MessageGraphError::MessageInFuture {
                max: executing_timestamp,
                actual: initiating_timestamp,
            },
        )
    }

    /// Message expiry invariant: the initiating message must be no more than the configured
    /// message expiry window (in seconds) in the past, relative to the executing message.
    ///
    /// Callers must run [`Self::check_message_ordering`] first: the subtraction relies on its
    /// guarantee that `initiating_timestamp <= executing_timestamp`.
    pub fn check_message_expiry<E: Debug>(
        &self,
        initiating_timestamp: u64,
        executing_timestamp: u64,
    ) -> Result<(), MessageGraphError<E>> {
        (executing_timestamp - initiating_timestamp <= self.message_expiry_window)
            .then_some(())
            .ok_or(MessageGraphError::MessageExpired { initiating_timestamp, executing_timestamp })
    }

    /// Same-timestamp cycle check over a whole set of executing messages: runs the crate-private
    /// `detect_cycles` helper once per distinct executing timestamp present in `messages`.
    pub fn check_no_cycles<E: Debug>(
        messages: &[EnrichedExecutingMessage],
    ) -> Result<(), MessageGraphError<E>> {
        // Collect distinct timestamps present in the message set.
        let mut timestamps: Vec<u64> = messages.iter().map(|m| m.executing_timestamp).collect();
        timestamps.sort_unstable();
        timestamps.dedup();

        for ts in timestamps {
            let cycle_chains = detect_cycles(messages, ts);
            if !cycle_chains.is_empty() {
                warn!(
                    target: "message_graph",
                    cycle_chains = %cycle_chains
                        .iter()
                        .map(ToString::to_string)
                        .collect::<Vec<_>>()
                        .join(", "),
                    timestamp = ts,
                    "Cyclic dependency detected among same-timestamp executing messages",
                );
                return Err(MessageGraphError::CyclicDependency { chain_ids: cycle_chains });
            }
        }

        Ok(())
    }
}

/// Static graph node representing an executing message in the cycle detection dependency graph.
#[derive(Debug)]
struct GraphNode {
    /// The chain ID this executing message belongs to.
    chain_id: u64,
    /// The log index of the executing message within its block.
    log_index: u32,
    /// The chain ID of the initiating message this EM references.
    target_chain_id: u64,
    /// The log index of the initiating message this EM references.
    target_log_index: u32,
}

/// Finds the index of the latest node in `chain_node_indices` with `log_index <= target_log_idx`.
/// `chain_node_indices` must be sorted by `log_index` ascending.
fn executing_message_before(
    nodes: &[GraphNode],
    chain_node_indices: &[usize],
    target_log_idx: u32,
) -> Option<usize> {
    // partition_point returns the first index where the predicate is false, i.e. the first
    // node with log_index > target. So pp - 1 is the last node with log_index <= target.
    // If pp == 0, every node is past the target and there's no match.
    let pp = chain_node_indices.partition_point(|&i| nodes[i].log_index <= target_log_idx);
    (pp > 0).then(|| chain_node_indices[pp - 1])
}

/// Runs Kahn's topological sort algorithm to detect cycles.
///
/// Operates on algorithm state (parallel vecs) separately from the immutable graph nodes.
/// Returns the indices of nodes participating in cycles, or an empty vec if acyclic.
fn check_cycles(depends_on: &[Vec<usize>], depended_on_by: &mut [Vec<usize>]) -> Vec<usize> {
    let n = depends_on.len();
    if n == 0 {
        return vec![];
    }

    let mut resolved = vec![false; n];

    loop {
        // Find nodes with no depended_on_by and mark them resolved.
        let mut remove_set = Vec::new();
        for (i, deps) in depended_on_by.iter().enumerate() {
            if !resolved[i] && deps.is_empty() {
                resolved[i] = true;
                remove_set.push(i);
            }
        }

        if remove_set.is_empty() {
            // No progress, so we collect unresolved nodes (cycle participants).
            return (0..n).filter(|&i| !resolved[i]).collect();
        }

        // Remove resolved nodes from depended_on_by of their dependencies.
        for &removed_idx in &remove_set {
            for &dep_idx in &depends_on[removed_idx] {
                depended_on_by[dep_idx].retain(|&x| x != removed_idx);
            }
        }
    }
}

/// Builds a dependency graph from executing messages and checks for cycles.
/// Returns the chain IDs of cycle participants, or an empty vec if acyclic.
///
/// Matches the semantics of op-supernode's `buildCycleGraph`:
/// - Only executing messages whose *executing* block timestamp *and* referenced *initiating*
///   message timestamp both equal `timestamp` are included as nodes. An EM that references a
///   historical initiating message is a dependency on finalized past state, not a concurrent
///   cross-chain dependency, and must not participate in the same-timestamp cycle graph.
/// - Intra-chain edges: each EM depends on the previous EM on the same chain.
/// - Cross-chain edges: each EM depends on `executingMessageBefore(targetChain, targetLogIdx)`.
pub(crate) fn detect_cycles(messages: &[EnrichedExecutingMessage], timestamp: u64) -> Vec<u64> {
    // Filter to same-timestamp messages and create nodes.
    let mut nodes = Vec::new();
    // BTreeMap for deterministic iteration order.
    let mut chain_nodes: BTreeMap<u64, Vec<usize>> = BTreeMap::new();

    for msg in messages {
        // Two filters, mirroring op-supernode (`verifyCycleMessages` + `buildCycleGraph`):
        //   1. The EM's executing block must be at `timestamp`.
        //   2. The EM's referenced initiating message must also be at `timestamp`.
        // An EM that passes (1) but fails (2) references historical state and must not be
        // admitted into the same-timestamp cycle graph.
        if msg.executing_timestamp != timestamp ||
            msg.inner.identifier.timestamp.saturating_to::<u64>() != timestamp
        {
            continue;
        }

        let initiating_chain_id: u64 = msg.inner.identifier.chainId.saturating_to();
        let initiating_log_index: u32 = msg.inner.identifier.logIndex.saturating_to();

        let idx = nodes.len();
        nodes.push(GraphNode {
            chain_id: msg.executing_chain_id,
            log_index: msg.executing_log_index,
            target_chain_id: initiating_chain_id,
            target_log_index: initiating_log_index,
        });
        chain_nodes.entry(msg.executing_chain_id).or_default().push(idx);
    }

    if nodes.is_empty() {
        return vec![];
    }

    // Sort each chain's node indices by log_index.
    for indices in chain_nodes.values_mut() {
        indices.sort_by_key(|&idx| nodes[idx].log_index);
    }

    // Build algorithm state: parallel vecs for depends_on / depended_on_by.
    let mut depends_on: Vec<Vec<usize>> = vec![Vec::new(); nodes.len()];
    let mut depended_on_by: Vec<Vec<usize>> = vec![Vec::new(); nodes.len()];

    // Add edges.
    for chain_indices in chain_nodes.values() {
        for (i, &node_idx) in chain_indices.iter().enumerate() {
            // Intra-chain: depends on previous EM on the same chain.
            if i > 0 {
                let prev_idx = chain_indices[i - 1];
                depends_on[node_idx].push(prev_idx);
                depended_on_by[prev_idx].push(node_idx);
            }

            // Cross-chain: depends on executingMessageBefore(targetChain, targetLogIdx).
            let target_chain = nodes[node_idx].target_chain_id;
            let target_log_idx = nodes[node_idx].target_log_index;
            if let Some(target_indices) = chain_nodes.get(&target_chain) &&
                let Some(dep_idx) =
                    executing_message_before(&nodes, target_indices, target_log_idx)
            {
                depends_on[node_idx].push(dep_idx);
                depended_on_by[dep_idx].push(node_idx);
            }
        }
    }

    // Run Kahn's algorithm.
    let cycle_indices = check_cycles(&depends_on, &mut depended_on_by);
    if cycle_indices.is_empty() {
        return vec![];
    }

    // Collect unique chain IDs of cycle participants.
    let mut cycle_chains: Vec<u64> = cycle_indices.iter().map(|&i| nodes[i].chain_id).collect();
    cycle_chains.sort();
    cycle_chains.dedup();
    cycle_chains
}
