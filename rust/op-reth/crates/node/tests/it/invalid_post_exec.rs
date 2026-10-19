//! Malformed post-exec (0x7d) blocks must be INVALID, not fatal, on both engine paths ([SDM-M1]).
//!
//! Cover both a 0x7d before Lagoon and a wrong block-number anchor with Lagoon active.
//! The shared [`setup`] stays Ecotone-only; [`setup_lagoon`] supplies the later fork attributes.
//!
//! [SDM-M1]: https://github.com/ethereum-optimism/protocol-team/issues/371

use alloy_consensus::{Sealable, proofs::calculate_transaction_root};
use alloy_genesis::Genesis;
use alloy_primitives::B64;
use alloy_rpc_types_engine::{PayloadStatus, PayloadStatusEnum, PayloadValidationError};
use op_alloy_consensus::{
    PostExecPayloadValidationError, SDMGasEntry, build_post_exec_tx, encode_jovian_extra_data,
};
use op_alloy_rpc_types_engine::{OpExecutionData, OpExecutionPayload};
use reth_chainspec::BaseFeeParams;
use reth_e2e_test_utils::{NodeHelperType, setup_engine_with_connection, wallet::Wallet};
use reth_optimism_chainspec::OpChainSpecBuilder;
use reth_optimism_node::{
    OpBuiltPayload, OpNode,
    payload::OpExecData,
    utils::{advance_chain, optimism_payload_attributes, setup},
};
use reth_optimism_primitives::OpTransactionSigned;
use std::sync::Arc;
use tokio::sync::Mutex;

/// Creates Lagoon-active nodes without changing the shared Ecotone fixture.
async fn setup_lagoon(num_nodes: usize) -> eyre::Result<(Vec<NodeHelperType<OpNode>>, Wallet)> {
    let mut genesis: Genesis = serde_json::from_str(include_str!("../assets/genesis.json"))?;
    // Lagoon is active at genesis, so the first child's base fee uses Jovian extra data.
    genesis.extra_data = encode_jovian_extra_data(B64::ZERO, BaseFeeParams::optimism(), 0)?;
    genesis.base_fee_per_gas = Some(1_000_000_000);
    let chain_spec = Arc::new(
        OpChainSpecBuilder::optimism_sepolia().genesis(genesis).lagoon_activated().build(),
    );
    setup_engine_with_connection(
        num_nodes,
        chain_spec,
        false,
        Default::default(),
        |timestamp| {
            let mut attributes = optimism_payload_attributes(timestamp);
            attributes.0.eip_1559_params = Some(B64::ZERO);
            attributes.0.min_base_fee = Some(0);
            attributes
        },
        // The buffered-child test must learn the parent only through its explicit newPayload.
        false,
    )
    .await
}

/// Adds a decodable `PostExec` with a wrong anchor, so rejection reaches executor validation
/// rather than the engine's earlier payload-conversion checks.
fn with_mismatched_post_exec_block_number(payload: &OpBuiltPayload) -> (OpExecData, String) {
    let mut block = payload.block().clone_block();
    let block_number = block.header.number;
    let payload_block_number = block_number + 1;
    assert!(!block.body.transactions.is_empty(), "fixture includes a regular transaction");
    let post_exec =
        build_post_exec_tx(payload_block_number, vec![SDMGasEntry { index: 0, gas_refund: 1 }])
            .seal_slow();
    block.body.transactions.push(OpTransactionSigned::PostExec(post_exec));
    block.header.transactions_root = calculate_transaction_root(&block.body.transactions);
    let (payload, sidecar) = OpExecutionPayload::from_block_slow(&block);
    let reason =
        PostExecPayloadValidationError::BlockNumberMismatch { payload_block_number, block_number }
            .to_string();
    (OpExecutionData::new(payload, sidecar).into(), reason)
}

/// Appends a 0x7d to `payload`'s block and reseals it. Returns it with the parser's rejection.
fn with_post_exec_tx(payload: &OpBuiltPayload) -> (OpExecData, String) {
    let tx_index = payload.block().body().transactions.len() as u64;
    let reason = PostExecPayloadValidationError::UnexpectedPostExecTx { tx_index }.to_string();
    // No entries, so the entry-count preflight passes and the fork gate rejects it.
    (with_post_exec_entries(payload, Vec::new()), reason)
}

/// Appends a `PostExec` transaction with the supplied refund entries and reseals the block.
fn with_post_exec_entries(payload: &OpBuiltPayload, entries: Vec<SDMGasEntry>) -> OpExecData {
    let mut block = payload.block().clone_block();
    let post_exec = build_post_exec_tx(block.header.number, entries).seal_slow();
    block.body.transactions.push(OpTransactionSigned::PostExec(post_exec));
    block.header.transactions_root = calculate_transaction_root(&block.body.transactions);
    let (payload, sidecar) = OpExecutionPayload::from_block_slow(&block);
    OpExecutionData::new(payload, sidecar).into()
}

fn assert_invalid(status: &PayloadStatus, expected_reason: &str) {
    match &status.status {
        PayloadStatusEnum::Invalid { validation_error } => assert!(
            validation_error.contains(expected_reason),
            "expected INVALID because {expected_reason:?}, got {validation_error:?}",
        ),
        other => panic!("expected INVALID because {expected_reason:?}, got {other:?}"),
    }
}

#[tokio::test]
async fn test_new_payload_with_malformed_post_exec_is_invalid() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let (mut nodes, wallet) = setup(1).await?;
    let mut node = nodes.pop().unwrap();
    let parent = advance_chain(1, &mut node, Arc::new(Mutex::new(wallet))).await?.remove(0);
    let (bad_child, reason) = with_post_exec_tx(&node.new_payload().await?);
    let engine = &node.inner.add_ons_handle.beacon_engine_handle;

    let status = engine.new_payload(bad_child.clone()).await.expect("not an internal error");
    assert_invalid(&status, &reason);
    assert_eq!(status.latest_valid_hash, Some(parent.block().hash()));

    // Cached as an invalid header.
    let status = engine.new_payload(bad_child).await.expect("not an internal error");
    assert_invalid(&status, &PayloadValidationError::LinksToRejectedPayload.to_string());

    Ok(())
}

#[tokio::test]
async fn test_lagoon_new_payload_with_wrong_post_exec_block_number_is_invalid() -> eyre::Result<()>
{
    reth_tracing::init_test_tracing();

    let (mut nodes, wallet) = setup_lagoon(1).await?;
    let mut node = nodes.pop().unwrap();
    let payloads = advance_chain(2, &mut node, Arc::new(Mutex::new(wallet))).await?;
    let parent = &payloads[0];
    let (bad_child, reason) = with_mismatched_post_exec_block_number(&payloads[1]);
    let engine = &node.inner.add_ons_handle.beacon_engine_handle;

    let status = engine.new_payload(bad_child.clone()).await.expect("not an internal error");
    assert_invalid(&status, &reason);
    assert_eq!(status.latest_valid_hash, Some(parent.block().hash()));

    let status = engine.new_payload(bad_child).await.expect("not an internal error");
    assert_invalid(&status, &PayloadValidationError::LinksToRejectedPayload.to_string());
    assert_eq!(status.latest_valid_hash, Some(parent.block().hash()));

    Ok(())
}

#[tokio::test]
async fn test_lagoon_buffered_child_with_wrong_post_exec_block_number_is_invalid()
-> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let (mut nodes, wallet) = setup_lagoon(2).await?;
    let verifier = nodes.pop().unwrap();
    let mut producer = nodes.pop().unwrap();
    let payloads = advance_chain(2, &mut producer, Arc::new(Mutex::new(wallet))).await?;
    let parent = &payloads[0];
    let (bad_child, _) = with_mismatched_post_exec_block_number(&payloads[1]);
    let engine = &verifier.inner.add_ons_handle.beacon_engine_handle;

    let status = engine.new_payload(bad_child.clone()).await.expect("not an internal error");
    assert_eq!(status.status, PayloadStatusEnum::Syncing);

    let status =
        engine.new_payload(parent.clone().into()).await.expect("engine survives the child");
    assert_eq!(status.status, PayloadStatusEnum::Valid);

    let status = engine.new_payload(bad_child).await.expect("not an internal error");
    assert_invalid(&status, &PayloadValidationError::LinksToRejectedPayload.to_string());
    assert_eq!(status.latest_valid_hash, Some(parent.block().hash()));

    Ok(())
}

#[tokio::test]
async fn test_new_payload_with_oversized_post_exec_entries_is_invalid() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let (mut nodes, wallet) = setup(1).await?;
    let mut node = nodes.pop().unwrap();
    let parent = advance_chain(1, &mut node, Arc::new(Mutex::new(wallet))).await?.remove(0);
    let child = node.new_payload().await?;
    let entry_count = child.block().body().transactions.len() + 1;
    let bad_child =
        with_post_exec_entries(&child, vec![SDMGasEntry { index: 0, gas_refund: 1 }; entry_count]);
    let engine = &node.inner.add_ons_handle.beacon_engine_handle;

    let reason = PostExecPayloadValidationError::TooManyGasRefundEntries {
        entry_count,
        preceding_transaction_count: entry_count - 1,
    }
    .to_string();
    let status = engine.new_payload(bad_child.clone()).await.expect("not an internal error");
    assert_invalid(&status, &reason);
    assert_eq!(status.latest_valid_hash, Some(parent.block().hash()));

    let status = engine.new_payload(bad_child).await.expect("not an internal error");
    assert_invalid(&status, &PayloadValidationError::LinksToRejectedPayload.to_string());

    Ok(())
}

#[tokio::test]
async fn test_buffered_malformed_post_exec_child_is_invalid() -> eyre::Result<()> {
    reth_tracing::init_test_tracing();

    let (mut nodes, wallet) = setup(2).await?;
    let verifier = nodes.pop().unwrap();
    let mut producer = nodes.pop().unwrap();
    let parent = advance_chain(1, &mut producer, Arc::new(Mutex::new(wallet))).await?.remove(0);
    let (bad_child, _) = with_post_exec_tx(&producer.new_payload().await?);
    let engine = &verifier.inner.add_ons_handle.beacon_engine_handle;

    // Parent unknown, so the child is buffered.
    let status = engine.new_payload(bad_child.clone()).await.expect("not an internal error");
    assert_eq!(status.status, PayloadStatusEnum::Syncing);

    // The parent connects the child on the block path; the engine must survive it.
    let status =
        engine.new_payload(parent.clone().into()).await.expect("engine survives the child");
    assert_eq!(status.status, PayloadStatusEnum::Valid);

    // Rejected while connecting, so already cached as invalid.
    let status = engine.new_payload(bad_child).await.expect("not an internal error");
    assert_invalid(&status, &PayloadValidationError::LinksToRejectedPayload.to_string());
    assert_eq!(status.latest_valid_hash, Some(parent.block().hash()));

    Ok(())
}
