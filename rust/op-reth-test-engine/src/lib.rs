//! `op-reth-test-engine`: a library-first, ephemeral, OP-flavored execution layer for tests.
//!
//! Tests drive this as a library (Rust action tests, fuzzers, replays) or, via the companion
//! binary, as a subprocess over a Unix socket (Go action tests). It exists so `op-e2e/actions`
//! can run an OP-flavored execution layer without embedding op-geth in-process.

mod chain;

pub use chain::EphemeralChain;

use alloy_consensus::Header;
use alloy_genesis::Genesis;
use alloy_primitives::B256;
use reth_db_common::init::InitStorageError;
use reth_optimism_primitives::{OpBlock, OpReceipt};
use reth_provider::ProviderError;

/// Errors produced by the test engine.
#[derive(Debug, thiserror::Error)]
pub enum Error {
    /// A reth provider or storage query failed.
    #[error(transparent)]
    Provider(#[from] ProviderError),
    /// Genesis initialization failed.
    #[error(transparent)]
    InitStorage(#[from] InitStorageError),
}

/// Result alias for this crate.
pub type Result<T> = core::result::Result<T, Error>;

/// The in-test OP execution engine.
///
/// Constructs an ephemeral, genesis-initialized chain and answers read-only chain queries.
#[derive(Debug)]
pub struct TestEngine {
    chain: EphemeralChain,
}

impl TestEngine {
    /// Construct an engine over a fresh ephemeral chain initialized from `genesis`.
    pub fn new(genesis: Genesis) -> Result<Self> {
        Ok(Self { chain: EphemeralChain::new(genesis)? })
    }

    /// Fetch a block by number, or `None` if unknown.
    pub fn block_by_number(&self, number: u64) -> Result<Option<OpBlock>> {
        self.chain.block_by_number(number)
    }

    /// Fetch a block by hash, or `None` if unknown.
    pub fn block_by_hash(&self, hash: B256) -> Result<Option<OpBlock>> {
        self.chain.block_by_hash(hash)
    }

    /// Fetch a header by number, or `None` if unknown.
    pub fn header_by_number(&self, number: u64) -> Result<Option<Header>> {
        self.chain.header_by_number(number)
    }

    /// Fetch the receipts of a block by hash, or `None` if unknown.
    pub fn receipts_by_block_hash(&self, hash: B256) -> Result<Option<Vec<OpReceipt>>> {
        self.chain.receipts_by_block_hash(hash)
    }
}
