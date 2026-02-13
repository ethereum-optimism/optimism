//! This module contains the [`TrieNode`] type, which represents a node within a standard Merkle
//! Patricia Trie.

use crate::{
    TrieHinter, TrieNodeError, TrieProvider,
    errors::TrieNodeResult,
    util::{rlp_list_element_length, unpack_path_to_nibbles},
};
use alloc::{boxed::Box, string::ToString, vec, vec::Vec};
use alloy_primitives::{B256, Bytes, keccak256};
use alloy_rlp::{Buf, Decodable, EMPTY_STRING_CODE, Encodable, Header, length_of_length};
use alloy_trie::{EMPTY_ROOT_HASH, Nibbles};

/// The length of the branch list when RLP encoded
const BRANCH_LIST_LENGTH: usize = 17;

/// The length of a leaf or extension node's RLP encoded list
const LEAF_OR_EXTENSION_LIST_LENGTH: usize = 2;

/// The number of nibbles traversed in a branch node.
const BRANCH_NODE_NIBBLES: usize = 1;

/// Prefix for even-nibbled extension node paths.
const PREFIX_EXTENSION_EVEN: u8 = 0;

/// Prefix for odd-nibbled extension node paths.
const PREFIX_EXTENSION_ODD: u8 = 1;

/// Prefix for even-nibbled leaf node paths.
const PREFIX_LEAF_EVEN: u8 = 2;

/// Prefix for odd-nibbled leaf node paths.
const PREFIX_LEAF_ODD: u8 = 3;

/// Nibble bit width.
const NIBBLE_WIDTH: usize = 4;

/// A [`TrieNode`] is a node within a standard Ethereum Merkle Patricia Trie. In this
/// implementation, keys are expected to be fixed-size nibble sequences, and values are arbitrary
/// byte sequences.
///
/// The [`TrieNode`] has several variants:
/// - [`TrieNode::Empty`] represents an empty node.
/// - [`TrieNode::Blinded`] represents a node that has been blinded by a commitment.
/// - [`TrieNode::Leaf`] represents a 2-item node with the encoding `rlp([encoded_path, value])`.
/// - [`TrieNode::Extension`] represents a 2-item pointer node with the encoding `rlp([encoded_path,
///   key])`.
/// - [`TrieNode::Branch`] represents a node that refers to up to 16 child nodes with the encoding
///   `rlp([ v0, ..., v15, value ])`.
///
/// In the Ethereum Merkle Patricia Trie, nodes longer than an encoded 32 byte string (33 total
/// bytes) are blinded with [keccak256] hashes. When a node is "opened", it is replaced with the
/// [`TrieNode`] that is decoded from to the preimage of the hash.
///
/// The [`alloy_rlp::Encodable`] and [`alloy_rlp::Decodable`] traits are implemented for
/// [`TrieNode`], allowing for RLP encoding and decoding of the types for storage and retrieval. The
/// implementation of these traits will implicitly blind nodes that are longer than 32 bytes in
/// length when encoding. When decoding, the implementation will leave blinded nodes in place.
///
/// ## SAFETY
/// As this implementation only supports uniform key sizes, the [`TrieNode`] data structure will
/// fail to behave correctly if confronted with keys of varying lengths. Namely, this is because it
/// does not support the `value` field in branch nodes, just like the Ethereum Merkle Patricia Trie.
#[derive(Debug, Clone, Eq, PartialEq)]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
pub enum TrieNode {
    /// An empty [`TrieNode`] is represented as an [`EMPTY_STRING_CODE`] (0x80).
    Empty,
    /// A blinded node is a node that has been blinded by a [keccak256] commitment.
    Blinded {
        /// The commitment that blinds the node.
        commitment: B256,
    },
    /// A leaf node is a 2-item node with the encoding `rlp([encoded_path, value])`
    Leaf {
        /// The key of the leaf node
        prefix: Nibbles,
        /// The value of the leaf node
        value: Bytes,
    },
    /// An extension node is a 2-item pointer node with the encoding `rlp([encoded_path, key])`
    Extension {
        /// The path prefix of the extension
        prefix: Nibbles,
        /// The pointer to the child node
        node: Box<Self>,
    },
    /// A branch node refers to up to 16 child nodes with the encoding
    /// `rlp([ v0, ..., v15, value ])`
    Branch {
        /// The 16 child nodes and value of the branch.
        stack: Vec<Self>,
    },
}

impl TrieNode {
    /// Creates a new [`TrieNode::Blinded`] node.
    ///
    /// ## Takes
    /// - `commitment` - The commitment that blinds the node
    ///
    /// ## Returns
    /// - `Self` - The new blinded [`TrieNode`].
    pub const fn new_blinded(commitment: B256) -> Self {
        Self::Blinded { commitment }
    }

    /// Blinds the [`TrieNode`].. Alternatively, if the [`TrieNode`] is a [`TrieNode::Blinded`] node
    /// already, its commitment is returned directly.
    pub fn blind(&self) -> B256 {
        match self {
            Self::Blinded { commitment } => *commitment,
            Self::Empty => EMPTY_ROOT_HASH,
            _ => {
                let mut rlp_buf = Vec::with_capacity(self.length());
                self.encode(&mut rlp_buf);
                keccak256(rlp_buf)
            }
        }
    }

    /// Unblinds the [`TrieNode`] if it is a [`TrieNode::Blinded`] node.
    pub fn unblind<F: TrieProvider>(&mut self, fetcher: &F) -> TrieNodeResult<()> {
        if let Self::Blinded { commitment } = self {
            if *commitment == EMPTY_ROOT_HASH {
                // If the commitment is the empty root hash, the node is empty, and we don't need to
                // reach out to the fetcher.
                *self = Self::Empty;
            } else {
                *self = fetcher
                    .trie_node_by_hash(*commitment)
                    .map_err(|e| TrieNodeError::Provider(e.to_string()))?;
            }
        }
        Ok(())
    }

    /// Walks down the trie to a leaf value with the given key, if it exists. Preimages for blinded
    /// nodes along the path are fetched using the `fetcher` function, and persisted in the inner
    /// [`TrieNode`] elements.
    ///
    /// ## Takes
    /// - `self` - The root trie node
    /// - `path` - The nibbles representation of the path to the leaf node
    /// - `fetcher` - The preimage fetcher for intermediate blinded nodes
    ///
    /// ## Returns
    /// - `Err(_)` - Could not retrieve the node with the given key from the trie.
    /// - `Ok(None)` - The node with the given key does not exist in the trie.
    /// - `Ok(Some(_))` - The value of the node
    pub fn open<'a, F: TrieProvider>(
        &'a mut self,
        path: &Nibbles,
        fetcher: &F,
    ) -> TrieNodeResult<Option<&'a mut Bytes>> {
        match self {
            Self::Branch { stack } => {
                let branch_nibble = path.get(0).ok_or(TrieNodeError::PathTooShort)? as usize;
                stack
                    .get_mut(branch_nibble)
                    .map(|node| node.open(&path.slice(BRANCH_NODE_NIBBLES..), fetcher))
                    .unwrap_or(Ok(None))
            }
            Self::Leaf { prefix, value } => Ok((path == prefix).then_some(value)),
            Self::Extension { prefix, node } => {
                if path.slice(..prefix.len()) == *prefix {
                    // Follow extension branch
                    node.unblind(fetcher)?;
                    node.open(&path.slice(prefix.len()..), fetcher)
                } else {
                    Ok(None)
                }
            }
            Self::Blinded { .. } => {
                self.unblind(fetcher)?;
                self.open(path, fetcher)
            }
            Self::Empty => Ok(None),
        }
    }

    /// Inserts a [`TrieNode`] at the given path into the trie rooted at Self.
    ///
    /// ## Takes
    /// - `self` - The root trie node
    /// - `path` - The nibbles representation of the path to the leaf node
    /// - `node` - The node to insert at the given path
    /// - `fetcher` - The preimage fetcher for intermediate blinded nodes
    ///
    /// ## Returns
    /// - `Err(_)` - Could not insert the node at the given path in the trie.
    /// - `Ok(())` - The node was successfully inserted at the given path.
    pub fn insert<F: TrieProvider>(
        &mut self,
        path: &Nibbles,
        value: Bytes,
        fetcher: &F,
    ) -> TrieNodeResult<()> {
        match self {
            Self::Empty => {
                // If the trie node is null, insert the leaf node at the current path.
                *self = Self::Leaf { prefix: *path, value };
                Ok(())
            }
            Self::Leaf { prefix, value: leaf_value } => {
                let shared_extension_nibbles = path.common_prefix_length(prefix);

                // If all nibbles are shared, update the leaf node with the new value.
                if path == prefix {
                    *self = Self::Leaf { prefix: *prefix, value };
                    return Ok(());
                }

                // Create a branch node stack containing the leaf node and the new value.
                let mut stack = vec![Self::Empty; BRANCH_LIST_LENGTH];

                // Insert the shortened extension into the branch stack.
                let extension_nibble =
                    prefix.get(shared_extension_nibbles).ok_or(TrieNodeError::PathTooShort)?
                        as usize;
                stack[extension_nibble] = Self::Leaf {
                    prefix: prefix.slice(shared_extension_nibbles + BRANCH_NODE_NIBBLES..),
                    value: leaf_value.clone(),
                };

                // Insert the new value into the branch stack.
                let branch_nibble_new =
                    path.get(shared_extension_nibbles).ok_or(TrieNodeError::PathTooShort)? as usize;
                stack[branch_nibble_new] = Self::Leaf {
                    prefix: path.slice(shared_extension_nibbles + BRANCH_NODE_NIBBLES..),
                    value,
                };

                // Replace the leaf node with the branch if no nibbles are shared, else create an
                // extension.
                if shared_extension_nibbles == 0 {
                    *self = Self::Branch { stack };
                } else {
                    let raw_ext_nibbles = path.slice(..shared_extension_nibbles);
                    *self = Self::Extension {
                        prefix: raw_ext_nibbles,
                        node: Box::new(Self::Branch { stack }),
                    };
                }
                Ok(())
            }
            Self::Extension { prefix, node } => {
                let shared_extension_nibbles = path.common_prefix_length(prefix);
                if shared_extension_nibbles == prefix.len() {
                    node.insert(&path.slice(shared_extension_nibbles..), value, fetcher)?;
                    return Ok(());
                }

                // Create a branch node stack containing the leaf node and the new value.
                let mut stack = vec![Self::Empty; BRANCH_LIST_LENGTH];

                // Insert the shortened extension into the branch stack.
                let extension_nibble =
                    prefix.get(shared_extension_nibbles).ok_or(TrieNodeError::PathTooShort)?
                        as usize;
                let new_prefix = prefix.slice(shared_extension_nibbles + BRANCH_NODE_NIBBLES..);
                stack[extension_nibble] = if new_prefix.is_empty() {
                    // In the case that the extension node no longer has a prefix, insert the node
                    // verbatim into the branch.
                    node.as_ref().clone()
                } else {
                    Self::Extension { prefix: new_prefix, node: node.clone() }
                };

                // Insert the new value into the branch stack.
                let branch_nibble_new =
                    path.get(shared_extension_nibbles).ok_or(TrieNodeError::PathTooShort)? as usize;
                stack[branch_nibble_new] = Self::Leaf {
                    prefix: path.slice(shared_extension_nibbles + BRANCH_NODE_NIBBLES..),
                    value,
                };

                // Replace the extension node with the branch if no nibbles are shared, else create
                // an extension.
                if shared_extension_nibbles == 0 {
                    *self = Self::Branch { stack };
                } else {
                    let extension = path.slice(..shared_extension_nibbles);
                    *self = Self::Extension {
                        prefix: extension,
                        node: Box::new(Self::Branch { stack }),
                    };
                }
                Ok(())
            }
            Self::Branch { stack } => {
                // Follow the branch node to the next node in the path.
                let branch_nibble = path.get(0).ok_or(TrieNodeError::PathTooShort)? as usize;
                stack[branch_nibble].insert(&path.slice(BRANCH_NODE_NIBBLES..), value, fetcher)
            }
            Self::Blinded { .. } => {
                // If a blinded node is approached, reveal the node and continue the insertion
                // recursion.
                self.unblind(fetcher)?;
                self.insert(path, value, fetcher)
            }
        }
    }

    /// Deletes a node in the trie at the given path.
    ///
    /// ## Takes
    /// - `self` - The root trie node
    /// - `path` - The nibbles representation of the path to the leaf node
    ///
    /// ## Returns
    /// - `Err(_)` - Could not delete the node at the given path in the trie.
    /// - `Ok(())` - The node was successfully deleted at the given path.
    pub fn delete<F: TrieProvider, H: TrieHinter>(
        &mut self,
        path: &Nibbles,
        fetcher: &F,
        hinter: &H,
    ) -> TrieNodeResult<()> {
        match self {
            Self::Empty => Err(TrieNodeError::KeyNotFound),
            Self::Leaf { prefix, .. } => {
                if path == prefix {
                    *self = Self::Empty;
                    Ok(())
                } else {
                    Err(TrieNodeError::KeyNotFound)
                }
            }
            Self::Extension { prefix, node } => {
                let shared_nibbles = path.common_prefix_length(prefix);
                if shared_nibbles < prefix.len() {
                    return Err(TrieNodeError::KeyNotFound);
                } else if shared_nibbles == path.len() {
                    *self = Self::Empty;
                    return Ok(());
                }

                node.delete(&path.slice(prefix.len()..), fetcher, hinter)?;

                // Simplify extension if possible after the deletion
                self.collapse_if_possible(fetcher, hinter)
            }
            Self::Branch { stack } => {
                let branch_nibble = path.get(0).ok_or(TrieNodeError::PathTooShort)? as usize;
                stack[branch_nibble].delete(&path.slice(BRANCH_NODE_NIBBLES..), fetcher, hinter)?;

                // Simplify the branch if possible after the deletion
                self.collapse_if_possible(fetcher, hinter)
            }
            Self::Blinded { .. } => {
                self.unblind(fetcher)?;
                self.delete(path, fetcher, hinter)
            }
        }
    }

    /// If applicable, collapses `self` into a more compact form.
    ///
    /// ## Takes
    /// - `self` - The root trie node
    ///
    /// ## Returns
    /// - `Ok(())` - The node was successfully collapsed
    /// - `Err(_)` - Could not collapse the node
    fn collapse_if_possible<F: TrieProvider, H: TrieHinter>(
        &mut self,
        fetcher: &F,
        hinter: &H,
    ) -> TrieNodeResult<()> {
        match self {
            Self::Extension { prefix, node } => match node.as_mut() {
                Self::Extension { prefix: child_prefix, node: child_node } => {
                    // Double extensions are collapsed into a single extension.
                    let new_prefix = Nibbles::from_nibbles_unchecked(
                        [prefix.to_vec(), child_prefix.to_vec()].concat(),
                    );
                    *self = Self::Extension { prefix: new_prefix, node: child_node.clone() };
                }
                Self::Leaf { prefix: child_prefix, value: child_value } => {
                    // If the child node is a leaf, convert the extension into a leaf with the full
                    // path.
                    let new_prefix = Nibbles::from_nibbles_unchecked(
                        [prefix.to_vec(), child_prefix.to_vec()].concat(),
                    );
                    *self = Self::Leaf { prefix: new_prefix, value: child_value.clone() };
                }
                Self::Empty => {
                    // If the child node is empty, convert the extension into an empty node.
                    *self = Self::Empty;
                }
                _ => {
                    // If the child is a (blinded?) branch then no need for collapse
                    // because deletion did not collapse the (blinded?) branch
                }
            },
            Self::Branch { stack } => {
                // Count non-empty children
                let mut non_empty_children = stack
                    .iter_mut()
                    .enumerate()
                    .filter(|(_, node)| !matches!(node, Self::Empty))
                    .collect::<Vec<_>>();

                if non_empty_children.len() == 1 {
                    let (index, non_empty_node) = &mut non_empty_children[0];

                    // If only one non-empty child and no value, convert to extension or leaf
                    match non_empty_node {
                        Self::Leaf { prefix, value } => {
                            let new_prefix = Nibbles::from_nibbles_unchecked(
                                [&[*index as u8], prefix.to_vec().as_slice()].concat(),
                            );
                            *self = Self::Leaf { prefix: new_prefix, value: value.clone() };
                        }
                        Self::Extension { prefix, node } => {
                            let new_prefix = Nibbles::from_nibbles_unchecked(
                                [&[*index as u8], prefix.to_vec().as_slice()].concat(),
                            );
                            *self = Self::Extension { prefix: new_prefix, node: node.clone() };
                        }
                        Self::Branch { .. } => {
                            *self = Self::Extension {
                                prefix: Nibbles::from_nibbles_unchecked([*index as u8]),
                                node: Box::new(non_empty_node.clone()),
                            };
                        }
                        Self::Blinded { commitment } => {
                            // In this special case, we need to send a hint to fetch the preimage of
                            // the blinded node, since it is outside of the paths that have been
                            // traversed so far.
                            hinter
                                .hint_trie_node(*commitment)
                                .map_err(|e| TrieNodeError::Provider(e.to_string()))?;

                            non_empty_node.unblind(fetcher)?;
                            self.collapse_if_possible(fetcher, hinter)?;
                        }
                        _ => {}
                    };
                }
            }
            _ => {}
        }
        Ok(())
    }

    /// Attempts to convert a `path` and `value` into a [`TrieNode`], if they correspond to a
    /// [`TrieNode::Leaf`] or [`TrieNode::Extension`].
    ///
    /// **Note:** This function assumes that the passed reader has already consumed the RLP header
    /// of the [`TrieNode::Leaf`] or [`TrieNode::Extension`] node.
    fn try_decode_leaf_or_extension_payload(buf: &mut &[u8]) -> TrieNodeResult<Self> {
        // Decode the path and value of the leaf or extension node.
        let path = Bytes::decode(buf).map_err(TrieNodeError::RLPError)?;
        let first_nibble = path[0] >> NIBBLE_WIDTH;
        let first = match first_nibble {
            PREFIX_EXTENSION_ODD | PREFIX_LEAF_ODD => Some(path[0] & 0x0F),
            PREFIX_EXTENSION_EVEN | PREFIX_LEAF_EVEN => None,
            _ => return Err(TrieNodeError::InvalidNodeType),
        };

        // Check the high-order nibble of the path to determine the type of node.
        match first_nibble {
            PREFIX_EXTENSION_EVEN | PREFIX_EXTENSION_ODD => {
                // Extension node
                let extension_node_value = Self::decode(buf).map_err(TrieNodeError::RLPError)?;
                Ok(Self::Extension {
                    prefix: unpack_path_to_nibbles(first, path[1..].as_ref()),
                    node: Box::new(extension_node_value),
                })
            }
            PREFIX_LEAF_EVEN | PREFIX_LEAF_ODD => {
                // Leaf node
                let value = Bytes::decode(buf).map_err(TrieNodeError::RLPError)?;
                Ok(Self::Leaf { prefix: unpack_path_to_nibbles(first, path[1..].as_ref()), value })
            }
            _ => Err(TrieNodeError::InvalidNodeType),
        }
    }

    /// Returns the RLP payload length of the [`TrieNode`].
    pub(crate) fn payload_length(&self) -> usize {
        match self {
            Self::Empty => 0,
            Self::Blinded { commitment } => commitment.len(),
            Self::Leaf { prefix, value } => {
                let mut encoded_key_len = prefix.len() / 2 + 1;
                if encoded_key_len != 1 {
                    encoded_key_len += length_of_length(encoded_key_len);
                }
                encoded_key_len + value.length()
            }
            Self::Extension { prefix, node } => {
                let mut encoded_key_len = prefix.len() / 2 + 1;
                if encoded_key_len != 1 {
                    encoded_key_len += length_of_length(encoded_key_len);
                }
                encoded_key_len + node.blinded_length()
            }
            Self::Branch { stack } => {
                // In branch nodes, if an element is longer than an encoded 32 byte string, it is
                // blinded. Assuming we have an open trie node, we must re-hash the
                // elements that are longer than an encoded 32 byte string
                // in length.
                stack.iter().fold(0, |mut acc, node| {
                    acc += node.blinded_length();
                    acc
                })
            }
        }
    }

    /// Returns the encoded length of the trie node, blinding it if it is longer than an encoded
    /// [B256] string in length.
    ///
    /// ## Returns
    /// - `usize` - The encoded length of the value, blinded if the raw encoded length is longer
    ///   than a [B256].
    fn blinded_length(&self) -> usize {
        let encoded_len = self.length();
        if encoded_len >= B256::ZERO.len() { B256::ZERO.length() } else { encoded_len }
    }
}

impl Encodable for TrieNode {
    fn encode(&self, out: &mut dyn alloy_rlp::BufMut) {
        let payload_length = self.payload_length();
        match self {
            Self::Empty => out.put_u8(EMPTY_STRING_CODE),
            Self::Blinded { commitment } => commitment.encode(out),
            Self::Leaf { prefix, value } => {
                // Encode the leaf node's header and key-value pair.
                Header { list: true, payload_length }.encode(out);
                alloy_trie::nodes::encode_path_leaf(prefix, true).as_slice().encode(out);
                value.encode(out);
            }
            Self::Extension { prefix, node } => {
                // Encode the extension node's header, prefix, and pointer node.
                Header { list: true, payload_length }.encode(out);
                alloy_trie::nodes::encode_path_leaf(prefix, false).as_slice().encode(out);
                if node.length() >= B256::ZERO.len() {
                    let hash = node.blind();
                    hash.encode(out);
                } else {
                    node.encode(out);
                }
            }
            Self::Branch { stack } => {
                // In branch nodes, if an element is longer than 32 bytes in length, it is blinded.
                // Assuming we have an open trie node, we must re-hash the elements
                // that are longer than 32 bytes in length.
                Header { list: true, payload_length }.encode(out);
                for node in stack {
                    if node.length() >= B256::ZERO.len() {
                        let hash = node.blind();
                        hash.encode(out);
                    } else {
                        node.encode(out);
                    }
                }
            }
        }
    }

    fn length(&self) -> usize {
        match self {
            Self::Empty => 1,
            Self::Blinded { commitment } => commitment.length(),
            Self::Leaf { .. } | Self::Extension { .. } | Self::Branch { .. } => {
                let payload_length = self.payload_length();
                Header { list: true, payload_length }.length() + payload_length
            }
        }
    }
}

impl Decodable for TrieNode {
    /// Attempts to decode the [`TrieNode`].
    fn decode(buf: &mut &[u8]) -> alloy_rlp::Result<Self> {
        // Peek at the header to determine the type of Trie node we're currently decoding.
        let header = Header::decode(&mut (**buf).as_ref())?;

        if header.list {
            // Peek at the RLP stream to determine the number of elements in the list.
            let list_length = rlp_list_element_length(&mut (**buf).as_ref())?;

            match list_length {
                BRANCH_LIST_LENGTH => {
                    let list = Vec::<Self>::decode(buf)?;
                    Ok(Self::Branch { stack: list })
                }
                LEAF_OR_EXTENSION_LIST_LENGTH => {
                    // Advance the buffer to the start of the list payload.
                    buf.advance(header.length());
                    // Decode the leaf or extension node's raw payload.
                    Self::try_decode_leaf_or_extension_payload(buf)
                        .map_err(|_| alloy_rlp::Error::UnexpectedList)
                }
                _ => Err(alloy_rlp::Error::UnexpectedLength),
            }
        } else {
            match header.payload_length {
                0 => {
                    buf.advance(header.length());
                    Ok(Self::Empty)
                }
                32 => {
                    let commitment = B256::decode(buf)?;
                    Ok(Self::new_blinded(commitment))
                }
                _ => Err(alloy_rlp::Error::UnexpectedLength),
            }
        }
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::{
        NoopTrieHinter, NoopTrieProvider, TrieNode, ordered_trie_with_encoder,
        test_util::TrieNodeProvider,
    };
    use alloc::{collections::BTreeMap, vec, vec::Vec};
    use alloy_primitives::{b256, bytes, hex, keccak256};
    use alloy_rlp::{Decodable, EMPTY_STRING_CODE, Encodable};
    use alloy_trie::{HashBuilder, Nibbles};
    use rand::prelude::IteratorRandom;

    #[test]
    fn test_empty_blinded() {
        let trie_node = TrieNode::Empty;
        assert_eq!(trie_node.blind(), EMPTY_ROOT_HASH);
    }

    #[test]
    fn test_decode_branch() {
        const BRANCH_RLP: [u8; 83] = hex!(
            "f851a0eb08a66a94882454bec899d3e82952dcc918ba4b35a09a84acd98019aef4345080808080808080a05d87a81d9bbf5aee61a6bfeab3a5643347e2c751b36789d988a5b6b163d496518080808080808080"
        );
        let expected = TrieNode::Branch {
            stack: vec![
                TrieNode::new_blinded(b256!(
                    "eb08a66a94882454bec899d3e82952dcc918ba4b35a09a84acd98019aef43450"
                )),
                TrieNode::Empty,
                TrieNode::Empty,
                TrieNode::Empty,
                TrieNode::Empty,
                TrieNode::Empty,
                TrieNode::Empty,
                TrieNode::Empty,
                TrieNode::new_blinded(b256!(
                    "5d87a81d9bbf5aee61a6bfeab3a5643347e2c751b36789d988a5b6b163d49651"
                )),
                TrieNode::Empty,
                TrieNode::Empty,
                TrieNode::Empty,
                TrieNode::Empty,
                TrieNode::Empty,
                TrieNode::Empty,
                TrieNode::Empty,
                TrieNode::Empty,
            ],
        };

        let mut rlp_buf = Vec::with_capacity(expected.length());
        expected.encode(&mut rlp_buf);
        assert_eq!(rlp_buf.len(), BRANCH_RLP.len());
        assert_eq!(expected.length(), BRANCH_RLP.len());

        assert_eq!(expected, TrieNode::decode(&mut BRANCH_RLP.as_slice()).unwrap());
        assert_eq!(rlp_buf.as_slice(), &BRANCH_RLP[..]);
    }

    #[test]
    fn test_encode_decode_extension_open_short() {
        const EXTENSION_RLP: [u8; 19] = hex!("d28300646fcd308b8a74657374207468726565");

        let opened = TrieNode::Leaf {
            prefix: Nibbles::from_nibbles([0x00]),
            value: bytes!("8a74657374207468726565"),
        };
        let expected =
            TrieNode::Extension { prefix: Nibbles::unpack(bytes!("646f")), node: Box::new(opened) };

        let mut rlp_buf = Vec::with_capacity(expected.length());
        expected.encode(&mut rlp_buf);

        assert_eq!(expected, TrieNode::decode(&mut EXTENSION_RLP.as_slice()).unwrap());
    }

    #[test]
    fn test_encode_decode_extension_blinded_long() {
        const EXTENSION_RLP: [u8; 38] =
            hex!("e58300646fa0f3fe8b3c5b21d3e52860f1e4a5825a6100bb341069c1e88f4ebf6bd98de0c190");
        let mut rlp_buf = Vec::new();

        let opened =
            TrieNode::Leaf { prefix: Nibbles::from_nibbles([0x00]), value: [0xFF; 64].into() };
        opened.encode(&mut rlp_buf);
        let blinded = TrieNode::new_blinded(keccak256(&rlp_buf));

        rlp_buf.clear();
        let opened_extension =
            TrieNode::Extension { prefix: Nibbles::unpack(bytes!("646f")), node: Box::new(opened) };
        opened_extension.encode(&mut rlp_buf);

        let expected = TrieNode::Extension {
            prefix: Nibbles::unpack(bytes!("646f")),
            node: Box::new(blinded),
        };
        assert_eq!(expected, TrieNode::decode(&mut EXTENSION_RLP.as_slice()).unwrap());
    }

    #[test]
    fn test_decode_leaf() {
        const LEAF_RLP: [u8; 11] = hex!("ca8320646f8576657262FF");
        let expected =
            TrieNode::Leaf { prefix: Nibbles::unpack(bytes!("646f")), value: bytes!("76657262FF") };
        assert_eq!(expected, TrieNode::decode(&mut LEAF_RLP.as_slice()).unwrap());
    }

    #[test]
    fn test_retrieve_from_trie_simple() {
        const VALUES: [&str; 5] = ["yeah", "dog", ", ", "laminar", "flow"];

        let mut trie = ordered_trie_with_encoder(&VALUES, |v, buf| {
            let mut encoded_value = Vec::with_capacity(v.length());
            v.encode(&mut encoded_value);
            TrieNode::new_blinded(keccak256(encoded_value)).encode(buf);
        });
        let root = trie.root();

        let preimages = trie.take_proof_nodes().into_inner().into_iter().fold(
            BTreeMap::default(),
            |mut acc, (_, value)| {
                acc.insert(keccak256(value.as_ref()), value);
                acc
            },
        );
        let fetcher = TrieNodeProvider::new(preimages);

        let mut root_node = fetcher.trie_node_by_hash(root).unwrap();
        for (i, value) in VALUES.iter().enumerate() {
            let path_nibbles = Nibbles::unpack([if i == 0 { EMPTY_STRING_CODE } else { i as u8 }]);
            let v = root_node.open(&path_nibbles, &fetcher).unwrap().unwrap();

            let mut encoded_value = Vec::with_capacity(value.length());
            value.encode(&mut encoded_value);
            let mut encoded_node = Vec::new();
            TrieNode::new_blinded(keccak256(&encoded_value)).encode(&mut encoded_node);

            assert_eq!(v, encoded_node.as_slice());
        }

        let commitment = root_node.blind();
        assert_eq!(commitment, root);
    }

    #[test]
    fn test_insert_static() {
        let mut node = TrieNode::Empty;
        let noop_fetcher = NoopTrieProvider;
        node.insert(&Nibbles::unpack(hex!("012345")), bytes!("01"), &noop_fetcher).unwrap();
        node.insert(&Nibbles::unpack(hex!("012346")), bytes!("02"), &noop_fetcher).unwrap();

        let expected = TrieNode::Extension {
            prefix: Nibbles::from_nibbles([0, 1, 2, 3, 4]),
            node: Box::new(TrieNode::Branch {
                stack: vec![
                    TrieNode::Empty,
                    TrieNode::Empty,
                    TrieNode::Empty,
                    TrieNode::Empty,
                    TrieNode::Empty,
                    TrieNode::Leaf { prefix: Nibbles::default(), value: bytes!("01") },
                    TrieNode::Leaf { prefix: Nibbles::default(), value: bytes!("02") },
                    TrieNode::Empty,
                    TrieNode::Empty,
                    TrieNode::Empty,
                    TrieNode::Empty,
                    TrieNode::Empty,
                    TrieNode::Empty,
                    TrieNode::Empty,
                    TrieNode::Empty,
                    TrieNode::Empty,
                    TrieNode::Empty,
                ],
            }),
        };

        assert_eq!(node, expected);
    }

    proptest::proptest! {
        /// Differential test for inserting an arbitrary number of keys into an empty `TrieNode` / `HashBuilder`.
        #[test]
        fn diff_hash_builder_insert(mut keys in proptest::collection::vec(proptest::prelude::any::<[u8; 32]>(), 1..4096)) {
            // Ensure the keys are sorted; `HashBuilder` expects sorted keys.`
            keys.sort();

            let mut hb = HashBuilder::default();
            let mut node = TrieNode::Empty;

            for key in keys {
                hb.add_leaf(Nibbles::unpack(key), key.as_ref());
                node.insert(&Nibbles::unpack(key), key.into(), &NoopTrieProvider).unwrap();
            }

            assert_eq!(node.blind(), hb.root());
        }

        /// Differential test for deleting an arbitrary number of keys from a `TrieNode` / `HashBuilder`.
        #[test]
        fn diff_hash_builder_delete(mut keys in proptest::collection::vec(proptest::prelude::any::<[u8; 32]>(), 1..4096)) {
            // Ensure the keys are sorted; `HashBuilder` expects sorted keys.`
            keys.sort();

            let mut hb = HashBuilder::default();
            let mut node = TrieNode::Empty;

            let mut rng = rand::rng();
            let deleted_keys =
            keys.clone().into_iter().choose_multiple(&mut rng, 5.min(keys.len()));

            // Insert the keys into the `HashBuilder` and `TrieNode`.
            for key in keys {
                // Don't add any keys that are to be deleted from the trie node to the `HashBuilder`.
                if !deleted_keys.contains(&key) {
                    hb.add_leaf(Nibbles::unpack(key), key.as_ref());
                }
                node.insert(&Nibbles::unpack(key), key.into(), &NoopTrieProvider).unwrap();
            }

            // Delete the keys that were randomly selected from the trie node.
            for deleted_key in deleted_keys {
                node.delete(&Nibbles::unpack(deleted_key), &NoopTrieProvider, &NoopTrieHinter)
                    .unwrap();
            }

            // Blind manually, since the single node remaining may be a leaf or empty node, and always must be blinded.
            let mut rlp_buf = Vec::with_capacity(node.length());
            node.encode(&mut rlp_buf);
            let trie_root = keccak256(rlp_buf);

            assert_eq!(trie_root, hb.root());
        }
    }
}
