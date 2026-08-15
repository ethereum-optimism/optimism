//! Node operating modes.

/// Rollup node mode.
#[derive(
    Debug,
    Default,
    Clone,
    Copy,
    PartialEq,
    Eq,
    derive_more::Display,
    derive_more::FromStr,
    strum::EnumIter,
)]
pub enum NodeMode {
    /// Follows unsafe gossip and derives the safe chain.
    #[display("Validator")]
    #[default]
    Validator,
    /// Also supports local unsafe block production.
    #[display("Sequencer")]
    Sequencer,
}

impl NodeMode {
    /// Returns whether this node supports local sequencing.
    pub const fn is_sequencer(self) -> bool {
        matches!(self, Self::Sequencer)
    }
}

/// Derivation pipeline mode.
#[derive(Debug, derive_more::Display, Default, Clone, Copy, PartialEq, Eq)]
pub enum InteropMode {
    /// Polls L1 data directly.
    #[display("Polled")]
    #[default]
    Polled,
    /// Uses indexed L1 data access.
    #[display("Indexed")]
    Indexed,
}
