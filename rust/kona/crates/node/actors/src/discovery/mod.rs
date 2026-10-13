//! Discovery actor and handle.
//!
//! The discovery actor runs a discv5 service: it looks up random nodes on an interval, passes
//! every valid peer it finds to the [`Network`] to dial, keeps the bootstore on disk up to date,
//! and answers queries and bans sent through its [`Handle`].
//!
//! The [`Builder`] creates the handle before any dependencies exist. [`Builder::build`] takes the
//! [`Discv5Driver`](kona_disc::Discv5Driver) that
//! [`NetworkConfig::start`](crate::NetworkConfig::start) returns, and the actor starts the discv5
//! service when its future is first polled.

mod actor;
pub use actor::{ActorError, Builder};

mod handle;
pub use crate::capacity::{Capacity, InvalidCapacity};
pub use handle::{Handle, HandleError};

mod network;
pub use network::Network;
