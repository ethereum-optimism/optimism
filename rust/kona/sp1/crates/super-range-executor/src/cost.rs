//! Upper-bound PGU estimates of log-commitment batches and consolidation requests.
//!
//! The measured unit costs are those of the "Unit costs" table in
//! `docs/ai/kona-sp1-proof-limits.md` (SP1 6.8.1); the others are starting values. Check them
//! again after an SP1 upgrade.

use std::collections::{BTreeMap, BTreeSet};

use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::B256;
use op_alloy_consensus::OpReceiptEnvelope;

/// Unit costs of the `log-commitment` program.
mod log_pgu {
    /// One batch: stdin decode, witness check, and the public-value commit.
    pub(super) const BATCH: u64 = 2_000_000;
    /// One block: its root lookup and entry.
    pub(super) const BLOCK: u64 = 200_000;
    /// One receipts-trie witness byte: preimage check, walk, and receipt decode.
    pub(super) const WITNESS_BYTE: u64 = 50;
    /// One log: its leaf and tree-node hashes.
    pub(super) const LOG: u64 = 7_800;
    /// One 136-byte keccak block of a log's topics and data.
    pub(super) const KECCAK_BLOCK: u64 = 2_200;
}

/// Unit costs of the super-range program in consolidation mode.
mod consolidation_pgu {
    /// One request: stdin decode and the output commit.
    pub(super) const BASE: u64 = 20_000_000;
    /// One chain at one timestamp: its headers, output roots, and super-root hashing.
    pub(super) const CHAIN_TIMESTAMP: u64 = 5_000_000;
    /// One check of one executing message, in one round.
    pub(super) const MESSAGE_CHECK: u64 = 20_000;
    /// One witness byte: preimage check and decode at load.
    pub(super) const WITNESS_BYTE: u64 = 25;
    /// One byte read from the oracle: trie walks and header lookups, with repeats.
    pub(super) const READ_BYTE: u64 = 50;
    /// One log check against an entry, without its siblings.
    pub(super) const LOG_CHECK: u64 = 15_000;
    /// One sibling of an opening that a log check verifies.
    pub(super) const SIBLING: u64 = 3_500;
    /// One entry of a batch in the stdin.
    pub(super) const ENTRY: u64 = 5_000;
    /// One `verify_sp1_proof` call.
    pub(super) const VERIFY_CALL: u64 = 84_000;
    /// The first `verify_sp1_proof` call, once.
    pub(super) const FIRST_VERIFY: u64 = 1_100_000;
    /// The recursion of one deferred proof, in PGU-equivalent.
    pub(super) const RECURSION: u64 = 1_200_000;
    /// One gas of a deposit-only block that the consolidation executes.
    pub(super) const REPLACEMENT_GAS: u64 = 11_000;
}

/// Witness-byte bounds that the receipts-path bound prices at the witness and read unit costs.
mod bound_bytes {
    /// Headers, output roots, and super-root preimages of one chain at one timestamp.
    pub(super) const CHAIN_TIMESTAMP: u64 = 8_192;
    /// One EIP-2935 header lookup step: about 7.2 KB of nodes, plus their keys.
    pub(super) const LOOKUP_STEP: u64 = 8_192;
    /// Steps of a walk to the far end of the expiry window: at most 7 days at 1 s blocks, 8,191
    /// blocks per step.
    pub(super) const FAR_WALK_STEPS: u64 = 74;
    /// Receipts-trie bytes per gas of a block: 8 gas per log data byte, 375 per topic, and
    /// 21,000 per receipt, with their keys.
    pub(super) const RECEIPTS_PER_GAS_NUMERATOR: u64 = 3;
    pub(super) const RECEIPTS_PER_GAS_DENOMINATOR: u64 = 20;
    /// Receipts-trie bytes of a block beyond its gas bound.
    pub(super) const RECEIPTS_PER_BLOCK: u64 = 4_096;
    /// Transactions-trie bytes per gas of a block: at least 4 gas per calldata byte.
    pub(super) const TRANSACTIONS_PER_GAS_DENOMINATOR: u64 = 4;
    /// State bytes per gas of a deposit-only block: a cold read of 24 KB of code costs 2,600 gas.
    pub(super) const STATE_PER_DEPOSIT_GAS: u64 = 12;
    /// Header bytes that the `BLOCKHASH` reads of a deposit-only block walk: 32,640 headers of
    /// at most 768 bytes, as each read walks back from the parent.
    pub(super) const BLOCKHASH_WALK: u64 = 32_640 * 768;
}

/// The most witness bytes of one request: under the 2 GiB rkyv limit of the stdin encoding, with
/// room for its per-entry overhead.
pub const MAX_WITNESS_BYTES: u64 = 3 << 29;

/// Stdin bytes of one compressed batch proof: 1.27 MB measured with SP1 6.8.1, with margin.
pub const BATCH_PROOF_BYTES: u64 = 1_400_000;

/// The most stdin bytes of one request, with its batch proofs: under the 5 GB upload limit.
pub const MAX_STDIN_BYTES: u64 = 4 << 30;

/// The most entries in one batch, so the consolidation's decode of a batch stays small.
pub const MAX_ENTRIES_PER_BATCH: usize = 4_096;

/// The witness bytes of the receipts trie of `receipts`: its nodes and their keys.
pub fn receipts_trie_bytes(receipts: &[OpReceiptEnvelope]) -> u64 {
    let mut trie =
        kona_mpt::ordered_trie_with_encoder(receipts, |receipt, out| receipt.encode_2718(out));
    trie.root();
    trie.take_proof_nodes()
        .into_inner()
        .values()
        .map(|node| (B256::len_bytes() + node.len()) as u64)
        .sum()
}

/// An upper bound of the PGU that one block adds to a `log-commitment` batch.
pub fn estimate_log_block_pgu(receipts: &[OpReceiptEnvelope]) -> u64 {
    let logs: u64 = receipts
        .iter()
        .flat_map(OpReceiptEnvelope::logs)
        .map(|log| {
            let payload = 32 * log.topics().len() + log.data.data.len();
            log_pgu::LOG + log_pgu::KECCAK_BLOCK * (payload as u64 + 1).div_ceil(136)
        })
        .sum();
    log_pgu::BLOCK + log_pgu::WITNESS_BYTE * receipts_trie_bytes(receipts) + logs
}

/// An upper bound of the PGU of a `log-commitment` batch whose blocks cost `block_pgu` each.
pub fn estimate_log_commitment_pgu(block_pgu: impl IntoIterator<Item = u64>) -> u64 {
    block_pgu.into_iter().fold(log_pgu::BATCH, u64::saturating_add)
}

/// A block of a log-commitment batch.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct LogBatchBlock {
    /// The block's chain.
    pub chain_id: u64,
    /// The block's number.
    pub number: u64,
    /// The block's hash, to fetch its receipts.
    pub hash: B256,
    /// The block's receipts root.
    pub receipts_root: B256,
    /// [`estimate_log_block_pgu`] of the block's receipts.
    pub pgu: u64,
    /// [`receipts_trie_bytes`] of the block's receipts.
    pub witness_bytes: u64,
}

/// Groups `blocks` into log-commitment batches of at most `budget` PGU by estimate, at most
/// [`MAX_ENTRIES_PER_BATCH`] entries, and at most `witness_limit` witness bytes.
///
/// Keeps one block per receipts root and orders blocks by `(chain_id, number)`. Cuts them with
/// [`split_log_batches`] into the fewest groups that the totals allow, then halves any group still
/// over a cap until it fits or holds one block. A one-block group may stay over a cap.
pub fn batch_log_blocks(
    mut blocks: Vec<LogBatchBlock>,
    budget: u64,
    witness_limit: u64,
) -> Vec<Vec<LogBatchBlock>> {
    blocks.sort_by_key(|block| (block.chain_id, block.number));
    let mut roots = BTreeSet::new();
    blocks.retain(|block| roots.insert(block.receipts_root));
    let fits = |group: &[LogBatchBlock]| {
        group.len() <= MAX_ENTRIES_PER_BATCH &&
            estimate_log_commitment_pgu(group.iter().map(|block| block.pgu)) <= budget &&
            group.iter().map(|block| block.witness_bytes).sum::<u64>() <= witness_limit
    };
    let parts = log_batch_parts(&blocks, budget, witness_limit);
    let mut pending = split_log_batches(&blocks, parts);
    pending.reverse();
    let mut batches = Vec::with_capacity(pending.len());
    while let Some(group) = pending.pop() {
        if group.len() > 1 && !fits(&group) {
            pending.extend(split_log_batches(&group, 2).into_iter().rev());
        } else {
            batches.push(group);
        }
    }
    batches
}

/// The fewest batches that can hold `blocks` within `budget` PGU, [`MAX_ENTRIES_PER_BATCH`]
/// entries, and `witness_limit` witness bytes each, if the blocks split evenly.
fn log_batch_parts(blocks: &[LogBatchBlock], budget: u64, witness_limit: u64) -> usize {
    let pgu: u64 = blocks.iter().map(|block| block.pgu).fold(0, u64::saturating_add);
    let witness_bytes: u64 = blocks.iter().map(|block| block.witness_bytes).sum();
    let by_pgu = pgu.div_ceil(budget.saturating_sub(log_pgu::BATCH).max(1));
    let by_witness = witness_bytes.div_ceil(witness_limit.max(1));
    let by_entries = blocks.len().div_ceil(MAX_ENTRIES_PER_BATCH) as u64;
    by_pgu.max(by_witness).max(by_entries).max(1) as usize
}

/// Cuts `blocks` into at most `parts` contiguous, non-empty groups of about equal estimated PGU:
/// a block goes to the group whose share of the total cost holds its midpoint. With two or more
/// blocks and `parts >= 2`, there are at least two groups.
pub fn split_log_batches(blocks: &[LogBatchBlock], parts: usize) -> Vec<Vec<LogBatchBlock>> {
    // Every block weighs at least 1, so blocks without an estimate still spread out.
    let weight = |block: &LogBatchBlock| u128::from(block.pgu.max(1));
    let total: u128 = blocks.iter().map(weight).sum();
    let parts = parts.clamp(1, blocks.len().max(1)) as u128;
    let mut groups: Vec<Vec<LogBatchBlock>> = Vec::new();
    let mut last_group = None;
    let mut before = 0;
    for block in blocks {
        let group = (2 * before + weight(block)) * parts / (2 * total);
        before += weight(block);
        if last_group != Some(group) {
            groups.push(Vec::new());
            last_group = Some(group);
        }
        groups.last_mut().expect("a group was just pushed").push(*block);
    }
    groups
}

/// What drives the PGU of one consolidation request, from its collection and the message scan.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct ConsolidationCost {
    /// Chains times transition timestamps.
    pub chain_timestamps: u64,
    /// Executing messages of the heads times the rounds that check them, at most.
    pub message_checks: u64,
    /// Witness preimage bytes.
    pub witness_bytes: u64,
    /// Bytes that the run read from the oracle, with repeats.
    pub read_bytes: u64,
    /// Log checks against an entry; each verifies an opening.
    pub log_checks: u64,
    /// Siblings of the openings that those checks verify.
    pub log_check_siblings: u64,
    /// Batch proofs in the stdin.
    pub batch_proofs: u64,
    /// Entries of those batches.
    pub batch_entries: u64,
    /// Gas of the deposit-only blocks that the run executes.
    pub replaced_gas: u64,
}

/// An upper bound of the PGU-equivalent of a consolidation request, with the recursion work of
/// its deferred proofs.
pub fn estimate_consolidation_pgu(cost: &ConsolidationCost) -> u64 {
    use consolidation_pgu::*;
    let deferred = if cost.batch_proofs == 0 {
        0
    } else {
        FIRST_VERIFY + (VERIFY_CALL + RECURSION) * cost.batch_proofs
    };
    [
        BASE,
        CHAIN_TIMESTAMP * cost.chain_timestamps,
        MESSAGE_CHECK * cost.message_checks,
        WITNESS_BYTE * cost.witness_bytes,
        READ_BYTE * cost.read_bytes,
        LOG_CHECK * cost.log_checks,
        SIBLING * cost.log_check_siblings,
        ENTRY * cost.batch_entries,
        deferred,
        REPLACEMENT_GAS * cost.replaced_gas,
    ]
    .into_iter()
    .fold(0, u64::saturating_add)
}

/// One transition timestamp of a request on the receipts path, from the message scan.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ScannedTimestampCost {
    /// Chains of the timestamp.
    pub chains: u64,
    /// Chains replaced at the timestamp; the run checks the messages once more per replacement.
    pub replaced_chains: u64,
    /// Executing messages of the heads.
    pub messages: u64,
    /// Receipts-trie bytes of the heads.
    pub head_receipts_bytes: u64,
    /// Deposit gas of the replaced heads.
    pub replaced_deposit_gas: u64,
    /// Gas used by the replaced heads, whose transactions tries the replacement walks.
    pub replaced_head_gas_used: u64,
    /// Gas used by each distinct block below a head that a message references, by chain. Each
    /// chain needs its own walk to the far end of the window.
    pub referenced_gas_used: BTreeMap<u64, Vec<u64>>,
}

/// An upper bound of [`estimate_consolidation_pgu`] for a request on the receipts path, before
/// its collection.
pub fn bound_receipts_path_pgu(timestamps: &[ScannedTimestampCost]) -> u64 {
    use consolidation_pgu::*;
    let per_byte = WITNESS_BYTE + READ_BYTE;
    timestamps.iter().fold(BASE, |total, timestamp| {
        let rounds = timestamp.replaced_chains + 1;
        let lookups = rounds.saturating_mul(timestamp_lookup_bytes(timestamp));
        let referenced = rounds.saturating_mul(timestamp_referenced_bytes(timestamp));
        [
            CHAIN_TIMESTAMP * timestamp.chains,
            MESSAGE_CHECK * rounds * timestamp.messages,
            (WITNESS_BYTE + READ_BYTE * rounds) * timestamp.head_receipts_bytes,
            per_byte * timestamp_chain_bytes(timestamp),
            REPLACEMENT_GAS * timestamp.replaced_deposit_gas,
            per_byte.saturating_mul(lookups),
            per_byte.saturating_mul(referenced),
        ]
        .into_iter()
        .fold(total, u64::saturating_add)
    })
}

/// An upper bound of the witness bytes of a request on the receipts path, before its collection.
pub fn bound_receipts_path_witness_bytes(timestamps: &[ScannedTimestampCost]) -> u64 {
    timestamps.iter().fold(0, |total, timestamp| {
        [
            timestamp.head_receipts_bytes,
            timestamp_chain_bytes(timestamp),
            timestamp_lookup_bytes(timestamp),
            timestamp_referenced_bytes(timestamp),
        ]
        .into_iter()
        .fold(total, u64::saturating_add)
    })
}

/// Header and output-root bytes of the timestamp's chains, and the transactions tries, deposit
/// state, and `BLOCKHASH` walks of its replaced heads.
const fn timestamp_chain_bytes(timestamp: &ScannedTimestampCost) -> u64 {
    bound_bytes::CHAIN_TIMESTAMP * timestamp.chains +
        timestamp.replaced_head_gas_used / bound_bytes::TRANSACTIONS_PER_GAS_DENOMINATOR +
        timestamp.replaced_deposit_gas * bound_bytes::STATE_PER_DEPOSIT_GAS +
        bound_bytes::BLOCKHASH_WALK * timestamp.replaced_chains
}

/// Header-lookup bytes of one round: a far walk per referenced chain, as each timestamp has its
/// own header cache, and a step per referenced block.
fn timestamp_lookup_bytes(timestamp: &ScannedTimestampCost) -> u64 {
    let chains = timestamp.referenced_gas_used.len() as u64;
    let blocks: usize = timestamp.referenced_gas_used.values().map(Vec::len).sum();
    bound_bytes::LOOKUP_STEP * (bound_bytes::FAR_WALK_STEPS * chains + blocks as u64)
}

/// Receipts-trie bytes of the timestamp's referenced blocks in one round.
fn timestamp_referenced_bytes(timestamp: &ScannedTimestampCost) -> u64 {
    timestamp
        .referenced_gas_used
        .values()
        .flatten()
        .map(|gas_used| {
            gas_used / bound_bytes::RECEIPTS_PER_GAS_DENOMINATOR *
                bound_bytes::RECEIPTS_PER_GAS_NUMERATOR +
                bound_bytes::RECEIPTS_PER_BLOCK
        })
        .fold(0, u64::saturating_add)
}

#[cfg(test)]
mod tests {
    use alloy_consensus::{Receipt, ReceiptWithBloom};
    use alloy_primitives::{Address, Bytes, Log, LogData};

    use super::*;

    fn block(chain_id: u64, number: u64, pgu: u64) -> LogBatchBlock {
        LogBatchBlock {
            chain_id,
            number,
            hash: B256::repeat_byte(number as u8),
            receipts_root: B256::left_padding_from(
                &[chain_id.to_be_bytes(), number.to_be_bytes()].concat(),
            ),
            pgu,
            witness_bytes: 1,
        }
    }

    fn receipt(logs: usize, topics: usize, data: usize) -> OpReceiptEnvelope {
        let log = Log {
            address: Address::repeat_byte(1),
            data: LogData::new_unchecked(vec![B256::ZERO; topics], Bytes::from(vec![1; data])),
        };
        OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
            receipt: Receipt {
                status: true.into(),
                cumulative_gas_used: 21_000,
                logs: vec![log; logs],
            },
            logs_bloom: Default::default(),
        })
    }

    fn numbers(groups: &[Vec<LogBatchBlock>]) -> Vec<Vec<u64>> {
        groups.iter().map(|group| group.iter().map(|block| block.number).collect()).collect()
    }

    /// Pins the part count as the largest of the PGU, witness, and entry terms.
    #[test]
    fn log_batch_parts_takes_the_largest_term() {
        let budget = 11 * log_pgu::BATCH;
        let blocks: Vec<_> = (0..10).map(|n| block(1, n, log_pgu::BATCH)).collect();
        assert_eq!(log_batch_parts(&blocks, budget, u64::MAX), 1);
        assert_eq!(log_batch_parts(&blocks, 6 * log_pgu::BATCH, u64::MAX), 2);
        assert_eq!(log_batch_parts(&blocks, budget, 3), 4);
        let cheap: Vec<_> =
            (0..2 * MAX_ENTRIES_PER_BATCH as u64 + 1).map(|n| block(2, n, 0)).collect();
        assert_eq!(log_batch_parts(&cheap, budget, u64::MAX), 3);
        assert_eq!(log_batch_parts(&[], budget, u64::MAX), 1);
    }

    /// Pins contiguous groups that hold every block once, in order, with costs within one block
    /// of an even share.
    #[test]
    fn split_log_batches_is_contiguous_and_balanced() {
        let blocks: Vec<_> = (0..20).map(|n| block(1, n, 1 + n % 7)).collect();
        let total: u64 = blocks.iter().map(|block| block.pgu).sum();
        let largest = blocks.iter().map(|block| block.pgu).max().unwrap();
        for parts in 1..=6 {
            let groups = split_log_batches(&blocks, parts);
            assert_eq!(groups.len(), parts);
            assert_eq!(groups.concat(), blocks);
            for group in &groups {
                let cost: u64 = group.iter().map(|block| block.pgu).sum();
                assert!(cost.abs_diff(total / parts as u64) <= largest, "{parts}: {cost}");
            }
        }
        let heavy = [block(1, 0, 1), block(1, 1, 1_000), block(1, 2, 1)];
        assert_eq!(numbers(&split_log_batches(&heavy, 2)), [vec![0], vec![1, 2]]);
        assert_eq!(split_log_batches(&blocks[..1], 2).len(), 1);
    }

    /// Pins the order, one block per root, and halving of a group over a cap until it fits or is
    /// one block.
    #[test]
    fn oversized_group_is_resplit_until_it_fits_or_is_one_block() {
        let budget = 10 * log_pgu::BATCH;
        let mut blocks: Vec<_> = (1..=6).rev().map(|n| block(1, n, log_pgu::BATCH)).collect();
        blocks.push(block(1, 3, log_pgu::BATCH));
        let groups = batch_log_blocks(blocks, budget, u64::MAX);
        assert_eq!(numbers(&groups), [vec![1, 2, 3, 4, 5, 6]]);

        let uneven = vec![
            block(1, 1, log_pgu::BATCH),
            block(1, 2, 7 * log_pgu::BATCH),
            block(1, 3, log_pgu::BATCH),
            block(1, 4, 20 * log_pgu::BATCH),
        ];
        let groups = batch_log_blocks(uneven, budget, u64::MAX);
        assert_eq!(numbers(&groups), [vec![1, 2], vec![3], vec![4]]);
        for group in &groups[..2] {
            assert!(estimate_log_commitment_pgu(group.iter().map(|block| block.pgu)) <= budget);
        }

        let wide: Vec<_> =
            (0..4).map(|n| LogBatchBlock { witness_bytes: 10, ..block(3, n, 1) }).collect();
        let groups = batch_log_blocks(wide, budget, 25);
        assert_eq!(numbers(&groups), [vec![0, 1], vec![2, 3]]);
    }

    /// Pins an estimate that never drops with more logs, topics, data, or receipts.
    #[test]
    fn estimator_is_monotonic() {
        let base = estimate_log_block_pgu(&[receipt(2, 1, 32)]);
        for larger in [
            vec![receipt(3, 1, 32)],
            vec![receipt(2, 2, 32)],
            vec![receipt(2, 1, 200)],
            vec![receipt(2, 1, 32), receipt(0, 0, 0)],
        ] {
            assert!(estimate_log_block_pgu(&larger) > base);
        }
        let consolidation = ConsolidationCost { read_bytes: 1, ..Default::default() };
        let base = estimate_consolidation_pgu(&consolidation);
        for larger in [
            ConsolidationCost { read_bytes: 2, ..consolidation },
            ConsolidationCost { witness_bytes: 1, ..consolidation },
            ConsolidationCost { log_checks: 1, ..consolidation },
            ConsolidationCost { batch_proofs: 1, ..consolidation },
            ConsolidationCost { replaced_gas: 1, ..consolidation },
        ] {
            assert!(estimate_consolidation_pgu(&larger) > base);
        }
    }

    /// Pins a bound that grows with each round, block, chain, message, and timestamp.
    #[test]
    fn receipts_path_bound_grows_per_round_chain_and_timestamp() {
        let timestamp = ScannedTimestampCost {
            chains: 2,
            messages: 10,
            head_receipts_bytes: 1_000,
            referenced_gas_used: [(1, vec![1_000_000])].into(),
            ..Default::default()
        };
        let one = bound_receipts_path_pgu(std::slice::from_ref(&timestamp));
        let two_timestamps = bound_receipts_path_pgu(&[timestamp.clone(), timestamp.clone()]);
        assert_eq!(two_timestamps - one, one - consolidation_pgu::BASE);
        let same_chain = ScannedTimestampCost {
            referenced_gas_used: [(1, vec![1_000_000; 2])].into(),
            ..timestamp
        };
        let other_chain = ScannedTimestampCost {
            referenced_gas_used: [(1, vec![1_000_000]), (2, vec![1_000_000])].into(),
            ..timestamp
        };
        assert!(
            bound_receipts_path_pgu(std::slice::from_ref(&other_chain)) >
                bound_receipts_path_pgu(std::slice::from_ref(&same_chain))
        );
        for larger in [
            ScannedTimestampCost { replaced_chains: 1, ..timestamp.clone() },
            same_chain,
            ScannedTimestampCost {
                referenced_gas_used: [(1, vec![2_000_000])].into(),
                ..timestamp.clone()
            },
            ScannedTimestampCost { messages: 11, ..timestamp.clone() },
        ] {
            assert!(bound_receipts_path_pgu(&[larger]) > one);
        }
        let replaced = ScannedTimestampCost { replaced_deposit_gas: 1_000, ..timestamp.clone() };
        assert!(
            bound_receipts_path_witness_bytes(&[replaced]) >
                bound_receipts_path_witness_bytes(&[timestamp])
        );
    }
}
