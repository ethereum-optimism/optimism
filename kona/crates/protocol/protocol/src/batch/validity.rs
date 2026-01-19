//! Contains the [`BatchValidity`], [`BatchDropReason`] and their encodings.

/// Reasons why a batch may be dropped.
///
/// This enum provides detailed context for why a batch was deemed invalid,
/// enabling more precise error handling and testing without relying on log message parsing.
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BatchDropReason {
    // === Timestamp-related drops ===
    /// Batch timestamp is in the future (Holocene active).
    FutureTimestampHolocene,
    /// Batch timestamp is in the past (Holocene inactive).
    PastTimestampPreHolocene,

    // === Parent/origin validation drops ===
    /// Parent hash does not match the L2 safe head.
    ParentHashMismatch,
    /// Batch was included too late (outside sequencer window).
    IncludedTooLate,
    /// Batch epoch is older than the current epoch.
    EpochTooOld,
    /// Batch epoch is too far in the future.
    EpochTooFarInFuture,
    /// Batch epoch hash does not match the L1 origin.
    EpochHashMismatch,

    // === Timestamp/origin relationship drops ===
    /// Batch timestamp is before the L1 origin timestamp.
    TimestampBeforeL1Origin,
    /// Sequencer drift overflow (checked_add failed).
    SequencerDriftOverflow,
    /// Batch exceeded sequencer time drift with non-empty transactions.
    SequencerDriftExceeded,
    /// Empty batch could have adopted next L1 origin but didn't.
    SequencerDriftNotAdoptedNextOrigin,

    // === Transaction validation drops ===
    /// Batch contains an empty transaction.
    EmptyTransaction,
    /// Batch contains a deposit transaction (not allowed in batch data).
    DepositTransaction,
    /// EIP-7702 transaction included before Isthmus activation.
    Eip7702PreIsthmus,
    /// Non-empty batch in Jovian or Interop transition block.
    NonEmptyTransitionBlock,

    // === Span batch specific drops ===
    /// Span batch received before Delta hard fork.
    SpanBatchPreDelta,
    /// Span batch has no new blocks after safe head (Holocene inactive).
    SpanBatchNoNewBlocksPreHolocene,
    /// Span batch timestamp is misaligned with block time.
    SpanBatchMisalignedTimestamp,
    /// Span batch timestamp does not overlap exactly.
    SpanBatchNotOverlappedExactly,
    /// Batch L1 origin is before safe head L1 origin.
    L1OriginBeforeSafeHead,
    /// Unable to find L1 origin for batch.
    MissingL1Origin,
    /// Overlapped block's transaction count does not match.
    OverlappedTxCountMismatch,
    /// Overlapped block's transaction does not match.
    OverlappedTxMismatch,
    /// Failed to extract L2BlockInfo from execution payload.
    L2BlockInfoExtractionFailed,
    /// Overlapped block's L1 origin number does not match.
    OverlappedL1OriginMismatch,
}

impl core::fmt::Display for BatchDropReason {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::FutureTimestampHolocene => {
                write!(f, "batch timestamp is in the future (Holocene active)")
            }
            Self::PastTimestampPreHolocene => {
                write!(f, "batch timestamp is in the past (Holocene inactive)")
            }
            Self::ParentHashMismatch => write!(f, "parent hash does not match L2 safe head"),
            Self::IncludedTooLate => write!(f, "batch was included too late"),
            Self::EpochTooOld => write!(f, "batch epoch is too old"),
            Self::EpochTooFarInFuture => write!(f, "batch epoch is too far in the future"),
            Self::EpochHashMismatch => write!(f, "batch epoch hash does not match L1 origin"),
            Self::TimestampBeforeL1Origin => {
                write!(f, "batch timestamp is before L1 origin timestamp")
            }
            Self::SequencerDriftOverflow => write!(f, "sequencer drift calculation overflow"),
            Self::SequencerDriftExceeded => write!(f, "batch exceeded sequencer time drift"),
            Self::SequencerDriftNotAdoptedNextOrigin => {
                write!(f, "empty batch could have adopted next L1 origin")
            }
            Self::EmptyTransaction => write!(f, "batch contains empty transaction"),
            Self::DepositTransaction => write!(f, "batch contains deposit transaction"),
            Self::Eip7702PreIsthmus => write!(f, "EIP-7702 transaction before Isthmus activation"),
            Self::NonEmptyTransitionBlock => write!(f, "non-empty batch in transition block"),
            Self::SpanBatchPreDelta => write!(f, "span batch received before Delta hard fork"),
            Self::SpanBatchNoNewBlocksPreHolocene => {
                write!(f, "span batch has no new blocks after safe head")
            }
            Self::SpanBatchMisalignedTimestamp => write!(f, "span batch has misaligned timestamp"),
            Self::SpanBatchNotOverlappedExactly => write!(f, "span batch does not overlap exactly"),
            Self::L1OriginBeforeSafeHead => {
                write!(f, "batch L1 origin is before safe head L1 origin")
            }
            Self::MissingL1Origin => write!(f, "unable to find L1 origin for batch"),
            Self::OverlappedTxCountMismatch => {
                write!(f, "overlapped block transaction count mismatch")
            }
            Self::OverlappedTxMismatch => write!(f, "overlapped block transaction mismatch"),
            Self::L2BlockInfoExtractionFailed => {
                write!(f, "failed to extract L2BlockInfo from execution payload")
            }
            Self::OverlappedL1OriginMismatch => {
                write!(f, "overlapped block L1 origin number mismatch")
            }
        }
    }
}

/// Batch Validity
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BatchValidity {
    /// The batch is invalid now and in the future, unless we reorg, so it can be discarded.
    /// Contains the reason for dropping the batch.
    Drop(BatchDropReason),
    /// The batch is valid and should be processed
    Accept,
    /// We are lacking L1 information until we can proceed batch filtering
    Undecided,
    /// The batch may be valid, but cannot be processed yet and should be checked again later
    Future,
    /// Introduced in Holocene, a special variant of the `Drop` variant that signals not to flush
    /// the active batch and channel, in the case of processing an old batch
    Past,
}

impl core::fmt::Display for BatchValidity {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::Drop(reason) => write!(f, "Drop({reason})"),
            Self::Accept => write!(f, "Accept"),
            Self::Undecided => write!(f, "Undecided"),
            Self::Future => write!(f, "Future"),
            Self::Past => write!(f, "Past"),
        }
    }
}

impl BatchValidity {
    /// Returns whether the batch is accepted.
    pub const fn is_accept(&self) -> bool {
        matches!(self, Self::Accept)
    }

    /// Returns whether the batch is dropped.
    pub const fn is_drop(&self) -> bool {
        matches!(self, Self::Drop(_))
    }

    /// Returns the drop reason if the batch was dropped.
    pub const fn drop_reason(&self) -> Option<BatchDropReason> {
        match self {
            Self::Drop(reason) => Some(*reason),
            _ => None,
        }
    }

    /// Returns whether the batch is outdated.
    pub const fn is_outdated(&self) -> bool {
        matches!(self, Self::Past)
    }

    /// Returns whether the batch is future.
    pub const fn is_future(&self) -> bool {
        matches!(self, Self::Future)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use alloc::format;

    #[test]
    fn test_batch_validity() {
        assert!(BatchValidity::Accept.is_accept());
        assert!(BatchValidity::Drop(BatchDropReason::ParentHashMismatch).is_drop());
        assert!(BatchValidity::Past.is_outdated());
        assert!(BatchValidity::Future.is_future());
    }

    #[test]
    fn test_drop_reason() {
        let validity = BatchValidity::Drop(BatchDropReason::EmptyTransaction);
        assert_eq!(validity.drop_reason(), Some(BatchDropReason::EmptyTransaction));
        assert!(BatchValidity::Accept.drop_reason().is_none());
    }

    #[test]
    fn test_batch_drop_reason_display() {
        assert_eq!(
            format!("{}", BatchDropReason::ParentHashMismatch),
            "parent hash does not match L2 safe head"
        );
        assert_eq!(
            format!("{}", BatchDropReason::EmptyTransaction),
            "batch contains empty transaction"
        );
    }

    #[test]
    fn test_batch_validity_display() {
        assert_eq!(
            format!("{}", BatchValidity::Drop(BatchDropReason::ParentHashMismatch)),
            "Drop(parent hash does not match L2 safe head)"
        );
        assert_eq!(format!("{}", BatchValidity::Accept), "Accept");
    }
}
