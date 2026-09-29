//! Re-prices the OP fees of pending transactions on every new canonical block.

use crate::{OpPooledTx, validator::op_fee_reservation};
use alloy_consensus::{BlockHeader, Transaction};
use alloy_primitives::{Address, TxHash, U256};
use futures_util::{FutureExt, Stream, StreamExt, future::BoxFuture};
use op_revm::L1BlockInfo;
use reth_chain_state::CanonStateNotification;
use reth_chainspec::ChainSpecProvider;
use reth_metrics::{Metrics, metrics::Counter};
use reth_optimism_forks::OpHardforks;
use reth_primitives_traits::{BlockBody, NodePrimitives};
use reth_storage_api::{AccountReader, StateProviderFactory};
use reth_transaction_pool::{TransactionPool, ValidPoolTransaction};
use std::{collections::HashMap, sync::Arc};
use tracing::{debug, warn};

/// Transaction pool OP fee maintenance metrics
#[derive(Metrics)]
#[metrics(scope = "transaction_pool")]
struct MaintainPoolOpFeesMetrics {
    /// Number of transactions removed from the pool because their sender can no longer pay their
    /// OP fees at the current L1 fee, including the removed transactions' descendants.
    removed_tx_op_fees: Counter,
}

/// Returns a spawnable future that re-prices the OP fees of pending transactions on every new
/// canonical block.
pub fn maintain_transaction_pool_op_fees_future<N, Pool, Client, St>(
    pool: Pool,
    client: Client,
    events: St,
) -> BoxFuture<'static, ()>
where
    N: NodePrimitives,
    Pool: TransactionPool + 'static,
    Pool::Transaction: OpPooledTx,
    Client: StateProviderFactory + ChainSpecProvider<ChainSpec: OpHardforks> + 'static,
    St: Stream<Item = CanonStateNotification<N>> + Send + Unpin + 'static,
{
    async move {
        maintain_transaction_pool_op_fees(pool, client, events).await;
    }
    .boxed()
}

/// Removes pending transactions whose sender can no longer pay their OP fees.
///
/// The validator reserves the L1 data fee and operator fee in `cost()` at admission, and reth's
/// pool only re-checks a sender's cumulative `cost()` against its balance when the account changes.
/// If the L1 fee rises after admission, a later nonce can stay pending although execution rejects
/// it for insufficient funds: every payload build retries it, and lifetime eviction never removes
/// it because that only covers queued transactions. This task prices every pending transaction at
/// the new tip's L1 fee and removes the first one per sender that the sender can no longer cover,
/// along with its descendants — the pool-side counterpart of op-geth's `demoteUnexecutables`.
pub async fn maintain_transaction_pool_op_fees<N, Pool, Client, St>(
    pool: Pool,
    client: Client,
    mut events: St,
) where
    N: NodePrimitives,
    Pool: TransactionPool,
    Pool::Transaction: OpPooledTx,
    Client: StateProviderFactory + ChainSpecProvider<ChainSpec: OpHardforks>,
    St: Stream<Item = CanonStateNotification<N>> + Send + Unpin + 'static,
{
    let metrics = MaintainPoolOpFeesMetrics::default();
    while let Some(event) = events.next().await {
        let tip = event.tip();
        // Every OP block starts with the L1-info deposit. Without one there is no L1 fee to price.
        let Some(Ok(l1_block_info)) =
            tip.body().transactions().first().map(reth_optimism_evm::extract_l1_info_from_tx)
        else {
            continue;
        };
        let state = match client.state_by_block_hash(tip.hash()) {
            Ok(state) => state,
            Err(err) => {
                warn!(target: "txpool", %err, block = %tip.hash(), "Failed to open state to re-price pending OP fees");
                continue;
            }
        };

        let unaffordable = unaffordable_pending_transactions(
            pool.pending_transactions(),
            l1_block_info,
            client.chain_spec(),
            tip.timestamp(),
            |sender| match state.basic_account(&sender) {
                Ok(account) => Some(
                    account.map_or((U256::ZERO, 0), |account| (account.balance, account.nonce)),
                ),
                Err(err) => {
                    warn!(target: "txpool", %err, %sender, "Failed to read sender account to re-price pending OP fees");
                    None
                }
            },
        );
        if !unaffordable.is_empty() {
            let removed = pool.remove_transactions_and_descendants(unaffordable);
            debug!(target: "txpool", count = removed.len(), "Removed pending transactions that can no longer pay their OP fees");
            metrics.removed_tx_op_fees.increment(removed.len() as u64);
        }
    }
}

/// Returns, per sender, the first pending transaction whose cumulative worst-case cost — L2 cost
/// plus OP fees priced with `l1_block_info` at `timestamp` — exceeds the sender's balance.
///
/// `account` returns a sender's `(balance, nonce)`, or `None` to skip the sender (e.g. when the
/// state lookup failed). Transactions below the account nonce are already mined and are skipped.
fn unaffordable_pending_transactions<T, C>(
    pending: Vec<Arc<ValidPoolTransaction<T>>>,
    mut l1_block_info: L1BlockInfo,
    chain_spec: C,
    timestamp: u64,
    mut account: impl FnMut(Address) -> Option<(U256, u64)>,
) -> Vec<TxHash>
where
    T: OpPooledTx,
    C: OpHardforks + Clone,
{
    let mut by_sender: HashMap<Address, Vec<Arc<ValidPoolTransaction<T>>>> = HashMap::new();
    for tx in pending {
        by_sender.entry(tx.sender()).or_default().push(tx);
    }

    let mut unaffordable = Vec::new();
    for (sender, mut txs) in by_sender {
        let Some((balance, state_nonce)) = account(sender) else { continue };
        txs.sort_unstable_by_key(|tx| tx.nonce());

        let mut cumulative_cost = U256::ZERO;
        for pooled in &txs {
            let tx = &pooled.transaction;
            if tx.nonce() < state_nonce {
                continue;
            }
            let encoded = tx.encoded_2718();
            let Ok(op_fees) = op_fee_reservation(
                &mut l1_block_info,
                chain_spec.clone(),
                timestamp,
                &encoded,
                tx.gas_limit(),
            ) else {
                break;
            };
            cumulative_cost = cumulative_cost.saturating_add(l2_cost(tx)).saturating_add(op_fees);
            if cumulative_cost > balance {
                unaffordable.push(*tx.hash());
                break;
            }
        }
    }
    unaffordable
}

/// The most a transaction can spend on L2 execution: `gas_limit * max_fee_per_gas + value`, the
/// same bound the EVM checks the balance against before adding the OP fees.
fn l2_cost(tx: &impl Transaction) -> U256 {
    U256::from(tx.max_fee_per_gas())
        .saturating_mul(U256::from(tx.gas_limit()))
        .saturating_add(tx.value())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{OpPooledTransaction, OpTransactionValidator};
    use alloy_consensus::{SignableTransaction, TxEip1559, transaction::Recovered};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Signature, TxKind};
    use reth_optimism_chainspec::OP_MAINNET;
    use reth_optimism_evm::{OpEvmConfig, RethL1BlockInfo};
    use reth_optimism_primitives::{OpPrimitives, OpTransactionSigned};
    use reth_provider::test_utils::{ExtendedAccount, MockEthProvider};
    use reth_transaction_pool::{
        CoinbaseTipOrdering, Pool, PoolConfig, PoolTransaction, TransactionOrigin,
        blobstore::InMemoryBlobStore, validate::EthTransactionValidatorBuilder,
    };

    const SIGNER: Address = Address::with_last_byte(1);

    fn make_tx(nonce: u64) -> OpPooledTransaction {
        let tx: OpTransactionSigned = TxEip1559 {
            chain_id: 10,
            nonce,
            gas_limit: 21_000,
            max_fee_per_gas: 1_000_000_000,
            to: TxKind::Call(Address::with_last_byte(0x42)),
            ..Default::default()
        }
        .into_signed(Signature::test_signature())
        .into();
        let recovered = Recovered::new_unchecked(tx, SIGNER);
        let len = recovered.encode_2718_len();
        OpPooledTransaction::new(recovered, len)
    }

    /// L1 block info whose data fee dominates a transfer's L2 cost, as on ink-mainnet.
    fn l1_block_info() -> L1BlockInfo {
        L1BlockInfo {
            l1_base_fee: U256::from(1_000_000_000_000u64),
            l1_base_fee_scalar: U256::from(1_000_000),
            ..Default::default()
        }
    }

    /// A pool whose validator saw no L1 fee at admission (genesis has no L1-info deposit), holding
    /// nonces 0 and 1 from a sender funded with `balance`.
    async fn pool_with_two_pending(
        balance: U256,
    ) -> impl TransactionPool<Transaction = OpPooledTransaction> {
        let client = MockEthProvider::<OpPrimitives>::new()
            .with_chain_spec(OP_MAINNET.clone())
            .with_genesis_block();
        client.add_account(SIGNER, ExtendedAccount::new(0, balance));
        let inner =
            EthTransactionValidatorBuilder::new(client, OpEvmConfig::optimism(OP_MAINNET.clone()))
                .build(InMemoryBlobStore::default());
        let pool = Pool::new(
            OpTransactionValidator::new(inner),
            CoinbaseTipOrdering::default(),
            InMemoryBlobStore::default(),
            PoolConfig::default(),
        );
        pool.add_transaction(TransactionOrigin::External, make_tx(0)).await.unwrap();
        pool.add_transaction(TransactionOrigin::External, make_tx(1)).await.unwrap();
        pool
    }

    fn pending_nonces(pool: &impl TransactionPool) -> Vec<u64> {
        let mut nonces: Vec<_> = pool.pending_transactions().iter().map(|tx| tx.nonce()).collect();
        nonces.sort_unstable();
        nonces
    }

    fn l1_fee(tx: &OpPooledTransaction) -> U256 {
        l1_block_info().l1_tx_data_fee(OP_MAINNET.clone(), 0, tx.encoded_2718(), false).unwrap()
    }

    /// The L1 fee rose after admission: both nonces were admitted as pending, but at the new fee
    /// the sender can only pay for nonce 0. Without re-pricing, nonce 1 stays pending and every
    /// payload build fails it for insufficient funds.
    #[tokio::test]
    async fn fee_rise_removes_descendant_the_sender_can_no_longer_afford() {
        let tx0 = make_tx(0);
        let fee = l1_fee(&tx0);
        let balance = *tx0.cost() * U256::from(2) + fee + fee / U256::from(2);
        let pool = pool_with_two_pending(balance).await;
        assert_eq!(pending_nonces(&pool), vec![0, 1], "both admitted as pending at the old fee");

        let unaffordable = unaffordable_pending_transactions(
            pool.pending_transactions(),
            l1_block_info(),
            OP_MAINNET.clone(),
            0,
            |_| Some((balance, 0)),
        );
        pool.remove_transactions_and_descendants(unaffordable);

        assert_eq!(pending_nonces(&pool), vec![0]);
    }

    #[tokio::test]
    async fn keeps_pending_transactions_the_sender_can_still_afford() {
        let tx0 = make_tx(0);
        let balance = (*tx0.cost() + l1_fee(&tx0)) * U256::from(2);
        let pool = pool_with_two_pending(balance).await;

        let unaffordable = unaffordable_pending_transactions(
            pool.pending_transactions(),
            l1_block_info(),
            OP_MAINNET.clone(),
            0,
            |_| Some((balance, 0)),
        );

        assert!(unaffordable.is_empty());
    }

    /// Nonce 0 was just mined: it is still in the pool until reth's own maintenance catches up,
    /// but must not count against the sender's remaining balance.
    #[tokio::test]
    async fn skips_transactions_below_the_account_nonce() {
        let tx1 = make_tx(1);
        let balance = *tx1.cost() + l1_fee(&tx1);
        let pool = pool_with_two_pending(balance * U256::from(2)).await;

        let unaffordable = unaffordable_pending_transactions(
            pool.pending_transactions(),
            l1_block_info(),
            OP_MAINNET.clone(),
            0,
            |_| Some((balance, 1)),
        );

        assert!(unaffordable.is_empty());
    }

    #[tokio::test]
    async fn skips_senders_whose_account_lookup_failed() {
        let pool = pool_with_two_pending(U256::from(u64::MAX)).await;

        let unaffordable = unaffordable_pending_transactions(
            pool.pending_transactions(),
            l1_block_info(),
            OP_MAINNET.clone(),
            0,
            |_| None,
        );

        assert!(unaffordable.is_empty());
    }
}
