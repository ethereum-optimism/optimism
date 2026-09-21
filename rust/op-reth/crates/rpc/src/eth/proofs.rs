//! Historical proofs RPC server implementation.

use crate::{metrics::EthApiExtMetrics, state::OpStateProviderFactory};
use alloy_eips::BlockId;
use alloy_primitives::{Address, B256, keccak256};
use alloy_rpc_types_eth::EIP1186AccountProofResponse;
use alloy_serde::JsonStorageKey;
use async_trait::async_trait;
use jsonrpsee::proc_macros::rpc;
use jsonrpsee_core::RpcResult;
use jsonrpsee_types::error::ErrorObject;
use reth_optimism_trie::{OpProofsStorage, OpProofsStore};
use reth_provider::StateProofProvider;
use reth_rpc_api::eth::helpers::FullEthApi;
use reth_rpc_server_types::result::internal_rpc_err;
use reth_trie_common::MultiProofTargets;
use std::time::Instant;

/// The `eth_` proof methods served from the historical proofs storage.
///
/// UPSTREAM-MIRROR(set): reth@rev:0fbe428 `reth_rpc_eth_api::EthApi`
///
/// Re-declares the proof methods that must be answered from historical proofs rather than live
/// state. A proof method added to upstream's `EthApi` produces no diff here and keeps falling
/// through to the stock handler, which cannot serve a block outside the node's state window —
/// so diff the two method sets on each bump and re-declare anything new.
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

    /// Returns the account and storage values of the specified targets including Merkle proofs.
    #[method(name = "getMultiProof")]
    async fn get_multi_proof(
        &self,
        targets: Vec<(Address, Vec<B256>)>,
        block_number: Option<BlockId>,
    ) -> RpcResult<Vec<EIP1186AccountProofResponse>>;
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

    async fn get_multi_proof(
        &self,
        targets: Vec<(Address, Vec<B256>)>,
        block_number: Option<BlockId>,
    ) -> RpcResult<Vec<EIP1186AccountProofResponse>> {
        let start = Instant::now();
        self.metrics.get_multi_proof_requests.increment(1);

        let result = async {
            let mut proof_targets = MultiProofTargets::with_capacity(targets.len());
            for (address, slots) in &targets {
                proof_targets
                    .entry(keccak256(address))
                    .or_default()
                    .extend(slots.iter().map(keccak256));
            }

            let multiproof = self
                .state_provider_factory
                .state_provider(block_number)
                .await
                .map_err(Into::into)?
                .multiproof(Default::default(), proof_targets)
                .map_err(Into::into)?;

            targets
                .into_iter()
                .map(|(address, slots)| {
                    let proof = multiproof
                        .account_proof(address, &slots)
                        .map_err(|err| internal_rpc_err(err.to_string()))?;
                    let storage_keys = slots.into_iter().map(JsonStorageKey::from).collect();
                    Ok(proof.into_eip1186_response(storage_keys))
                })
                .collect::<RpcResult<Vec<_>>>()
        }
        .await;

        match &result {
            Ok(_) => {
                self.metrics.get_multi_proof_latency.record(start.elapsed().as_secs_f64());
                self.metrics.get_multi_proof_successful_responses.increment(1);
            }
            Err(_) => self.metrics.get_multi_proof_failures.increment(1),
        }

        result
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_primitives::{address, b256};
    use std::sync::{Arc, Mutex};

    const ADDRESS: Address = address!("0x000000000000000000000000000000000000dead");
    const SLOT: B256 = b256!("0x0000000000000000000000000000000000000000000000000000000000000001");

    /// The `(account, slots)` pairs the RPC layer decoded, shared with the test.
    type SeenTargets = Arc<Mutex<Vec<(Address, Vec<B256>)>>>;

    /// Stand-in for [`EthApiExt`] that records what the RPC layer decoded instead of walking a
    /// trie, so these tests pin which methods the override claims and with what parameters.
    #[derive(Debug, Default)]
    struct RecordingEthApi {
        seen: SeenTargets,
    }

    #[async_trait]
    impl EthApiOverrideServer for RecordingEthApi {
        async fn get_proof(
            &self,
            address: Address,
            keys: Vec<JsonStorageKey>,
            _block_number: Option<BlockId>,
        ) -> RpcResult<EIP1186AccountProofResponse> {
            self.seen
                .lock()
                .unwrap()
                .push((address, keys.iter().map(JsonStorageKey::as_b256).collect()));
            Ok(EIP1186AccountProofResponse::default())
        }

        async fn get_multi_proof(
            &self,
            targets: Vec<(Address, Vec<B256>)>,
            _block_number: Option<BlockId>,
        ) -> RpcResult<Vec<EIP1186AccountProofResponse>> {
            let len = targets.len();
            self.seen.lock().unwrap().extend(targets);
            Ok(vec![EIP1186AccountProofResponse::default(); len])
        }
    }

    /// `eth_getMultiProof` must be served by the override too: falling through to the stock
    /// handler would answer from live state, which a pruned proofs node does not have. The
    /// parameters have to match upstream's, or a client that reaches this node gets an
    /// "Invalid params" it would not get from a node without the ExEx.
    #[tokio::test]
    async fn get_multi_proof_is_served_by_the_override() {
        let api = RecordingEthApi::default();
        let seen = api.seen.clone();
        let module = api.into_rpc();

        let responses: Vec<EIP1186AccountProofResponse> = module
            .call("eth_getMultiProof", (vec![(ADDRESS, vec![SLOT])], Option::<BlockId>::None))
            .await
            .expect("eth_getMultiProof should be handled by the proofs override");

        assert_eq!(responses.len(), 1);
        assert_eq!(*seen.lock().unwrap(), vec![(ADDRESS, vec![SLOT])]);
    }

    /// The override enumerates upstream's proof methods by hand, so pin the set: a method
    /// dropped from this trait silently falls back to the stock, live-state handler.
    #[tokio::test]
    async fn override_claims_every_proof_method() {
        let module = RecordingEthApi::default().into_rpc();
        let mut names = module.method_names().collect::<Vec<_>>();
        names.sort_unstable();
        assert_eq!(names, ["eth_getMultiProof", "eth_getProof"]);
    }
}
