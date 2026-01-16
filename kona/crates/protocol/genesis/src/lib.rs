#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/op-rs/kona/issues/"
)]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]
#![cfg_attr(not(feature = "std"), no_std)]

extern crate alloc;

mod params;
pub use params::{
    BASE_MAINNET_BASE_FEE_CONFIG, BASE_MAINNET_EIP1559_BASE_FEE_MAX_CHANGE_DENOMINATOR_CANYON,
    BASE_MAINNET_EIP1559_DEFAULT_BASE_FEE_MAX_CHANGE_DENOMINATOR,
    BASE_MAINNET_EIP1559_DEFAULT_ELASTICITY_MULTIPLIER, BASE_SEPOLIA_BASE_FEE_CONFIG,
    BASE_SEPOLIA_BASE_FEE_PARAMS, BASE_SEPOLIA_BASE_FEE_PARAMS_CANYON,
    BASE_SEPOLIA_EIP1559_BASE_FEE_MAX_CHANGE_DENOMINATOR_CANYON,
    BASE_SEPOLIA_EIP1559_DEFAULT_BASE_FEE_MAX_CHANGE_DENOMINATOR,
    BASE_SEPOLIA_EIP1559_DEFAULT_ELASTICITY_MULTIPLIER, BaseFeeConfig, OP_MAINNET_BASE_FEE_CONFIG,
    OP_MAINNET_BASE_FEE_PARAMS, OP_MAINNET_BASE_FEE_PARAMS_CANYON,
    OP_MAINNET_EIP1559_BASE_FEE_MAX_CHANGE_DENOMINATOR_CANYON,
    OP_MAINNET_EIP1559_DEFAULT_BASE_FEE_MAX_CHANGE_DENOMINATOR,
    OP_MAINNET_EIP1559_DEFAULT_ELASTICITY_MULTIPLIER, OP_SEPOLIA_BASE_FEE_CONFIG,
    OP_SEPOLIA_BASE_FEE_PARAMS, OP_SEPOLIA_BASE_FEE_PARAMS_CANYON,
    OP_SEPOLIA_EIP1559_BASE_FEE_MAX_CHANGE_DENOMINATOR_CANYON,
    OP_SEPOLIA_EIP1559_DEFAULT_BASE_FEE_MAX_CHANGE_DENOMINATOR,
    OP_SEPOLIA_EIP1559_DEFAULT_ELASTICITY_MULTIPLIER, base_fee_config, base_fee_params,
    base_fee_params_canyon,
};

mod superchain;
pub use superchain::{
    Chain, ChainList, FaultProofs, Superchain, SuperchainConfig, SuperchainL1Info, SuperchainLevel,
    SuperchainParent, Superchains,
};

mod updates;
pub use updates::{
    BatcherUpdate, DaFootprintGasScalarUpdate, Eip1559Update, GasConfigUpdate, GasLimitUpdate,
    MinBaseFeeUpdate, OperatorFeeUpdate, UnsafeBlockSignerUpdate,
};

mod system;
pub use system::{
    BatcherUpdateError, CONFIG_UPDATE_EVENT_VERSION_0, CONFIG_UPDATE_TOPIC,
    DaFootprintGasScalarUpdateError, EIP1559UpdateError, GasConfigUpdateError, GasLimitUpdateError,
    LogProcessingError, MinBaseFeeUpdateError, OperatorFeeUpdateError, SystemConfig,
    SystemConfigLog, SystemConfigUpdate, SystemConfigUpdateError, SystemConfigUpdateKind,
    UnsafeBlockSignerUpdateError,
};

mod chain;
pub use chain::{
    AddressList, AltDAConfig, BASE_MAINNET_CHAIN_ID, BASE_SEPOLIA_CHAIN_ID, ChainConfig,
    HardForkConfig, L1ChainConfig, OP_MAINNET_CHAIN_ID, OP_SEPOLIA_CHAIN_ID, Roles,
};

mod genesis;
pub use genesis::ChainGenesis;

mod rollup;
pub use rollup::{
    DEFAULT_INTEROP_MESSAGE_EXPIRY_WINDOW, FJORD_MAX_SEQUENCER_DRIFT, GRANITE_CHANNEL_TIMEOUT,
    MAX_RLP_BYTES_PER_CHANNEL_BEDROCK, MAX_RLP_BYTES_PER_CHANNEL_FJORD, RollupConfig,
};
