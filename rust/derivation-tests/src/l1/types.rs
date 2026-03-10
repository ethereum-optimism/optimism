//! L1 block and batch submission types.

use alloy_consensus::Header;
use alloy_primitives::{B256, Bytes, Sealed};

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
