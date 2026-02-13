use alloy_consensus::BlockBody;
use reth_optimism_primitives::{DepositReceipt, transaction::OpTransaction};
use reth_payload_primitives::PayloadBuilderAttributes;
use reth_primitives_traits::{FullBlockHeader, NodePrimitives, SignedTransaction, WithEncoded};

use crate::OpPayloadBuilderAttributes;

/// Helper trait to encapsulate common bounds on [`NodePrimitives`] for OP payload builder.
pub trait OpPayloadPrimitives:
    NodePrimitives<
        Receipt: DepositReceipt,
        SignedTx = Self::_TX,
        BlockBody = BlockBody<Self::_TX, Self::_Header>,
        BlockHeader = Self::_Header,
    >
{
    /// Helper AT to bound [`NodePrimitives::Block`] type without causing bound cycle.
    type _TX: SignedTransaction + OpTransaction;
    /// Helper AT to bound [`NodePrimitives::Block`] type without causing bound cycle.
    type _Header: FullBlockHeader;
}

impl<Tx, T, Header> OpPayloadPrimitives for T
where
    Tx: SignedTransaction + OpTransaction,
    T: NodePrimitives<
            SignedTx = Tx,
            Receipt: DepositReceipt,
            BlockBody = BlockBody<Tx, Header>,
            BlockHeader = Header,
        >,
    Header: FullBlockHeader,
{
    type _TX = Tx;
    type _Header = Header;
}

/// Attributes for the OP payload builder.
pub trait OpAttributes: PayloadBuilderAttributes {
    /// Primitive transaction type.
    type Transaction: SignedTransaction;

    /// Whether to use the transaction pool for the payload.
    fn no_tx_pool(&self) -> bool;

    /// Sequencer transactions to include in the payload.
    fn sequencer_transactions(&self) -> &[WithEncoded<Self::Transaction>];
}

impl<T: SignedTransaction> OpAttributes for OpPayloadBuilderAttributes<T> {
    type Transaction = T;

    fn no_tx_pool(&self) -> bool {
        self.no_tx_pool
    }

    fn sequencer_transactions(&self) -> &[WithEncoded<Self::Transaction>] {
        &self.transactions
    }
}
