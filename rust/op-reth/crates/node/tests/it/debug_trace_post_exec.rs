//! RPC replay over blocks containing an SDM post-exec (0x7d) transaction.
//!
//! The settlement of an SDM refund (canonical gas, sender credit, fee-vault debits) only happens
//! inside the block executor, so every RPC path that replays block transactions must go through it.
//! A replay on the bare EVM reports pre-refund gas and unsettled balances ([#485]).
//!
//! Tracing must also not fail whole-block on the 0x7d tx itself ([premium#163]): the block executor
//! and `OpEvm::transact_raw` both give it the consensus-defined no-op result.
//!
//! [#485]: https://github.com/ethereum-optimism/protocol-team/issues/485
//! [premium#163]: https://github.com/ethereum-optimism/optimism-premium/issues/163

use crate::lagoon;
use alloy_consensus::{
    Block, BlockBody, BlockHeader, Header, Sealable, SignableTransaction, TxEip1559, TxReceipt,
};
use alloy_network::eip2718::Encodable2718;
use alloy_primitives::{Address, B256, Bytes, Signature, TxKind, U256, map::HashSet};
use alloy_rpc_types_eth::{BlockNumberOrTag, TransactionInput, TransactionRequest};
use alloy_rpc_types_trace::{
    common::TraceResult,
    geth::{
        GethDebugBuiltInTracerType, GethDebugTracerConfig, GethDebugTracerType,
        GethDebugTracingCallOptions, GethDebugTracingOptions, GethTrace, PreStateConfig,
        PreStateFrame,
    },
    parity::{Delta, StateDiff, TraceType},
};
use op_alloy_consensus::{OpTxEnvelope, SDMGasEntry, TxDeposit, build_post_exec_tx};
use reth_chainspec::EthChainSpec;
use reth_e2e_test_utils::{
    node::NodeTestContext, transaction::TransactionTestContext, wallet::Wallet,
};
use reth_node_builder::{NodeBuilder, NodeHandle};
use reth_node_core::{
    args::{NetworkArgs, RpcServerArgs},
    node_config::NodeConfig,
};
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
use reth_optimism_node::{OpNode, args::RollupArgs};
use reth_optimism_primitives::OpTransactionSigned;
use reth_provider::{ReceiptProvider, StateProviderFactory};
use reth_rpc_api::clients::{DebugApiClient, TraceApiClient};
use reth_rpc_server_types::RpcModuleSelection;
use reth_tasks::Runtime;
use std::sync::Arc;

/// Gas used by a plain transfer before the SDM refund.
const TRANSFER_GAS: u64 = 21_000;

/// Gas used before the SDM refund by a deployment of the one-byte init code `STOP`: intrinsic
/// 21000, CREATE 32000, one zero calldata byte 4, one init code word 2, and no code deposit.
const DEPLOY_GAS: u64 = 53_006;

/// `debug_traceBlock` over a raw block that ends in a post-exec (0x7d) transaction must return
/// one successful trace per transaction while preserving diagnostic pre-refund results. Before the
/// `transact_raw` short-circuit this failed for the whole block with `-32000: invalid chain ID`.
///
/// The refund takes the transfer below the EIP-7623 floor: the producer never refunds that far,
/// but a verifier applies whatever refund the payload carries.
#[rstest::rstest]
#[case::exhausted_block_budget(21_000)]
#[case::normal_block_budget(30_000_000)]
#[tokio::test]
async fn test_debug_trace_block_with_post_exec_tx(
    #[case] block_gas_limit: u64,
) -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let chain_spec = Arc::new(OpChainSpecBuilder::optimism_sepolia().lagoon_activated().build());

    let NodeHandle { node, node_exit_future: _ } = NodeBuilder::new(rpc_node_config(&chain_spec))
        .testing_node(Runtime::test())
        .node(OpNode::default())
        .launch()
        .await?;
    let client = node.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();

    // `debug_traceBlock` replays a caller-supplied raw block on its parent's state, so the block
    // only has to decode and recover senders — consensus fields like the state root are never
    // validated. Replaying on genesis state keeps the fixture self-contained: no SDM production
    // machinery is needed to put a 0x7d tx in a block.
    let genesis = chain_spec.genesis_header();
    let header = Header {
        parent_hash: chain_spec.genesis_hash(),
        number: 1,
        timestamp: genesis.timestamp + 2,
        gas_limit: block_gas_limit,
        base_fee_per_gas: genesis.base_fee_per_gas,
        // Post-Ecotone replay runs the EIP-4788 pre-block system call, which requires a parent
        // beacon block root; the blob-gas and withdrawals fields must be present for the later
        // optionals to round-trip through RLP.
        withdrawals_root: Some(alloy_consensus::EMPTY_ROOT_HASH),
        blob_gas_used: Some(0),
        excess_blob_gas: Some(0),
        parent_beacon_block_root: Some(B256::ZERO),
        ..Default::default()
    };

    // Fund the regular transaction's recovered sender with a preceding deposit. A valid version-1
    // post-exec payload must be non-empty and may only target regular transactions.
    let regular = TxEip1559 {
        chain_id: chain_spec.chain_id(),
        // The preceding deposit increments the depositor nonce after Regolith.
        nonce: 1,
        gas_limit: TRANSFER_GAS,
        max_fee_per_gas: u128::from(header.base_fee_per_gas.unwrap_or_default()),
        to: TxKind::Call(Address::with_last_byte(1)),
        ..Default::default()
    }
    .into_signed(Signature::test_signature());
    let signer = regular.recover_signer()?;
    let deposit = TxDeposit {
        source_hash: B256::with_last_byte(1),
        from: signer,
        to: TxKind::Call(Address::with_last_byte(1)),
        mint: 1_000_000_000_000_000_000,
        value: U256::ZERO,
        gas_limit: 21_000,
        is_system_transaction: false,
        input: Bytes::new(),
    };
    let transactions = vec![
        OpTxEnvelope::Deposit(deposit.seal_slow()),
        OpTxEnvelope::Eip1559(regular),
        OpTxEnvelope::PostExec(
            build_post_exec_tx(1, vec![SDMGasEntry { index: 1, gas_refund: 1 }]).seal_slow(),
        ),
    ];
    let block = Block::new(header, BlockBody { transactions, ommers: vec![], withdrawals: None });

    let traces = DebugApiClient::<TransactionRequest>::debug_trace_block(
        &client,
        alloy_rlp::encode(&block).into(),
        Some(GethDebugTracingOptions::default()),
    )
    .await?;

    assert_eq!(traces.len(), 3, "one trace per transaction, including the post-exec tx");
    for (index, trace) in traces.iter().enumerate() {
        assert!(
            matches!(trace, TraceResult::Success { .. }),
            "transaction {index} must trace successfully, got {trace:?}",
        );
    }
    assert_eq!(default_trace_gas(&traces[1]), TRANSFER_GAS, "raw block traces are pre-refund");

    Ok(())
}

/// Every RPC path that replays a block's transactions must agree with what the chain committed for
/// an SDM-refunded transaction: the receipt's canonical gas, the post-settlement balances, and the
/// header's state root.
///
/// The block is sequenced by a Lagoon node running the fixed test policy, which refunds every
/// regular transaction one gas, so it is a real, imported block with a 0x7d tx and a settled state.
/// The refunded transaction is a contract deployment: the producer never refunds below the EIP-7623
/// floor, which a plain transfer already sits on.
#[tokio::test]
async fn test_rpc_replay_matches_sdm_settlement() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let chain_spec = lagoon::chain_spec()?;
    let mut node = launch_sdm_node(&chain_spec).await?;
    let client = node.inner.add_ons_handle.rpc_server_handles().rpc.http_client().unwrap();

    let wallet = Wallet::default().with_chain_id(chain_spec.chain_id());
    let sender = wallet.inner.address();
    // Init code `STOP` deploys an empty contract.
    let deploy = TransactionRequest {
        chain_id: Some(wallet.chain_id),
        nonce: Some(0),
        gas: Some(100_000),
        max_fee_per_gas: Some(20_000_000_000),
        max_priority_fee_per_gas: Some(1_000_000_000),
        to: Some(TxKind::Create),
        input: TransactionInput::new(Bytes::from_static(&[0x00])),
        ..Default::default()
    };
    let deploy = TransactionTestContext::sign_tx(wallet.inner, deploy).await;
    node.rpc.inject_tx(deploy.encoded_2718().into()).await?;
    let payload = node.advance_block().await?;
    let block = payload.block();

    // The fixture must be what the assertions below assume: a refunded deployment, then the 0x7d.
    let transactions = &block.body().transactions;
    assert_eq!(transactions.len(), 2, "deployment and post-exec tx, got {transactions:?}");
    let OpTransactionSigned::PostExec(post_exec) = &transactions[1] else {
        panic!("block must end in a post-exec tx, got {:?}", transactions[1]);
    };
    assert_eq!(
        post_exec.inner().payload.gas_refund_entries,
        vec![SDMGasEntry { index: 0, gas_refund: 1 }],
    );
    let tx_hash = transactions[0].tx_hash();
    let canonical_gas = node
        .inner
        .provider
        .receipt_by_hash(tx_hash)?
        .expect("deployment receipt")
        .cumulative_gas_used();
    // Pins the settlement itself: every check below compares a replay with what the node committed.
    assert_eq!(canonical_gas, DEPLOY_GAS - 1, "receipt holds canonical gas");
    let state = node.inner.provider.state_by_block_hash(block.hash())?;
    let committed_balance = |address| -> eyre::Result<U256> {
        Ok(state.account_balance(&address)?.unwrap_or_default())
    };

    // debug_traceTransaction: canonical gas and a settled prestate diff.
    let trace =
        DebugApiClient::<TransactionRequest>::debug_trace_transaction(&client, tx_hash, None)
            .await?;
    let GethTrace::Default(frame) = trace else { panic!("unexpected trace shape: {trace:?}") };
    assert_eq!(frame.gas, canonical_gas, "debug_traceTransaction gas");

    let trace = DebugApiClient::<TransactionRequest>::debug_trace_transaction(
        &client,
        tx_hash,
        Some(prestate_tracer(PreStateConfig { diff_mode: Some(true), ..Default::default() })),
    )
    .await?;
    let GethTrace::PreStateTracer(PreStateFrame::Diff(diff)) = trace else {
        panic!("unexpected trace shape: {trace:?}")
    };
    // Sender, block beneficiary and base-fee vault all change, and each must land on the committed
    // balance; the 0x7d tx after the deployment leaves state untouched.
    assert_eq!(
        diff.post[&sender].balance,
        Some(committed_balance(sender)?),
        "prestate diff balance of the sender",
    );
    for (address, account) in &diff.post {
        if let Some(balance) = account.balance {
            assert_eq!(balance, committed_balance(*address)?, "prestate diff balance of {address}");
        }
    }

    // debug_traceBlockByNumber: canonical gas, and the 0x7d tx still traces.
    let traces = DebugApiClient::<TransactionRequest>::debug_trace_block_by_number(
        &client,
        BlockNumberOrTag::Number(block.number()),
        None,
    )
    .await?;
    assert_eq!(traces.len(), 2, "one trace per transaction, including the post-exec tx");
    assert_eq!(default_trace_gas(&traces[0]), canonical_gas, "debug_traceBlockByNumber gas");
    assert!(matches!(traces[1], TraceResult::Success { .. }), "0x7d tx trace: {:?}", traces[1]);

    // debug_traceCall after the deployment: the replayed prefix (shared with eth_callMany and
    // debug_traceCallMany) must leave the sender with its settled balance.
    let trace = DebugApiClient::<TransactionRequest>::debug_trace_call(
        &client,
        TransactionRequest::default().from(sender).to(Address::with_last_byte(1)),
        Some(block.hash().into()),
        Some(GethDebugTracingCallOptions {
            tracing_options: prestate_tracer(PreStateConfig::default()),
            tx_index: Some(1),
            ..Default::default()
        }),
    )
    .await?;
    let GethTrace::PreStateTracer(PreStateFrame::Default(prestate)) = trace else {
        panic!("unexpected trace shape: {trace:?}")
    };
    assert_eq!(
        prestate.0[&sender].balance,
        Some(committed_balance(sender)?),
        "debug_traceCall prestate after the deployment",
    );

    // debug_intermediateRoots: the root after the 0x7d tx is the header's.
    let roots =
        DebugApiClient::<TransactionRequest>::debug_intermediate_roots(&client, block.hash(), None)
            .await?;
    assert_eq!(roots.len(), 2, "one root per transaction");
    assert_eq!(roots[1], block.state_root(), "debug_intermediateRoots");

    // trace_replayTransaction and trace_replayBlockTransactions: settled state diffs.
    let state_diff = HashSet::from_iter([TraceType::StateDiff]);
    let results = TraceApiClient::<TransactionRequest>::replay_transaction(
        &client,
        tx_hash,
        state_diff.clone(),
    )
    .await?;
    assert_state_diff_settled(results.state_diff.as_ref(), sender, &committed_balance)?;
    let results = TraceApiClient::<TransactionRequest>::replay_block_transactions(
        &client,
        block.hash().into(),
        state_diff,
    )
    .await?
    .expect("block exists");
    assert_eq!(results.len(), 2, "one replay per transaction, including the post-exec tx");
    assert_state_diff_settled(
        results[0].full_trace.state_diff.as_ref(),
        sender,
        &committed_balance,
    )?;

    Ok(())
}

/// Returns a test node config serving every RPC module over HTTP on unused ports.
pub(super) fn rpc_node_config(chain_spec: &Arc<OpChainSpec>) -> NodeConfig<OpChainSpec> {
    let mut network_args = NetworkArgs::default().with_unused_ports();
    network_args.discovery.discv5_port = Some(0);
    network_args.discovery.discv5_port_ipv6 = Some(0);
    NodeConfig::test().map_chain(chain_spec.clone()).with_network(network_args).with_rpc(
        RpcServerArgs::default()
            .with_unused_ports()
            .with_http()
            // `debug_*` is not part of the default HTTP module selection.
            .with_http_api(RpcModuleSelection::All)
            .with_ws()
            .with_ws_api(RpcModuleSelection::All),
    )
}

/// Returns tracing options selecting the prestate tracer with the given config.
pub(super) fn prestate_tracer(config: PreStateConfig) -> GethDebugTracingOptions {
    GethDebugTracingOptions::default()
        .with_tracer(GethDebugTracerType::BuiltInTracer(GethDebugBuiltInTracerType::PreStateTracer))
        .with_config(GethDebugTracerConfig::from(config))
}

/// Returns the gas reported by a successful default (struct logger) trace.
pub(super) fn default_trace_gas(trace: &TraceResult<GethTrace, String>) -> u64 {
    match trace {
        TraceResult::Success { result: GethTrace::Default(frame), .. } => frame.gas,
        other => panic!("unexpected trace shape: {other:?}"),
    }
}

/// Asserts that the sender's balance changes and that every balance a parity state diff ends on is
/// the committed one.
fn assert_state_diff_settled(
    state_diff: Option<&StateDiff>,
    sender: Address,
    committed_balance: &impl Fn(Address) -> eyre::Result<U256>,
) -> eyre::Result<()> {
    let state_diff = state_diff.expect("state diff requested");
    assert!(
        matches!(state_diff[&sender].balance, Delta::Changed(_)),
        "sender balance must change: {:?}",
        state_diff[&sender],
    );
    for (address, account) in state_diff.iter() {
        let to = match &account.balance {
            Delta::Unchanged => continue,
            Delta::Added(to) => to,
            Delta::Changed(changed) => &changed.to,
            Delta::Removed(_) => panic!("{address} must not be removed"),
        };
        assert_eq!(*to, committed_balance(*address)?, "state diff balance of {address}");
    }
    Ok(())
}

pub(super) async fn launch_sdm_node(
    chain_spec: &Arc<OpChainSpec>,
) -> eyre::Result<reth_e2e_test_utils::NodeHelperType<OpNode>> {
    launch_sdm_node_with_attributes(chain_spec, lagoon::payload_attributes).await
}

pub(super) async fn launch_sdm_node_with_attributes(
    chain_spec: &Arc<OpChainSpec>,
    attributes: impl Fn(u64) -> reth_optimism_payload_builder::OpPayloadAttrs + Send + Sync + 'static,
) -> eyre::Result<reth_e2e_test_utils::NodeHelperType<OpNode>> {
    let op_node = OpNode::new(RollupArgs {
        operator_sdm_opt_in: true,
        testing_sdm_fixed_policy: Some(None),
        ..Default::default()
    });
    let NodeHandle { node, node_exit_future: _ } = NodeBuilder::new(rpc_node_config(chain_spec))
        .testing_node(Runtime::test())
        .node(op_node)
        .launch()
        .await?;
    NodeTestContext::new(node, attributes).await
}
