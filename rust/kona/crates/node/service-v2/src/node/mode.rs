//! Node operating mode.

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
