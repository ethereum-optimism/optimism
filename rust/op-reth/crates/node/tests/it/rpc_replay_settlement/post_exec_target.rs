//! The post-exec (0x7d) transaction as a replay target, and `debug_intermediateRoots`.

use super::fixture::{POST_EXEC_INDEX, SettledBlock};
use crate::lagoon::default_trace_gas;
use alloy_primitives::U256;
use alloy_rpc_types_eth::TransactionRequest;
use alloy_rpc_types_trace::geth::GethTrace;
use jsonrpsee::{core::client::ClientT, rpc_params};
use reth_rpc_api::clients::DebugApiClient;
use serde_json::json;

/// Every tracer gives the post-exec transaction an empty trace at its receipt gas, and a traced
/// transfer carries nothing from the replayed prefix.
#[tokio::test]
async fn post_exec_transaction_traces_as_a_no_op() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;
    let receipt_gas = fixture.receipt_gas()?;

    let GethTrace::Default(post_exec_frame) =
        fixture.trace_transaction(POST_EXEC_INDEX, None).await?
    else {
        panic!("unexpected default trace of the post-exec transaction");
    };
    assert!(post_exec_frame.struct_logs.is_empty(), "debug_traceTransaction of the post-exec tx");
    assert_eq!(
        post_exec_frame.gas, receipt_gas[POST_EXEC_INDEX],
        "debug_traceTransaction gas of the post-exec tx",
    );
    let block_traces = fixture.block_traces().await?;
    assert_eq!(
        default_trace_gas(&block_traces[POST_EXEC_INDEX]),
        receipt_gas[POST_EXEC_INDEX],
        "debug_traceBlockByHash gas of the post-exec tx",
    );
    let post_exec_call: GethTrace = fixture
        .client
        .request(
            "debug_traceTransaction",
            rpc_params![fixture.transaction_hash(POST_EXEC_INDEX), json!({"tracer": "callTracer"})],
        )
        .await?;
    let GethTrace::CallTracer(post_exec_call_frame) = post_exec_call else {
        panic!("unexpected call trace of the post-exec tx: {post_exec_call:?}");
    };
    assert_eq!(post_exec_call_frame.gas_used, U256::ZERO, "callTracer gas of the post-exec tx");
    let post_exec_diff = fixture.prestate_diff(POST_EXEC_INDEX).await?;
    assert!(
        post_exec_diff.pre.is_empty() && post_exec_diff.post.is_empty(),
        "prestate diff of the post-exec tx must be empty: {post_exec_diff:?}",
    );

    let GethTrace::Default(transfer_frame) = fixture.trace_transaction(1, None).await? else {
        panic!("unexpected default trace of the transfer");
    };
    assert!(transfer_frame.struct_logs.is_empty(), "debug_traceTransaction of the transfer");
    Ok(())
}

/// Each regular transaction moves the state root, the post-exec transaction does not, and the
/// last root is the header's.
#[tokio::test]
async fn intermediate_roots_follow_each_boundary() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;

    let roots = DebugApiClient::<TransactionRequest>::debug_intermediate_roots(
        &fixture.client,
        fixture.block_hash(),
        None,
    )
    .await?;

    assert_eq!(roots.len(), POST_EXEC_INDEX + 1, "debug_intermediateRoots root count");
    let last_regular = POST_EXEC_INDEX - 1;
    for index in 0..last_regular {
        assert_ne!(
            roots[index],
            roots[index + 1],
            "debug_intermediateRoots after tx {}",
            index + 1
        );
    }
    assert_eq!(roots[last_regular], roots[POST_EXEC_INDEX], "debug_intermediateRoots after 0x7d");
    assert_eq!(roots.last(), Some(&fixture.state_root()), "debug_intermediateRoots last root");
    Ok(())
}
