//! Interop [`MessageGraph`].

use crate::{
    RawMessagePayload,
    errors::{MessageGraphError, MessageGraphResult},
    message::{EnrichedExecutingMessage, parse_log_to_executing_message},
    traits::InteropProvider,
};
use alloc::{collections::BTreeMap, string::ToString, vec, vec::Vec};
use alloy_consensus::{Header, Sealed};
use alloy_primitives::keccak256;
use kona_genesis::{DependencySet, RollupConfig};
use kona_registry::{HashMap, ROLLUP_CONFIGS};
use tracing::{info, warn};

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
fn detect_cycles(messages: &[EnrichedExecutingMessage], timestamp: u64) -> Vec<u64> {
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

/// The [`MessageGraph`] represents a set of blocks — possibly at different timestamps, one per
/// chain — and the interop dependencies between them.
///
/// This structure is used to determine whether or not any interop messages are invalid within the
/// set of blocks within the graph. An "invalid message" is one that was relayed from one chain to
/// another, but the original [`MessageIdentifier`] is not present within the graph or from a
/// dependency referenced via the [`InteropProvider`] (or otherwise is invalid, such as being older
/// than the message expiry window).
///
/// Message validity rules: <https://specs.optimism.io/interop/messaging.html#invalid-messages>
///
/// [`MessageIdentifier`]: crate::MessageIdentifier
#[derive(Debug)]
pub struct MessageGraph<'a, P> {
    /// The edges within the graph.
    ///
    /// These are derived from the transactions within the blocks.
    messages: Vec<EnrichedExecutingMessage>,
    /// The data provider for the graph. Required for fetching headers, receipts and remote
    /// messages within history during resolution.
    provider: &'a P,
    /// Backup rollup configs for each chain.
    rollup_configs: &'a HashMap<u64, RollupConfig>,
    /// The dependency set for the cluster being validated.
    dependency_set: &'a DependencySet,
    /// The message expiry window (in seconds) for validating initiating message timestamps.
    message_expiry_window: u64,
}

impl<'a, P> MessageGraph<'a, P>
where
    P: InteropProvider,
{
    /// Derives the edges from the blocks within the graph by scanning all receipts within the
    /// blocks and searching for [`ExecutingMessage`]s.
    ///
    /// [`ExecutingMessage`]: crate::ExecutingMessage
    pub async fn derive(
        blocks: &HashMap<u64, Sealed<Header>>,
        provider: &'a P,
        rollup_configs: &'a HashMap<u64, RollupConfig>,
        dependency_set: &'a DependencySet,
        message_expiry_window: u64,
    ) -> MessageGraphResult<Self, P> {
        info!(
            target: "message_graph",
            num_chains = blocks.len(),
            "Deriving message graph",
        );

        let mut messages = Vec::with_capacity(blocks.len());
        for (chain_id, header) in blocks {
            let receipts = provider.receipts_by_hash(*chain_id, header.hash()).await?;

            // Track the global log index across all receipts in the block so we can
            // record each executing message's position, needed for cycle detection.
            let mut global_log_index: u32 = 0;
            for receipt in receipts.as_slice() {
                for log in receipt.logs() {
                    if let Some(exec_msg) = parse_log_to_executing_message(log) {
                        messages.push(EnrichedExecutingMessage::new(
                            exec_msg,
                            *chain_id,
                            header.timestamp,
                            global_log_index,
                        ));
                    }
                    global_log_index += 1;
                }
            }
        }

        info!(
            target: "message_graph",
            num_chains = blocks.len(),
            num_messages = messages.len(),
            "Derived message graph successfully",
        );
        Ok(Self { messages, provider, rollup_configs, dependency_set, message_expiry_window })
    }

    /// Checks the validity of all messages within the graph.
    ///
    /// First, detects cyclic dependencies among same-timestamp executing messages using Kahn's
    /// topological sort. If a cycle is found, returns [`MessageGraphError::CyclicDependency`]
    /// with the chain IDs of cycle participants. Then, validates each message independently.
    ///
    /// _Note_: This function does not account for cascading dependency failures. When
    /// [`MessageGraphError::InvalidMessages`] or [`MessageGraphError::CyclicDependency`] is
    /// returned by this function, the consumer must re-execute the bad blocks with deposit
    /// transactions only per the [interop derivation rules][int-block-replacement]. Once the bad
    /// blocks have been replaced, a new [`MessageGraph`] should be constructed and resolution
    /// should be re-attempted. This process should repeat recursively until no invalid
    /// dependencies remain, with the terminal case being all blocks reduced to deposits-only.
    ///
    /// [int-block-replacement]: https://specs.optimism.io/interop/derivation.html#replacing-invalid-blocks
    pub async fn resolve(self) -> MessageGraphResult<(), P> {
        info!(
            target: "message_graph",
            "Checking the message graph for invalid messages"
        );

        // Check for cyclic dependencies among same-timestamp executing messages before
        // validating individual messages. Cycles are a structural property of the graph
        // that cannot be detected by per-message validation.
        if !self.messages.is_empty() {
            // Collect distinct timestamps present in the message set.
            let mut timestamps: Vec<u64> =
                self.messages.iter().map(|m| m.executing_timestamp).collect();
            timestamps.sort_unstable();
            timestamps.dedup();

            for ts in timestamps {
                let cycle_chains = detect_cycles(&self.messages, ts);
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
        }

        // Create a new vector to store invalid edges
        let mut invalid_messages = HashMap::default();

        // Prune all valid messages, collecting errors for any chain whose block contains an invalid
        // message. Errors are de-duplicated by chain ID in a map, since a single invalid
        // message is cause for invalidating a block.
        for message in &self.messages {
            if let Err(e) = self.check_single_dependency(message).await {
                warn!(
                    target: "message_graph",
                    executing_chain_id = message.executing_chain_id,
                    message_hash = ?message.inner.payloadHash,
                    err = %e,
                    "Invalid ExecutingMessage found",
                );
                invalid_messages.insert(message.executing_chain_id, e);
            }
        }

        info!(
            target: "message_graph",
            num_invalid_messages = invalid_messages.len(),
            "Successfully reduced the message graph",
        );

        // Check if the graph is now empty. If not, there are invalid messages.
        if !invalid_messages.is_empty() {
            warn!(
                target: "message_graph",
                bad_chain_ids = %invalid_messages
                    .keys()
                    .map(ToString::to_string)
                    .collect::<Vec<_>>()
                    .join(", "),
                "Failed to reduce the message graph entirely",
            );

            // Return an error with the chain IDs of the blocks containing invalid messages.
            return Err(MessageGraphError::InvalidMessages(invalid_messages));
        }

        Ok(())
    }

    /// Checks the dependency of a single [`EnrichedExecutingMessage`]. If the message's
    /// dependencies are unavailable, the message is considered invalid and an [`Err`] is
    /// returned.
    async fn check_single_dependency(
        &self,
        message: &EnrichedExecutingMessage,
    ) -> MessageGraphResult<(), P> {
        let initiating_chain_id = message.inner.identifier.chainId.saturating_to();
        let initiating_timestamp = message.inner.identifier.timestamp.saturating_to::<u64>();

        if !self.dependency_set.dependencies.contains_key(&message.executing_chain_id) {
            return Err(MessageGraphError::ChainNotInDependencySet(message.executing_chain_id));
        }
        if !self.dependency_set.dependencies.contains_key(&initiating_chain_id) {
            return Err(MessageGraphError::ChainNotInDependencySet(initiating_chain_id));
        }

        // Attempt to fetch the rollup config for the executing chain from the registry. If the
        // rollup config is not found, fall back to the local rollup configs.
        let exec_rollup_config = ROLLUP_CONFIGS
            .get(&message.executing_chain_id)
            .or_else(|| self.rollup_configs.get(&message.executing_chain_id))
            .ok_or(MessageGraphError::MissingRollupConfig(message.executing_chain_id))?;

        // Activation invariant: Interop must be active on the executing chain AND the executing
        // block must not be the activation block.
        if !exec_rollup_config.is_interop_active(message.executing_timestamp) ||
            exec_rollup_config.is_first_interop_block(message.executing_timestamp)
        {
            return Err(MessageGraphError::ExecutedTooEarly {
                activation_time: exec_rollup_config.hardforks.interop_time.unwrap_or_default(),
                executing_message_time: message.executing_timestamp,
            });
        }

        // Attempt to fetch the rollup config for the initiating chain from the registry. If the
        // rollup config is not found, fall back to the local rollup configs.
        let rollup_config = ROLLUP_CONFIGS
            .get(&initiating_chain_id)
            .or_else(|| self.rollup_configs.get(&initiating_chain_id))
            .ok_or(MessageGraphError::MissingRollupConfig(initiating_chain_id))?;

        // Timestamp invariant: The timestamp at the time of inclusion of the initiating message
        // MUST be less than or equal to the timestamp of the executing message as well as greater
        // than the Interop activation block's timestamp.
        if initiating_timestamp > message.executing_timestamp {
            return Err(MessageGraphError::MessageInFuture {
                max: message.executing_timestamp,
                actual: initiating_timestamp,
            });
        } else if !rollup_config.is_interop_active(initiating_timestamp) ||
            rollup_config.is_first_interop_block(initiating_timestamp)
        {
            return Err(MessageGraphError::InitiatedTooEarly {
                activation_time: rollup_config.hardforks.interop_time.unwrap_or_default(),
                initiating_message_time: initiating_timestamp,
            });
        }

        // Message expiry invariant: The timestamp of the initiating message must be no more than
        // `MESSAGE_EXPIRY_WINDOW` seconds in the past, relative to the timestamp of the executing
        // message.
        if initiating_timestamp <
            message.executing_timestamp.saturating_sub(self.message_expiry_window)
        {
            return Err(MessageGraphError::MessageExpired {
                initiating_timestamp,
                executing_timestamp: message.executing_timestamp,
            });
        }

        // Fetch the header & receipts for the message's claimed origin block on the remote chain.
        let remote_header = self
            .provider
            .header_by_number(
                message.inner.identifier.chainId.saturating_to(),
                message.inner.identifier.blockNumber.saturating_to(),
            )
            .await?;
        let remote_receipts = self
            .provider
            .receipts_by_number(
                message.inner.identifier.chainId.saturating_to(),
                message.inner.identifier.blockNumber.saturating_to(),
            )
            .await?;

        // Find the log that matches the message's claimed log index. Note that the
        // log index is global to the block, so we chain the full block's logs together
        // to find it.
        let remote_log = remote_receipts
            .iter()
            .flat_map(|receipt| receipt.logs())
            .nth(message.inner.identifier.logIndex.saturating_to())
            .ok_or(MessageGraphError::RemoteMessageNotFound {
                chain_id: message.inner.identifier.chainId.to(),
                message_hash: message.inner.payloadHash,
            })?;

        // Validate the message's origin is correct.
        if remote_log.address != message.inner.identifier.origin {
            return Err(MessageGraphError::InvalidMessageOrigin {
                expected: message.inner.identifier.origin,
                actual: remote_log.address,
            });
        }

        // Validate that the message hash is correct.
        let remote_message = RawMessagePayload::from(remote_log);
        let remote_message_hash = keccak256(remote_message.as_ref());
        if remote_message_hash != message.inner.payloadHash {
            return Err(MessageGraphError::InvalidMessageHash {
                expected: message.inner.payloadHash,
                actual: remote_message_hash,
            });
        }

        // Validate that the timestamp of the block header containing the log is correct.
        if remote_header.timestamp != initiating_timestamp {
            return Err(MessageGraphError::InvalidMessageTimestamp {
                expected: initiating_timestamp,
                actual: remote_header.timestamp,
            });
        }

        Ok(())
    }
}

#[cfg(test)]
#[allow(clippy::zero_sized_map_values)]
mod test {
    use super::{MessageGraph, detect_cycles};
    use crate::{
        MESSAGE_EXPIRY_WINDOW, MessageGraphError,
        message::EnrichedExecutingMessage,
        test_util::{ExecutingMessageBuilder, SuperchainBuilder},
    };
    use alloc::collections::BTreeMap;
    use alloy_primitives::{Address, B256, U256, hex, keccak256};
    use kona_genesis::{ChainDependency, DependencySet};
    use std::sync::OnceLock;

    const MOCK_MESSAGE: [u8; 4] = hex!("deadbeef");
    const CHAIN_A_ID: u64 = 1;
    const CHAIN_B_ID: u64 = 2;
    const CHAIN_C_ID: u64 = 3;

    fn default_dep_set() -> &'static DependencySet {
        static DEP_SET: OnceLock<DependencySet> = OnceLock::new();
        DEP_SET.get_or_init(|| {
            let mut dependencies = BTreeMap::new();
            dependencies.insert(CHAIN_A_ID, ChainDependency {});
            dependencies.insert(CHAIN_B_ID, ChainDependency {});
            dependencies.insert(CHAIN_C_ID, ChainDependency {});
            DependencySet { dependencies, override_message_expiry_window: None }
        })
    }

    /// Returns a [`SuperchainBuilder`] with two chains (ids: `CHAIN_A_ID` and `CHAIN_B_ID`),
    /// configured with interop activating at timestamp `0`, the current block at timestamp `2`,
    /// and a block time of `2` seconds.
    fn default_superchain() -> SuperchainBuilder {
        let mut superchain = SuperchainBuilder::new();
        superchain
            .chain(CHAIN_A_ID)
            .with_timestamp(2)
            .with_block_time(2)
            .with_interop_activation_time(0);
        superchain
            .chain(CHAIN_B_ID)
            .with_timestamp(2)
            .with_block_time(2)
            .with_interop_activation_time(0);

        superchain
    }

    #[tokio::test]
    async fn test_derive_and_resolve_simple_graph_no_cycles() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        graph.resolve().await.unwrap();
    }

    #[tokio::test]
    async fn test_executing_chain_not_in_dep_set_rejected() {
        let mut superchain = default_superchain();
        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let mut deps = BTreeMap::new();
        deps.insert(CHAIN_A_ID, ChainDependency {});
        let dep_set = DependencySet { dependencies: deps, override_message_expiry_window: None };

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, &dep_set, MESSAGE_EXPIRY_WINDOW)
                .await
                .unwrap();
        let MessageGraphError::InvalidMessages(invalid) = graph.resolve().await.unwrap_err() else {
            panic!("Expected InvalidMessages")
        };
        assert_eq!(
            *invalid.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::ChainNotInDependencySet(CHAIN_B_ID)
        );
    }

    #[tokio::test]
    async fn test_initiating_chain_not_in_dep_set_rejected() {
        let mut superchain = default_superchain();
        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let mut deps = BTreeMap::new();
        deps.insert(CHAIN_B_ID, ChainDependency {});
        let dep_set = DependencySet { dependencies: deps, override_message_expiry_window: None };

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, &dep_set, MESSAGE_EXPIRY_WINDOW)
                .await
                .unwrap();
        let MessageGraphError::InvalidMessages(invalid) = graph.resolve().await.unwrap_err() else {
            panic!("Expected InvalidMessages")
        };
        assert_eq!(
            *invalid.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::ChainNotInDependencySet(CHAIN_A_ID)
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_mutual_cycle_detected() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;
        let chain_b_time = superchain.chain(CHAIN_B_ID).header.timestamp;

        // Chain A: executing message at log index 0, referencing chain B log index 0.
        // Chain B: executing message at log index 0, referencing chain A log index 0.
        // This creates a mutual cycle: A depends on B and B depends on A.
        superchain.chain(CHAIN_A_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_B_ID)
                .with_origin_timestamp(chain_b_time),
        );
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::CyclicDependency { mut chain_ids } =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected CyclicDependency error")
        };

        chain_ids.sort();
        assert_eq!(chain_ids, vec![CHAIN_A_ID, CHAIN_B_ID]);
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_message_in_future() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time + 1),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::MessageInFuture { max: 2, actual: chain_a_time + 1 }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_initiating_before_interop() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain
            .chain(CHAIN_A_ID)
            .with_interop_activation_time(50)
            .add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::InitiatedTooEarly {
                activation_time: 50,
                initiating_message_time: chain_a_time
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_initiating_before_interop_unaligned_activation() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        // Chain A activates @ `1s`, which is unaligned with the block time of `2s`. The first
        // block, at `2s`, should be the activation block.
        superchain
            .chain(CHAIN_A_ID)
            .with_interop_activation_time(1)
            .add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::InitiatedTooEarly {
                activation_time: 1,
                initiating_message_time: chain_a_time
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_initiating_at_interop_activation() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain
            .chain(CHAIN_A_ID)
            .with_interop_activation_time(chain_a_time)
            .add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::InitiatedTooEarly { activation_time: 2, initiating_message_time: 2 }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_initiating_chain_interop_time_none_rejected() {
        // A chain with no interop activation cannot produce valid init messages.
        // Op-supervisor's `IsInterop(ts) = InteropTime != nil && ts >= *InteropTime`
        // rejects in this case, and `check_single_dependency` must do the same via
        // `is_interop_active`.
        //
        // Attack: an attacker plants an arbitrary log on a chain whose kona-visible
        // rollup config has `interop_time = None` (stale registry, oracle-supplied
        // config, or a misconfigured dep set), then references it from an executing
        // message on another chain. Without a `None`-rejecting gate, kona accepts the
        // forged init message and downstream consumers (e.g. `L2toL2CrossDomainMessenger`)
        // deliver the call.
        let mut superchain = default_superchain();

        // Init chain (A): `interop_time = None`. Simulates a chain whose rollup config
        // (bundled registry stale, oracle-supplied, or genuinely pre-interop but
        // incorrectly included in the dep set) has no interop activation.
        superchain.chain(CHAIN_A_ID).modify_rollup_cfg(|cfg| cfg.hardforks.interop_time = None);
        // Sanity-check the precondition; otherwise we'd be testing the wrong thing.
        assert!(
            superchain.chain(CHAIN_A_ID).rollup_config.hardforks.interop_time.is_none(),
            "test precondition: init chain must have interop_time = None"
        );

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        // Attacker plants an arbitrary log on A and references it from an executing
        // message on B. With the broken gate, kona accepts. With a spec-correct gate,
        // kona rejects because A has no interop activation configured.
        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages — forged init message must be rejected")
        };

        assert!(
            invalid_messages.contains_key(&CHAIN_B_ID),
            "exec chain B's message referencing a non-interop init chain must be invalidated, got {:?}",
            invalid_messages
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_executing_before_interop() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;
        let chain_b_time = superchain.chain(CHAIN_B_ID).header.timestamp;

        // Move CHAIN_B (the executing chain) activation to t=50, so its block at t=2 is
        // pre-interop. CHAIN_A (initiating) stays at the default (interop_time=0, well
        // past activation). The executing-chain guard must reject via `!is_interop_active`.
        superchain.chain(CHAIN_B_ID).with_interop_activation_time(50);
        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::ExecutedTooEarly {
                activation_time: 50,
                executing_message_time: chain_b_time,
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_executing_before_interop_unaligned_activation() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;
        let chain_b_time = superchain.chain(CHAIN_B_ID).header.timestamp;

        // CHAIN_B activates @ `1s`, unaligned with the block time of `2s`. The first
        // CHAIN_B block at t=2 IS its activation block (`is_first_interop_block` is
        // true under unaligned activation: `is_interop_active(2) && !is_interop_active(0)`).
        superchain.chain(CHAIN_B_ID).with_interop_activation_time(1);
        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::ExecutedTooEarly {
                activation_time: 1,
                executing_message_time: chain_b_time,
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_executing_at_interop_activation() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;
        let chain_b_time = superchain.chain(CHAIN_B_ID).header.timestamp;

        // CHAIN_B activates @ `chain_b_time`, exactly aligned with block_time. The block
        // at `chain_b_time` IS the activation block. Mirrors the existing init-side test
        // which uses `with_interop_activation_time(chain_a_time)`.
        superchain.chain(CHAIN_B_ID).with_interop_activation_time(chain_b_time);
        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::ExecutedTooEarly {
                activation_time: chain_b_time,
                executing_message_time: chain_b_time,
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_message_expired() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain
            .chain(CHAIN_B_ID)
            .with_timestamp(chain_a_time + MESSAGE_EXPIRY_WINDOW + 1)
            .add_executing_message(
                ExecutingMessageBuilder::default()
                    .with_message_hash(keccak256(MOCK_MESSAGE))
                    .with_origin_chain_id(CHAIN_A_ID)
                    .with_origin_timestamp(chain_a_time),
            );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::MessageExpired {
                initiating_timestamp: chain_a_time,
                executing_timestamp: chain_a_time + MESSAGE_EXPIRY_WINDOW + 1
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_remote_message_not_found() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::RemoteMessageNotFound {
                chain_id: CHAIN_A_ID,
                message_hash: keccak256(MOCK_MESSAGE)
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_invalid_origin_address() {
        let mut superchain = default_superchain();
        let mock_address = Address::left_padding_from(&[0xFF]);

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_address(mock_address)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::InvalidMessageOrigin {
                expected: mock_address,
                actual: Address::ZERO
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_invalid_message_hash() {
        let mut superchain = default_superchain();
        let mock_message_hash = keccak256([0xBE, 0xEF]);

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(mock_message_hash)
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::InvalidMessageHash {
                expected: mock_message_hash,
                actual: keccak256(MOCK_MESSAGE)
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_invalid_timestamp() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).with_timestamp(4).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).with_timestamp(4).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time - 1),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::InvalidMessageTimestamp {
                expected: chain_a_time - 1,
                actual: chain_a_time
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_message_expired_custom_window() {
        let mut superchain = default_superchain();
        const CUSTOM_EXPIRY: u64 = 10;

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain
            .chain(CHAIN_B_ID)
            .with_timestamp(chain_a_time + CUSTOM_EXPIRY + 1)
            .add_executing_message(
                ExecutingMessageBuilder::default()
                    .with_message_hash(keccak256(MOCK_MESSAGE))
                    .with_origin_chain_id(CHAIN_A_ID)
                    .with_origin_timestamp(chain_a_time),
            );

        let (headers, cfgs, provider) = superchain.build();

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, default_dep_set(), CUSTOM_EXPIRY)
                .await
                .unwrap();
        let MessageGraphError::InvalidMessages(invalid_messages) =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected invalid messages")
        };

        assert_eq!(invalid_messages.len(), 1);
        assert_eq!(
            *invalid_messages.get(&CHAIN_B_ID).unwrap(),
            MessageGraphError::MessageExpired {
                initiating_timestamp: chain_a_time,
                executing_timestamp: chain_a_time + CUSTOM_EXPIRY + 1
            }
        );
    }

    #[tokio::test]
    async fn test_derive_and_resolve_graph_message_not_expired_within_custom_window() {
        let mut superchain = default_superchain();
        const CUSTOM_EXPIRY: u64 = 10;

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain
            .chain(CHAIN_B_ID)
            .with_timestamp(chain_a_time + CUSTOM_EXPIRY - 1)
            .add_executing_message(
                ExecutingMessageBuilder::default()
                    .with_message_hash(keccak256(MOCK_MESSAGE))
                    .with_origin_chain_id(CHAIN_A_ID)
                    .with_origin_timestamp(chain_a_time),
            );

        let (headers, cfgs, provider) = superchain.build();

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, default_dep_set(), CUSTOM_EXPIRY)
                .await
                .unwrap();
        graph.resolve().await.unwrap();
    }

    /// When a chain has been replaced with a deposit-only block, it is excluded from the headers
    /// passed to `derive` (since deposit-only blocks cannot contain executing messages). Executing
    /// messages on other chains that reference initiating messages from the replaced chain must
    /// still resolve successfully, because the provider retains the replaced chain's data.
    #[tokio::test]
    async fn test_resolve_with_replaced_chain_excluded_from_headers() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        // Chain A has an initiating message. Chain B executes it.
        superchain.chain(CHAIN_A_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        // Simulate chain A having been replaced with a deposit-only block by excluding it
        // from the headers passed to derive. The provider still has chain A's data.
        let filtered_headers =
            headers.into_iter().filter(|(chain_id, _)| *chain_id != CHAIN_A_ID).collect();

        let graph = MessageGraph::derive(
            &filtered_headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        graph.resolve().await.unwrap();
    }

    /// Triangle cycle: A→B→C→A. All three chains should be detected as cycle participants.
    #[tokio::test]
    async fn test_derive_and_resolve_triangle_cycle_detected() {
        const CHAIN_C_ID: u64 = 3;

        let mut superchain = SuperchainBuilder::new();
        superchain
            .chain(CHAIN_A_ID)
            .with_timestamp(2)
            .with_block_time(2)
            .with_interop_activation_time(0);
        superchain
            .chain(CHAIN_B_ID)
            .with_timestamp(2)
            .with_block_time(2)
            .with_interop_activation_time(0);
        superchain
            .chain(CHAIN_C_ID)
            .with_timestamp(2)
            .with_block_time(2)
            .with_interop_activation_time(0);

        // A executes from C, B executes from A, C executes from B.
        superchain.chain(CHAIN_A_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_C_ID)
                .with_origin_timestamp(2),
        );
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(2),
        );
        superchain.chain(CHAIN_C_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_B_ID)
                .with_origin_timestamp(2),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::CyclicDependency { mut chain_ids } =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected CyclicDependency error")
        };

        chain_ids.sort();
        assert_eq!(chain_ids, vec![CHAIN_A_ID, CHAIN_B_ID, CHAIN_C_ID]);
    }

    /// Bystander chain: A↔B form a cycle, but C has a one-way dependency on A (not in the cycle).
    /// Only A and B should be reported as cycle participants.
    #[tokio::test]
    async fn test_derive_and_resolve_cycle_with_bystander_chain() {
        const CHAIN_C_ID: u64 = 3;

        let mut superchain = SuperchainBuilder::new();
        superchain
            .chain(CHAIN_A_ID)
            .with_timestamp(2)
            .with_block_time(2)
            .with_interop_activation_time(0);
        superchain
            .chain(CHAIN_B_ID)
            .with_timestamp(2)
            .with_block_time(2)
            .with_interop_activation_time(0);
        superchain
            .chain(CHAIN_C_ID)
            .with_timestamp(2)
            .with_block_time(2)
            .with_interop_activation_time(0);

        // A↔B cycle: both EMs at log index 0, referencing each other at log index 0.
        superchain.chain(CHAIN_A_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_B_ID)
                .with_origin_timestamp(2),
        );
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(2),
        );

        // C executes from A (one-way, not part of the cycle).
        superchain.chain(CHAIN_C_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(2),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();
        let MessageGraphError::CyclicDependency { mut chain_ids } =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected CyclicDependency error")
        };

        chain_ids.sort();
        // Only A and B are in the cycle; C is a bystander.
        assert_eq!(chain_ids, vec![CHAIN_A_ID, CHAIN_B_ID]);
    }

    /// Helper to build an [`EnrichedExecutingMessage`] for direct `detect_cycles` tests.
    ///
    /// `target_timestamp` is the timestamp of the referenced initiating message and is
    /// recorded in `MessageIdentifier.timestamp`. It is kept separate from
    /// `executing_timestamp` so tests can construct EMs whose initiating message is at a
    /// different (e.g. historical) timestamp than the executing block.
    fn make_em(
        executing_chain_id: u64,
        executing_log_index: u32,
        executing_timestamp: u64,
        target_chain_id: u64,
        target_log_index: u64,
        target_timestamp: u64,
    ) -> EnrichedExecutingMessage {
        use crate::{ExecutingMessage, MessageIdentifier};
        EnrichedExecutingMessage::new(
            ExecutingMessage {
                payloadHash: B256::ZERO,
                identifier: MessageIdentifier {
                    origin: Address::ZERO,
                    blockNumber: U256::ZERO,
                    logIndex: U256::from(target_log_index),
                    timestamp: U256::from(target_timestamp),
                    chainId: U256::from(target_chain_id),
                },
            },
            executing_chain_id,
            executing_timestamp,
            executing_log_index,
        )
    }

    /// An executing message with a past timestamp is filtered out of the cycle graph.
    /// Mirrors op-supernode's "past timestamp filtered out" test case.
    #[test]
    fn test_detect_cycles_past_timestamp_filtered_out() {
        let ts: u64 = 1000;
        // Chain A has an EM at the wrong timestamp (ts - 100), referencing chain B.
        let messages = vec![make_em(CHAIN_A_ID, 0, ts - 100, CHAIN_B_ID, 0, ts - 100)];
        let result = detect_cycles(&messages, ts);
        assert!(result.is_empty(), "Past-timestamp EM should be excluded from cycle graph");
    }

    /// An executing message whose *referenced initiating message* is at a historical
    /// timestamp must be filtered out of the cycle graph, even if its executing block
    /// is at the current timestamp.
    ///
    /// This mirrors op-supernode's `buildCycleGraph` secondary filter (`em.Timestamp == ts`,
    /// where `em.Timestamp` is the identifier timestamp). Two EMs that reference each
    /// other at historical timestamps are dependencies on finalized past state, not a
    /// concurrent same-timestamp cycle.
    #[test]
    fn test_detect_cycles_historical_identifier_timestamp_filtered_out() {
        let ts: u64 = 1000;
        let historical: u64 = 500;
        // Both EMs are at the current timestamp but reference initiating messages at a
        // historical timestamp. Without the secondary filter these would form a spurious
        // A→B, B→A cycle.
        let messages = vec![
            make_em(CHAIN_A_ID, 0, ts, CHAIN_B_ID, 0, historical),
            make_em(CHAIN_B_ID, 0, ts, CHAIN_A_ID, 0, historical),
        ];
        let result = detect_cycles(&messages, ts);
        assert!(
            result.is_empty(),
            "EMs referencing historical initiating messages must not participate in the \
             same-timestamp cycle graph",
        );
    }

    /// One-way reference to a chain with no executing messages — no cycle.
    /// Mirrors op-supernode's "one-way ref to chain with no EMs - no cycle" test case.
    #[test]
    fn test_detect_cycles_one_way_ref_to_chain_with_no_ems() {
        let ts: u64 = 1000;
        // Chain A has an EM referencing chain B at log index 0, but chain B has no EMs.
        let messages = vec![make_em(CHAIN_A_ID, 0, ts, CHAIN_B_ID, 0, ts)];
        let result = detect_cycles(&messages, ts);
        assert!(result.is_empty(), "Reference to chain with no EMs should not create a cycle");
    }

    /// Reference before target EM — no dependency edge, no cycle.
    /// Chain A references chain B at log index 2, but chain B's only EM is at log index 3.
    /// Since there is no EM at or before index 2 on chain B, no cross-chain edge is created.
    /// Mirrors op-supernode's "ref before target EM - no dependency, no cycle" test case.
    #[test]
    fn test_detect_cycles_ref_before_target_em_no_cycle() {
        let ts: u64 = 1000;
        let messages = vec![
            make_em(CHAIN_A_ID, 0, ts, CHAIN_B_ID, 2, ts), // A refs B@2
            make_em(CHAIN_B_ID, 3, ts, CHAIN_A_ID, 0, ts), // B's EM is at index 3, refs A@0
        ];
        let result = detect_cycles(&messages, ts);
        // A references B at log index 2, but B's EM is at index 3 (> 2), so
        // executingMessageBefore(B, 2) returns None. Only B→A edge exists, no cycle.
        assert!(result.is_empty(), "Ref before target EM should not create a dependency edge");
    }

    /// Multiple EMs on the same chain with no cross-chain cycle — intra-chain sequential.
    /// Mirrors op-supernode's "intra-chain sequential EMs - no cycle" test case.
    #[test]
    fn test_detect_cycles_intra_chain_sequential_ems_no_cycle() {
        let ts: u64 = 1000;
        // Chain A has two sequential EMs (indices 0 and 5), both referencing chain B.
        // Chain B has no EMs, so no cross-chain edges are created.
        let messages = vec![
            make_em(CHAIN_A_ID, 0, ts, CHAIN_B_ID, 0, ts),
            make_em(CHAIN_A_ID, 5, ts, CHAIN_B_ID, 3, ts),
        ];
        let result = detect_cycles(&messages, ts);
        assert!(
            result.is_empty(),
            "Intra-chain sequential EMs without cross-chain cycle should not be flagged"
        );
    }

    /// Regression test: executing messages whose referenced initiating messages are at
    /// historical timestamps must not be treated as same-timestamp cycle participants.
    ///
    /// op-supernode's `buildCycleGraph` filters candidate nodes by the referenced
    /// initiating-message's timestamp (`em.Timestamp == ts`), where
    /// `ExecutingMessage.Timestamp` is populated from `Identifier.Timestamp` in
    /// `op-supervisor/.../executing_message.go`. Executing messages whose initiating
    /// message is at a historical timestamp therefore never participate in the
    /// same-timestamp cycle graph — they represent a dependency on finalized historical
    /// state, not a concurrent cross-chain dependency.
    ///
    /// This test sets up two chains at the same executing-block timestamp, each with a
    /// single EM that references the *other* chain's initiating message at a historical
    /// timestamp. No same-timestamp cycle can exist: both dependencies are on historical,
    /// already-finalized state. The messages may independently fail per-message validation
    /// (the test harness does not supply historical blocks), but they must not be flagged
    /// as cyclic.
    ///
    /// See ethereum-optimism/optimism#20303 review discussion.
    #[tokio::test]
    async fn test_historical_cross_chain_refs_not_flagged_as_same_ts_cycle() {
        const CURRENT_TS: u64 = 200;
        const HISTORICAL_TS: u64 = 100;

        let mut superchain = SuperchainBuilder::new();
        superchain
            .chain(CHAIN_A_ID)
            .with_timestamp(CURRENT_TS)
            .with_block_time(2)
            .with_interop_activation_time(0);
        superchain
            .chain(CHAIN_B_ID)
            .with_timestamp(CURRENT_TS)
            .with_block_time(2)
            .with_interop_activation_time(0);

        // Chain A's block at CURRENT_TS contains an EM referencing an initiating message
        // on chain B at HISTORICAL_TS. Chain B's block at CURRENT_TS contains an EM
        // referencing an initiating message on chain A at HISTORICAL_TS. Neither reference
        // is a same-timestamp dependency, so no same-timestamp cycle exists.
        superchain.chain(CHAIN_A_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_B_ID)
                .with_origin_timestamp(HISTORICAL_TS),
        );
        superchain.chain(CHAIN_B_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_A_ID)
                .with_origin_timestamp(HISTORICAL_TS),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph = MessageGraph::derive(
            &headers,
            &provider,
            &cfgs,
            default_dep_set(),
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .unwrap();

        // Cycle detection must not flag these messages as cyclic. Per-message validation
        // may still reject them (the test provider has no historical blocks), but that is
        // a separate failure mode; this test is scoped to cycle detection.
        if let Err(MessageGraphError::CyclicDependency { chain_ids }) = graph.resolve().await {
            panic!(
                "Historical cross-chain references must not be flagged as a same-timestamp \
                 cycle. Both EMs reference initiating messages at a historical timestamp and \
                 have no concurrent cross-chain dependency. Chain IDs wrongly flagged: {:?}",
                chain_ids,
            );
        }
    }

    /// Three-chain extension of `ref before target EM`: A:5 refs B:3, B:5 refs C:3, C:5 refs A:3.
    /// Each chain's only EM is at logIdx 5, but every cross-ref targets logIdx 3, so
    /// `executing_message_before` returns `None` for all three lookups and no cross-chain
    /// edges form. Mirrors op-supernode's `triangle with missing leg - no cycle` case.
    #[test]
    fn test_detect_cycles_triangle_with_missing_leg_no_cycle() {
        const CHAIN_C_ID: u64 = 3;
        let ts: u64 = 1000;
        let messages = vec![
            make_em(CHAIN_A_ID, 5, ts, CHAIN_B_ID, 3, ts),
            make_em(CHAIN_B_ID, 5, ts, CHAIN_C_ID, 3, ts),
            make_em(CHAIN_C_ID, 5, ts, CHAIN_A_ID, 3, ts),
        ];
        let result = detect_cycles(&messages, ts);
        assert!(
            result.is_empty(),
            "Triangle where every cross-ref points before the target chain's EM should not cycle"
        );
    }

    /// Multi-hop one-way chain that terminates at an absent chain. A:0 refs B:5; B:3 refs C:0,
    /// where chain C is not in the graph. The B-side ref resolves via `executing_message_before`
    /// to B:3 (the only EM on B, logIdx 3 ≤ 5), giving A:0 → B:3. B:3's cross-ref to C produces
    /// no edge because C has no EMs. Linear, acyclic.
    /// Mirrors op-supernode's `one-way dependency - no cycle` case.
    #[test]
    fn test_detect_cycles_one_way_multi_hop_no_cycle() {
        const CHAIN_C_ID: u64 = 3;
        let ts: u64 = 1000;
        let messages = vec![
            make_em(CHAIN_A_ID, 0, ts, CHAIN_B_ID, 5, ts),
            make_em(CHAIN_B_ID, 3, ts, CHAIN_C_ID, 0, ts),
        ];
        let result = detect_cycles(&messages, ts);
        assert!(
            result.is_empty(),
            "Multi-hop one-way chain ending at an absent chain must not cycle"
        );
    }

    /// Diamond shape: `em_d` depends on both `em_b` and `em_c`, and both `em_b` and `em_c`
    /// depend on `em_a`. Acyclic. Mirrors op-supernode's `TestCheckCycle` diamond case.
    ///
    /// Each EM gets at most one cross-chain edge and one intra-chain edge, so to give
    /// `em_d` two incoming dependencies we make it both the intra-chain successor of
    /// `em_b` (same chain, higher logIdx) and the cross-chain reference to `em_c`.
    /// `em_a` targets an absent chain so it has no outgoing edges.
    #[test]
    fn test_detect_cycles_diamond_no_cycle() {
        const CHAIN_C_ID: u64 = 3;
        const CHAIN_ABSENT: u64 = 99;
        let ts: u64 = 1000;
        let messages = vec![
            // em_a on chain 1, logIdx 0, targets absent chain → no edges.
            make_em(CHAIN_A_ID, 0, ts, CHAIN_ABSENT, 0, ts),
            // em_b on chain 2, logIdx 0, targets em_a → edge em_b → em_a.
            make_em(CHAIN_B_ID, 0, ts, CHAIN_A_ID, 0, ts),
            // em_c on chain 3, logIdx 0, targets em_a → edge em_c → em_a.
            make_em(CHAIN_C_ID, 0, ts, CHAIN_A_ID, 0, ts),
            // em_d on chain 2, logIdx 5, targets em_c → cross-chain edge em_d → em_c,
            // plus intra-chain edge em_d → em_b (em_b precedes em_d on chain 2).
            make_em(CHAIN_B_ID, 5, ts, CHAIN_C_ID, 0, ts),
        ];
        let result = detect_cycles(&messages, ts);
        assert!(result.is_empty(), "Diamond pattern is acyclic and must not be flagged");
    }

    /// Bystander chain whose EM references a chain absent from the graph entirely.
    /// A↔C form a mutual cycle. B's only EM points at chain D, which has no EMs in the
    /// message set. B must be spared. Complements the existing bystander test, where the
    /// bystander references a chain that *is* in the graph.
    /// Mirrors op-supernode's `A↔C cycle with B as bystander` case.
    #[test]
    fn test_detect_cycles_bystander_refs_absent_chain() {
        const CHAIN_C_ID: u64 = 3;
        const CHAIN_D_ID: u64 = 4;
        let ts: u64 = 1000;
        let messages = vec![
            make_em(CHAIN_A_ID, 0, ts, CHAIN_C_ID, 0, ts),
            make_em(CHAIN_C_ID, 0, ts, CHAIN_A_ID, 0, ts),
            make_em(CHAIN_B_ID, 0, ts, CHAIN_D_ID, 0, ts),
        ];
        let mut result = detect_cycles(&messages, ts);
        result.sort();
        assert_eq!(result, vec![CHAIN_A_ID, CHAIN_C_ID]);
        assert!(!result.contains(&CHAIN_B_ID), "Bystander chain B must not be flagged");
    }
}
