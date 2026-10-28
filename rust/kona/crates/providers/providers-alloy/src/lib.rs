#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod metrics;
mod source;
pub use source::{BlockSource, Cached, ReceiptSource};

mod rpc_source;
pub use rpc_source::RpcSource;

mod source_error;
pub use beacon_client::BeaconClientError;
pub use metrics::Metrics;
pub use source_error::{SourceError, SourceError as AlloyChainProviderError};

mod beacon_client;
pub use beacon_client::{
    APIConfigResponse, APIGenesisResponse, BeaconClient, OnlineBeaconClient, ReducedConfigData,
    ReducedGenesisData, SlotDurationError,
};

mod blobs;
pub use blobs::{BlobWithCommitmentAndProof, BoxedBlob, OnlineBlobProvider};

mod chain_provider;
pub use chain_provider::AlloyChainProvider;

mod l2_chain_provider;
pub use l2_chain_provider::{AlloyL2ChainProvider, AlloyL2ChainProviderError};

mod buffered_l2_chain_provider;
pub use buffered_l2_chain_provider::BufferedAlloyL2ChainProvider;

mod pipeline;
pub use pipeline::OnlinePipeline;

mod runtime_config;
pub use runtime_config::unsafe_block_signer;
