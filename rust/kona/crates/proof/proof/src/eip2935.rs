//! EIP-2935 history lookup utilities.

use crate::errors::OracleProviderError;
use alloc::string::ToString;
use alloy_consensus::Header;
use alloy_eips::eip2935::HISTORY_STORAGE_ADDRESS;
use alloy_primitives::{B256, U256, b256, keccak256};
use alloy_rlp::Decodable;
use alloy_trie::TrieAccount;
use kona_mpt::{Nibbles, TrieHinter, TrieNode, TrieNodeError, TrieProvider};
use kona_preimage::errors::PreimageOracleError;

/// The [`keccak256`] hash of the address of the EIP-2935 history storage contract.
const HASHED_HISTORY_STORAGE_ADDRESS: B256 =
    b256!("6c9d57be05dd69371c4dd2e871bce6e9f4124236825bb612ee18a45e5675be51");

/// The number of blocks that the EIP-2935 contract serves historical block hashes for. (8192 - 1)
const HISTORY_SERVE_WINDOW: u64 = 2u64.pow(13) - 1;

/// Performs a historical block hash lookup using the EIP-2935 contract. If the block number is out
/// of bounds of the history lookup window size, the oldest block hash within the window is
/// returned.
pub async fn eip_2935_history_lookup<P, H>(
    header: &Header,
    block_number: u64,
    provider: &P,
    hinter: &H,
) -> Result<B256, OracleProviderError>
where
    P: TrieProvider,
    H: TrieHinter,
{
    // Compute the storage slot for the block hash. If the distance between the current header and
    // the desired block number is within the window size, we compute the slot based on the
    // target block number, as the result is present within the ring. Otherwise, we return the
    // oldest block in the window.
    let slot = if header.number.saturating_sub(block_number) <= HISTORY_SERVE_WINDOW {
        block_number
    } else {
        header.number
    } % HISTORY_SERVE_WINDOW;

    // Send a hint to fetch the storage slot proof prior to traversing the state / account tries.
    hinter
        .hint_storage_proof(HISTORY_STORAGE_ADDRESS, U256::from(slot), header.number)
        .map_err(|e| PreimageOracleError::Other(e.to_string()))?;

    // Fetch the trie account for the history accumulator.
    let mut state_trie = TrieNode::new_blinded(header.state_root);
    let account_key = Nibbles::unpack(HASHED_HISTORY_STORAGE_ADDRESS);
    let raw_account = state_trie.open(&account_key, provider)?.ok_or(TrieNodeError::KeyNotFound)?;
    let account =
        TrieAccount::decode(&mut raw_account.as_ref()).map_err(OracleProviderError::Rlp)?;

    // Fetch the storage slot value from the account.
    let mut storage_trie = TrieNode::new_blinded(account.storage_root);
    let slot_key = Nibbles::unpack(keccak256(U256::from(slot).to_be_bytes::<32>()));
    let slot_value = storage_trie.open(&slot_key, provider)?.ok_or(TrieNodeError::KeyNotFound)?;

    B256::decode(&mut slot_value.as_ref()).map_err(OracleProviderError::Rlp)
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::{vec, vec::Vec};
    use alloy_primitives::Bytes;
    use alloy_rlp::Encodable;
    use alloy_trie::{HashBuilder, proof::ProofRetainer};
    use kona_mpt::NoopTrieHinter;
    use kona_registry::HashMap;
    use rstest::rstest;

    // Mock TrieProvider implementation for testing EIP-2935 history lookup
    #[derive(Default, Clone)]
    struct MockTrieProvider {
        pub(crate) state_root: B256,
        pub(crate) nodes: HashMap<B256, Bytes>,
    }

    impl MockTrieProvider {
        pub(crate) fn new(ring_index: u64, block_hash: B256) -> Self {
            let (storage_root, storage_proof) = {
                let slot_key =
                    Nibbles::unpack(keccak256(U256::from(ring_index).to_be_bytes::<32>()));
                let mut storage_hb =
                    HashBuilder::default().with_proof_retainer(ProofRetainer::new(vec![slot_key]));

                let mut encoded = Vec::with_capacity(block_hash.length());
                block_hash.encode(&mut encoded);
                storage_hb.add_leaf(slot_key, encoded.as_slice());

                let storage_root = storage_hb.root();
                let storage_proof = storage_hb.take_proof_nodes();
                (storage_root, storage_proof)
            };

            let (state_root, state_proof) = {
                let account_key = Nibbles::unpack(HASHED_HISTORY_STORAGE_ADDRESS);
                let mut state_hb = HashBuilder::default()
                    .with_proof_retainer(ProofRetainer::new(vec![account_key]));

                let account =
                    TrieAccount { storage_root, code_hash: keccak256(""), ..Default::default() };
                let mut encoded = Vec::with_capacity(account.length());
                account.encode(&mut encoded);
                state_hb.add_leaf(account_key, encoded.as_slice());

                let state_root = state_hb.root();
                let state_proof = state_hb.take_proof_nodes();
                (state_root, state_proof)
            };

            let nodes = storage_proof
                .values()
                .chain(state_proof.values())
                .cloned()
                .map(|v| (keccak256(v.as_ref()), v))
                .collect::<HashMap<_, _>>();

            Self { state_root, nodes }
        }
    }

    impl TrieProvider for MockTrieProvider {
        type Error = OracleProviderError;

        fn trie_node_by_hash(&self, hash: B256) -> Result<TrieNode, Self::Error> {
            self.nodes
                .get(&hash)
                .cloned()
                .map(|bytes| TrieNode::decode(&mut bytes.as_ref()).expect("valid node"))
                .ok_or(OracleProviderError::TrieNode(TrieNodeError::KeyNotFound))
        }
    }

    #[rstest]
    #[case::block_number_in_window(1000, 999)]
    #[case::block_number_outside_window(9000, 100)]
    #[case::block_number_at_window_boundary(8192 * 2, 8192)]
    #[case::block_number_at_window_boundary_plus_one(8192 * 2, 8191)]
    #[tokio::test]
    async fn test_eip_2935_history_lookup(
        #[case] head_block_number: u64,
        #[case] target_block_number: u64,
    ) {
        let expected_hash = B256::from([0xFF; 32]);
        let ring_index = if head_block_number - target_block_number <= HISTORY_SERVE_WINDOW {
            target_block_number
        } else {
            head_block_number
        } % HISTORY_SERVE_WINDOW;

        let provider = MockTrieProvider::new(ring_index, expected_hash);
        let header = Header {
            number: head_block_number,
            state_root: provider.state_root,
            ..Default::default()
        };

        let result =
            eip_2935_history_lookup(&header, target_block_number, &provider, &NoopTrieHinter)
                .await
                .unwrap();

        assert_eq!(result, expected_hash);
    }
}
