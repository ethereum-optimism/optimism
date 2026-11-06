//! Task structs — one per engine action variant.
//!
//! Each task owns its input data and reply channel. Its `execute` method
//! takes `&mut EngineState`, calls the appropriate state method, and sends
//! the reply. The engine dispatcher is a thin match with no business logic.

mod execute_block;
#[cfg(any(test, feature = "test-utils"))]
mod flush;
mod index_block;
mod reorg;
mod sync_to;
#[cfg(any(test, feature = "test-utils"))]
mod tip;
mod unwind;

pub(super) use execute_block::{ExecuteBlockTask, run as execute_block};
#[cfg(any(test, feature = "test-utils"))]
pub(super) use flush::FlushTask;
pub(super) use index_block::{IndexBlockTask, run as index_block};
pub use reorg::ReorgBlock;
pub(super) use reorg::ReorgTask;
pub(super) use sync_to::SyncToTask;
#[cfg(any(test, feature = "test-utils"))]
pub(super) use tip::TipTask;
pub(super) use unwind::UnwindTask;
