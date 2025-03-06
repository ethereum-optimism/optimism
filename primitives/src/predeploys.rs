//! Addresses of OP pre-deploys.
// todo: move to op-alloy

use alloy_primitives::{address, Address};

/// The L2 contract `L2ToL1MessagePasser`, stores commitments to withdrawal transactions.
pub const ADDRESS_L2_TO_L1_MESSAGE_PASSER: Address =
    address!("0x4200000000000000000000000000000000000016");
