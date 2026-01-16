/// Defines a contract for types that can report metrics.
/// This trait is intended to be implemented by types that need to report metrics
pub trait MetricsReporter {
    /// Reports metrics for the implementing type.
    /// This function is intended to be called periodically to collect and report metrics.
    /// The implementation should gather relevant metrics and report them to the configured metrics
    /// backend.
    fn report_metrics(&self);
}
