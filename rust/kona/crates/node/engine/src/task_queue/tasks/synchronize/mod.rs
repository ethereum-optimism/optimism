//! The forkchoice engine update shared by the engine tasks, and its error type.

mod helper;
pub(in crate::task_queue) use helper::{
    ForkchoiceDelivery, ForkchoiceReply, record_forkchoice_reply, restore_forkchoice,
    send_forkchoice, synchronize,
};

mod error;
pub use error::{SafeLabel, SynchronizeTaskError};

#[cfg(test)]
mod tests;
