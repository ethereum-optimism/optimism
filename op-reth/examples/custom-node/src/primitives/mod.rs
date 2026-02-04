//! Contains the primitive types of this node.

pub mod header;
pub use header::*;
pub mod block;
pub use block::*;
pub mod tx;
pub use tx::*;

pub mod tx_type;
pub use tx_type::*;
pub mod tx_custom;
pub use tx_custom::*;

use reth_ethereum::primitives::NodePrimitives;
use reth_op::OpReceipt;

#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct CustomNodePrimitives;

impl NodePrimitives for CustomNodePrimitives {
    type Block = Block;
    type BlockHeader = CustomHeader;
    type BlockBody = BlockBody;
    type SignedTx = CustomTransaction;
    type Receipt = OpReceipt;
}
