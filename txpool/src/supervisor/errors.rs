use core::error;

/// Failures occurring during validation of inbox entries.
#[derive(thiserror::Error, Debug)]
pub enum InteropTxValidatorError {
    /// Inbox entry validation against the Supervisor took longer than allowed.
    #[error("inbox entry validation timed out, timeout: {0} secs")]
    Timeout(u64),

    /// Catch-all variant.
    #[error("supervisor server error: {0}")]
    Other(Box<dyn error::Error + Send + Sync>),
}

impl InteropTxValidatorError {
    /// Returns a new instance of [`Other`](Self::Other) error variant.
    pub fn other<E>(err: alloy_json_rpc::RpcError<E>) -> Self
    where
        E: error::Error + Send + Sync + 'static,
    {
        Self::Other(Box::new(err))
    }
}
