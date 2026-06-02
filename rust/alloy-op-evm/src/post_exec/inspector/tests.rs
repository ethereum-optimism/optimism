use super::*;
use alloy_primitives::{address, b256};

const ACCOUNT_A: Address = address!("00000000000000000000000000000000000000aa");
const ACCOUNT_B: Address = address!("00000000000000000000000000000000000000bb");
const SLOT_1: B256 = b256!("0000000000000000000000000000000000000000000000000000000000000001");

fn run_account(
    insp: &mut SDMWarmingInspector,
    tx_index: u64,
    kind: PostExecTxKind,
    addr: Address,
) -> u64 {
    insp.begin_tx(PostExecTxContext { tx_index, kind });
    insp.observe_account_touch(addr, true);
    insp.finish_tx().refund_total
}

fn run_slot(
    insp: &mut SDMWarmingInspector,
    tx_index: u64,
    addr: Address,
    slot: B256,
    is_sstore: bool,
) -> u64 {
    insp.begin_tx(PostExecTxContext { tx_index, kind: PostExecTxKind::Normal });
    insp.observe_slot_touch(addr, slot, is_sstore);
    insp.finish_tx().refund_total
}

#[test]
fn repeated_account_touch_refunds_once() {
    let mut insp = SDMWarmingInspector::default();

    assert_eq!(run_account(&mut insp, 0, PostExecTxKind::Normal, ACCOUNT_A), 0);
    assert_eq!(run_account(&mut insp, 1, PostExecTxKind::Normal, ACCOUNT_A), ACCOUNT_REWARM_REFUND,);
}

#[test]
fn repeated_storage_refunds_without_account_double_count() {
    for (is_sstore, expected) in [(false, SLOAD_REWARM_REFUND), (true, SSTORE_REWARM_REFUND)] {
        let mut insp = SDMWarmingInspector::default();

        assert_eq!(run_slot(&mut insp, 0, ACCOUNT_A, SLOT_1, is_sstore), 0);
        assert_eq!(run_slot(&mut insp, 1, ACCOUNT_A, SLOT_1, is_sstore), expected);
    }
}

#[test]
fn deposit_warms_but_does_not_claim() {
    let mut insp = SDMWarmingInspector::default();

    assert_eq!(run_account(&mut insp, 0, PostExecTxKind::Deposit, ACCOUNT_B), 0);
    assert_eq!(run_account(&mut insp, 1, PostExecTxKind::Normal, ACCOUNT_B), ACCOUNT_REWARM_REFUND,);
}

#[test]
fn post_exec_tx_never_claims_refunds() {
    let mut insp = SDMWarmingInspector::default();

    let _ = run_account(&mut insp, 0, PostExecTxKind::Normal, ACCOUNT_A);
    assert_eq!(run_account(&mut insp, 1, PostExecTxKind::PostExec, ACCOUNT_A), 0);
}

#[test]
fn intrinsic_access_list_warmth_does_not_claim() {
    let mut insp = SDMWarmingInspector::default();

    insp.begin_tx(PostExecTxContext { tx_index: 0, kind: PostExecTxKind::Normal });
    insp.current_tx.intrinsic_warm_accounts.insert(ACCOUNT_A);
    insp.current_tx.intrinsic_warm_slots.insert((ACCOUNT_A, SLOT_1));
    insp.observe_slot_touch(ACCOUNT_A, SLOT_1, false);
    assert_eq!(insp.finish_tx().refund_total, 0);

    assert_eq!(run_slot(&mut insp, 1, ACCOUNT_A, SLOT_1, false), SLOAD_REWARM_REFUND);
}

#[test]
fn take_last_tx_result_round_trips() {
    let mut insp = SDMWarmingInspector::default();

    let _ = run_account(&mut insp, 0, PostExecTxKind::Normal, ACCOUNT_A);
    let _ = run_account(&mut insp, 1, PostExecTxKind::Normal, ACCOUNT_A);

    assert_eq!(insp.take_last_tx_result().refund_total, ACCOUNT_REWARM_REFUND);
    assert_eq!(insp.take_last_tx_result().refund_total, 0);
}

// The carry mechanism (`warming_state` / `seed_warming_state`) exists so a builder that executes a
// block across several flashblock executors (each with a fresh inspector) still attributes the
// block-scoped warming refund set that a single canonical pass would. These tests pin that a slot
// or account warmed in one inspector is refundable in a freshly seeded one — i.e. the boundary
// does not reset warming.

#[test]
fn seeded_account_warmth_refunds_across_inspectors() {
    // First flashblock: tx 0 warms ACCOUNT_A (first warmer, no refund).
    let mut first = SDMWarmingInspector::default();
    assert_eq!(run_account(&mut first, 0, PostExecTxKind::Normal, ACCOUNT_A), 0);

    // Next flashblock's fresh inspector, seeded with the carried warming state, must treat
    // ACCOUNT_A as already warmed and refund the re-touch — matching whole-block execution.
    let mut next = SDMWarmingInspector::default();
    next.seed_warming_state(first.warming_state());
    assert_eq!(run_account(&mut next, 0, PostExecTxKind::Normal, ACCOUNT_A), ACCOUNT_REWARM_REFUND);
}

#[test]
fn seeded_slot_warmth_refunds_across_inspectors() {
    for (is_sstore, expected) in [(false, SLOAD_REWARM_REFUND), (true, SSTORE_REWARM_REFUND)] {
        let mut first = SDMWarmingInspector::default();
        assert_eq!(run_slot(&mut first, 0, ACCOUNT_A, SLOT_1, is_sstore), 0);

        let mut next = SDMWarmingInspector::default();
        next.seed_warming_state(first.warming_state());
        assert_eq!(run_slot(&mut next, 0, ACCOUNT_A, SLOT_1, is_sstore), expected);
    }
}

#[test]
fn unseeded_inspector_resets_warmth() {
    // Guards the negative case the fix addresses: without seeding, a fresh inspector treats
    // ACCOUNT_A as cold (first warmer), so it would *not* refund — the exact divergence from
    // canonical that flashblock boundaries used to introduce.
    let mut first = SDMWarmingInspector::default();
    assert_eq!(run_account(&mut first, 0, PostExecTxKind::Normal, ACCOUNT_A), 0);

    let mut next = SDMWarmingInspector::default();
    assert_eq!(run_account(&mut next, 0, PostExecTxKind::Normal, ACCOUNT_A), 0);
}
