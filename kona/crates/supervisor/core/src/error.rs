//! [`SupervisorService`](crate::SupervisorService) errors.

use crate::syncnode::ManagedNodeError;
use derive_more;
use jsonrpsee::types::{ErrorCode, ErrorObjectOwned};
use kona_supervisor_storage::StorageError;
use kona_supervisor_types::AccessListError;
use op_alloy_rpc_types::SuperchainDAError;
use thiserror::Error;

/// Custom error type for the Supervisor core logic.
#[derive(Debug, Error)]
pub enum SupervisorError {
    /// Indicates that a feature or method is not yet implemented.
    #[error("functionality not implemented")]
    Unimplemented,

    /// No chains are configured for supervision.
    #[error("empty dependency set")]
    EmptyDependencySet,

    /// Unsupported chain ID.
    #[error("unsupported chain ID")]
    UnsupportedChainId,

    /// Data availability errors.
    ///
    /// Spec <https://github.com/ethereum-optimism/specs/blob/main/specs/interop/supervisor.md#protocol-specific-error-codes>.
    #[error(transparent)]
    SpecError(#[from] SpecError),

    /// Indicates that error occurred while interacting with the storage layer.
    #[error(transparent)]
    StorageError(#[from] StorageError),

    /// Indicates that managed node not found for the chain.
    #[error("managed node not found for chain: {0}")]
    ManagedNodeMissing(u64),

    /// Indicates the error occurred while interacting with the managed node.
    #[error(transparent)]
    ManagedNodeError(#[from] ManagedNodeError),

    /// Indicates the error occurred while parsing the access_list
    #[error(transparent)]
    AccessListError(#[from] AccessListError),

    /// Indicates the error occurred while serializing or deserializing JSON.
    #[error(transparent)]
    SerdeJson(#[from] serde_json::Error),

    /// Indicates the L1 block does not match the expected L1 block.
    #[error("L1 block number mismatch. expected: {expected}, but got {got}")]
    L1BlockMismatch {
        /// Expected L1 block.
        expected: u64,
        /// Received L1 block.
        got: u64,
    },

    /// Indicates that the chain ID could not be parsed from the access list.
    #[error("failed to parse chain id from access list")]
    ChainIdParseError(),
}

impl PartialEq for SupervisorError {
    fn eq(&self, other: &Self) -> bool {
        use SupervisorError::*;
        match (self, other) {
            (Unimplemented, Unimplemented) => true,
            (EmptyDependencySet, EmptyDependencySet) => true,
            (SpecError(a), SpecError(b)) => a == b,
            (StorageError(a), StorageError(b)) => a == b,
            (ManagedNodeMissing(a), ManagedNodeMissing(b)) => a == b,
            (ManagedNodeError(a), ManagedNodeError(b)) => a == b,
            (AccessListError(a), AccessListError(b)) => a == b,
            (SerdeJson(a), SerdeJson(b)) => a.to_string() == b.to_string(),
            (L1BlockMismatch { expected: a, got: b }, L1BlockMismatch { expected: c, got: d }) => {
                a == c && b == d
            }
            _ => false,
        }
    }
}

impl Eq for SupervisorError {}

/// Extending the [`SuperchainDAError`] to include errors not in the spec.
#[derive(Error, Debug, PartialEq, Eq, derive_more::TryFrom)]
#[repr(i32)]
#[try_from(repr)]
pub enum SpecError {
    /// [`SuperchainDAError`] from the spec.
    #[error(transparent)]
    SuperchainDAError(#[from] SuperchainDAError),

    /// Error not in spec.
    #[error("error not in spec")]
    ErrorNotInSpec,
}

impl SpecError {
    /// Maps the proper error code from SuperchainDAError.
    /// Introduced a new error code for errors not in the spec.
    pub const fn code(&self) -> i32 {
        match self {
            Self::SuperchainDAError(e) => *e as i32,
            Self::ErrorNotInSpec => -321300,
        }
    }
}

impl From<SpecError> for ErrorObjectOwned {
    fn from(err: SpecError) -> Self {
        ErrorObjectOwned::owned(err.code(), err.to_string(), None::<()>)
    }
}

impl From<SupervisorError> for ErrorObjectOwned {
    fn from(err: SupervisorError) -> Self {
        match err {
            // todo: handle these errors more gracefully
            SupervisorError::Unimplemented |
            SupervisorError::EmptyDependencySet |
            SupervisorError::UnsupportedChainId |
            SupervisorError::L1BlockMismatch { .. } |
            SupervisorError::ManagedNodeMissing(_) |
            SupervisorError::ManagedNodeError(_) |
            SupervisorError::StorageError(_) |
            SupervisorError::AccessListError(_) |
            SupervisorError::ChainIdParseError() |
            SupervisorError::SerdeJson(_) => ErrorObjectOwned::from(ErrorCode::InternalError),
            SupervisorError::SpecError(err) => err.into(),
        }
    }
}

impl From<StorageError> for SpecError {
    fn from(err: StorageError) -> Self {
        match err {
            StorageError::Database(_) => Self::from(SuperchainDAError::DataCorruption),
            StorageError::FutureData => Self::from(SuperchainDAError::FutureData),
            StorageError::EntryNotFound(_) => Self::from(SuperchainDAError::MissedData),
            StorageError::ConflictError => Self::from(SuperchainDAError::ConflictingData),
            StorageError::BlockOutOfOrder => Self::from(SuperchainDAError::OutOfOrder),
            StorageError::DatabaseNotInitialised => Self::ErrorNotInSpec,
            _ => Self::ErrorNotInSpec,
        }
    }
}

#[cfg(test)]
mod test {
    use kona_supervisor_storage::EntryNotFoundError;

    use super::*;

    #[test]
    fn test_storage_error_conversion() {
        let test_err = SpecError::from(StorageError::DatabaseNotInitialised);
        let expected_err = SpecError::ErrorNotInSpec;

        assert_eq!(test_err, expected_err);
    }

    #[test]
    fn test_unmapped_storage_error_conversion() {
        let spec_err = ErrorObjectOwned::from(SpecError::ErrorNotInSpec);
        let expected_err = SpecError::ErrorNotInSpec;

        assert_eq!(spec_err, expected_err.into());

        let spec_err = ErrorObjectOwned::from(SpecError::from(StorageError::LockPoisoned));
        let expected_err = SpecError::ErrorNotInSpec;

        assert_eq!(spec_err, expected_err.into());

        let spec_err = ErrorObjectOwned::from(SpecError::from(StorageError::FutureData));
        let expected_err = SpecError::SuperchainDAError(SuperchainDAError::FutureData);

        assert_eq!(spec_err, expected_err.into());

        let spec_err = ErrorObjectOwned::from(SpecError::from(StorageError::EntryNotFound(
            EntryNotFoundError::DerivedBlockNotFound(12),
        )));
        let expected_err = SpecError::SuperchainDAError(SuperchainDAError::MissedData);

        assert_eq!(spec_err, expected_err.into());
    }

    #[test]
    fn test_supervisor_error_conversion() {
        // This will happen implicitly in server rpc response calls.
        let supervisor_err = ErrorObjectOwned::from(SupervisorError::SpecError(SpecError::from(
            StorageError::LockPoisoned,
        )));
        let expected_err = SpecError::ErrorNotInSpec;

        assert_eq!(supervisor_err, expected_err.into());

        let supervisor_err = ErrorObjectOwned::from(SupervisorError::SpecError(SpecError::from(
            StorageError::FutureData,
        )));
        let expected_err = SpecError::SuperchainDAError(SuperchainDAError::FutureData);

        assert_eq!(supervisor_err, expected_err.into());
    }
}
