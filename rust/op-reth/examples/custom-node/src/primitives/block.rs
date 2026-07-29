use crate::primitives::{CustomHeader, CustomTransaction};

/// The Block type of this node
pub type Block = alloy_consensus::Block<CustomTransaction, CustomHeader>;

/// The body type of this node
pub type BlockBody = alloy_consensus::BlockBody<CustomTransaction, CustomHeader>;
