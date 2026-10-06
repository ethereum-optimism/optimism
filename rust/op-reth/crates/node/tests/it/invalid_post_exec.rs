//! Malformed post-exec (0x7d) blocks must be INVALID, not fatal, on both engine paths ([SDM-M1]).
//!
//! The trigger is a 0x7d before Lagoon; [`setup`] activates forks only up to Ecotone.
//!
//! [SDM-M1]: https://github.com/ethereum-optimism/protocol-team/issues/371

use alloy_consensus::{Sealable, proofs::calculate_transaction_root};
use alloy_rpc_types_engine::{PayloadStatus, PayloadStatusEnum, PayloadValidationError};
use op_alloy_consensus::{PostExecPayloadValidationError, SDMGasEntry, build_post_exec_tx};
use op_alloy_rpc_types_engine::{OpExecutionData, OpExecutionPayload};
use reth_optimism_node::{
    OpBuiltPayload,
    payload::OpExecData,
    utils::{advance_chain, setup},
};
use reth_optimism_primitives::OpTransactionSigned;
use std::sync::Arc;
use tokio::sync::Mutex;

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
