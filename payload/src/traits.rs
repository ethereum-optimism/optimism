use alloy_consensus::BlockBody;
use reth_optimism_primitives::{transaction::OpTransaction, DepositReceipt};
use reth_primitives_traits::{FullBlockHeader, NodePrimitives, SignedTransaction};

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
