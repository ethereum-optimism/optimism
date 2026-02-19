#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/alloy.jpg",
    html_favicon_url = "https://raw.githubusercontent.com/alloy-rs/core/main/assets/favicon.ico"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]

extern crate alloc;

#[cfg(feature = "alloy-compat")]
mod alloy_compat;

mod receipts;
pub use receipts::{
    OpDepositReceipt, OpDepositReceiptWithBloom, OpReceipt, OpReceiptEnvelope, OpTxReceipt,
};

pub mod transaction;
pub use transaction::{
    DEPOSIT_TX_TYPE_ID, DepositTransaction, OpPooledTransaction, OpTransaction, OpTxEnvelope,
    OpTxType, OpTypedTransaction, TxDeposit,
};

pub mod eip1559;
pub use eip1559::{
    EIP1559ParamError, decode_eip_1559_params, decode_holocene_extra_data,
    decode_jovian_extra_data, encode_holocene_extra_data, encode_jovian_extra_data,
};

mod source;
pub use source::*;

mod block;
pub use block::OpBlock;

pub mod interop;

pub mod predeploys;
pub use predeploys::L2_TO_L1_MESSAGE_PASSER_ADDRESS;

#[cfg(feature = "serde")]
pub use transaction::serde_deposit_tx_rpc;

/// Bincode-compatible serde implementations for consensus types.
///
/// `bincode` crate doesn't work well with optionally serializable serde fields, but some of the
/// consensus types require optional serialization for RPC compatibility. This module makes so that
/// all fields are serialized.
///
/// Read more: <https://github.com/bincode-org/bincode/issues/326>
#[cfg(all(feature = "serde", feature = "serde-bincode-compat"))]
pub mod serde_bincode_compat {
    pub use super::{
        receipts::{
            deposit::serde_bincode_compat::OpDepositReceipt,
            receipt::serde_bincode_compat::OpReceipt,
        },
        transaction::{serde_bincode_compat as transaction, serde_bincode_compat::TxDeposit},
    };
}
