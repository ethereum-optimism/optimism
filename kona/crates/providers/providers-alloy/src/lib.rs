#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/op-rs/kona/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]

mod metrics;
pub use beacon_client::BeaconClientError;
pub use metrics::Metrics;

mod beacon_client;
pub use beacon_client::{
    APIConfigResponse, APIGenesisResponse, BeaconClient, OnlineBeaconClient, ReducedConfigData,
    ReducedGenesisData,
};

mod blobs;
pub use blobs::{BoxedBlobWithIndex, OnlineBlobProvider};

mod chain_provider;
pub use chain_provider::{AlloyChainProvider, AlloyChainProviderError};

mod l2_chain_provider;
pub use l2_chain_provider::{AlloyL2ChainProvider, AlloyL2ChainProviderError};

mod pipeline;
pub use pipeline::OnlinePipeline;
