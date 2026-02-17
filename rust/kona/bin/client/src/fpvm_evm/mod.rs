//! Custom [`EvmFactory`] for the fault proof virtual machine's EVM.
//!
//! [`EvmFactory`]: alloy_evm::EvmFactory

mod precompiles;

mod factory;
pub use factory::FpvmOpEvmFactory;
