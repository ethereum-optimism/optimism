//! [`InteropProvider`] trait implementation using a [`CommsClient`] data source.

use crate::{BootInfo, HintType};
use alloc::{boxed::Box, collections::BTreeMap, string::ToString, sync::Arc, vec::Vec};
use alloy_consensus::{Header, Sealed};
use alloy_eips::eip2718::Decodable2718;
use alloy_primitives::{Address, B256};
use alloy_rlp::Decodable;
use async_trait::async_trait;
use core::fmt::Debug;
use kona_interop::{BlocksWithReceipts, InitiatingLogs, InteropProvider, LogCommitment};
use kona_mpt::{OrderedListWalker, TrieHinter, TrieNode, TrieProvider};
use kona_preimage::{
    CommsClient, PreimageKey, PreimageKeyType, PreimageOracleClient, errors::PreimageOracleError,
};
use kona_proof::{eip_2935_history_lookup, errors::OracleProviderError};
use kona_registry::HashMap;
use op_alloy_consensus::OpReceiptEnvelope;
use spin::Mutex;

/// Commitments to the logs of canonical blocks, which an [`OracleInteropProvider`] serves for
/// blocks below the local safe heads.
pub trait LogCommitmentSource: Debug + Send + Sync {
    /// The commitment to the logs of block `header` on `chain_id`, or `None` to read its receipts.
    fn log_commitment(&self, chain_id: u64, header: &Header) -> Option<Box<dyn LogCommitment>>;
}

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
    /// Log commitments for blocks below the local safe heads; `None` reads every block's receipts.
    log_commitments: Option<Arc<dyn LogCommitmentSource>>,
}

impl<C> Clone for OracleInteropProvider<C> {
    fn clone(&self) -> Self {
        Self {
            oracle: self.oracle.clone(),
            boot: self.boot.clone(),
            local_safe_heads: self.local_safe_heads.clone(),
            headers_by_number: Mutex::new(self.headers_by_number.lock().clone()),
            receipts_by_hash: Mutex::new(self.receipts_by_hash.lock().clone()),
            log_commitments: self.log_commitments.clone(),
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
            log_commitments: None,
        }
    }

    /// Serves log commitments from `source` for blocks below the local safe heads.
    pub fn with_log_commitments(mut self, source: Arc<dyn LogCommitmentSource>) -> Self {
        self.log_commitments = Some(source);
        self
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

        // Verify the receipts against the header's receipts root.
        HintType::L2Receipts
            .with_data(&[block_hash.as_ref(), chain_id.to_be_bytes().as_slice()])
            .send(self.oracle.as_ref())
            .await?;
        let receipts = receipts_from_trie(self.oracle.as_ref(), header.receipts_root)?;

        if self.local_safe_heads.get(&chain_id).is_some_and(|head| head.hash() == block_hash) {
            self.receipts_by_hash.lock().insert((chain_id, block_hash), receipts.clone());
        }
        Ok(receipts)
    }

    /// Fetch all receipts for a given block by hash.
    pub async fn receipts_by_hash(
        &self,
        chain_id: u64,
        block_hash: B256,
    ) -> Result<Vec<OpReceiptEnvelope>, <Self as InteropProvider>::Error> {
        if let Some(receipts) = self.receipts_by_hash.lock().get(&(chain_id, block_hash)) {
            return Ok(receipts.clone());
        }
        let header = self.header_by_hash(chain_id, block_hash).await?;
        self.derive_receipts(chain_id, block_hash, &header).await
    }

    /// The blocks in `heads` with their receipts, as [`MessageGraph::derive`] takes them.
    ///
    /// [`MessageGraph::derive`]: kona_interop::MessageGraph::derive
    pub async fn blocks_with_receipts(
        &self,
        heads: &HashMap<u64, Sealed<Header>>,
    ) -> Result<BlocksWithReceipts, <Self as InteropProvider>::Error> {
        let mut blocks = HashMap::default();
        for (chain_id, head) in heads {
            let receipts = self.receipts_by_hash(*chain_id, head.hash()).await?;
            blocks.insert(*chain_id, (head.clone(), receipts));
        }
        Ok(blocks)
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
        let Some(sealed) = self.local_safe_heads.get(&chain_id) else {
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
        let (mut current_hash, mut header) = cached.unwrap_or_else(|| {
            let hash = sealed.hash();
            let header = sealed.inner().clone();
            if number < header.number {
                self.headers_by_number
                    .lock()
                    .insert((chain_id, header.number), (hash, header.clone()));
            }
            (hash, header)
        });
        if header.number == number {
            return Ok(header);
        }

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

    /// A local safe head can be replaced, so only a block below it is served a commitment.
    async fn initiating_logs(
        &self,
        chain_id: u64,
        header: &Header,
    ) -> Result<InitiatingLogs, Self::Error> {
        let below_head =
            self.local_safe_heads.get(&chain_id).is_some_and(|head| header.number < head.number);
        if below_head &&
            let Some(commitment) = self
                .log_commitments
                .as_ref()
                .and_then(|source| source.log_commitment(chain_id, header))
        {
            return Ok(InitiatingLogs::Committed(commitment));
        }
        let receipts = self.derive_receipts(chain_id, header.hash_slow(), header).await?;
        Ok(InitiatingLogs::Receipts(
            receipts.into_iter().flat_map(OpReceiptEnvelope::into_logs).collect(),
        ))
    }
}

impl<C> TrieProvider for OracleInteropProvider<C>
where
    C: CommsClient + Send + Sync + Clone,
{
    type Error = OracleProviderError;

    fn trie_node_by_hash(&self, key: B256) -> Result<TrieNode, Self::Error> {
        OracleTrieProvider(self.oracle.as_ref()).trie_node_by_hash(key)
    }
}

/// Walks the receipts trie under `receipts_root` and decodes its receipts in transaction order.
///
/// The SP1 log-commitment program must see exactly the receipts that [`InteropProvider`] lookups
/// return, so both decode through this function.
///
/// An error does not tell a missing trie node from one that does not decode. A caller that maps an
/// error to a verdict needs an oracle that panics on a missing key, like SP1's `WitnessOracle`.
pub fn receipts_from_trie<O: PreimageOracleClient>(
    oracle: &O,
    receipts_root: B256,
) -> Result<Vec<OpReceiptEnvelope>, OracleProviderError> {
    let trie_walker =
        OrderedListWalker::try_new_hydrated(receipts_root, &OracleTrieProvider(oracle))
            .map_err(OracleProviderError::TrieWalker)?;

    trie_walker
        .into_iter()
        .map(|(_, rlp)| {
            let envelope = OpReceiptEnvelope::decode_2718(&mut rlp.as_ref())?;
            Ok(envelope)
        })
        .collect::<Result<Vec<_>, _>>()
        .map_err(OracleProviderError::Rlp)
}

/// A [`TrieProvider`] that reads keccak-keyed trie nodes from a preimage oracle.
struct OracleTrieProvider<'a, O>(&'a O);

impl<O: PreimageOracleClient> TrieProvider for OracleTrieProvider<'_, O> {
    type Error = OracleProviderError;

    fn trie_node_by_hash(&self, key: B256) -> Result<TrieNode, Self::Error> {
        kona_proof::block_on(async move {
            let trie_node_rlp = self
                .0
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
    use alloc::{
        collections::{BTreeMap, BTreeSet},
        format,
        string::String,
        sync::Arc,
        vec,
        vec::Vec,
    };
    use alloy_consensus::{EMPTY_ROOT_HASH, Header};
    use alloy_primitives::{B256, Sealable, U256, keccak256};
    use alloy_rlp::{Decodable, EMPTY_STRING_CODE};
    use async_trait::async_trait;
    use kona_genesis::{ChainDependency, RollupConfig};
    use kona_interop::{
        ChainBuilder, DependencySet, ExecutingMessageBuilder, LeafList, MESSAGE_EXPIRY_WINDOW,
        MessageGraph, MessageGraphError, SuperRoot, SuperchainBuilder,
    };
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

        OracleInteropProvider::new(Arc::new(client), boot_info(rollup_configs), local_safe_heads)
    }

    fn boot_info(rollup_configs: HashMap<u64, RollupConfig>) -> BootInfo {
        BootInfo {
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
        }
    }

    /// A stale block number with a recent timestamp is rejected without reading chain A's history.
    ///
    /// Chain B's head executes a message whose identifier pairs chain A's recent timestamp with
    /// block number `1`. The verdict must come from the identifier alone: resolving block `1`
    /// through the real [`OracleInteropProvider`] would walk back through chain A's entire
    /// history one header at a time.
    #[tokio::test(flavor = "multi_thread")]
    async fn test_stale_initiating_block_number_rejected_without_history_walk() {
        const CHAIN_A: u64 = u64::MAX - 1;
        const CHAIN_B: u64 = u64::MAX;
        const BLOCK_TIME: u64 = 2;
        const HEAD: u64 = 1_000;
        const STALE_TS: u64 = (HEAD - 1) * BLOCK_TIME;

        let mut client = MockCommsClient { preimages: BTreeMap::new(), reads: Default::default() };
        client
            .preimages
            .insert(PreimageKey::new_keccak256(*EMPTY_ROOT_HASH).into(), vec![EMPTY_STRING_CODE]);

        // Chain A: a linear history `0..=HEAD` without EIP-2935, so resolving an old block by
        // number can only walk parent hashes back from the head.
        let mut history = BTreeSet::new();
        let mut head_a = insert_test_header(
            &mut client,
            Header { receipts_root: EMPTY_ROOT_HASH, ..Default::default() },
        );
        for number in 1..=HEAD {
            history.insert(PreimageKey::new_keccak256(*head_a.hash()));
            head_a = insert_test_header(
                &mut client,
                Header {
                    number,
                    timestamp: number * BLOCK_TIME,
                    parent_hash: head_a.hash(),
                    receipts_root: EMPTY_ROOT_HASH,
                    ..Default::default()
                },
            );
        }

        // Chain B: the head executes the stale-numbered message.
        let mut chain_b = ChainBuilder::default();
        chain_b.add_executing_message(
            ExecutingMessageBuilder::default()
                .with_message_hash(B256::repeat_byte(0x11))
                .with_origin_chain_id(CHAIN_A)
                .with_origin_block_number(1)
                .with_origin_timestamp(STALE_TS),
        );
        let receipts_root = insert_test_receipts(&mut client, &chain_b.receipts);
        let head_b = insert_test_header(
            &mut client,
            Header {
                number: HEAD,
                timestamp: HEAD * BLOCK_TIME,
                receipts_root,
                ..Default::default()
            },
        );

        let mut rollup_config = RollupConfig { block_time: BLOCK_TIME, ..Default::default() };
        rollup_config.hardforks.lagoon_time = Some(0);
        let cfgs: HashMap<u64, RollupConfig> =
            [(CHAIN_A, rollup_config.clone()), (CHAIN_B, rollup_config)].into_iter().collect();
        let heads: HashMap<_, _> = [(CHAIN_A, head_a), (CHAIN_B, head_b)].into_iter().collect();
        let dep_set = DependencySet {
            dependencies: [(CHAIN_A, ChainDependency {}), (CHAIN_B, ChainDependency {})].into(),
            override_message_expiry_window: None,
        };

        let provider = OracleInteropProvider::new(
            Arc::new(client.clone()),
            boot_info(cfgs.clone()),
            heads.clone(),
        );
        let blocks = provider.blocks_with_receipts(&heads).await.unwrap();
        let result =
            MessageGraph::derive(&blocks, &provider, &cfgs, &dep_set, MESSAGE_EXPIRY_WINDOW)
                .resolve()
                .await;

        let walked = client.reads.lock().iter().filter(|key| history.contains(*key)).count();
        assert_eq!(walked, 0, "proof walked {walked} headers of chain A's history");
        let Err(MessageGraphError::InvalidMessages(invalid)) = result else {
            panic!("expected InvalidMessages, got {result:?}");
        };
        assert_eq!(invalid.len(), 1);
        assert!(
            matches!(
                invalid[&CHAIN_B],
                MessageGraphError::InvalidMessageBlockNumber { block_number, timestamp: STALE_TS }
                    if block_number == U256::from(1)
            ),
            "unexpected verdict: {:?}",
            invalid[&CHAIN_B]
        );
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
        let reads = client.reads.clone();
        let provider = build_provider(client, &fixture);
        let expected_hash: B256 = fixture.target_block_hash.parse().unwrap();

        let header = provider
            .header_by_number(fixture.chain_id, fixture.target_block_number)
            .await
            .expect("header_by_number should succeed via EIP-2935 fast path");

        assert_eq!(header.hash_slow(), expected_hash);
        assert_eq!(header.number, fixture.target_block_number);
        let reads_after_first = reads.lock().len();
        assert!(reads_after_first > 0);
        let cached =
            provider.header_by_number(fixture.chain_id, fixture.target_block_number).await.unwrap();
        assert_eq!(header, cached);
        assert_eq!(
            reads.lock().len(),
            reads_after_first,
            "repeated lookup must reuse verified headers"
        );
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
    async fn test_header_cache_handles_zero_and_maximum_keys() {
        for (chain_id, number) in
            [(0, 0), (0, u64::MAX - 1), (u64::MAX, 0), (u64::MAX, u64::MAX - 1)]
        {
            let (mut client, fixture) = load_fixture();
            let parent = insert_test_header(&mut client, Header { number, ..Default::default() });
            let head =
                Header { number: number + 1, parent_hash: parent.hash(), ..Default::default() }
                    .seal_slow();
            let reads = client.reads.clone();
            let mut provider = build_provider(client, &fixture);
            let config = provider.boot.rollup_configs[&fixture.chain_id].clone();
            provider.boot.rollup_configs.insert(chain_id, config);
            provider.replace_local_safe_head(chain_id, head.clone());

            assert_eq!(provider.header_by_number(chain_id, number).await.unwrap(), *parent.inner());
            assert_eq!(reads.lock().len(), 1);
            for expected in [parent.inner(), head.inner()] {
                assert_eq!(
                    provider.header_by_number(chain_id, expected.number).await.unwrap(),
                    *expected
                );
            }
            assert_eq!(reads.lock().len(), 1, "boundary cache hits must not reread the oracle");
        }
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
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_message_graph_revalidates_cached_source_after_replacement() {
        use alloy_consensus::{Receipt, ReceiptWithBloom};

        let mut superchain = SuperchainBuilder::new();
        for chain in [1, 2] {
            superchain
                .chain(chain)
                .with_number(1)
                .with_timestamp(2)
                .with_block_time(2)
                .with_lagoon_activation_time(0);
        }
        superchain.chain(1).add_initiating_message(b"payload".to_vec().into());
        superchain.chain(2).add_executing_message(
            ExecutingMessageBuilder::default()
                .with_origin_chain_id(1)
                .with_origin_block_number(1)
                .with_origin_timestamp(2)
                .with_message_hash(keccak256(b"payload")),
        );
        let (headers, configs, mock) = superchain.build();
        let (mut client, fixture) = load_fixture();
        let mut safe_heads = HashMap::default();
        for (chain_id, (header, receipts)) in headers {
            let receipts_root = insert_test_receipts(&mut client, &receipts);
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
            let blocks = provider.blocks_with_receipts(provider.local_safe_heads()).await.unwrap();
            let graph = MessageGraph::derive(
                &blocks,
                &provider,
                &configs,
                &dependency_set,
                MESSAGE_EXPIRY_WINDOW,
            );
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
        let blocks = provider.blocks_with_receipts(&heads_to_check).await.unwrap();
        let graph = MessageGraph::derive(
            &blocks,
            &provider,
            &configs,
            &dependency_set,
            MESSAGE_EXPIRY_WINDOW,
        );
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
        provider.initiating_logs(fixture.chain_id, &headers[0]).await.unwrap();
        assert_eq!(provider.receipts_by_hash.lock().len(), 1);
        provider.replace_local_safe_head(fixture.chain_id, headers[1].clone());
        assert_eq!(provider.receipts_by_hash.lock().len(), 0, "replaced heads must be evicted");
        provider.initiating_logs(fixture.chain_id, &headers[1]).await.unwrap();
        assert_eq!(provider.receipts_by_hash.lock().len(), 1);
    }

    /// Serves a [`LeafList`] for each receipts root it holds and records the requested blocks.
    #[derive(Debug, Default)]
    struct TestLogCommitments {
        receipts: BTreeMap<B256, Vec<OpReceiptEnvelope>>,
        requests: spin::Mutex<Vec<(u64, u64)>>,
    }

    impl LogCommitmentSource for TestLogCommitments {
        fn log_commitment(&self, chain_id: u64, header: &Header) -> Option<Box<dyn LogCommitment>> {
            self.requests.lock().push((chain_id, header.number));
            let receipts = self.receipts.get(&header.receipts_root)?;
            Some(Box::new(LeafList::new(receipts)))
        }
    }

    /// Chains A and B with block `1` below head `2`: A's head executes a wrong message into B's
    /// block `1`, and B's head executes messages into A's blocks `1` and `2`.
    struct RoundsFixture {
        client: MockCommsClient,
        heads: HashMap<u64, Sealed<Header>>,
        replacements: HashMap<u64, Sealed<Header>>,
        below_head: Vec<(B256, Vec<OpReceiptEnvelope>)>,
        configs: HashMap<u64, RollupConfig>,
        dependency_set: DependencySet,
    }

    fn rounds_fixture() -> RoundsFixture {
        use alloy_consensus::{Receipt, ReceiptWithBloom};

        const A: u64 = 1;
        const B: u64 = 2;
        let mut client = MockCommsClient { preimages: BTreeMap::new(), reads: Default::default() };
        let message = |chain, number, payload: &[u8]| {
            ExecutingMessageBuilder::default()
                .with_message_hash(keccak256(payload))
                .with_origin_chain_id(chain)
                .with_origin_block_number(number)
                .with_origin_timestamp(2 * number)
        };
        let mut a1 = ChainBuilder::default();
        a1.add_initiating_message(b"a1".to_vec().into());
        let mut b1 = ChainBuilder::default();
        b1.add_initiating_message(b"b1".to_vec().into());
        let mut a2 = ChainBuilder::default();
        a2.add_initiating_message(b"a2".to_vec().into())
            .add_executing_message(message(B, 1, b"wrong"));
        let mut b2 = ChainBuilder::default();
        b2.add_executing_message(message(A, 1, b"a1")).add_executing_message(message(A, 2, b"a2"));

        let mut heads = HashMap::default();
        let mut replacements = HashMap::default();
        let mut below_head = Vec::new();
        for (chain_id, block_1, block_2) in
            [(A, a1.receipts, a2.receipts), (B, b1.receipts, b2.receipts)]
        {
            let receipts_root = insert_test_receipts(&mut client, &block_1);
            let block_1_header = insert_test_header(
                &mut client,
                Header { number: 1, timestamp: 2, receipts_root, ..Default::default() },
            );
            below_head.push((receipts_root, block_1));
            let head = |receipts_root| Header {
                number: 2,
                timestamp: 4,
                parent_hash: block_1_header.hash(),
                receipts_root,
                ..Default::default()
            };
            let receipts_root = insert_test_receipts(&mut client, &block_2);
            heads.insert(chain_id, insert_test_header(&mut client, head(receipts_root)));
            let no_logs = vec![OpReceiptEnvelope::Eip1559(ReceiptWithBloom {
                receipt: Receipt::default(),
                ..Default::default()
            })];
            let receipts_root = insert_test_receipts(&mut client, &no_logs);
            replacements.insert(chain_id, insert_test_header(&mut client, head(receipts_root)));
        }
        let mut config = RollupConfig { block_time: 2, ..Default::default() };
        config.hardforks.lagoon_time = Some(0);
        RoundsFixture {
            client,
            heads,
            replacements,
            below_head,
            configs: [(A, config.clone()), (B, config)].into_iter().collect(),
            dependency_set: DependencySet {
                dependencies: [(A, ChainDependency {}), (B, ChainDependency {})].into(),
                override_message_expiry_window: None,
            },
        }
    }

    /// The invalid chains of each `consolidate_once` round, with stand-in replacement heads.
    async fn run_rounds(
        mut provider: OracleInteropProvider<MockCommsClient>,
        fixture: &RoundsFixture,
    ) -> Vec<BTreeSet<u64>> {
        let mut replaced = BTreeSet::new();
        let mut rounds = Vec::new();
        loop {
            let heads: HashMap<_, _> = provider
                .local_safe_heads()
                .iter()
                .filter(|(chain_id, _)| !replaced.contains(*chain_id))
                .map(|(chain_id, head)| (*chain_id, head.clone()))
                .collect();
            let blocks = provider.blocks_with_receipts(&heads).await.unwrap();
            let result = MessageGraph::derive(
                &blocks,
                &provider,
                &fixture.configs,
                &fixture.dependency_set,
                MESSAGE_EXPIRY_WINDOW,
            )
            .resolve()
            .await;
            let invalid: BTreeSet<u64> = match result {
                Ok(()) => return rounds,
                Err(MessageGraphError::InvalidMessages(invalid)) => invalid.into_keys().collect(),
                Err(err) => panic!("unexpected resolution {err:?}"),
            };
            for chain_id in &invalid {
                provider.replace_local_safe_head(*chain_id, fixture.replacements[chain_id].clone());
            }
            replaced.extend(invalid.iter().copied());
            rounds.push(invalid);
        }
    }

    fn rounds_boot_info(fixture: &RoundsFixture) -> BootInfo {
        BootInfo {
            dependency_set: fixture.dependency_set.clone(),
            ..boot_info(fixture.configs.clone())
        }
    }

    /// Pins equal rounds with and without commitments, and no read of a served receipts trie.
    #[tokio::test(flavor = "multi_thread")]
    async fn test_rounds_with_log_commitments_match_receipts() {
        let fixture = rounds_fixture();
        let provider = OracleInteropProvider::new(
            Arc::new(fixture.client.clone()),
            rounds_boot_info(&fixture),
            fixture.heads.clone(),
        );
        let expected = run_rounds(provider, &fixture).await;
        assert_eq!(expected, vec![BTreeSet::from([1]), BTreeSet::from([2])]);

        let mut client = MockCommsClient { reads: Default::default(), ..fixture.client.clone() };
        let mut source = TestLogCommitments::default();
        for (receipts_root, receipts) in &fixture.below_head {
            source.receipts.insert(*receipts_root, receipts.clone());
            client.preimages.remove(&<[u8; 32]>::from(PreimageKey::new_keccak256(**receipts_root)));
        }
        let reads = client.reads.clone();
        let source = Arc::new(source);
        let provider = OracleInteropProvider::new(
            Arc::new(client),
            rounds_boot_info(&fixture),
            fixture.heads.clone(),
        )
        .with_log_commitments(source.clone());

        assert_eq!(run_rounds(provider, &fixture).await, expected);
        assert!(!source.requests.lock().is_empty());
        assert!(
            source.requests.lock().iter().all(|(_, number)| *number == 1),
            "only blocks below the heads are requested"
        );
        let below_head_roots: BTreeSet<_> =
            fixture.below_head.iter().map(|(root, _)| PreimageKey::new_keccak256(**root)).collect();
        assert!(reads.lock().iter().all(|key| !below_head_roots.contains(key)));
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn test_log_commitment_served_only_below_head() {
        let fixture = rounds_fixture();
        let mut source = TestLogCommitments::default();
        for (receipts_root, receipts) in &fixture.below_head {
            source.receipts.insert(*receipts_root, receipts.clone());
        }
        let block_1 = |provider: &OracleInteropProvider<_>| {
            kona_proof::block_on(provider.header_by_number(1, 1)).unwrap()
        };
        let committed = |provider: &OracleInteropProvider<_>, chain_id, header: &Header| {
            let logs = kona_proof::block_on(provider.initiating_logs(chain_id, header)).unwrap();
            matches!(logs, InitiatingLogs::Committed(_))
        };
        let mut provider = OracleInteropProvider::new(
            Arc::new(fixture.client.clone()),
            rounds_boot_info(&fixture),
            fixture.heads.clone(),
        );
        assert!(!committed(&provider, 1, &block_1(&provider)), "no source");

        provider = provider.with_log_commitments(Arc::new(source));
        assert!(committed(&provider, 1, &block_1(&provider)));
        assert!(!committed(&provider, 1, fixture.heads[&1].inner()));
        assert!(!committed(&provider, 3, &block_1(&provider)), "unknown chain");

        provider.replace_local_safe_head(1, fixture.replacements[&1].clone());
        assert!(!committed(&provider, 1, fixture.replacements[&1].inner()));
        assert!(committed(&provider, 1, &block_1(&provider)));
        assert!(committed(&provider.clone(), 1, &block_1(&provider)), "clone");
    }
}
