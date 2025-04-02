#![cfg_attr(
    not(any(test, feature = "integration")),
    warn(unused_crate_dependencies)
)]

use dotenv as _;
use rustls as _;

mod client;
pub use client::{auth::*, http::*, rpc::*};

mod cli;
pub use cli::*;

mod debug_api;
pub use debug_api::*;

mod health;
pub use health::{HealthLayer, HealthService};

#[cfg(all(feature = "integration", test))]
mod integration;

mod metrics;
pub use metrics::*;

mod proxy;
pub use proxy::*;

mod server;
pub use server::*;

mod tracing;
pub use tracing::*;
