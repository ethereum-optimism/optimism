//! Model-based property tests for SDM post-exec verification through [`OpBlockExecutor`].
//!
//! Arbitrary post-exec payloads run through `PostExecMode::Verify` exactly as block validation
//! drives it, over synthetic blocks of legacy transactions with a trailing `0x7D` transaction.
//! Accept/reject decisions are compared against [`model_accepts`], an independent restatement of
//! the consensus rules. The stateless fault-proof executor is checked against the same rules in
//! kona-executor's proptests, so a drift in either integration layer fails one of the two suites.

use super::*;
use op_alloy::consensus::OpReceiptEnvelope;
use proptest::prelude::*;
use std::collections::HashSet;

/// EVM gas used by an empty-calldata legacy call to a codeless account: the 21k intrinsic cost,
/// which also equals its EIP-7623 calldata floor.
const LEGACY_TX_EVM_GAS: u64 = 21_000;

/// `build_executor` leaves `BlockEnv::number` at its default, so valid payloads anchor to 0.
const VERIFY_BLOCK_NUMBER: u64 = 0;

const BLOCK_GAS_LIMIT: u64 = 500_000;

/// One generated verification scenario: a block of `transaction_count` legacy transactions
/// followed by a `0x7D` transaction carrying `entries` anchored to `payload_block_number`.
#[derive(Clone, Debug)]
struct VerificationCase {
    entries: Vec<SDMGasEntry>,
    payload_block_number: u64,
    transaction_count: u64,
}

fn recovered_post_exec(block_number: u64, entries: Vec<SDMGasEntry>) -> Recovered<OpTxEnvelope> {
    Recovered::new_unchecked(
        OpTxEnvelope::PostExec(build_post_exec_tx(block_number, entries).seal_slow()),
        Address::ZERO,
    )
}

fn legacy_tx(nonce: u64) -> Recovered<OpTxEnvelope> {
    recovered_legacy(TxLegacy {
        nonce,
        gas_limit: 50_000,
        to: TxKind::Call(Address::ZERO),
        ..Default::default()
    })
}

/// Independent restatement of the consensus acceptance rules for a block whose trailing `0x7D`
/// transaction carries `case`'s payload: the payload must anchor to the containing block number,
/// and every entry must carry a non-zero refund, target a unique non-post-exec transaction in
/// range, and stay within that transaction's EVM gas.
fn model_accepts(case: &VerificationCase) -> bool {
    if case.payload_block_number != VERIFY_BLOCK_NUMBER {
        return false;
    }

    let mut seen_indices = HashSet::new();
    case.entries.iter().all(|entry| {
        entry.gas_refund > 0 &&
            seen_indices.insert(entry.index) &&
            entry.index < case.transaction_count &&
            entry.gas_refund <= LEGACY_TX_EVM_GAS
    })
}

/// Generates one refund entry, biased toward valid entries with each violation class
/// (zero refund, refund above EVM gas, post-exec/out-of-range target) as a lower-weight arm.
fn refund_entry_strategy(transaction_count: u64) -> impl Strategy<Value = SDMGasEntry> {
    prop_oneof![
        10 => (0..transaction_count, 1..=LEGACY_TX_EVM_GAS)
            .prop_map(|(index, gas_refund)| SDMGasEntry { index, gas_refund }),
        1 => (0..transaction_count).prop_map(|index| SDMGasEntry { index, gas_refund: 0 }),
        1 => (0..transaction_count, prop_oneof![Just(LEGACY_TX_EVM_GAS + 1), Just(u64::MAX)])
            .prop_map(|(index, gas_refund)| SDMGasEntry { index, gas_refund }),
        // The trailing post-exec tx sits at index `transaction_count`; anything beyond it stays
        // unconsumed. Both must reject.
        1 => (transaction_count..transaction_count + 3, 1..1_000u64)
            .prop_map(|(index, gas_refund)| SDMGasEntry { index, gas_refund }),
    ]
}

fn verification_case_strategy() -> impl Strategy<Value = VerificationCase> {
    (1..=4u64).prop_flat_map(|transaction_count| {
        (
            prop::collection::vec(refund_entry_strategy(transaction_count), 0..8),
            prop_oneof![
                8 => Just(VERIFY_BLOCK_NUMBER),
                1 => Just(VERIFY_BLOCK_NUMBER + 1),
            ],
        )
            .prop_map(move |(entries, payload_block_number)| VerificationCase {
                entries,
                payload_block_number,
                transaction_count,
            })
    })
}

proptest! {
    #[test]
    fn verify_mode_matches_consensus_model(case in verification_case_strategy()) {
        let expected_accept = model_accepts(&case);

        let mut fixture = JovianExecutorFixture::new(
            DEFAULT_DA_FOOTPRINT_GAS_SCALAR,
            BLOCK_GAS_LIMIT,
            JOVIAN_TIMESTAMP,
        );
        let mut transactions: Vec<Recovered<OpTxEnvelope>> =
            (0..case.transaction_count).map(legacy_tx).collect();
        transactions.push(recovered_post_exec(case.payload_block_number, case.entries.clone()));
        let mut verifier = fixture.verifier(case.payload_block_number, case.entries.clone());
        // `execute_block` runs the pre-execution system calls, and the EIP-4788 call requires a
        // parent beacon block root on post-Ecotone blocks.
        verifier.ctx.parent_beacon_block_root = Some(B256::ZERO);

        match verifier.execute_block(transactions.iter()) {
            Ok(result) => {
                prop_assert!(
                    expected_accept,
                    "executor accepted a payload the consensus model rejects: {:?}",
                    case,
                );

                let total_refund: u64 = case.entries.iter().map(|entry| entry.gas_refund).sum();
                prop_assert_eq!(
                    result.gas_used,
                    case.transaction_count * LEGACY_TX_EVM_GAS - total_refund,
                    "canonical block gas must drop by exactly the refunded total",
                );
                prop_assert_eq!(result.receipts.len(), case.transaction_count as usize + 1);
                prop_assert!(matches!(
                    result.receipts.last(),
                    Some(OpReceiptEnvelope::PostExec(_))
                ));
            }
            Err(err) => {
                prop_assert!(
                    !expected_accept,
                    "executor rejected a payload the consensus model accepts: {:?}, error: {}",
                    case,
                    err,
                );
                let message = err.to_string().to_lowercase();
                prop_assert!(
                    message.contains("invalid post-exec payload"),
                    "rejection must surface as a post-exec validation failure, got: {}",
                    err,
                );
            }
        }
    }
}
