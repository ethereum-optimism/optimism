//! OP stack variation of chain spec constants.

use alloy_primitives::hex;

//------------------------------- BASE MAINNET -------------------------------//

/// Max gas limit on Base: <https://basescan.org/block/17208876>
pub const BASE_MAINNET_MAX_GAS_LIMIT: u64 = 105_000_000;

//------------------------------- BASE SEPOLIA -------------------------------//

/// Max gas limit on Base Sepolia: <https://sepolia.basescan.org/block/12506483>
pub const BASE_SEPOLIA_MAX_GAS_LIMIT: u64 = 45_000_000;

//----------------------------------- DEV ------------------------------------//

/// Dummy system transaction for dev mode
/// OP Mainnet transaction at index 0 in block 124665056.
///
/// <https://optimistic.etherscan.io/tx/0x312e290cf36df704a2217b015d6455396830b0ce678b860ebfcc30f41403d7b1>
pub const TX_SET_L1_BLOCK_OP_MAINNET_BLOCK_124665056: [u8; 251] = hex!(
    "7ef8f8a0683079df94aa5b9cf86687d739a60a9b4f0835e520ec4d664e2e415dca17a6df94deaddeaddeaddeaddeaddeaddeaddeaddead00019442000000000000000000000000000000000000158080830f424080b8a4440a5e200000146b000f79c500000000000000040000000066d052e700000000013ad8a3000000000000000000000000000000000000000000000000000000003ef1278700000000000000000000000000000000000000000000000000000000000000012fdf87b89884a61e74b322bbcf60386f543bfae7827725efaaf0ab1de2294a590000000000000000000000006887246668a3b87f54deb3b94ba47a6f63f32985"
);
