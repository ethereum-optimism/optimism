//! `debug_trace*` and `debug_accountAt` against the receipts and the committed state.

use super::fixture::{POST_EXEC_INDEX, REVERTING_INDEX, SettledBlock, TRANSFER_INDEX};
use crate::lagoon::{DEPLOY_GAS, default_trace_gas};
use alloy_primitives::U256;
use alloy_rpc_types_trace::geth::GethTrace;

/// The default, call and prestate tracers report each regular transaction's canonical gas and
/// the balances `debug_accountAt` replays at that index; after the last regular transaction those
/// balances are the committed ones.
#[tokio::test]
async fn debug_tracers_report_settled_gas_and_balances() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;
    let receipt_gas = fixture.receipt_gas()?;
    // Pins the settlement itself; the gas checks below compare with receipts, and the balances are
    // anchored to committed state after the last regular transaction.
    assert_eq!(receipt_gas[0], DEPLOY_GAS - 1, "receipt holds canonical gas");

    let block_traces = fixture.block_traces().await?;
    assert_eq!(block_traces.len(), POST_EXEC_INDEX + 1, "debug_traceBlockByHash trace count");

    for index in 0..POST_EXEC_INDEX {
        let gas_used = receipt_gas[index];
        let GethTrace::Default(frame) = fixture.trace_transaction(index, None).await? else {
            panic!("unexpected default trace of transaction {index}");
        };
        assert_eq!(frame.gas, gas_used, "debug_traceTransaction gas of transaction {index}");
        // The tracer must have watched the execution, not only been handed the settled result:
        // every deployment steps through its init code, and only the reverting one fails.
        assert_eq!(
            frame.struct_logs.is_empty(),
            index == TRANSFER_INDEX,
            "default tracer steps of transaction {index}",
        );
        assert_eq!(
            frame.failed,
            index == REVERTING_INDEX,
            "default tracer status of transaction {index}"
        );
        assert_eq!(
            default_trace_gas(&block_traces[index]),
            gas_used,
            "debug_traceBlockByHash gas of transaction {index}",
        );
        let call_frame = fixture.call_trace(index).await?;
        assert_eq!(
            call_frame.gas_used,
            U256::from(gas_used),
            "callTracer gas of transaction {index}",
        );
        assert_eq!(call_frame.from, fixture.sender, "callTracer frame of transaction {index}");

        // Every balance the prestate diff ends on is the balance debug_accountAt replays at that
        // index, for the sender as well as the fee vaults.
        let diff = fixture.prestate_diff(index).await?;
        assert!(
            diff.post.get(&fixture.sender).is_some_and(|account| account.balance.is_some()),
            "prestate diff of transaction {index} lacks the sender's balance: {diff:?}",
        );
        for (address, account) in &diff.post {
            if let Some(balance) = account.balance {
                assert_eq!(
                    balance,
                    fixture.replayed_balance(index, *address).await?,
                    "prestate diff balance of {address} after transaction {index}",
                );
            }
        }
    }

    // The last regular transaction leaves the committed state, since the post-exec transaction
    // changes nothing; this anchors the replayed balances above to the provider.
    let last_regular = POST_EXEC_INDEX - 1;
    let committed = fixture.committed_balance(fixture.sender)?;
    assert_eq!(
        fixture.replayed_balance(last_regular, fixture.sender).await?,
        committed,
        "debug_accountAt after the last regular transaction",
    );
    assert_eq!(
        fixture.replayed_balance(POST_EXEC_INDEX, fixture.sender).await?,
        committed,
        "debug_accountAt after the post-exec transaction",
    );
    let diff = fixture.prestate_diff(last_regular).await?;
    assert!(
        diff.post.get(&fixture.sender).is_some_and(|account| account.balance.is_some()),
        "prestate diff of transaction {last_regular} lacks the sender's balance: {diff:?}",
    );
    for (address, account) in &diff.post {
        if let Some(balance) = account.balance {
            assert_eq!(
                balance,
                fixture.committed_balance(*address)?,
                "committed balance of {address} after transaction {last_regular}",
            );
        }
    }
    Ok(())
}
