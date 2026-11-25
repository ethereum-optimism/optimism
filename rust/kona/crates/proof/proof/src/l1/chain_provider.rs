//! Contains the concrete implementation of the [`ChainProvider`] trait for the proof.

use crate::{HintType, errors::OracleProviderError};
use alloc::{boxed::Box, collections::BTreeMap, sync::Arc, vec::Vec};
use alloy_consensus::{Header, Receipt, ReceiptEnvelope, TxEnvelope};
use alloy_eips::eip2718::Decodable2718;
use alloy_primitives::B256;
use alloy_rlp::Decodable;
use async_trait::async_trait;
use kona_derive::ChainProvider;
use kona_mpt::{OrderedListWalker, TrieNode, TrieProvider};
use kona_preimage::{CommsClient, PreimageKey, PreimageKeyType};
use kona_protocol::BlockInfo;

/// The oracle-backed L1 chain provider for the client program.
#[derive(Debug, Clone)]
pub struct OracleL1ChainProvider<T: CommsClient> {
    /// The L1 head hash.
    pub l1_head: B256,
    /// The preimage oracle client.
    pub oracle: Arc<T>,
    /// Cache of [`BlockInfo`] by block number for ancestors of [`Self::l1_head`].
    ///
    /// The L1 chain seen by this provider is fixed and canonical (rooted at `l1_head`), so
    /// entries never need invalidation. Populated as a side effect of
    /// [`Self::block_info_by_number`] walking back from the head.
    block_info_by_number: BTreeMap<u64, BlockInfo>,
}

impl<T: CommsClient> OracleL1ChainProvider<T> {
    /// Creates a new [`OracleL1ChainProvider`] with the given boot information and oracle client.
    pub const fn new(l1_head: B256, oracle: Arc<T>) -> Self {
        Self { l1_head, oracle, block_info_by_number: BTreeMap::new() }
    }
}

#[async_trait]
impl<T: CommsClient + Sync + Send> ChainProvider for OracleL1ChainProvider<T> {
    type Error = OracleProviderError;

    async fn header_by_hash(&mut self, hash: B256) -> Result<Header, Self::Error> {
        // Fetch the header RLP from the oracle.
        HintType::L1BlockHeader.with_data(&[hash.as_ref()]).send(self.oracle.as_ref()).await?;
        let header_rlp = self.oracle.get(PreimageKey::new_keccak256(*hash)).await?;

        // Decode the header RLP into a Header.
        Header::decode(&mut header_rlp.as_slice()).map_err(OracleProviderError::Rlp)
    }

    async fn block_info_by_number(&mut self, block_number: u64) -> Result<BlockInfo, Self::Error> {
        if let Some(cached) = self.block_info_by_number.get(&block_number) {
            return Ok(*cached);
        }

        // Start from the closest cached ancestor whose number is greater than the target,
        // falling back to the L1 head when no usable cache entry exists.
        let cached_ancestor = block_number
            .checked_add(1)
            .and_then(|n| self.block_info_by_number.range(n..).next().map(|(_, info)| *info));
        let mut current = match cached_ancestor {
            Some(info) => info,
            None => {
                let header = self.header_by_hash(self.l1_head).await?;
                if block_number > header.number {
                    return Err(OracleProviderError::BlockNumberPastHead(
                        block_number,
                        header.number,
                    ));
                }
                let info = BlockInfo {
                    hash: self.l1_head,
                    number: header.number,
                    parent_hash: header.parent_hash,
                    timestamp: header.timestamp,
                };
                self.block_info_by_number.insert(info.number, info);
                info
            }
        };

        // Walk back the block headers to the desired block number, caching each visited block.
        while current.number > block_number {
            let parent_hash = current.parent_hash;
            let header = self.header_by_hash(parent_hash).await?;
            current = BlockInfo {
                hash: parent_hash,
                number: header.number,
                parent_hash: header.parent_hash,
                timestamp: header.timestamp,
            };
            self.block_info_by_number.insert(current.number, current);
        }

        Ok(current)
    }

    async fn receipts_by_hash(&mut self, hash: B256) -> Result<Vec<Receipt>, Self::Error> {
        // Fetch the block header to find the receipts root.
        let header = self.header_by_hash(hash).await?;

        // Send a hint for the block's receipts, and walk through the receipts trie in the header to
        // verify them.
        HintType::L1Receipts.with_data(&[hash.as_ref()]).send(self.oracle.as_ref()).await?;
        let trie_walker = OrderedListWalker::try_new_hydrated(header.receipts_root, self)
            .map_err(OracleProviderError::TrieWalker)?;

        // Decode the receipts within the receipts trie.
        let receipts = trie_walker
            .into_iter()
            .map(|(_, rlp)| {
                let envelope = ReceiptEnvelope::decode_2718(&mut rlp.as_ref())?;
                Ok(envelope.as_receipt().expect("Infallible").clone())
            })
            .collect::<Result<Vec<_>, _>>()
            .map_err(OracleProviderError::Rlp)?;

        Ok(receipts)
    }

    async fn block_info_and_transactions_by_hash(
        &mut self,
        hash: B256,
    ) -> Result<(BlockInfo, Vec<TxEnvelope>), Self::Error> {
        // Fetch the block header to construct the block info.
        let header = self.header_by_hash(hash).await?;
        let block_info = BlockInfo {
            hash,
            number: header.number,
            parent_hash: header.parent_hash,
            timestamp: header.timestamp,
        };

        // Send a hint for the block's transactions, and walk through the transactions trie in the
        // header to verify them.
        HintType::L1Transactions.with_data(&[hash.as_ref()]).send(self.oracle.as_ref()).await?;
        let trie_walker = OrderedListWalker::try_new_hydrated(header.transactions_root, self)
            .map_err(OracleProviderError::TrieWalker)?;

        // Decode the transactions within the transactions trie.
        let transactions = trie_walker
            .into_iter()
            .map(|(_, rlp)| {
                // note: not short-handed for error type coercion w/ `?`.
                let rlp = TxEnvelope::decode_2718(&mut rlp.as_ref())?;
                Ok(rlp)
            })
            .collect::<Result<Vec<_>, _>>()
            .map_err(OracleProviderError::Rlp)?;

        Ok((block_info, transactions))
    }
}

impl<T: CommsClient> TrieProvider for OracleL1ChainProvider<T> {
    type Error = OracleProviderError;

    fn trie_node_by_hash(&self, key: B256) -> Result<TrieNode, Self::Error> {
        // On L1, trie node preimages are stored as keccak preimage types in the oracle. We assume
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

#[cfg(test)]
mod tests {
    use super::*;
    use alloy_rlp::Encodable;
    use core::sync::atomic::{AtomicUsize, Ordering};
    use kona_preimage::{
        HintWriterClient, PreimageKey, PreimageKeyType, PreimageOracleClient,
        errors::PreimageOracleResult,
    };

    /// A minimal in-memory [`CommsClient`] used to drive [`OracleL1ChainProvider`] in tests.
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

    fn provider(
        headers: &[Header],
        preimages: BTreeMap<PreimageKey, Vec<u8>>,
    ) -> (OracleL1ChainProvider<MockOracle>, Arc<AtomicUsize>) {
        let oracle =
            MockOracle { preimages: Arc::new(preimages), get_calls: Arc::new(AtomicUsize::new(0)) };
        let calls = oracle.get_calls.clone();
        let head = headers.last().unwrap().hash_slow();
        (OracleL1ChainProvider::new(head, Arc::new(oracle)), calls)
    }

    #[tokio::test]
    async fn block_info_by_number_returns_correct_block() {
        let (headers, preimages) = build_chain(10);
        let (mut p, _) = provider(&headers, preimages);

        for target in 0..10 {
            let info = p.block_info_by_number(target).await.unwrap();
            assert_eq!(info.number, target);
            assert_eq!(info.hash, headers[target as usize].hash_slow());
            assert_eq!(info.timestamp, 1_000 + target);
        }
    }

    #[tokio::test]
    async fn block_info_by_number_caches_repeat_lookups() {
        let (headers, preimages) = build_chain(10);
        let (mut p, calls) = provider(&headers, preimages);

        // First lookup walks from head (number 9) back to 0: 10 fetches.
        p.block_info_by_number(0).await.unwrap();
        let after_first = calls.load(Ordering::SeqCst);
        assert_eq!(after_first, 10);

        // Repeating any lookup we have already walked through hits the cache for free.
        for target in 0..10 {
            p.block_info_by_number(target).await.unwrap();
        }
        assert_eq!(calls.load(Ordering::SeqCst), after_first);
    }

    #[tokio::test]
    async fn block_info_by_number_resumes_from_closest_cached_ancestor() {
        let (headers, preimages) = build_chain(20);
        let (mut p, calls) = provider(&headers, preimages);

        // Walk head (19) down to 15: 5 fetches.
        p.block_info_by_number(15).await.unwrap();
        let after_first = calls.load(Ordering::SeqCst);
        assert_eq!(after_first, 5);

        // Walk further back to 10. Should resume from cached block 15, not from the head, so
        // only 5 additional fetches (15 -> 14 -> 13 -> 12 -> 11 -> 10).
        p.block_info_by_number(10).await.unwrap();
        assert_eq!(calls.load(Ordering::SeqCst), after_first + 5);
    }

    #[tokio::test]
    async fn block_info_by_number_rejects_numbers_past_head() {
        let (headers, preimages) = build_chain(5);
        let (mut p, _) = provider(&headers, preimages);

        let err = p.block_info_by_number(99).await.unwrap_err();
        assert!(matches!(err, OracleProviderError::BlockNumberPastHead(99, 4)));
    }

    #[tokio::test]
    async fn block_info_by_number_handles_u64_max_without_overflow() {
        let (headers, preimages) = build_chain(5);
        let (mut p, _) = provider(&headers, preimages);

        let err = p.block_info_by_number(u64::MAX).await.unwrap_err();
        assert!(matches!(err, OracleProviderError::BlockNumberPastHead(n, 4) if n == u64::MAX));
    }
}
