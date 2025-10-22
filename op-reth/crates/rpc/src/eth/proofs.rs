//! Historical proofs RPC server implementation.

use crate::metrics::EthApiExtMetrics;
use alloy_eips::BlockId;
use alloy_primitives::Address;
use alloy_rpc_types_eth::EIP1186AccountProofResponse;
use alloy_serde::JsonStorageKey;
use async_trait::async_trait;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee_core::RpcResult;
use jsonrpsee_types::error::ErrorObject;
use reth_optimism_trie::{provider::OpProofsStateProviderRef, OpProofsStorage, OpProofsStore};
use reth_provider::{BlockIdReader, ProviderError, ProviderResult, StateProofProvider};
use reth_rpc_api::eth::helpers::FullEthApi;
use reth_rpc_eth_types::EthApiError;
use std::time::Instant;

#[cfg_attr(not(test), rpc(server, namespace = "eth"))]
#[cfg_attr(test, rpc(server, client, namespace = "eth"))]
pub trait EthApiOverride {
    /// Returns the account and storage values of the specified account including the Merkle-proof.
    /// This call can be used to verify that the data you are pulling from is not tampered with.
    #[method(name = "getProof")]
    async fn get_proof(
        &self,
        address: Address,
        keys: Vec<JsonStorageKey>,
        block_number: Option<BlockId>,
    ) -> RpcResult<EIP1186AccountProofResponse>;
}

#[derive(Debug)]
/// Overrides applied to the `eth_` namespace of the RPC API for historical proofs ExEx.
pub struct EthApiExt<Eth, P> {
    eth_api: Eth,
    preimage_store: OpProofsStorage<P>,
    metrics: EthApiExtMetrics,
}

impl<Eth, P> EthApiExt<Eth, P>
where
    Eth: FullEthApi + Send + Sync + 'static,
    ErrorObject<'static>: From<Eth::Error>,
    P: OpProofsStore + Clone + 'static,
{
    /// Creates a new instance with the provided ETH API and proof store.
    pub fn new(eth_api: Eth, preimage_store: OpProofsStorage<P>) -> Self {
        let metrics = EthApiExtMetrics::default();
        Self { eth_api, preimage_store, metrics }
    }

    async fn state_provider(
        &self,
        block_id: Option<BlockId>,
    ) -> ProviderResult<impl StateProofProvider> {
        let block_id = block_id.unwrap_or_default();
        // Check whether the distance to the block exceeds the maximum configured window.
        let block_number = self
            .eth_api
            .provider()
            .block_number_for_id(block_id)?
            .ok_or(EthApiError::HeaderNotFound(block_id))
            .map_err(ProviderError::other)?;

        let historical_provider =
            self.eth_api.state_at_block_id(block_id).await.map_err(ProviderError::other)?;

        let (Some((latest_block_number, _)), Some((earliest_block_number, _))) = (
            self.preimage_store
                .get_latest_block_number()
                .await
                .map_err(|e| ProviderError::Database(e.into()))?,
            self.preimage_store
                .get_earliest_block_number()
                .await
                .map_err(|e| ProviderError::Database(e.into()))?,
        ) else {
            // if no earliest block, db is empty - use historical provider
            return Ok(historical_provider as Box<dyn StateProofProvider>);
        };

        if block_number < earliest_block_number || block_number > latest_block_number {
            return Ok(historical_provider as Box<dyn StateProofProvider>);
        }

        let external_overlay_provider =
            OpProofsStateProviderRef::new(historical_provider, &self.preimage_store, block_number);

        Ok(Box::new(external_overlay_provider))
    }
}

#[async_trait]
impl<Eth, P> EthApiOverrideServer for EthApiExt<Eth, P>
where
    Eth: FullEthApi + Send + Sync + 'static,
    ErrorObject<'static>: From<Eth::Error>,
    P: OpProofsStore + Clone + 'static,
{
    async fn get_proof(
        &self,
        address: Address,
        keys: Vec<JsonStorageKey>,
        block_number: Option<BlockId>,
    ) -> RpcResult<EIP1186AccountProofResponse> {
        self.metrics.get_proof_requests.increment(1);
        let start = Instant::now();

        let result = async {
            let state = self.state_provider(block_number).await.map_err(Into::into)?;
            let storage_keys: Vec<_> = keys.iter().map(|key| key.as_b256()).collect();

            let proof =
                state.proof(Default::default(), address, &storage_keys).map_err(Into::into)?;

            Ok(proof.into_eip1186_response(keys))
        }
        .await;

        match &result {
            Ok(_) => {
                self.metrics.get_proof_latency.record(start.elapsed().as_secs_f64());
                self.metrics.get_proof_successful_responses.increment(1);
            }
            Err(_) => self.metrics.get_proof_failures.increment(1),
        }

        result
    }
}
