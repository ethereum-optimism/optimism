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
use reth_errors::RethError;
use reth_optimism_trie::{OpProofsStorage, OpProofsStore};
use reth_provider::StateProofProvider;
use reth_rpc_api::eth::helpers::FullEthApi;
use reth_rpc_eth_api::FromEthApiError;
use reth_rpc_eth_types::EthApiError;

/// The `eth_` proof methods served from the historical proofs storage.
///
/// UPSTREAM-MIRROR(set): reth@rev:fe5a0dd `reth_rpc_eth_api::EthApi`
///
/// Re-declares the proof methods that are answered from historical proofs rather than live state.
/// A proof method added to upstream's `EthApi` produces no diff here, so diff the two method sets
/// on each bump: every new one must either be declared here or dropped from the served surface in
/// `reth_optimism_node::node`, so that a proofs-history node never answers it from live state.
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
    /// Creates a new instance of the `EthApiExt`.
    pub fn new(eth_api: Eth, preimage_store: OpProofsStorage<P>) -> Self {
        Self { eth_api, preimage_store, metrics: EthApiExtMetrics::default() }
    }

    /// UPSTREAM-MIRROR(copy): reth@rev:fe5a0dd
    /// `reth_rpc_eth_api::helpers::state::EthState::get_proof`
    ///
    /// Uses the OP proofs-history state provider while preserving upstream permit and blocking-task
    /// behavior.
    async fn get_proof_inner(
        &self,
        address: Address,
        keys: Vec<JsonStorageKey>,
        block_id: BlockId,
    ) -> Result<EIP1186AccountProofResponse, Eth::Error> {
        let permit = self
            .eth_api
            .acquire_owned_tracing()
            .await
            .map_err(RethError::other)
            .map_err(EthApiError::Internal)?;
        let preimage_store = self.preimage_store.clone();

        self.eth_api
            .spawn_blocking_io_fut(move |eth_api| async move {
                // Hold the proof permit for the full lifetime of the blocking task, including
                // after the requesting future is cancelled.
                let _permit = permit;
                let state_provider_factory = OpStateProviderFactory::new(eth_api, preimage_store);
                let state = state_provider_factory
                    .state_provider(block_id)
                    .await
                    .map_err(Eth::Error::from_eth_err)?;
                let storage_keys = keys.iter().map(|key| key.as_b256()).collect::<Vec<_>>();
                let proof = state
                    .proof(Default::default(), address, &storage_keys)
                    .map_err(Eth::Error::from_eth_err)?;
                Ok(proof.into_eip1186_response(keys))
            })
            .await
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
        self.metrics
            .record_get_proof(self.get_proof_inner(address, keys, block_number.unwrap_or_default()))
            .await
            .map_err(Into::into)
    }
}
