use super::{
    debug_trace_post_exec::{
        default_trace_gas, launch_sdm_node, launch_sdm_node_with_attributes, prestate_tracer,
    },
    lagoon,
};
use alloy_consensus::{BlockHeader, TxReceipt};
use alloy_network::eip2718::Encodable2718;
use alloy_primitives::{Address, Bytes, TxKind, U256, map::HashSet};
use alloy_rpc_types_eth::{BlockNumberOrTag, TransactionInput, TransactionRequest};
use alloy_rpc_types_trace::{
    geth::{
        ChainBlockTraceResult, GethDebugTracingOptions, GethTrace, PreStateConfig, PreStateFrame,
    },
    parity::{Delta, TraceType},
};
use jsonrpsee::{
    core::client::{ClientT, Subscription, SubscriptionClientT},
    rpc_params,
};
use reth_chainspec::EthChainSpec;
use reth_e2e_test_utils::{transaction::TransactionTestContext, wallet::Wallet};
use reth_evm::ConfigureEvm;
use reth_optimism_primitives::OpTransactionSigned;
use reth_provider::{ReceiptProvider, StateProviderFactory};
use reth_rpc_api::clients::{DebugApiClient, TraceApiClient};
use reth_rpc_eth_api::{
    RpcNodeCore,
    helpers::{Call, LoadBlock, Trace},
};
use revm::inspector::NoOpInspector;
use serde_json::json;
use std::time::Duration;

#[tokio::test]
async fn later_transactions_include_prior_sdm_settlement() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let chain_spec = lagoon::chain_spec()?;
    let mut node = launch_sdm_node(&chain_spec).await?;
    let rpc = node.inner.add_ons_handle.rpc_server_handles().rpc.clone();
    let client = rpc.http_client().unwrap();
    let wallet = Wallet::default().with_chain_id(chain_spec.chain_id());
    let sender = wallet.inner.address();
    let revert_output = U256::from(42).to_be_bytes::<32>();
    let init_codes = [
        Bytes::from_static(&[0x00]),
        Bytes::new(),
        Bytes::from_static(&[0x00]),
        Bytes::from_static(&[0x60, 0x2a, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xfd]),
    ];
    for (nonce, init_code) in init_codes.into_iter().enumerate() {
        let transaction = TransactionRequest {
            chain_id: Some(wallet.chain_id),
            gas: Some(if nonce == 1 { 21_000 } else { 100_000 }),
            input: TransactionInput::new(init_code),
            max_fee_per_gas: Some(20_000_000_000),
            max_priority_fee_per_gas: Some(1_000_000_000),
            nonce: Some(nonce as u64),
            to: Some(if nonce == 1 {
                TxKind::Call(Address::repeat_byte(0x99))
            } else {
                TxKind::Create
            }),
            ..Default::default()
        };
        let transaction = TransactionTestContext::sign_tx(wallet.inner.clone(), transaction).await;
        node.rpc.inject_tx(transaction.encoded_2718().into()).await?;
    }
    let payload = node.advance_block().await?;
    let block = payload.block();
    let transactions = &block.body().transactions;
    assert_eq!(transactions.len(), 5);
    let OpTransactionSigned::PostExec(post_exec) = &transactions[4] else {
        panic!("missing post-exec transaction");
    };
    assert!(post_exec.inner().payload.gas_refund_entries.iter().any(|entry| entry.index == 2));

    assert!(!post_exec.inner().payload.gas_refund_entries.iter().any(|entry| entry.index == 1));

    let block_traces = DebugApiClient::<TransactionRequest>::debug_trace_block_by_hash(
        &client,
        block.hash(),
        None,
    )
    .await?;
    assert_eq!(block_traces.len(), transactions.len());
    let mut cumulative_gas = 0;
    for (index, transaction) in transactions[..4].iter().enumerate() {
        let hash = transaction.tx_hash();
        let receipt = node.inner.provider.receipt_by_hash(hash)?.unwrap();
        let gas_used = receipt.cumulative_gas_used() - cumulative_gas;
        cumulative_gas = receipt.cumulative_gas_used();
        let trace =
            DebugApiClient::<TransactionRequest>::debug_trace_transaction(&client, hash, None)
                .await?;
        let GethTrace::Default(frame) = trace else {
            panic!("unexpected trace");
        };
        assert_eq!(frame.gas, gas_used, "transaction {index}");
        assert_eq!(default_trace_gas(&block_traces[index]), gas_used);

        let account = DebugApiClient::<TransactionRequest>::debug_account_info_at(
            &client,
            block.hash().into(),
            index.into(),
            sender,
        )
        .await?
        .unwrap();
        let trace = DebugApiClient::<TransactionRequest>::debug_trace_transaction(
            &client,
            hash,
            Some(prestate_tracer(PreStateConfig { diff_mode: Some(true), ..Default::default() })),
        )
        .await?;
        let GethTrace::PreStateTracer(PreStateFrame::Diff(diff)) = trace else {
            panic!("unexpected prestate trace");
        };
        assert_eq!(diff.post[&sender].balance, Some(account.balance));
        let replay = TraceApiClient::<TransactionRequest>::replay_transaction(
            &client,
            hash,
            HashSet::from_iter([TraceType::StateDiff, TraceType::Trace]),
        )
        .await?;
        let call: GethTrace = client
            .request("debug_traceTransaction", rpc_params![hash, json!({"tracer": "callTracer"})])
            .await?;
        let GethTrace::CallTracer(frame) = call else {
            panic!("unexpected call trace");
        };
        assert_eq!(frame.gas_used, U256::from(gas_used));
        let state_diff = replay.state_diff.unwrap();
        let Delta::Changed(balance) = &state_diff[&sender].balance else {
            panic!("sender balance must change");
        };
        assert_eq!(balance.to, account.balance);
    }
    let revert: Option<Bytes> =
        client.request("ots_getTransactionError", rpc_params![transactions[3].tx_hash()]).await?;
    assert_eq!(revert, Some(Bytes::copy_from_slice(&revert_output)));

    let eth_api = node.rpc.inner.eth_api().clone();
    let recovered = eth_api.recovered_block(block.hash().into()).await?.unwrap();
    let recipient = Address::repeat_byte(0x55);
    let modified = eth_api
        .spawn_with_state_at_block(block.parent_hash(), move |api, mut database| {
            let mut environment =
                api.evm_config().tx_env(recovered.transactions_recovered().nth(2).unwrap());
            environment.base.gas_limit = 21_000;
            environment.base.data = Bytes::new();
            environment.base.kind = TxKind::Call(recipient);
            environment.base.value = U256::from(1);
            api.inspect_transaction_in_block(
                &recovered,
                &mut database,
                NoOpInspector {},
                2,
                environment,
                None,
            )
            .map(|(output, _)| output)
        })
        .await?;
    assert_eq!(modified.state[&recipient].info.balance, U256::from(1));
    assert_eq!(modified.result.tx_gas_used(), 21_000);

    let state = node.inner.provider.state_by_block_hash(block.hash())?;
    let sender_balance = state.account_balance(&sender)?.unwrap();
    let roots =
        DebugApiClient::<TransactionRequest>::debug_intermediate_roots(&client, block.hash(), None)
            .await?;
    assert_eq!(roots.len(), transactions.len());
    assert_eq!(roots.last(), Some(&block.state_root()));

    let balance_reader = Address::repeat_byte(0x77);
    let mut balance_code = vec![0x73];
    balance_code.extend_from_slice(sender.as_slice());
    balance_code.extend_from_slice(&[0x31, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3]);
    let bundles = json!([{"transactions": [{"from": Address::ZERO, "gasPrice": "0x4a817c800", "to": balance_reader}]}]);
    let context = json!({"blockNumber": block.hash(), "transactionIndex": 4});
    let overrides = json!({balance_reader.to_string(): {"code": Bytes::from(balance_code)}, Address::ZERO.to_string(): {"balance": "0xffffffffffffffff"}});
    let output: Vec<Vec<alloy_rpc_types_eth::EthCallResponse>> = client
        .request("eth_callMany", rpc_params![bundles.clone(), context.clone(), overrides.clone()])
        .await?;
    assert_eq!(U256::from_be_slice(output[0][0].value.as_ref().unwrap()), sender_balance);
    let traces: Vec<Vec<GethTrace>> = client
        .request(
            "debug_traceCallMany",
            rpc_params![bundles, context, json!({"stateOverrides": overrides})],
        )
        .await?;
    let GethTrace::Default(frame) = &traces[0][0] else {
        panic!("unexpected call-many trace");
    };
    assert_eq!(U256::from_be_slice(&frame.return_value), sender_balance);

    let ws = rpc.ws_client().await.unwrap();
    let mut subscription: Subscription<ChainBlockTraceResult> = ws
        .subscribe(
            "debug_subscribe",
            rpc_params![
                "traceChain",
                BlockNumberOrTag::Number(0),
                BlockNumberOrTag::Number(block.number()),
                GethDebugTracingOptions::default()
            ],
            "debug_unsubscribe",
        )
        .await?;
    let traced =
        tokio::time::timeout(Duration::from_secs(10), subscription.next()).await?.unwrap()?;
    assert_eq!(traced.traces, block_traces.into_iter().map(Some).collect::<Vec<_>>());
    subscription.unsubscribe().await?;
    let mut failed: Subscription<ChainBlockTraceResult> = ws
        .subscribe(
            "debug_subscribe",
            rpc_params![
                "traceChain",
                BlockNumberOrTag::Number(0),
                BlockNumberOrTag::Number(block.number()),
                json!({
                    "tracer": "callTracer", "tracerConfig": {"onlyTopCall": "invalid"}
                })
            ],
            "debug_unsubscribe",
        )
        .await?;
    let failure = tokio::time::timeout(Duration::from_secs(10), failed.next()).await?.unwrap()?;
    assert_eq!(failure.traces.len(), transactions.len());
    assert!(matches!(
        failure.traces[0],
        Some(alloy_rpc_types_trace::geth::TraceResult::Error { .. })
    ));
    assert!(failure.traces[1..].iter().all(Option::is_none));
    failed.unsubscribe().await?;

    Ok(())
}

#[tokio::test]
async fn later_transaction_affordability_depends_on_prior_refund() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let base = lagoon::chain_spec()?;
    let wallet = Wallet::default().with_chain_id(base.chain_id());
    let sender = wallet.inner.address();
    let gas_price = 20_000_000_000u128;
    let first_gas_used = 53_005u64;
    let transaction_gas_limit = 100_000u64;
    let initial_balance =
        U256::from(first_gas_used + transaction_gas_limit) * U256::from(gas_price);
    let mut genesis = base.genesis().clone();
    genesis.alloc.entry(sender).or_default().balance = initial_balance;
    let chain_spec = std::sync::Arc::new(
        reth_optimism_chainspec::OpChainSpecBuilder::optimism_sepolia()
            .genesis(genesis)
            .lagoon_activated()
            .build(),
    );
    let mut forced_transactions = Vec::new();
    for nonce in 0..2 {
        let transaction = TransactionRequest {
            chain_id: Some(wallet.chain_id),
            gas: Some(transaction_gas_limit),
            input: TransactionInput::new(Bytes::from_static(&[0x00])),
            max_fee_per_gas: Some(gas_price),
            max_priority_fee_per_gas: Some(gas_price),
            nonce: Some(nonce),
            to: Some(TxKind::Create),
            ..Default::default()
        };
        let transaction = TransactionTestContext::sign_tx(wallet.inner.clone(), transaction).await;
        forced_transactions.push(transaction.encoded_2718().into());
    }
    let mut node = launch_sdm_node_with_attributes(&chain_spec, move |timestamp| {
        let mut attributes = lagoon::payload_attributes(timestamp);
        // Local sequencing produces refunds; derived mode only verifies embedded refunds.
        attributes.0.no_tx_pool = Some(false);
        attributes.0.transactions = Some(forced_transactions.clone());
        attributes
    })
    .await?;
    let payload = node.advance_block().await?;
    let block = payload.block();
    assert_eq!(block.body().transactions.len(), 3);
    let client = node.inner.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();
    let sender_before_target = DebugApiClient::<TransactionRequest>::debug_account_info_at(
        &client,
        block.hash().into(),
        0usize.into(),
        sender,
    )
    .await?
    .unwrap();
    assert_eq!(
        sender_before_target.balance,
        U256::from(transaction_gas_limit) * U256::from(gas_price)
    );
    let second = block.body().transactions[1].tx_hash();
    let trace =
        DebugApiClient::<TransactionRequest>::debug_trace_transaction(&client, second, None)
            .await?;
    let GethTrace::Default(frame) = trace else {
        panic!("unexpected trace");
    };
    assert_eq!(frame.gas, first_gas_used);
    let traces = DebugApiClient::<TransactionRequest>::debug_trace_block_by_number(
        &client,
        BlockNumberOrTag::Number(block.number()),
        None,
    )
    .await?;
    assert_eq!(default_trace_gas(&traces[1]), first_gas_used);
    Ok(())
}
