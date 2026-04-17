use alloy_consensus::BlockBody;
use alloy_primitives::B256;
use alloy_rpc_types_engine::PayloadId;
use reth_optimism_primitives::{DepositReceipt, transaction::OpTransaction};
use reth_payload_builder_primitives::PayloadBuilderError;
use reth_primitives_traits::{FullBlockHeader, NodePrimitives, SignedTransaction, WithEncoded};

use crate::{OpPayloadAttributes, OpPayloadBuilderAttributes};

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
pub trait OpAttributes: Send + Sync + core::fmt::Debug + 'static {
    /// Primitive transaction type.
    type Transaction: SignedTransaction;

    /// The RPC payload attributes type used to create these builder attributes.
    type RpcPayloadAttributes;

    /// Creates a new instance from the parent hash and RPC payload attributes.
    fn try_new(
        parent: B256,
        attributes: Self::RpcPayloadAttributes,
        version: u8,
    ) -> Result<Self, PayloadBuilderError>
    where
        Self: Sized;

    /// Returns the identifier of the payload.
    fn payload_id(&self) -> PayloadId;

    /// Returns the timestamp for the payload.
    fn timestamp(&self) -> u64;

    /// Whether to use the transaction pool for the payload.
    fn no_tx_pool(&self) -> bool;

    /// Sequencer transactions to include in the payload.
    fn sequencer_transactions(&self) -> &[WithEncoded<Self::Transaction>];
}

impl<T: SignedTransaction> OpAttributes for OpPayloadBuilderAttributes<T> {
    type Transaction = T;
    type RpcPayloadAttributes = OpPayloadAttributes;

    fn try_new(
        parent: B256,
        attributes: OpPayloadAttributes,
        version: u8,
    ) -> Result<Self, PayloadBuilderError> {
        Self::try_new(parent, attributes, version).map_err(PayloadBuilderError::other)
    }

    fn payload_id(&self) -> PayloadId {
        self.id
    }

    fn timestamp(&self) -> u64 {
        self.timestamp
    }

    fn no_tx_pool(&self) -> bool {
        self.no_tx_pool
    }

    fn sequencer_transactions(&self) -> &[WithEncoded<Self::Transaction>] {
        &self.transactions
    }
}
