//! Contains the concrete implementation of the [`L2ChainProvider`] trait for the client program.

use crate::{HintType, eip2935::eip_2935_history_lookup, errors::OracleProviderError};
use alloc::{boxed::Box, collections::BTreeMap, sync::Arc, vec::Vec};
use alloy_consensus::{BlockBody, Header};
use alloy_eips::eip2718::Decodable2718;
use alloy_primitives::{Address, B256, Bytes};
use alloy_rlp::Decodable;
use async_trait::async_trait;
use kona_derive::L2ChainProvider;
use kona_driver::PipelineCursor;
use kona_executor::TrieDBProvider;
use kona_genesis::{RollupConfig, SystemConfig};
use kona_mpt::{OrderedListWalker, TrieHinter, TrieNode, TrieProvider};
use kona_preimage::{CommsClient, PreimageKey, PreimageKeyType};
use kona_protocol::{BatchValidationProvider, L2BlockInfo, to_system_config};
use op_alloy_consensus::{OpBlock, OpTxEnvelope};
use spin::RwLock;

/// The oracle-backed L2 chain provider for the client program.
#[derive(Debug, Clone)]
pub struct OracleL2ChainProvider<T: CommsClient> {
    /// The L2 safe head block hash.
    l2_head: B256,
    /// The rollup configuration.
    rollup_config: Arc<RollupConfig>,
    /// The preimage oracle client.
    oracle: Arc<T>,
    /// The derivation pipeline cursor
    cursor: Option<Arc<RwLock<PipelineCursor>>>,
    /// The L2 chain ID to use for the provider's hints.
    chain_id: Option<u64>,
    /// Cache of canonical headers and their hashes, keyed by block number, for ancestors of the
    /// L2 safe head.
    ///
    /// Entries are discovered by walking parent hashes (or EIP-2935 lookups) back from the safe
    /// head, so they are canonical by construction, and the safe head only advances along the
    /// same lineage within a program run, so entries never need invalidation. This turns the
    /// by-number lookups of span batch overlap validation (one per overlapped block, walked in
    /// ascending order) from a quadratic number of header preimage reads into a linear one, and
    /// mirrors [`OracleL1ChainProvider`](crate::l1::OracleL1ChainProvider)'s by-number cache.
    header_by_number: BTreeMap<u64, (B256, Header)>,
}

impl<T: CommsClient> OracleL2ChainProvider<T> {
    /// Creates a new [`OracleL2ChainProvider`] with the given boot information and oracle client.
    pub const fn new(l2_head: B256, rollup_config: Arc<RollupConfig>, oracle: Arc<T>) -> Self {
        Self {
            l2_head,
            rollup_config,
            oracle,
            cursor: None,
            chain_id: None,
            header_by_number: BTreeMap::new(),
        }
    }

    /// Sets the L2 chain ID to use for the provider's hints.
    pub const fn set_chain_id(&mut self, chain_id: Option<u64>) {
        self.chain_id = chain_id;
    }

    /// Updates the derivation pipeline cursor
    pub fn set_cursor(&mut self, cursor: Arc<RwLock<PipelineCursor>>) {
        self.cursor = Some(cursor);
    }

    /// Fetches the latest known safe head block hash according to the derivation pipeline cursor
    /// or uses the initial `l2_head` value if no cursor is set.
    pub async fn l2_safe_head(&self) -> Result<B256, OracleProviderError> {
        self.cursor
            .as_ref()
            .map_or(Ok(self.l2_head), |cursor| Ok(cursor.read().l2_safe_head().block_info.hash))
    }
}

impl<T: CommsClient> OracleL2ChainProvider<T> {
    /// Returns a [Header] corresponding to the given L2 block number, by walking back from the
    /// closest cached ancestor, or from the L2 safe head when nothing closer is cached.
    async fn header_by_number(&mut self, block_number: u64) -> Result<Header, OracleProviderError> {
        if let Some((_, header)) = self.header_by_number.get(&block_number) {
            return Ok(header.clone());
        }

        // Start from the closest cached ancestor whose number is greater than the target,
        // falling back to the L2 safe head when no usable cache entry exists.
        let cached_ancestor = block_number
            .checked_add(1)
            .and_then(|n| self.header_by_number.range(n..).next())
            .map(|(_, (hash, header))| (*hash, header.clone()));
        let (mut current_hash, mut header) = match cached_ancestor {
            Some(entry) => entry,
            None => {
                // Fetch the starting block header.
                let safe_hash = self.l2_safe_head().await?;
                let safe_header = self.header_by_hash(safe_hash)?;

                // Check if the block number is in range. If not, we can fail early.
                if block_number > safe_header.number {
                    return Err(OracleProviderError::BlockNumberPastHead(
                        block_number,
                        safe_header.number,
                    ));
                }
                self.header_by_number.insert(safe_header.number, (safe_hash, safe_header.clone()));
                (safe_hash, safe_header)
            }
        };

        let mut linear_fallback = false;
        while header.number > block_number {
            if self.rollup_config.is_isthmus_active(header.timestamp) && !linear_fallback {
                // If Isthmus is active, the EIP-2935 contract is used to perform leaping lookbacks
                // through consulting the ring buffer within the contract. If this
                // lookup fails for any reason, we fall back to linear walk back.
                let block_hash =
                    match eip_2935_history_lookup(&header, block_number, current_hash, self, self)
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
                header = self.header_by_hash(block_hash)?;
            } else {
                // Walk back the block headers one-by-one until the desired block number is reached.
                current_hash = header.parent_hash;
                header = self.header_by_hash(header.parent_hash)?;
            }
            // Cache every header discovered on the walk, so later lookups at or below this
            // height resume from here instead of re-reading the chain above it.
            self.header_by_number.insert(header.number, (current_hash, header.clone()));
        }

        Ok(header)
    }
}

#[async_trait]
impl<T: CommsClient + Send + Sync> BatchValidationProvider for OracleL2ChainProvider<T> {
    type Error = OracleProviderError;

    async fn l2_block_info_by_number(&mut self, number: u64) -> Result<L2BlockInfo, Self::Error> {
        // Get the block at the given number.
        let block = self.block_by_number(number).await?;

        // Construct the system config from the payload.
        L2BlockInfo::from_block_and_genesis(&block, &self.rollup_config.genesis)
            .map_err(OracleProviderError::BlockInfo)
    }

    async fn block_by_number(&mut self, number: u64) -> Result<OpBlock, Self::Error> {
        // Fetch the header for the given block number.
        let header = self.header_by_number(number).await?;
        let header_hash = header.hash_slow();
        self.block_from_header(header, header_hash).await
    }
}

impl<T: CommsClient + Send + Sync> OracleL2ChainProvider<T> {
    /// Hydrates a [`Header`] into a full [`OpBlock`] by walking its transactions trie.
    ///
    /// `header_hash` must be the hash of `header`; it is taken as an argument because callers that
    /// looked the header up by hash already have it.
    async fn block_from_header(
        &self,
        header: Header,
        header_hash: B256,
    ) -> Result<OpBlock, OracleProviderError> {
        let Header { transactions_root, timestamp, .. } = header;

        // Fetch the transactions in the block.
        HintType::L2Transactions
            .with_data(&[header_hash.as_ref()])
            .with_data(self.chain_id.map_or_else(Vec::new, |id| id.to_be_bytes().to_vec()))
            .send(self.oracle.as_ref())
            .await?;
        let trie_walker = OrderedListWalker::try_new_hydrated(transactions_root, self)
            .map_err(OracleProviderError::TrieWalker)?;

        // Decode the transactions within the transactions trie.
        let transactions = trie_walker
            .into_iter()
            .map(|(_, rlp)| {
                let res = OpTxEnvelope::decode_2718(&mut rlp.as_ref())?;
                Ok(res)
            })
            .collect::<Result<Vec<_>, _>>()
            .map_err(OracleProviderError::Rlp)?;

        let optimism_block = OpBlock {
            header,
            body: BlockBody {
                transactions,
                ommers: Vec::new(),
                withdrawals: self
                    .rollup_config
                    .is_canyon_active(timestamp)
                    .then(|| alloy_eips::eip4895::Withdrawals::new(Vec::new())),
            },
        };
        Ok(optimism_block)
    }
}

#[async_trait]
impl<T: CommsClient + Send + Sync> L2ChainProvider for OracleL2ChainProvider<T> {
    type Error = OracleProviderError;

    async fn system_config_by_l2_hash(
        &mut self,
        hash: B256,
        rollup_config: Arc<RollupConfig>,
    ) -> Result<SystemConfig, <Self as L2ChainProvider>::Error> {
        // A hash addresses the header directly, sparing the walk back from the safe head that a
        // lookup by number requires.
        let block = self.block_from_header(self.header_by_hash(hash)?, hash).await?;

        // Construct the system config from the payload.
        to_system_config(&block, rollup_config.as_ref())
            .map_err(OracleProviderError::OpBlockConversion)
    }
}

impl<T: CommsClient> TrieProvider for OracleL2ChainProvider<T> {
    type Error = OracleProviderError;

    fn trie_node_by_hash(&self, key: B256) -> Result<TrieNode, OracleProviderError> {
        // On L2, trie node preimages are stored as keccak preimage types in the oracle. We assume
        // that a hint for these preimages has already been sent, prior to this call.
        crate::block_on(async move {
            TrieNode::decode(
                &mut self
                    .oracle
                    .get(PreimageKey::new(*key, PreimageKeyType::Keccak256))
                    .await
                    .map_err(OracleProviderError::Preimage)?
                    .as_ref(),
            )
            .map_err(OracleProviderError::Rlp)
        })
    }
}

impl<T: CommsClient> TrieDBProvider for OracleL2ChainProvider<T> {
    fn bytecode_by_hash(&self, hash: B256) -> Result<Bytes, OracleProviderError> {
        // Fetch the bytecode preimage from the caching oracle.
        crate::block_on(async move {
            HintType::L2Code
                .with_data(&[hash.as_slice()])
                .with_data(self.chain_id.map_or_else(Vec::new, |id| id.to_be_bytes().to_vec()))
                .send(self.oracle.as_ref())
                .await?;
            self.oracle
                .get(PreimageKey::new_keccak256(*hash))
                .await
                .map(Into::into)
                .map_err(OracleProviderError::Preimage)
        })
    }

    fn header_by_hash(&self, hash: B256) -> Result<Header, OracleProviderError> {
        // Fetch the header from the caching oracle.
        crate::block_on(async move {
            HintType::L2BlockHeader
                .with_data(&[hash.as_slice()])
                .with_data(self.chain_id.map_or_else(Vec::new, |id| id.to_be_bytes().to_vec()))
                .send(self.oracle.as_ref())
                .await?;
            let header_bytes = self.oracle.get(PreimageKey::new_keccak256(*hash)).await?;

            Header::decode(&mut header_bytes.as_slice()).map_err(OracleProviderError::Rlp)
        })
    }
}

impl<T: CommsClient> TrieHinter for OracleL2ChainProvider<T> {
    type Error = OracleProviderError;

    fn hint_trie_node(&self, hash: B256) -> Result<(), Self::Error> {
        crate::block_on(async move {
            HintType::L2StateNode
                .with_data(&[hash.as_slice()])
                .with_data(self.chain_id.map_or_else(Vec::new, |id| id.to_be_bytes().to_vec()))
                .send(self.oracle.as_ref())
                .await
        })
    }

    fn hint_account_proof(&self, address: Address, block_hash: B256) -> Result<(), Self::Error> {
        crate::block_on(async move {
            HintType::L2AccountProof
                .with_data(&[block_hash.as_slice(), address.as_slice()])
                .with_data(self.chain_id.map_or_else(Vec::new, |id| id.to_be_bytes().to_vec()))
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
        crate::block_on(async move {
            HintType::L2AccountStorageProof
                .with_data(&[
                    block_hash.as_slice(),
                    address.as_slice(),
                    slot.to_be_bytes::<32>().as_ref(),
                ])
                .with_data(self.chain_id.map_or_else(Vec::new, |id| id.to_be_bytes().to_vec()))
                .send(self.oracle.as_ref())
                .await
        })
    }

    fn hint_execution_witness(
        &self,
        parent_hash: B256,
        op_payload_attributes: &op_alloy_rpc_types_engine::OpPayloadAttributes,
    ) -> Result<(), Self::Error> {
        crate::block_on(async move {
            let encoded_attributes =
                serde_json::to_vec(op_payload_attributes).map_err(OracleProviderError::Serde)?;

            HintType::L2PayloadWitness
                .with_data(&[parent_hash.as_slice(), &encoded_attributes])
                .with_data(self.chain_id.map_or_else(Vec::new, |id| id.to_be_bytes().to_vec()))
                .send(self.oracle.as_ref())
                .await
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rlp::Encodable;
    use core::sync::atomic::{AtomicUsize, Ordering};
    use kona_preimage::{HintWriterClient, PreimageOracleClient, errors::PreimageOracleResult};

    /// A minimal in-memory [`CommsClient`] used to drive [`OracleL2ChainProvider`] in tests.
    #[derive(Clone, Default)]
    struct MockOracle {
        preimages: Arc<BTreeMap<PreimageKey, Vec<u8>>>,
        get_calls: Arc<AtomicUsize>,
    }

    #[async_trait]
    impl PreimageOracleClient for MockOracle {
        async fn get(&self, key: PreimageKey) -> PreimageOracleResult<Vec<u8>> {
            self.get_calls.fetch_add(1, Ordering::SeqCst);
            Ok(self.preimages.get(&key).expect("missing preimage in mock").clone())
        }

        async fn get_exact(&self, key: PreimageKey, buf: &mut [u8]) -> PreimageOracleResult<()> {
            let v = self.get(key).await?;
            buf.copy_from_slice(&v);
            Ok(())
        }
    }

    #[async_trait]
    impl HintWriterClient for MockOracle {
        async fn write(&self, _hint: &str) -> PreimageOracleResult<()> {
            Ok(())
        }
    }

    /// Build a linear chain of `n` headers and return the headers (oldest first) plus a
    /// preimage map keyed by `Keccak256(header_hash)`.
    fn build_chain(n: u64) -> (Vec<Header>, BTreeMap<PreimageKey, Vec<u8>>) {
        let mut headers = Vec::with_capacity(n as usize);
        let mut parent_hash = B256::ZERO;
        for i in 0..n {
            let header =
                Header { number: i, parent_hash, timestamp: 1_000 + i, ..Default::default() };
            parent_hash = header.hash_slow();
            headers.push(header);
        }
        let mut preimages = BTreeMap::new();
        for h in &headers {
            let mut rlp = Vec::new();
            h.encode(&mut rlp);
            preimages.insert(PreimageKey::new(*h.hash_slow(), PreimageKeyType::Keccak256), rlp);
        }
        (headers, preimages)
    }

    /// Provider anchored at the last header of `headers`, with a pre-Isthmus rollup config so
    /// the linear parent-hash walk is exercised.
    fn provider(
        headers: &[Header],
        preimages: BTreeMap<PreimageKey, Vec<u8>>,
    ) -> (OracleL2ChainProvider<MockOracle>, Arc<AtomicUsize>) {
        let oracle =
            MockOracle { preimages: Arc::new(preimages), get_calls: Arc::new(AtomicUsize::new(0)) };
        let calls = oracle.get_calls.clone();
        let head = headers.last().unwrap().hash_slow();
        (
            OracleL2ChainProvider::new(head, Arc::new(RollupConfig::default()), Arc::new(oracle)),
            calls,
        )
    }

    // Note: tests run on a multi-threaded runtime because `header_by_hash` goes through
    // `crate::block_on`, whose std implementation uses `tokio::task::block_in_place`.

    #[tokio::test(flavor = "multi_thread")]
    async fn header_by_number_returns_correct_header() {
        let (headers, preimages) = build_chain(10);
        let (mut p, _) = provider(&headers, preimages);

        for target in 0..10 {
            let header = p.header_by_number(target).await.unwrap();
            assert_eq!(header.number, target);
            assert_eq!(header.hash_slow(), headers[target as usize].hash_slow());
        }
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn header_by_number_ascending_overlap_walk_is_linear() {
        // Pins the fault proof cost of span batch overlap validation, which looks up each
        // overlapped block in ascending order: without the by-number cache, each lookup
        // re-walks from the safe head, so five lookups against a five-deep overlap cost
        // 5 + 4 + 3 + 2 + 1 = 15 header preimage reads. With the cache, the first (lowest)
        // lookup walks the range once and the rest are hits: 5 reads total.
        let (headers, preimages) = build_chain(6);
        let (mut p, calls) = provider(&headers, preimages);

        for target in 1..=5 {
            let header = p.header_by_number(target).await.unwrap();
            assert_eq!(header.number, target);
        }
        assert_eq!(calls.load(Ordering::SeqCst), 5);

        // Repeating the sweep is free.
        for target in 1..=5 {
            p.header_by_number(target).await.unwrap();
        }
        assert_eq!(calls.load(Ordering::SeqCst), 5);
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn header_by_number_resumes_from_closest_cached_ancestor() {
        let (headers, preimages) = build_chain(20);
        let (mut p, calls) = provider(&headers, preimages);

        // Walk head (19) down to 15: 5 reads.
        p.header_by_number(15).await.unwrap();
        let after_first = calls.load(Ordering::SeqCst);
        assert_eq!(after_first, 5);

        // Walk further back to 10: resumes from cached 15, not from the head, so only 5
        // additional reads (14, 13, 12, 11, 10).
        p.header_by_number(10).await.unwrap();
        assert_eq!(calls.load(Ordering::SeqCst), after_first + 5);
    }

    #[tokio::test(flavor = "multi_thread")]
    async fn header_by_number_rejects_numbers_past_head() {
        let (headers, preimages) = build_chain(5);
        let (mut p, _) = provider(&headers, preimages);

        let err = p.header_by_number(99).await.unwrap_err();
        assert!(matches!(err, OracleProviderError::BlockNumberPastHead(99, 4)));
    }
}
