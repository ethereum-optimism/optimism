//! Contains the enum that configures the mode for the node to operate in.

/// The node's mode of operation.
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
    /// Validator mode.
    #[display("Validator")]
    #[default]
    Validator,
    /// Sequencer mode.
    #[display("Sequencer")]
    Sequencer,
}

impl NodeMode {
    /// Returns `true` if [`Self`] is [`Self::Validator`].
    pub const fn is_validator(&self) -> bool {
        matches!(self, Self::Validator)
    }

    /// Returns `true` if [`Self`] is [`Self::Sequencer`].
    pub const fn is_sequencer(&self) -> bool {
        matches!(self, Self::Sequencer)
    }
}
