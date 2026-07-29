//! Contains the [`TrieDBProvider`] trait for fetching EVM bytecode hash preimages as well as
//! [Header] preimages.

use alloc::string::String;
use alloy_consensus::Header;
use alloy_primitives::{B256, Bytes};
use kona_mpt::{TrieNode, TrieProvider};

/// The [`TrieDBProvider`] trait defines the synchronous interface for fetching EVM bytecode hash
/// preimages as well as [Header] preimages.
pub trait TrieDBProvider: TrieProvider {
    /// Fetches the preimage of the bytecode hash provided.
    ///
    /// ## Takes
    /// - `hash`: The hash of the bytecode.
    ///
    /// ## Returns
    /// - Ok(Bytes): The bytecode of the contract.
    /// - `Err(Self::Error)`: If the bytecode hash could not be fetched.
    ///
    /// [TrieDB]: crate::TrieDB
    fn bytecode_by_hash(&self, code_hash: B256) -> Result<Bytes, Self::Error>;

    /// Fetches the preimage of [Header] hash provided.
    ///
    /// ## Takes
    /// - `hash`: The hash of the RLP-encoded [Header].
    ///
    /// ## Returns
    /// - Ok(Bytes): The [Header].
    /// - `Err(Self::Error)`: If the [Header] could not be fetched.
    ///
    /// [TrieDB]: crate::TrieDB
    fn header_by_hash(&self, hash: B256) -> Result<Header, Self::Error>;
}

/// The default, no-op implementation of the [`TrieDBProvider`] trait, used for testing.
#[derive(Debug, Clone, Copy)]
pub struct NoopTrieDBProvider;

impl TrieProvider for NoopTrieDBProvider {
    type Error = String;

    fn trie_node_by_hash(&self, _key: B256) -> Result<TrieNode, Self::Error> {
        Ok(TrieNode::Empty)
    }
}

impl TrieDBProvider for NoopTrieDBProvider {
    fn bytecode_by_hash(&self, _code_hash: B256) -> Result<Bytes, Self::Error> {
        Ok(Bytes::default())
    }

    fn header_by_hash(&self, _hash: B256) -> Result<Header, Self::Error> {
        Ok(Header::default())
    }
}
