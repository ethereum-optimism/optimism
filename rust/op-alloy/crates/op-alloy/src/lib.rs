#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/alloy.jpg",
    html_favicon_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/favicon.ico"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(any(feature = "full", feature = "std")), no_std)]

#[cfg(feature = "consensus")]
#[doc(inline)]
pub use op_alloy_consensus as consensus;

#[cfg(feature = "provider")]
#[doc(inline)]
pub use op_alloy_provider as provider;

#[cfg(feature = "network")]
#[doc(inline)]
pub use op_alloy_network as network;

#[cfg(feature = "rpc-types")]
#[doc(inline)]
pub use op_alloy_rpc_types as rpc_types;

#[cfg(feature = "rpc-types-engine")]
#[doc(inline)]
pub use op_alloy_rpc_types_engine as rpc_types_engine;

#[cfg(feature = "rpc-jsonrpsee")]
#[doc(inline)]
pub use op_alloy_rpc_jsonrpsee as rpc_jsonrpsee;
