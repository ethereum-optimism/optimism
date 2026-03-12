//! Primitive types for `kona-derive`.

mod results;
pub use results::PipelineResult;

mod signals;
pub use signals::{ActivationSignal, ResetSignal, Signal};
