use alloy_primitives::B256;

/// A parsed executing message extracted from a log emitted by the
/// `CrossL2Inbox` contract on an L2 chain.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ExecutingMessage {
    /// The chain ID where the message was observed.
    pub chain_id: u64,
    /// The block number that contained the log.
    pub block_number: u64,
    /// The log index within the block.
    pub log_index: u32,
    /// The timestamp of the block.
    pub timestamp: u64,
    /// A unique hash identifying the log (based on payload and origin).
    pub hash: B256,
}
