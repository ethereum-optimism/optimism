// The MIT License (MIT)
// Copyright (c) 2022-2025 Reth Contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

//! Superchain consensus errors

use derive_more;

/// Supervisor data availability error codes.
///
/// Specs: <https://specs.optimism.io/interop/supervisor.html#protocol-specific-error-codes>
#[derive(thiserror::Error, Debug, Clone, Copy, PartialEq, Eq, derive_more::TryFrom)]
#[repr(i32)]
#[try_from(repr)]
pub enum SuperchainDAError {
    //-------------------------------- -3205XX NOT_FOUND errors --------------------------------//
    /// Happens when we try to retrieve data that is not available (pruned).
    /// It may also happen if we erroneously skip data, that was not considered a conflict, if the
    /// DB is corrupted.
    #[error("data was skipped or pruned and is not available")]
    SkippedData = -320500,

    /// Happens when a chain is unknown, not in the dependency set.
    #[error("unsupported chain id")]
    UnknownChain = -320501,

    //--------------------------------- -3206XX ALREADY_EXISTS ---------------------------------//
    /// Happens when we know for sure that there is different canonical data.
    #[error("conflicting data exists in the database")]
    ConflictingData = -320600,

    /// Happens when data is accepted as compatible, but did not change anything.
    /// This happens when a node is deriving an L2 block we already know of being
    /// derived from the given source,
    /// but without path to skip forward to newer source blocks without doing the known
    /// derivation work first.
    #[error("data is already known and didn't change anything")]
    IneffectiveData = -320601,

    //-------------------------- -3209XX FAILED_PRECONDITION errors ----------------------------//
    /// Happens when you try to add data to the DB, but it does not actually fit onto
    /// the latest data.
    /// (by being too old or new).
    #[error("data is out of order (too old or new)")]
    OutOfOrder = -320900,

    /// Happens when something was assumed from the DB, but then invalidated due to e.g. a reorg.
    #[error("invalidated read")]
    InvalidatedRead = -32901,

    /// Happens when we know for sure that a replacement block is needed before progress
    /// can be made.
    #[error("waiting for replacement block before progress can be made")]
    AwaitingReplacement = -320901,

    //--------------------------------- -3210XX ABORTED errors ---------------------------------//
    /// Happens when we fail to rewind the chain (reorg response).
    #[error("rewind failed")]
    RewindFailed = -321000,

    // -3211XX OUT_OF_RANGE errors
    /// Happens when data is accessed, but access is not allowed, because of a limited
    /// scope.
    /// E.g. when limiting scope to L2 blocks derived from a specific subset of the L1
    /// chain.
    #[error("data access not allowed due to limited scope")]
    OutOfScope = -321100,

    //------------------------------ -3212XX UNIMPLEMENTED errors ------------------------------//
    /// Happens when you try to get the previous block of the first block.
    /// E.g. when trying to determine the previous source block for the first L1 block
    /// in the database.
    #[error("cannot get parent of first block in database")]
    NoParentForFirstBlock = -321200,

    //------------------------------- -3214XX UNAVAILABLE errors -------------------------------//
    /// Happens when data is just not yet available.
    #[error("data is not yet available (from the future)")]
    FutureData = -321401,

    //-------------------------------- -3215XX DATA_LOSS errors --------------------------------//
    /// Happens when we search the DB, know the data may be there, but is not (e.g.
    /// different revision).
    #[error("data may exist but was not found (possibly different revision)")]
    MissedData = -321500,

    /// Happens when the underlying DB has some I/O issue.
    #[error("underlying database has I/O issues or is corrupted")]
    DataCorruption = -321501,
}

#[cfg(feature = "jsonrpsee")]
impl From<SuperchainDAError> for jsonrpsee::types::ErrorObjectOwned {
    fn from(err: SuperchainDAError) -> Self {
        use crate::alloc::string::ToString;

        jsonrpsee::types::ErrorObjectOwned::owned(err as i32, err.to_string(), None::<()>)
    }
}
