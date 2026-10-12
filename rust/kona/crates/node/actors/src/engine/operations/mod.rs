//! The [`Engine`] and the operations it performs.

mod core;
pub(super) use core::Engine;

mod synchronize;

mod insert;

mod build;

mod seal;
use seal::BuildSealCoupling;

mod consolidate;

mod finalize;

mod util;
pub(super) use util::BuildAndSealError;
