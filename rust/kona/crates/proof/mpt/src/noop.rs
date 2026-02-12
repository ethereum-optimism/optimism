//! Trait implementations for `kona-mpt` traits that are effectively a no-op.
//! Provides trait implementations for downstream users who do not require hinting.

use crate::{TrieHinter, TrieNode, TrieProvider};
use alloc::string::String;
use alloy_primitives::{Address, B256, U256};
use core::fmt::Debug;

/// The default, no-op implementation of the [`TrieProvider`] trait, used for testing.
#[derive(Debug, Clone, Copy)]
pub struct NoopTrieProvider;

impl TrieProvider for NoopTrieProvider {
    type Error = String;

    fn trie_node_by_hash(&self, _key: B256) -> Result<TrieNode, Self::Error> {
        Ok(TrieNode::Empty)
    }
}

/// The default, no-op implementation of the [`TrieHinter`] trait, used for testing.
#[derive(Debug, Clone, Copy)]
pub struct NoopTrieHinter;

impl TrieHinter for NoopTrieHinter {
    type Error = String;

    fn hint_trie_node(&self, _hash: B256) -> Result<(), Self::Error> {
        Ok(())
    }

    fn hint_account_proof(&self, _address: Address, _block_number: u64) -> Result<(), Self::Error> {
        Ok(())
    }

    fn hint_storage_proof(
        &self,
        _address: Address,
        _slot: U256,
        _block_number: u64,
    ) -> Result<(), Self::Error> {
        Ok(())
    }

    fn hint_execution_witness(
        &self,
        _parent_hash: B256,
        _op_payload_attributes: &op_alloy_rpc_types_engine::OpPayloadAttributes,
    ) -> Result<(), Self::Error> {
        Ok(())
    }
}
