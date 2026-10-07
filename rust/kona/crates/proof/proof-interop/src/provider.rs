//! [`InteropProvider`] trait implementation using a [`CommsClient`] data source.

use crate::{BootInfo, HintType};
use alloc::{boxed::Box, collections::BTreeMap, string::ToString, sync::Arc, vec::Vec};
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
use spin::Mutex;

/// A [`CommsClient`] backed [`InteropProvider`] implementation.
#[derive(Debug)]
pub struct OracleInteropProvider<C> {
    /// The oracle client.
    oracle: Arc<C>,
    /// The [`BootInfo`] for the current program execution.
    boot: BootInfo,
    /// The local safe head block header cache.
    local_safe_heads: HashMap<u64, Sealed<Header>>,
    /// Canonical headers and hashes, keyed by chain ID and block number.
    headers_by_number: Mutex<BTreeMap<(u64, u64), (B256, Header)>>,
    /// Verified local safe-head receipts, keyed by chain ID and block hash.
    receipts_by_hash: Mutex<BTreeMap<(u64, B256), Vec<OpReceiptEnvelope>>>,
}

impl<C> Clone for OracleInteropProvider<C> {
    fn clone(&self) -> Self {
        Self {
            oracle: self.oracle.clone(),
            boot: self.boot.clone(),
            local_safe_heads: self.local_safe_heads.clone(),
            headers_by_number: Mutex::new(self.headers_by_number.lock().clone()),
            receipts_by_hash: Mutex::new(self.receipts_by_hash.lock().clone()),
        }
    }
}

/// A chain-scoped [`TrieHinter`] that annotates all hints with a fixed chain ID.
///
/// Created via [`OracleInteropProvider::scoped_hinter`]. This avoids storing mutable chain ID
/// state in the provider, making it explicit which chain ID is used for each trie hint.
#[derive(Debug, Clone)]
pub struct ChainScopedHinter<'a, C> {
    /// The oracle client, borrowed from the parent provider.
    oracle: &'a Arc<C>,
    /// The chain ID to annotate hints with.
    chain_id: u64,
}

impl<C> OracleInteropProvider<C>
where
    C: CommsClient + Send + Sync,
{
    /// Creates a new [`OracleInteropProvider`] with the given oracle client and [`BootInfo`].
    pub const fn new(
        oracle: Arc<C>,
        boot: BootInfo,
        local_safe_headers: HashMap<u64, Sealed<Header>>,
    ) -> Self {
        Self {
            oracle,
            boot,
            local_safe_heads: local_safe_headers,
            headers_by_number: Mutex::new(BTreeMap::new()),
            receipts_by_hash: Mutex::new(BTreeMap::new()),
        }
    }

    /// Sends an [`HintType::L2Transactions`] hint for the given block, instructing the host to
    /// pre-fetch the transaction trie nodes into the preimage oracle's key-value store.
    pub async fn hint_transactions(
        &self,
        chain_id: u64,
        block_hash: B256,
    ) -> Result<(), <Self as InteropProvider>::Error> {
        HintType::L2Transactions
            .with_data(&[block_hash.as_slice(), chain_id.to_be_bytes().as_ref()])
            .send(self.oracle.as_ref())
            .await
    }

    /// Returns a reference to the local safe heads map.
    pub const fn local_safe_heads(&self) -> &HashMap<u64, Sealed<Header>> {
        &self.local_safe_heads
    }

    /// Creates a [`ChainScopedHinter`] bound to the given chain ID.
    ///
    /// The returned hinter implements [`TrieHinter`] and annotates all hints with the specified
    /// chain ID. This makes the chain context explicit at each call site rather than relying on
    /// mutable state within the provider.
    pub const fn scoped_hinter(&self, chain_id: u64) -> ChainScopedHinter<'_, C> {
        ChainScopedHinter { oracle: &self.oracle, chain_id }
    }

    /// Replaces a local safe head with the given header.
    pub fn replace_local_safe_head(&mut self, chain_id: u64, header: Sealed<Header>) {
        self.headers_by_number.get_mut().retain(|(cached_chain, _), _| *cached_chain != chain_id);
        self.receipts_by_hash.get_mut().retain(|(cached_chain, _), _| *cached_chain != chain_id);
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
        if let Some(receipts) = self.receipts_by_hash.lock().get(&(chain_id, block_hash)) {
            return Ok(receipts.clone());
        }

        // Verify the receipts against the header's trie root before caching them.
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

        if self.local_safe_heads.get(&chain_id).is_some_and(|head| head.hash() == block_hash) {
            self.receipts_by_hash.lock().insert((chain_id, block_hash), receipts.clone());
        }
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
        let Some(sealed) = self.local_safe_heads.get(&chain_id).cloned() else {
            return Err(PreimageOracleError::Other("Missing local safe header".to_string()).into());
        };
        if number > sealed.number {
            return Err(OracleProviderError::BlockNumberPastHead(number, sealed.number));
        }
        let rollup_config = self.boot.rollup_config(chain_id).ok_or_else(|| {
            PreimageOracleError::Other("Missing rollup config for chain ID".to_string())
        })?;
        let cached = self
            .headers_by_number
            .lock()
            .range((chain_id, number)..=(chain_id, u64::MAX))
            .next()
            .map(|(_, entry)| entry.clone());
        let (mut current_hash, mut header) =
            cached.unwrap_or_else(|| (sealed.hash(), sealed.into_inner()));
        if header.number == number {
            return Ok(header);
        }
        self.headers_by_number
            .lock()
            .insert((chain_id, header.number), (current_hash, header.clone()));

        let hinter = self.scoped_hinter(chain_id);
        let mut linear_fallback = false;

        while header.number > number {
            if rollup_config.is_isthmus_active(header.timestamp) && !linear_fallback {
                // If Isthmus is active, the EIP-2935 contract is used to perform leaping lookbacks
                // through consulting the ring buffer within the contract. If this
                // lookup fails for any reason, we fall back to linear walk back.
                let block_hash =
                    match eip_2935_history_lookup(&header, number, current_hash, self, &hinter)
                        .await
                    {
                        Ok(hash) => hash,
                        Err(_) => {
                            // If the EIP-2935 lookup fails for any reason, attempt fallback to
                            // linear walk back.
                            linear_fallback = true;
                            continue;
                        }
                    };

                current_hash = block_hash;
                header = self.header_by_hash(chain_id, block_hash).await?;
            } else {
                // Walk back the block headers one-by-one until the desired block number is reached.
                current_hash = header.parent_hash;
                header = self.header_by_hash(chain_id, header.parent_hash).await?;
            }
            self.headers_by_number
                .lock()
                .insert((chain_id, header.number), (current_hash, header.clone()));
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
        if let Some(receipts) = self.receipts_by_hash.lock().get(&(chain_id, block_hash)) {
            return Ok(receipts.clone());
        }
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

impl<C: CommsClient> TrieHinter for ChainScopedHinter<'_, C> {
    type Error = OracleProviderError;

    fn hint_trie_node(&self, hash: B256) -> Result<(), Self::Error> {
        kona_proof::block_on(async move {
            HintType::L2StateNode
                .with_data(&[hash.as_slice()])
                .with_data(self.chain_id.to_be_bytes())
                .send(self.oracle.as_ref())
                .await
        })
    }

    fn hint_account_proof(&self, address: Address, block_hash: B256) -> Result<(), Self::Error> {
        kona_proof::block_on(async move {
            HintType::L2AccountProof
                .with_data(&[block_hash.as_slice(), address.as_slice()])
                .with_data(self.chain_id.to_be_bytes())
                .send(self.oracle.as_ref())
                .await
        })
    }

    fn hint_storage_proof(
        &self,
        address: alloy_primitives::Address,
        slot: alloy_primitives::U256,
        block_hash: B256,
    ) -> Result<(), Self::Error> {
        kona_proof::block_on(async move {
            HintType::L2AccountStorageProof
                .with_data(&[
                    block_hash.as_slice(),
                    address.as_slice(),
                    slot.to_be_bytes::<32>().as_ref(),
                ])
                .with_data(self.chain_id.to_be_bytes())
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
                .with_data(self.chain_id.to_be_bytes())
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
    use kona_interop::{DependencySet, SuperRoot};
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
        reads: Arc<spin::Mutex<Vec<PreimageKey>>>,
    }

    #[async_trait]
    impl PreimageOracleClient for MockCommsClient {
        async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
            self.reads.lock().push(key);
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

        (MockCommsClient { preimages, reads: Default::default() }, fixture)
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
            dependency_set: DependencySet {
                dependencies: Default::default(),
                override_message_expiry_window: None,
            },
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
    #[tokio::test(flavor = "multi_thread")]
    async fn test_header_by_number_caches_repeated_lookups() {
        let (client, fixture) = load_fixture();
        let reads = client.reads.clone();
        let provider = build_provider(client, &fixture);
        let first =
            provider.header_by_number(fixture.chain_id, fixture.target_block_number).await.unwrap();
        let reads_after_first = reads.lock().len();
        assert!(reads_after_first > 0);
        let second =
            provider.header_by_number(fixture.chain_id, fixture.target_block_number).await.unwrap();
        assert_eq!(first, second);
        assert_eq!(
            reads.lock().len(),
            reads_after_first,
            "repeated lookup must reuse verified headers"
        );
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_receipts_cached_across_number_and_hash_lookups() {
        use alloy_consensus::{Receipt, ReceiptWithBloom};
        use alloy_eips::eip2718::Encodable2718;
        use kona_mpt::Nibbles;

        let (mut client, fixture) = load_fixture();
        let receipt = OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
            receipt: Receipt { status: true.into(), ..Default::default() },
            ..Default::default()
        });
        let node = TrieNode::Leaf {
            prefix: Nibbles::unpack([0x80]),
            value: receipt.encoded_2718().into(),
        };
        let node_rlp = alloy_rlp::encode(&node);
        let receipts_root = keccak256(&node_rlp);
        let trie_key = PreimageKey::new(*receipts_root, PreimageKeyType::Keccak256);
        client.preimages.insert(trie_key.into(), node_rlp);
        let header = Header { number: 1, receipts_root, ..Default::default() }.seal_slow();
        let header_key = PreimageKey::new(*header.hash(), PreimageKeyType::Keccak256);
        client.preimages.insert(header_key.into(), alloy_rlp::encode(header.inner()));
        let reads = client.reads.clone();
        let mut provider = build_provider(client, &fixture);
        provider.replace_local_safe_head(fixture.chain_id, header.clone());

        assert_eq!(
            provider.receipts_by_hash(fixture.chain_id, header.hash()).await.unwrap(),
            vec![receipt.clone()]
        );
        assert_eq!(
            provider.receipts_by_number(fixture.chain_id, header.number).await.unwrap(),
            vec![receipt]
        );
        assert_eq!(
            reads.lock().iter().filter(|key| **key == trie_key).count(),
            1,
            "receipts trie must be decoded once per block hash"
        );
    }

    fn insert_test_header(client: &mut MockCommsClient, header: Header) -> Sealed<Header> {
        let header = header.seal_slow();
        let key = PreimageKey::new(*header.hash(), PreimageKeyType::Keccak256);
        client.preimages.insert(key.into(), alloy_rlp::encode(header.inner()));
        header
    }

    fn insert_test_receipts(client: &mut MockCommsClient, receipts: &[OpReceiptEnvelope]) -> B256 {
        use alloy_eips::eip2718::Encodable2718;
        let mut trie =
            kona_mpt::ordered_trie_with_encoder(receipts, |receipt, out| receipt.encode_2718(out));
        let root = trie.root();
        for node in trie.take_proof_nodes().into_inner().into_values() {
            let key = PreimageKey::new(*keccak256(&node), PreimageKeyType::Keccak256);
            client.preimages.insert(key.into(), node.to_vec());
        }
        root
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_header_cache_reuses_walked_ancestors() {
        let (mut client, fixture) = load_fixture();
        let mut head = Header::default().seal_slow();
        for number in 1..=8 {
            head = insert_test_header(
                &mut client,
                Header { number, parent_hash: head.hash(), ..Default::default() },
            );
        }
        let reads = client.reads.clone();
        let mut provider = build_provider(client, &fixture);
        provider.replace_local_safe_head(fixture.chain_id, head);
        assert_eq!(provider.header_by_number(fixture.chain_id, 4).await.unwrap().number, 4);
        assert_eq!(reads.lock().len(), 4);
        for number in 4..=8 {
            assert_eq!(
                provider.header_by_number(fixture.chain_id, number).await.unwrap().number,
                number
            );
        }
        assert_eq!(reads.lock().len(), 4, "cached ancestors must not be walked again");
        assert_eq!(provider.header_by_number(fixture.chain_id, 2).await.unwrap().number, 2);
        assert_eq!(reads.lock().len(), 6, "lookback must start at the closest cached header");
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_header_cache_isolated_by_chain_and_clone_after_replacement() {
        let (mut client, fixture) = load_fixture();
        let original_parent = insert_test_header(
            &mut client,
            Header { number: 1, timestamp: 1, ..Default::default() },
        );
        let replacement_parent = insert_test_header(
            &mut client,
            Header { number: 1, timestamp: 2, ..Default::default() },
        );
        let original =
            Header { number: 2, parent_hash: original_parent.hash(), ..Default::default() }
                .seal_slow();
        let replacement =
            Header { number: 2, parent_hash: replacement_parent.hash(), ..Default::default() }
                .seal_slow();
        let reads = client.reads.clone();
        let mut provider = build_provider(client, &fixture);
        let other_chain = fixture.chain_id + 1;
        let config = provider.boot.rollup_configs[&fixture.chain_id].clone();
        provider.boot.rollup_configs.insert(other_chain, config);
        provider.replace_local_safe_head(fixture.chain_id, original.clone());
        provider.replace_local_safe_head(other_chain, replacement.clone());
        assert_eq!(
            provider.header_by_number(fixture.chain_id, 1).await.unwrap(),
            *original_parent.inner()
        );
        assert_eq!(
            provider.header_by_number(other_chain, 1).await.unwrap(),
            *replacement_parent.inner()
        );
        let clone = provider.clone();
        provider.replace_local_safe_head(fixture.chain_id, replacement.clone());
        assert_eq!(
            provider.header_by_number(fixture.chain_id, 2).await.unwrap(),
            *replacement.inner()
        );
        assert_eq!(
            provider.header_by_number(fixture.chain_id, 1).await.unwrap(),
            *replacement_parent.inner()
        );
        let after_replacement = reads.lock().len();
        assert_eq!(
            clone.header_by_number(fixture.chain_id, 1).await.unwrap(),
            *original_parent.inner()
        );
        assert_eq!(clone.header_by_number(fixture.chain_id, 2).await.unwrap(), *original.inner());
        assert_eq!(
            provider.header_by_number(other_chain, 1).await.unwrap(),
            *replacement_parent.inner()
        );
        assert_eq!(
            reads.lock().len(),
            after_replacement,
            "unaffected chain and clone retain their canonical caches"
        );
        assert!(matches!(
            provider.header_by_number(fixture.chain_id, 3).await,
            Err(OracleProviderError::BlockNumberPastHead(3, 2))
        ));
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_message_graph_revalidates_cached_source_after_replacement() {
        use alloy_consensus::{Receipt, ReceiptWithBloom};
        use kona_genesis::ChainDependency;
        use kona_interop::{
            ExecutingMessageBuilder, MessageGraph, MessageGraphError, SuperchainBuilder,
        };

        let mut superchain = SuperchainBuilder::new();
        for chain in [1, 2] {
            superchain
                .chain(chain)
                .with_timestamp(2)
                .with_block_time(2)
                .with_lagoon_activation_time(0);
        }
        superchain.chain(1).add_initiating_message(b"payload".to_vec().into());
        superchain.chain(2).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_origin_chain_id(1)
                .with_origin_timestamp(2)
                .with_message_hash(keccak256(b"payload")),
        );
        let (headers, configs, mock) = superchain.build();
        let (mut client, fixture) = load_fixture();
        let mut safe_heads = HashMap::default();
        for (chain_id, header) in headers {
            let receipts_root =
                insert_test_receipts(&mut client, &mock.receipts[&chain_id][&header.number]);
            let header =
                insert_test_header(&mut client, Header { receipts_root, ..header.into_inner() });
            safe_heads.insert(chain_id, header);
        }
        let source = safe_heads[&1].clone();
        let replacement_receipts = vec![OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
            receipt: Receipt::default(),
            ..Default::default()
        })];
        let receipts_root = insert_test_receipts(&mut client, &replacement_receipts);
        let replacement =
            insert_test_header(&mut client, Header { receipts_root, ..source.inner().clone() });
        let replacement_key = PreimageKey::new(*replacement.hash(), PreimageKeyType::Keccak256);
        client.preimages.remove(&<[u8; 32]>::from(replacement_key));
        let reads = client.reads.clone();
        let mut provider = build_provider(client, &fixture);
        provider.boot.rollup_configs = configs.clone();
        let dependency_set = DependencySet {
            dependencies: [(1, ChainDependency {}), (2, ChainDependency {})].into_iter().collect(),
            override_message_expiry_window: None,
        };
        provider.local_safe_heads = safe_heads;
        for _ in 0..2 {
            let graph = MessageGraph::derive(
                provider.local_safe_heads(),
                &provider,
                &configs,
                &dependency_set,
                604800,
            )
            .await
            .unwrap();
            graph.resolve().await.unwrap();
        }
        let source_key = PreimageKey::new(*source.receipts_root, PreimageKeyType::Keccak256);
        assert_eq!(
            reads.lock().iter().filter(|key| **key == source_key).count(),
            1,
            "consolidation iterations must reuse decoded source receipts"
        );
        provider.replace_local_safe_head(1, replacement);
        let mut heads_to_check = provider.local_safe_heads().clone();
        heads_to_check.remove(&1);
        let graph =
            MessageGraph::derive(&heads_to_check, &provider, &configs, &dependency_set, 604800)
                .await
                .unwrap();
        let MessageGraphError::InvalidMessages(invalid) = graph.resolve().await.unwrap_err() else {
            panic!("cached source validity must not survive replacement");
        };
        assert_eq!(invalid.len(), 1);
        assert!(matches!(
            invalid[&2],
            MessageGraphError::RemoteMessageNotFound { chain_id: 1, .. }
        ));
        let before_old_hash_lookup = reads.lock().len();
        assert_eq!(
            provider.receipts_by_hash(1, source.hash()).await.unwrap(),
            mock.receipts[&1][&source.number]
        );
        assert!(
            reads.lock().len() > before_old_hash_lookup,
            "replaced head receipts must be decoded again rather than retained"
        );
        assert!(
            reads.lock().iter().all(|key| *key != replacement_key),
            "replacement header must be served from memory"
        );
        assert_ne!(
            provider.header_by_number(1, source.number).await.unwrap().hash_slow(),
            source.hash(),
            "old hash lookup must not restore the replaced canonical header"
        );
    }
    #[tokio::test(flavor = "multi_thread")]
    async fn test_receipt_cache_does_not_retain_historical_blocks() {
        use alloy_consensus::{Receipt, ReceiptWithBloom};

        let (mut client, fixture) = load_fixture();
        let receipt = OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
            receipt: Receipt::default(),
            ..Default::default()
        });
        let receipts_root = insert_test_receipts(&mut client, core::slice::from_ref(&receipt));
        let mut headers = Vec::new();
        for number in 1..=8 {
            headers.push(insert_test_header(
                &mut client,
                Header { number, receipts_root, ..Default::default() },
            ));
        }
        let mut provider = build_provider(client, &fixture);
        for header in &headers {
            assert_eq!(
                provider.receipts_by_hash(fixture.chain_id, header.hash()).await.unwrap(),
                vec![receipt.clone()]
            );
        }
        assert_eq!(
            provider.receipts_by_hash.lock().len(),
            0,
            "historical sources must not accumulate decoded receipts"
        );
        provider.replace_local_safe_head(fixture.chain_id, headers[0].clone());
        provider.receipts_by_number(fixture.chain_id, headers[0].number).await.unwrap();
        assert_eq!(provider.receipts_by_hash.lock().len(), 1);
        provider.replace_local_safe_head(fixture.chain_id, headers[1].clone());
        assert_eq!(provider.receipts_by_hash.lock().len(), 0, "replaced heads must be evicted");
        provider.receipts_by_number(fixture.chain_id, headers[1].number).await.unwrap();
        assert_eq!(provider.receipts_by_hash.lock().len(), 1);
    }
}
