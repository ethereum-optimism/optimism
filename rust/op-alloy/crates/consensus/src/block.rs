//! Optimism Block Type.

use crate::OpTxEnvelope;

/// An Optimism block type.
pub type OpBlock = alloy_consensus::Block<OpTxEnvelope>;
