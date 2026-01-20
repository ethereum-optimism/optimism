#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/op-rs/kona/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]

#[macro_use]
extern crate tracing;

mod admin;
pub use admin::{AdminRpc, NetworkAdminQuery, RollupBoostAdminQuery};

mod client;
pub use client::{
    EngineRpcClient, RollupBoostAdminClient, SequencerAdminAPIClient, SequencerAdminAPIError,
};

mod config;
pub use config::RpcBuilder;

mod net;
pub use net::P2pRpc;

mod p2p;

mod response;
pub use response::SafeHeadResponse;

mod output;
pub use output::OutputResponse;

mod dev;
pub use dev::DevEngineRpc;

mod jsonrpsee;
pub use jsonrpsee::{
    AdminApiServer, DevEngineApiServer, HealthzApiServer, MinerApiExtServer, OpAdminApiServer,
    OpP2PApiServer, RollupBoostHealthzApiServer, RollupNodeApiServer, WsServer,
};

#[cfg(feature = "client")]
pub use jsonrpsee::RollupNodeApiClient;

mod rollup;
pub use rollup::RollupRpc;

mod l1_watcher;
pub use l1_watcher::{L1State, L1WatcherQueries, L1WatcherQuerySender};

mod ws;
pub use ws::WsRPC;

mod health;
pub use health::{
    HealthzResponse, HealthzRpc, RollupBoostHealth, RollupBoostHealthQuery,
    RollupBoostHealthzResponse,
};
