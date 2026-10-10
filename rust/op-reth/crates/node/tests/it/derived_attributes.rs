//! SDM-H1: derived (`noTxPool`) attributes with invalid `PostExec` data must be rejected at
//! `engine_forkchoiceUpdated`, so the rollup node can recover with a deposits-only block.

use alloy_consensus::{Sealable, SignableTransaction, TxEip1559};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{Address, B64, B256, Bytes, Signature, TxKind};
use alloy_rpc_types_engine::{ForkchoiceState, ForkchoiceUpdated};
use jsonrpsee::{
    core::{ClientError, client::ClientT},
    rpc_params,
};
use op_alloy_consensus::{OpTxEnvelope, SDMGasEntry, TxDeposit, build_post_exec_tx};
use op_alloy_rpc_types_engine::OpExecutionPayloadEnvelopeV4;
use reth_chainspec::EthChainSpec;
use reth_node_builder::{Node, NodeBuilder, NodeConfig};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::{OpNode, utils::optimism_payload_attributes};
use reth_optimism_payload_builder::OpPayloadAttrs;
use reth_tasks::Runtime;
use std::sync::Arc;

/// `INVALID_PAYLOAD_ATTRIBUTES` JSON-RPC error code.
const INVALID_PAYLOAD_ATTRIBUTES: i32 = -38003;

async fn fcu(
    client: &impl ClientT,
    state: ForkchoiceState,
    attrs: &OpPayloadAttrs,
) -> Result<ForkchoiceUpdated, ClientError> {
    client.request("engine_forkchoiceUpdatedV3", rpc_params![state, attrs.0.clone()]).await
}

#[track_caller]
fn assert_invalid_attributes(result: Result<ForkchoiceUpdated, ClientError>, case: &str) {
    match result {
        Err(ClientError::Call(err)) => {
            assert_eq!(err.code(), INVALID_PAYLOAD_ATTRIBUTES, "{case}: unexpected error {err:?}")
        }
        other => panic!("{case}: expected INVALID_PAYLOAD_ATTRIBUTES, got {other:?}"),
    }
}

// SDM-H1: reject malformed PostExec attributes at FCU, not later at getPayload.
#[tokio::test(flavor = "multi_thread")]
async fn derived_attributes_with_invalid_post_exec_are_rejected_at_fcu() -> eyre::Result<()> {
    let chain_spec = Arc::new(OpChainSpecBuilder::optimism_sepolia().lagoon_activated().build());
    let chain_id = chain_spec.chain_id();
    let genesis = chain_spec.genesis_header().clone();
    let mut config = NodeConfig::test().map_chain(chain_spec.clone()).with_unused_ports();
    config.network.discovery.discv5_port = Some(0);
    config.network.discovery.discv5_port_ipv6 = Some(0);

    let op_node = OpNode::default();
    let node = NodeBuilder::new(config)
        .testing_node(Runtime::test())
        .with_types::<OpNode>()
        .with_components(op_node.components())
        .with_add_ons(op_node.add_ons())
        .launch()
        .await?
        .node;
    let client = node.auth_server_handle().http_client();
    let head = ForkchoiceState::same_hash(chain_spec.genesis_hash());

    // Fund the regular tx at nonce 1; it uses 21_000 gas.
    let regular = TxEip1559 {
        chain_id,
        nonce: 1,
        gas_limit: 21_000,
        max_fee_per_gas: 1_000_000_000_000,
        to: TxKind::Call(Address::with_last_byte(1)),
        ..Default::default()
    }
    .into_signed(Signature::test_signature());
    let deposit = TxDeposit {
        source_hash: B256::with_last_byte(1),
        from: regular.recover_signer()?,
        to: TxKind::Call(Address::with_last_byte(1)),
        mint: 1_000_000_000_000_000_000,
        gas_limit: 21_000,
        ..Default::default()
    };
    let deposit: Bytes = OpTxEnvelope::Deposit(deposit.seal_slow()).encoded_2718().into();
    let regular: Bytes = OpTxEnvelope::Eip1559(regular).encoded_2718().into();
    let post_exec = |block_number, entries: Vec<SDMGasEntry>| -> Bytes {
        OpTxEnvelope::PostExec(build_post_exec_tx(block_number, entries).seal_slow())
            .encoded_2718()
            .into()
    };
    let refund = |index, gas_refund| SDMGasEntry { index, gas_refund };
    let derived = |transactions: Vec<Bytes>| {
        let mut attrs = optimism_payload_attributes(genesis.timestamp + 2);
        attrs.0.no_tx_pool = Some(true);
        attrs.0.eip_1559_params = Some(B64::ZERO);
        attrs.0.min_base_fee = Some(0);
        attrs.0.transactions = Some(transactions);
        attrs
    };

    let cases = [
        (
            "E1: two 0x7D",
            vec![
                deposit.clone(),
                regular.clone(),
                post_exec(1, vec![refund(1, 1)]),
                post_exec(1, vec![refund(1, 1)]),
            ],
        ),
        (
            "E3: 0x7D anchored to the next block",
            vec![deposit.clone(), regular.clone(), post_exec(2, vec![refund(1, 1)])],
        ),
        ("E4: empty refund entries", vec![deposit.clone(), regular.clone(), post_exec(1, vec![])]),
        (
            "E4: refund targets a deposit",
            vec![deposit.clone(), regular.clone(), post_exec(1, vec![refund(0, 1)])],
        ),
        (
            "E5: refund above evmGasUsed",
            vec![deposit.clone(), regular.clone(), post_exec(1, vec![refund(1, 21_001)])],
        ),
        (
            "E6: refund entry past the last tx",
            vec![deposit.clone(), regular.clone(), post_exec(1, vec![refund(1, 1), refund(5, 1)])],
        ),
        (
            "0x7D is not the last tx",
            vec![deposit.clone(), post_exec(1, vec![refund(1, 1)]), regular.clone()],
        ),
    ];
    for (case, transactions) in cases {
        assert_invalid_attributes(fcu(&client, head, &derived(transactions)).await, case);
    }

    // Control: the same fixture with a well-formed 0x7D builds a 3-tx block.
    let valid = derived(vec![deposit, regular, post_exec(1, vec![refund(1, 1)])]);
    let updated = fcu(&client, head, &valid).await?;
    assert!(updated.payload_status.is_valid(), "valid derived attributes: {updated:?}");
    let id = updated.payload_id.expect("valid derived attributes start a payload job");
    let payload: OpExecutionPayloadEnvelopeV4 =
        client.request("engine_getPayloadV5", rpc_params![id]).await?;
    assert_eq!(
        payload.execution_payload.payload_inner.payload_inner.payload_inner.transactions.len(),
        3
    );

    Ok(())
}
