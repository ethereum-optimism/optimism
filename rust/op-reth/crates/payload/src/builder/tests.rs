use super::{OpPayloadBuilder, build_post_exec_recovered_tx, try_include_post_exec_tx};
use crate::config::{OpBuilderConfig, OpDAConfig, OpGasLimitConfig};
use alloy_consensus::{Typed2718, transaction::Recovered};
use alloy_evm::RecoveredTx;
use alloy_primitives::Address;
use op_alloy_consensus::SDMGasEntry;
use reth_evm::execute::BlockExecutionError;
use reth_optimism_primitives::OpTransactionSigned;
use reth_payload_builder_primitives::PayloadBuilderError;
use std::cell::Cell;

fn entries(specs: &[(u64, u64)]) -> Vec<SDMGasEntry> {
    specs.iter().map(|&(index, gas_refund)| SDMGasEntry { index, gas_refund }).collect()
}

fn unwrap_post_exec(tx: Recovered<OpTransactionSigned>) -> (u8, u64, Vec<SDMGasEntry>) {
    let ty = tx.tx().ty();
    let OpTransactionSigned::PostExec(signed) = tx.into_inner() else {
        panic!("expected post-exec transaction");
    };
    (ty, signed.inner().payload.block_number, signed.inner().payload.gas_refund_entries.clone())
}

// Ensures the payload builder keeps SDM disabled by default and preserves the explicit
// integration-test override when swapping in a transaction source.
#[test]
fn payload_builder_preserves_sdm_config() {
    let default = OpBuilderConfig::new(OpDAConfig::default(), OpGasLimitConfig::default());
    assert!(!default.sdm_enabled);

    let builder = OpPayloadBuilder::<(), (), (), (), ()>::with_builder_config(
        (),
        (),
        (),
        OpBuilderConfig::new_with_sdm(OpDAConfig::default(), OpGasLimitConfig::default(), true),
    )
    .with_transactions(42u64);
    assert!(builder.config.sdm_enabled);
    assert_eq!(builder.best_transactions, 42);
}

#[test]
fn build_post_exec_recovered_tx_wraps_entries_in_post_exec_tx() {
    let block_number = 42;
    let payload_entries = entries(&[(3, 17), (5, 23)]);
    let recovered =
        build_post_exec_recovered_tx::<OpTransactionSigned>(block_number, payload_entries.clone());

    assert_eq!(recovered.signer(), Address::ZERO);
    let (ty, decoded_block, decoded_entries) = unwrap_post_exec(recovered);
    assert_eq!(ty, op_alloy_consensus::POST_EXEC_TX_TYPE_ID);
    assert_eq!(decoded_block, block_number);
    assert_eq!(decoded_entries, payload_entries);
}

#[test]
fn try_include_post_exec_tx_skips_when_no_entries() {
    let called = Cell::new(false);
    let result = try_include_post_exec_tx::<OpTransactionSigned, _>(1, Vec::new(), |_tx| {
        called.set(true);
        Ok::<_, BlockExecutionError>(0)
    });
    assert!(matches!(result, Ok(false)));
    assert!(!called.get(), "execute must not run when there are no entries");
}

#[test]
fn try_include_post_exec_tx_executes_post_exec_tx_on_happy_path() {
    let block_number = 99;
    let payload_entries = entries(&[(0, 7)]);
    let captured: Cell<Option<(u8, u64, Vec<SDMGasEntry>)>> = Cell::new(None);

    let result = try_include_post_exec_tx::<OpTransactionSigned, _>(
        block_number,
        payload_entries.clone(),
        |tx| {
            captured.set(Some(unwrap_post_exec(tx)));
            Ok::<_, BlockExecutionError>(21_000)
        },
    );

    assert!(matches!(result, Ok(true)));
    let (ty, decoded_block, decoded_entries) = captured.take().expect("execute closure ran");
    assert_eq!(ty, op_alloy_consensus::POST_EXEC_TX_TYPE_ID);
    assert_eq!(decoded_block, block_number);
    assert_eq!(decoded_entries, payload_entries);
}

/// Consensus-critical: if the post-exec tx fails to execute, the payload build MUST abort.
/// Returning `Ok(_)` (e.g. an empty block, or silently dropping the tx) would diverge the
/// producer from any honest verifier, because the verifier observes refunds from the normal
/// txs and expects a matching post-exec tx.
#[test]
fn try_include_post_exec_tx_aborts_when_execution_fails() {
    let called = Cell::new(false);
    let result = try_include_post_exec_tx::<OpTransactionSigned, _>(1, entries(&[(0, 7)]), |_tx| {
        called.set(true);
        Err::<u64, _>(BlockExecutionError::msg("forced post-exec tx failure"))
    });

    assert!(called.get(), "execute must be invoked so its error can propagate");
    match result {
        Err(PayloadBuilderError::EvmExecutionError(err)) => {
            assert!(err.to_string().contains("forced post-exec tx failure"));
        }
        other => panic!("expected EvmExecutionError, got {other:?}"),
    }
}
