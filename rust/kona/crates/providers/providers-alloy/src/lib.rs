#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/ethereum-optimism/optimism/develop/rust/kona/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/ethereum-optimism/optimism/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg))]

mod metrics;
pub use beacon_client::BeaconClientError;
pub use metrics::Metrics;

mod beacon_client;
pub use beacon_client::{
    APIConfigResponse, APIGenesisResponse, BeaconClient, OnlineBeaconClient, ReducedConfigData,
    ReducedGenesisData,
};

mod blobs;
pub use blobs::{BlobWithCommitmentAndProof, BoxedBlob, OnlineBlobProvider};

mod blob_decode;
pub use blob_decode::decode_blob;

mod chain_provider;
pub use chain_provider::{AlloyChainProvider, AlloyChainProviderError};

mod l1_data;
pub use l1_data::{L1BlockData, L1FetchError, fetch_l1_block_data};

mod l2_chain_provider;
pub use l2_chain_provider::{AlloyL2ChainProvider, AlloyL2ChainProviderError};
