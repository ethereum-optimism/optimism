//! Network actor and handle.
//!
//! The network actor runs the libp2p gossip swarm. It publishes the sequencer's signed blocks,
//! passes the blocks it receives to the [`Engine`], dials the peers discovery finds, disconnects
//! peers whose score falls below the ban threshold, and makes the changes the P2P RPC asks for
//! through its [`Handle`].
//!
//! [`NetworkConfig::start`] starts the swarm listening. [`Builder::new`] takes the listening gossip
//! driver and publishes its state, and its handle can then be given to discovery and the RPC before
//! [`Builder::build`] supplies the remaining dependencies.

mod actor;
pub use actor::{ActorError, Builder};

mod config;
pub use config::NetworkConfig;

mod discovery;
pub use discovery::Discovery;

mod engine;
pub use engine::Engine;

mod error;
pub use error::StartError;

mod handle;
pub use handle::{Handle, HandleError};
