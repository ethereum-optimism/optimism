//! The [`AuthorityActor`], following a cross-chain verification authority.

mod actor;
pub use actor::{AUTHORITY_POLL_INTERVAL, AuthorityActor, AuthorityActorError};

mod epoch;
pub use epoch::{AppliedEpoch, AppliedEpochRpc};

#[cfg(test)]
mod tests;
