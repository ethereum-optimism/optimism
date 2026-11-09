//! Synthetic calls on a settled prefix: `debug_traceCall` at an index, `eth_callMany` and
//! `debug_traceCallMany`.

use super::fixture::{POST_EXEC_INDEX, REFUNDED_DEPLOYMENT_INDEX, SettledBlock};
use crate::lagoon::{TRANSFER_GAS, prestate_tracer};
use alloy_primitives::{Address, Bytes, U256};
use alloy_rpc_types_eth::{EthCallResponse, TransactionRequest};
use alloy_rpc_types_trace::geth::{
    GethDebugTracingCallOptions, GethTrace, PreStateConfig, PreStateFrame,
};
use jsonrpsee::{core::client::ClientT, rpc_params};
use reth_rpc_api::clients::DebugApiClient;
use serde_json::json;

/// `debug_traceCall` at an index runs on the settled prefix but earns no refund itself.
#[tokio::test]
async fn debug_trace_call_runs_on_settled_prefix_without_refund() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;

    let prefix_call = DebugApiClient::<TransactionRequest>::debug_trace_call(
        &fixture.client(),
        TransactionRequest::default().from(fixture.sender).to(Address::repeat_byte(0x55)),
        Some(fixture.block_hash().into()),
        Some(GethDebugTracingCallOptions {
            tracing_options: prestate_tracer(PreStateConfig::default()),
            tx_index: Some(1),
            ..Default::default()
        }),
    )
    .await?;
    let GethTrace::PreStateTracer(PreStateFrame::Default(prestate)) = prefix_call else {
        panic!("unexpected debug_traceCall prestate: {prefix_call:?}");
    };
    assert_eq!(
        prestate.0.get(&fixture.sender).expect("sender in prestate").balance,
        Some(fixture.replayed_balance(0, fixture.sender).await?),
        "debug_traceCall prestate after the first deployment",
    );

    let synthetic_transfer = DebugApiClient::<TransactionRequest>::debug_trace_call(
        &fixture.client(),
        TransactionRequest::default()
            .from(fixture.sender)
            .to(Address::repeat_byte(0x55))
            .value(U256::from(1)),
        Some(fixture.block_hash().into()),
        // At the refunded slot, a refund entry leaking into the synthetic call would show as gas
        // below the intrinsic cost.
        Some(GethDebugTracingCallOptions {
            tx_index: Some(REFUNDED_DEPLOYMENT_INDEX as u64),
            ..Default::default()
        }),
    )
    .await?;
    let GethTrace::Default(synthetic_frame) = synthetic_transfer else {
        panic!("unexpected debug_traceCall trace: {synthetic_transfer:?}");
    };
    assert_eq!(synthetic_frame.gas, TRANSFER_GAS, "debug_traceCall gas of a synthetic transfer");
    Ok(())
}

/// `eth_callMany` and `debug_traceCallMany` run on the settled prefix: a contract reading the
/// sender's balance after the whole block sees the committed one.
#[tokio::test]
async fn call_many_reads_settled_balances() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;
    let committed = fixture.committed_balance(fixture.sender)?;

    let balance_reader = Address::repeat_byte(0x77);
    let mut balance_code = vec![0x73];
    balance_code.extend_from_slice(fixture.sender.as_slice());
    // BALANCE PUSH1 0 MSTORE PUSH1 32 PUSH1 0 RETURN
    balance_code.extend_from_slice(&[0x31, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3]);
    let bundles = json!([{
        "transactions": [{"from": Address::ZERO, "gasPrice": "0x4a817c800", "to": balance_reader}]
    }]);
    let context = json!({"blockNumber": fixture.block_hash(), "transactionIndex": POST_EXEC_INDEX});
    let overrides = json!({
        balance_reader.to_string(): {"code": Bytes::from(balance_code)},
        Address::ZERO.to_string(): {"balance": "0xffffffffffffffff"},
    });

    let output: Vec<Vec<EthCallResponse>> = fixture
        .client()
        .request("eth_callMany", rpc_params![bundles.clone(), context.clone(), overrides.clone()])
        .await?;
    let response = &output[0][0];
    assert!(response.error.is_none(), "eth_callMany failed: {:?}", response.error);
    assert_eq!(
        U256::from_be_slice(response.value.as_ref().expect("a successful call returns data")),
        committed,
        "eth_callMany balance read",
    );

    let traces: Vec<Vec<GethTrace>> = fixture
        .client()
        .request(
            "debug_traceCallMany",
            rpc_params![bundles, context, json!({"stateOverrides": overrides})],
        )
        .await?;
    let GethTrace::Default(frame) = &traces[0][0] else {
        panic!("unexpected debug_traceCallMany trace: {:?}", traces[0][0]);
    };
    assert_eq!(
        U256::from_be_slice(&frame.return_value),
        committed,
        "debug_traceCallMany balance read",
    );
    Ok(())
}
