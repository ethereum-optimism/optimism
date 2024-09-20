use thiserror::Error;

#[derive(Error, Debug)]
pub enum Error {
    #[error("Invalid jwt token: {0}")]
    InvalidJWTToken(String),
    #[error("Error Initializing RPC Server: {0}")]
    InitRPCServerError(String),
}
