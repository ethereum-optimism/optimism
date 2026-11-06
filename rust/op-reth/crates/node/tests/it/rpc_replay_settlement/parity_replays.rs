//! `trace_replayTransaction`, `trace_replayBlockTransactions` and `trace_filter` against
//! `debug_accountAt`.

use super::fixture::{POST_EXEC_INDEX, SettledBlock};
use alloy_primitives::{Address, U256, map::HashSet};
use alloy_rpc_types_eth::TransactionRequest;
use alloy_rpc_types_trace::{
    filter::TraceFilter,
    parity::{Delta, StateDiff, TraceType},
};
use reth_rpc_api::clients::TraceApiClient;

/// Parity state diffs leave the sender with the balance `debug_accountAt` replays at that index,
/// per transaction and per block.
#[tokio::test]
async fn parity_replays_report_settled_state_diffs() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;

    for index in 0..POST_EXEC_INDEX {
        let replay = TraceApiClient::<TransactionRequest>::replay_transaction(
            &fixture.client,
            fixture.transaction_hash(index),
            HashSet::from_iter([TraceType::StateDiff, TraceType::Trace]),
        )
        .await?;
        assert_eq!(
            settled_balance(replay.state_diff.as_ref(), fixture.sender),
            fixture.replayed_balance(index, fixture.sender).await?,
            "trace_replayTransaction state diff of transaction {index}",
        );
    }

    let replays = TraceApiClient::<TransactionRequest>::replay_block_transactions(
        &fixture.client,
        fixture.block_hash().into(),
        HashSet::from_iter([TraceType::StateDiff]),
    )
    .await?
    .expect("block exists");
    assert_eq!(replays.len(), POST_EXEC_INDEX + 1, "trace_replayBlockTransactions replay count");
    for (index, replay) in replays.iter().take(POST_EXEC_INDEX).enumerate() {
        assert_eq!(
            settled_balance(replay.full_trace.state_diff.as_ref(), fixture.sender),
            fixture.replayed_balance(index, fixture.sender).await?,
            "trace_replayBlockTransactions state diff of transaction {index}",
        );
    }
    Ok(())
}

/// `trace_filter` over the block yields a trace for every transaction, the post-exec one included.
#[tokio::test]
async fn trace_filter_covers_every_transaction() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;

    let filtered = TraceApiClient::<TransactionRequest>::trace_filter(
        &fixture.client,
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

/// The balance a parity state diff leaves `address` with.
fn settled_balance(state_diff: Option<&StateDiff>, address: Address) -> U256 {
    let state_diff = state_diff.expect("state diff requested");
    match &state_diff[&address].balance {
        Delta::Changed(changed) => changed.to,
        Delta::Added(to) => *to,
        other => panic!("balance of {address} must change: {other:?}"),
    }
}
