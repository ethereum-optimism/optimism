//! Historical proofs RPC server implementation.

use crate::{metrics::EthApiExtMetrics, state::OpStateProviderFactory};
use alloy_eips::BlockId;
use alloy_primitives::Address;
use alloy_rpc_types_eth::EIP1186AccountProofResponse;
use alloy_serde::JsonStorageKey;
use async_trait::async_trait;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee_core::RpcResult;
use jsonrpsee_types::error::ErrorObject;
use reth_optimism_trie::{OpProofsStorage, OpProofsStore};
use reth_provider::StateProofProvider;
use reth_rpc_api::eth::helpers::FullEthApi;
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
    state_provider_factory: OpStateProviderFactory<Eth, P>,
    metrics: EthApiExtMetrics,
}

impl<Eth, P> EthApiExt<Eth, P>
where
    Eth: FullEthApi + Send + Sync + 'static,
    ErrorObject<'static>: From<Eth::Error>,
    P: OpProofsStore + Clone + 'static,
{
    /// Creates a new instance of the `EthApiExt`.
    pub fn new(eth_api: Eth, preimage_store: OpProofsStorage<P>) -> Self {
        let metrics = EthApiExtMetrics::default();
        Self {
            state_provider_factory: OpStateProviderFactory::new(eth_api, preimage_store),
            metrics,
        }
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
        let start = Instant::now();
        self.metrics.get_proof_requests.increment(1);

        let storage_keys = keys.iter().map(|key| key.as_b256()).collect::<Vec<_>>();

        let result = async {
            let proof = self
                .state_provider_factory
                .state_provider(block_number)
                .await
                .map_err(Into::into)?
                .proof(Default::default(), address, &storage_keys)
                .map_err(Into::into)?;

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
