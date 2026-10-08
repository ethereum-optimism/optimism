#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

#[macro_use]
extern crate tracing;

mod admin;
pub use admin::{
    AdminRpc, SequencerAdminAPIError, SequencerAdminCommand, SequencerAdminHandle, SequencerState,
};

mod config;
pub use config::RpcBuilder;

mod p2p;
pub use p2p::P2pRpc;

mod jsonrpsee;
pub use jsonrpsee::{
    AdminApiServer, HealthzApiServer, MinerApiExtServer, OpAdminApiServer, OpP2PApiServer,
    RollupNodeApiServer,
};

#[cfg(feature = "client")]
pub use jsonrpsee::RollupNodeApiClient;

mod rollup;
pub use rollup::{OutputError, OutputProvider, OutputResponse, RollupRpc};

mod l1_watcher;
pub use l1_watcher::{L1State, L1WatcherQueries, L1WatcherQuerySender};

mod health;
pub use health::{HealthzResponse, HealthzRpc};
