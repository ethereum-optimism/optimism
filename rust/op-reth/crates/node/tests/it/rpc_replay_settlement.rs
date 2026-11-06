//! RPC replay over a sequenced Lagoon block whose transactions carry SDM refunds.
//!
//! The settlement of an SDM refund (canonical gas, sender credit, fee-vault debits) only happens
//! inside the block executor, so every RPC path that replays block transactions must go through it.
//! A replay on the bare EVM reports pre-refund gas and unsettled balances ([#485]).
//!
//! [#485]: https://github.com/ethereum-optimism/protocol-team/issues/485

use crate::lagoon::{
    self, DEPLOY_GAS, default_trace_gas, launch_sdm_node, launch_sdm_node_with_attributes,
    prestate_tracer,
};
use alloy_consensus::{BlockHeader, TxReceipt};
use alloy_network::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, TxKind, U256, map::HashSet};
use alloy_rpc_types_eth::{
    BlockNumberOrTag, EthCallResponse, TransactionInput, TransactionRequest,
};
use alloy_rpc_types_trace::{
    common::TraceResult,
    filter::TraceFilter,
    geth::{
        ChainBlockTraceResult, GethDebugTracingCallOptions, GethDebugTracingOptions, GethTrace,
        PreStateConfig, PreStateFrame,
    },
    parity::{Delta, StateDiff, TraceType},
};
use jsonrpsee::{
    core::client::{ClientT, Subscription, SubscriptionClientT},
    http_client::HttpClient,
    rpc_params,
};
use reth_chainspec::EthChainSpec;
use reth_e2e_test_utils::{NodeHelperType, transaction::TransactionTestContext, wallet::Wallet};
use reth_optimism_node::OpNode;
use reth_optimism_payload_builder::OpBuiltPayload;
use reth_optimism_primitives::OpTransactionSigned;
use reth_provider::{ReceiptProvider, StateProviderFactory};
use reth_rpc_api::clients::{DebugApiClient, TraceApiClient};
use serde_json::json;
use std::time::Duration;

/// Index of the post-exec transaction in a [`SettledBlock`].
const POST_EXEC_INDEX: usize = 4;

/// Revert data returned by the reverting deployment in a [`SettledBlock`].
const REVERT_OUTPUT: U256 = U256::from_limbs([42, 0, 0, 0]);

/// A Lagoon block sequenced by a node running the fixed test policy, which refunds every regular
/// transaction one gas: a real, imported block with a 0x7d transaction and a settled state.
///
/// The sender submits four transactions: a deployment of the init code `STOP` (refunded), a plain
/// transfer (not refunded, since it already sits on the EIP-7623 floor), a second deployment
/// (refunded) and a deployment whose init code reverts with [`REVERT_OUTPUT`].
struct SettledBlock {
    node: NodeHelperType<OpNode>,
    client: HttpClient,
    payload: OpBuiltPayload,
    sender: Address,
}

impl SettledBlock {
    async fn sequence() -> eyre::Result<Self> {
        let chain_spec = lagoon::chain_spec()?;
        let mut node = launch_sdm_node(&chain_spec).await?;
        let client = node.inner.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();
        let wallet = Wallet::default().with_chain_id(chain_spec.chain_id());
        let sender = wallet.inner.address();
        let deployments = [
            (TxKind::Create, Bytes::from_static(&[0x00])),
            (TxKind::Call(Address::repeat_byte(0x99)), Bytes::new()),
            (TxKind::Create, Bytes::from_static(&[0x00])),
            // PUSH1 42 PUSH1 0 MSTORE PUSH1 32 PUSH1 0 REVERT
            (
                TxKind::Create,
                Bytes::from_static(&[0x60, 0x2a, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xfd]),
            ),
        ];
        for (nonce, (to, init_code)) in deployments.into_iter().enumerate() {
            let transaction = TransactionRequest {
                chain_id: Some(wallet.chain_id),
                gas: Some(if matches!(to, TxKind::Call(_)) { 21_000 } else { 100_000 }),
                input: TransactionInput::new(init_code),
                max_fee_per_gas: Some(20_000_000_000),
                max_priority_fee_per_gas: Some(1_000_000_000),
                nonce: Some(nonce as u64),
                to: Some(to),
                ..Default::default()
            };
            let signed = TransactionTestContext::sign_tx(wallet.inner.clone(), transaction).await;
            node.rpc.inject_tx(signed.encoded_2718().into()).await?;
        }
        let payload = node.advance_block().await?;

        let fixture = Self { node, client, payload, sender };
        let transactions = fixture.transactions();
        assert_eq!(transactions.len(), 5, "four regular transactions and the post-exec tx");
        let OpTransactionSigned::PostExec(post_exec) = &transactions[POST_EXEC_INDEX] else {
            panic!("block must end in a post-exec tx, got {:?}", transactions[POST_EXEC_INDEX]);
        };
        let refund_entries = &post_exec.inner().payload.gas_refund_entries;
        assert!(
            refund_entries.iter().any(|entry| entry.index == 2 && entry.gas_refund > 0),
            "the second deployment must be refunded: {refund_entries:?}",
        );
        assert!(
            !refund_entries.iter().any(|entry| entry.index == 1),
            "the transfer must not be refunded: {refund_entries:?}",
        );
        Ok(fixture)
    }

    fn block_hash(&self) -> B256 {
        self.payload.block().hash()
    }

    fn block_number(&self) -> u64 {
        self.payload.block().number()
    }

    fn transactions(&self) -> &[OpTransactionSigned] {
        &self.payload.block().body().transactions
    }

    /// Gas each regular transaction used according to its receipt.
    fn receipt_gas(&self) -> eyre::Result<Vec<u64>> {
        let mut cumulative_gas = 0;
        self.transactions()
            .iter()
            .map(|transaction| {
                let receipt =
                    self.node.inner.provider.receipt_by_hash(transaction.tx_hash())?.unwrap();
                let gas_used = receipt.cumulative_gas_used() - cumulative_gas;
                cumulative_gas = receipt.cumulative_gas_used();
                Ok(gas_used)
            })
            .collect()
    }

    /// Balance of `address` in the state the node committed for the block.
    fn committed_balance(&self, address: Address) -> eyre::Result<U256> {
        let state = self.node.inner.provider.state_by_block_hash(self.block_hash())?;
        Ok(state.account_balance(&address)?.unwrap_or_default())
    }

    /// Balance of `address` right after the transaction at `index`, as `debug_accountAt` replays
    /// it.
    async fn replayed_balance(&self, index: usize, address: Address) -> eyre::Result<U256> {
        let account = DebugApiClient::<TransactionRequest>::debug_account_info_at(
            &self.client,
            self.block_hash().into(),
            index.into(),
            address,
        )
        .await?
        .unwrap_or_default();
        Ok(account.balance)
    }

    async fn block_traces(&self) -> eyre::Result<Vec<TraceResult<GethTrace, String>>> {
        Ok(DebugApiClient::<TransactionRequest>::debug_trace_block_by_hash(
            &self.client,
            self.block_hash(),
            None,
        )
        .await?)
    }

    async fn trace_transaction(
        &self,
        index: usize,
        options: Option<GethDebugTracingOptions>,
    ) -> eyre::Result<GethTrace> {
        Ok(DebugApiClient::<TransactionRequest>::debug_trace_transaction(
            &self.client,
            self.transactions()[index].tx_hash(),
            options,
        )
        .await?)
    }

    async fn prestate_diff(
        &self,
        index: usize,
    ) -> eyre::Result<alloy_rpc_types_trace::geth::DiffMode> {
        let options =
            prestate_tracer(PreStateConfig { diff_mode: Some(true), ..Default::default() });
        match self.trace_transaction(index, Some(options)).await? {
            GethTrace::PreStateTracer(PreStateFrame::Diff(diff)) => Ok(diff),
            other => panic!("unexpected prestate trace of transaction {index}: {other:?}"),
        }
    }
}

/// Every RPC path that replays the block's own transactions agrees with what the node committed:
/// the receipts' canonical gas, the balances after each transaction, and the header's state root.
#[tokio::test]
async fn replays_match_sdm_settlement() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;
    let SettledBlock { client, sender, .. } = &fixture;
    let client = client.clone();
    let sender = *sender;
    let transactions = fixture.transactions().to_vec();
    let block_hash = fixture.block_hash();
    let receipt_gas = fixture.receipt_gas()?;

    // Pins the settlement itself: every check below compares a replay with what the node committed.
    assert_eq!(receipt_gas[0], DEPLOY_GAS - 1, "receipt holds canonical gas");

    let block_traces = fixture.block_traces().await?;
    assert_eq!(block_traces.len(), transactions.len(), "debug_traceBlockByHash trace count");

    for index in 0..POST_EXEC_INDEX {
        let hash = transactions[index].tx_hash();
        let gas_used = receipt_gas[index];

        let GethTrace::Default(frame) = fixture.trace_transaction(index, None).await? else {
            panic!("unexpected default trace of transaction {index}");
        };
        assert_eq!(frame.gas, gas_used, "debug_traceTransaction gas of transaction {index}");
        assert_eq!(
            default_trace_gas(&block_traces[index]),
            gas_used,
            "debug_traceBlockByHash gas of transaction {index}",
        );
        let call: GethTrace = client
            .request("debug_traceTransaction", rpc_params![hash, json!({"tracer": "callTracer"})])
            .await?;
        let GethTrace::CallTracer(call_frame) = call else {
            panic!("unexpected call trace of transaction {index}: {call:?}");
        };
        assert_eq!(
            call_frame.gas_used,
            U256::from(gas_used),
            "callTracer gas of transaction {index}",
        );

        // Every balance the prestate diff ends on is the balance debug_accountAt replays at that
        // index, for the sender as well as the fee vaults.
        let diff = fixture.prestate_diff(index).await?;
        assert!(
            diff.post.contains_key(&sender),
            "prestate diff of transaction {index} lacks sender"
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

        let replay = TraceApiClient::<TransactionRequest>::replay_transaction(
            &client,
            hash,
            HashSet::from_iter([TraceType::StateDiff, TraceType::Trace]),
        )
        .await?;
        assert_eq!(
            settled_balance(replay.state_diff.as_ref(), sender),
            fixture.replayed_balance(index, sender).await?,
            "trace_replayTransaction state diff of transaction {index}",
        );
    }

    // The last regular transaction leaves the committed state, since the post-exec transaction
    // changes nothing; this anchors the replayed balances above to the provider.
    let last_regular = POST_EXEC_INDEX - 1;
    let committed = fixture.committed_balance(sender)?;
    assert_eq!(
        fixture.replayed_balance(last_regular, sender).await?,
        committed,
        "debug_accountAt after the last regular transaction",
    );
    assert_eq!(
        fixture.replayed_balance(POST_EXEC_INDEX, sender).await?,
        committed,
        "debug_accountAt after the post-exec transaction",
    );
    for (address, account) in &fixture.prestate_diff(last_regular).await?.post {
        if let Some(balance) = account.balance {
            assert_eq!(
                balance,
                fixture.committed_balance(*address)?,
                "committed balance of {address} after transaction {last_regular}",
            );
        }
    }

    let replays = TraceApiClient::<TransactionRequest>::replay_block_transactions(
        &client,
        block_hash.into(),
        HashSet::from_iter([TraceType::StateDiff]),
    )
    .await?
    .expect("block exists");
    assert_eq!(replays.len(), transactions.len(), "trace_replayBlockTransactions replay count");
    for (index, replay) in replays.iter().take(POST_EXEC_INDEX).enumerate() {
        assert_eq!(
            settled_balance(replay.full_trace.state_diff.as_ref(), sender),
            fixture.replayed_balance(index, sender).await?,
            "trace_replayBlockTransactions state diff of transaction {index}",
        );
    }

    // debug_traceCall at an index runs on the settled prefix but earns no refund itself.
    let prefix_call = DebugApiClient::<TransactionRequest>::debug_trace_call(
        &client,
        TransactionRequest::default().from(sender).to(Address::with_last_byte(1)),
        Some(block_hash.into()),
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
        prestate.0[&sender].balance,
        Some(fixture.replayed_balance(0, sender).await?),
        "debug_traceCall prestate after the first deployment",
    );
    let recipient = Address::repeat_byte(0x55);
    let synthetic_transfer = DebugApiClient::<TransactionRequest>::debug_trace_call(
        &client,
        TransactionRequest::default().from(sender).to(recipient).value(U256::from(1)),
        Some(block_hash.into()),
        Some(GethDebugTracingCallOptions {
            tx_index: Some(POST_EXEC_INDEX as u64),
            ..Default::default()
        }),
    )
    .await?;
    let GethTrace::Default(synthetic_frame) = synthetic_transfer else {
        panic!("unexpected debug_traceCall trace: {synthetic_transfer:?}");
    };
    assert_eq!(synthetic_frame.gas, 21_000, "debug_traceCall gas of a synthetic transfer");

    // The post-exec transaction is its own replay target: an empty trace at its receipt gas.
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
    let post_exec_call: GethTrace = client
        .request(
            "debug_traceTransaction",
            rpc_params![transactions[POST_EXEC_INDEX].tx_hash(), json!({"tracer": "callTracer"})],
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

    // Each regular transaction moves the state root; the post-exec transaction does not, and the
    // last root is the header's.
    let roots =
        DebugApiClient::<TransactionRequest>::debug_intermediate_roots(&client, block_hash, None)
            .await?;
    assert_eq!(roots.len(), transactions.len(), "debug_intermediateRoots root count");
    for index in 0..last_regular {
        assert_ne!(
            roots[index],
            roots[index + 1],
            "debug_intermediateRoots after tx {}",
            index + 1
        );
    }
    assert_eq!(roots[last_regular], roots[POST_EXEC_INDEX], "debug_intermediateRoots after 0x7d");
    assert_eq!(
        roots.last(),
        Some(&fixture.payload.block().state_root()),
        "debug_intermediateRoots"
    );

    let filtered = TraceApiClient::<TransactionRequest>::trace_filter(
        &client,
        TraceFilter {
            from_block: Some(fixture.block_number()),
            to_block: Some(fixture.block_number()),
            ..Default::default()
        },
    )
    .await?;
    let filtered_hashes: HashSet<_> =
        filtered.iter().filter_map(|trace| trace.transaction_hash).collect();
    let block_hashes: HashSet<_> = transactions.iter().map(|tx| tx.tx_hash()).collect();
    assert_eq!(filtered_hashes, block_hashes, "trace_filter covers every transaction");

    // eth_callMany and debug_traceCallMany run on the settled prefix: a contract reading the
    // sender's balance after the whole block sees the committed one.
    let balance_reader = Address::repeat_byte(0x77);
    let mut balance_code = vec![0x73];
    balance_code.extend_from_slice(sender.as_slice());
    // BALANCE PUSH1 0 MSTORE PUSH1 32 PUSH1 0 RETURN
    balance_code.extend_from_slice(&[0x31, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3]);
    let bundles = json!([{
        "transactions": [{"from": Address::ZERO, "gasPrice": "0x4a817c800", "to": balance_reader}]
    }]);
    let context = json!({"blockNumber": block_hash, "transactionIndex": POST_EXEC_INDEX});
    let overrides = json!({
        balance_reader.to_string(): {"code": Bytes::from(balance_code)},
        Address::ZERO.to_string(): {"balance": "0xffffffffffffffff"},
    });
    let output: Vec<Vec<EthCallResponse>> = client
        .request("eth_callMany", rpc_params![bundles.clone(), context.clone(), overrides.clone()])
        .await?;
    assert_eq!(
        U256::from_be_slice(output[0][0].value.as_ref().unwrap()),
        committed,
        "eth_callMany balance read",
    );
    let traces: Vec<Vec<GethTrace>> = client
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

/// `traceChain` streams the same settled traces as `debug_traceBlockByHash`.
#[tokio::test]
async fn trace_chain_matches_block_traces() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;
    let block_traces = fixture.block_traces().await?;

    let ws = fixture.node.inner.add_ons_handle.rpc_server_handles().rpc.ws_client().await.unwrap();
    let mut subscription: Subscription<ChainBlockTraceResult> = ws
        .subscribe(
            "debug_subscribe",
            rpc_params![
                "traceChain",
                BlockNumberOrTag::Number(0),
                BlockNumberOrTag::Number(fixture.block_number()),
                GethDebugTracingOptions::default()
            ],
            "debug_unsubscribe",
        )
        .await?;
    let traced =
        tokio::time::timeout(Duration::from_secs(10), subscription.next()).await?.unwrap()?;
    subscription.unsubscribe().await?;

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

    let ws = fixture.node.inner.add_ons_handle.rpc_server_handles().rpc.ws_client().await.unwrap();
    let mut subscription: Subscription<ChainBlockTraceResult> = ws
        .subscribe(
            "debug_subscribe",
            rpc_params![
                "traceChain",
                BlockNumberOrTag::Number(0),
                BlockNumberOrTag::Number(fixture.block_number()),
                json!({"tracer": "callTracer", "tracerConfig": {"onlyTopCall": "invalid"}})
            ],
            "debug_unsubscribe",
        )
        .await?;
    let failed =
        tokio::time::timeout(Duration::from_secs(10), subscription.next()).await?.unwrap()?;
    subscription.unsubscribe().await?;

    assert_eq!(failed.traces.len(), fixture.transactions().len(), "traceChain trace count");
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

/// Otterscan replays a reverted transaction on an SDM block and returns its revert data.
#[tokio::test]
async fn otterscan_returns_revert_output_on_sdm_block() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let fixture = SettledBlock::sequence().await?;

    let revert: Option<Bytes> = fixture
        .client
        .request("ots_getTransactionError", rpc_params![fixture.transactions()[3].tx_hash()])
        .await?;

    assert_eq!(
        revert,
        Some(Bytes::copy_from_slice(&REVERT_OUTPUT.to_be_bytes::<32>())),
        "ots_getTransactionError revert data",
    );
    Ok(())
}

/// The sender starts with exactly the settled cost of the first deployment plus the second
/// transaction's upfront `gas_limit * max_fee`, so the second transaction replays only if the
/// first one's refund was credited; a bare-EVM replay fails with insufficient funds.
#[tokio::test]
async fn later_transaction_affordability_depends_on_prior_refund() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();
    let wallet = Wallet::default();
    let sender = wallet.inner.address();
    let gas_price = 20_000_000_000u128;
    let settled_deploy_gas = DEPLOY_GAS - 1;
    let transaction_gas_limit = 100_000u64;
    let initial_balance =
        U256::from(settled_deploy_gas + transaction_gas_limit) * U256::from(gas_price);
    let chain_spec = lagoon::chain_spec_with(|genesis| {
        genesis.alloc.entry(sender).or_default().balance = initial_balance;
    })?;
    let wallet = wallet.with_chain_id(chain_spec.chain_id());

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
        let signed = TransactionTestContext::sign_tx(wallet.inner.clone(), transaction).await;
        forced_transactions.push(signed.encoded_2718().into());
    }
    let mut node = launch_sdm_node_with_attributes(&chain_spec, move |timestamp| {
        let mut attributes = lagoon::payload_attributes(timestamp);
        // Local sequencing produces refunds; derived mode only verifies embedded refunds. The
        // transactions are forced through attributes because the pool's balance check does not
        // know about refunds and would reject the second one.
        attributes.0.no_tx_pool = Some(false);
        attributes.0.transactions = Some(forced_transactions.clone());
        attributes
    })
    .await?;
    let payload = node.advance_block().await?;
    let block = payload.block();
    assert_eq!(block.body().transactions.len(), 3, "two deployments and the post-exec tx");
    let client = node.inner.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();

    let sender_before_second = DebugApiClient::<TransactionRequest>::debug_account_info_at(
        &client,
        block.hash().into(),
        0usize.into(),
        sender,
    )
    .await?
    .unwrap();
    assert_eq!(
        sender_before_second.balance,
        U256::from(transaction_gas_limit) * U256::from(gas_price),
        "debug_accountAt balance after the first deployment",
    );
    let second = block.body().transactions[1].tx_hash();
    let trace =
        DebugApiClient::<TransactionRequest>::debug_trace_transaction(&client, second, None)
            .await?;
    let GethTrace::Default(frame) = trace else {
        panic!("unexpected trace of the second deployment: {trace:?}");
    };
    assert_eq!(
        frame.gas, settled_deploy_gas,
        "debug_traceTransaction gas of the second deployment"
    );
    let traces = DebugApiClient::<TransactionRequest>::debug_trace_block_by_number(
        &client,
        BlockNumberOrTag::Number(block.number()),
        None,
    )
    .await?;
    assert_eq!(
        default_trace_gas(&traces[1]),
        settled_deploy_gas,
        "debug_traceBlockByNumber gas of the second deployment",
    );
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
