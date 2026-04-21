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

#[derive(Debug, Clone)]
struct KnownBlock {
    header: Sealed<Header>,
    receipts: Vec<OpReceiptEnvelope>,
}

/// A [`CommsClient`] backed [`InteropProvider`] implementation.
#[derive(Debug, Clone)]
pub struct OracleInteropProvider<C> {
    /// The oracle client.
    oracle: Arc<C>,
    /// The [`BootInfo`] for the current program execution.
    boot: BootInfo,
    /// The local safe head block header cache.
    local_safe_heads: HashMap<u64, Sealed<Header>>,
    /// Blocks built locally during consolidation, keyed by chain ID and block hash.
    known_blocks: HashMap<u64, HashMap<B256, KnownBlock>>,
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
            known_blocks: HashMap::default(),
            chain_id: Arc::new(RwLock::new(None)),
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

    /// Replaces a local safe head with the given header.
    pub fn replace_local_safe_head(&mut self, chain_id: u64, header: Sealed<Header>) {
        self.local_safe_heads.insert(chain_id, header);
    }

    /// Records a block that was built locally so future lookups can avoid the oracle.
    pub fn remember_known_block(
        &mut self,
        chain_id: u64,
        header: Sealed<Header>,
        receipts: Vec<OpReceiptEnvelope>,
    ) {
        self.known_blocks
            .entry(chain_id)
            .or_default()
            .insert(header.hash(), KnownBlock { header, receipts });
    }

    fn known_block_by_hash(&self, chain_id: u64, block_hash: B256) -> Option<&KnownBlock> {
        self.known_blocks.get(&chain_id)?.get(&block_hash)
    }

    fn known_head_block_by_number(&self, chain_id: u64, number: u64) -> Option<&KnownBlock> {
        let head = self.local_safe_heads.get(&chain_id)?;
        (head.number == number).then(|| self.known_block_by_hash(chain_id, head.hash()))?
    }

    /// Fetch the [Header] for the block with the given hash.
    pub async fn header_by_hash(
        &self,
        chain_id: u64,
        block_hash: B256,
    ) -> Result<Header, <Self as InteropProvider>::Error> {
        if let Some(block) = self.known_block_by_hash(chain_id, block_hash) {
            return Ok(block.header.clone().into_inner());
        }
        if let Some(head) =
            self.local_safe_heads.get(&chain_id).filter(|head| head.hash() == block_hash)
        {
            return Ok(head.clone().into_inner());
        }

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
        if let Some(block) = self.known_block_by_hash(chain_id, block_hash) {
            return Ok(block.receipts.clone());
        }

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
        let Some(sealed) = self.local_safe_heads.get(&chain_id).cloned() else {
            return Err(PreimageOracleError::Other("Missing local safe header".to_string()).into());
        };
        let mut current_hash = sealed.hash();
        let mut header = sealed.into_inner();

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
                let block_hash = match eip_2935_history_lookup(
                    &header,
                    number,
                    current_hash,
                    self,
                    self,
                )
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
        }

        Ok(header)
    }

    /// Fetch all receipts for a given block by number.
    async fn receipts_by_number(
        &self,
        chain_id: u64,
        number: u64,
    ) -> Result<Vec<OpReceiptEnvelope>, Self::Error> {
        if let Some(block) = self.known_head_block_by_number(chain_id, number) {
            return Ok(block.receipts.clone());
        }

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

    fn hint_account_proof(&self, address: Address, block_hash: B256) -> Result<(), Self::Error> {
        kona_proof::block_on(async move {
            HintType::L2AccountProof
                .with_data(&[block_hash.as_slice(), address.as_slice()])
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
        block_hash: B256,
    ) -> Result<(), Self::Error> {
        kona_proof::block_on(async move {
            HintType::L2AccountStorageProof
                .with_data(&[
                    block_hash.as_slice(),
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
    use alloc::{collections::BTreeMap, format, string::String, sync::Arc, vec, vec::Vec};
    use alloy_consensus::{Header, Receipt, ReceiptWithBloom};
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Address, B256, Log, LogData, Sealable, U256, keccak256};
    use alloy_rlp::{Decodable, Encodable};
    use alloy_sol_types::SolEvent;
    use async_trait::async_trait;
    use kona_genesis::RollupConfig;
    use kona_interop::{
        DependencySet, ExecutingMessage, MESSAGE_EXPIRY_WINDOW, MessageGraph, MessageIdentifier,
        RawMessagePayload, SuperRoot,
    };
    use kona_mpt::ordered_trie_with_encoder;
    use kona_preimage::{
        HintWriterClient, PreimageKey, PreimageKeyType, PreimageOracleClient,
        errors::PreimageOracleResult,
    };
    use kona_proof::errors::OracleProviderError;
    use kona_protocol::Predeploys;
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

        let mut rollup_config = RollupConfig::default();
        rollup_config.hardforks.isthmus_time = Some(1746806401);

        build_provider_with_safe_head(client, fixture.chain_id, safe_head_header, rollup_config)
    }

    fn build_provider_with_safe_head(
        client: MockCommsClient,
        chain_id: u64,
        safe_head_header: Header,
        rollup_config: RollupConfig,
    ) -> OracleInteropProvider<MockCommsClient> {
        let mut local_safe_heads = HashMap::default();
        local_safe_heads.insert(chain_id, safe_head_header.seal_slow());

        let mut rollup_configs = HashMap::default();
        rollup_configs.insert(chain_id, rollup_config);

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

    fn build_provider_with_heads(
        client: MockCommsClient,
        local_safe_heads: HashMap<u64, Sealed<Header>>,
        rollup_configs: HashMap<u64, RollupConfig>,
    ) -> OracleInteropProvider<MockCommsClient> {
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

    fn interop_rollup_config() -> RollupConfig {
        RollupConfig {
            block_time: 1,
            hardforks: kona_genesis::HardForkConfig { interop_time: Some(0), ..Default::default() },
            ..Default::default()
        }
    }

    fn insert_header_preimage(preimages: &mut BTreeMap<[u8; 32], Vec<u8>>, header: &Header) {
        let mut header_rlp = Vec::new();
        header.encode(&mut header_rlp);
        let key: [u8; 32] =
            PreimageKey::new(*header.hash_slow(), PreimageKeyType::Keccak256).into();
        preimages.insert(key, header_rlp);
    }

    fn insert_receipt_trie_preimages(
        preimages: &mut BTreeMap<[u8; 32], Vec<u8>>,
        receipts: &[OpReceiptEnvelope],
    ) -> B256 {
        let mut trie = ordered_trie_with_encoder(receipts, |receipt, buf| receipt.encode_2718(buf));
        let root = trie.root();

        for (_, value) in trie.take_proof_nodes().into_inner() {
            let node_hash = keccak256(value.as_ref());
            let key: [u8; 32] = PreimageKey::new(*node_hash, PreimageKeyType::Keccak256).into();
            preimages.insert(key, value.to_vec());
        }

        root
    }

    fn origin_log(origin_address: Address) -> Log {
        Log {
            address: origin_address,
            data: LogData::new(vec![B256::repeat_byte(0x11)], b"remote-origin".to_vec().into())
                .expect("valid origin log"),
        }
    }

    fn remote_origin_receipt(
        origin_chain_id: u64,
        origin_block_number: u64,
        origin_timestamp: u64,
        payload_hash: B256,
    ) -> OpReceiptEnvelope {
        let event = ExecutingMessage {
            payloadHash: payload_hash,
            identifier: MessageIdentifier {
                origin: Address::repeat_byte(0x77),
                blockNumber: U256::from(origin_block_number),
                logIndex: U256::ZERO,
                timestamp: U256::from(origin_timestamp),
                chainId: U256::from(origin_chain_id),
            },
        };

        OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
            receipt: Receipt {
                logs: vec![Log {
                    address: Predeploys::CROSS_L2_INBOX,
                    data: ExecutingMessage::encode_log_data(&event),
                }],
                ..Default::default()
            },
            ..Default::default()
        })
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
    async fn test_message_graph_resolve_succeeds_when_remote_origin_is_only_known_locally() {
        let origin_chain_id = 10;
        let executing_chain_id = 11;
        let origin_block_number = 42;
        let origin_timestamp = 100;
        let executing_timestamp = 101;
        let origin_address = Address::repeat_byte(0x77);

        let origin_head = Header {
            number: origin_block_number,
            timestamp: origin_timestamp,
            receipts_root: keccak256(b"origin-receipts-not-in-oracle"),
            ..Default::default()
        };

        let origin_receipt = OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
            receipt: Receipt { logs: vec![origin_log(origin_address)], ..Default::default() },
            ..Default::default()
        });
        let payload_hash = keccak256(RawMessagePayload::from(&origin_receipt.logs()[0]).as_ref());
        let executing_receipt = remote_origin_receipt(
            origin_chain_id,
            origin_block_number,
            origin_timestamp,
            payload_hash,
        );
        let mut preimages = BTreeMap::default();
        let receipts_root = insert_receipt_trie_preimages(
            &mut preimages,
            core::slice::from_ref(&executing_receipt),
        );

        let executing_head = Header {
            number: 7,
            timestamp: executing_timestamp,
            receipts_root,
            ..Default::default()
        };
        insert_header_preimage(&mut preimages, &executing_head);

        let provider = build_provider_with_heads(
            MockCommsClient { preimages },
            HashMap::from_iter([
                (origin_chain_id, origin_head.clone().seal_slow()),
                (executing_chain_id, executing_head.clone().seal_slow()),
            ]),
            HashMap::from_iter([
                (origin_chain_id, interop_rollup_config()),
                (executing_chain_id, interop_rollup_config()),
            ]),
        );
        let mut provider = provider;
        provider.remember_known_block(
            origin_chain_id,
            origin_head.clone().seal_slow(),
            vec![origin_receipt],
        );

        let heads_to_check =
            HashMap::from_iter([(executing_chain_id, executing_head.clone().seal_slow())]);
        let rollup_configs = HashMap::from_iter([
            (origin_chain_id, interop_rollup_config()),
            (executing_chain_id, interop_rollup_config()),
        ]);

        let graph = MessageGraph::derive(
            &heads_to_check,
            &provider,
            &rollup_configs,
            MESSAGE_EXPIRY_WINDOW,
        )
        .await
        .expect("graph derivation should succeed using the executing chain oracle data");

        graph.resolve().await.expect(
            "once locally built replacement blocks are treated as known, remote-origin \
                 resolution should not require oracle receipt preimages for them",
        );
    }
}
