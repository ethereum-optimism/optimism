use super::*;
use alloc::{vec, vec::Vec};
use kona_genesis::HardForkConfig;

fn tx(ty: OpTxType) -> Bytes {
    Bytes::copy_from_slice(&[u8::from(ty)])
}

fn cfg_with(hardforks: HardForkConfig) -> RollupConfig {
    RollupConfig { hardforks, ..Default::default() }
}

#[test]
fn test_accepts_unrestricted_txs() {
    let txs = vec![
        // A legacy transaction: an RLP list header, not a type byte.
        Bytes::copy_from_slice(&[0xc0]),
        tx(OpTxType::Eip2930),
        tx(OpTxType::Eip1559),
    ];
    assert_eq!(check_sequencer_txs(&RollupConfig::default(), &txs, 0), BatchValidity::Accept);
}

#[test]
fn test_accepts_no_txs() {
    assert_eq!(
        check_sequencer_txs(&RollupConfig::default(), &Vec::new(), 0),
        BatchValidity::Accept
    );
}

#[test]
fn test_drops_empty_tx() {
    let txs = vec![Bytes::new()];
    assert_eq!(
        check_sequencer_txs(&RollupConfig::default(), &txs, 0),
        BatchValidity::Drop(BatchDropReason::EmptyTransaction)
    );
}

#[test]
fn test_drops_deposit_tx() {
    // Deposits are sequencer-forbidden at every fork, so an all-forks-active config must
    // still drop them.
    let cfg = cfg_with(HardForkConfig {
        isthmus_time: Some(0),
        lagoon_time: Some(0),
        ..Default::default()
    });
    let txs = vec![tx(OpTxType::Eip1559), tx(OpTxType::Deposit)];
    assert_eq!(
        check_sequencer_txs(&cfg, &txs, 0),
        BatchValidity::Drop(BatchDropReason::DepositTransaction)
    );
}

#[test]
fn test_gates_eip7702_on_isthmus() {
    let txs = vec![tx(OpTxType::Eip7702)];
    let cfg = cfg_with(HardForkConfig { isthmus_time: Some(10), ..Default::default() });
    assert_eq!(
        check_sequencer_txs(&cfg, &txs, 9),
        BatchValidity::Drop(BatchDropReason::Eip7702PreIsthmus)
    );
    assert_eq!(check_sequencer_txs(&cfg, &txs, 10), BatchValidity::Accept);
}

#[test]
fn test_gates_post_exec_on_sdm() {
    let txs = vec![tx(OpTxType::PostExec)];
    let cfg = cfg_with(HardForkConfig { lagoon_time: Some(10), ..Default::default() });
    assert_eq!(
        check_sequencer_txs(&cfg, &txs, 9),
        BatchValidity::Drop(BatchDropReason::PostExecPreLagoon)
    );
    assert_eq!(check_sequencer_txs(&cfg, &txs, 10), BatchValidity::Accept);
}
