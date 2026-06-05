use super::{
    ExecutionInfo, OpPayloadBuilderCtx, build_post_exec_recovered_tx, try_include_post_exec_tx,
};
use crate::{OpPayloadBuilderAttributes, config::OpBuilderConfig};
use alloy_consensus::{
    Header, SignableTransaction, Transaction, TxEip1559, Typed2718,
    transaction::{Recovered, TxHashRef},
};
use alloy_eips::{
    eip2718::{Encodable2718, WithEncoded},
    eip2930::AccessList,
    eip7702::SignedAuthorization,
};
use alloy_evm::RecoveredTx;
use alloy_primitives::{Address, B256, Bytes, Signature, TxHash, TxKind, U256};
use alloy_rpc_types_eth::erc4337::TransactionConditional;
use op_alloy_consensus::SDMGasEntry;
use reth_basic_payload_builder::PayloadConfig;
use reth_chainspec::MIN_TRANSACTION_GAS;
use reth_evm::execute::{BlockBuilder, BlockExecutionError};
use reth_optimism_chainspec::{OpChainSpec, OpChainSpecBuilder};
use reth_optimism_evm::OpEvmConfig;
use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
use reth_optimism_txpool::{
    OpPooledTransaction, OpPooledTx, conditional::MaybeConditionalTransaction,
    estimated_da_size::DataAvailabilitySized, interop::MaybeInteropTransaction,
};
use reth_payload_builder_primitives::PayloadBuilderError;
use reth_payload_util::PayloadTransactionsFixed;
use reth_primitives_traits::{Account, InMemorySize, SealedHeader};
use reth_revm::{database::StateProviderDatabase, db::State, test_utils::StateProviderTest};
use reth_transaction_pool::PoolTransaction;
use std::{borrow::Cow, cell::Cell, sync::Arc};

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

fn run_execute_best_transactions<T>(
    signer: Address,
    txs: Vec<T>,
    gas_limit_cap: Option<u64>,
    committed_txs: Option<&mut Vec<Recovered<OpTransactionSigned>>>,
) -> (ExecutionInfo, Vec<TxHash>)
where
    T: PoolTransaction<Consensus = OpTransactionSigned> + OpPooledTx,
{
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

/// Regression test for the ordering of `effective_tip_per_gas()` and `into_consensus()` in the
/// payload builder loop: the miner fee must be read from the pool wrapper before the consensus tx
/// is exposed, so that callers can override the tip (e.g. to convert non-native fee denominations
/// into a native-wei tip). If a future change moves the read after `into_consensus()`, the tip
/// would come from the unwrapped consensus tx and this test would fail.
#[test]
fn miner_fee_uses_pool_wrapper_tip() {
    /// Pool-tx wrapper whose `max_priority_fee_per_gas` (and therefore the default
    /// `effective_tip_per_gas`) returns a forced value, while `into_consensus()` still exposes
    /// the unmodified inner tx. Production callers do this to convert non-native fee
    /// denominations into a native-wei tip.
    #[derive(Debug, Clone)]
    struct ForcedTipPooledTx {
        inner: OpPooledTransaction,
        forced_priority_fee: u128,
    }

    impl Typed2718 for ForcedTipPooledTx {
        fn ty(&self) -> u8 {
            self.inner.ty()
        }
    }

    impl InMemorySize for ForcedTipPooledTx {
        fn size(&self) -> usize {
            self.inner.size()
        }
    }

    impl Transaction for ForcedTipPooledTx {
        fn chain_id(&self) -> Option<u64> {
            self.inner.chain_id()
        }
        fn nonce(&self) -> u64 {
            self.inner.nonce()
        }
        fn gas_limit(&self) -> u64 {
            self.inner.gas_limit()
        }
        fn gas_price(&self) -> Option<u128> {
            self.inner.gas_price()
        }
        fn max_fee_per_gas(&self) -> u128 {
            // High enough that the default `effective_tip_per_gas` impl won't cap
            // `forced_priority_fee` via `min(max_fee_per_gas - base_fee, priority_fee)`.
            u128::MAX
        }
        fn max_priority_fee_per_gas(&self) -> Option<u128> {
            Some(self.forced_priority_fee)
        }
        fn max_fee_per_blob_gas(&self) -> Option<u128> {
            self.inner.max_fee_per_blob_gas()
        }
        fn priority_fee_or_price(&self) -> u128 {
            self.inner.priority_fee_or_price()
        }
        fn effective_gas_price(&self, base_fee: Option<u64>) -> u128 {
            self.inner.effective_gas_price(base_fee)
        }
        fn is_dynamic_fee(&self) -> bool {
            self.inner.is_dynamic_fee()
        }
        fn kind(&self) -> TxKind {
            self.inner.kind()
        }
        fn is_create(&self) -> bool {
            self.inner.is_create()
        }
        fn value(&self) -> U256 {
            self.inner.value()
        }
        fn input(&self) -> &Bytes {
            self.inner.input()
        }
        fn access_list(&self) -> Option<&AccessList> {
            self.inner.access_list()
        }
        fn blob_versioned_hashes(&self) -> Option<&[B256]> {
            self.inner.blob_versioned_hashes()
        }
        fn authorization_list(&self) -> Option<&[SignedAuthorization]> {
            self.inner.authorization_list()
        }
    }

    impl MaybeConditionalTransaction for ForcedTipPooledTx {
        fn set_conditional(&mut self, conditional: TransactionConditional) {
            self.inner.set_conditional(conditional);
        }
        fn conditional(&self) -> Option<&TransactionConditional> {
            self.inner.conditional()
        }
    }

    impl MaybeInteropTransaction for ForcedTipPooledTx {
        fn set_interop_deadline(&self, deadline: u64) {
            self.inner.set_interop_deadline(deadline);
        }
        fn interop_deadline(&self) -> Option<u64> {
            self.inner.interop_deadline()
        }
    }

    impl DataAvailabilitySized for ForcedTipPooledTx {
        fn estimated_da_size(&self) -> u64 {
            self.inner.estimated_da_size()
        }
    }

    impl PoolTransaction for ForcedTipPooledTx {
        type TryFromConsensusError =
            <OpPooledTransaction as PoolTransaction>::TryFromConsensusError;
        type Consensus = <OpPooledTransaction as PoolTransaction>::Consensus;
        type Pooled = <OpPooledTransaction as PoolTransaction>::Pooled;

        fn clone_into_consensus(&self) -> Recovered<Self::Consensus> {
            self.inner.clone_into_consensus()
        }
        fn consensus_ref(&self) -> Recovered<&Self::Consensus> {
            self.inner.consensus_ref()
        }
        fn into_consensus(self) -> Recovered<Self::Consensus> {
            self.inner.into_consensus()
        }
        fn into_consensus_with2718(self) -> WithEncoded<Recovered<Self::Consensus>> {
            self.inner.into_consensus_with2718()
        }
        fn from_pooled(_pooled: Recovered<Self::Pooled>) -> Self {
            unreachable!("payload builder loop never calls from_pooled on the wrapper")
        }
        fn hash(&self) -> &TxHash {
            self.inner.hash()
        }
        fn sender(&self) -> Address {
            self.inner.sender()
        }
        fn sender_ref(&self) -> &Address {
            self.inner.sender_ref()
        }
        fn cost(&self) -> &U256 {
            self.inner.cost()
        }
        fn encoded_length(&self) -> usize {
            self.inner.encoded_length()
        }
    }

    impl OpPooledTx for ForcedTipPooledTx {
        fn encoded_2718(&self) -> Cow<'_, Bytes> {
            OpPooledTx::encoded_2718(&self.inner)
        }
    }

    let signer = Address::repeat_byte(0x11);
    let inner = op_pooled_tx(0, signer, Address::repeat_byte(0x22));
    // The inner consensus tx has `max_priority_fee_per_gas = 1`, so its natural effective tip at
    // `base_fee = 0` is 1 wei/gas. Forcing the wrapper's priority fee to something much larger
    // guarantees the wrapper-derived total differs from the consensus-tx-derived total.
    let natural_priority_fee: u128 = 1;
    let forced_priority_fee: u128 = 1_000_000;
    let tx = ForcedTipPooledTx { inner, forced_priority_fee };

    let (info, _) = run_execute_best_transactions(signer, vec![tx], None, None);

    let gas_used = U256::from(info.cumulative_gas_used);
    let expected_fees = U256::from(forced_priority_fee) * gas_used;
    let natural_fees = U256::from(natural_priority_fee) * gas_used;
    // Sanity check: if a future change reorders the read back behind `into_consensus()`, the
    // builder would compute `natural_fees`. Asserting both sides defends against that even if
    // somebody later tweaks the helper's fee fields and happens to land on `forced_priority_fee`.
    assert_eq!(info.total_fees, expected_fees);
    assert_ne!(info.total_fees, natural_fees);
}
