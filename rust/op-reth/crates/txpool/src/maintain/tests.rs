use super::*;
use crate::{
    InvalidCrossTx, OpPooledTransaction,
    interop_filter::{CROSS_L2_INBOX_ADDRESS, InteropTxValidatorError},
};
use alloy_consensus::{SignableTransaction, TxEip1559, transaction::Recovered};
use alloy_eips::eip2930::{AccessList, AccessListItem};
use alloy_primitives::{Address, B256, Signature, TxKind, U256};
use op_alloy_rpc_types::SuperchainDAError;
use reth_execution_types::{Chain, ExecutionOutcome};
use reth_optimism_primitives::{OpBlock, OpPrimitives};
use reth_primitives_traits::RecoveredBlock;
use reth_transaction_pool::{
    CoinbaseTipOrdering, Pool, PoolConfig, TransactionOrigin, blobstore::InMemoryBlobStore,
    noop::MockTransactionValidator,
};
use rstest::rstest;
use std::{collections::BTreeMap, sync::Arc};

/// A pool of [`OpPooledTransaction`] backed by the always-valid mock validator, so a built tx
/// lands in the pending subpool and is visible to the maintenance sweep.
type TestOpPool = Pool<
    MockTransactionValidator<OpPooledTransaction>,
    CoinbaseTipOrdering<OpPooledTransaction>,
    InMemoryBlobStore,
>;

fn build_pool() -> TestOpPool {
    Pool::new(
        MockTransactionValidator::default(),
        CoinbaseTipOrdering::default(),
        InMemoryBlobStore::default(),
        PoolConfig::default(),
    )
}

/// Builds an interop [`OpPooledTransaction`]: an EIP-1559 tx whose access list targets the
/// cross-L2 inbox. The signature is a dummy; the mock validator does not verify it and the
/// sender is set explicitly.
fn interop_pooled_tx() -> OpPooledTransaction {
    let tx = TxEip1559 {
        chain_id: 10,
        nonce: 0,
        gas_limit: 21_000,
        max_fee_per_gas: 20_000_000_000,
        max_priority_fee_per_gas: 1,
        to: TxKind::Call(Address::ZERO),
        value: U256::ZERO,
        access_list: AccessList(vec![AccessListItem {
            address: CROSS_L2_INBOX_ADDRESS,
            storage_keys: vec![B256::ZERO],
        }]),
        input: Default::default(),
    };
    let signed = tx.into_signed(Signature::new(U256::from(1u64), U256::from(1u64), false));
    OpPooledTransaction::from_pooled(Recovered::new_unchecked(
        op_alloy_consensus::OpPooledTransaction::Eip1559(signed),
        Address::with_last_byte(1),
    ))
}

/// Builds a non-interop EIP-1559 [`OpPooledTransaction`]: same shape as [`interop_pooled_tx`]
/// but with an empty access list, so `is_interop_tx` is false and the reorg sweep must leave it
/// in place. Uses a distinct sender so it shares no nonce sequence with the interop tx.
fn non_interop_pooled_tx() -> OpPooledTransaction {
    let tx = TxEip1559 {
        chain_id: 10,
        nonce: 0,
        gas_limit: 21_000,
        max_fee_per_gas: 20_000_000_000,
        max_priority_fee_per_gas: 1,
        to: TxKind::Call(Address::ZERO),
        value: U256::ZERO,
        access_list: AccessList::default(),
        input: Default::default(),
    };
    let signed = tx.into_signed(Signature::new(U256::from(1u64), U256::from(1u64), false));
    OpPooledTransaction::from_pooled(Recovered::new_unchecked(
        op_alloy_consensus::OpPooledTransaction::Eip1559(signed),
        Address::with_last_byte(2),
    ))
}

/// A minimal reorg notification. The interop reorg sweep evicts unconditionally and never
/// inspects the reverted/committed chains, so a single default block on both sides suffices.
fn reorg_event() -> CanonStateNotification<OpPrimitives> {
    let block: RecoveredBlock<OpBlock> = Default::default();
    let chain = Arc::new(Chain::new([block], ExecutionOutcome::default(), BTreeMap::new()));
    CanonStateNotification::Reorg { old: chain.clone(), new: chain }
}

/// A minimal commit notification with a single default block.
fn commit_event() -> CanonStateNotification<OpPrimitives> {
    let block: RecoveredBlock<OpBlock> = Default::default();
    let chain = Arc::new(Chain::new([block], ExecutionOutcome::default(), BTreeMap::new()));
    CanonStateNotification::Commit { new: chain }
}

fn definitive_invalid() -> InvalidCrossTx {
    InvalidCrossTx::ValidationError(InteropTxValidatorError::InvalidEntry(
        SuperchainDAError::ConflictingData,
    ))
}

fn transient_failure() -> InvalidCrossTx {
    InvalidCrossTx::ValidationError(InteropTxValidatorError::QuorumNotReached {
        received: 0,
        required: 1,
    })
}

struct MockInteropFilter {
    failsafe_enabled: bool,
    invalid_result: fn() -> InvalidCrossTx,
}

impl MockInteropFilter {
    fn new_definitive_invalid() -> Self {
        Self { failsafe_enabled: false, invalid_result: definitive_invalid }
    }

    fn new_transient_failure() -> Self {
        Self { failsafe_enabled: false, invalid_result: transient_failure }
    }

    fn new_failsafe_enabled() -> Self {
        Self { failsafe_enabled: true, invalid_result: transient_failure }
    }
}

#[async_trait]
impl InteropFilter for MockInteropFilter {
    fn is_failsafe_enabled_cached(&self) -> bool {
        self.failsafe_enabled
    }

    async fn is_failsafe_enabled(&self) -> Result<bool, InteropTxValidatorError> {
        Ok(self.failsafe_enabled)
    }

    async fn revalidate_interop_txs<Tx>(
        &self,
        txs_to_revalidate: Vec<Tx>,
        _current_timestamp: u64,
        _revalidation_window: u64,
    ) -> Vec<InteropValidationResult<Tx>>
    where
        Tx: PoolTransaction + Send,
    {
        let invalid_result = self.invalid_result;
        txs_to_revalidate
            .into_iter()
            .map(|tx| InteropValidationResult::Invalid(tx, invalid_result()))
            .collect()
    }
}

/// Adds a stale-but-still-valid interop tx with the given `origin` to the pool and runs the
/// sweep against `filter`, returning whether the tx survived.
async fn stale_interop_tx_survives_sweep(
    filter: &MockInteropFilter,
    origin: TransactionOrigin,
) -> bool {
    let pool = build_pool();
    let tx = interop_pooled_tx();
    let hash = *tx.hash();
    let timestamp = 1_000;
    // Deadline in (timestamp, timestamp + OFFSET_TIME]: stale (triggers revalidation) yet still
    // valid (not expired), so the sweep revalidates rather than expiry-evicts it.
    tx.set_interop_deadline(timestamp + 30);
    pool.add_transaction(origin, tx).await.unwrap();
    assert!(pool.get(&hash).is_some(), "tx should be pooled before the sweep");

    revalidate_stale_interop_txs(&pool, filter, timestamp, &MaintainPoolInteropMetrics::default())
        .await;

    pool.get(&hash).is_some()
}

#[rstest]
#[case::external(TransactionOrigin::External)]
#[case::private(TransactionOrigin::Private)]
#[tokio::test(flavor = "multi_thread")]
async fn sweep_evicts_definitive_invalid_tx(#[case] origin: TransactionOrigin) {
    // The filter returns a definitive invalid verdict (InvalidEntry). The sweep must remove the
    // tx from the pool. This fails if eviction is gated on `is_bad_transaction`.
    let filter = MockInteropFilter::new_definitive_invalid();
    assert!(
        !stale_interop_tx_survives_sweep(&filter, origin).await,
        "a definitive invalid verdict must be evicted from the pool by the sweep"
    );
}

#[tokio::test(flavor = "multi_thread")]
async fn reorg_evicts_pooled_interop_txs() {
    // The filter answers transient errors, so passing proves eviction is independent of
    // revalidation: the interop tx must go, the plain tx must stay.
    let pool = build_pool();
    let interop = interop_pooled_tx();
    let interop_hash = *interop.hash();
    let plain = non_interop_pooled_tx();
    let plain_hash = *plain.hash();
    pool.add_transaction(TransactionOrigin::External, interop).await.unwrap();
    pool.add_transaction(TransactionOrigin::External, plain).await.unwrap();
    assert!(pool.get(&interop_hash).is_some(), "interop tx should be pooled before the reorg");
    assert!(pool.get(&plain_hash).is_some(), "plain tx should be pooled before the reorg");

    let events = futures_util::stream::iter(vec![reorg_event()]);

    maintain_transaction_pool_interop(
        pool.clone(),
        events,
        MockInteropFilter::new_transient_failure(),
    )
    .await;

    assert!(pool.get(&interop_hash).is_none(), "interop tx must be evicted on reorg");
    assert!(pool.get(&plain_hash).is_some(), "non-interop tx must survive the reorg");
}

#[tokio::test(flavor = "multi_thread")]
async fn reorg_evicts_non_propagatable_interop_tx() {
    // A `Private`-origin interop tx is hidden from `pooled_transactions()` but still buildable,
    // so the reorg sweep must reach it via `all_transactions()` and evict it too.
    let pool = build_pool();
    let interop = interop_pooled_tx();
    let interop_hash = *interop.hash();
    pool.add_transaction(TransactionOrigin::Private, interop).await.unwrap();
    assert!(pool.get(&interop_hash).is_some(), "interop tx should be pooled before the reorg");

    let events = futures_util::stream::iter(vec![reorg_event()]);

    maintain_transaction_pool_interop(
        pool.clone(),
        events,
        MockInteropFilter::new_transient_failure(),
    )
    .await;

    assert!(
        pool.get(&interop_hash).is_none(),
        "non-propagatable interop tx must be evicted on reorg"
    );
}

#[tokio::test(flavor = "multi_thread")]
async fn commit_with_failsafe_evicts_all_interop_txs() {
    // With failsafe active, a block commit must evict every interop tx, not revalidate. The
    // interop tx carries no deadline, so the revalidation path would leave it untouched;
    // requiring it to be gone proves the commit arm took the failsafe branch.
    let pool = build_pool();
    let interop = interop_pooled_tx();
    let interop_hash = *interop.hash();
    let plain = non_interop_pooled_tx();
    let plain_hash = *plain.hash();
    pool.add_transaction(TransactionOrigin::External, interop).await.unwrap();
    pool.add_transaction(TransactionOrigin::External, plain).await.unwrap();

    let events = futures_util::stream::iter(vec![commit_event()]);

    maintain_transaction_pool_interop(
        pool.clone(),
        events,
        MockInteropFilter::new_failsafe_enabled(),
    )
    .await;

    assert!(pool.get(&interop_hash).is_none(), "failsafe commit must evict the interop tx");
    assert!(pool.get(&plain_hash).is_some(), "non-interop tx must survive the failsafe commit");
}

#[tokio::test(flavor = "multi_thread")]
async fn sweep_retains_tx_on_transient_failure() {
    // A JSON-RPC internal error (-32603) is a non-response: with a single endpoint the quorum
    // is not reached. A flapping/unreachable interop filter must not drain the pool, so the
    // tx stays.
    let filter = MockInteropFilter::new_transient_failure();
    assert!(
        stale_interop_tx_survives_sweep(&filter, TransactionOrigin::External).await,
        "a transient/non-decisive verdict must NOT evict the tx"
    );
}
