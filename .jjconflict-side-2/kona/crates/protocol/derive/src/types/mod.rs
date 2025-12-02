//! Primitive types for `kona-derive`.

mod results;
pub use results::{PipelineResult, StepResult};

mod signals;
pub use signals::{ActivationSignal, ResetSignal, Signal};
