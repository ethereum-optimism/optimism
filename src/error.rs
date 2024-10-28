use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("Invalid arguments: {0}")]
    InvalidArgs(String),
    #[error("Error Initializing RPC Client: {0}")]
    InitRPCClient(String),
    #[error("Error Initializing RPC Server: {0}")]
    InitRPCServer(String),
    #[error("Error Initializing Prometheus Metrics: {0}")]
    InitMetrics(String),
}
