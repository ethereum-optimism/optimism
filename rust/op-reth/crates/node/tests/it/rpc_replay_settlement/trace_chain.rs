//! `traceChain` over the block, streamed through `debug_subscribe`.

use super::fixture::{POST_EXEC_INDEX, SettledBlock};
use crate::lagoon::{DEPLOY_GAS, default_trace_gas};
use alloy_rpc_types_eth::BlockNumberOrTag;
use alloy_rpc_types_trace::{
    common::TraceResult,
    geth::{ChainBlockTraceResult, GethDebugTracingOptions},
};
use jsonrpsee::{
    core::client::{Subscription, SubscriptionClientT},
    rpc_params,
};
use serde_json::json;
use std::time::Duration;

/// `traceChain` streams the same settled traces as `debug_traceBlockByHash`.
#[tokio::test]
async fn trace_chain_matches_block_traces() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;
    let block_traces = fixture.block_traces().await?;

    let traced = trace_chain_over_block(&fixture, GethDebugTracingOptions::default()).await?;

    let first = traced.traces[0].as_ref().expect("traceChain traces the first transaction");
    assert_eq!(
        default_trace_gas(first),
        DEPLOY_GAS - 1,
        "traceChain gas of the refunded deployment"
    );
    assert_eq!(
        traced.traces,
        block_traces.into_iter().map(Some).collect::<Vec<_>>(),
        "traceChain traces of the block",
    );
    Ok(())
}

/// A tracer configuration the tracer rejects fails `traceChain` per block, not per subscription:
/// the first transaction carries the error and the rest of the block is left untraced.
#[tokio::test]
async fn trace_chain_reports_tracer_config_errors_per_block() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;
    let invalid_options: GethDebugTracingOptions = serde_json::from_value(
        json!({"tracer": "callTracer", "tracerConfig": {"onlyTopCall": "invalid"}}),
    )?;

    let failed = trace_chain_over_block(&fixture, invalid_options).await?;

    assert_eq!(failed.traces.len(), POST_EXEC_INDEX + 1, "traceChain trace count");
    assert!(
        matches!(failed.traces[0], Some(TraceResult::Error { .. })),
        "traceChain must report the tracer error on the first transaction: {:?}",
        failed.traces[0],
    );
    assert!(
        failed.traces[1..].iter().all(Option::is_none),
        "traceChain must leave the rest of the block untraced: {:?}",
        failed.traces,
    );
    Ok(())
}

/// Subscribes to `traceChain` over `(parent, block]` and returns the block's result.
async fn trace_chain_over_block(
    fixture: &SettledBlock,
    options: GethDebugTracingOptions,
) -> eyre::Result<ChainBlockTraceResult> {
    let ws = fixture.ws_client().await;
    let mut subscription: Subscription<ChainBlockTraceResult> = ws
        .subscribe(
            "debug_subscribe",
            rpc_params![
                "traceChain",
                BlockNumberOrTag::Number(fixture.block_number() - 1),
                BlockNumberOrTag::Number(fixture.block_number()),
                options
            ],
            "debug_unsubscribe",
        )
        .await?;
    let result =
        tokio::time::timeout(Duration::from_secs(10), subscription.next()).await?.unwrap()?;
    subscription.unsubscribe().await?;
    Ok(result)
}
