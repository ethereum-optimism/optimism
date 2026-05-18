use alloy_primitives::Address;

#[derive(Debug, thiserror::Error)]
pub enum GasLimitError {
    #[error(
        "Address {address} exceeded gas limit: {requested} gwei requested, {available} gwei available"
    )]
    AddressLimitExceeded {
        address: Address,
        requested: u64,
        available: u64,
    },
}
