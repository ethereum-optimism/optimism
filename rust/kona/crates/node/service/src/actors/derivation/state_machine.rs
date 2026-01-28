use derive_more::PartialEq;
use kona_protocol::{L2BlockInfo, OpAttributesWithParent};
use thiserror::Error;
use tracing::info;

/// The possible states of the [`DerivationStateMachine`] implemented by the
/// [`crate::DerivationActor`].
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum DerivationState {
    /// The [`crate::DerivationActor`] is waiting for notification that the EL sync has completed
    /// before it can start derivation.
    AwaitingELSyncCompletion,
    /// The [`crate::DerivationActor`] is idle awaiting data.
    AwaitingL1Data,
    /// [`kona_protocol::OpAttributesWithParent`] were sent to the [`crate::EngineActor`], and the
    /// [`crate::DerivationActor`] is waiting for confirmation that they were processed into a safe
    /// head.
    AwaitingSafeHeadConfirmation,
    /// A reorg or some other inconsistency was detected, necessitating a [`kona_derive::Signal`] to
    /// be processed before continuing derivation.
    AwaitingSignal,
    /// After receiving a [`kona_derive::Signal`], we need an update of L1 data or a new engine
    /// safe head to start deriving again. This represents the state waiting for one of the two.
    AwaitingUpdateAfterSignal,
    /// The [`crate::DerivationActor`] is actively attempting derivation.
    Deriving,
}

/// The possible updates of the [`DerivationStateMachine`] implemented by the
/// [`crate::DerivationActor`].
#[derive(Clone, Debug, PartialEq)]
pub enum DerivationStateUpdate {
    /// The initial EL sync has completed along with the current safe head, allowing derivation to
    /// start.
    ELSyncCompleted(Box<L2BlockInfo>),
    /// More L1 data has become available to process.
    L1DataReceived,
    /// Further derivation is not possible without additional L1 data becoming available.
    MoreDataNeeded,
    /// Derivation has produced new [`kona_protocol::OpAttributesWithParent`].
    NewAttributesDerived(Box<OpAttributesWithParent>),
    /// The EL has confirmed the derived [`kona_protocol::OpAttributesWithParent`] as the new safe
    /// head.
    NewAttributesConfirmed(Box<L2BlockInfo>),
    /// A [`kona_derive::Signal`] is necessary to update the derivation pipeline in order to
    /// continue.
    SignalNeeded,
    /// A [`kona_derive::Signal`] has been received and processed.
    SignalProcessed,
}

/// An error processing a [DerivationStateMachine] state transition.
#[derive(Debug, Error)]
pub enum DerivationStateTransitionError {
    /// An invalid state transition was attempted.
    #[error("Invalid state transition, starting state: {state:?}, state_update: {update:?}.")]
    InvalidTransition {
        /// The [`DerivationState`] from which an invalid transition was attempted.
        state: DerivationState,
        /// The [`DerivationStateUpdate`] that is invalid from the [`DerivationState`].
        update: DerivationStateUpdate,
    },
}

// Details all valid state transitions.
fn transition(
    state: &DerivationState,
    update: &DerivationStateUpdate,
) -> Result<DerivationState, DerivationStateTransitionError> {
    match state {
        // NB: initial state. Once we transition away from this, we never go back.
        DerivationState::AwaitingELSyncCompletion => match update {
            DerivationStateUpdate::ELSyncCompleted(_) => Ok(DerivationState::Deriving),
            DerivationStateUpdate::NewAttributesConfirmed(_) |
            DerivationStateUpdate::SignalProcessed |
            DerivationStateUpdate::L1DataReceived => Ok(DerivationState::AwaitingELSyncCompletion),
            _ => Err(DerivationStateTransitionError::InvalidTransition {
                state: *state,
                update: update.clone(),
            }),
        },
        DerivationState::AwaitingL1Data => match update {
            DerivationStateUpdate::L1DataReceived => Ok(DerivationState::Deriving),
            DerivationStateUpdate::SignalProcessed => {
                Ok(DerivationState::AwaitingUpdateAfterSignal)
            }
            _ => Err(DerivationStateTransitionError::InvalidTransition {
                state: *state,
                update: update.clone(),
            }),
        },
        DerivationState::AwaitingSafeHeadConfirmation => match update {
            DerivationStateUpdate::NewAttributesConfirmed(_) => Ok(DerivationState::Deriving),
            DerivationStateUpdate::SignalProcessed => {
                Ok(DerivationState::AwaitingUpdateAfterSignal)
            }
            DerivationStateUpdate::L1DataReceived => {
                Ok(DerivationState::AwaitingSafeHeadConfirmation)
            }
            _ => Err(DerivationStateTransitionError::InvalidTransition {
                state: *state,
                update: update.clone(),
            }),
        },
        DerivationState::AwaitingSignal => match update {
            DerivationStateUpdate::SignalProcessed => {
                Ok(DerivationState::AwaitingUpdateAfterSignal)
            }
            DerivationStateUpdate::L1DataReceived | DerivationStateUpdate::MoreDataNeeded => {
                Ok(DerivationState::AwaitingSignal)
            }
            _ => Err(DerivationStateTransitionError::InvalidTransition {
                state: *state,
                update: update.clone(),
            }),
        },
        DerivationState::AwaitingUpdateAfterSignal => match update {
            DerivationStateUpdate::L1DataReceived |
            DerivationStateUpdate::NewAttributesConfirmed(_) => Ok(DerivationState::Deriving),
            DerivationStateUpdate::SignalProcessed => {
                Ok(DerivationState::AwaitingUpdateAfterSignal)
            }
            _ => Err(DerivationStateTransitionError::InvalidTransition {
                state: *state,
                update: update.clone(),
            }),
        },
        DerivationState::Deriving => match update {
            DerivationStateUpdate::NewAttributesDerived(_) => {
                Ok(DerivationState::AwaitingSafeHeadConfirmation)
            }
            DerivationStateUpdate::SignalNeeded => Ok(DerivationState::AwaitingSignal),
            DerivationStateUpdate::MoreDataNeeded => Ok(DerivationState::AwaitingL1Data),
            _ => Err(DerivationStateTransitionError::InvalidTransition {
                state: *state,
                update: update.clone(),
            }),
        },
    }
}

/// The state machine that controls the state of the [`crate::DerivationActor`].
/// This machine enforces the following conditions:
///
/// ## General prerequisites:
/// 1. Derivation may not occur until EL sync has completed
/// 2. Derivation may not happen until the Engine L2 safe head is known
///
/// ## Derive -> Message EL -> Receive confirmation
/// When new [`kona_protocol::OpAttributesWithParent`] are derived, they must be sent to the EL,
/// and the EL must confirm them by creating a new L2 safe head from them prior to further
/// derivation. There will be at most one [`kona_protocol::OpAttributesWithParent`] awaiting
/// confirmation at any given time.
///
/// ## Signal handling
/// Certain conditions require a [`kona_derive::Signal`] to be processed by the
/// [`kona_derive::Pipeline`], updating derivation state before continuing derivation. This struct
/// allows a caller to register that it is waiting on a signal as well as mark that it was
/// processed.
#[derive(Debug)]
pub struct DerivationStateMachine {
    confirmed_safe_head: L2BlockInfo,
    state: DerivationState,
}

impl Default for DerivationStateMachine {
    fn default() -> Self {
        Self::new()
    }
}

impl DerivationStateMachine {
    /// Constructs a new [`DerivationStateMachine`].
    fn new() -> Self {
        Self {
            confirmed_safe_head: L2BlockInfo::default(),
            state: DerivationState::AwaitingELSyncCompletion,
        }
    }

    /// Gets the current [`DerivationState`] of the state machine.
    pub const fn current_state(&self) -> DerivationState {
        self.state
    }

    /// Gets the last [`L2BlockInfo`] confirmed by the engine.
    pub const fn last_confirmed_safe_head(&self) -> L2BlockInfo {
        self.confirmed_safe_head
    }

    /// Applies the provided  [`DerivationStateUpdate`], returning an
    /// [`DerivationStateTransitionError`] if the state transition was invalid.
    pub fn update(
        &mut self,
        state_update: &DerivationStateUpdate,
    ) -> Result<(), DerivationStateTransitionError> {
        if let DerivationStateUpdate::NewAttributesConfirmed(safe_head) = state_update {
            if safe_head.block_info.hash == self.confirmed_safe_head.block_info.hash {
                info!(target: "derivation", ?safe_head, "Re-received safe head. Skipping state transition.");
            }
        }

        info!(target: "derivation", state=?self.state, ?state_update, "Executing derivation state update.");
        self.state = transition(&self.state, state_update)?;

        if let DerivationStateUpdate::NewAttributesConfirmed(safe_head) = state_update {
            self.confirmed_safe_head = **safe_head;
        } else if let DerivationStateUpdate::ELSyncCompleted(safe_head) = state_update {
            self.confirmed_safe_head = **safe_head;
        }

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::{
        DerivationState::*, DerivationStateMachine, DerivationStateTransitionError,
        DerivationStateUpdate::*, L2BlockInfo, transition,
    };
    use alloy_eips::BlockNumHash;
    use alloy_primitives::{BlockHash, b256};
    use kona_protocol::{BlockInfo, OpAttributesWithParent};
    use op_alloy_rpc_types_engine::OpPayloadAttributes;
    use rstest::rstest;

    /// Creates a dummy L2BlockInfo for testing
    fn dummy_l2_block_info() -> L2BlockInfo {
        L2BlockInfo {
            block_info: BlockInfo {
                hash: b256!("0000000000000000000000000000000000000000000000000000000000000001"),
                number: 1,
                parent_hash: BlockHash::default(),
                timestamp: 0,
            },
            l1_origin: BlockNumHash { hash: BlockHash::default(), number: 0 },
            seq_num: 0,
        }
    }

    /// Creates a dummy OpAttributesWithParent for testing
    fn dummy_op_attributes() -> OpAttributesWithParent {
        OpAttributesWithParent {
            attributes: OpPayloadAttributes::default(),
            parent: dummy_l2_block_info(),
            derived_from: None,
            is_last_in_span: false,
        }
    }

    // This is just here to shrink the #[case(...)] statements below for readability.
    fn attrs() -> Box<OpAttributesWithParent> {
        Box::new(dummy_op_attributes())
    }

    // This is just here to shrink the #[case(...)] statements below for readability.
    fn block() -> Box<L2BlockInfo> {
        Box::new(dummy_l2_block_info())
    }

    #[rstest]
    // AwaitingELSyncCompletion valid transitions
    #[case(AwaitingELSyncCompletion, ELSyncCompleted(block()), Deriving)]
    #[case(AwaitingELSyncCompletion, NewAttributesConfirmed(block()), AwaitingELSyncCompletion)]
    #[case(AwaitingELSyncCompletion, SignalProcessed, AwaitingELSyncCompletion)]
    #[case(AwaitingELSyncCompletion, L1DataReceived, AwaitingELSyncCompletion)]
    // AwaitingL1Data valid transitions
    #[case(AwaitingL1Data, L1DataReceived, Deriving)]
    #[case(AwaitingL1Data, SignalProcessed, AwaitingUpdateAfterSignal)]
    // AwaitingSafeHeadConfirmation valid transitions
    #[case(AwaitingSafeHeadConfirmation, NewAttributesConfirmed(block()), Deriving)]
    #[case(AwaitingSafeHeadConfirmation, SignalProcessed, AwaitingUpdateAfterSignal)]
    #[case(AwaitingSafeHeadConfirmation, L1DataReceived, AwaitingSafeHeadConfirmation)]
    // AwaitingSignal valid transitions
    #[case(AwaitingSignal, SignalProcessed, AwaitingUpdateAfterSignal)]
    #[case(AwaitingSignal, L1DataReceived, AwaitingSignal)]
    #[case(AwaitingSignal, MoreDataNeeded, AwaitingSignal)]
    // AwaitingUpdateAfterSignal valid transitions
    #[case(AwaitingUpdateAfterSignal, L1DataReceived, Deriving)]
    #[case(AwaitingUpdateAfterSignal, NewAttributesConfirmed(block()), Deriving)]
    #[case(AwaitingUpdateAfterSignal, SignalProcessed, AwaitingUpdateAfterSignal)]
    // Deriving valid transitions
    #[case(Deriving, NewAttributesDerived(attrs()), AwaitingSafeHeadConfirmation)]
    #[case(Deriving, SignalNeeded, AwaitingSignal)]
    #[case(Deriving, MoreDataNeeded, AwaitingL1Data)]
    fn test_valid_transitions(
        #[case] state: super::DerivationState,
        #[case] update: super::DerivationStateUpdate,
        #[case] expected_state: super::DerivationState,
    ) {
        let result = transition(&state, &update);
        assert!(result.is_ok(), "Expected valid transition from {state:?} with {update:?}");
        assert_eq!(
            result.unwrap(),
            expected_state,
            "Transition from {state:?} with {update:?} should result in {expected_state:?}"
        );
    }

    #[rstest]
    // AwaitingELSyncCompletion invalid transitions
    #[case(AwaitingELSyncCompletion, MoreDataNeeded)]
    #[case(AwaitingELSyncCompletion, NewAttributesDerived(attrs()))]
    #[case(AwaitingELSyncCompletion, SignalNeeded)]
    // AwaitingL1Data invalid transitions
    #[case(AwaitingL1Data, ELSyncCompleted(block()))]
    #[case(AwaitingL1Data, MoreDataNeeded)]
    #[case(AwaitingL1Data, NewAttributesDerived(attrs()))]
    #[case(AwaitingL1Data, NewAttributesConfirmed(block()))]
    #[case(AwaitingL1Data, SignalNeeded)]
    // AwaitingSafeHeadConfirmation invalid transitions
    #[case(AwaitingSafeHeadConfirmation, ELSyncCompleted(block()))]
    #[case(AwaitingSafeHeadConfirmation, MoreDataNeeded)]
    #[case(AwaitingSafeHeadConfirmation, NewAttributesDerived(attrs()))]
    #[case(AwaitingSafeHeadConfirmation, SignalNeeded)]
    // AwaitingSignal invalid transitions
    #[case(AwaitingSignal, ELSyncCompleted(block()))]
    #[case(AwaitingSignal, NewAttributesDerived(attrs()))]
    #[case(AwaitingSignal, NewAttributesConfirmed(block()))]
    #[case(AwaitingSignal, SignalNeeded)]
    // AwaitingUpdateAfterSignal invalid transitions
    #[case(AwaitingUpdateAfterSignal, ELSyncCompleted(block()))]
    #[case(AwaitingUpdateAfterSignal, MoreDataNeeded)]
    #[case(AwaitingUpdateAfterSignal, NewAttributesDerived(attrs()))]
    #[case(AwaitingUpdateAfterSignal, SignalNeeded)]
    // Deriving invalid transitions
    #[case(Deriving, ELSyncCompleted(block()))]
    #[case(Deriving, L1DataReceived)]
    #[case(Deriving, NewAttributesConfirmed(block()))]
    #[case(Deriving, SignalProcessed)]
    fn test_invalid_transitions(
        #[case] state: super::DerivationState,
        #[case] update: super::DerivationStateUpdate,
    ) {
        let result = transition(&state, &update);
        assert!(result.is_err(), "Expected invalid transition from {state:?} with {update:?}");
        match result.unwrap_err() {
            DerivationStateTransitionError::InvalidTransition {
                state: err_state,
                update: err_update,
            } => {
                assert_eq!(err_state, state);
                assert_eq!(err_update, update);
            }
        }
    }

    #[test]
    fn test_state_machine_initial_state() {
        let machine = DerivationStateMachine::new();
        assert_eq!(machine.current_state(), AwaitingELSyncCompletion);
        assert_eq!(machine.last_confirmed_safe_head(), L2BlockInfo::default());
    }

    #[test]
    fn test_state_machine_sync_completed_safe_head_update() {
        let mut machine = DerivationStateMachine::new();
        let safe_head = dummy_l2_block_info();

        machine.update(&ELSyncCompleted(Box::new(safe_head))).unwrap();

        assert_eq!(machine.current_state(), Deriving);
        assert_eq!(machine.last_confirmed_safe_head(), safe_head);
    }

    #[test]
    fn test_state_machine_update_preserves_confirmed_safe_head() {
        let mut machine = DerivationStateMachine::new();
        let first_safe_head = dummy_l2_block_info();

        machine.update(&ELSyncCompleted(Box::new(first_safe_head))).unwrap();

        // Transition to AwaitingL1Data
        machine.update(&MoreDataNeeded).unwrap();

        // Receive L1 data and go back to Deriving
        machine.update(&L1DataReceived).unwrap();

        // Safe head should still be the first one
        assert_eq!(machine.last_confirmed_safe_head(), first_safe_head);
    }

    #[test]
    fn test_state_machine_updates_safe_head_on_confirmation() {
        let mut machine = DerivationStateMachine::new();
        let initial_safe_head = dummy_l2_block_info();

        machine.update(&ELSyncCompleted(Box::new(initial_safe_head))).unwrap();

        // Derive new attributes
        machine.update(&NewAttributesDerived(Box::new(dummy_op_attributes()))).unwrap();

        let new_safe_head = L2BlockInfo {
            block_info: BlockInfo {
                hash: b256!("0000000000000000000000000000000000000000000000000000000000000002"),
                number: 2,
                parent_hash: initial_safe_head.block_info.hash,
                timestamp: 1,
            },
            l1_origin: BlockNumHash { hash: BlockHash::default(), number: 0 },
            seq_num: 0,
        };

        // Confirm new attributes
        machine.update(&NewAttributesConfirmed(Box::new(new_safe_head))).unwrap();

        assert_eq!(machine.current_state(), Deriving);
        assert_eq!(machine.last_confirmed_safe_head(), new_safe_head);
    }

    #[test]
    fn test_state_machine_invalid_transition_error() {
        let mut machine = DerivationStateMachine::new();

        let result = machine.update(&MoreDataNeeded);
        assert!(result.is_err());

        match result.unwrap_err() {
            DerivationStateTransitionError::InvalidTransition { state, update } => {
                assert_eq!(state, AwaitingELSyncCompletion);
                assert!(matches!(update, MoreDataNeeded));
            }
        }
    }
}
