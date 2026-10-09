//! Addresses of OP Stack pre-deployed contracts.

use alloy_primitives::{Address, address};

/// The address of the `L1Block` predeploy.
pub const L1_BLOCK_ADDRESS: Address = address!("0x4200000000000000000000000000000000000015");

/// The address of the `L2ToL1MessagePasser` predeploy.
pub const L2_TO_L1_MESSAGE_PASSER_ADDRESS: Address =
    address!("0x4200000000000000000000000000000000000016");
