//! Why the execution layer rejected a forkchoice update that followed delegated safe heads.

/// Why the execution layer rejected a forkchoice update that followed delegated safe heads, as
/// far as the engine can tell.
#[derive(Debug, Clone, Copy, PartialEq, Eq, derive_more::Display)]
pub enum DelegatedRejection {
    /// The update moved the head onto a chain that does not contain the engine's finalized block.
    /// Every later head on that chain is rejected too.
    #[display("chain below finality")]
    ChainBelowFinality,
    /// The update moved the head onto a chain that contains the finalized block, and the execution
    /// layer rejected its labels, such as a safe block that is not an ancestor of the head. The
    /// same heads are rejected again.
    #[display("head labels")]
    HeadLabels,
    /// The update moved only the safe label. The execution layer may not have stored that block
    /// yet, so the same heads may be accepted on a later poll.
    #[display("safe label")]
    SafeLabel,
}

impl DelegatedRejection {
    /// Whether the same delegated heads are rejected again until the engine resets.
    pub const fn is_permanent(self) -> bool {
        !matches!(self, Self::SafeLabel)
    }
}
