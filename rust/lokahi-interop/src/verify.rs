//! Verifying one round's frontier: the message-validity rules applied to the blocks a round
//! observed.
//!
//! The rules themselves are not implemented here. They live in [`kona_interop::MessageRules`],
//! shared with the fault-proof consolidation path, so that a node's verdict and a proof's verdict
//! on the same message cannot differ. This module supplies the two things the rules cannot know by
//! themselves: which blocks are in the round, and whether a referenced initiating message exists.
//!
//! Existence is answered from two places, in this order:
//!
//! 1. the [`FrontierView`] — the blocks of the round's own timestamp, which are not in a log store
//!    yet because a round's blocks are only persisted once it decides to advance; and
//! 2. the per-chain log stores, which hold every earlier timestamp.
//!
//! Between them they cover every timestamp an initiating message may legally carry, so an
//! existence question that neither can answer is a claim about a block that cannot exist.

use crate::{
    chain::{BlockLogs, RoundError},
    checksum::{ChecksumArgs, MessageChecksum, log_hash, log_to_log_hash},
    decide::ChainFrontier,
    error::StoreError,
    logs::{BlockSeal, ContainsQuery, LogsDb, StoredExecutingMessage},
    verified::VerifiedResult,
};
use alloy_eips::BlockNumHash;
use alloy_primitives::{Address, B256, ChainId, U256};
use kona_interop::{
    EnrichedExecutingMessage, ExecutingMessage, MessageGraphError, MessageIdentifier, MessageRules,
    parse_log_to_executing_message,
};
use kona_protocol::BlockInfo;
use std::{collections::BTreeMap, convert::Infallible, sync::Arc};

/// The rules' error type, with the provider error the node side never produces.
type RuleError = MessageGraphError<Infallible>;

/// Why one executing message is invalid.
///
/// One variant per rule, mapped from the [`MessageRules`] error variant that fired rather than
/// re-derived from the rule's condition: which rule an invalid message broke is kona's answer, and
/// this enum only renames it for the node's logs and metrics.
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum RuleViolation {
    /// The message references a chain the verifier does not follow.
    #[error("references chain {0}, which is not in the verifier's chain set")]
    UnknownChain(U256),
    /// Interop had not been active for a full block on the executing chain.
    #[error("interop not active for a full block on the executing chain at {0}")]
    ExecutedTooEarly(u64),
    /// Interop had not been active for a full block on the initiating chain.
    #[error("interop not active for a full block on the initiating chain at {0}")]
    InitiatedTooEarly(u64),
    /// The initiating message is in the executing message's future.
    #[error("initiating timestamp {initiating} is after executing timestamp {executing}")]
    OutOfOrder {
        /// The initiating message's timestamp.
        initiating: u64,
        /// The executing message's timestamp.
        executing: u64,
    },
    /// The initiating message is older than the expiry window allows.
    #[error("initiating message at {initiating} has expired by executing timestamp {executing}")]
    Expired {
        /// The initiating message's timestamp.
        initiating: u64,
        /// The executing message's timestamp.
        executing: u64,
    },
    /// No initiating message with that checksum sits at the claimed position.
    #[error("no initiating message at the claimed position: {0}")]
    NoSuchMessage(&'static str),
    /// A rule reported a failure the node side has no narrower name for.
    ///
    /// Reachable only if [`MessageRules`] grows a variant this crate has not mapped; carrying the
    /// rendering rather than dropping it keeps such a case diagnosable instead of silent.
    #[error("{0}")]
    Other(String),
}

impl RuleViolation {
    /// Renames a [`MessageRules`] error into the node side's vocabulary.
    fn from_rule_error(error: RuleError) -> Self {
        match error {
            RuleError::ExecutedTooEarly { executing_message_time, .. } => {
                Self::ExecutedTooEarly(executing_message_time)
            }
            RuleError::InitiatedTooEarly { initiating_message_time, .. } => {
                Self::InitiatedTooEarly(initiating_message_time)
            }
            RuleError::MessageInFuture { max, actual } => {
                Self::OutOfOrder { initiating: actual, executing: max }
            }
            RuleError::MessageExpired { initiating_timestamp, executing_timestamp } => {
                Self::Expired { initiating: initiating_timestamp, executing: executing_timestamp }
            }
            other => Self::Other(other.to_string()),
        }
    }
}

/// Why a block was found invalid.
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
pub enum InvalidReason {
    /// One of the block's executing messages is invalid.
    #[error("executing message at log index {log_index}: {violation}")]
    Message {
        /// The log index of the offending executing message within its block.
        log_index: u32,
        /// The rule it broke.
        violation: RuleViolation,
    },
    /// The block's same-timestamp messages form a dependency cycle.
    #[error("takes part in a cycle of same-timestamp messages")]
    Cycle,
}

/// One executing message of a frontier block, with the fields the round needs from it.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct FrontierExecutingMessage {
    /// The address that emitted the referenced initiating message.
    pub origin: Address,
    /// The hash of the referenced initiating message's payload.
    pub payload_hash: B256,
    /// The referenced message, in the shape the log store holds and queries it by.
    pub stored: StoredExecutingMessage,
}

impl FrontierExecutingMessage {
    /// Returns the existence question this message asks of the initiating chain.
    pub const fn query(&self) -> ContainsQuery {
        ContainsQuery {
            block_number: self.stored.block_number,
            log_index: self.stored.log_index,
            timestamp: self.stored.timestamp,
            checksum: self.stored.checksum,
        }
    }

    /// Rebuilds the kona message the shared rules and the cycle check operate on.
    fn enriched(
        &self,
        executing_chain: ChainId,
        executing_timestamp: u64,
        executing_log_index: u32,
    ) -> EnrichedExecutingMessage {
        EnrichedExecutingMessage::new(
            ExecutingMessage {
                identifier: MessageIdentifier {
                    origin: self.origin,
                    blockNumber: U256::from(self.stored.block_number),
                    logIndex: U256::from(self.stored.log_index),
                    timestamp: U256::from(self.stored.timestamp),
                    chainId: self.stored.chain_id,
                },
                payloadHash: self.payload_hash,
            },
            executing_chain,
            executing_timestamp,
            executing_log_index,
        )
    }
}

/// One frontier block, indexed for the two questions a round asks of it: which executing messages
/// it holds, and whether it holds an initiating message at a given position.
///
/// The per-log arrays are also exactly what sealing the block into a log store needs, so a block
/// fetched to verify a round is not fetched a second time to persist it.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct FrontierBlock {
    /// The block itself.
    pub block: BlockInfo,
    /// Each log's hash, by log index — the form a log store persists.
    pub log_hashes: Vec<B256>,
    /// Each log's checksum as an initiating message, by log index.
    ///
    /// Answering existence by indexing rather than by lookup is the same shape the log store's
    /// fixed-offset record has, for the same reason: the position is the key.
    checksums: Vec<MessageChecksum>,
    /// The executing messages among the logs, by log index.
    pub executing_messages: BTreeMap<u32, FrontierExecutingMessage>,
}

impl FrontierBlock {
    /// Indexes a fetched block's logs.
    ///
    /// This is the round loop's only checksum call site. Every existence question the round asks
    /// or answers is expressed in terms of the checksums computed here, so relocating the
    /// computation — the open question about whether it belongs to the store layer or to
    /// kona-interop's shared rules — changes this function's imports and nothing else.
    pub fn index(chain_id: ChainId, logs: &BlockLogs) -> Self {
        let BlockLogs { block, logs } = logs;
        let mut log_hashes = Vec::with_capacity(logs.len());
        let mut checksums = Vec::with_capacity(logs.len());
        let mut executing_messages = BTreeMap::new();

        for (log_index, log) in logs.iter().enumerate() {
            // `logs` is the whole block's log sequence, so the enumeration index is the log's
            // block-global index — the one an initiating message is referenced by.
            let log_index = log_index as u32;
            let hash = log_to_log_hash(log);
            log_hashes.push(hash);
            checksums.push(
                ChecksumArgs {
                    block_number: block.number,
                    log_index,
                    timestamp: block.timestamp,
                    chain_id: U256::from(chain_id),
                    log_hash: hash,
                }
                .checksum(),
            );

            if let Some(message) = parse_log_to_executing_message(log)
                .and_then(|message| frontier_executing_message(&message))
            {
                executing_messages.insert(log_index, message);
            }
        }

        Self { block: *block, log_hashes, checksums, executing_messages }
    }

    /// Returns how many logs the block holds.
    pub const fn log_count(&self) -> u32 {
        self.log_hashes.len() as u32
    }

    /// Returns the block's seal.
    pub const fn seal(&self) -> BlockSeal {
        BlockSeal {
            hash: self.block.hash,
            number: self.block.number,
            timestamp: self.block.timestamp,
        }
    }

    /// Answers whether this block holds the queried initiating message.
    pub fn contains(&self, query: &ContainsQuery) -> Option<BlockSeal> {
        (query.block_number == self.block.number && query.timestamp == self.block.timestamp)
            .then(|| self.checksums.get(query.log_index as usize))
            .flatten()
            .is_some_and(|checksum| *checksum == query.checksum)
            .then(|| self.seal())
    }
}

/// Builds the round's view of one executing message, or [`None`] when its identifier does not fit
/// the widths the protocol uses.
///
/// kona's parser accepts the full 256-bit identifier fields the event declares; the Go
/// implementation rejects any non-zero padding at decode time and, because its caller discards the
/// decode error, treats such a log as an ordinary one. A block number, log index or timestamp that
/// wide cannot come from a real block, so both ends agree in practice — and where they could
/// differ, this narrows to Go's behaviour rather than inventing a third.
fn frontier_executing_message(message: &ExecutingMessage) -> Option<FrontierExecutingMessage> {
    let identifier = &message.identifier;
    let block_number = u64::try_from(identifier.blockNumber).ok()?;
    let log_index = u32::try_from(identifier.logIndex).ok()?;
    let timestamp = u64::try_from(identifier.timestamp).ok()?;
    let referenced_log_hash = log_hash(identifier.origin, message.payloadHash);

    Some(FrontierExecutingMessage {
        origin: identifier.origin,
        payload_hash: message.payloadHash,
        stored: StoredExecutingMessage {
            chain_id: identifier.chainId,
            block_number,
            log_index,
            timestamp,
            checksum: ChecksumArgs {
                block_number,
                log_index,
                timestamp,
                chain_id: identifier.chainId,
                log_hash: referenced_log_hash,
            }
            .checksum(),
        },
    })
}

/// Every chain's frontier block for one round.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FrontierView {
    blocks: BTreeMap<ChainId, FrontierBlock>,
}

impl FrontierView {
    /// Builds a view from each chain's indexed frontier block.
    pub const fn new(blocks: BTreeMap<ChainId, FrontierBlock>) -> Self {
        Self { blocks }
    }

    /// Returns one chain's frontier block.
    pub fn block(&self, chain_id: ChainId) -> Option<&FrontierBlock> {
        self.blocks.get(&chain_id)
    }

    /// Returns every chain's frontier block.
    pub const fn blocks(&self) -> &BTreeMap<ChainId, FrontierBlock> {
        &self.blocks
    }

    /// Answers an existence question against `chain_id`'s frontier block.
    pub fn contains(&self, chain_id: ChainId, query: &ContainsQuery) -> Option<BlockSeal> {
        self.block(chain_id)?.contains(query)
    }
}

/// A round's verification outcome: the frontier it establishes, and the blocks it found invalid.
///
/// Invalid blocks are named, not described: turning one into an
/// [`InvalidHead`](crate::InvalidHead) needs the block's output preimage, which is an I/O read the
/// caller does. Verification stays free of it so the verdict is a function of the round's inputs.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct RoundVerdict {
    /// The frontier the round establishes.
    pub verified: VerifiedResult,
    /// Why each invalid block is invalid, by chain.
    pub invalid: BTreeMap<ChainId, InvalidReason>,
}

impl RoundVerdict {
    /// Returns whether every block in the round was valid.
    pub fn is_valid(&self) -> bool {
        self.invalid.is_empty()
    }
}

/// The log stores a round reads existence answers from, by chain.
pub type LogStores = BTreeMap<ChainId, Arc<dyn LogsDb>>;

/// Verifies one round's frontier.
///
/// Every chain in `frontier` must have an entry in `view` and in `stores`; a missing one is a
/// broken invariant rather than an invalid block, because it describes the verifier's own wiring
/// and not the chain's blocks.
pub fn verify_round(
    timestamp: u64,
    l1_inclusion: BlockNumHash,
    frontier: &BTreeMap<ChainId, ChainFrontier>,
    view: &FrontierView,
    stores: &LogStores,
    rules: &MessageRules<'_>,
) -> Result<RoundVerdict, RoundError> {
    let mut verdict = RoundVerdict {
        verified: VerifiedResult { timestamp, l1_inclusion, l2_heads: BTreeMap::new() },
        invalid: BTreeMap::new(),
    };

    for (&chain_id, chain) in frontier {
        let block = view.block(chain_id).ok_or_else(|| {
            RoundError::Invariant(format!("chain {chain_id}: no frontier block was fetched"))
        })?;
        if block.block.id() != chain.block {
            // The frontier block id came from the chain's own atomic snapshot and the logs were
            // fetched for exactly that id. A source answering with a different block is not an
            // invalid L2 block — it is a source that cannot be reasoned about at all.
            return Err(RoundError::Invariant(format!(
                "chain {chain_id}: asked for block {:?} and was given {:?}",
                chain.block,
                block.block.id()
            )));
        }

        verdict.verified.l2_heads.insert(chain_id, chain.block);

        for (&log_index, message) in &block.executing_messages {
            if let Some(violation) = verify_executing_message(
                chain_id,
                block.block.timestamp,
                message,
                view,
                stores,
                rules,
            )? {
                // First violation settles the block: the block is replaced whole, so the
                // remaining messages in it have nothing left to decide.
                verdict.invalid.insert(chain_id, InvalidReason::Message { log_index, violation });
                break;
            }
        }
    }

    for chain_id in cyclic_chains(view) {
        // A cycle is a property of the message set, not of one message, so it names its chains
        // directly. It does not overwrite a rule violation already found on that chain: the
        // earlier verdict is the more specific one.
        verdict.invalid.entry(chain_id).or_insert(InvalidReason::Cycle);
    }

    Ok(verdict)
}

/// Applies the shared rules and the existence check to one executing message, returning the rule
/// it broke or [`None`] when it is valid.
fn verify_executing_message(
    executing_chain: ChainId,
    executing_timestamp: u64,
    message: &FrontierExecutingMessage,
    view: &FrontierView,
    stores: &LogStores,
    rules: &MessageRules<'_>,
) -> Result<Option<RuleViolation>, RoundError> {
    let referenced = message.stored;

    let Ok(initiating_chain) = ChainId::try_from(referenced.chain_id) else {
        return Ok(Some(RuleViolation::UnknownChain(referenced.chain_id)));
    };
    let Some(store) = stores.get(&initiating_chain) else {
        return Ok(Some(RuleViolation::UnknownChain(referenced.chain_id)));
    };

    let Ok(executing_config) = rules.rollup_config_for::<Infallible>(executing_chain) else {
        return Err(RoundError::Invariant(format!(
            "chain {executing_chain}: no rollup config for a chain the verifier follows"
        )));
    };
    let Ok(initiating_config) = rules.rollup_config_for::<Infallible>(initiating_chain) else {
        return Err(RoundError::Invariant(format!(
            "chain {initiating_chain}: no rollup config for a chain the verifier follows"
        )));
    };

    // Rule order matches the shared rules' own, so a message that breaks several is reported
    // against the same one on both sides.
    if let Err(error) = MessageRules::check_executing_activation::<Infallible>(
        executing_config,
        executing_timestamp,
    ) {
        return Ok(Some(RuleViolation::from_rule_error(error)));
    }
    if let Err(error) = MessageRules::check_initiating_activation::<Infallible>(
        initiating_config,
        referenced.timestamp,
    ) {
        return Ok(Some(RuleViolation::from_rule_error(error)));
    }
    if let Err(error) = MessageRules::check_message_ordering::<Infallible>(
        referenced.timestamp,
        executing_timestamp,
    ) {
        return Ok(Some(RuleViolation::from_rule_error(error)));
    }
    if let Err(error) =
        rules.check_message_expiry::<Infallible>(referenced.timestamp, executing_timestamp)
    {
        return Ok(Some(RuleViolation::from_rule_error(error)));
    }

    let query = message.query();

    // A message referencing its own timestamp names a block of this very round, which no log store
    // holds yet — a round's blocks are sealed only once it decides to advance.
    if referenced.timestamp == executing_timestamp &&
        view.contains(initiating_chain, &query).is_some()
    {
        return Ok(None);
    }

    match store.contains(&query) {
        Ok(_) => Ok(None),
        // Every store holds every timestamp below this round's, and the view holds this round's,
        // so none of these is a "come back later": the claim is about a position that no block
        // occupies or will occupy.
        Err(StoreError::Conflict(reason)) => Ok(Some(RuleViolation::NoSuchMessage(reason))),
        Err(StoreError::Future) => {
            Ok(Some(RuleViolation::NoSuchMessage("claims a block beyond the sealed history")))
        }
        Err(StoreError::Skipped) => {
            Ok(Some(RuleViolation::NoSuchMessage("claims a block below the sealed history")))
        }
        Err(StoreError::NotFound) => {
            Ok(Some(RuleViolation::NoSuchMessage("no block is sealed at that height")))
        }
        Err(other) => Err(RoundError::Store(other)),
    }
}

/// Returns the chains taking part in a same-timestamp dependency cycle.
///
/// The cycle rule reads nothing but the message set, so it takes no configuration — unlike the
/// expiry rule, it is an associated function of [`MessageRules`] rather than a method.
fn cyclic_chains(view: &FrontierView) -> Vec<ChainId> {
    let messages: Vec<_> = view
        .blocks()
        .iter()
        .flat_map(|(&chain_id, block)| {
            block.executing_messages.iter().map(move |(&log_index, message)| {
                message.enriched(chain_id, block.block.timestamp, log_index)
            })
        })
        .collect();

    match MessageRules::check_no_cycles::<Infallible>(&messages) {
        Err(RuleError::CyclicDependency { chain_ids }) => chain_ids,
        // `check_no_cycles` reports cycles and nothing else, so anything else is "no cycle".
        Ok(()) | Err(_) => Vec::new(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{LogData, address};
    use kona_protocol::BlockInfo;

    const CHAIN: ChainId = 990_901;

    fn log(seed: u8) -> alloy_primitives::Log {
        alloy_primitives::Log {
            address: address!("4200000000000000000000000000000000000023"),
            data: LogData::new_unchecked(vec![B256::repeat_byte(seed)], vec![seed].into()),
        }
    }

    fn indexed() -> FrontierBlock {
        FrontierBlock::index(
            CHAIN,
            &BlockLogs {
                block: BlockInfo {
                    hash: B256::repeat_byte(0x10),
                    number: 42,
                    parent_hash: B256::repeat_byte(0x0f),
                    timestamp: 1_000,
                },
                logs: vec![log(1), log(2)],
            },
        )
    }

    /// The query a correct reference to the log at `log_index` produces.
    fn query_for(block: &FrontierBlock, log_index: u32) -> ContainsQuery {
        ContainsQuery {
            block_number: block.block.number,
            log_index,
            timestamp: block.block.timestamp,
            checksum: ChecksumArgs {
                block_number: block.block.number,
                log_index,
                timestamp: block.block.timestamp,
                chain_id: U256::from(CHAIN),
                log_hash: block.log_hashes[log_index as usize],
            }
            .checksum(),
        }
    }

    #[test]
    fn indexing_covers_every_log_in_order() {
        let block = indexed();
        assert_eq!(block.log_count(), 2);
        assert_eq!(block.log_hashes, vec![log_to_log_hash(&log(1)), log_to_log_hash(&log(2))]);
        // Neither log is an executing message: they are not `CrossL2Inbox` events.
        assert!(block.executing_messages.is_empty());
    }

    #[test]
    fn a_correct_reference_is_found_at_every_position() {
        let block = indexed();
        for log_index in 0..block.log_count() {
            assert_eq!(block.contains(&query_for(&block, log_index)), Some(block.seal()));
        }
    }

    #[test]
    fn a_reference_to_another_block_is_not_found() {
        let block = indexed();
        let query = ContainsQuery { block_number: 43, ..query_for(&block, 0) };
        assert_eq!(block.contains(&query), None);
    }

    #[test]
    fn a_reference_claiming_the_wrong_timestamp_is_not_found() {
        let block = indexed();
        let query = ContainsQuery { timestamp: 1_001, ..query_for(&block, 0) };
        assert_eq!(block.contains(&query), None);
    }

    #[test]
    fn a_reference_past_the_last_log_is_not_found() {
        let block = indexed();
        let query = ContainsQuery { log_index: 2, ..query_for(&block, 0) };
        assert_eq!(block.contains(&query), None);
    }

    #[test]
    fn a_reference_to_the_right_position_with_the_wrong_checksum_is_not_found() {
        let block = indexed();
        // The log at index 1's checksum, claimed at index 0: position and payload must agree.
        let query = ContainsQuery { log_index: 0, ..query_for(&block, 1) };
        assert_eq!(block.contains(&query), None);
    }

    #[test]
    fn the_view_answers_only_for_the_chain_asked_about() {
        let block = indexed();
        let query = query_for(&block, 0);
        let view = FrontierView::new(BTreeMap::from([(CHAIN, block.clone())]));
        assert_eq!(view.contains(CHAIN, &query), Some(block.seal()));
        assert_eq!(view.contains(CHAIN + 1, &query), None);
    }

    #[test]
    fn rule_errors_keep_their_identity_when_renamed() {
        let cases = [
            (
                RuleError::ExecutedTooEarly { activation_time: 100, executing_message_time: 99 },
                RuleViolation::ExecutedTooEarly(99),
            ),
            (
                RuleError::InitiatedTooEarly { activation_time: 100, initiating_message_time: 98 },
                RuleViolation::InitiatedTooEarly(98),
            ),
            (
                RuleError::MessageInFuture { max: 100, actual: 101 },
                RuleViolation::OutOfOrder { initiating: 101, executing: 100 },
            ),
            (
                RuleError::MessageExpired { initiating_timestamp: 1, executing_timestamp: 100 },
                RuleViolation::Expired { initiating: 1, executing: 100 },
            ),
        ];
        for (error, expected) in cases {
            assert_eq!(RuleViolation::from_rule_error(error), expected);
        }
    }

    #[test]
    fn an_unmapped_rule_error_keeps_its_rendering() {
        let error = RuleError::MissingRollupConfig(7);
        let rendered = error.to_string();
        assert_eq!(RuleViolation::from_rule_error(error), RuleViolation::Other(rendered));
    }
}
