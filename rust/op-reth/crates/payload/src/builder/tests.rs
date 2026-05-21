use super::{
    ExecutionInfo, OpPayloadBuilder, OpPayloadBuilderCtx, build_post_exec_recovered_tx,
    try_include_post_exec_tx,
};
use crate::{
    OpPayloadBuilderAttributes,
    config::{OpBuilderConfig, OpDAConfig, OpGasLimitConfig},
};
use alloy_consensus::{
    Header, SignableTransaction, TxEip1559, Typed2718,
    transaction::{Recovered, TxHashRef},
};
use alloy_eips::eip2718::Encodable2718;
use alloy_evm::RecoveredTx;
use alloy_primitives::{Address, B256, Signature, TxHash, TxKind, U256};
use op_alloy_consensus::SDMGasEntry;
use reth_basic_payload_builder::PayloadConfig;
use reth_chainspec::MIN_TRANSACTION_GAS;
use reth_evm::execute::{BlockBuilder, BlockExecutionError};
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
use reth_optimism_evm::OpEvmConfig;
use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
use reth_optimism_txpool::OpPooledTransaction;
use reth_payload_builder_primitives::PayloadBuilderError;
use reth_payload_util::PayloadTransactionsFixed;
use reth_primitives_traits::{Account, SealedHeader};
use reth_revm::{database::StateProviderDatabase, db::State, test_utils::StateProviderTest};
use reth_transaction_pool::PoolTransaction;
use std::{cell::Cell, sync::Arc};

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

fn payload_builder_ctx(
    chain_spec: Arc<OpChainSpec>,
    gas_limit: u64,
) -> OpPayloadBuilderCtx<
    OpEvmConfig<OpChainSpec, OpPrimitives>,
    OpChainSpec,
    OpPayloadBuilderAttributes<OpTransactionSigned>,
> {
    let parent = SealedHeader::seal_slow(Header {
        gas_limit,
        number: 0,
        timestamp: 0,
        ..Default::default()
    });
    let attributes = OpPayloadBuilderAttributes {
        timestamp: 1,
        gas_limit: Some(gas_limit),
        ..Default::default()
    };

    OpPayloadBuilderCtx {
        evm_config: OpEvmConfig::optimism(chain_spec.clone()),
        builder_config: OpBuilderConfig::default(),
        chain_spec,
        config: PayloadConfig {
            parent_header: Arc::new(parent),
            payload_id: attributes.id,
            attributes,
        },
        cancel: Default::default(),
        best_payload: None,
    }
}

fn op_pooled_tx(nonce: u64, signer: Address, recipient: Address) -> OpPooledTransaction {
    let tx: OpTransactionSigned = TxEip1559 {
        chain_id: 8453,
        nonce,
        gas_limit: MIN_TRANSACTION_GAS,
        max_fee_per_gas: 1,
        max_priority_fee_per_gas: 1,
        to: TxKind::Call(recipient),
        value: U256::ZERO,
        ..Default::default()
    }
    .into_signed(Signature::test_signature())
    .into();
    let encoded_len = tx.encode_2718_len();

    OpPooledTransaction::new(Recovered::new_unchecked(tx, signer), encoded_len)
}

fn tx_hashes<'a>(txs: impl IntoIterator<Item = &'a Recovered<OpTransactionSigned>>) -> Vec<TxHash> {
    txs.into_iter().map(|tx| *TxHashRef::tx_hash(tx)).collect()
}

fn run_execute_best_transactions(
    signer: Address,
    txs: Vec<OpPooledTransaction>,
    gas_limit_cap: Option<u64>,
    committed_txs: Option<&mut Vec<Recovered<OpTransactionSigned>>>,
) -> (ExecutionInfo, Vec<TxHash>) {
    let gas_limit = 1_000_000;
    let chain_spec = Arc::new(OpChainSpecBuilder::base_mainnet().regolith_activated().build());
    let ctx = payload_builder_ctx(chain_spec, gas_limit);

    let mut state_provider = StateProviderTest::default();
    state_provider.insert_account(
        signer,
        Account { balance: U256::MAX, ..Default::default() },
        None,
        Default::default(),
    );

    let best_txs = PayloadTransactionsFixed::new(txs);

    let mut db = State::builder()
        .with_database(StateProviderDatabase::new(&state_provider))
        .with_bundle_update()
        .build();
    let mut builder = ctx.block_builder(&mut db).expect("block builder can be created");
    let mut info = ExecutionInfo::new();

    assert!(
        ctx.execute_best_transactions(
            &mut info,
            &mut builder,
            best_txs,
            gas_limit_cap,
            committed_txs
        )
        .expect("best transactions execute")
        .is_none()
    );

    let outcome = builder
        .finish(state_provider.clone(), Some((B256::ZERO, Default::default())))
        .expect("builder finishes");
    let included_tx_hashes = outcome
        .block
        .into_block()
        .body
        .transactions
        .iter()
        .map(|tx| *TxHashRef::tx_hash(tx))
        .collect();

    (info, included_tx_hashes)
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
fn execute_best_transactions_committed_txs_preserves_execution() {
    let mut committed_txs = Vec::new();
    let signer = Address::repeat_byte(0x11);
    let tx0 = op_pooled_tx(0, signer, Address::repeat_byte(0x22));
    let nonce_too_low = tx0.clone();
    let tx1 = op_pooled_tx(1, signer, Address::repeat_byte(0x33));
    let expected_committed = vec![*tx0.hash(), *tx1.hash()];
    let txs = vec![tx0, nonce_too_low, tx1];

    let (committed_info, committed_block_tx_hashes) =
        run_execute_best_transactions(signer, txs.clone(), None, Some(&mut committed_txs));
    let (none_info, none_block_tx_hashes) = run_execute_best_transactions(signer, txs, None, None);

    assert_eq!(tx_hashes(&committed_txs), expected_committed);
    assert_eq!(committed_block_tx_hashes, expected_committed);
    assert_eq!(none_block_tx_hashes, expected_committed);
    assert_eq!(committed_info.cumulative_gas_used, none_info.cumulative_gas_used);
    assert_eq!(committed_info.cumulative_da_bytes_used, none_info.cumulative_da_bytes_used);
    assert_eq!(committed_info.total_fees, none_info.total_fees);
}

#[test]
fn execute_best_transactions_respects_gas_limit_cap() {
    let mut committed_txs = Vec::new();
    let signer = Address::repeat_byte(0x11);
    let tx0 = op_pooled_tx(0, signer, Address::repeat_byte(0x22));
    let tx1 = op_pooled_tx(1, signer, Address::repeat_byte(0x33));
    let expected_committed = vec![*tx0.hash()];
    let txs = vec![tx0, tx1];

    let (_info, block_tx_hashes) = run_execute_best_transactions(
        signer,
        txs,
        Some(MIN_TRANSACTION_GAS),
        Some(&mut committed_txs),
    );

    assert_eq!(tx_hashes(&committed_txs), expected_committed);
    assert_eq!(block_tx_hashes, expected_committed);
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
