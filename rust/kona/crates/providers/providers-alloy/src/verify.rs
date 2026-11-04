//! Verification of consensus data supplied by an untrusted RPC.

use alloy_consensus::{Header, ReceiptEnvelope, proofs, transaction::TxHashRef};
use alloy_eips::eip2718::Encodable2718;
use alloy_primitives::{B256, keccak256};

/// An RPC response disagrees with its header or requested identifier.
#[derive(Debug, thiserror::Error)]
pub(crate) enum Error {
    #[error("Header hash mismatch: expected {expected}, got {actual}")]
    HeaderHash { expected: B256, actual: B256 },
    #[error("Block number mismatch: expected {expected}, got {actual}")]
    BlockNumber { expected: u64, actual: u64 },
    #[error("Transactions root mismatch: expected {expected}, got {actual}")]
    TransactionsRoot { expected: B256, actual: B256 },
    #[error("Transaction {index} hash mismatch: expected {expected}, got {actual}")]
    TransactionHash { index: usize, expected: B256, actual: B256 },
    #[error("Receipts root mismatch: expected {expected}, got {actual}")]
    ReceiptsRoot { expected: B256, actual: B256 },
}

pub(crate) fn header_hash(header: &Header, expected: B256) -> Result<(), Error> {
    let actual = header.hash_slow();
    if actual != expected {
        return Err(Error::HeaderHash { expected, actual });
    }
    Ok(())
}

pub(crate) const fn block_number(header: &Header, expected: u64) -> Result<(), Error> {
    if header.number != expected {
        return Err(Error::BlockNumber { expected, actual: header.number });
    }
    Ok(())
}

pub(crate) fn transactions<T: Encodable2718 + TxHashRef>(
    header: &Header,
    transactions: &[T],
) -> Result<(), Error> {
    let actual = proofs::calculate_transaction_root(transactions);
    if actual != header.transactions_root {
        return Err(Error::TransactionsRoot { expected: header.transactions_root, actual });
    }
    for (index, tx) in transactions.iter().enumerate() {
        // RPC decoding can retain an unverified hash in Signed/Sealed, including PostExec
        // transactions. Neither that cached hash nor trie_hash() authenticates the encoding.
        let actual = keccak256(tx.encoded_2718());
        let expected = *tx.tx_hash();
        if actual != expected {
            return Err(Error::TransactionHash { index, expected, actual });
        }
    }
    Ok(())
}

pub(crate) fn receipts(header: &Header, receipts: &[ReceiptEnvelope]) -> Result<(), Error> {
    let actual = proofs::calculate_receipt_root(receipts);
    if actual != header.receipts_root {
        return Err(Error::ReceiptsRoot { expected: header.receipts_root, actual });
    }
    Ok(())
}
