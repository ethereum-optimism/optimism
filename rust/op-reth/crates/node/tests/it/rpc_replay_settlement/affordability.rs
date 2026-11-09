//! A later transaction that is only affordable because of an earlier refund.

use crate::lagoon::{
    self, DEPLOY_GAS, default_trace_gas, http_client, launch_sdm_node_with_attributes,
    signed_transaction,
};
use alloy_consensus::BlockHeader;
use alloy_primitives::{Bytes, TxKind, U256};
use alloy_rpc_types_eth::{BlockNumberOrTag, TransactionRequest};
use alloy_rpc_types_trace::geth::GethTrace;
use reth_chainspec::EthChainSpec;
use reth_e2e_test_utils::wallet::Wallet;
use reth_rpc_api::clients::DebugApiClient;

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
        let stop = Bytes::from_static(&[0x00]);
        let signed = signed_transaction(
            &wallet,
            nonce,
            transaction_gas_limit,
            gas_price,
            TxKind::Create,
            stop,
        )
        .await;
        forced_transactions.push(signed);
    }
    let mut node = launch_sdm_node_with_attributes(&chain_spec, move |timestamp| {
        let mut attributes = lagoon::payload_attributes(timestamp);
        // The transactions are forced through attributes because the pool's balance check does
        // not know about refunds and would reject the second one. Local sequencing (the default
        // mode) produces the refunds; derived mode would only verify embedded ones.
        attributes.0.transactions = Some(forced_transactions.clone());
        attributes
    })
    .await?;
    let payload = node.advance_block().await?;
    let block = payload.block();
    assert_eq!(block.body().transactions.len(), 3, "two deployments and the post-exec tx");
    let client = http_client(&node);

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
