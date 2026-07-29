//! Contains types related to the [`SystemConfig`].

use alloy_primitives::{B256, b256};

/// `keccak256("ConfigUpdate(uint256,uint8,bytes)")`
pub const CONFIG_UPDATE_TOPIC: B256 =
    b256!("1d2b0bda21d56b8bd12d4f94ebacffdfb35f5e226f84b461103bb8beab6353be");

/// The initial version of the system config event log.
pub const CONFIG_UPDATE_EVENT_VERSION_0: B256 = B256::ZERO;

mod config;
pub use config::SystemConfig;

mod log;
pub use log::SystemConfigLog;

mod update;
pub use update::SystemConfigUpdate;

mod kind;
pub use kind::SystemConfigUpdateKind;

mod errors;
pub use errors::{
    BatcherUpdateError, DaFootprintGasScalarUpdateError, EIP1559UpdateError, GasConfigUpdateError,
    GasLimitUpdateError, LogProcessingError, MinBaseFeeUpdateError, OperatorFeeUpdateError,
    SystemConfigUpdateError, UnsafeBlockSignerUpdateError,
};
