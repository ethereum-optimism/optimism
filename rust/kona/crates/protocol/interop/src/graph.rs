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
use kona_genesis::RollupConfig;
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
/// - Only same-timestamp executing messages are included as nodes.
/// - Intra-chain edges: each EM depends on the previous EM on the same chain.
/// - Cross-chain edges: each EM depends on `executingMessageBefore(targetChain, targetLogIdx)`.
fn detect_cycles(messages: &[EnrichedExecutingMessage], timestamp: u64) -> Vec<u64> {
    // Filter to same-timestamp messages and create nodes.
    let mut nodes = Vec::new();
    // BTreeMap for deterministic iteration order.
    let mut chain_nodes: BTreeMap<u64, Vec<usize>> = BTreeMap::new();

    for msg in messages {
        if msg.executing_timestamp != timestamp {
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
            if let Some(target_indices) = chain_nodes.get(&target_chain)
                && let Some(dep_idx) =
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

/// The [`MessageGraph`] represents a set of blocks at a given timestamp and the interop
/// dependencies between them.
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
        Ok(Self { messages, provider, rollup_configs, message_expiry_window })
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
        // ChainID Invariant: The chain id of the initiating message MUST be in the dependency set
        // This is enforced implicitly by the graph constructor and the provider.

        let initiating_chain_id = message.inner.identifier.chainId.saturating_to();
        let initiating_timestamp = message.inner.identifier.timestamp.saturating_to::<u64>();

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
        } else if initiating_timestamp
            < rollup_config.hardforks.interop_time.unwrap_or_default() + rollup_config.block_time
        {
            return Err(MessageGraphError::InitiatedTooEarly {
                activation_time: rollup_config.hardforks.interop_time.unwrap_or_default(),
                initiating_message_time: initiating_timestamp,
            });
        }

        // Message expiry invariant: The timestamp of the initiating message must be no more than
        // `MESSAGE_EXPIRY_WINDOW` seconds in the past, relative to the timestamp of the executing
        // message.
        if initiating_timestamp
            < message.executing_timestamp.saturating_sub(self.message_expiry_window)
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
mod test {
    use super::MessageGraph;
    use crate::{
        MESSAGE_EXPIRY_WINDOW, MessageGraphError,
        test_util::{ExecutingMessageBuilder, SuperchainBuilder},
    };
    use alloy_primitives::{Address, hex, keccak256};

    const MOCK_MESSAGE: [u8; 4] = hex!("deadbeef");
    const CHAIN_A_ID: u64 = 1;
    const CHAIN_B_ID: u64 = 2;

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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
        graph.resolve().await.unwrap();
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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
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

        let graph = MessageGraph::derive(&headers, &provider, &cfgs, CUSTOM_EXPIRY).await.unwrap();
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

        let graph = MessageGraph::derive(&headers, &provider, &cfgs, CUSTOM_EXPIRY).await.unwrap();
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

        let graph =
            MessageGraph::derive(&filtered_headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW)
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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
        let MessageGraphError::CyclicDependency { mut chain_ids } =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected CyclicDependency error")
        };

        chain_ids.sort();
        assert_eq!(chain_ids, vec![CHAIN_A_ID, CHAIN_B_ID, CHAIN_C_ID]);
    }

    /// One-way dependency: A executes a message from B. No cycle. Should resolve successfully.
    #[tokio::test]
    async fn test_derive_and_resolve_one_way_dependency_no_cycle() {
        let mut superchain = default_superchain();

        let chain_a_time = superchain.chain(CHAIN_A_ID).header.timestamp;

        // B has an initiating message. A executes it. No reverse dependency.
        superchain.chain(CHAIN_B_ID).add_initiating_message(MOCK_MESSAGE.into());
        superchain.chain(CHAIN_A_ID).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(MOCK_MESSAGE))
                .with_origin_chain_id(CHAIN_B_ID)
                .with_origin_timestamp(chain_a_time),
        );

        let (headers, cfgs, provider) = superchain.build();

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
        graph.resolve().await.unwrap();
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

        let graph =
            MessageGraph::derive(&headers, &provider, &cfgs, MESSAGE_EXPIRY_WINDOW).await.unwrap();
        let MessageGraphError::CyclicDependency { mut chain_ids } =
            graph.resolve().await.unwrap_err()
        else {
            panic!("Expected CyclicDependency error")
        };

        chain_ids.sort();
        // Only A and B are in the cycle; C is a bystander.
        assert_eq!(chain_ids, vec![CHAIN_A_ID, CHAIN_B_ID]);
    }
}
