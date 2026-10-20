//! The [`EngineActor`], [`EngineRpcActor`], and their components.

mod actor;
pub use actor::EngineActor;

pub(crate) mod handle;
pub use handle::{
    BuildToken, Consolidated, EngineContext, EngineHandle, Sealed, UnsafeAdmissionError,
};

mod config;
pub use config::EngineConfig;

mod error;
pub use error::{EngineClientError, EngineClientResult, EngineError};

mod request;
pub use request::EngineRpcRequest;

mod rpc_actor;
pub use rpc_actor::EngineRpcActor;
