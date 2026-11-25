//! `trace_replayTransaction`, `trace_replayBlockTransactions` and `trace_filter` against
//! `debug_accountAt`.

use super::fixture::{POST_EXEC_INDEX, SettledBlock};
use alloy_primitives::{U256, map::HashSet};
use alloy_rpc_types_eth::TransactionRequest;
use alloy_rpc_types_trace::{
    filter::TraceFilter,
    parity::{Delta, StateDiff, TraceType},
};
use op_revm::constants::BASE_FEE_RECIPIENT;
use reth_rpc_api::clients::TraceApiClient;

/// Parity state diffs leave every account they change, the sender and the fee vaults alike, with
/// the balance `debug_accountAt` replays at that index, per transaction and per block.
#[tokio::test]
async fn parity_replays_report_settled_state_diffs() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;

    for index in 0..POST_EXEC_INDEX {
        let replay = TraceApiClient::<TransactionRequest>::replay_transaction(
            fixture.client(),
            fixture.transaction_hash(index),
            HashSet::from_iter([TraceType::StateDiff, TraceType::Trace]),
        )
        .await?
        .expect("transaction exists");
        assert_settled_state_diff(
            &fixture,
            index,
            replay.full_trace.state_diff.as_ref(),
            "trace_replayTransaction",
        )
        .await?;
    }

    let replays = TraceApiClient::<TransactionRequest>::replay_block_transactions(
        fixture.client(),
        fixture.block_hash().into(),
        HashSet::from_iter([TraceType::StateDiff]),
    )
    .await?
    .expect("block exists");
    assert_eq!(replays.len(), POST_EXEC_INDEX + 1, "trace_replayBlockTransactions replay count");
    for (index, replay) in replays.iter().take(POST_EXEC_INDEX).enumerate() {
        assert_settled_state_diff(
            &fixture,
            index,
            replay.full_trace.state_diff.as_ref(),
            "trace_replayBlockTransactions",
        )
        .await?;
    }
    Ok(())
}

/// `trace_filter` over the block yields a trace for every transaction, the post-exec one included.
#[tokio::test]
async fn trace_filter_covers_every_transaction() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;

    let filtered = TraceApiClient::<TransactionRequest>::trace_filter(
        fixture.client(),
        TraceFilter {
            from_block: Some(fixture.block_number()),
            to_block: Some(fixture.block_number()),
            ..Default::default()
        },
    )
    .await?;

    let filtered_hashes: HashSet<_> =
        filtered.iter().filter_map(|trace| trace.transaction_hash).collect();
    let block_hashes: HashSet<_> = fixture.transactions().iter().map(|tx| tx.tx_hash()).collect();
    assert_eq!(filtered_hashes, block_hashes, "trace_filter covers every transaction");
    Ok(())
}

/// Requires the diff to change the sender's and the base-fee vault's balances, and every balance
/// it changes to end on the one `debug_accountAt` replays at `index`.
async fn assert_settled_state_diff(
    fixture: &SettledBlock,
    index: usize,
    state_diff: Option<&StateDiff>,
    method: &str,
) -> eyre::Result<()> {
    let state_diff = state_diff.expect("state diff requested");
    for (name, address) in [("sender", fixture.sender), ("base-fee vault", BASE_FEE_RECIPIENT)] {
        assert!(
            state_diff
                .get(&address)
                .and_then(|account| settled_balance(&account.balance))
                .is_some(),
            "{method} state diff of transaction {index} must change the {name}'s balance",
        );
    }
    for (address, account) in state_diff.iter() {
        if let Some(balance) = settled_balance(&account.balance) {
            assert_eq!(
                balance,
                fixture.replayed_balance(index, *address).await?,
                "{method} state diff balance of {address} after transaction {index}",
            );
        }
    }
    Ok(())
}

/// The balance a parity balance delta ends on, if the diff changes it.
const fn settled_balance(delta: &Delta<U256>) -> Option<U256> {
    match delta {
        Delta::Changed(changed) => Some(changed.to),
        Delta::Added(to) => Some(*to),
        _ => None,
    }
}
