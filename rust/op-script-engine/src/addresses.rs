//! Well-known addresses, mirroring `op-chain-ops/script/addresses/addresses.go`
//! and `op-chain-ops/script/deterministic.go`.

use alloy_primitives::{Address, address};

/// hevm cheat-code precompile address (`keccak256("hevm cheat code")`).
pub const VM_ADDR: Address = address!("0x7109709ecfa91a80626ff3989d68f67f5b1dd12d");

/// The `console.log` sink ("console.log" in ascii).
pub const CONSOLE_ADDR: Address = address!("0x000000000000000000636f6e736f6c652e6c6f67");

/// Temporary script deployer (`keccak256("op-stack script deployer")`).
pub const SCRIPT_DEPLOYER: Address = address!("0x76ce131128f3616871f8cda86d18fab44e4d0d8b");

/// Default forge deployer address (`makeAddr("deployer")`).
pub const FORGE_DEPLOYER: Address = address!("0xae0bdc4eeac5e950b67c6819b118761caaf61946");

/// Foundry `DEFAULT_SENDER` (`keccak256("foundry default caller")`).
pub const DEFAULT_SENDER: Address = address!("0x1804c8ab1f12e6bbf3894d4083f33e07309d1f38");

/// Arachnid deterministic CREATE2 deployer used by forge broadcasts.
pub const CREATE2_DEPLOYER: Address = address!("0x4e59b44847b379578588920ca78fbf26c0b4956c");
