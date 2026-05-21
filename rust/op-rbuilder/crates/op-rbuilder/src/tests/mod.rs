// base
mod framework;
pub use framework::*;

#[cfg(test)]
mod flashblocks;

#[cfg(test)]
mod flashtestations;

#[cfg(test)]
mod data_availability;

#[cfg(test)]
mod miner_gas_limit;

#[cfg(test)]
mod gas_limiter;

#[cfg(test)]
mod ordering;

#[cfg(test)]
mod revert;

#[cfg(test)]
mod smoke;

#[cfg(test)]
mod txpool;

#[cfg(test)]
mod forks;
// If the order of deployment from the signer changes the address will change
#[cfg(test)]
const FLASHBLOCKS_NUMBER_ADDRESS: alloy_primitives::Address =
    alloy_primitives::address!("95bd8d42f30351685e96c62eddc0d0613bf9a87a");
#[cfg(test)]
const MOCK_DCAP_ADDRESS: alloy_primitives::Address =
    alloy_primitives::address!("700b6a60ce7eaaea56f065753d8dcb9653dbad35");
#[cfg(test)]
const FLASHTESTATION_REGISTRY_ADDRESS: alloy_primitives::Address =
    alloy_primitives::address!("a15bb66138824a1c7167f5e85b957d04dd34e468");
#[cfg(test)]
const BLOCK_BUILDER_POLICY_ADDRESS: alloy_primitives::Address =
    alloy_primitives::address!("8ce361602b935680e8dec218b820ff5056beb7af");
