//! Supervisor core syncnode module
//! This module provides the core functionality for managing nodes in the supervisor environment.

mod command;
pub use command::ManagedNodeCommand;

mod node;
pub use node::ManagedNode;

mod error;
pub use error::{AuthenticationError, ClientError, ManagedNodeError};

mod traits;
pub use traits::{
    BlockProvider, ManagedNodeController, ManagedNodeDataProvider, ManagedNodeProvider,
    SubscriptionHandler,
};

mod client;
pub use client::{Client, ClientConfig, ManagedNodeClient};

pub(super) mod metrics;
pub(super) mod resetter;
