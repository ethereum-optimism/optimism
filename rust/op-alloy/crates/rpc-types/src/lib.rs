#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/alloy.jpg",
    html_favicon_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/favicon.ico"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]

extern crate alloc;

mod genesis;
pub use genesis::{OpBaseFeeInfo, OpChainInfo, OpGenesisInfo};

mod receipt;
pub use receipt::{L1BlockInfo, OpTransactionReceipt, OpTransactionReceiptFields};

mod transaction;
pub use transaction::{OpTransactionFields, OpTransactionRequest, Transaction};

pub mod error;
pub use error::SuperchainDAError;
