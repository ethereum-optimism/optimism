use kona_protocol::BlockInfo;

/// Progress published by the derivation actor.
#[derive(Debug, Default, Clone, Copy, PartialEq, Eq)]
pub struct DerivationStatus {
    /// The L1 block at the derivation pipeline's current origin.
    /// This block may not yet have been fully derived into L2 data.
    pub current_l1: Option<BlockInfo>,
}
