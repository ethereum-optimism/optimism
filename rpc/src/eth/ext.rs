//! Eth API extension.

use crate::{error::TxConditionalErr, OpEthApiError, SequencerClient};
use alloy_consensus::BlockHeader;
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::{Bytes, StorageKey, B256, U256};
use alloy_rpc_types_eth::erc4337::{AccountStorage, TransactionConditional};
use jsonrpsee_core::RpcResult;
use reth_optimism_txpool::conditional::MaybeConditionalTransaction;
use reth_rpc_eth_api::L2EthApiExtServer;
use reth_rpc_eth_types::utils::recover_raw_transaction;
use reth_storage_api::{BlockReaderIdExt, StateProviderFactory};
use reth_transaction_pool::{PoolTransaction, TransactionOrigin, TransactionPool};
use std::sync::Arc;
use tokio::sync::Semaphore;

/// Maximum execution const for conditional transactions.
const MAX_CONDITIONAL_EXECUTION_COST: u64 = 5000;

const MAX_CONCURRENT_CONDITIONAL_VALIDATIONS: usize = 3;

/// OP-Reth `Eth` API extensions implementation.
///
/// Separate from [`super::OpEthApi`] to allow to enable it conditionally,
#[derive(Clone, Debug)]
pub struct OpEthExtApi<Pool, Provider> {
    /// Sequencer client, configured to forward submitted transactions to sequencer of given OP
    /// network.
    sequencer_client: Option<SequencerClient>,
    inner: Arc<OpEthExtApiInner<Pool, Provider>>,
}

impl<Pool, Provider> OpEthExtApi<Pool, Provider>
where
    Provider: BlockReaderIdExt + StateProviderFactory + Clone + 'static,
{
    /// Creates a new [`OpEthExtApi`].
    pub fn new(sequencer_client: Option<SequencerClient>, pool: Pool, provider: Provider) -> Self {
        let inner = Arc::new(OpEthExtApiInner::new(pool, provider));
        Self { sequencer_client, inner }
    }

    /// Returns the configured sequencer client, if any.
    const fn sequencer_client(&self) -> Option<&SequencerClient> {
        self.sequencer_client.as_ref()
    }

    #[inline]
    fn pool(&self) -> &Pool {
        self.inner.pool()
    }

    #[inline]
    fn provider(&self) -> &Provider {
        self.inner.provider()
    }

    /// Validates the conditional's `known accounts` settings against the current state.
    async fn validate_known_accounts(
        &self,
        condition: &TransactionConditional,
    ) -> Result<(), TxConditionalErr> {
        if condition.known_accounts.is_empty() {
            return Ok(());
        }

        let _permit =
            self.inner.validation_semaphore.acquire().await.map_err(TxConditionalErr::internal)?;

        let state = self
            .provider()
            .state_by_block_number_or_tag(BlockNumberOrTag::Latest)
            .map_err(TxConditionalErr::internal)?;

        for (address, storage) in &condition.known_accounts {
            match storage {
                AccountStorage::Slots(slots) => {
                    for (slot, expected_value) in slots {
                        let current = state
                            .storage(*address, StorageKey::from(*slot))
                            .map_err(TxConditionalErr::internal)?
                            .unwrap_or_default();

                        if current != U256::from_be_bytes(**expected_value) {
                            return Err(TxConditionalErr::StorageValueMismatch);
                        }
                    }
                }
                AccountStorage::RootHash(expected_root) => {
                    let actual_root = state
                        .storage_root(*address, Default::default())
                        .map_err(TxConditionalErr::internal)?;

                    if *expected_root != actual_root {
                        return Err(TxConditionalErr::StorageRootMismatch);
                    }
                }
            }
        }

        Ok(())
    }
}

#[async_trait::async_trait]
impl<Pool, Provider> L2EthApiExtServer for OpEthExtApi<Pool, Provider>
where
    Provider: BlockReaderIdExt + StateProviderFactory + Clone + 'static,
    Pool: TransactionPool<Transaction: MaybeConditionalTransaction> + 'static,
{
    async fn send_raw_transaction_conditional(
        &self,
        bytes: Bytes,
        condition: TransactionConditional,
    ) -> RpcResult<B256> {
        // calculate and validate cost
        let cost = condition.cost();
        if cost > MAX_CONDITIONAL_EXECUTION_COST {
            return Err(TxConditionalErr::ConditionalCostExceeded.into());
        }

        let recovered_tx = recover_raw_transaction(&bytes).map_err(|_| {
            OpEthApiError::Eth(reth_rpc_eth_types::EthApiError::FailedToDecodeSignedTransaction)
        })?;

        let mut tx = <Pool as TransactionPool>::Transaction::from_pooled(recovered_tx);

        // get current header
        let header_not_found = || {
            OpEthApiError::Eth(reth_rpc_eth_types::EthApiError::HeaderNotFound(
                alloy_eips::BlockId::Number(BlockNumberOrTag::Latest),
            ))
        };
        let header = self
            .provider()
            .latest_header()
            .map_err(|_| header_not_found())?
            .ok_or_else(header_not_found)?;

        // Ensure that the condition can still be met by checking the max bounds
        if condition.has_exceeded_block_number(header.header().number()) ||
            condition.has_exceeded_timestamp(header.header().timestamp())
        {
            return Err(TxConditionalErr::InvalidCondition.into());
        }

        // Validate Account
        self.validate_known_accounts(&condition).await?;

        if let Some(sequencer) = self.sequencer_client() {
            // If we have a sequencer client, forward the transaction
            let _ = sequencer
                .forward_raw_transaction_conditional(bytes.as_ref(), condition)
                .await
                .map_err(OpEthApiError::Sequencer)?;
            Ok(*tx.hash())
        } else {
            // otherwise, add to pool with the appended conditional
            tx.set_conditional(condition);
            let hash =
                self.pool().add_transaction(TransactionOrigin::Private, tx).await.map_err(|e| {
                    OpEthApiError::Eth(reth_rpc_eth_types::EthApiError::PoolError(e.into()))
                })?;

            Ok(hash)
        }
    }
}

#[derive(Debug)]
struct OpEthExtApiInner<Pool, Provider> {
    /// The transaction pool of the node.
    pool: Pool,
    /// The provider type used to interact with the node.
    provider: Provider,
    /// The semaphore used to limit the number of concurrent conditional validations.
    validation_semaphore: Semaphore,
}

impl<Pool, Provider> OpEthExtApiInner<Pool, Provider> {
    fn new(pool: Pool, provider: Provider) -> Self {
        Self {
            pool,
            provider,
            validation_semaphore: Semaphore::new(MAX_CONCURRENT_CONDITIONAL_VALIDATIONS),
        }
    }

    #[inline]
    const fn pool(&self) -> &Pool {
        &self.pool
    }

    #[inline]
    const fn provider(&self) -> &Provider {
        &self.provider
    }
}
