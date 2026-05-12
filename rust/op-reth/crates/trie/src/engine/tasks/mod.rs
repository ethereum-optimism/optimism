//! Task structs — one per engine action variant.
//!
//! Each task owns its input data and reply channel. Its `execute` method
//! takes `&mut EngineState`, calls the appropriate state method, and sends
//! the reply. The engine dispatcher is a thin match with no business logic.

mod execute_block;
#[cfg(test)]
mod flush;
mod index_block;
mod reorg;
mod sync_to;
mod unwind;

pub(super) use execute_block::{ExecuteBlockTask, run as execute_block};
#[cfg(test)]
pub(super) use flush::FlushTask;
pub(super) use index_block::IndexBlockTask;
pub(super) use reorg::ReorgTask;
pub(super) use sync_to::SyncToTask;
pub(super) use unwind::UnwindTask;
