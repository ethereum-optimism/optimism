use kona_protocol::BlockInfo;

/// Latest successful, independently polled L1 observations.
///
/// These heads need not describe one immutable chain snapshot during a reorg. Missing initial
/// observations remain unknown until their own poll succeeds.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct L1State {
    /// Latest observed L1 head.
    pub head: Option<BlockInfo>,
    /// Latest observed safe L1 block.
    pub safe: Option<BlockInfo>,
    /// Latest observed finalized L1 block.
    pub finalized: Option<BlockInfo>,
}

/// Progress reported by the active derivation source, independent of applied L2 safe heads.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct DerivationStatus {
    /// Actual L1 origin reached by the active derivation source; it may not be fully processed
    /// yet.
    pub current_l1: Option<BlockInfo>,
}
