//! This module contains the [`OrderedListWalker`] struct, which allows for traversing an MPT root
//! of a derivable ordered list.

use crate::{
    TrieNode, TrieNodeError, TrieProvider,
    errors::{OrderedListWalkerError, OrderedListWalkerResult},
};
use alloc::{collections::VecDeque, string::ToString, vec};
use alloy_primitives::{B256, Bytes};
use alloy_rlp::EMPTY_STRING_CODE;
use core::marker::PhantomData;

/// A [`OrderedListWalker`] allows for traversing over a Merkle Patricia Trie containing a derivable
/// ordered list.
///
/// Once it has been hydrated with [`Self::hydrate`], the elements in the derivable list can be
/// iterated over using the [Iterator] implementation.
#[derive(Debug, Clone, Eq, PartialEq)]
pub struct OrderedListWalker<F: TrieProvider> {
    /// The Merkle Patricia Trie root.
    root: B256,
    /// The leaf nodes of the derived list, in order. [None] if the tree has yet to be fully
    /// traversed with [`Self::hydrate`].
    inner: Option<VecDeque<(Bytes, Bytes)>>,
    /// Phantom data
    _phantom: PhantomData<F>,
}

impl<F> OrderedListWalker<F>
where
    F: TrieProvider,
{
    /// Creates a new [`OrderedListWalker`], yet to be hydrated.
    pub const fn new(root: B256) -> Self {
        Self { root, inner: None, _phantom: PhantomData }
    }

    /// Creates a new [`OrderedListWalker`] and hydrates it with [`Self::hydrate`] and the given
    /// fetcher immediately.
    pub fn try_new_hydrated(root: B256, fetcher: &F) -> OrderedListWalkerResult<Self> {
        let mut walker = Self { root, inner: None, _phantom: PhantomData };
        walker.hydrate(fetcher)?;
        Ok(walker)
    }

    /// Hydrates the [`OrderedListWalker`]'s iterator with the leaves of the derivable list. If
    /// `Self::inner` is [Some], this function will fail fast.
    pub fn hydrate(&mut self, fetcher: &F) -> OrderedListWalkerResult<()> {
        // Do not allow for re-hydration if `inner` is `Some` and still contains elements.
        if self.inner.is_some() && self.inner.as_ref().map(|s| s.len()).unwrap_or_default() > 0 {
            return Err(OrderedListWalkerError::AlreadyHydrated);
        }

        // Get the preimage to the root node.
        let root_trie_node = Self::get_trie_node(self.root, fetcher)?;

        // With small lists the iterator seems to use 0x80 (RLP empty string, unlike the others)
        // as key for item 0, causing it to come last. We need to account for this, pulling the
        // first element into its proper position.
        let mut ordered_list = Self::fetch_leaves(&root_trie_node, fetcher)?;
        if !ordered_list.is_empty() {
            if ordered_list.len() <= EMPTY_STRING_CODE as usize {
                // If the list length is < 0x80, the final element is the first element.
                let first = ordered_list.pop_back().expect("Cannot be empty");
                ordered_list.push_front(first);
            } else {
                // If the list length is > 0x80, the element at index 0x80-1 is the first element.
                let first =
                    ordered_list.remove((EMPTY_STRING_CODE - 1) as usize).expect("Cannot be empty");
                ordered_list.push_front(first);
            }
        }

        self.inner = Some(ordered_list);
        Ok(())
    }

    /// Takes the inner list of the [`OrderedListWalker`], returning it and setting the inner list
    /// to [None].
    pub const fn take_inner(&mut self) -> Option<VecDeque<(Bytes, Bytes)>> {
        self.inner.take()
    }

    /// Traverses a [`TrieNode`], returning all values of child [`TrieNode::Leaf`] variants.
    fn fetch_leaves(
        trie_node: &TrieNode,
        fetcher: &F,
    ) -> OrderedListWalkerResult<VecDeque<(Bytes, Bytes)>> {
        match trie_node {
            TrieNode::Branch { stack } => {
                let mut leaf_values = VecDeque::with_capacity(stack.len());
                for item in stack {
                    match item {
                        TrieNode::Blinded { commitment } => {
                            // If the string is a hash, we need to grab the preimage for it and
                            // continue recursing.
                            let trie_node = Self::get_trie_node(commitment.as_ref(), fetcher)?;
                            leaf_values.append(&mut Self::fetch_leaves(&trie_node, fetcher)?);
                        }
                        TrieNode::Empty => { /* Skip over empty nodes, we're looking for values. */
                        }
                        item => {
                            // If the item is already retrieved, recurse on it.
                            leaf_values.append(&mut Self::fetch_leaves(item, fetcher)?);
                        }
                    }
                }
                Ok(leaf_values)
            }
            TrieNode::Leaf { prefix, value } => {
                Ok(vec![(prefix.to_vec().into(), value.clone())].into())
            }
            TrieNode::Extension { node, .. } => {
                // If the node is a hash, we need to grab the preimage for it and continue
                // recursing. If it is already retrieved, recurse on it.
                match node.as_ref() {
                    TrieNode::Blinded { commitment } => {
                        let trie_node = Self::get_trie_node(commitment.as_ref(), fetcher)?;
                        Ok(Self::fetch_leaves(&trie_node, fetcher)?)
                    }
                    node => Ok(Self::fetch_leaves(node, fetcher)?),
                }
            }
            TrieNode::Empty => Ok(VecDeque::new()),
            _ => Err(TrieNodeError::InvalidNodeType.into()),
        }
    }

    /// Grabs the preimage of `hash` using `fetcher`, and attempts to decode the preimage data into
    /// a [`TrieNode`]. Will error if the conversion of `T` into [B256] fails.
    fn get_trie_node<T>(hash: T, fetcher: &F) -> OrderedListWalkerResult<TrieNode>
    where
        T: Into<B256>,
    {
        fetcher
            .trie_node_by_hash(hash.into())
            .map_err(|e| TrieNodeError::Provider(e.to_string()))
            .map_err(Into::into)
    }
}

impl<F> Iterator for OrderedListWalker<F>
where
    F: TrieProvider,
{
    type Item = (Bytes, Bytes);

    fn next(&mut self) -> Option<Self::Item> {
        match self.inner {
            Some(ref mut leaves) => {
                let item = leaves.pop_front();
                if leaves.is_empty() {
                    self.inner = None;
                }
                item
            }
            _ => None,
        }
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::{
        NoopTrieProvider, ordered_trie_with_encoder,
        test_util::{
            TrieNodeProvider, get_live_derivable_receipts_list,
            get_live_derivable_transactions_list,
        },
    };
    use alloc::{collections::BTreeMap, string::String, vec::Vec};
    use alloy_consensus::{ReceiptEnvelope, TxEnvelope};
    use alloy_primitives::keccak256;
    use alloy_provider::network::eip2718::Decodable2718;
    use alloy_rlp::{Decodable, Encodable};

    #[tokio::test]
    async fn test_online_list_walker_receipts() {
        let (root, preimages, envelopes) = get_live_derivable_receipts_list().await.unwrap();
        let fetcher = TrieNodeProvider::new(preimages);
        let list = OrderedListWalker::try_new_hydrated(root, &fetcher).unwrap();

        assert_eq!(
            list.into_iter()
                .map(|(_, rlp)| ReceiptEnvelope::decode_2718(&mut rlp.as_ref()).unwrap())
                .collect::<Vec<_>>(),
            envelopes
        );
    }

    #[tokio::test]
    async fn test_online_list_walker_transactions() {
        let (root, preimages, envelopes) = get_live_derivable_transactions_list().await.unwrap();
        let fetcher = TrieNodeProvider::new(preimages);
        let list = OrderedListWalker::try_new_hydrated(root, &fetcher).unwrap();

        assert_eq!(
            list.into_iter()
                .map(|(_, rlp)| TxEnvelope::decode(&mut rlp.as_ref()).unwrap())
                .collect::<Vec<_>>(),
            envelopes
        );
    }

    #[test]
    fn test_list_walker() {
        const VALUES: [&str; 3] = ["test one", "test two", "test three"];

        let mut trie = ordered_trie_with_encoder(&VALUES, |v, buf| v.encode(buf));
        let root = trie.root();

        let preimages = trie.take_proof_nodes().into_inner().into_iter().fold(
            BTreeMap::default(),
            |mut acc, (_, value)| {
                acc.insert(keccak256(value.as_ref()), value);
                acc
            },
        );

        let fetcher = TrieNodeProvider::new(preimages);
        let list = OrderedListWalker::try_new_hydrated(root, &fetcher).unwrap();

        assert_eq!(
            list.inner
                .unwrap()
                .iter()
                .map(|(_, v)| { String::decode(&mut v.as_ref()).unwrap() })
                .collect::<Vec<_>>(),
            VALUES
        );
    }

    #[test]
    fn test_empty_list_walker() {
        assert!(
            OrderedListWalker::fetch_leaves(&TrieNode::Empty, &NoopTrieProvider)
                .expect("Failed to traverse empty trie")
                .is_empty()
        );
    }
}
