//! Eth API extension.

use crate::{error::TxConditionalErr, OpEthApiError, SequencerClient};
use alloy_consensus::BlockHeader;
use alloy_eips::BlockNumberOrTag;
use alloy_primitives::{Bytes, B256};
use alloy_rpc_types_eth::erc4337::TransactionConditional;
use jsonrpsee_core::RpcResult;
use reth_optimism_txpool::conditional::MaybeConditionalTransaction;
use reth_provider::{BlockReaderIdExt, StateProviderFactory};
use reth_rpc_eth_api::L2EthApiExtServer;
use reth_rpc_eth_types::utils::recover_raw_transaction;
use reth_transaction_pool::{PoolTransaction, TransactionOrigin, TransactionPool};
use std::sync::Arc;

/// Maximum execution const for conditional transactions.
const MAX_CONDITIONAL_EXECUTION_COST: u64 = 5000;

#[derive(Debug)]
struct OpEthExtApiInner<Pool, Provider> {
    /// The transaction pool of the node.
    pool: Pool,
    /// The provider type used to interact with the node.
    provider: Provider,
}

impl<Pool, Provider> OpEthExtApiInner<Pool, Provider> {
    fn new(pool: Pool, provider: Provider) -> Self {
        Self { pool, provider }
    }

    #[inline]
    fn pool(&self) -> &Pool {
        &self.pool
    }

    #[inline]
    fn provider(&self) -> &Provider {
        &self.provider
    }
}

/// OP-Reth `Eth` API extensions implementation.
///
/// Separate from [`super::OpEthApi`] to allow to enable it conditionally,
#[derive(Clone, Debug)]
#[allow(dead_code)]
pub struct OpEthExtApi<Pool, Provider> {
    /// Sequencer client, configured to forward submitted transactions to sequencer of given OP
    /// network.
    sequencer_client: Option<SequencerClient>,
    inner: Arc<OpEthExtApiInner<Pool, Provider>>,
}

impl<Pool, Provider> OpEthExtApi<Pool, Provider>
where
    Provider: BlockReaderIdExt + Clone + 'static,
{
    /// Creates a new [`OpEthExtApi`].
    pub fn new(sequencer_client: Option<SequencerClient>, pool: Pool, provider: Provider) -> Self {
        let inner = Arc::new(OpEthExtApiInner::new(pool, provider));
        Self { sequencer_client, inner }
    }

    /// Returns the configured sequencer client, if any.
    fn sequencer_client(&self) -> Option<&SequencerClient> {
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

        // TODO: check condition against state

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
