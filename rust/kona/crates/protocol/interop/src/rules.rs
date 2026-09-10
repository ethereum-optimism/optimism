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

/// The cycle-detection dependency graph, held as one adjacency list per direction.
///
/// `depends_on` holds the forward edges, from an executing message to the node it depends on.
/// `depended_on_by` holds the same edges reversed. Kosaraju's algorithm needs both directions,
/// and building them together keeps their lengths equal by construction.
#[derive(Debug)]
struct DependencyGraph {
    /// Forward edges: `depends_on[from]` lists the nodes `from` depends on.
    depends_on: Vec<Vec<usize>>,
    /// Reverse edges: `depended_on_by[to]` lists the nodes that depend on `to`.
    depended_on_by: Vec<Vec<usize>>,
}

impl DependencyGraph {
    /// Creates an edgeless graph over `node_count` nodes.
    fn new(node_count: usize) -> Self {
        Self {
            depends_on: vec![Vec::new(); node_count],
            depended_on_by: vec![Vec::new(); node_count],
        }
    }

    /// Records that `from` depends on `to`.
    fn add_edge(&mut self, from: usize, to: usize) {
        self.depends_on[from].push(to);
        self.depended_on_by[to].push(from);
    }

    /// Returns the indices of the nodes that lie on a directed cycle, or an empty vec if the
    /// graph is acyclic.
    ///
    /// Finds strongly connected components with Kosaraju's algorithm: collect the components
    /// over the reverse edges, in decreasing finish order over the forward edges.
    fn cycle_nodes(&self) -> Vec<usize> {
        let finish_order = self.finish_order();

        let mut assigned = vec![false; self.depends_on.len()];
        let mut on_cycle = Vec::new();
        for &start in finish_order.iter().rev() {
            if assigned[start] {
                continue;
            }

            // The component doubles as the queue: every appended node is still unvisited.
            assigned[start] = true;
            let mut component = vec![start];
            let mut next = 0;
            while next < component.len() {
                for &dependent in &self.depended_on_by[component[next]] {
                    if !assigned[dependent] {
                        assigned[dependent] = true;
                        component.push(dependent);
                    }
                }
                next += 1;
            }

            if self.component_has_cycle(&component) {
                on_cycle.extend_from_slice(&component);
            }
        }

        on_cycle
    }

    /// Reports whether a strongly connected component holds a directed cycle. A component of
    /// more than one node always does. A single node holds one only through a self-edge.
    fn component_has_cycle(&self, component: &[usize]) -> bool {
        component.len() > 1 || self.depends_on[component[0]].contains(&component[0])
    }

    /// Returns the nodes in depth-first finish order over the forward edges. The traversal is
    /// iterative, so an adversarial dependency depth cannot exhaust the call stack.
    fn finish_order(&self) -> Vec<usize> {
        let node_count = self.depends_on.len();

        let mut visited = vec![false; node_count];
        let mut finish_order = Vec::with_capacity(node_count);
        let mut stack = Vec::with_capacity(node_count);

        for start in 0..node_count {
            if visited[start] {
                continue;
            }

            visited[start] = true;
            stack.push((start, 0));

            while let Some((node, next_edge)) = stack.pop() {
                if next_edge == self.depends_on[node].len() {
                    finish_order.push(node);
                    continue;
                }

                stack.push((node, next_edge + 1));
                let adjacent = self.depends_on[node][next_edge];
                if !visited[adjacent] {
                    visited[adjacent] = true;
                    stack.push((adjacent, 0));
                }
            }
        }

        finish_order
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

    let mut graph = DependencyGraph::new(nodes.len());
    for chain_indices in chain_nodes.values() {
        for (i, &node_idx) in chain_indices.iter().enumerate() {
            // Intra-chain: depends on previous EM on the same chain.
            if i > 0 {
                graph.add_edge(node_idx, chain_indices[i - 1]);
            }

            // Cross-chain: depends on executingMessageBefore(targetChain, targetLogIdx).
            let target_chain = nodes[node_idx].target_chain_id;
            let target_log_idx = nodes[node_idx].target_log_index;
            if let Some(target_indices) = chain_nodes.get(&target_chain) &&
                let Some(dep_idx) =
                    executing_message_before(&nodes, target_indices, target_log_idx)
            {
                graph.add_edge(node_idx, dep_idx);
            }
        }
    }

    // Collect the unique chain IDs of the nodes on a cycle.
    let mut cycle_chains: Vec<u64> =
        graph.cycle_nodes().into_iter().map(|i| nodes[i].chain_id).collect();
    cycle_chains.sort();
    cycle_chains.dedup();
    cycle_chains
}
