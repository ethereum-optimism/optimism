//! Contains the enums that configure the mode for the node to operate in and how it syncs.

/// The [`NodeMode`] enum represents the modes of operation for the [`RollupNode`].
///
/// [`RollupNode`]: crate::RollupNode
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

/// How a [`RollupNode`] validator starts syncing. Named after op-node's `--syncmode` values.
///
/// [`RollupNode`]: crate::RollupNode
#[derive(
    Debug,
    Default,
    Clone,
    Copy,
    PartialEq,
    Eq,
    strum::Display,
    strum::EnumString,
    strum::EnumIter,
    strum::IntoStaticStr,
)]
#[strum(serialize_all = "kebab-case")]
pub enum SyncMode {
    /// Derivation waits until an unsafe payload from gossip makes the execution layer answer a
    /// forkchoice update with `VALID`, which finishes EL sync.
    #[default]
    ExecutionLayer,
    /// A validator resets its engine at startup. If the execution layer answers that forkchoice
    /// update with `VALID`, derivation starts from L1 without waiting for an unsafe payload.
    /// Otherwise it waits for an unsafe payload as in [`Self::ExecutionLayer`].
    ///
    /// This is not op-node's full consensus-layer sync: an unsafe payload with a gap before it
    /// still makes the execution layer sync, and there is no queue holding such payloads until
    /// derivation fills the gap.
    ConsensusLayer,
}

impl SyncMode {
    /// Whether the engine actor of a node in `mode` starts with a reset pending: only a validator
    /// in consensus-layer sync mode. A sequencer's sequencer actor requests a reset at startup in
    /// either mode.
    pub const fn resets_engine_at_startup(self, mode: NodeMode) -> bool {
        matches!((self, mode), (Self::ConsensusLayer, NodeMode::Validator))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::rstest;

    #[rstest]
    #[case::el_validator(SyncMode::ExecutionLayer, NodeMode::Validator, false)]
    #[case::el_sequencer(SyncMode::ExecutionLayer, NodeMode::Sequencer, false)]
    #[case::cl_validator(SyncMode::ConsensusLayer, NodeMode::Validator, true)]
    #[case::cl_sequencer(SyncMode::ConsensusLayer, NodeMode::Sequencer, false)]
    fn only_a_consensus_layer_validator_resets_its_engine_at_startup(
        #[case] sync_mode: SyncMode,
        #[case] mode: NodeMode,
        #[case] resets: bool,
    ) {
        assert_eq!(sync_mode.resets_engine_at_startup(mode), resets);
    }
}
