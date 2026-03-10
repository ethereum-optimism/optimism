//! L1 block and batch submission types.

use alloy_consensus::Header;
use alloy_primitives::{Address, B256, Bytes, Sealed, U256};

/// A constructed L1 block.
#[derive(Debug, Clone)]
pub struct L1Block {
    /// Sealed header with computed block hash.
    pub header: Sealed<Header>,
    /// Raw transaction data (batch submissions and other txs).
    pub transactions: Vec<Bytes>,
    /// Receipts (simple success receipts for each tx).
    pub receipts: Vec<alloy_consensus::ReceiptEnvelope>,
}

/// How a batch is submitted to L1.
#[derive(Debug, Clone)]
pub enum BatchSubmission {
    /// Batch data in transaction calldata.
    Calldata(Bytes),
    /// Batch data encoded as a blob.
    Blob(BlobWithCommitment),
}

/// A system config update to emit as an L1 log event.
#[derive(Debug, Clone)]
pub enum SystemConfigUpdate {
    /// Update the batcher address.
    BatcherAddress(Address),
    /// Update the gas config (overhead + scalar).
    GasConfig {
        /// L1 fee overhead.
        overhead: U256,
        /// L1 fee scalar.
        scalar: U256,
    },
}

/// A blob with its KZG commitment and proof.
#[derive(Debug, Clone)]
pub struct BlobWithCommitment {
    /// The blob data.
    pub blob: Box<alloy_eips::eip4844::Blob>,
    /// KZG commitment.
    pub commitment: alloy_eips::eip4844::BlobTransactionSidecar,
    /// Versioned hash of the commitment.
    pub versioned_hash: B256,
}
