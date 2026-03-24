//! [`InteropProvider`] trait implementation using a [`CommsClient`] data source.

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

/// A [`CommsClient`] backed [`InteropProvider`] implementation.
#[derive(Debug, Clone)]
pub struct OracleInteropProvider<C> {
    /// The oracle client.
    oracle: Arc<C>,
    /// The [`BootInfo`] for the current program execution.
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
    /// Creates a new [`OracleInteropProvider`] with the given oracle client and [`BootInfo`].
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

    /// Fetch the [`OpReceiptEnvelope`]s for the block with the given hash.
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
                let block_hash = match eip_2935_history_lookup(&header, number, self, self).await {
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

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::{collections::BTreeMap, format, string::String, sync::Arc, vec::Vec};
    use alloy_consensus::Header;
    use alloy_primitives::{B256, Sealable, keccak256};
    use alloy_rlp::Decodable;
    use async_trait::async_trait;
    use kona_genesis::RollupConfig;
    use kona_interop::SuperRoot;
    use kona_preimage::{
        HintWriterClient, PreimageKey, PreimageKeyType, PreimageOracleClient,
        errors::PreimageOracleResult,
    };
    use kona_proof::errors::OracleProviderError;
    use kona_registry::HashMap;

    use crate::{BootInfo, PreState};

    /// A single step in the EIP-2935 lookup chain. Each step contains the trie proof
    /// data needed for one iteration and the block header it resolves to.
    #[derive(serde::Deserialize)]
    struct ProofStep {
        account_proof: Vec<String>,
        storage_proof: Vec<String>,
        resolved_block_hash: String,
        resolved_block_header_rlp: String,
    }

    /// Fixture data for EIP-2935 `header_by_number` tests. Works for both single-iteration
    /// (1 step) and multi-iteration (2+ steps) lookups via the `steps` array.
    #[derive(serde::Deserialize)]
    struct FixtureData {
        chain_id: u64,
        safe_head_number: u64,
        safe_head_header_rlp: String,
        target_block_number: u64,
        target_block_hash: String,
        steps: Vec<ProofStep>,
    }

    /// In-memory preimage oracle for testing.
    #[derive(Debug, Clone)]
    struct MockCommsClient {
        preimages: BTreeMap<[u8; 32], Vec<u8>>,
    }

    #[async_trait]
    impl PreimageOracleClient for MockCommsClient {
        async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
            let raw_key: [u8; 32] = key.into();
            self.preimages.get(&raw_key).cloned().ok_or_else(|| {
                kona_preimage::errors::PreimageOracleError::Other(format!(
                    "preimage not found: 0x{}",
                    alloy_primitives::hex::encode(raw_key)
                ))
            })
        }

        async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
            let data = self.get(key).await?;
            if data.len() != buf.len() {
                return Err(kona_preimage::errors::PreimageOracleError::Other(
                    "length mismatch".into(),
                ));
            }
            buf.copy_from_slice(&data);
            Ok(())
        }
    }

    #[async_trait]
    impl HintWriterClient for MockCommsClient {
        async fn write(&self, _hint: &str) -> PreimageOracleResult<()> {
            Ok(())
        }
    }

    fn hex_to_bytes(hex: &str) -> Vec<u8> {
        let hex = hex.strip_prefix("0x").unwrap_or(hex);
        alloy_primitives::hex::decode(hex).expect("valid hex")
    }

    fn load_fixture_from(json: &str) -> (MockCommsClient, FixtureData) {
        let fixture: FixtureData = serde_json::from_str(json).expect("valid fixture JSON");

        let mut preimages = BTreeMap::new();

        for step in &fixture.steps {
            // Load account proof nodes (state trie).
            for node_hex in &step.account_proof {
                let node_bytes = hex_to_bytes(node_hex);
                let hash = keccak256(&node_bytes);
                let key: [u8; 32] = PreimageKey::new(*hash, PreimageKeyType::Keccak256).into();
                preimages.insert(key, node_bytes);
            }

            // Load storage proof nodes.
            for node_hex in &step.storage_proof {
                let node_bytes = hex_to_bytes(node_hex);
                let hash = keccak256(&node_bytes);
                let key: [u8; 32] = PreimageKey::new(*hash, PreimageKeyType::Keccak256).into();
                preimages.insert(key, node_bytes);
            }

            // Load resolved block header RLP, keyed by its block hash.
            let header_rlp = hex_to_bytes(&step.resolved_block_header_rlp);
            let block_hash: B256 = step.resolved_block_hash.parse().expect("valid hash");
            assert_eq!(
                keccak256(&header_rlp),
                block_hash,
                "resolved header RLP hash must match resolved block hash"
            );
            let key: [u8; 32] = PreimageKey::new(*block_hash, PreimageKeyType::Keccak256).into();
            preimages.insert(key, header_rlp);
        }

        (MockCommsClient { preimages }, fixture)
    }

    fn load_fixture() -> (MockCommsClient, FixtureData) {
        load_fixture_from(include_str!(concat!(
            env!("CARGO_MANIFEST_DIR"),
            "/testdata/eip2935_header_by_number.json"
        )))
    }

    fn load_multi_iter_fixture() -> (MockCommsClient, FixtureData) {
        load_fixture_from(include_str!(concat!(
            env!("CARGO_MANIFEST_DIR"),
            "/testdata/eip2935_multi_iteration.json"
        )))
    }

    fn build_provider(
        client: MockCommsClient,
        fixture: &FixtureData,
    ) -> OracleInteropProvider<MockCommsClient> {
        let safe_head_rlp = hex_to_bytes(&fixture.safe_head_header_rlp);
        let safe_head_header =
            Header::decode(&mut safe_head_rlp.as_ref()).expect("valid safe head header RLP");
        assert_eq!(safe_head_header.number, fixture.safe_head_number);

        let sealed_safe_head = safe_head_header.seal_slow();

        let mut local_safe_heads = HashMap::default();
        local_safe_heads.insert(fixture.chain_id, sealed_safe_head);

        let mut rollup_config = RollupConfig::default();
        rollup_config.hardforks.isthmus_time = Some(1746806401);

        let mut rollup_configs = HashMap::default();
        rollup_configs.insert(fixture.chain_id, rollup_config);

        let boot = BootInfo {
            l1_head: B256::ZERO,
            agreed_pre_state_commitment: B256::ZERO,
            agreed_pre_state: PreState::SuperRoot(SuperRoot::new(0, Vec::new())),
            claimed_post_state: B256::ZERO,
            claimed_l2_timestamp: 0,
            rollup_configs,
            l1_config: Default::default(),
        };

        OracleInteropProvider::new(Arc::new(client), boot, local_safe_heads)
    }

    /// Tests the EIP-2935 fast path: looking up a block at the boundary of the 8,191-block
    /// history window using real OP Mainnet trie proof data (1 step).
    ///
    /// Safe head: block 149,340,000
    /// Target: block 149,331,809 (exactly 8,191 blocks behind — at the EIP-2935 window boundary)
    ///
    /// Exercises the full path: `header_by_number` → Isthmus check → `eip_2935_history_lookup`
    /// (real state + storage trie traversal) → `header_by_hash` → return.
    #[tokio::test(flavor = "multi_thread")]
    async fn test_header_by_number_eip2935_fast_path() {
        let (client, fixture) = load_fixture();
        let provider = build_provider(client, &fixture);
        let expected_hash: B256 = fixture.target_block_hash.parse().unwrap();

        let header = provider
            .header_by_number(fixture.chain_id, fixture.target_block_number)
            .await
            .expect("header_by_number should succeed via EIP-2935 fast path");

        assert_eq!(header.hash_slow(), expected_hash);
        assert_eq!(header.number, fixture.target_block_number);
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_header_by_number_block_past_head() {
        let (client, fixture) = load_fixture();
        let provider = build_provider(client, &fixture);

        let result =
            provider.header_by_number(fixture.chain_id, fixture.safe_head_number + 1).await;

        assert!(matches!(result, Err(OracleProviderError::BlockNumberPastHead(_, _))));
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_header_by_number_same_block() {
        let (client, fixture) = load_fixture();
        let provider = build_provider(client, &fixture);

        let header = provider
            .header_by_number(fixture.chain_id, fixture.safe_head_number)
            .await
            .expect("looking up current head should succeed");

        assert_eq!(header.number, fixture.safe_head_number);
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_header_by_number_missing_chain_id() {
        let (client, fixture) = load_fixture();
        let provider = build_provider(client, &fixture);

        let result = provider.header_by_number(999, 1).await;
        assert!(result.is_err());
    }

    /// Tests multi-iteration EIP-2935 lookup: target block is beyond the 8,191-block window,
    /// requiring two EIP-2935 lookups through an intermediate block (2 steps).
    ///
    /// Safe head: block 149,388,609
    /// Intermediate: block 149,380,418 (8,191 blocks behind safe head — oldest in window)
    /// Target: block 149,380,413 (5 blocks before intermediate — 8,196 behind safe head)
    ///
    /// Iteration 1: `eip_2935_history_lookup(N, M)` → target outside window →
    ///   reads slot `N % 8191` from N's state → returns intermediate block hash.
    /// Iteration 2: `eip_2935_history_lookup(I, M)` → target inside window →
    ///   reads slot `M % 8191` from I's state → returns target block hash.
    #[tokio::test(flavor = "multi_thread")]
    async fn test_header_by_number_eip2935_multi_iteration() {
        let (client, fixture) = load_multi_iter_fixture();
        let provider = build_provider(client, &fixture);
        let expected_hash: B256 = fixture.target_block_hash.parse().unwrap();

        let header = provider
            .header_by_number(fixture.chain_id, fixture.target_block_number)
            .await
            .expect("header_by_number should succeed via multi-iteration EIP-2935 lookup");

        assert_eq!(header.hash_slow(), expected_hash);
        assert_eq!(header.number, fixture.target_block_number);
    }
}
