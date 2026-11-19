//! The forkchoice engine update shared by the engine tasks, and its error type.

mod helper;
pub(in crate::task_queue) use helper::{
    ForkchoiceReply, restore_forkchoice, synchronize, synchronize_unconditionally,
};

mod error;
pub use error::{SafeLabel, SynchronizeTaskError};

#[cfg(test)]
mod tests;
