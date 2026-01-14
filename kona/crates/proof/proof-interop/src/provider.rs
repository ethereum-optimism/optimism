//! [InteropProvider] trait implementation using a [CommsClient] data source.

use crate::{BootInfo, HintType};
use alloc::{boxed::Box, string::ToString, sync::Arc, vec::Vec};
use alloy_consensus::{Header, Sealed};
use alloy_eips::eip2718::Decodable2718;
use alloy_primitives::{Address, B256};
use alloy_rlp::Decodable;
use async_trait::async_trait;
use kona_interop::InteropProvider;
use kona_mpt::{OrderedListWalker, TrieHinter, TrieNode, TrieProvider};
use kona_preimage::{CommsClient, PreimageKey, PreimageKeyType, errors::PreimageOracleError};
use kona_proof::{eip_2935_history_lookup, errors::OracleProviderError};
use kona_registry::HashMap;
use op_alloy_consensus::OpReceiptEnvelope;
use spin::RwLock;

/// A [CommsClient] backed [InteropProvider] implementation.
#[derive(Debug, Clone)]
pub struct OracleInteropProvider<C> {
    /// The oracle client.
    oracle: Arc<C>,
    /// The [BootInfo] for the current program execution.
    boot: BootInfo,
    /// The local safe head block header cache.
    local_safe_heads: HashMap<u64, Sealed<Header>>,
    /// The chain ID for the current call context. Used to declare the chain ID for the trie hints.
    chain_id: Arc<RwLock<Option<u64>>>,
}

impl<C> OracleInteropProvider<C>
where
    C: CommsClient + Send + Sync,
{
    /// Creates a new [OracleInteropProvider] with the given oracle client and [BootInfo].
    pub fn new(
        oracle: Arc<C>,
        boot: BootInfo,
        local_safe_headers: HashMap<u64, Sealed<Header>>,
    ) -> Self {
        Self {
            oracle,
            boot,
            local_safe_heads: local_safe_headers,
            chain_id: Arc::new(RwLock::new(None)),
        }
    }

    /// Returns a reference to the local safe heads map.
    pub const fn local_safe_heads(&self) -> &HashMap<u64, Sealed<Header>> {
        &self.local_safe_heads
    }

    /// Replaces a local safe head with the given header.
    pub fn replace_local_safe_head(&mut self, chain_id: u64, header: Sealed<Header>) {
        self.local_safe_heads.insert(chain_id, header);
    }

    /// Fetch the [Header] for the block with the given hash.
    pub async fn header_by_hash(
        &self,
        chain_id: u64,
        block_hash: B256,
    ) -> Result<Header, <Self as InteropProvider>::Error> {
        HintType::L2BlockHeader
            .with_data(&[block_hash.as_slice(), chain_id.to_be_bytes().as_ref()])
            .send(self.oracle.as_ref())
            .await?;

        let header_rlp = self
            .oracle
            .get(PreimageKey::new(*block_hash, PreimageKeyType::Keccak256))
            .await
            .map_err(OracleProviderError::Preimage)?;

        Header::decode(&mut header_rlp.as_ref()).map_err(OracleProviderError::Rlp)
    }

    /// Fetch the [OpReceiptEnvelope]s for the block with the given hash.
    async fn derive_receipts(
        &self,
        chain_id: u64,
        block_hash: B256,
        header: &Header,
    ) -> Result<Vec<OpReceiptEnvelope>, <Self as InteropProvider>::Error> {
        // Send a hint for the block's receipts, and walk through the receipts trie in the header to
        // verify them.
        HintType::L2Receipts
            .with_data(&[block_hash.as_ref(), chain_id.to_be_bytes().as_slice()])
            .send(self.oracle.as_ref())
            .await?;
        let trie_walker = OrderedListWalker::try_new_hydrated(header.receipts_root, self)
            .map_err(OracleProviderError::TrieWalker)?;

        // Decode the receipts within the receipts trie.
        let receipts = trie_walker
            .into_iter()
            .map(|(_, rlp)| {
                let envelope = OpReceiptEnvelope::decode_2718(&mut rlp.as_ref())?;
                Ok(envelope)
            })
            .collect::<Result<Vec<_>, _>>()
            .map_err(OracleProviderError::Rlp)?;

        Ok(receipts)
    }
}

#[async_trait]
impl<C> InteropProvider for OracleInteropProvider<C>
where
    C: CommsClient + Send + Sync,
{
    type Error = OracleProviderError;

    /// Fetch a [Header] by its number.
    async fn header_by_number(&self, chain_id: u64, number: u64) -> Result<Header, Self::Error> {
        let Some(mut header) =
            self.local_safe_heads.get(&chain_id).cloned().map(|h| h.into_inner())
        else {
            return Err(PreimageOracleError::Other("Missing local safe header".to_string()).into());
        };

        // Check if the block number is in range. If not, we can fail early.
        if number > header.number {
            return Err(OracleProviderError::BlockNumberPastHead(number, header.number));
        }

        // Set the chain ID for the trie hints, and explicitly drop the lock.
        let mut chain_id_lock = self.chain_id.write();
        *chain_id_lock = Some(chain_id);
        drop(chain_id_lock);

        // Walk back the block headers to the desired block number.
        let rollup_config = self.boot.rollup_config(chain_id).ok_or_else(|| {
            PreimageOracleError::Other("Missing rollup config for chain ID".to_string())
        })?;
        let mut linear_fallback = false;

        while header.number > number {
            if rollup_config.is_isthmus_active(header.timestamp) && !linear_fallback {
                // If Isthmus is active, the EIP-2935 contract is used to perform leaping lookbacks
                // through consulting the ring buffer within the contract. If this
                // lookup fails for any reason, we fall back to linear walk back.
                let block_hash = match eip_2935_history_lookup(&header, 0, self, self).await {
                    Ok(hash) => hash,
                    Err(_) => {
                        // If the EIP-2935 lookup fails for any reason, attempt fallback to linear
                        // walk back.
                        linear_fallback = true;
                        continue;
                    }
                };

                header = self.header_by_hash(chain_id, block_hash).await?;
            } else {
                // Walk back the block headers one-by-one until the desired block number is reached.
                header = self.header_by_hash(chain_id, header.parent_hash).await?;
            }
        }

        Ok(header)
    }

    /// Fetch all receipts for a given block by number.
    async fn receipts_by_number(
        &self,
        chain_id: u64,
        number: u64,
    ) -> Result<Vec<OpReceiptEnvelope>, Self::Error> {
        let header = self.header_by_number(chain_id, number).await?;
        self.derive_receipts(chain_id, header.hash_slow(), &header).await
    }

    /// Fetch all receipts for a given block by hash.
    async fn receipts_by_hash(
        &self,
        chain_id: u64,
        block_hash: B256,
    ) -> Result<Vec<OpReceiptEnvelope>, Self::Error> {
        let header = self.header_by_hash(chain_id, block_hash).await?;
        self.derive_receipts(chain_id, block_hash, &header).await
    }
}

impl<C> TrieProvider for OracleInteropProvider<C>
where
    C: CommsClient + Send + Sync + Clone,
{
    type Error = OracleProviderError;

    fn trie_node_by_hash(&self, key: B256) -> Result<TrieNode, Self::Error> {
        kona_proof::block_on(async move {
            let trie_node_rlp = self
                .oracle
                .get(PreimageKey::new(*key, PreimageKeyType::Keccak256))
                .await
                .map_err(OracleProviderError::Preimage)?;
            TrieNode::decode(&mut trie_node_rlp.as_ref()).map_err(OracleProviderError::Rlp)
        })
    }
}

impl<C: CommsClient> TrieHinter for OracleInteropProvider<C> {
    type Error = OracleProviderError;

    fn hint_trie_node(&self, hash: B256) -> Result<(), Self::Error> {
        kona_proof::block_on(async move {
            HintType::L2StateNode
                .with_data(&[hash.as_slice()])
                .with_data(
                    self.chain_id.read().map_or_else(Vec::new, |id| id.to_be_bytes().to_vec()),
                )
                .send(self.oracle.as_ref())
                .await
        })
    }

    fn hint_account_proof(&self, address: Address, block_number: u64) -> Result<(), Self::Error> {
        kona_proof::block_on(async move {
            HintType::L2AccountProof
                .with_data(&[block_number.to_be_bytes().as_ref(), address.as_slice()])
                .with_data(
                    self.chain_id.read().map_or_else(Vec::new, |id| id.to_be_bytes().to_vec()),
                )
                .send(self.oracle.as_ref())
                .await
        })
    }

    fn hint_storage_proof(
        &self,
        address: alloy_primitives::Address,
        slot: alloy_primitives::U256,
        block_number: u64,
    ) -> Result<(), Self::Error> {
        kona_proof::block_on(async move {
            HintType::L2AccountStorageProof
                .with_data(&[
                    block_number.to_be_bytes().as_ref(),
                    address.as_slice(),
                    slot.to_be_bytes::<32>().as_ref(),
                ])
                .with_data(
                    self.chain_id.read().map_or_else(Vec::new, |id| id.to_be_bytes().to_vec()),
                )
                .send(self.oracle.as_ref())
                .await
        })
    }

    fn hint_execution_witness(
        &self,
        parent_hash: B256,
        op_payload_attributes: &op_alloy_rpc_types_engine::OpPayloadAttributes,
    ) -> Result<(), Self::Error> {
        kona_proof::block_on(async move {
            let encoded_attributes =
                serde_json::to_vec(op_payload_attributes).map_err(OracleProviderError::Serde)?;

            HintType::L2PayloadWitness
                .with_data(&[parent_hash.as_slice(), &encoded_attributes])
                .with_data(
                    self.chain_id.read().map_or_else(Vec::new, |id| id.to_be_bytes().to_vec()),
                )
                .send(self.oracle.as_ref())
                .await
        })
    }
}
